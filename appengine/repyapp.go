package repyapp

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"cloud.google.com/go/storage"

	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/encoding/charmap"

	"google.golang.org/appengine"
	"google.golang.org/appengine/file"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"

	"github.com/lutzky/repy"
	"github.com/lutzky/repy/recode"
	"github.com/lutzky/repy/writerlogger"
	"github.com/pkg/errors"
)

func downloadREPYZip(ctx context.Context) ([]byte, error) {
	resp, err := urlfetch.Client(ctx).Get(repy.RepFileURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to download %q", repy.RepFileURL)
	}
	defer resp.Body.Close()

	return repy.ExtractFromZip(resp.Body)
}

func init() {
	http.HandleFunc("/update", handler)
}

type repyStorer struct {
	ctx     context.Context
	bucket  *storage.BucketHandle
	data    []byte
	sha1sum [20]byte
}

func newRepyStorer(ctx context.Context, data []byte) (*repyStorer, error) {
	bucketName, err := file.DefaultBucketName(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get default bucket name")
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get storage client")
	}

	result := &repyStorer{
		ctx:     ctx,
		bucket:  client.Bucket(bucketName),
		data:    data,
		sha1sum: sha1.Sum(data),
	}

	log.Infof(ctx, "REPY SHA1SUM: %x", result.sha1sum)

	return result, nil
}

func (rs *repyStorer) writeAllREPYFiles() error {
	repyBytesISO8859_8, err := recode.Recode(charmap.CodePage862, charmap.ISO8859_8, rs.data)
	if err != nil {
		return errors.Wrap(err, "failed to convert REPY to ISO8859-8")
	}

	baseFileName := fmt.Sprintf("%x.repy", rs.sha1sum)

	isMissing := true

	if exists, err := rs.fileExists(baseFileName); err != nil {
		return errors.Wrapf(err, "Coudln't check if %q already exists", baseFileName)
	} else if exists {
		log.Infof(rs.ctx, "%q already exists", baseFileName)
		isMissing = false
	}

	destinations := []struct {
		filename      string
		contentType   string
		data          []byte
		onlyIfMissing bool
	}{
		{baseFileName, "text/plain; charset=cp862", rs.data, true},
		{fmt.Sprintf("%x.txt", rs.sha1sum), "text/plain; charset=iso8859-8", repyBytesISO8859_8, true},
		{"latest.txt", "text/plain; charset=iso8859-8", repyBytesISO8859_8, false},
		{"latest.repy", "text/plain; charset=cp862", rs.data, false},
	}

	var g errgroup.Group
	for _, dest := range destinations {
		dest := dest
		g.Go(func() error {
			if dest.onlyIfMissing && !isMissing {
				return nil
			}
			log.Infof(rs.ctx, "writing %q with content-type %q", dest.filename, dest.contentType)
			if err := rs.copyToFile(dest.filename, bytes.NewReader(dest.data)); err != nil {
				return errors.Wrapf(err, "failed to write %q", dest.filename)
			}
			if err := rs.setContentType(dest.filename, dest.contentType); err != nil {
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	if isMissing {
		f := fmt.Sprintf("%x.timestamp", rs.sha1sum)
		log.Infof(rs.ctx, "Writing timestamp file %q", f)
		if err := rs.writeTimeStamp(f, time.Now()); err != nil {
			return err
		}
	}

	return nil
}

func (rs *repyStorer) parseJSONAndWrite() error {
	parseLogWriter, parseLogCloser := rs.makePublicObject("latest.parse.log")
	defer parseLogCloser()

	catalog, err := repy.ReadFile(bytes.NewReader(rs.data), writerlogger.Logger{W: parseLogWriter})
	if err != nil {
		fmt.Fprintf(parseLogWriter, "Read returned error: %v\n", err)
		return errors.Wrap(err, "failed to read catalog")
	}

	for _, filename := range []string{fmt.Sprintf("%x.json", rs.sha1sum), "latest.json"} {
		jsonWriter, jsonCloser := rs.makePublicObject(filename)
		defer jsonCloser()
		enc := json.NewEncoder(jsonWriter)

		if err := enc.Encode(catalog); err != nil {
			return errors.Wrapf(err, "failed to write %q", filename)
		}
	}

	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	repyBytes, err := downloadREPYZip(ctx)
	if err != nil {
		httpErrorWrap(ctx, w, err, "Failed to download REPY zip")
		return
	}

	rs, err := newRepyStorer(ctx, repyBytes)
	if err != nil {
		httpErrorWrap(ctx, w, err, "Failed to initialize REPY App")
		return
	}

	if err := rs.writeAllREPYFiles(); err != nil {
		httpErrorWrap(ctx, w, err, "Failed to write REPY files")
		return
	}

	if err := rs.parseJSONAndWrite(); err != nil {
		httpErrorWrap(ctx, w, err, "Failed to complete JSON Parsing")
		return
	}

	log.Infof(ctx, "Successfully parsed REPY")
	fmt.Fprintf(w, "Success")
}

func (rs *repyStorer) copyToFile(filename string, r io.Reader) error {
	w, closer := rs.makePublicObject(filename)
	defer closer()

	_, err := io.Copy(w, r)

	return err
}

func httpErrorWrap(ctx context.Context, w http.ResponseWriter, err error, msg string) {
	log.Errorf(ctx, errors.Wrap(err, msg).Error())
	http.Error(w, msg, http.StatusInternalServerError)
}

// makePublicObject opens a file in the specified client and bucket with the
// given name, and returns a writer to it as well as a closer function. Caller
// must call the closer function when done writing to the file (e.g. using
// defer). The object will be made public upon closing.
func (rs *repyStorer) makePublicObject(filename string) (io.Writer, func()) {
	obj := rs.bucket.Object(filename)
	w := obj.NewWriter(rs.ctx)
	closer := func() {
		w.Close()
		if err := obj.ACL().Set(rs.ctx, storage.AllUsers, storage.RoleReader); err != nil {
			log.Errorf(rs.ctx, "Failed to make %q public: %v", filename, err)
		}
	}
	return w, closer
}

func (rs *repyStorer) setContentType(filename string, contentType string) error {
	obj := rs.bucket.Object(filename)
	if _, err := obj.Update(rs.ctx, storage.ObjectAttrsToUpdate{ContentType: contentType}); err != nil {
		return errors.Wrapf(err, "failed to set content-type for %q to %q", filename, contentType)
	}

	return nil
}

func (rs *repyStorer) fileExists(filename string) (bool, error) {
	obj := rs.bucket.Object(filename)
	_, err := obj.Attrs(rs.ctx)
	switch err {
	case nil:
		return true, nil
	case storage.ErrObjectNotExist:
		return false, nil
	default:
		return false, errors.Wrapf(err, "couldn't check if %q exists", filename)
	}
}

func (rs *repyStorer) writeTimeStamp(filename string, t time.Time) error {
	w, closer := rs.makePublicObject(filename)
	defer closer()
	if _, err := fmt.Fprintf(w, "%s\n", t.UTC().Format(time.UnixDate)); err != nil {
		return errors.Wrapf(err, "couldn't write timestamp %q", filename)
	}
	return nil
}
