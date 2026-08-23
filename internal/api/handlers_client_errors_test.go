package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandleClientErrorsAcceptsReporterPayload guards D7.
//
// ui/src/utils/error-reporter.ts has always POSTed batches to
// /api/v1/client-errors, but the route was never registered — every flush
// returned 404, and the reporter's .catch() could not soften it because a 404
// is a resolved fetch rather than a rejection. No client-side error had ever
// reached the server, including the wizard crash the sweep found.
//
// The body here is the reporter's real shape, not a Go struct literal: the
// whole failure mode was a route/contract mismatch, which a struct-built
// fixture cannot express.
func TestHandleClientErrorsAcceptsReporterPayload(t *testing.T) {
	server := &Server{logger: slog.Default()}

	body := `{"errors":[{"message":"Cannot read properties of null (reading 'length')",` +
		`"stack":"at PreflightStep","context":"react-error-boundary",` +
		`"timestamp":1755830000000,"url":"http://127.0.0.1:8445/new-simulation"}]}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/client-errors", strings.NewReader(body))
	server.handleClientErrors(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}

func TestHandleClientErrorsRejectsOversizedBatch(t *testing.T) {
	server := &Server{logger: slog.Default()}

	entries := make([]string, 0, maxClientErrorBatch+1)
	for range maxClientErrorBatch + 1 {
		entries = append(entries, `{"message":"boom","context":"x","timestamp":1,"url":"u"}`)
	}
	body := `{"errors":[` + strings.Join(entries, ",") + `]}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/client-errors", strings.NewReader(body))
	server.handleClientErrors(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
