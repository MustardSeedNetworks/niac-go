package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
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
	clock         func() time.Time
}

type options struct {
	configPath  string
	analysisID  string
	latest      bool
	unitMAC     string
	niacVersion string
}

type analysisSummary struct {
	ID           string    `json:"_id"`
	AnalysisType string    `json:"analysisType"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UnitMAC      string    `json:"unitMac"`
}

// report is the acceptance result and the provenance needed to re-derive it.
// A result pinned in the ledger has to say which artifact it compared, against
// which analysis, on which build, and when; otherwise it is a claim rather than
// evidence.
type report struct {
	AnalysisID      string             `json:"analysisId"`
	ComparedAt      string             `json:"comparedAt"`
	NIACVersion     string             `json:"niacVersion,omitempty"`
	ConfigPath      string             `json:"configPath"`
	ConfigSHA256    string             `json:"configSha256"`
	UnitMAC         string             `json:"unitMac,omitempty"`
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
	digest, err := fileDigest(opts.configPath)
	if err != nil {
		return report{}, err
	}
	result := buildReport(analysisID, linklive.FromConfig(authoredConfig), observed)
	result.ComparedAt = r.now().UTC().Format(time.RFC3339)
	result.NIACVersion = opts.niacVersion
	result.ConfigPath = opts.configPath
	result.ConfigSHA256 = digest
	result.UnitMAC = normalizeMAC(opts.unitMAC)

	return result, nil
}

// fileDigest identifies the artifact that was compared, so a report cannot be
// paired after the fact with a config it never read.
func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read NIAC scenario: %w", err)
	}
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:]), nil
}

func (r runner) now() time.Time {
	if r.clock != nil {
		return r.clock()
	}

	return time.Now()
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
	set.StringVar(&opts.niacVersion, "niac-version", "",
		"NIAC build that served the scenario, recorded in the report")
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
