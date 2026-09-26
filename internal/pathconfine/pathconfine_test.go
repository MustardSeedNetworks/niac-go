// SPDX-License-Identifier: BUSL-1.1

package pathconfine_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/pathconfine"
)

func TestOpen(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(target, []byte("hello"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	// A traversal sequence still resolves under the same directory once
	// filepath.Abs/Dir/Base collapse it, so it must succeed, not fail.
	traversal := filepath.Join(dir, "..", filepath.Base(dir), "file.txt")

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "plain file", path: target},
		{name: "traversal that collapses back inside the directory", path: traversal},
		{name: "missing file", path: filepath.Join(dir, "missing.txt"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := pathconfine.Open(tt.path)
			if tt.wantErr {
				if err == nil {
					_ = f.Close()
					t.Fatal("Open() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Open() error = %v, want nil", err)
			}
			defer func() { _ = f.Close() }()

			got := make([]byte, 5)
			if _, readErr := f.Read(got); readErr != nil {
				t.Fatalf("Read() error = %v", readErr)
			}
			if string(got) != "hello" {
				t.Errorf("Read() = %q, want %q", got, "hello")
			}
		})
	}
}

func TestOpen_RejectsSymlinkEscape(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("do not read"), 0o600); err != nil {
		t.Fatalf("seed secret file: %v", err)
	}

	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	// A symlink whose target lies outside the Root's directory must be
	// refused, not silently followed — that is the TOCTOU protection Root
	// exists for.
	_, err := pathconfine.Open(link)
	if err == nil {
		t.Fatal("Open() followed a symlink escaping its directory, want error")
	}
}

func TestLstat(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(target, []byte("hi"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	info, err := pathconfine.Lstat(target)
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Name() != "file.txt" {
		t.Errorf("Lstat().Name() = %q, want %q", info.Name(), "file.txt")
	}

	if _, missingErr := pathconfine.Lstat(filepath.Join(dir, "missing.txt")); missingErr == nil {
		t.Error("Lstat() on missing file: error = nil, want error")
	}
}

func TestStat(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(target, []byte("hi"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	if _, err := pathconfine.Stat(target); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if _, err := pathconfine.Stat(filepath.Join(dir, "missing.txt")); err == nil {
		t.Error("Stat() on missing file: error = nil, want error")
	}
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")

	if err := pathconfine.WriteFile(target, []byte("written"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Clean(target))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "written" {
		t.Errorf("content = %q, want %q", got, "written")
	}

	missingDir := filepath.Join(dir, "missing-dir", "out.txt")
	if missingDirErr := pathconfine.WriteFile(missingDir, []byte("x"), 0o600); missingDirErr == nil {
		t.Error("WriteFile() into a nonexistent directory: error = nil, want error")
	}
}

func TestWriteFile_RejectsSymlinkEscape(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "clobbered.txt")
	if err := os.WriteFile(target, []byte("original"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if err := pathconfine.WriteFile(link, []byte("clobbered"), 0o600); err == nil {
		t.Fatal("WriteFile() followed a symlink escaping its directory, want error")
	}

	got, err := os.ReadFile(filepath.Clean(target))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "original" {
		t.Errorf("symlink target was modified: content = %q, want %q", got, "original")
	}
}
