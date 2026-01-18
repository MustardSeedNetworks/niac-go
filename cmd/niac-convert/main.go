// Package main implements a converter from Java NIAC DSL config format to YAML
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/krisarmstrong/niac-go/internal/converter"
)

// ErrNoCfgFilesFound is returned when no .cfg files are found in the directory.
var ErrNoCfgFilesFound = errors.New("no .cfg files found")

func main() {
	inputFile := flag.String("input", "", "Input Java DSL config file (.cfg)")
	outputFile := flag.String("output", "", "Output YAML config file (optional, defaults to <input>.yaml)")
	batchDir := flag.String("batch", "", "Convert all .cfg files in directory")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()
	logger := slog.Default()

	if *inputFile == "" && *batchDir == "" {
		fmt.Fprintf(os.Stderr, "Usage: niac-convert -input <file.cfg> [-output <file.yaml>] [-v]\n")
		fmt.Fprintf(os.Stderr, "   or: niac-convert -batch <directory> [-v]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Batch conversion mode
	if *batchDir != "" {
		err := convertBatch(*batchDir, *verbose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		return
	}

	// Single file conversion mode
	if *outputFile == "" {
		base := filepath.Base(*inputFile)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]
		*outputFile = name + ".yaml"
	}

	if *verbose {
		logger.Info("Converting", "input", *inputFile, "output", *outputFile)
	}

	err := converter.ConvertFile(*inputFile, *outputFile, *verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		logger.Info("Conversion successful!")
	}
}

func convertBatch(dir string, verbose bool) error {
	logger := slog.Default()
	files, err := filepath.Glob(filepath.Join(dir, "*.cfg"))
	if err != nil {
		return fmt.Errorf("error finding .cfg files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("%w in %s", ErrNoCfgFilesFound, dir)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Found %d config files to convert\n", len(files))

	for _, file := range files {
		base := filepath.Base(file)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]
		output := filepath.Join(dir, name+".yaml")

		if verbose {
			logger.Info("Converting", "input", file, "output", output)
		}

		if convertErr := converter.ConvertFile(file, output, verbose); convertErr != nil {
			fmt.Fprintf(os.Stderr, "Error converting %s: %v\n", file, convertErr)

			continue
		}

		if verbose {
			logger.Info("Converted successfully", "file", base)
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Batch conversion complete\n")

	return nil
}
