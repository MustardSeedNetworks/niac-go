package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleTemplateUseRejectsUnknownField pins P1-11 item 1: handleTemplateUse
// must decode via decodeJSONStrict like every other mutating handler, so a
// typoed or unexpected field is a 400, not silently ignored.
func TestHandleTemplateUseRejectsUnknownField(t *testing.T) {
	server, _ := newTestServer(t)

	body := []byte(`{"templateName": "hospital", "bogusField": "x"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/use", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.handleTemplateUse(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d: %s", rec.Code, rec.Body.String())
	}
}
