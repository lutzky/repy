package repyapp

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"cloud.google.com/go/storage"

	"golang.org/x/net/context"
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

func DownloadREPYZip(ctx context.Context) ([]byte, error) {
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

	destinations := []struct {
		filename    string
		contentType string
		data        []byte
	}{
		{fmt.Sprintf("%x.repy", rs.sha1sum), "text/plain; charset=cp862", rs.data},
		{fmt.Sprintf("%x.txt", rs.sha1sum), "text/plain; charset=iso8859-8", repyBytesISO8859_8},
		{"latest.repy", "text/plain; charset=cp862", rs.data},
	}

	for _, dest := range destinations {
		if err := rs.copyToFile(dest.filename, bytes.NewReader(dest.data)); err != nil {
			return errors.Wrapf(err, "failed to write %q", dest.filename)
		}
		if err := rs.setContentType(dest.filename, dest.contentType); err != nil {
			return err
		}
	}

	return nil
}

func (rs *repyStorer) parseJSONAndWrite() error {
	parseLogWriter, parseLogCloser := rs.makePublicObject("latest.parse.log")
	defer parseLogCloser()

	catalog, err := repy.ReadFile(bytes.NewReader(rs.data), writerlogger.Logger{parseLogWriter})
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

	repyBytes, err := DownloadREPYZip(ctx)
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
