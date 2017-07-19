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
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"

	"github.com/lutzky/repy"
	"github.com/pkg/errors"
)

const tempFilename = "REPY"

type appEngineLogger struct {
	ctx context.Context
}

func (ael appEngineLogger) Infof(format string, args ...interface{}) {
	log.Infof(ael.ctx, format, args...)
}

func (ael appEngineLogger) Warningf(format string, args ...interface{}) {
	log.Warningf(ael.ctx, format, args...)
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

	jsonObj := client.Bucket(bucket).Object("latest.json")
	wc := jsonObj.NewWriter(ctx)
	defer wc.Close()

	catalog, err := repy.ReadFile(bytes.NewReader(repyBytes), appEngineLogger{ctx})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	enc := json.NewEncoder(wc)

	if err := enc.Encode(catalog); err != nil {
		http.Error(w, errors.Wrap(err, "failed to write latest.json").Error(), http.StatusInternalServerError)
		return
	}

	wc.Close() // Ensure close now so that sharing-with-all-users works

	repyObj := client.Bucket(bucket).Object("latest.repy")
	wc2 := repyObj.NewWriter(ctx)
	defer wc2.Close()
	if _, err := io.Copy(wc2, bytes.NewReader(repyBytes)); err != nil {
		http.Error(w, errors.Wrap(err, "failed to write latest.repy").Error(), http.StatusInternalServerError)
		return
	}

	wc2.Close() // Ensure close now so that sharing-with-all-users works

	if err := repyObj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		http.Error(w, errors.Wrap(err, "Failed to make latest.repy public").Error(), http.StatusInternalServerError)
		return
	}
	if err := jsonObj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		http.Error(w, errors.Wrap(err, "Failed to make latest.json public").Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Success, wrote new REPY files. Bucket is %q\n", bucket)
}
