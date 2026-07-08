package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/niac-go/internal/sanitize"
)

type sanitizeOptions struct {
	mappingFile string
	domain      string
	location    string
	contact     string
	community   string
	batch       bool
	inputDir    string
	outputDir   string
}

func addSanitizeCommand(root *cobra.Command, _ *serviceOptions) {
	options := new(sanitizeOptions)
	defaults := sanitize.DefaultOptions()

	sanitizeCmd := &cobra.Command{
		Use:   "sanitize <input-walk> <output-walk>",
		Short: "Sanitize SNMP walk files with NiAC-Go branding",
		Long: `Sanitize SNMP walk files by replacing real network data with consistent
NiAC-Go branded data. IP addresses are mapped deterministically so the
same input IP always produces the same output IP.

What is KEPT (not sensitive):
  • Serial numbers
  • MAC addresses
  • Hardware models
  • Interface counts/types
  • VLAN IDs

What is TRANSFORMED (deterministic):
  • IP addresses → 10.0.0.0/8 (NiAC-Go network)
  • Hostnames → niac-<location>-<type>-<number>
  • DNS domains → niac-go.com / niac-go.local
  • Contact info → netadmin@niac-go.com
  • Location strings → NiAC-Go - DC-WEST
  • Community strings → public or niac-go-ro`,
		Example: `  # Sanitize a single walk file
  niac sanitize device.walk device-sanitized.walk

  # Batch mode - sanitize all walks in a directory
  niac sanitize --batch --input-dir walks/ --output-dir sanitized/

  # Use persistent mapping file
  niac sanitize --mapping-file ip-map.json device.walk output.walk`,
		Args: func(_ *cobra.Command, args []string) error {
			if options.batch {
				// Batch mode requires --input-dir and --output-dir
				if options.inputDir == "" || options.outputDir == "" {
					return errors.New("batch mode requires --input-dir and --output-dir")
				}
				return nil
			}
			// Single file mode requires exactly 2 args
			if len(args) != minArgsForConfig {
				return errors.New("requires <input-walk> and <output-walk> arguments")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			return runSanitize(args, options)
		},
	}

	sanitizeCmd.Flags().StringVar(&options.mappingFile, "mapping-file", "", "JSON file to load/save IP mappings")
	sanitizeCmd.Flags().StringVar(&options.domain, "domain", defaults.Domain, "Domain for hostnames and DNS")
	sanitizeCmd.Flags().StringVar(&options.location, "location", defaults.Location, "Default location suffix")
	sanitizeCmd.Flags().StringVar(&options.contact, "contact", defaults.Contact, "Contact email")
	sanitizeCmd.Flags().StringVar(&options.community, "community", defaults.Community, "SNMP community string")
	sanitizeCmd.Flags().BoolVar(&options.batch, "batch", false, "Batch process multiple files")
	sanitizeCmd.Flags().StringVar(&options.inputDir, "input-dir", "", "Input directory for batch mode")
	sanitizeCmd.Flags().StringVar(&options.outputDir, "output-dir", "", "Output directory for batch mode")

	root.AddCommand(sanitizeCmd)
}

// validateSingleFilePaths validates input and output file paths for single file mode.
func validateSingleFilePaths(inputFile, outputFile string) error {
	if pathErr := validateFilePath(inputFile, false); pathErr != nil {
		return fmt.Errorf("invalid input file: %w", pathErr)
	}

	if pathErr := validateFilePath(outputFile, true); pathErr != nil {
		return fmt.Errorf("invalid output file: %w", pathErr)
	}

	return nil
}

// validateBatchPaths validates input and output directory paths for batch mode.
func validateBatchPaths(inputDir, outputDir string) error {
	if dirErr := validateDirPath(inputDir, false); dirErr != nil {
		return fmt.Errorf("invalid input directory: %w", dirErr)
	}

	if dirErr := validateDirPath(outputDir, true); dirErr != nil {
		return fmt.Errorf("invalid output directory: %w", dirErr)
	}

	return nil
}

func runSanitize(args []string, options *sanitizeOptions) error {
	batch := options.batch
	mappingFile := options.mappingFile
	opts := sanitize.Options{
		Domain:    options.domain,
		Location:  options.location,
		Contact:   options.contact,
		Community: options.community,
	}

	// Validate input paths (Fix #67 - Input validation)
	if batch {
		if pathErr := validateBatchPaths(options.inputDir, options.outputDir); pathErr != nil {
			return pathErr
		}
	} else {
		if pathErr := validateSingleFilePaths(args[0], args[1]); pathErr != nil {
			return pathErr
		}
	}

	// Validate mapping file path if provided
	if mappingFile != "" {
		pathErr := validateFilePath(mappingFile, true)
		if pathErr != nil {
			return fmt.Errorf("invalid mapping file path: %w", pathErr)
		}
	}

	mapping := sanitize.NewMapping()
	if mappingFile != "" {
		loadErr := loadMapping(mappingFile, mapping)
		if loadErr != nil && !os.IsNotExist(loadErr) {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not load mapping file: %v\n", loadErr)
		}
	}

	if batch {
		return sanitizeBatch(options.inputDir, options.outputDir, mapping, opts, mappingFile)
	}

	// Single file mode
	inputFile := args[0]
	outputFile := args[1]

	if sanitizeErr := sanitizeFile(inputFile, outputFile, mapping, opts); sanitizeErr != nil {
		return fmt.Errorf("sanitization failed: %w", sanitizeErr)
	}

	if mappingFile != "" {
		if saveErr := saveMapping(mappingFile, mapping); saveErr != nil {
			return fmt.Errorf("failed to save mapping: %w", saveErr)
		}
	}

	fmt.Fprintf(os.Stdout, "✅ Sanitized %s → %s\n", inputFile, outputFile)
	fmt.Fprintf(os.Stdout, "   IPs transformed: %d\n", mapping.Statistics.IPsTransformed)
	fmt.Fprintf(os.Stdout, "   Hostnames transformed: %d\n", mapping.Statistics.HostnamesTransformed)

	return nil
}

func sanitizeBatch(
	inputDir, outputDir string,
	mapping *sanitize.Mapping,
	opts sanitize.Options,
	mappingFile string,
) error {
	mkdirErr := os.MkdirAll(outputDir, 0o750)
	if mkdirErr != nil {
		return fmt.Errorf("failed to create output directory: %w", mkdirErr)
	}

	walkFiles, err := filepath.Glob(filepath.Join(inputDir, "*.walk"))
	if err != nil {
		return fmt.Errorf("failed to list walk files: %w", err)
	}

	if len(walkFiles) == 0 {
		return fmt.Errorf("no .walk files found in %s", inputDir)
	}

	fmt.Fprintf(os.Stdout, "🔍 Found %d walk files\n", len(walkFiles))

	for i, inputFile := range walkFiles {
		basename := filepath.Base(inputFile)
		outputFile := filepath.Join(outputDir, basename)

		fmt.Fprintf(os.Stdout, "[%d/%d] Sanitizing %s...\n", i+1, len(walkFiles), basename)

		if sanitizeErr := sanitizeFile(inputFile, outputFile, mapping, opts); sanitizeErr != nil {
			fmt.Fprintf(os.Stderr, "   ⚠️  Error: %v\n", sanitizeErr)
			continue
		}

		mapping.Statistics.FilesProcessed++
	}

	if mappingFile != "" {
		if saveErr := saveMapping(mappingFile, mapping); saveErr != nil {
			return fmt.Errorf("failed to save mapping: %w", saveErr)
		}
		fmt.Fprintf(os.Stdout, "\n💾 Saved mapping to %s\n", mappingFile)
	}

	fmt.Fprintf(os.Stdout, "\n✅ Batch sanitization complete!\n")
	fmt.Fprintf(os.Stdout, "   Files processed: %d\n", mapping.Statistics.FilesProcessed)
	fmt.Fprintf(os.Stdout, "   IPs transformed: %d\n", mapping.Statistics.IPsTransformed)
	fmt.Fprintf(os.Stdout, "   Hostnames transformed: %d\n", mapping.Statistics.HostnamesTransformed)

	return nil
}

// sanitizeFile reads inputFile, sanitizes it via internal/sanitize, and
// atomically writes the result to outputFile (write to a .tmp sibling,
// then rename) so a failed run never leaves a partial file behind.
func sanitizeFile(inputFile, outputFile string, mapping *sanitize.Mapping, opts sanitize.Options) error {
	content, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	sanitized, _, err := sanitize.Content(content, mapping, opts)
	if err != nil {
		return fmt.Errorf("failed to sanitize content: %w", err)
	}

	return writeSanitizedAtomic(outputFile, sanitized)
}

// sanitizedFileMode is the permission for sanitized walk output: owner
// read/write only, since walks can carry pre-scrub network detail on disk
// during the write.
const sanitizedFileMode = 0o600

// writeSanitizedAtomic writes content to outputFile via a same-directory
// ".tmp" sibling and an atomic rename, so a failed run never leaves a
// partial file. The write and rename go through an os.Root opened on the
// output directory: every leaf operation is confined to that directory by
// the OS (openat2 RESOLVE_BENEATH semantics), which is what makes path
// traversal impossible by construction — the same structural containment
// the library package relies on, and what lets gosec's taint analysis see
// the write target as bounded rather than an arbitrary sink.
func writeSanitizedAtomic(outputFile string, content []byte) error {
	dir := filepath.Dir(outputFile)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("failed to open output directory: %w", err)
	}
	defer func() { _ = root.Close() }()

	tmpLeaf := filepath.Base(outputFile) + ".tmp"
	f, err := root.OpenFile(tmpLeaf, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, sanitizedFileMode)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	if _, writeErr := f.Write(content); writeErr != nil {
		_ = f.Close()
		_ = root.Remove(tmpLeaf)
		return fmt.Errorf("failed to write output file: %w", writeErr)
	}
	if closeErr := f.Close(); closeErr != nil {
		_ = root.Remove(tmpLeaf)
		return fmt.Errorf("failed to close output file: %w", closeErr)
	}

	if renameErr := root.Rename(tmpLeaf, filepath.Base(outputFile)); renameErr != nil {
		_ = root.Remove(tmpLeaf)
		return fmt.Errorf("failed to rename temp file: %w", renameErr)
	}

	return nil
}

func loadMapping(filename string, mapping *sanitize.Mapping) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read mapping file: %w", err)
	}

	unmarshalErr := json.Unmarshal(data, mapping)
	if unmarshalErr != nil {
		return fmt.Errorf("failed to unmarshal mapping: %w", unmarshalErr)
	}
	return nil
}

func saveMapping(filename string, mapping *sanitize.Mapping) error {
	data, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	// Fix #68: Atomic write for mapping file
	tempFile := filename + ".tmp"
	writeErr := os.WriteFile(tempFile, data, 0o600)
	if writeErr != nil {
		return fmt.Errorf("failed to write temp mapping file: %w", writeErr)
	}

	renameErr := os.Rename(tempFile, filename)
	if renameErr != nil {
		return fmt.Errorf("failed to rename mapping file: %w", renameErr)
	}
	return nil
}

// validateFilePath validates file paths to prevent path traversal attacks
// Fix #67: Input validation.
func validateFilePath(path string, allowCreate bool) error {
	if path == "" {
		return errors.New("empty path")
	}

	cleanPath := filepath.Clean(path)

	if err := checkPathTraversal(cleanPath, path); err != nil {
		return err
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if err = validatePathWithinAllowedDirs(absPath, path); err != nil {
		return err
	}

	if !allowCreate {
		return validateExistingFile(absPath, path)
	}

	return nil
}

// checkPathTraversal detects path traversal attempts in the cleaned path.
func checkPathTraversal(cleanPath, originalPath string) error {
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected: %s", originalPath)
	}
	return nil
}

// validatePathWithinAllowedDirs checks if path is within CWD, temp, or home directories.
func validatePathWithinAllowedDirs(absPath, originalPath string) error {
	validPrefixes := buildAllowedPrefixes()

	for _, prefix := range validPrefixes {
		if isPathUnderPrefix(absPath, prefix) {
			return nil
		}
	}

	return fmt.Errorf("path outside allowed directories: %s", originalPath)
}

// buildAllowedPrefixes constructs the list of allowed directory prefixes.
func buildAllowedPrefixes() []string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	prefixes := []string{cwd, os.TempDir()}

	if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
		prefixes = append(prefixes, homeDir)
	}

	return prefixes
}

// isPathUnderPrefix checks if absPath is under the given prefix directory.
func isPathUnderPrefix(absPath, prefix string) bool {
	absPrefix, err := filepath.Abs(prefix)
	if err != nil {
		return false
	}

	pathWithSep := absPath + string(filepath.Separator)
	prefixWithSep := absPrefix + string(filepath.Separator)

	return strings.HasPrefix(pathWithSep, prefixWithSep)
}

// validateExistingFile checks that a file exists and is a regular file (not symlink).
func validateExistingFile(absPath, originalPath string) error {
	info, err := os.Lstat(absPath)
	if err != nil {
		return handleStatError(err, originalPath)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlinks not allowed: %s", originalPath)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", originalPath)
	}

	return nil
}

// handleStatError converts stat errors to user-friendly messages.
func handleStatError(err error, path string) error {
	if os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}
	return fmt.Errorf("cannot access file: %w", err)
}

// validateDirPath validates directory paths
// Fix #67: Input validation.
func validateDirPath(path string, allowCreate bool) error {
	if path == "" {
		return errors.New("empty path")
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for path traversal
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected: %s", path)
	}

	// Get absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if path is within allowed directories
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	validPrefixes := []string{cwd, os.TempDir()}
	if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
		validPrefixes = append(validPrefixes, homeDir)
	}

	isValid := false
	for _, prefix := range validPrefixes {
		absPrefix, absErr := filepath.Abs(prefix)
		if absErr != nil {
			continue
		}
		if strings.HasPrefix(absPath+string(filepath.Separator), absPrefix+string(filepath.Separator)) {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("path outside allowed directories: %s", path)
	}

	// For input dirs, ensure they exist
	if !allowCreate {
		info, statErr := os.Stat(absPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return fmt.Errorf("directory does not exist: %s", path)
			}
			return fmt.Errorf("cannot access directory: %w", statErr)
		}

		if !info.IsDir() {
			return fmt.Errorf("not a directory: %s", path)
		}
	}

	return nil
}
