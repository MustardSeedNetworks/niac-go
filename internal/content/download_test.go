package content_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/content"
)

func TestDownloadHonoursChecksum(t *testing.T) {
	body := []byte("synthetic-bundle-bytes")
	sum := sha256.Sum256(body)
	hexSum := hex.EncodeToString(sum[:])

	mux := http.NewServeMux()
	mux.HandleFunc("/bundle.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	})
	mux.HandleFunc("/bundle.tar.gz.sha256", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(hexSum + "  bundle.tar.gz\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dest := tmpDestPath(t)
	res, err := content.Download(context.Background(), "v0.0.1-test", content.DownloadOptions{
		URL:         srv.URL + "/bundle.tar.gz",
		ChecksumURL: srv.URL + "/bundle.tar.gz.sha256",
		Dest:        dest,
		Client:      srv.Client(),
	})
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if !res.ChecksumOK {
		t.Errorf("checksum should have verified, reason: %q", res.ChecksumReason)
	}
	if res.Bytes != int64(len(body)) {
		t.Errorf("byte count: got %d want %d", res.Bytes, len(body))
	}
}

func TestDownloadDetectsChecksumMismatch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/bundle.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("payload-A"))
	})
	mux.HandleFunc("/bundle.tar.gz.sha256", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("0", 64) + "  bundle.tar.gz\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dest := tmpDestPath(t)
	_, err := content.Download(context.Background(), "v0.0.1-test", content.DownloadOptions{
		URL:         srv.URL + "/bundle.tar.gz",
		ChecksumURL: srv.URL + "/bundle.tar.gz.sha256",
		Dest:        dest,
		Client:      srv.Client(),
	})
	if !errors.Is(err, content.ErrChecksumMismatch) {
		t.Fatalf("want ErrChecksumMismatch, got %v", err)
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Errorf("partial file was not cleaned up: %v", statErr)
	}
}

func TestDownloadProceedsWhenChecksumMissing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/bundle.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("payload"))
	})
	// no sha256 route → 404
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dest := tmpDestPath(t)
	res, err := content.Download(context.Background(), "v0.0.1-test", content.DownloadOptions{
		URL:         srv.URL + "/bundle.tar.gz",
		ChecksumURL: srv.URL + "/bundle.tar.gz.sha256",
		Dest:        dest,
		Client:      srv.Client(),
	})
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if res.ChecksumOK {
		t.Error("ChecksumOK should be false when sidecar is missing")
	}
	if res.ChecksumReason == "" {
		t.Error("ChecksumReason should explain why verification was skipped")
	}
}

func TestDownloadRequiresVersion(t *testing.T) {
	_, err := content.Download(context.Background(), "", content.DownloadOptions{})
	if !errors.Is(err, content.ErrEmptyVersion) {
		t.Fatalf("want ErrEmptyVersion, got %v", err)
	}
}

// tmpDestPath returns an absolute path inside t.TempDir() that does
// NOT yet exist on disk, suitable as Download.Dest. We avoid
// os.CreateTemp here so the download path can create the file fresh.
func tmpDestPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "bundle.tar.gz")
}
