package corpusreplay

import (
	"compress/bzip2"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// Statuses the corpus layer adds to the agent's replay statuses.
const (
	// StatusSourceError means the recording could not be read or expanded.
	StatusSourceError = "source_error"
	// StatusCanceled means the run stopped before this recording replayed.
	StatusCanceled = "canceled"
)

// maxExpandedBytes caps a decompressed .bz2 recording. The largest qualified
// recording expands to well under this; anything bigger is not a recording.
const maxExpandedBytes = 1 << 30

// ErrExpandedTooLarge reports a compressed recording that expands past
// maxExpandedBytes.
var ErrExpandedTooLarge = errors.New("compressed recording expands past the size cap")

// Options controls a corpus run.
type Options struct {
	// Jobs is how many recordings replay concurrently; 0 means GOMAXPROCS.
	Jobs int
	// UDPSmoke adds the loopback client smoke test to every replay.
	UDPSmoke bool
	// Timing keeps each receipt's wall time. Off by default so that two runs
	// over the same corpus produce byte-identical output.
	Timing bool
}

// Result is one recording's replay and how it compares with its baseline.
type Result struct {
	Path          string `json:"path"`
	EffectivePath string `json:"effective_path"`
	// Error explains a StatusSourceError.
	Error string `json:"error,omitempty"`
	// SourceChanged means the file's SHA-256 no longer matches the digest it
	// qualified with, so its baseline may not apply.
	SourceChanged bool `json:"source_changed,omitempty"`
	// Drift lists every baseline field the replay disagrees with, as
	// "field: baseline X, replay Y", in manifest column order.
	Drift   []string           `json:"drift,omitempty"`
	Receipt snmp.ReplayReceipt `json:"receipt"`
}

// Run replays rows and returns one result per row, sorted by Path. Results do
// not depend on Jobs or on scheduling.
func Run(ctx context.Context, rows []Row, opts Options) []Result {
	jobs := opts.Jobs
	if jobs <= 0 {
		jobs = runtime.GOMAXPROCS(0)
	}
	results := make([]Result, len(rows))
	work := make(chan int)
	var wg sync.WaitGroup
	for range min(jobs, max(len(rows), 1)) {
		wg.Go(func() {
			for i := range work {
				results[i] = replayRow(ctx, rows[i], opts)
			}
		})
	}
	for i := range rows {
		work <- i
	}
	close(work)
	wg.Wait()

	sort.SliceStable(results, func(a, b int) bool { return results[a].Path < results[b].Path })

	return results
}

func replayRow(ctx context.Context, row Row, opts Options) Result {
	result := Result{Path: row.Path, EffectivePath: row.EffectivePath}
	if ctx.Err() != nil {
		result.Receipt = snmp.ReplayReceipt{Status: StatusCanceled, FailureTypes: map[string]int{}}

		return result
	}

	file := row.File()
	digest, err := fileSHA256(file)
	if err != nil {
		return sourceError(result, err)
	}
	result.SourceChanged = row.SourceSHA256 != "" && digest != row.SourceSHA256

	replayPath, cleanup, err := expandedRecording(file)
	if err != nil {
		return sourceError(result, err)
	}
	defer cleanup()

	result.Receipt = snmp.ReplayRecording(replayPath, snmp.ReplayOptions{UDPSmoke: opts.UDPSmoke})
	if !opts.Timing {
		result.Receipt.Milliseconds = 0
	}
	result.Drift = Compare(row.Baseline, result.Receipt, opts.UDPSmoke)

	return result
}

func sourceError(result Result, err error) Result {
	result.Error = err.Error()
	result.Receipt = snmp.ReplayReceipt{Status: StatusSourceError, FailureTypes: map[string]int{}}

	return result
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open recording: %w", err)
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash recording: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// expandedRecording returns a path the walk reader can load. The reader does
// not decompress, so a .bz2 recording is expanded to a private temporary file
// that cleanup removes; any other recording is returned as is.
func expandedRecording(path string) (string, func(), error) {
	noop := func() {}
	if !strings.HasSuffix(path, ".bz2") {
		return path, noop, nil
	}
	source, err := os.Open(path)
	if err != nil {
		return "", noop, fmt.Errorf("open recording: %w", err)
	}
	defer func() { _ = source.Close() }()

	dir, err := os.MkdirTemp("", "niac-corpus-replay-")
	if err != nil {
		return "", noop, fmt.Errorf("create expansion directory: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	target := filepath.Join(dir, strings.TrimSuffix(filepath.Base(path), ".bz2"))
	if err = expandBzip2(source, target); err != nil {
		cleanup()

		return "", noop, err
	}

	return target, cleanup, nil
}

func expandBzip2(source io.Reader, target string) error {
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create expanded recording: %w", err)
	}
	written, err := io.CopyN(out, bzip2.NewReader(source), maxExpandedBytes+1)
	closeErr := out.Close()
	switch {
	case err != nil && !errors.Is(err, io.EOF):
		return fmt.Errorf("expand recording: %w", err)
	case written > maxExpandedBytes:
		return ErrExpandedTooLarge
	case closeErr != nil:
		return fmt.Errorf("write expanded recording: %w", closeErr)
	}

	return nil
}
