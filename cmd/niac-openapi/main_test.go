package main

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
)

// generated renders the committed source against the live registry, which is
// what `make openapi` writes.
func generated(t *testing.T) map[string]any {
	t.Helper()
	src, err := os.ReadFile("../../docs/openapi-source.yaml")
	if err != nil {
		t.Fatalf("reading source: %v", err)
	}
	out, err := generate(src, api.RouteManifest())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	var doc map[string]any
	if unmarshalErr := yaml.Unmarshal(out, &doc); unmarshalErr != nil {
		t.Fatalf("generated document is not valid YAML: %v", unmarshalErr)
	}
	return doc
}

func pathsOf(t *testing.T, doc map[string]any) map[string]map[string]any {
	t.Helper()
	raw, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatal("generated document has no paths")
	}
	out := make(map[string]map[string]any, len(raw))
	for p, item := range raw {
		out[p] = item.(map[string]any)
	}
	return out
}

// TestEveryRegistryRouteIsDocumented is P1-7's acceptance: no registered route
// may be absent, and each must carry exactly the methods the registry gates.
// The spec this replaced covered 9 of them.
func TestEveryRegistryRouteIsDocumented(t *testing.T) {
	paths := pathsOf(t, generated(t))

	// Index the document by the registry path each entry came from, undoing
	// the {param} templating so the comparison is against the registry.
	byPrefix := map[string]map[string]any{}
	for docPath, item := range paths {
		key := docPath
		if i := strings.Index(docPath, "/{"); i >= 0 {
			key = docPath[:i+1]
		}
		byPrefix[key] = item
	}

	for _, rt := range api.RouteManifest() {
		if rt.Hidden {
			continue // the SPA shell and the /api/ not-found fallback
		}
		item, ok := byPrefix[rt.Path]
		if !ok {
			t.Errorf("registered route %s is missing from the generated document", rt.Path)
			continue
		}
		for _, m := range rt.Methods {
			if _, documented := item[strings.ToLower(m)]; !documented {
				t.Errorf("%s: method %s is registered but not documented", rt.Path, m)
			}
		}
	}
}

// TestPolicyIsDocumented checks the half of the description a hand-written
// spec cannot keep true: CSRF, admin scope and rate limiting per route.
func TestPolicyIsDocumented(t *testing.T) {
	paths := pathsOf(t, generated(t))

	// /api/v1/config/import is the admin-scoped whole-topology replace.
	admin, ok := paths["/api/v1/config/import"]["post"].(map[string]any)
	if !ok {
		t.Fatal("POST /api/v1/config/import is missing")
	}
	if _, has403 := admin["responses"].(map[string]any)["403"]; !has403 {
		t.Error("the admin-scoped route documents no 403")
	}
	if !strings.Contains(admin["description"].(string), "admin") {
		t.Errorf("admin scope is not documented: %v", admin["description"])
	}

	// A CSRF-protected write declares the CsrfToken scheme; a safe read
	// on the same route must not, because csrf.Protect skips safe methods.
	post := paths["/api/v1/simulation"]["post"].(map[string]any)
	sec, ok := post["security"].([]any)
	if !ok || len(sec) == 0 {
		t.Fatalf("POST /api/v1/simulation declares no security requirement: %v", post["security"])
	}
	if _, needsCSRF := sec[0].(map[string]any)["CsrfToken"]; !needsCSRF {
		t.Errorf("POST /api/v1/simulation does not require a CSRF token: %v", sec[0])
	}
	if _, declared := paths["/api/v1/simulation"]["get"].(map[string]any)["security"]; declared {
		t.Error("GET /api/v1/simulation should inherit the global bearer requirement, not declare CSRF")
	}

	// The two unauthenticated introspection routes opt out of security.
	for _, p := range []string{"/__version", "/__capabilities"} {
		public, isList := paths[p]["get"].(map[string]any)["security"].([]any)
		if !isList || len(public) != 0 {
			t.Errorf("%s should declare an empty security requirement, got %v", p, public)
		}
	}
}

// TestErrorSchemaMatchesTheGoType guards the specific drift that motivated
// this: the replaced spec documented `error_code` and `request_id`, keys
// api.ErrorResponse has never had.
func TestErrorSchemaMatchesTheGoType(t *testing.T) {
	doc := generated(t)
	schema := doc["components"].(map[string]any)["schemas"].(map[string]any)["Error"].(map[string]any)
	props := schema["properties"].(map[string]any)

	for _, want := range []string{"error", "message", "requestId", "timestamp", "path", "method", "details"} {
		if _, ok := props[want]; !ok {
			t.Errorf("error schema is missing %q", want)
		}
	}
	for _, gone := range []string{"error_code", "request_id"} {
		if _, ok := props[gone]; ok {
			t.Errorf("error schema still carries the drifted key %q", gone)
		}
	}
}

// TestNotFoundFallbackIsNotDocumentedAsOperations — the /api/ catch-all
// accepts every method only to answer 404 in the JSON envelope; documenting
// it would make a client generator emit seven methods for a 404.
func TestNotFoundFallbackIsNotDocumentedAsOperations(t *testing.T) {
	for docPath := range pathsOf(t, generated(t)) {
		if strings.HasPrefix(docPath, "/api/{") {
			t.Errorf("the /api/ not-found fallback is documented as %s", docPath)
		}
	}
}
