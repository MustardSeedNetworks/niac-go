package config_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// yamlFence captures the body of every ```yaml block in a markdown file.
var yamlFence = regexp.MustCompile("(?s)```yaml\n(.*?)```")

// TestAuthoringGuideExamplesDecodeStrictly holds the P1b-5 clause that the
// guide teaches the vocabulary the loader actually accepts.
//
// Every example is decoded with KnownFields(true), the same strictness the
// loader uses, so a field the guide misspells or nests wrongly fails here
// rather than after a reader has copied it. A lenient decode would accept the
// exact class of mistake the guide exists to prevent.
func TestAuthoringGuideExamplesDecodeStrictly(t *testing.T) {
	guide := filepath.Join("..", "..", "docs", "AUTHORING_GUIDE.md")
	raw, err := os.ReadFile(guide)
	if err != nil {
		t.Fatalf("read authoring guide: %v", err)
	}

	matches := yamlFence.FindAllStringSubmatch(string(raw), -1)
	if len(matches) == 0 {
		t.Fatal("authoring guide contains no yaml examples; the guide or this test is wrong")
	}

	for i, match := range matches {
		body := match[1]
		decoder := yaml.NewDecoder(strings.NewReader(body))
		decoder.KnownFields(true)

		var cfg converter.Config
		decodeErr := decoder.Decode(&cfg)
		if decodeErr != nil && decodeErr.Error() != "EOF" {
			t.Errorf("yaml example %d does not decode against converter.Config: %v\n%s",
				i+1, decodeErr, body)
		}
	}
}

// TestClinicExampleValidates holds the other half of the clause: the guide's
// complete scenario is a real file, and it validates cleanly.
//
// Warnings count as failures here. The example is what a new author copies, so
// shipping one that validates "with a few warnings" teaches the warnings as
// normal.
func TestClinicExampleValidates(t *testing.T) {
	// Absolute, because the loader refuses a path containing "..".
	path, err := filepath.Abs(filepath.Join("..", "..", "docs", "examples", "clinic-branch.yaml"))
	if err != nil {
		t.Fatalf("resolve clinic example path: %v", err)
	}

	cfg, err := config.LoadYAML(path)
	if err != nil {
		t.Fatalf("load clinic example: %v", err)
	}

	result := config.NewValidator(path).Validate(cfg)

	if !result.Valid {
		t.Errorf("clinic example has validation errors: %#v", result.Errors)
	}
	if result.HasWarnings() {
		t.Errorf("clinic example has validation warnings: %#v", result.Warnings)
	}
}
