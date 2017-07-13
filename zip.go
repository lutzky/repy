package repy

import (
	"archive/zip"
	"bytes"
	"io"

	"github.com/pkg/errors"
)

// RepFileURL points at the official Technion REPFILE.zip.
const RepFileURL = "http://ug.technion.ac.il/rep/REPFILE.zip"

// REPYFileName is the name of the file to extract from the zip file at
// RepFileURL.
const REPYFileName = "REPY"

// ExtractFromZip reads the ZIP file from r and returns the bytes of the
// REPY file extracted from it.
func ExtractFromZip(r io.Reader) ([]byte, error) {
	var repyBuffer, zipBuffer bytes.Buffer

	size, err := io.Copy(&zipBuffer, r)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing zip archive")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipBuffer.Bytes()), size)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing zip archive")
	}

	for _, f := range zipReader.File {
		if f.Name == REPYFileName {
			of, err := f.Open()
			if err != nil {
				return nil, errors.Wrapf(err, "error extracting file %q from zip archive", f.Name)
			}
			defer of.Close()

			_, err = io.Copy(&repyBuffer, of)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract %q", f.Name)
			}
			return repyBuffer.Bytes(), nil
		}
	}

	return nil, errors.Errorf("didn't find a file called %q in zip archive", REPYFileName)
}
