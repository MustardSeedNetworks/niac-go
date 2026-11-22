package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/securedir"
)

// serviceTree points the service's data and certificate directories at a
// temporary ProgramData and returns its root.
func serviceTree(t *testing.T) string {
	t.Helper()
	programData := t.TempDir()
	t.Setenv("ProgramData", programData)
	t.Setenv("NIAC_CERT_DIR", filepath.Join(programData, "NiAC", "certs"))
	return programData
}

func TestPrepareServiceDirsCreatesPrivateDirs(t *testing.T) {
	programData := serviceTree(t)
	opts := testDirOptions(t)

	dirs, err := prepareServiceDirs(opts)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := securedir.Canonical(filepath.Join(programData, "NiAC"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if dirs.data != canonical || dirs.cert != filepath.Join(canonical, "certs") {
		t.Fatalf("dirs = %+v, want data %s and certs beneath it", dirs, canonical)
	}
	for _, dir := range dirs.all() {
		if verifyErr := securedir.Verify(dir, opts); verifyErr != nil {
			t.Errorf("Verify(%s) = %v", dir, verifyErr)
		}
	}
	if _, err = prepareServiceDirs(opts); err != nil {
		t.Fatalf("second start over the prepared directories: %v", err)
	}
}

func TestPrepareServiceDirsRefusesLinkedDataDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("securedir follows a link an Administrator owns; the install job's junction cases cover Windows")
	}
	programData := serviceTree(t)
	elsewhere := t.TempDir()
	if err := os.Symlink(elsewhere, filepath.Join(programData, "NiAC")); err != nil {
		t.Skipf("cannot create symlinks here: %v", err)
	}

	_, err := prepareServiceDirs(testDirOptions(t))
	if !errors.Is(err, securedir.ErrReparsePoint) {
		t.Fatalf("prepareServiceDirs over a linked data directory = %v, want ErrReparsePoint", err)
	}
}

func TestPrepareServiceDirsRefusesWritableContent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows judges ACLs, which the install job's refusal cases exercise")
	}
	programData := serviceTree(t)
	dataDir := filepath.Join(programData, "NiAC")
	if err := os.Mkdir(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	planted := filepath.Join(dataDir, "planted.pem")
	if err := os.WriteFile(planted, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(planted, 0o666); err != nil {
		t.Fatal(err)
	}

	_, err := prepareServiceDirs(testDirOptions(t))
	if !errors.Is(err, securedir.ErrUnsafeAccess) {
		t.Fatalf("prepareServiceDirs with a world-writable file = %v, want ErrUnsafeAccess", err)
	}
}

func TestResolveServiceDirsRefusesRelativeCertDir(t *testing.T) {
	serviceTree(t)
	t.Setenv("NIAC_CERT_DIR", "certs")
	if _, err := resolveServiceDirs(testDirOptions(t)); err == nil {
		t.Fatal("resolveServiceDirs accepted a relative certificate directory")
	}
}
