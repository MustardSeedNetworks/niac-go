package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/acceptance/linklive"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const requestTimeout = 90 * time.Second

type environment func(string) string

type runner struct {
	getenv        environment
	output        io.Writer
	allowInsecure bool
}

type options struct {
	configPath string
	analysisID string
}

type report struct {
	AnalysisID      string             `json:"analysisId"`
	AuthoredDevices int                `json:"authoredDevices"`
	AuthoredLinks   int                `json:"authoredLinks"`
	Findings        []linklive.Finding `json:"findings"`
	Passed          bool               `json:"passed"`
}

func newRunner(getenv environment, output io.Writer) runner {
	return runner{getenv: getenv, output: output}
}

func (r runner) run(ctx context.Context, args []string) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	result, err := r.compare(ctx, opts)
	if err != nil {
		return err
	}
	if err = writeReport(r.output, result); err != nil {
		return err
	}
	if !result.Passed {
		return fmt.Errorf("Link-Live acceptance failed with %d mismatches", len(result.Findings))
	}
	return nil
}

func (r runner) compare(ctx context.Context, opts options) (report, error) {
	authoredConfig, err := config.LoadYAML(opts.configPath)
	if err != nil {
		return report{}, fmt.Errorf("load NIAC scenario: %w", err)
	}
	client, err := r.client()
	if err != nil {
		return report{}, err
	}
	raw, err := client.Topology(ctx, opts.analysisID)
	if err != nil {
		return report{}, err
	}
	observed, err := linklive.ParseTopology(raw)
	if err != nil {
		return report{}, err
	}
	return buildReport(opts.analysisID, linklive.FromConfig(authoredConfig), observed), nil
}

func (r runner) client() (*linklive.Client, error) {
	return linklive.New(linklive.Config{
		IdentityURL:   valueOr(r.getenv("LINKLIVE_IDENTITY_URL"), "https://id.link-live.com"),
		APIURL:        valueOr(r.getenv("LINKLIVE_API_URL"), "https://link-live.com"),
		Username:      r.getenv("LINKLIVE_USERNAME"),
		Password:      r.getenv("LINKLIVE_PASSWORD"),
		MFACode:       r.getenv("LINKLIVE_MFA_CODE"),
		AllowInsecure: r.allowInsecure,
	})
}

func parseOptions(args []string) (options, error) {
	set := flag.NewFlagSet("linklive-acceptance", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	var opts options
	set.StringVar(&opts.configPath, "config", "", "NIAC scenario YAML path")
	set.StringVar(&opts.analysisID, "analysis", "", "Link-Live analysis ID")
	if err := set.Parse(args); err != nil {
		return options{}, err
	}
	if opts.configPath == "" || opts.analysisID == "" {
		return options{}, errors.New("-config and -analysis are required")
	}
	return opts, nil
}

func buildReport(
	analysisID string,
	authored linklive.AuthoredSnapshot,
	observed linklive.ObservedSnapshot,
) report {
	findings := linklive.Compare(authored, observed)
	return report{
		AnalysisID: analysisID, AuthoredDevices: len(authored.Devices),
		AuthoredLinks: len(authored.Links), Findings: findings, Passed: len(findings) == 0,
	}
}

func writeReport(output io.Writer, result report) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("write acceptance report: %w", err)
	}
	return nil
}

func valueOr(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
