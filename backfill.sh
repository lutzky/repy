#!/bin/bash

set -e

PATH="$GOPATH/bin:$PATH"
GLOB="*.repy"
PROJECT="$(gcloud config get-value project)"
BUCKET="$PROJECT.appspot.com"

cd "$(dirname "$0")"/cmd/repy-convert/
echo "Installing repy-convert"
go install
if ! hash repy-convert 2>/dev/null; then
	echo "Failed to install repy-convert in GOPATH."
fi

echo -n "Download all REPY files, convert to JSON and re-upload? [y/N] "
read response
if [[ $response != "y" ]]; then
	exit 0
fi

DIR="$(mktemp -d repy-backfill.XXXXXX --tmpdir)"

echo "Performing processing in $DIR"
cd "$DIR"

gsutil -m cp "gs://${BUCKET}/${GLOB}" .

for repy_file in *.repy; do
	echo "Processing $repy_file"
	repy-convert \
		-input_file "$repy_file" \
		-output_file "${repy_file%*.repy}.json"
done

gsutil -m cp *.json "gs://${BUCKET}/"

rm -rf ${DIR}
