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

// TestHandleSegmentsGroupsFlatConfigByDeviceVLAN guards D12.
//
// None of the shipped scenario packs use an explicit `segments:` block; they
// carry VLAN membership per device. The handler used to fall straight through
// to NormalizedSegments(), whose "bare devices = one untagged segment"
// compatibility branch was therefore taken 100% of the time — a config the
// daemon reports as {200:40, 210:7, 240:6, None:3} rendered as a single
// "Untagged" bucket on a page whose entire purpose is per-VLAN grouping.
//
// Tag 0 stays a real bucket for genuinely untagged devices, so a mixed config
// reports both rather than one swallowing the other.
func TestHandleSegmentsGroupsFlatConfigByDeviceVLAN(t *testing.T) {
	server, _ := newTestServer(t)
	server.cfg.Config = &config.Config{
		Devices: []config.Device{
			{Name: "mgmt-1", Type: "switch", VLAN: 200},
			{Name: "mgmt-2", Type: "switch", VLAN: 200},
			{Name: "data-1", Type: "host", VLAN: 210},
			{Name: "native-1", Type: "router"}, // genuinely untagged
		},
	}

	segs := decodeSegments(t, server)
	if len(segs) != 3 {
		t.Fatalf("segments = %d, want 3 (untagged, 200, 210): %+v", len(segs), segs)
	}

	// Untagged first, then ascending by tag, so the view is stable.
	if segs[0].VLANTag != config.UntaggedTag || !segs[0].Untagged {
		t.Errorf("segs[0] = %+v, want the untagged bucket", segs[0])
	}
	if len(segs[0].Devices) != 1 {
		t.Errorf("untagged devices = %d, want 1", len(segs[0].Devices))
	}
	if segs[1].VLANTag != 200 || len(segs[1].Devices) != 2 {
		t.Errorf("segs[1] = tag %d with %d devices, want tag 200 with 2", segs[1].VLANTag, len(segs[1].Devices))
	}
	if segs[2].VLANTag != 210 || len(segs[2].Devices) != 1 {
		t.Errorf("segs[2] = tag %d with %d devices, want tag 210 with 1", segs[2].VLANTag, len(segs[2].Devices))
	}
	if segs[1].Untagged || segs[2].Untagged {
		t.Error("tagged segments must not be marked untagged")
	}
}

// TestHandleSegmentsExplicitBlockWins keeps the engine's own binding
// authoritative: a config that declares `segments:` is grouped by that, not
// re-derived from per-device VLAN.
func TestHandleSegmentsExplicitBlockWins(t *testing.T) {
	server, _ := newTestServer(t)
	server.cfg.Config = &config.Config{
		Segments: []config.Segment{
			{Tag: 300, Devices: []config.Device{{Name: "a", Type: "router", VLAN: 999}}},
		},
	}

	segs := decodeSegments(t, server)
	if len(segs) != 1 || segs[0].VLANTag != 300 {
		t.Fatalf("segments = %+v, want a single tag-300 segment from the explicit block", segs)
	}
}

func TestHandleSegmentsNilConfig(t *testing.T) {
	server, _ := newTestServer(t)
	server.cfg.Config = nil

	if segs := decodeSegments(t, server); len(segs) != 0 {
		t.Errorf("nil config segments = %d, want 0", len(segs))
	}
}
