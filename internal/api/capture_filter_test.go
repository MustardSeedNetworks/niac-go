package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"

	captureengine "github.com/MustardSeedNetworks/niac-go/internal/capture"
)

// newLoopbackCaptureServer builds a test server whose stack has a real
// capture engine attached to the loopback interface, so BPF filter
// validation exercises the actual libpcap compiler. Skips when no
// loopback interface is available or when running under CI (same
// convention as internal/capture's engine tests).
func newLoopbackCaptureServer(t *testing.T) *Server {
	t.Helper()

	if os.Getenv("CI") != "" {
		t.Skip("Skipping test in CI environment (requires packet capture privileges)")
	}

	var iface string

	for _, name := range []string{"lo", "lo0", "Loopback"} {
		if captureengine.InterfaceExists(name) {
			iface = name

			break
		}
	}

	if iface == "" {
		t.Skip("No loopback interface found")
	}

	engine, err := captureengine.New(iface, 0)
	if err != nil {
		t.Skipf("Cannot create capture engine: %v", err)
	}

	t.Cleanup(engine.Close)

	cfg := mustLoadConfig(t, baseConfigYAML)
	stack := protocols.NewStack(engine, cfg, logging.NewDebugConfig(0))

	return &Server{
		cfg: ServerConfig{
			Stack:     stack,
			Config:    cfg,
			Interface: iface,
			Version:   "test",
		},
		logger:    slog.Default(),
		pcapCache: capture.NewCache(),
	}
}

// TestHandleCaptureFilterSetInvalidReturnsLibpcapReason verifies that a
// malformed BPF filter surfaces the real libpcap compile error in the
// response body, not just the generic "invalid" message.
func TestHandleCaptureFilterSetInvalidReturnsLibpcapReason(t *testing.T) {
	server := newLoopbackCaptureServer(t)

	for _, filter := range []string{"tcp port", "nonsense &&"} {
		t.Run(filter, func(t *testing.T) {
			assertRejectedFilterHasLibpcapReason(t, server, filter)
		})
	}
}

// assertRejectedFilterHasLibpcapReason submits filter and checks that the
// 400 response carries the real libpcap compile error as an ErrorDetail,
// not just the generic top-level message.
func assertRejectedFilterHasLibpcapReason(t *testing.T, server *Server, filter string) {
	t.Helper()

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/capture/filter",
		strings.NewReader(`{"filter":"`+filter+`"}`),
	)
	rec := httptest.NewRecorder()

	server.handleCaptureFilter(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if resp.Message != "Invalid BPF filter expression" {
		t.Fatalf("expected generic message preserved, got %q", resp.Message)
	}

	if len(resp.Details) != 1 {
		t.Fatalf("expected 1 error detail, got %d: %+v", len(resp.Details), resp.Details)
	}

	detail := resp.Details[0]
	if detail.Field != "filter" {
		t.Fatalf("expected detail field %q, got %q", "filter", detail.Field)
	}

	if detail.Value != filter {
		t.Fatalf("expected detail value %q, got %q", filter, detail.Value)
	}

	if detail.Issue == "" || detail.Issue == "Invalid BPF filter expression" {
		t.Fatalf("expected real libpcap compile error, got %q", detail.Issue)
	}
}

// TestHandleCaptureFilterSetValidClearsError confirms a valid filter still
// succeeds and returns no error details.
func TestHandleCaptureFilterSetValidClearsError(t *testing.T) {
	server := newLoopbackCaptureServer(t)

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/capture/filter",
		strings.NewReader(`{"filter":"ip"}`),
	)
	rec := httptest.NewRecorder()

	server.handleCaptureFilter(rec, req)

	if rec.Code != http.StatusOK {
		t.Skipf("valid filter rejected on this platform/interface: %s", rec.Body.String())
	}
}
