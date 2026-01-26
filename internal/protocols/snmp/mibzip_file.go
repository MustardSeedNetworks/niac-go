package snmp

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IsMibZipFile checks if a file is in MibZip format.
func IsMibZipFile(filename string) (bool, error) {
	cleanPath := filepath.Clean(filename)

	file, err := os.Open(cleanPath)
	if err != nil {
		return false, fmt.Errorf("failed to open file: %w", err)
	}

	defer func() { _ = file.Close() }()

	magic := make([]byte, len(mibzipMagic))

	n, err := file.Read(magic)
	if err != nil || n < len(mibzipMagic) {
		return false, nil
	}

	return string(magic) == mibzipMagic, nil
}

// ParseMibZipFile reads and expands a MibZip file.
func ParseMibZipFile(filename string) ([]WalkEntry, error) {
	// Security: validate path
	cleanPath := filepath.Clean(filename)
	if strings.Contains(cleanPath, "..") {
		return nil, ErrDirectoryTraversal
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mibzip file: %w", err)
	}

	reader, err := NewMibZipReader(data)
	if err != nil {
		return nil, err
	}

	return reader.Expand()
}

// CompressMibZipFile compresses a walk file to MibZip format.
func CompressMibZipFile(inputFile, outputFile string) error {
	entries, parseErr := ParseWalkFile(inputFile)
	if parseErr != nil {
		return fmt.Errorf("failed to parse walk file: %w", parseErr)
	}

	writer := NewMibZipWriter()

	compressErr := writer.Compress(entries)
	if compressErr != nil {
		return fmt.Errorf("failed to compress: %w", compressErr)
	}

	// SECURITY FIX #163: Create file with restricted permissions (owner-only)
	outFile, openErr := os.OpenFile(
		filepath.Clean(outputFile),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o600,
	)
	if openErr != nil {
		return fmt.Errorf("failed to create output file: %w", openErr)
	}

	defer func() { _ = outFile.Close() }()

	bufWriter := bufio.NewWriter(outFile)

	_, writeErr := writer.WriteTo(bufWriter)
	if writeErr != nil {
		return fmt.Errorf("failed to write: %w", writeErr)
	}

	flushErr := bufWriter.Flush()
	if flushErr != nil {
		return fmt.Errorf("failed to flush buffer: %w", flushErr)
	}

	return nil
}
