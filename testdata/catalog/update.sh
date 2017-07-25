#!/bin/bash

set -e

cd $(dirname $0)

for repyfile in *.repy; do
  jsonfile="$(basename "$repyfile" .repy).json"
  echo "$repyfile -> $jsonfile"
  go run ../../examples/json-export.go --input_file "$repyfile" > "$jsonfile"
done

