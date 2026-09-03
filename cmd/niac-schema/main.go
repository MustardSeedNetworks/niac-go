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
//	# yaml-language-server: $schema=https://raw.githubusercontent.com/MustardSeedNetworks/niac-go/main/docs/schemas/niac.schema.json
//
// at the top of a config file, or via the editor's settings.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
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
		Anonymous:                 true,
		AllowAdditionalProperties: false,
	}
	if err := reflector.AddGoComments("github.com/MustardSeedNetworks/niac-go", "./internal/converter"); err != nil {
		fmt.Fprintf(os.Stderr, "comments: %v\n", err)
		os.Exit(1)
	}

	// Use yaml struct tags as the field-name source. Falling back to camelCase
	// (the default) would mis-document the actual YAML schema.
	reflector.KeyNamer = func(s string) string { return s }
	reflector.FieldNameTag = "yaml"

	schema := reflector.Reflect(&converter.Config{})
	applyOneofEnums(schema, converter.Config{})

	// Annotate the root with project metadata so external tooling identifies it.
	schema.Title = "NIAC configuration"
	schema.Description = "Network In A Can simulation configuration. Generated from the Go " +
		"struct internal/converter.Config; refresh with `make schema` after struct changes."
	schema.ID = "https://raw.githubusercontent.com/MustardSeedNetworks/niac-go/main/docs/schemas/niac.schema.json"

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

// applyOneofEnums copies the `validate:"...,oneof=a b c"` vocabularies onto the
// generated schema as JSON Schema `enum` lists.
//
// invopop/jsonschema reads yaml and jsonschema tags but knows nothing about
// go-playground/validator, so a field the parser accepts only twelve values
// for was published as a bare string. Both consumers suffered: an editor
// fetching the schema offered no completion, and the UI's schema-generated
// device forms had no way to tell a free-text field from a closed set.
//
// The vocabulary stays declared once, on the struct tag the parser enforces.
func applyOneofEnums(schema *jsonschema.Schema, root any) {
	for typ, fields := range oneofVocabularies(reflect.TypeOf(root)) {
		def, found := schema.Definitions[typ]
		if !found {
			continue
		}
		for name, values := range fields {
			property, ok := def.Properties.Get(name)
			if !ok {
				continue
			}
			property.Enum = make([]any, 0, len(values))
			for _, value := range values {
				property.Enum = append(property.Enum, value)
			}
		}
	}
}

// oneofVocabularies walks every struct reachable from root and returns, per
// type name, the yaml field names carrying a validator `oneof` and its values.
func oneofVocabularies(root reflect.Type) map[string]map[string][]string {
	out := map[string]map[string][]string{}
	seen := map[reflect.Type]bool{}

	var walk func(reflect.Type)
	walk = func(typ reflect.Type) {
		typ = elemType(typ)
		if typ.Kind() != reflect.Struct || seen[typ] {
			return
		}
		seen[typ] = true

		for field := range typ.Fields() {
			if name, values := oneofField(field); name != "" {
				if out[typ.Name()] == nil {
					out[typ.Name()] = map[string][]string{}
				}
				out[typ.Name()][name] = values
			}
			walk(field.Type)
		}
	}
	walk(root)

	return out
}

// elemType unwraps pointers, slices and arrays down to the value type.
func elemType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
		typ = typ.Elem()
	}

	return typ
}

// oneofField returns the field's yaml name and vocabulary, or "" when it has
// no `oneof` rule or no name to publish it under.
func oneofField(field reflect.StructField) (string, []string) {
	values := oneofValues(field.Tag.Get("validate"))
	if values == nil {
		return "", nil
	}
	name, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
	if name == "" || name == "-" {
		return "", nil
	}

	return name, values
}

func oneofValues(tag string) []string {
	for rule := range strings.SplitSeq(tag, ",") {
		if after, found := strings.CutPrefix(rule, "oneof="); found {
			return strings.Fields(after)
		}
	}

	return nil
}
