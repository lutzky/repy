package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/lutzky/repy"
)

var (
	inputFile  = flag.String("input_file", "/dev/stdin", "File to read for input")
	outputFile = flag.String("output_file", "/dev/stdout", "File to read for output")
)

func main() {
	flag.Parse()

	catalog, err := readREPYFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read REPY input: %v\n", err)
		os.Exit(1)
	}

	if err := writeJSONFile(*outputFile, catalog); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode JSON output: %v\n", err)
	}
}

func readREPYFile(filename string) (*repy.Catalog, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return repy.ReadFile(f, repy.GLogger{})
}

func writeJSONFile(filename string, catalog *repy.Catalog) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(catalog)
}
