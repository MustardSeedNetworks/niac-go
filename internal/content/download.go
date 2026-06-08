package content

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Sentinel errors for the download path.
var (
	ErrChecksumMismatch = errors.New("downloaded bundle checksum does not match expected value")
	ErrEmptyVersion     = errors.New("version is required to build the bundle URL")
)

// DefaultBundleURLTemplate is the GitHub releases pattern the install
// command uses when --url is not given. Both %s slots take the version
// tag (with leading 'v', e.g. "v0.66.41"). Callers wanting to redirect
// to a private mirror should pass DownloadOptions.URL explicitly.
const DefaultBundleURLTemplate = "https://github.com/MustardSeedNetworks/niac-go/releases/download/%s/niac-content-%s.tar.gz"

// DefaultChecksumURLTemplate sits next to the bundle and contains the
// hex sha256 on its first whitespace-separated token (the format
// `sha256sum` produces). Optional — if the file 404s, the install
// proceeds without verification but warns the caller.
const DefaultChecksumURLTemplate = "https://github.com/MustardSeedNetworks/niac-go/releases/download/%s/niac-content-%s.tar.gz.sha256"

// httpTimeout caps any single HTTP request the install path makes.
// Walks bundles can hit a couple of hundred MB so we leave plenty of
// room without going so high that a stuck server hangs the CLI forever.
const httpTimeout = 10 * time.Minute

// maxChecksumBytes bounds the sha256 sidecar read so a hostile server
// can't stream gigabytes at us before we notice. Real sidecars are
// ~70 bytes ("<64-hex>  <filename>\n"); 1 KiB is generous headroom.
const maxChecksumBytes = 1 << 10

// DownloadOptions tunes Download. Zero value uses the GitHub release
// pattern + a fresh http.Client with httpTimeout.
type DownloadOptions struct {
	// URL overrides DefaultBundleURLTemplate. Useful for offline mirrors
	// or local fixtures.
	URL string
	// ChecksumURL overrides DefaultChecksumURLTemplate.
	ChecksumURL string
	// Client lets tests inject a custom http.Client (e.g. one pointing
	// at httptest.Server). nil → a fresh Client with httpTimeout.
	Client *http.Client
	// Dest is the file path the bundle is written to. nil → a temp file
	// inside os.TempDir() that the caller is responsible for removing.
	Dest string
}

// DownloadResult tells the caller where the bundle landed and whether
// the checksum was verified.
type DownloadResult struct {
	Path           string
	Bytes          int64
	ChecksumOK     bool
	ChecksumReason string // empty when verified, populated on skip/failure
}

// Download fetches the bundle for the given version into a local file.
// When the matching .sha256 sidecar is reachable, its hash is checked
// against the downloaded bytes; mismatch returns ErrChecksumMismatch
// and removes the partial file.
func Download(ctx context.Context, version string, opts DownloadOptions) (DownloadResult, error) {
	if version == "" {
		return DownloadResult{}, ErrEmptyVersion
	}

	bundleURL := opts.URL
	if bundleURL == "" {
		bundleURL = fmt.Sprintf(DefaultBundleURLTemplate, version, version)
	}
	checksumURL := opts.ChecksumURL
	if checksumURL == "" {
		checksumURL = fmt.Sprintf(DefaultChecksumURLTemplate, version, version)
	}

	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: httpTimeout}
	}

	dest := opts.Dest
	if dest == "" {
		f, err := os.CreateTemp("", "niac-content-*.tar.gz")
		if err != nil {
			return DownloadResult{}, fmt.Errorf("create temp bundle file: %w", err)
		}
		dest = f.Name()
		_ = f.Close()
	}

	bytes, sum, err := fetchTo(ctx, client, bundleURL, dest)
	if err != nil {
		_ = os.Remove(dest)
		return DownloadResult{}, err
	}

	res := DownloadResult{Path: dest, Bytes: bytes}

	expected, err := fetchChecksum(ctx, client, checksumURL)
	switch {
	case errors.Is(err, errChecksumNotFound):
		res.ChecksumReason = "no checksum sidecar published; bundle was not verified"
	case err != nil:
		res.ChecksumReason = fmt.Sprintf("could not fetch checksum: %v", err)
	case !strings.EqualFold(expected, sum):
		_ = os.Remove(dest)
		return DownloadResult{}, fmt.Errorf("%w: got %s, want %s", ErrChecksumMismatch, sum, expected)
	default:
		res.ChecksumOK = true
	}
	return res, nil
}

// fetchTo streams a GET to dest while computing sha256 in the same
// pass — single read of the body, single write to disk.
func fetchTo(ctx context.Context, client *http.Client, url, dest string) (int64, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", fmt.Errorf("build request for %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("GET %s returned %d", url, resp.StatusCode)
	}

	if mkErr := os.MkdirAll(filepath.Dir(dest), libraryDirMode); mkErr != nil {
		return 0, "", fmt.Errorf("mkdir for %s: %w", dest, mkErr)
	}
	// G304 waiver: dest is the destination path the caller chose for
	// the download — either DownloadOptions.Dest (set programmatically
	// in cmd/niac/content.go) or a freshly-created os.CreateTemp file
	// from Download() above. In neither case is it ever taken from
	// untrusted input. The fetched body, on the other hand, is
	// untrusted — but it is bounded by the HTTP response and the
	// caller is responsible for choosing a safe dest.
	out, err := os.Create(dest) // #nosec G304
	if err != nil {
		return 0, "", fmt.Errorf("create %s: %w", dest, err)
	}
	defer out.Close()

	hasher := sha256.New()
	n, err := io.Copy(io.MultiWriter(out, hasher), resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("download body: %w", err)
	}
	return n, hex.EncodeToString(hasher.Sum(nil)), nil
}

var errChecksumNotFound = errors.New("checksum sidecar not found")

func fetchChecksum(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", errChecksumNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s returned %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxChecksumBytes))
	if err != nil {
		return "", fmt.Errorf("read checksum: %w", err)
	}
	// `sha256sum` format: "<hex>  filename\n". Take the first whitespace
	// token; tolerates a bare-hex file too.
	first := strings.Fields(strings.TrimSpace(string(body)))
	if len(first) == 0 {
		return "", errors.New("checksum file is empty")
	}
	return first[0], nil
}
