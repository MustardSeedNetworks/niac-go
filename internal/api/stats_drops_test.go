package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// lossyTransport is a PacketTransport whose link layer reports ring drops,
// standing in for a libpcap handle that overflowed.
type lossyTransport struct {
	stats capture.Stats
	err   error
}

func (t *lossyTransport) ReadPacket([]byte) ([]byte, error) { return nil, io.EOF }
func (t *lossyTransport) SendPacket([]byte) error           { return nil }
func (t *lossyTransport) SetFilter(string) error            { return nil }
func (t *lossyTransport) Filter() string                    { return "" }
func (t *lossyTransport) Stats() (capture.Stats, error)     { return t.stats, t.err }

func serverWithTransport(t *testing.T, transport protocols.PacketTransport) *Server {
	t.Helper()

	cfg, err := config.LoadYAMLBytes([]byte(`
devices:
  - name: router1
    mac: "00:11:22:33:44:55"
    ips: ["10.0.0.1"]
    type: router
`))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	return &Server{
		cfg: ServerConfig{
			Stack:  protocols.NewStackWithTransport(transport, cfg, logging.NewDebugConfig(0)),
			Config: cfg,
		},
		logger: slog.Default(),
	}
}

// TestHandleStatsReportsRingDrops is D-NIAC-3's acceptance on /api/v1/stats:
// a capture that lost frames in the libpcap ring must not read as complete.
func TestHandleStatsReportsRingDrops(t *testing.T) {
	server := serverWithTransport(t, &lossyTransport{stats: capture.Stats{
		PacketsReceived:  100,
		PacketsDropped:   7,
		PacketsIfDropped: 3,
	}})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	rec := httptest.NewRecorder()
	server.handleStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response struct {
		Stack map[string]uint64 `json:"stack"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got := response.Stack["packetsDropped"]; got != 7 {
		t.Errorf("packetsDropped = %d, want 7", got)
	}
	if got := response.Stack["packetsIfDropped"]; got != 3 {
		t.Errorf("packetsIfDropped = %d, want 3", got)
	}
}

// TestHandleRuntimeReportsRingDrops covers the second unscoped surface the
// defect named; /runtime had the same blind spot as /stats.
func TestHandleRuntimeReportsRingDrops(t *testing.T) {
	server := serverWithTransport(t, &lossyTransport{stats: capture.Stats{
		PacketsReceived: 100, PacketsDropped: 7, PacketsIfDropped: 3,
	}})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runtime", nil)
	rec := httptest.NewRecorder()
	server.handleRuntime(rec, req)

	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got, ok := response["packets_dropped"]; !ok || got.(float64) != 7 {
		t.Errorf("packets_dropped = %v (present=%t), want 7", got, ok)
	}
	if got, ok := response["packets_if_dropped"]; !ok || got.(float64) != 3 {
		t.Errorf("packets_if_dropped = %v (present=%t), want 3", got, ok)
	}
}

// TestStatsTransportWithoutCountersStaysZero: a transport with no link-layer
// counters (a trunk VLAN slice) must report zero, not fail the endpoint.
func TestStatsTransportWithoutCountersStaysZero(t *testing.T) {
	server := serverWithTransport(t, &countlessTransport{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	rec := httptest.NewRecorder()
	server.handleStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response struct {
		Stack map[string]uint64 `json:"stack"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, ok := response.Stack["packetsDropped"]; !ok || got != 0 {
		t.Errorf("packetsDropped = %d (present=%t), want 0", got, ok)
	}
}

type countlessTransport struct{}

func (countlessTransport) ReadPacket([]byte) ([]byte, error) { return nil, io.EOF }
func (countlessTransport) SendPacket([]byte) error           { return nil }
func (countlessTransport) SetFilter(string) error            { return nil }
func (countlessTransport) Filter() string                    { return "" }
