package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
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
	latest     bool
	unitMAC    string
}

type analysisSummary struct {
	ID           string    `json:"_id"`
	AnalysisType string    `json:"analysisType"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UnitMAC      string    `json:"unitMac"`
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
	analysisID := opts.analysisID
	if opts.latest {
		analysisID, err = latestReadyDiscovery(ctx, client, opts.unitMAC)
		if err != nil {
			return report{}, err
		}
	}
	raw, err := client.Topology(ctx, analysisID)
	if err != nil {
		return report{}, err
	}
	observed, err := linklive.ParseTopology(raw)
	if err != nil {
		return report{}, err
	}
	return buildReport(analysisID, linklive.FromConfig(authoredConfig), observed), nil
}

func latestReadyDiscovery(
	ctx context.Context,
	client *linklive.Client,
	unitMAC string,
) (string, error) {
	raw, err := client.Analyses(ctx)
	if err != nil {
		return "", err
	}
	var analyses []analysisSummary
	if err = json.Unmarshal(raw, &analyses); err != nil {
		return "", errors.New("Link-Live analysis list returned invalid JSON")
	}
	wantedMAC := normalizeMAC(unitMAC)
	var latest analysisSummary
	for _, candidate := range analyses {
		if candidate.AnalysisType != "discovery" ||
			(wantedMAC != "" && normalizeMAC(candidate.UnitMAC) != wantedMAC) ||
			candidate.CreatedAt.Before(latest.CreatedAt) {
			continue
		}
		latest = candidate
	}
	if latest.ID == "" {
		return "", errors.New("Link-Live returned no matching discovery analyses")
	}
	if latest.Status != "ready" {
		return "", fmt.Errorf("latest Link-Live discovery analysis is %s", latest.Status)
	}
	return latest.ID, nil
}

func normalizeMAC(value string) string {
	replacer := strings.NewReplacer(":", "", "-", "", ".", "")
	return strings.ToUpper(replacer.Replace(strings.TrimSpace(value)))
}

func (r runner) client() (*linklive.Client, error) {
	return linklive.New(linklive.Config{
		IdentityURL:    valueOr(r.getenv("LINKLIVE_IDENTITY_URL"), "https://id.link-live.com"),
		APIURL:         valueOr(r.getenv("LINKLIVE_API_URL"), "https://link-live.com"),
		Username:       r.getenv("LINKLIVE_USERNAME"),
		Password:       r.getenv("LINKLIVE_PASSWORD"),
		MFACode:        r.getenv("LINKLIVE_MFA_CODE"),
		OrganizationID: r.getenv("LINKLIVE_ORGANIZATION_ID"),
		AccessToken:    r.getenv("LINKLIVE_ACCESS_TOKEN"),
		AllowInsecure:  r.allowInsecure,
	})
}

func parseOptions(args []string) (options, error) {
	set := flag.NewFlagSet("linklive-acceptance", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	var opts options
	set.StringVar(&opts.configPath, "config", "", "NIAC scenario YAML path")
	set.StringVar(&opts.analysisID, "analysis", "", "Link-Live analysis ID")
	set.BoolVar(&opts.latest, "latest", false, "use the latest ready Link-Live discovery analysis")
	set.StringVar(&opts.unitMAC, "unit-mac", "", "limit -latest to one NetAlly unit MAC")
	if err := set.Parse(args); err != nil {
		return options{}, err
	}
	if opts.configPath == "" {
		return options{}, errors.New("-config is required")
	}
	if (opts.analysisID == "") == !opts.latest {
		return options{}, errors.New("exactly one of -analysis or -latest is required")
	}
	if opts.unitMAC != "" && !opts.latest {
		return options{}, errors.New("-unit-mac requires -latest")
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
