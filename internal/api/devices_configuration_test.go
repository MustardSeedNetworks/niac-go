package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestDeviceListConfigurationPresence(t *testing.T) {
	for _, loaded := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent", true: "empty-but-loaded"}[loaded], func(t *testing.T) {
			server := &Server{logger: slog.Default()}
			if loaded {
				server.cfg.Config = &config.Config{}
			}
			rec := httptest.NewRecorder()
			server.handleDeviceList(rec, httptest.NewRequest(http.MethodGet, "/api/v1/config/devices", nil))
			var body struct {
				ConfigurationLoaded *bool `json:"configurationLoaded"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.ConfigurationLoaded == nil || *body.ConfigurationLoaded != loaded {
				t.Fatalf("configurationLoaded = %v, want %v", body.ConfigurationLoaded, loaded)
			}
		})
	}
}
