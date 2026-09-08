package support_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/support"
)

// libraryFixture builds a tree shaped like a content library: the three kinds,
// a nested walk, and a file whose mode matters.
func libraryFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"networks/office.yaml":       "devices:\n  - name: sw1\n",
		"walks/cisco/c2960.walk":     ".1.3.6.1.2.1.1.1.0 = STRING: sw1\n",
		"pcaps/capture.pcap":         "\xd4\xc3\xb2\xa1binary",
		"drafts/office.yaml.partial": "devices: []\n",
	}
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return root
}

// treeContents reads a directory into a path→content map so two trees can be
// compared as values rather than by walking twice.
func treeContents(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		out[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return out
}

func TestBackupRestoreRoundTrip(t *testing.T) {
	source := libraryFixture(t)

	var archive bytes.Buffer
	if err := support.Backup(source, &archive); err != nil {
		t.Fatalf("Backup: %v", err)
	}

	target := filepath.Join(t.TempDir(), "library")
	if err := support.Restore(bytes.NewReader(archive.Bytes()), target); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	want := treeContents(t, source)
	got := treeContents(t, target)
	if len(want) != len(got) {
		t.Fatalf("restored %d files, want %d\n got: %v\nwant: %v", len(got), len(want), got, want)
	}
	for name, content := range want {
		if got[name] != content {
			t.Errorf("%s: restored %q, want %q", name, got[name], content)
		}
	}
}

// A backup taken twice from an unchanged tree must be byte-identical, so an
// operator can tell a real content change from archive noise.
func TestBackupIsDeterministic(t *testing.T) {
	source := libraryFixture(t)

	var first, second bytes.Buffer
	if err := support.Backup(source, &first); err != nil {
		t.Fatalf("Backup (first): %v", err)
	}
	if err := support.Backup(source, &second); err != nil {
		t.Fatalf("Backup (second): %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Errorf("two backups of the same tree differ: %d vs %d bytes", first.Len(), second.Len())
	}
}

// Restore must refuse an entry that escapes the target root rather than
// writing outside it.
func TestRestoreRefusesPathTraversal(t *testing.T) {
	archive := traversalArchive(t, "../escaped.yaml")

	target := filepath.Join(t.TempDir(), "library")
	err := support.Restore(bytes.NewReader(archive), target)
	if err == nil {
		t.Fatal("Restore accepted an entry outside the root")
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(target), "escaped.yaml")); statErr == nil {
		t.Error("Restore wrote outside the target root")
	}
}

// A failed restore must leave the existing library untouched.
func TestRestoreLeavesTargetIntactOnFailure(t *testing.T) {
	target := libraryFixture(t)
	before := treeContents(t, target)

	err := support.Restore(bytes.NewReader(traversalArchive(t, "../escaped.yaml")), target)
	if err == nil {
		t.Fatal("Restore accepted an entry outside the root")
	}

	after := treeContents(t, target)
	if len(after) != len(before) {
		t.Fatalf("failed restore changed the target: %d files, want %d", len(after), len(before))
	}
	for name, content := range before {
		if after[name] != content {
			t.Errorf("%s: now %q, want %q", name, after[name], content)
		}
	}
}
