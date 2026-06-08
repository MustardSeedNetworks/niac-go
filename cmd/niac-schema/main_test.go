package main_test

import (
	"encoding/json"
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

func defKeys(defs jsonschema.Definitions) []string {
	out := make([]string, 0, len(defs))
	for k := range defs {
		out = append(out, k)
	}
	return out
}
