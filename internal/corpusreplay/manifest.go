// Package corpusreplay replays a corpus of SNMP recordings through NIAC's
// walk readers and agent, and compares each result with the receipt that
// qualified the recording.
//
// The corpus is niac-demo-catalog's qualified recording selection: the
// recordings under walks/ and the manifest runtime/niac/qualified.csv that
// names each one, its SHA-256 and the replay counters it qualified with. The
// recordings are licensed variously and stay in the catalog; nothing here
// ships or copies them.
package corpusreplay

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Catalog layout, relative to a niac-demo-catalog checkout.
const (
	ManifestPath  = "runtime/niac/qualified.csv"
	RecordingsDir = "walks"
)

// CatalogDirEnv names an existing niac-demo-catalog checkout. It is the
// variable scripts/sync-demo-catalog.sh already reads.
const CatalogDirEnv = "NIAC_DEMO_CATALOG_DIR"

// Manifest columns the replay needs. Every column is also kept as the
// baseline the replay is compared against.
const (
	colOriginalPath  = "original_path"
	colEffectivePath = "effective_path"
	colSourceSHA256  = "source_sha256"
)

// firstDataLine is the 1-based file line of the first row after the header.
const firstDataLine = 2

var (
	// ErrCatalogMissing reports a catalog directory without the qualified
	// manifest. Callers that run where the corpus is optional skip on it.
	ErrCatalogMissing = errors.New("qualified recording manifest not found")
	// ErrManifestColumn reports a manifest without a required column.
	ErrManifestColumn = errors.New("manifest is missing a required column")
	// ErrUnsafePath reports a manifest path that leaves the recordings tree.
	ErrUnsafePath = errors.New("manifest path is not local to the recordings directory")
)

// Row is one recording to replay.
type Row struct {
	// Path is the recording's stable identity: the manifest's original_path,
	// or the file name given on the command line.
	Path string
	// EffectivePath is the file actually replayed, relative to Root. A
	// repaired recording replays from repaired/ under its original identity.
	EffectivePath string
	// Root is the directory EffectivePath is relative to.
	Root string
	// SourceSHA256 is the qualified digest of the effective file's bytes, or
	// empty when there is nothing to check against.
	SourceSHA256 string
	// Baseline holds the manifest's columns by name; nil without a manifest.
	Baseline map[string]string
}

// File is the path of the recording on disk.
func (r Row) File() string { return filepath.Join(r.Root, filepath.FromSlash(r.EffectivePath)) }

// LocateManifest returns the manifest path inside catalog, or
// ErrCatalogMissing when the catalog or its manifest is absent.
func LocateManifest(catalog string) (string, error) {
	if catalog == "" {
		return "", fmt.Errorf("%w: no catalog directory given (set %s)", ErrCatalogMissing, CatalogDirEnv)
	}
	manifest := filepath.Join(catalog, filepath.FromSlash(ManifestPath))
	info, err := os.Stat(manifest)
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrCatalogMissing, manifest)
	}

	return manifest, nil
}

// LoadCatalog reads the qualified manifest of a catalog checkout.
func LoadCatalog(catalog string) ([]Row, error) {
	manifest, err := LocateManifest(catalog)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(manifest)
	if err != nil {
		return nil, fmt.Errorf("open manifest: %w", err)
	}
	defer func() { _ = file.Close() }()

	// The walk reader refuses any path that still contains "..", so a catalog
	// named relative to a sibling checkout must be made absolute first.
	root, err := filepath.Abs(filepath.Join(catalog, RecordingsDir))
	if err != nil {
		return nil, fmt.Errorf("resolve recordings directory: %w", err)
	}

	return ReadManifest(file, root)
}

// ReadManifest parses a qualified.csv stream. Recording paths resolve against
// root and must stay inside it.
func ReadManifest(reader io.Reader, root string) ([]Row, error) {
	records, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrManifestColumn, colEffectivePath)
	}
	header := records[0]
	present := make(map[string]bool, len(header))
	for _, name := range header {
		present[strings.TrimSpace(name)] = true
	}
	for _, required := range []string{colOriginalPath, colEffectivePath} {
		if !present[required] {
			return nil, fmt.Errorf("%w: %s", ErrManifestColumn, required)
		}
	}

	rows := make([]Row, 0, len(records)-1)
	for line, record := range records[1:] {
		row, rowErr := manifestRow(header, record, root)
		if rowErr != nil {
			return nil, fmt.Errorf("manifest line %d: %w", line+firstDataLine, rowErr)
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func manifestRow(header, record []string, root string) (Row, error) {
	fields := make(map[string]string, len(header))
	for i, name := range header {
		if i < len(record) {
			fields[strings.TrimSpace(name)] = record[i]
		}
	}
	effective := fields[colEffectivePath]
	if effective == "" || !filepath.IsLocal(filepath.FromSlash(effective)) {
		return Row{}, fmt.Errorf("%w: %q", ErrUnsafePath, effective)
	}
	row := Row{
		Path:          fields[colOriginalPath],
		EffectivePath: effective,
		Root:          root,
		SourceSHA256:  strings.ToLower(fields[colSourceSHA256]),
		Baseline:      fields,
	}
	if row.Path == "" {
		row.Path = effective
	}

	return row, nil
}

// FileRows builds rows for recordings named directly, with no manifest and so
// nothing to compare against.
func FileRows(paths []string) ([]Row, error) {
	rows := make([]Row, 0, len(paths))
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", path, err)
		}
		rows = append(rows, Row{
			Path:          filepath.ToSlash(path),
			EffectivePath: filepath.Base(abs),
			Root:          filepath.Dir(abs),
		})
	}

	return rows, nil
}
