// Package main implements a converter from Java NIAC DSL config format to YAML
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/krisarmstrong/niac-go/internal/converter"
)

func main() {
	inputFile := flag.String("input", "", "Input Java DSL config file (.cfg)")
	outputFile := flag.String("output", "", "Output YAML config file (optional, defaults to <input>.yaml)")
	batchDir := flag.String("batch", "", "Convert all .cfg files in directory")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	if *inputFile == "" && *batchDir == "" {
		fmt.Fprintf(os.Stderr, "Usage: niac-convert -input <file.cfg> [-output <file.yaml>] [-v]\n")
		fmt.Fprintf(os.Stderr, "   or: niac-convert -batch <directory> [-v]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Batch conversion mode
	if *batchDir != "" {
		if err := convertBatch(*batchDir, *verbose); err != nil {
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
		fmt.Printf("Converting %s -> %s\n", *inputFile, *outputFile)
	}

	if err := converter.ConvertFile(*inputFile, *outputFile, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Println("Conversion successful!")
	}
}

func convertBatch(dir string, verbose bool) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.cfg"))
	if err != nil {
		return fmt.Errorf("error finding .cfg files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no .cfg files found in %s", dir)
	}

	fmt.Printf("Found %d config files to convert\n", len(files))

	for _, file := range files {
		base := filepath.Base(file)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]
		output := filepath.Join(dir, name+".yaml")

		if verbose {
			fmt.Printf("Converting %s -> %s\n", file, output)
		}

		if err := converter.ConvertFile(file, output, verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Error converting %s: %v\n", file, err)
			continue
		}

		if verbose {
			fmt.Printf("  ✓ %s\n", base)
		}
	}

	fmt.Printf("Batch conversion complete\n")
	return nil
}
