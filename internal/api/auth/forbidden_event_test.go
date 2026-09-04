// SPDX-License-Identifier: BUSL-1.1

package auth_test

// forbidden_event_test.go pins the fleet-wide SIEM contract from #1257: an
// authorization denial in seed, stem or niac emits event=auth.forbidden, and
// `reason` says which mechanism denied it — `scope` here, `role` in seed
// (TestRequireRole_EmitsForbiddenEvent). One rule filters authz denials across
// the fleet; a detection that cares about the mechanism splits on reason.
//
// Without a fixture on both sides, a field rename on either one silently breaks
// every rule downstream, and nothing in either repo notices.

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/auth"
	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
)

// captureLogs runs fn with a JSON logger and returns each record as a map.
func captureLogs(t *testing.T, fn func(*slog.Logger)) []map[string]any {
	t.Helper()

	var buf strings.Builder
	fn(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	var records []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("log line is not JSON: %v (%s)", err, line)
		}
		records = append(records, record)
	}

	return records
}

func findForbidden(t *testing.T, records []map[string]any) map[string]any {
	t.Helper()
	for _, record := range records {
		if record["event"] == "auth.forbidden" {
			return record
		}
	}
	t.Fatalf("no event=auth.forbidden record: %v", records)

	return nil
}

// TestAdminProtect_EmitsForbiddenEvent covers the admin-scope denial.
func TestAdminProtect_EmitsForbiddenEvent(t *testing.T) {
	t.Parallel()

	var status int
	records := captureLogs(t, func(logger *slog.Logger) {
		gated := auth.AdminProtect(logger, testClientIP, recordErr(&status),
			func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

		req := httptest.NewRequest(http.MethodPost, "/api/v1/config/import", nil)
		req = req.WithContext(auth.WithScope(req.Context(), tokenstore.ScopeReadWrite))
		gated(httptest.NewRecorder(), req)
	})

	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}

	forbidden := findForbidden(t, records)
	if forbidden["reason"] != "scope" {
		t.Errorf("reason = %v, want scope — seed emits role on the same event", forbidden["reason"])
	}
	for _, field := range []string{"reason", "clientIP", "path", "method", "requiredScope"} {
		if _, ok := forbidden[field]; !ok {
			t.Errorf("event=auth.forbidden missing field %q: %v", field, forbidden)
		}
	}
	if forbidden["method"] != http.MethodPost {
		t.Errorf("method = %v, want POST", forbidden["method"])
	}
	if forbidden["path"] != "/api/v1/config/import" {
		t.Errorf("path = %v, want /api/v1/config/import", forbidden["path"])
	}
}
