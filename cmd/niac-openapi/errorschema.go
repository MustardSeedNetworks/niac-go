package main

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
)

// errorSchema renders api.ErrorResponse — the envelope every writeError call
// produces — as an OpenAPI schema object. Reflecting the Go type is the point:
// the hand-written spec described this envelope as `error_code`/`request_id`,
// neither of which the daemon has ever sent.
func errorSchema() (map[string]any, error) {
	reflector := &jsonschema.Reflector{
		ExpandedStruct:            true,
		DoNotReference:            true,
		AllowAdditionalProperties: true,
	}
	schema := reflector.Reflect(&api.ErrorResponse{})

	// Round-trip through JSON so the result is plain maps: OpenAPI 3.0 shares
	// JSON Schema's object vocabulary, and yaml.Marshal needs a value it can
	// walk rather than the reflector's ordered-map types.
	data, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("marshalling the error schema: %w", err)
	}
	var out map[string]any
	if unmarshalErr := json.Unmarshal(data, &out); unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshalling the error schema: %w", unmarshalErr)
	}
	// $schema and $id describe the JSON Schema document, not the type; an
	// OpenAPI components entry carries neither.
	delete(out, "$schema")
	delete(out, "$id")
	return out, nil
}
