# repy

[![Build Status](https://travis-ci.org/lutzky/repy.svg?branch=master)](https://travis-ci.org/lutzky/repy)
[![Coverage Status](https://coveralls.io/repos/github/lutzky/repy/badge.svg?branch=master)](https://coveralls.io/github/lutzky/repy?branch=master)

This is a parsing library for Technion REPY files, written in Go. The intent is
to provide better-tested and more robust code, focused entirely on parsing the
REPY file, and providing it in a usable format for other software, replacing
the aging [ttime](http://lutzky.github.io/ttime).

## Testing

The `testdata` directory contains pairs of files:

* `FILENAME.repy` is a REPY file in the cp862 encoding (vim: `:e FILENAME.repy ++enc=cp862`). While this file represents an entire "catalog" (faculties and courses), some files only have one course (in one faculty), such as `testdata/course_statistics.repy`.
* `FILENAME.json` is a JSON file representing the parsed REPY file.

Running `go test -v` will parse the REPY files and compare them against output. Running `go test -update` will update the `.json` files with the actual output; use this when adding new REPY files, or when the difference in output is otherwise known-good.

## Running example

Example usage:
```shell
go run examples/json-export.go -input_file REPY
```

## On AppEngine

The `appengine` directory contains a Google AppEngine app built to poll the Technion servers for the latest REPY file and make a (cached) parsed JSON version available for download.

Running a test instance on google cloud shell:

```shell
go get github.com/lutzky/repy
gopath=~/gopath # Or wherever your gopath is
cd ${gopath}/src/github.com/lutzky/repy/appengine
dev_appserver.py --default_gcs_bucket_name staging.repy-176217.appspot.com app.yaml
```

Afterwards, browse to the preview URL at the `/update` endpoint (http://0.0.0.0:8080/update if running locally). This should write the files to the *staging* file bucket.

If satisfied, deploy to production:

```shell
gcloud app deploy
```
