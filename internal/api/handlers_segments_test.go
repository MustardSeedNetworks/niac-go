package api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func decodeSegments(t *testing.T, server *Server) []segmentResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/segments", nil)
	server.handleSegments(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var out []segmentResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestHandleSegmentsMultiVLAN(t *testing.T) {
	server, _ := newTestServer(t)
	// Two segments reusing the same IP on different tags — the isolation the
	// segment demux provides. The endpoint must keep them grouped separately.
	server.cfg.Config = &config.Config{
		Segments: []config.Segment{
			{Tag: 200, Devices: []config.Device{
				{Name: "sw-200", Type: "switch", IPAddresses: []net.IP{net.ParseIP("10.0.0.1")}},
			}},
			{Tag: 300, Devices: []config.Device{
				{Name: "sw-300", Type: "switch", IPAddresses: []net.IP{net.ParseIP("10.0.0.1")}},
				{Name: "rtr-300", Type: "router"},
			}},
		},
	}

	segs := decodeSegments(t, server)
	if len(segs) != 2 {
		t.Fatalf("segments = %d, want 2 (%+v)", len(segs), segs)
	}
	if segs[0].VLANTag != 200 || len(segs[0].Devices) != 1 {
		t.Errorf("segment[0] = %+v, want tag 200 / 1 device", segs[0])
	}
	if segs[1].VLANTag != 300 || len(segs[1].Devices) != 2 {
		t.Errorf("segment[1] = %+v, want tag 300 / 2 devices", segs[1])
	}
	if segs[0].Untagged || segs[1].Untagged {
		t.Errorf("tagged segments marked untagged: %+v", segs)
	}
	if segs[0].Devices[0]["name"] != "sw-200" {
		t.Errorf("device name = %v, want sw-200", segs[0].Devices[0]["name"])
	}
}

func TestHandleSegmentsFlatConfigIsUntagged(t *testing.T) {
	server, _ := newTestServer(t)
	server.cfg.Config = &config.Config{
		Devices: []config.Device{{Name: "d1", Type: "router"}, {Name: "d2", Type: "switch"}},
	}

	segs := decodeSegments(t, server)
	if len(segs) != 1 {
		t.Fatalf("segments = %d, want 1", len(segs))
	}
	if segs[0].VLANTag != config.UntaggedTag || !segs[0].Untagged {
		t.Errorf("flat config segment = %+v, want untagged tag 0", segs[0])
	}
	if len(segs[0].Devices) != 2 {
		t.Errorf("devices = %d, want 2", len(segs[0].Devices))
	}
}

func TestHandleSegmentsNilConfig(t *testing.T) {
	server, _ := newTestServer(t)
	server.cfg.Config = nil

	if segs := decodeSegments(t, server); len(segs) != 0 {
		t.Errorf("nil config segments = %d, want 0", len(segs))
	}
}
