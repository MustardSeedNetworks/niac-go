package main

import (
	"os"
	"sort"
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
		if rt.Path == notFoundFallback {
			continue // documented as a fallback, not as seven operations
		}
		item, ok := byPrefix[rt.Path]
		if !ok {
			t.Errorf("registered route %s is missing from the generated document", rt.Path)
			continue
		}
		for _, m := range methodsOf(rt) {
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

// TestStaleSourceEntryFails proves the source file cannot outlive its routes.
func TestStaleSourceEntryFails(t *testing.T) {
	src := []byte("preamble:\n  openapi: 3.0.3\noperations:\n  GET /api/v1/gone:\n    summary: nope\n")
	_, err := generate(src, []api.RoutePolicy{{Path: "/api/v1/here", Methods: []string{"GET"}}})
	if err == nil || !strings.Contains(err.Error(), "GET /api/v1/gone") {
		t.Errorf("a source entry for an unserved route should fail, got %v", err)
	}
}

// TestSourceCannotOverridePaths keeps the generated half generated.
func TestSourceCannotOverridePaths(t *testing.T) {
	src := []byte("preamble:\n  paths:\n    /fake: {}\n")
	if _, err := generate(src, nil); err == nil || !strings.Contains(err.Error(), "paths") {
		t.Errorf("a preamble defining paths should fail, got %v", err)
	}
}

// TestCollidingDocumentedPathFails covers the prefix-route hazard: before the
// documented template fed the operation id, /api/v1/thing and /api/v1/thing/
// produced one id twice. A source file that collapses two routes onto one
// documented path would now silently drop an operation instead, so that is
// what fails.
func TestCollidingDocumentedPathFails(t *testing.T) {
	routes := []api.RoutePolicy{
		{Path: "/api/v1/thing", Methods: []string{"GET"}},
		{Path: "/api/v1/thing/", Methods: []string{"GET"}},
	}
	src := []byte("preamble:\n  openapi: 3.0.3\noperations:\n" +
		"  GET /api/v1/thing/:\n    pathTemplate: /api/v1/thing\n")
	if _, err := generate(src, routes); err == nil ||
		!strings.Contains(err.Error(), "claimed by more than one registered route") {
		t.Errorf("two routes on one documented path should fail, got %v", err)
	}
}

// TestPathTemplateMustAgreeAcrossMethods — one route, one documented shape.
func TestPathTemplateMustAgreeAcrossMethods(t *testing.T) {
	routes := []api.RoutePolicy{{Path: "/api/v1/thing/", Methods: []string{"GET", "DELETE"}}}
	src := []byte("preamble:\n  openapi: 3.0.3\noperations:\n" +
		"  GET /api/v1/thing/:\n    pathTemplate: /api/v1/thing/{id}\n" +
		"  DELETE /api/v1/thing/:\n    pathTemplate: /api/v1/thing/{name}\n")
	if _, err := generate(src, routes); err == nil || !strings.Contains(err.Error(), "pathTemplate disagrees") {
		t.Errorf("disagreeing templates should fail, got %v", err)
	}
}

// TestPrefixRoutesDeclarePathParameters — a documented {param} must be
// declared, or the description is not usable by a client generator.
func TestPrefixRoutesDeclarePathParameters(t *testing.T) {
	paths := pathsOf(t, generated(t))
	for docPath, item := range paths {
		if !strings.Contains(docPath, "{") {
			continue
		}
		params, ok := item["parameters"].([]any)
		if !ok || len(params) == 0 {
			t.Errorf("%s templates a parameter but declares none", docPath)
			continue
		}
		if params[0].(map[string]any)["in"] != "path" {
			t.Errorf("%s: parameter is not declared in the path", docPath)
		}
	}
}

// TestEveryOperationDeclaresResponses — an operation object without
// `responses` is invalid OpenAPI, and is the shape a merge bug would produce.
func TestEveryOperationDeclaresResponses(t *testing.T) {
	for docPath, item := range pathsOf(t, generated(t)) {
		for method, value := range item {
			if method == "parameters" {
				continue
			}
			op, isOp := value.(map[string]any)
			if !isOp {
				t.Errorf("%s %s is not an operation object", method, docPath)
				continue
			}
			responses, has := op["responses"].(map[string]any)
			if !has || len(responses) == 0 {
				t.Errorf("%s %s declares no responses", method, docPath)
			}
			if _, hasID := op["operationId"].(string); !hasID {
				t.Errorf("%s %s has no operationId", method, docPath)
			}
		}
	}
}

// TestSourceResponsesDoNotDropPolicyCodes — a source entry documenting a 200
// must not discard the 401/429 the middleware actually returns. Nothing in
// the committed source file supplies `responses` yet, so this is the guard
// for the first one that does.
func TestSourceResponsesDoNotDropPolicyCodes(t *testing.T) {
	routes := []api.RoutePolicy{
		{Path: "/api/v1/thing", Methods: []string{"POST"}, CSRF: true, RateLimited: true},
	}
	src := []byte("preamble:\n  openapi: 3.0.3\noperations:\n" +
		"  POST /api/v1/thing:\n    responses:\n      '200':\n        description: ok\n")
	out, err := generate(src, routes)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	var doc map[string]any
	if unmarshalErr := yaml.Unmarshal(out, &doc); unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}
	responses := doc["paths"].(map[string]any)["/api/v1/thing"].(map[string]any)["post"].(map[string]any)["responses"].(map[string]any)
	for _, code := range []string{"200", "401", "403", "429", "500"} {
		if _, ok := responses[code]; !ok {
			t.Errorf("response %s was dropped by the source merge: got %v", code, keysOf(responses))
		}
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
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
