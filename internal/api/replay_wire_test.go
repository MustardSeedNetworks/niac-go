package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/replay"
)

// wireReplay answers every call with one fixed state and records the request
// the handler hands it, so the tests below pin the replay endpoints' wire
// shape in both directions (#2434).
type wireReplay struct {
	state replay.State
	got   replay.Request
}

func (w *wireReplay) Status() replay.State { return w.state }

func (w *wireReplay) Start(req replay.Request) (replay.State, error) {
	w.got = req

	return w.state, nil
}

func (w *wireReplay) Stop() (replay.State, error) { return w.state, nil }

var fullReplayState = replay.State{
	Running:         true,
	File:            "/srv/pcaps/demo.pcap",
	LoopMs:          250,
	Scale:           1.5,
	RateMode:        "pps",
	Pps:             1000,
	MbpsCap:         50,
	LoopCount:       3,
	BPFFilter:       "udp port 53",
	StartedAt:       time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC),
	PacketsSent:     40,
	BytesSent:       4096,
	PacketsTotal:    100,
	BytesTotal:      10240,
	PercentComplete: 40,
	Passes:          2,
	PacketsFiltered: 7,
}

const fullReplayStateJSON = `{
  "running": true,
  "file": "/srv/pcaps/demo.pcap",
  "loopMs": 250,
  "scale": 1.5,
  "rateMode": "pps",
  "pps": 1000,
  "mbpsCap": 50,
  "loopCount": 3,
  "bpfFilter": "udp port 53",
  "startedAt": "2026-10-04T12:00:00Z",
  "packetsSent": 40,
  "bytesSent": 4096,
  "packetsTotal": 100,
  "bytesTotal": 10240,
  "percentComplete": 40,
  "passes": 2,
  "packetsFiltered": 7
}
`

const zeroReplayStateJSON = `{
  "running": false,
  "file": "",
  "loopMs": 0,
  "scale": 0,
  "packetsSent": 0,
  "bytesSent": 0,
  "packetsTotal": 0,
  "bytesTotal": 0,
  "passes": 0,
  "packetsFiltered": 0
}
`

func TestReplayEndpointsWireShape(t *testing.T) {
	tests := []struct {
		name   string
		method string
		state  replay.State
		want   string
	}{
		{"status full", http.MethodGet, fullReplayState, fullReplayStateJSON},
		{"status zero", http.MethodGet, replay.State{}, zeroReplayStateJSON},
		{"stop full", http.MethodDelete, fullReplayState, fullReplayStateJSON},
		{"stop zero", http.MethodDelete, replay.State{}, zeroReplayStateJSON},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server, _ := newTestServer(t)
			server.cfg.Replay = &wireReplay{state: tc.state}

			rec := httptest.NewRecorder()
			server.handleReplay(rec, httptest.NewRequest(tc.method, "/api/v1/replay", nil))

			if rec.Code != http.StatusOK {
				t.Fatalf("%s /replay = %d: %s", tc.method, rec.Code, rec.Body.String())
			}
			if got := rec.Body.String(); got != tc.want {
				t.Fatalf("%s /replay body:\n%s\nwant:\n%s", tc.method, got, tc.want)
			}
		})
	}
}

func TestReplayStartWireShape(t *testing.T) {
	server, configPath := newTestServer(t)
	configDir := filepath.Dir(configPath)
	pcapPath := filepath.Join(configDir, "demo.pcap")
	if err := os.WriteFile(pcapPath, []byte("pcap"), 0o600); err != nil {
		t.Fatalf("write pcap: %v", err)
	}
	manager := &wireReplay{state: fullReplayState}
	server.cfg.Replay = manager

	body := `{"file":` + strconvJSON(pcapPath) + `,"loopMs":250,"scale":1.5,"rateMode":"pps",` +
		`"pps":1000,"loopCount":3,"bpfFilter":"udp port 53"}`
	rec := httptest.NewRecorder()
	server.handleReplay(rec, httptest.NewRequest(http.MethodPost, "/api/v1/replay", strings.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /replay = %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != fullReplayStateJSON {
		t.Fatalf("POST /replay body:\n%s\nwant:\n%s", got, fullReplayStateJSON)
	}

	want := replay.Request{
		File:      pcapPath,
		RootDir:   configDir,
		LoopMs:    250,
		Scale:     1.5,
		RateMode:  "pps",
		Pps:       1000,
		LoopCount: 3,
		BPFFilter: "udp port 53",
	}
	if manager.got != want {
		t.Fatalf("manager got %+v, want %+v", manager.got, want)
	}
}
