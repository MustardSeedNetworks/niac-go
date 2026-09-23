// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package authstate_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/authstate"
)

func TestDirectorySyncFailureIsUncertain(t *testing.T) {
	t.Parallel()
	directory, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = directory.Close(); err != nil {
		t.Fatal(err)
	}
	if err = authstate.SyncDirectoryForTest(directory); !errors.Is(err, authstate.ErrUncertain) {
		t.Fatalf("sync failure must fence authentication: %v", err)
	}
}

func TestWriteRecordRejectsWriteAndSyncFailures(t *testing.T) {
	t.Parallel()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	// A pipe accepts the bytes but cannot be synced. The second attempt uses
	// the now-closed writer and exercises the earlier write-failure path.
	for range 2 {
		if err = authstate.WriteRecordForTest(writer, []byte("changed")); err == nil {
			t.Fatal("accepted failed write/sync")
		}
	}
}

func TestPublishWriteFailurePreservesRecord(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := authstate.Publish(dir, []byte("original")); err != nil {
		t.Fatal(err)
	}
	if err := authstate.PublishWriteFailureForTest(dir, []byte("changed")); err == nil {
		t.Fatal("accepted failed write")
	}
	got, err := os.ReadFile(filepath.Join(dir, "auth.json"))
	if err != nil || string(got) != "original" {
		t.Fatalf("published failed write: %q, %v", got, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files remain: %v, %v", entries, err)
	}
}

func TestPublishPrivateModeAndNoSymlinkWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "auth.json")
	if err := os.Symlink(outside, target); err != nil {
		t.Fatal(err)
	}
	if err := authstate.Publish(dir, []byte("new state")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(target)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Fatalf("unprotected target: %v, %v", info, err)
	}
	got, err := os.ReadFile(outside)
	if err != nil || string(got) != "untouched" {
		t.Fatalf("symlink destination changed: %q, %v", got, err)
	}
}
