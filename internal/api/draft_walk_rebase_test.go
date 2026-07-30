package api

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

func TestRebaseCapturedDraftContentUsesCurrentLibrary(t *testing.T) {
	server := walkProfileTestServer(t)
	content := `include_path: /old/machine/library/walks
devices:
  - name: access-1
    type: switch
    vendor: cisco
    snmp_agent:
      enabled: true
      walk_file: captured/access.walk
`
	rebased, err := server.rebaseCapturedDraftContent(content)
	if err != nil {
		t.Fatalf("rebaseCapturedDraftContent() error = %v", err)
	}
	walkRoot := server.library.SubDir(library.KindWalks)
	if !strings.Contains(rebased, "include_path: "+walkRoot) ||
		!strings.Contains(rebased, "walk_file: captured/access.walk") {
		t.Fatalf("rebased content = %s", rebased)
	}
	if strings.Contains(rebased, filepath.ToSlash("/old/machine")) {
		t.Fatalf("rebased content retains old machine path: %s", rebased)
	}
}

func TestRebaseCapturedDraftContentRestoresRelativeWalksAfterMutation(t *testing.T) {
	server := walkProfileTestServer(t)
	oldRoot := filepath.Join(t.TempDir(), "walks")
	content := "include_path: " + oldRoot + `
devices:
  - name: access-1
    snmp_agent:
      walk_file: ` + filepath.Join(oldRoot, "captured", "access.walk") + `
      walk_files:
        - ` + filepath.Join(oldRoot, "captured", "interfaces.walk") + `
`
	rebased, err := server.rebaseCapturedDraftContent(content)
	if err != nil {
		t.Fatalf("rebaseCapturedDraftContent() error = %v", err)
	}
	for _, expected := range []string{"walk_file: captured/access.walk", "- captured/interfaces.walk"} {
		if !strings.Contains(rebased, expected) {
			t.Fatalf("rebased content missing %q: %s", expected, rebased)
		}
	}
	if strings.Contains(rebased, oldRoot) {
		t.Fatalf("rebased content retains old root: %s", rebased)
	}
}

func TestRebaseCapturedDraftContentPreservesOtherWalkRoot(t *testing.T) {
	server := walkProfileTestServer(t)
	if err := server.library.WriteFile(
		library.KindWalks, "vendor/access.walk", []byte(capturedProfileFixture),
	); err != nil {
		t.Fatalf("write vendor walk: %v", err)
	}
	previousRoot := filepath.Join(t.TempDir(), "old-library", "walks")
	content := "include_path: " + previousRoot + `
devices:
  - name: access-1
    snmp_agent:
      walk_files:
        - captured/access.walk
        - vendor/access.walk
`
	rebased, err := server.rebaseCapturedDraftContent(content)
	if err != nil {
		t.Fatalf("rebaseCapturedDraftContent() error = %v", err)
	}
	if !strings.Contains(rebased, "- captured/access.walk") ||
		!strings.Contains(rebased, "- vendor/access.walk") {
		t.Fatalf("rebased content did not preserve mixed roots: %s", rebased)
	}
}

func TestRebaseCapturedDraftContentRejectsWalkOutsideLibrary(t *testing.T) {
	server := walkProfileTestServer(t)
	previousRoot := filepath.Join(t.TempDir(), "walks")
	content := "include_path: " + previousRoot + `
devices:
  - name: access-1
    snmp_agent:
      walk_files: [captured/access.walk, vendor/access.walk]
`
	if _, err := server.rebaseCapturedDraftContent(content); err == nil {
		t.Fatal("rebaseCapturedDraftContent() accepted a walk outside the content library")
	}
}

func TestRebaseCapturedDraftContentRejectsAmbiguousMixedRelativeWalks(t *testing.T) {
	server := walkProfileTestServer(t)
	content := `devices:
  - name: access-1
    snmp_agent:
      walk_files: [captured/access.walk, vendor/access.walk]
`
	if _, err := server.rebaseCapturedDraftContent(content); err == nil {
		t.Fatal("rebaseCapturedDraftContent() accepted mixed relative walks without a root")
	}
}
