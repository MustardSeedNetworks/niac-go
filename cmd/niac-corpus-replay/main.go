// Command niac-corpus-replay replays SNMP recordings through NIAC's walk
// readers and SNMP agent and reports, per recording, whether every recorded
// binding reaches the wire intact.
//
// With -catalog it replays niac-demo-catalog's qualified selection
// (runtime/niac/qualified.csv) and compares each receipt with the one the
// recording qualified with. Given file arguments instead, it replays those
// files with nothing to compare against. See docs/CORPUS_REPLAY.md.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/corpusreplay"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// Exit codes.
const (
	exitOK      = 0
	exitFailed  = 1
	exitUsage   = 2
	filePrivate = 0o600
)

type options struct {
	catalog     string
	match       string
	limit       int
	jobs        int
	smoke       bool
	timing      bool
	out         string
	summary     string
	skipMissing bool
	verbose     bool
	files       []string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseFlags(args, stderr)
	if err != nil {
		return exitUsage
	}
	if !opts.verbose {
		logging.SetOutput(io.Discard)
		slog.SetDefault(slog.New(slog.DiscardHandler))
	}

	rows, err := loadRows(opts)
	if errors.Is(err, corpusreplay.ErrCatalogMissing) && opts.skipMissing {
		_, _ = fmt.Fprintf(stderr, "niac-corpus-replay: skipped: %v\n", err)

		return exitOK
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "niac-corpus-replay: %v\n", err)

		return exitUsage
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	results := corpusreplay.Run(ctx, rows, corpusreplay.Options{
		Jobs: opts.jobs, UDPSmoke: opts.smoke, Timing: opts.timing,
	})
	summary := corpusreplay.Summarize(results)

	if err = writeOutputs(opts, stdout, results, summary); err != nil {
		_, _ = fmt.Fprintf(stderr, "niac-corpus-replay: %v\n", err)

		return exitFailed
	}
	_ = summary.WriteText(stderr)
	if !summary.OK() {
		return exitFailed
	}

	return exitOK
}

func parseFlags(args []string, stderr io.Writer) (options, error) {
	var opts options
	fs := flag.NewFlagSet("niac-corpus-replay", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&opts.catalog, "catalog", os.Getenv(corpusreplay.CatalogDirEnv),
		"niac-demo-catalog checkout to replay (default $"+corpusreplay.CatalogDirEnv+")")
	fs.StringVar(&opts.match, "match", "", "replay only recordings whose path contains this text")
	fs.IntVar(&opts.limit, "limit", 0, "replay at most this many recordings after -match (0 = all)")
	fs.IntVar(&opts.jobs, "jobs", 0, "recordings to replay concurrently (0 = GOMAXPROCS)")
	fs.BoolVar(&opts.smoke, "smoke", false, "also serve each recording on loopback UDP and query it with a client")
	fs.BoolVar(&opts.timing, "timing", false, "include per-recording wall time (makes output differ between runs)")
	fs.StringVar(&opts.out, "out", "", "write per-recording JSON lines here instead of stdout")
	fs.StringVar(&opts.summary, "summary", "", "also write the run summary as JSON to this file")
	fs.BoolVar(&opts.skipMissing, "skip-missing", false, "exit 0 without replaying when the catalog is absent")
	fs.BoolVar(&opts.verbose, "v", false, "keep agent logging")
	fs.Usage = func() {
		_, _ = fmt.Fprintln(stderr, "usage: niac-corpus-replay [flags] [-catalog DIR | RECORDING...]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return opts, fmt.Errorf("parse flags: %w", err)
	}
	opts.files = fs.Args()

	return opts, nil
}

func loadRows(opts options) ([]corpusreplay.Row, error) {
	var rows []corpusreplay.Row
	var err error
	if len(opts.files) > 0 {
		rows, err = corpusreplay.FileRows(opts.files)
	} else {
		rows, err = corpusreplay.LoadCatalog(opts.catalog)
	}
	if err != nil {
		return nil, fmt.Errorf("load recordings: %w", err)
	}
	if opts.match != "" {
		filtered := rows[:0]
		for _, row := range rows {
			if strings.Contains(row.Path, opts.match) {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	if opts.limit > 0 && len(rows) > opts.limit {
		rows = rows[:opts.limit]
	}

	return rows, nil
}

func writeOutputs(opts options, stdout io.Writer, results []corpusreplay.Result, summary corpusreplay.Summary) error {
	out := stdout
	if opts.out != "" {
		file, err := os.OpenFile(opts.out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, filePrivate)
		if err != nil {
			return fmt.Errorf("create %s: %w", opts.out, err)
		}
		defer func() { _ = file.Close() }()
		out = file
	}
	if err := corpusreplay.WriteJSONL(out, results); err != nil {
		return fmt.Errorf("write results: %w", err)
	}
	if opts.summary == "" {
		return nil
	}
	body, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("encode summary: %w", err)
	}
	if err = os.WriteFile(opts.summary, append(body, '\n'), filePrivate); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}

	return nil
}
