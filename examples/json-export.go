package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/golang/glog"
	"github.com/lutzky/repy"
	"github.com/pkg/errors"
)

var (
	inputFile       = flag.String("input_file", "", "REPY file to read")
	repfileURL      = flag.String("repfile_url", "http://ug.technion.ac.il/rep/REPFILE.zip", "URL to REPFILE.zip")
	desiredREPYFile = flag.String("desired_repy_file", "REPY", "Desired name of REPY file inside REPFILE.zip")
)

const tempFilename = "REPY"

func DownloadREPYZip(url, dir string) error {
	tmp, err := ioutil.TempFile("", "repy")
	if err != nil {
		return errors.Wrap(err, "Couldn't allocate temporary file")
	}
	defer os.Remove(tmp.Name())

	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrapf(err, "Failed to download")
	}
	defer resp.Body.Close()
	size, err := io.Copy(tmp, resp.Body)

	zr, err := zip.NewReader(tmp, size)
	if err != nil {
		return errors.Wrapf(err, "Error parsing zip archive")
	}

	for _, f := range zr.File {
		if f.Name == *desiredREPYFile {
			of, err := f.Open()
			if err != nil {
				return errors.Wrapf(err, "Error extracting file %q from zip archive", f.Name)
			}
			defer of.Close()

			out, err := os.Create(path.Join(dir, tempFilename))
			if err != nil {
				return errors.Wrapf(err, "Failed to allocate temporary file in %q", dir)
			}

			_, err = io.Copy(out, of)
			if err != nil {
				return errors.Wrapf(err, "Failed to extract %q to %q", f.Name, dir)
			}
			return nil
		}
	}

	return errors.Errorf("Didn't find a file called %q in zip archive", *desiredREPYFile, url)
}

func main() {
	flag.Parse()

	if *inputFile == "" {
		tmpdir, err := ioutil.TempDir("", "repy")
		if err != nil {
			glog.Exitf("Failed to allocate temp directory: %v", err)
		}
		defer os.RemoveAll(tmpdir)

		if err := DownloadREPYZip(*repfileURL, tmpdir); err != nil {
			glog.Exitf("Error downloading REPY from %s: %v", *repfileURL, err)
		}
		*inputFile = path.Join(tmpdir, "REPY")
	}

	catalog, err := repy.ReadFile(*inputFile)
	if err != nil {
		glog.Exitf("Error reading %q: %v", *inputFile, err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	if err := enc.Encode(catalog); err != nil {
		glog.Exitf("Error serializing JSON: %v", err)
	}
}
