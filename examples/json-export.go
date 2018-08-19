package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"

	"github.com/golang/glog"
	"github.com/lutzky/repy"
)

var (
	inputFile       = flag.String("input_repy_file", "", "REPY file to read")
	repfileURL      = flag.String("repfile_url", repy.RepFileURL, "URL to REPFILE.zip")
	desiredREPYFile = flag.String("desired_repy_file", "REPY", "Desired name of REPY file inside REPFILE.zip")
)

func main() {
	flag.Parse()

	var repyReader io.Reader
	var err error

	if *inputFile == "" {
		resp, err := http.Get(*repfileURL)
		if err != nil {
			glog.Exitf("Error downloading REPY from %s: %v", *repfileURL, err)
		}

		if zipBytes, err := repy.ExtractFromZip(resp.Body); err != nil {
			glog.Exitf("Failed to extract REPY from zip in %s: %v", *repfileURL, err)
		} else {
			repyReader = bytes.NewReader(zipBytes)
		}
	} else {
		f, err := os.Open(*inputFile)
		if err != nil {
			glog.Exitf("Failed to open %s: %v", *inputFile, err)
		}
		repyReader = f
		defer f.Close()
	}

	catalog, err := repy.ReadFile(repyReader, repy.GLogger{})
	if err != nil {
		glog.Exitf("Error reading %q: %v", *inputFile, err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	if err := enc.Encode(catalog); err != nil {
		glog.Exitf("Error serializing JSON: %v", err)
	}
}
