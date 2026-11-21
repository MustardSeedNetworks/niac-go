package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyDataDir(t *testing.T) {
	root := t.TempDir()
	plain := filepath.Join(root, "plain")
	file := filepath.Join(root, "file")
	link := filepath.Join(root, "link")
	if err := os.Mkdir(plain, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	linked := os.Symlink(plain, link) == nil

	tests := []struct {
		name    string
		path    string
		wantErr bool
		skip    bool
	}{
		{name: "plain directory", path: plain},
		{name: "regular file", path: file, wantErr: true},
		{name: "symlink to a directory", path: link, wantErr: true, skip: !linked},
		{name: "missing", path: filepath.Join(root, "missing"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("cannot create symlinks here")
			}
			err := verifyDataDir(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("verifyDataDir(%s) = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestVerifyDataDirNamesUnsafeDirs(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyDataDir(file); !errors.Is(err, errUnsafeDir) {
		t.Fatalf("verifyDataDir(file) = %v, want errUnsafeDir", err)
	}
}

func TestEnsureDataDirCreatesNested(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b")
	if err := ensureDataDir(path); err != nil {
		t.Fatal(err)
	}
	if err := verifyDataDir(path); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareServiceDirs(t *testing.T) {
	programData := t.TempDir()
	t.Setenv("ProgramData", programData)
	t.Setenv("NIAC_CERT_DIR", "")
	if defaultCertDir() == "" {
		t.Skip("the default certificate directory exists only on Windows")
	}
	if err := prepareServiceDirs(); err != nil {
		t.Fatal(err)
	}
	if err := verifyDataDir(filepath.Join(programData, "NiAC", "certs")); err != nil {
		t.Fatal(err)
	}
}
