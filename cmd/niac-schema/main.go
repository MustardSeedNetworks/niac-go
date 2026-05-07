// niac-schema generates a JSON Schema for the NIAC YAML config from the
// converter.Config Go struct, so it stays in sync with the parser.
//
// Usage:
//
//	niac-schema [-o docs/schemas/niac.schema.json]
//
// The output is the YAML-language-server compatible schema editors can fetch
// for inline validation. Wire it via either:
//
//	# yaml-language-server: $schema=https://raw.githubusercontent.com/krisarmstrong/niac-go/main/docs/schemas/niac.schema.json
//
// at the top of a config file, or via the editor's settings.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/invopop/jsonschema"

	"github.com/krisarmstrong/niac-go/internal/converter"
)

func main() {
	out := flag.String("o", "", "output file path; '-' or empty writes to stdout")
	flag.Parse()

	reflector := &jsonschema.Reflector{
		// Inline definitions for top-level types — keeps the schema readable
		// and lets yaml-language-server resolve $ref-less paths.
		ExpandedStruct: false,
		// Allow YAML-style snake_case keys to match what the converter parser
		// actually accepts (the struct tags already say `yaml:"snake_case"`).
		// invopop/jsonschema honours yaml: tags when configured.
		Anonymous: true,
		// Comments-as-descriptions would require a comment map — skip for now.
		AllowAdditionalProperties: false,
	}

	// Use yaml struct tags as the field-name source. Falling back to camelCase
	// (the default) would mis-document the actual YAML schema.
	reflector.KeyNamer = func(s string) string { return s }
	reflector.FieldNameTag = "yaml"

	schema := reflector.Reflect(&converter.Config{})

	// Annotate the root with project metadata so external tooling identifies it.
	schema.Title = "NIAC configuration"
	schema.Description = "Network In A Can simulation configuration. Generated from the Go " +
		"struct internal/converter.Config; refresh with `make schema` after struct changes."
	schema.ID = "https://raw.githubusercontent.com/krisarmstrong/niac-go/main/docs/schemas/niac.schema.json"

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}
	data = append(data, '\n')

	if *out == "" || *out == "-" {
		_, _ = os.Stdout.Write(data)
		return
	}
	if writeErr := os.WriteFile(*out, data, 0o600); writeErr != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *out, writeErr)
		os.Exit(1)
	}
}
