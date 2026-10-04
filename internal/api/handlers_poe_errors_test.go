package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// PoE faults arm only on a PSE port NIAC synthesizes. The refusal has to say
// so, because the two ports it refuses look like PoE ports to an operator: a
// switch port with no power behind it, and a port on a capture that carries
// its own POWER-ETHERNET-MIB (niac-go#1972).
func TestHandleErrorsRefusesPoELossOffASynthesizedPSEPort(t *testing.T) {
	captured := filepath.Join(t.TempDir(), "pse.walk")
	if err := os.WriteFile(captured, []byte(""+
		".1.3.6.1.2.1.1.5.0 = STRING: closet-1\r\n"+
		".1.3.6.1.2.1.2.2.1.1.1 = INTEGER: 1\r\n"+
		".1.3.6.1.2.1.2.2.1.2.1 = STRING: Gi0/1\r\n"+
		".1.3.6.1.2.1.2.2.1.3.1 = INTEGER: 6\r\n"+
		".1.3.6.1.2.1.2.2.1.10.1 = Counter32: 5\r\n"+
		".1.3.6.1.2.1.105.1.1.1.3.1.1 = INTEGER: 1\r\n"+
		".1.3.6.1.2.1.105.1.1.1.6.1.1 = INTEGER: 3\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		poe  *config.PoEConfig
		walk string
	}{
		{name: "a port that supplies no power"},
		{name: "a captured PSE table", poe: &config.PoEConfig{BudgetWatts: 370}, walk: captured},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			device := config.Device{
				Name: "closet-1", Type: "switch", IPAddresses: []net.IP{{192, 0, 2, 1}},
				Interfaces: []config.Interface{{Name: "Gi0/1", Address: "192.0.2.1/24", Speed: 100}},
				TrunkPorts: []config.TrunkPort{{Interface: "Gi0/1"}},
				SNMPConfig: config.SNMPConfig{Community: "public", WalkFile: test.walk},
				PoEConfig:  test.poe,
			}
			response := postPoELoss(t, &config.Config{Devices: []config.Device{device}})
			if response.Error != "fault_no_pse_port" {
				t.Errorf("error code = %q, want fault_no_pse_port", response.Error)
			}
			for _, says := range []string{"synthesized PoE port", "supplies no power", "comes from a capture"} {
				if !strings.Contains(response.Message, says) {
					t.Errorf("message %q does not say %q", response.Message, says)
				}
			}
		})
	}
}

func postPoELoss(t *testing.T, cfg *config.Config) ErrorResponse {
	t.Helper()

	server := &Server{
		cfg: ServerConfig{
			Stack:  protocols.NewStack(nil, cfg, logging.NewDebugConfig(0)),
			Config: cfg,
		},
		logger: slog.Default(),
	}
	body, err := json.Marshal(errorInjectionRequest{
		Device: "closet-1", Interface: "Gi0/1", ErrorType: "PoE Loss", Value: new(1),
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	recorder := httptest.NewRecorder()
	server.handleErrors(recorder, httptest.NewRequest(
		http.MethodPost, "/api/v1/errors", bytes.NewReader(body)))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}
	var response ErrorResponse
	if err = json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	return response
}
