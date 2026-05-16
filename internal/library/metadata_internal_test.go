package library

import "testing"

// metadata.go's parseHeaderMetadata is package-private; tested from
// inside the package since the API surface (NetworkEntry.Description /
// NetworkEntry.UseCase) folds many edge cases together. Tested
// directly so each parsing branch is hit deterministically.

func TestParseHeaderMetadataExplicitFields(t *testing.T) {
	content := []byte(`# Description: Three-router lab
# Use Case: Verify OSPF area boundary
# Some other note
devices:
  - name: r1
`)
	desc, uc := parseHeaderMetadata(content)
	if desc != "Three-router lab" {
		t.Errorf("description = %q, want Three-router lab", desc)
	}
	if uc != "Verify OSPF area boundary" {
		t.Errorf("use case = %q, want Verify OSPF area boundary", uc)
	}
}

func TestParseHeaderMetadataFallsBackToFirstComment(t *testing.T) {
	// No explicit Description: line — first non-empty comment wins.
	content := []byte(`# Quick smoke test
# Use Case: Just a sanity run
devices:
`)
	desc, uc := parseHeaderMetadata(content)
	if desc != "Quick smoke test" {
		t.Errorf("description = %q, want Quick smoke test", desc)
	}
	if uc != "Just a sanity run" {
		t.Errorf("use case = %q, want Just a sanity run", uc)
	}
}

func TestParseHeaderMetadataIgnoresBody(t *testing.T) {
	// Comments inside the YAML body should NOT pollute metadata —
	// scanning stops at the first non-comment line.
	content := []byte(`# Real header line
devices:
  # this comment is inside the body and must be ignored
  - name: r1
`)
	desc, _ := parseHeaderMetadata(content)
	if desc != "Real header line" {
		t.Errorf("description = %q, want Real header line", desc)
	}
}

func TestParseHeaderMetadataEmpty(t *testing.T) {
	desc, uc := parseHeaderMetadata(nil)
	if desc != "" || uc != "" {
		t.Errorf("nil content: desc=%q, uc=%q (want both empty)", desc, uc)
	}
	desc, uc = parseHeaderMetadata([]byte(""))
	if desc != "" || uc != "" {
		t.Errorf("empty content: desc=%q, uc=%q (want both empty)", desc, uc)
	}
}

func TestParseHeaderMetadataPrefixIsCaseInsensitive(t *testing.T) {
	// matchPrefix lowercases both sides; verify a lowercase header
	// still gets picked up. Documented behaviour — there's a real
	// "# description: …" line in the wild from converted templates.
	content := []byte(`# description: ospf transit
devices:
`)
	desc, _ := parseHeaderMetadata(content)
	if desc != "ospf transit" {
		t.Errorf("description = %q, want ospf transit", desc)
	}
}

func TestIsYAMLFilename(t *testing.T) {
	cases := map[string]bool{
		"foo.yaml":  true,
		"foo.yml":   true,
		"FOO.YAML":  false, // case-sensitive on purpose; daemon writes lowercase
		"foo.json":  false,
		"":          false,
		"foo":       false,
		"foo.yaml/": false,
	}
	for name, want := range cases {
		if got := isYAMLFilename(name); got != want {
			t.Errorf("isYAMLFilename(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestValidateName(t *testing.T) {
	// Validates the rules documented on the validateName helper:
	// only [a-zA-Z0-9._-]; no slashes, no leading dot, no ".." anywhere.
	ok := []string{
		"foo",
		"foo-bar",
		"foo_bar",
		"foo.bar",
		"a1B2c3",
		"3-router-network",
	}
	bad := []string{
		"",
		".hidden",
		"..",
		"../escape",
		"a/b",
		"a\\b",
		"has space",
		"emoji-🥲",
	}
	for _, n := range ok {
		if err := validateName(n); err != nil {
			t.Errorf("validateName(%q) unexpected error: %v", n, err)
		}
	}
	for _, n := range bad {
		if err := validateName(n); err == nil {
			t.Errorf("validateName(%q) should have errored", n)
		}
	}
}
