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

	"google.golang.org/appengine"
	"google.golang.org/appengine/file"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"

	"github.com/lutzky/repy"
	"github.com/pkg/errors"
)

const tempFilename = "REPY"

type writerLogger struct {
	w io.Writer
}

func (wl writerLogger) Infof(format string, args ...interface{}) {
	fmt.Fprint(wl.w, "I ")
	fmt.Fprintf(wl.w, format, args...)
	fmt.Fprint(wl.w, "\n")
}

func (wl writerLogger) Warningf(format string, args ...interface{}) {
	fmt.Fprint(wl.w, "W ")
	fmt.Fprintf(wl.w, format, args...)
	fmt.Fprint(wl.w, "\n")
}

func DownloadREPYZip(ctx context.Context) ([]byte, error) {
	resp, err := urlfetch.Client(ctx).Get(repy.RepFileURL)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to download %q", repy.RepFileURL)
	}
	defer resp.Body.Close()

	return repy.ExtractFromZip(resp.Body)
}

func init() {
	http.HandleFunc("/update", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	bucketName, err := file.DefaultBucketName(ctx)
	if err != nil {
		httpErrorWrap(ctx, w, err, "Failed to get default bucket name")
		return
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		httpErrorWrap(ctx, w, err, "Failed to get storage client")
		return
	}

	bucket := client.Bucket(bucketName)

	repyBytes, err := DownloadREPYZip(ctx)
	if err != nil {
		httpErrorWrap(ctx, w, err, "Failed to download REPY zip")
		return
	}

	repySHA1Sum := sha1.Sum(repyBytes)

	log.Infof(ctx, "REPY SHA1SUM: %x", repySHA1Sum)

	repyWriter, repyCloser := makePublicObject(ctx, bucket, "latest.repy")
	defer repyCloser()

	if _, err := io.Copy(repyWriter, bytes.NewReader(repyBytes)); err != nil {
		httpErrorWrap(ctx, w, err, "Failed to write latest.repy")
		return
	}

	historicREPYWriter, historicREPYCloser := makePublicObject(ctx, bucket, fmt.Sprintf("%x.repy", repySHA1Sum))
	defer historicREPYCloser()

	if _, err := io.Copy(historicREPYWriter, bytes.NewReader(repyBytes)); err != nil {
		httpErrorWrap(ctx, w, err, "Failed to write historic %x.repy", repySHA1Sum)
		return
	}

	parseLogWriter, parseLogCloser := makePublicObject(ctx, bucket, "latest.parse.log")
	defer parseLogCloser()

	catalog, err := repy.ReadFile(bytes.NewReader(repyBytes), writerLogger{parseLogWriter})
	if err != nil {
		fmt.Fprintf(parseLogWriter, "Read returned error: %v\n", err)
		httpErrorWrap(ctx, w, err, "Failed to read catalog")
		return
	}

	jsonWriter, jsonCloser := makePublicObject(ctx, bucket, "latest.json")
	defer jsonCloser()
	enc := json.NewEncoder(jsonWriter)

	if err := enc.Encode(catalog); err != nil {
		httpErrorWrap(ctx, w, err, "Failed to write latest.json")
		return
	}

	log.Infof(ctx, "Successfully parsed REPY")
	fmt.Fprintf(w, "Success")
}

func httpErrorWrap(ctx context.Context, w http.ResponseWriter, err error, format string, args ...interface{}) {
	wrapedErr := errors.Wrapf(err, format, args...)
	log.Errorf(ctx, wrapedErr.Error())
	http.Error(w, wrapedErr.Error(), http.StatusInternalServerError)
}

// makePublicObject opens a file in the specified client and bucket with the
// given name, and returns a writer to it as well as a closer function. Caller
// must call the closer function when done writing to the file (e.g. using
// defer). The object will be made public upon closing.
func makePublicObject(ctx context.Context, bucket *storage.BucketHandle, filename string) (io.Writer, func()) {
	obj := bucket.Object(filename)
	w := obj.NewWriter(ctx)
	closer := func() {
		w.Close()
		if err := obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
			log.Errorf(ctx, "Failed to make %q public: %v", filename, err)
		}
	}
	return w, closer
}
