package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
)

// writeTokenFile drops a single-token JSON document on disk with the
// 0o600 mode LoadTokenFile demands. Returns the path so the caller can
// hand it back into Config{TokenFile: ...}.
func writeTokenFile(t *testing.T, dir, value, scope string) string {
	t.Helper()
	path := filepath.Join(dir, "tokens.json")
	body, err := json.Marshal(map[string]any{
		"tokens": []map[string]string{{"value": value, "scope": scope}},
	})
	if err != nil {
		t.Fatalf("marshal token file body: %v", err)
	}
	if writeErr := os.WriteFile(path, body, 0o600); writeErr != nil {
		t.Fatalf("write token file: %v", writeErr)
	}
	return path
}

// authedRequest issues GET /api/v1/csrf-token against the daemon's
// bound address with the given bearer token and returns the response
// status. csrf-token is the cheapest authed endpoint — it returns 200
// unconditionally and does not require a simulation to be running, so
// the status reliably reports auth outcome rather than handler state.
func authedRequest(t *testing.T, addr, token string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("http://%s/api/v1/csrf-token", addr),
		http.NoBody,
	)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("issue request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

// newRotationDaemon spins up a daemon bound to a kernel-assigned
// loopback port with the supplied TokenFile. Returns the live daemon
// and its bound address; registers a t.Cleanup so the test does not
// need to remember to call Shutdown.
func newRotationDaemon(t *testing.T, tokenFile string) (*Daemon, string) {
	t.Helper()
	cfg := Config{
		ListenAddr:  "127.0.0.1:0",
		TokenFile:   tokenFile,
		StoragePath: filepath.Join(t.TempDir(), "rotation.db"),
		Version:     "test",
	}
	d, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	if startErr := d.Start(); startErr != nil {
		t.Fatalf("Daemon.Start: %v", startErr)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := d.Shutdown(ctx); shutdownErr != nil {
			t.Logf("Daemon.Shutdown returned: %v", shutdownErr)
		}
	})
	addr := d.apiServer.BoundAddr()
	if addr == "" {
		t.Fatal("apiServer.BoundAddr() returned empty string after Start")
	}
	return d, addr
}

// TestDaemon_ReloadTokens_RotatesTokenFile is the integration test
// behind issue #92. The Wave 2 SIGHUP handler (cmd/niac/main.go) is a
// trivial signal.Notify loop that calls Daemon.ReloadTokens — the
// interesting behaviour all lives in ReloadTokens + the token store,
// and this test exercises the whole path end-to-end with a real HTTP
// listener and real bearer-token middleware.
//
// Coverage:
//   - The initial token unlocks /api/v1/stats.
//   - An unrelated token is rejected (sanity, so the success below
//     is meaningful).
//   - After rewriting the token file and calling ReloadTokens, the
//     OLD token is rejected and the NEW token is accepted.
//   - ReloadTokens returns the count of active tokens after the swap.
func TestDaemon_ReloadTokens_RotatesTokenFile(t *testing.T) {
	tmpDir := t.TempDir()
	const initialToken = "initial-rw-12345"
	const rotatedToken = "rotated-rw-67890"

	tokenPath := writeTokenFile(t, tmpDir, initialToken, "read-write")
	d, addr := newRotationDaemon(t, tokenPath)

	if got := authedRequest(t, addr, initialToken); got != http.StatusOK {
		t.Errorf("initial token: GET /api/v1/csrf-token = %d, want 200", got)
	}
	if got := authedRequest(t, addr, "not-a-real-token"); got != http.StatusUnauthorized {
		t.Errorf("bogus token: GET /api/v1/csrf-token = %d, want 401", got)
	}

	// Rewrite the file and rotate. Overwriting in place mirrors the
	// real operator workflow (write new file → kill -HUP <pid>).
	if writeErr := os.WriteFile(
		tokenPath,
		mustMarshalSingleToken(t, rotatedToken, "read-write"),
		0o600,
	); writeErr != nil {
		t.Fatalf("rewrite token file: %v", writeErr)
	}
	count, err := d.ReloadTokens()
	if err != nil {
		t.Fatalf("ReloadTokens after rewrite: %v", err)
	}
	if count != 1 {
		t.Errorf("ReloadTokens returned count %d, want 1", count)
	}

	if got := authedRequest(t, addr, initialToken); got != http.StatusUnauthorized {
		t.Errorf("rotated-out token: GET /api/v1/csrf-token = %d, want 401", got)
	}
	if got := authedRequest(t, addr, rotatedToken); got != http.StatusOK {
		t.Errorf("rotated-in token: GET /api/v1/csrf-token = %d, want 200", got)
	}
}

// TestDaemon_ReloadTokens_PreservesPreviousTokensOnBadFile pins the
// fail-safe behaviour ReloadTokens promises in its doc comment:
// "On error the previously-active tokens stay in effect — the caller
// (typically the SIGHUP handler in cmd/niac) should log and keep
// serving."
//
// We trigger a load failure by widening the token file's permissions
// past 0o077 (LoadTokenFile refuses anything broader than 0o600) and
// confirm the original token still unlocks /api/v1/stats afterwards.
func TestDaemon_ReloadTokens_PreservesPreviousTokensOnBadFile(t *testing.T) {
	tmpDir := t.TempDir()
	const originalToken = "kept-after-bad-reload"

	tokenPath := writeTokenFile(t, tmpDir, originalToken, "read-write")
	d, addr := newRotationDaemon(t, tokenPath)

	if got := authedRequest(t, addr, originalToken); got != http.StatusOK {
		t.Fatalf("pre-reload sanity: GET /api/v1/csrf-token = %d, want 200", got)
	}

	if chmodErr := os.Chmod(tokenPath, 0o644); chmodErr != nil {
		t.Fatalf("chmod token file: %v", chmodErr)
	}
	_, err := d.ReloadTokens()
	if err == nil {
		t.Fatal("ReloadTokens accepted a world-readable file; expected refusal")
	}
	if !errors.Is(err, api.ErrTokenFileWorldReadable) {
		t.Errorf("ReloadTokens returned %v, want ErrTokenFileWorldReadable", err)
	}

	if got := authedRequest(t, addr, originalToken); got != http.StatusOK {
		t.Errorf("post-failed-reload: GET /api/v1/csrf-token = %d, want 200 (prior token must still work)", got)
	}
}

func mustMarshalSingleToken(t *testing.T, value, scope string) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"tokens": []map[string]string{{"value": value, "scope": scope}},
	})
	if err != nil {
		t.Fatalf("marshal token file body: %v", err)
	}
	return body
}
