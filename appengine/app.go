package main

import (
	"bytes"
	"encoding/json"
	"net/http"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
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
	client := urlfetch.Client(ctx)
	resp, err := client.Get(repy.RepFileURL)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to download")
	}
	defer resp.Body.Close()

	return repy.ExtractFromZip(resp.Body)
}

func init() {
	http.HandleFunc("/", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	repyReader, err := DownloadREPYZip(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	catalog, err := repy.ReadFile(bytes.NewReader(repyReader), appEngineLogger{ctx})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	enc := json.NewEncoder(w)

	if err := enc.Encode(catalog); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
