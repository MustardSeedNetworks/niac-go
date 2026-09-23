// SPDX-License-Identifier: BUSL-1.1

package authstate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/authstate"
)

func TestPublishReplacesCompleteRecord(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, record := range []string{`{"epoch":1}`, `{"epoch":2}`} {
		if err := authstate.Publish(dir, []byte(record)); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(dir, "auth.json"))
		if err != nil || string(got) != record {
			t.Fatalf("record = %q, error = %v", got, err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 1 {
			t.Fatalf("temporary files remain: %v, %v", entries, err)
		}
	}
}

func TestPublishRequiresAbsoluteDirectory(t *testing.T) {
	t.Parallel()
	for _, directory := range []string{"", ".", "relative"} {
		if err := authstate.Publish(directory, []byte("state")); err == nil {
			t.Fatalf("accepted non-absolute directory %q", directory)
		}
	}
}

func TestPublishFailurePreservesExistingTarget(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "auth.json")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := authstate.Publish(dir, []byte(`{"epoch":1}`)); err == nil {
		t.Fatal("replaced directory")
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		t.Fatalf("target changed: %v, %v", info, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files remain: %v, %v", entries, err)
	}
	if err = authstate.Publish(filepath.Join(dir, "missing"), nil); err == nil {
		t.Fatal("accepted missing directory")
	}
}
