package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/invopop/jsonschema"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// TestSchemaContainsConfig is a regression check: the generator must produce
// a schema rooted in converter.Config, with a non-empty $defs (proves the
// reflector ran and the struct is reachable from this package).
func TestSchemaContainsConfig(t *testing.T) {
	reflector := &jsonschema.Reflector{
		ExpandedStruct:            false,
		AllowAdditionalProperties: false,
		Anonymous:                 true,
	}
	reflector.KeyNamer = func(s string) string { return s }
	reflector.FieldNameTag = "yaml"

	schema := reflector.Reflect(&converter.Config{})

	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Sanity: schema is non-empty JSON
	if len(data) < 100 {
		t.Fatalf("schema is suspiciously short (%d bytes): %s", len(data), data)
	}

	// $defs map should contain Config (the root type) plus several
	// well-known sub-types. If a future struct rename quietly drops these,
	// this test catches it.
	if len(schema.Definitions) == 0 {
		t.Fatalf("schema has no $defs; reflector produced an empty schema")
	}
	wantDefs := []string{"Config", "Device", "SnmpAgent", "DhcpServer", "DNSServer"}
	for _, def := range wantDefs {
		if _, ok := schema.Definitions[def]; !ok {
			t.Errorf("schema missing $def %q (got %d defs total: %v)",
				def, len(schema.Definitions), defKeys(schema.Definitions))
		}
	}
}

func TestSchemaRestrictsSyslogReceivers(t *testing.T) {
	reflector := &jsonschema.Reflector{AllowAdditionalProperties: false, Anonymous: true}
	reflector.FieldNameTag = "yaml"
	schema := reflector.Reflect(&converter.Config{})
	syslog := schema.Definitions["SyslogConfig"]
	receivers, found := syslog.Properties.Get("receivers")
	if !found || receivers.Items == nil {
		t.Fatal("SyslogConfig receivers schema is missing")
	}
	pattern := regexp.MustCompile(receivers.Items.Pattern)
	for _, valid := range []string{"192.0.2.10:514", "255.255.255.255:65535"} {
		if !pattern.MatchString(valid) {
			t.Errorf("receiver pattern rejected %q", valid)
		}
	}
	for _, invalid := range []string{
		"999.999.999.999:514", "001.002.003.004:514", "192.0.2.1:0", "192.0.2.1:0514", "192.0.2.1:65536",
	} {
		if pattern.MatchString(invalid) {
			t.Errorf("receiver pattern accepted %q", invalid)
		}
	}
}

func defKeys(defs jsonschema.Definitions) []string {
	out := make([]string, 0, len(defs))
	for k := range defs {
		out = append(out, k)
	}
	return out
}

// TestEveryPropertyIsDescribed holds the P1b-5 parity clause: every field an
// author can write must carry a description.
//
// The descriptions are the single source the authoring guide, the YAML
// editor's completion and the device editor's per-field help all read from —
// the generated forms render field.description as their only help text, so an
// undescribed field is an unlabelled control as well as an undocumented one.
// It asserts against the committed schema rather than a fresh reflection
// because that file is what those three consumers actually load.
func TestEveryPropertyIsDescribed(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "schemas", "niac.schema.json"))
	if err != nil {
		t.Fatalf("read committed schema: %v", err)
	}

	var schema struct {
		Defs map[string]struct {
			Properties map[string]struct {
				Description string `json:"description"`
			} `json:"properties"`
		} `json:"$defs"`
	}
	if unmarshalErr := json.Unmarshal(raw, &schema); unmarshalErr != nil {
		t.Fatalf("unmarshal committed schema: %v", unmarshalErr)
	}

	total := 0
	var undescribed []string
	for typeName, def := range schema.Defs {
		for field, property := range def.Properties {
			total++
			if strings.TrimSpace(property.Description) == "" {
				undescribed = append(undescribed, typeName+"."+field)
			}
		}
	}

	if total == 0 {
		t.Fatal("committed schema exposes no properties; the generator or this test is wrong")
	}
	if len(undescribed) > 0 {
		sort.Strings(undescribed)
		t.Errorf("%d of %d schema properties have no description; add a Go doc comment "+
			"on the field in internal/converter/types.go and run `make schema`:\n  %s",
			len(undescribed), total, strings.Join(undescribed, "\n  "))
	}
}
