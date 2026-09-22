package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/instance"
)

// `niac content install` used to extract into the library root itself, with
// nothing to notice that a daemon had already opened that tree. These tests
// pin the routing the single-writer rule needs: a running daemon owns the
// library, so the bundle travels to its API; no daemon means the CLI does the
// work itself, but only while it holds the single-instance lock.

// contentBundle builds the smallest bundle Extract accepts: one file under
// each library kind directory.
func contentBundle(t *testing.T) []byte {
	t.Helper()
	var raw bytes.Buffer
	gz := gzip.NewWriter(&raw)
	tw := tar.NewWriter(gz)
	for name, body := range map[string]string{
		"walks/switch1.walk": "1.3.6.1.2.1.1.1.0 = STRING: test\n",
		"networks/lab.yaml":  "devices: []\n",
	} {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatalf("write tar body: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return raw.Bytes()
}

// writeBundleFile drops a bundle on disk and returns its path.
func writeBundleFile(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "bundle.tar.gz")
	if err := os.WriteFile(path, contentBundle(t), 0o600); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	return path
}

// trustStubServer writes the stub daemon's own certificate where
// findDaemonCert looks, so the CLI verifies the connection instead of
// skipping the check — the same trust path an operator gets.
func trustStubServer(t *testing.T, server *httptest.Server) {
	t.Helper()
	certDir := t.TempDir()
	encoded := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	})
	if err := os.WriteFile(filepath.Join(certDir, "server.crt"), encoded, 0o600); err != nil {
		t.Fatalf("write stub certificate: %v", err)
	}
	t.Setenv("NIAC_CERT_DIR", certDir)
}

// holdInstanceLock takes the lock on dataDir and publishes port, standing in
// for a daemon that is already running against that data directory.
func holdInstanceLock(t *testing.T, dataDir string, port int) {
	t.Helper()
	lock, err := instance.Acquire(dataDir)
	if err != nil {
		t.Fatalf("acquire instance lock: %v", err)
	}
	t.Cleanup(func() { _ = lock.Release() })
	if portErr := lock.SetPort(port); portErr != nil {
		t.Fatalf("publish port: %v", portErr)
	}
}

// serverPort is the TCP port an httptest server settled on.
func serverPort(t *testing.T, server *httptest.Server) int {
	t.Helper()
	_, port, err := net.SplitHostPort(strings.TrimPrefix(server.URL, "https://"))
	if err != nil {
		t.Fatalf("split stub address: %v", err)
	}
	number, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("parse stub port: %v", err)
	}
	return number
}

// TestContentInstallGoesThroughTheRunningDaemon is the routing rule: with a
// daemon holding the data directory, the bundle reaches its library-install
// endpoint and the CLI writes nothing itself.
func TestContentInstallGoesThroughTheRunningDaemon(t *testing.T) {
	dataDir := t.TempDir()
	libRoot := filepath.Join(dataDir, "library")
	t.Setenv("NIAC_LIBRARY_ROOT", libRoot)

	var installed struct {
		hit      bool
		filename string
		bytes    int
	}
	stub := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/csrf-token":
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "csrf"})
		case "/api/v1/library/install":
			var req struct {
				Filename string `json:"filename"`
				Data     string `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode install request: %v", err)
			}
			installed.hit = true
			installed.filename = req.Filename
			installed.bytes = len(req.Data)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true, "files": 2, "directories": 2, "bytes": 64,
				"perKind": map[string]int{"walks": 1, "networks": 1},
				"message": "Content bundle installed",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer stub.Close()

	trustStubServer(t, stub)
	holdInstanceLock(t, dataDir, serverPort(t, stub))

	bundlePath := writeBundleFile(t, t.TempDir())
	if err := runContentInstall(contentInstallArgs{bundlePath: bundlePath}); err != nil {
		t.Fatalf("content install: %v", err)
	}

	if !installed.hit {
		t.Fatal("the running daemon never received the bundle: the CLI installed it locally")
	}
	if installed.filename != "bundle.tar.gz" || installed.bytes == 0 {
		t.Fatalf("daemon received %+v, want the named bundle with a body", installed)
	}
	if entries, err := os.ReadDir(filepath.Join(libRoot, "walks")); err == nil && len(entries) > 0 {
		t.Fatalf("the CLI wrote %d file(s) into the daemon's library behind its back", len(entries))
	}
}

// TestContentInstallRefusesAnUnreachableHolder covers the holder that owns the
// data directory without serving an API — `niac daemon --once`. Guessing a
// port would send the bundle to whatever else is listening, so the CLI says
// what is in the way instead of installing under it.
func TestContentInstallRefusesAnUnreachableHolder(t *testing.T) {
	dataDir := t.TempDir()
	libRoot := filepath.Join(dataDir, "library")
	t.Setenv("NIAC_LIBRARY_ROOT", libRoot)

	holdInstanceLock(t, dataDir, 0)

	bundlePath := writeBundleFile(t, t.TempDir())
	err := runContentInstall(contentInstallArgs{bundlePath: bundlePath})
	if err == nil {
		t.Fatal("install succeeded while another instance held the data directory")
	}
	if !strings.Contains(err.Error(), "another instance") {
		t.Fatalf("error = %q, want it to name the instance holding the data directory", err)
	}
}

// TestContentInstallWithoutADaemonTakesTheLock proves the no-daemon branch is
// not simply unguarded: the install still happens, and it happens while the
// single-instance lock is held, so a daemon cannot start underneath it.
func TestContentInstallWithoutADaemonInstallsLocally(t *testing.T) {
	dataDir := t.TempDir()
	libRoot := filepath.Join(dataDir, "library")
	t.Setenv("NIAC_LIBRARY_ROOT", libRoot)

	bundlePath := writeBundleFile(t, t.TempDir())
	if err := runContentInstall(contentInstallArgs{bundlePath: bundlePath}); err != nil {
		t.Fatalf("content install: %v", err)
	}

	if _, err := os.Stat(filepath.Join(libRoot, "walks", "switch1.walk")); err != nil {
		t.Fatalf("bundle did not land in the library: %v", err)
	}
	// The lock must be free again: an install that keeps it would block the
	// next daemon start for the life of the process.
	if _, held, err := instance.Probe(dataDir); err != nil || held {
		t.Fatalf("instance lock still held after install (held=%v, err=%v)", held, err)
	}
}
