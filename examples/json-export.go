package main

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/lutzky/repy"
)

type Bla struct {
	X, Y int
}

var inputFile = flag.String("input_file", "", "REPY file to read")

func main() {
	flag.Parse()

	if *inputFile == "" {
		glog.Exit("Missing required flag -input_file")
	}
	catalog, err := repy.ReadFile(*inputFile)
	if err != nil {
		glog.Exitf("Error reading %q: %v", *inputFile, err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	var f repy.Faculty
	f = (*catalog)[0]

	if err := enc.Encode(&f); err != nil {
		glog.Exitf("Error serializing JSON: %v", err)
	}
}
