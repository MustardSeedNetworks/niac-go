package api

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
)

func TestConfigReplacementScope(t *testing.T) {
	for _, scope := range []tokenstore.TokenScope{tokenstore.ScopeReadOnly, tokenstore.ScopeReadWrite, tokenstore.ScopeAdmin} {
		for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodPost} {
			t.Run(scope.String()+"/"+method, func(t *testing.T) {
				checkConfigReplacementScope(t, scope, method)
			})
		}
	}
}

func checkConfigReplacementScope(t *testing.T, scope tokenstore.TokenScope, method string) {
	t.Helper()
	server, _, token := newLibraryInstallServer(t, scope)
	var logs bytes.Buffer
	server.logger = slog.New(slog.NewJSONHandler(&logs, nil))
	mux := server.apiHandler()
	before := readScopeConfig(t, server)
	beforeConfig := server.cfg.Config
	req := httptest.NewRequest(method, "/api/v1/config",
		strings.NewReader(`{"content":`+strconvJSON(updatedConfigYAML)+`}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Csrf-Token", testCSRFToken(t, server, token))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	denied := method != http.MethodGet && scope != tokenstore.ScopeAdmin
	wantStatus := http.StatusOK
	if denied {
		wantStatus = http.StatusForbidden
	}
	if rec.Code != wantStatus {
		t.Errorf("status=%d want=%d body=%s", rec.Code, wantStatus, rec.Body.String())
	}
	after := readScopeConfig(t, server)
	if denied || method == http.MethodGet {
		if !bytes.Equal(before, after) || server.cfg.Config != beforeConfig {
			t.Error("non-mutating request changed configuration")
		}
	} else if !bytes.Equal(after, []byte(updatedConfigYAML)) || server.cfg.Config == beforeConfig {
		t.Error("admin replacement did not update disk and runtime configuration")
	}
	if denied && (!strings.Contains(logs.String(), `"event":"auth.forbidden"`) ||
		!strings.Contains(logs.String(), `"reason":"scope"`)) {
		t.Errorf("missing scope-denial audit: %s", logs.String())
	}
}

func readScopeConfig(t *testing.T, server *Server) []byte {
	t.Helper()
	data, err := os.ReadFile(server.configPath())
	if err != nil {
		t.Fatal(err)
	}
	return data
}
