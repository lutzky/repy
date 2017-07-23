package repyapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"cloud.google.com/go/storage"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/file"
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
		return nil, errors.Wrapf(err, "Failed to download")
	}
	defer resp.Body.Close()

	return repy.ExtractFromZip(resp.Body)
}

func init() {
	http.HandleFunc("/update", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	bucket, err := file.DefaultBucketName(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	repyBytes, err := DownloadREPYZip(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	repyObj := client.Bucket(bucket).Object("latest.repy")
	wc2 := repyObj.NewWriter(ctx)
	defer func() {
		wc2.Close()
		if err := makePublic(ctx, repyObj); err != nil {
			httpErrorWrap(w, err, "Failed to make latest.repypublic")
			return
		}
	}()

	if _, err := io.Copy(wc2, bytes.NewReader(repyBytes)); err != nil {
		http.Error(w, errors.Wrap(err, "failed to write latest.repy").Error(), http.StatusInternalServerError)
		return
	}

	jsonObj := client.Bucket(bucket).Object("latest.json")
	wc := jsonObj.NewWriter(ctx)
	defer func() {
		wc.Close()
		if err := makePublic(ctx, jsonObj); err != nil {
			httpErrorWrap(w, err, "Failed to make latest.json public")
			return
		}
	}()

	parseLogObj := client.Bucket(bucket).Object("latest.parse.log")
	parseLogWriter := parseLogObj.NewWriter(ctx)
	defer func() {
		parseLogWriter.Close()
		if err := makePublic(ctx, parseLogObj); err != nil {
			httpErrorWrap(w, err, "Failed to make parselog public")
		}
	}()

	catalog, err := repy.ReadFile(bytes.NewReader(repyBytes), writerLogger{parseLogWriter})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	enc := json.NewEncoder(wc)

	if err := enc.Encode(catalog); err != nil {
		http.Error(w, errors.Wrap(err, "failed to write latest.json").Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Success, wrote new REPY files. Bucket is %q\n", bucket)
}

func makePublic(ctx context.Context, obj *storage.ObjectHandle) error {
	return obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader)
}

func httpErrorWrap(w http.ResponseWriter, err error, format string, args ...interface{}) {
	http.Error(w, errors.Wrapf(err, format, args...).Error(), http.StatusInternalServerError)
}
