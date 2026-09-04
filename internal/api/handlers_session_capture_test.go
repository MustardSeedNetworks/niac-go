package api

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gopacket/gopacket/pcapgo"

	"github.com/MustardSeedNetworks/niac-go/internal/capturering"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// ethFrame builds a minimal Ethernet frame with the given EtherType so the
// export's BPF filter has something real to select on.
func ethFrame(etherType uint16, tag byte) []byte {
	frame := make([]byte, 60)
	copy(frame, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x02, 0, 0, 0, 0, 1})
	frame[12] = byte(etherType >> 8)
	frame[13] = byte(etherType)
	frame[14] = tag

	return frame
}

func serverWithCapture(t *testing.T, frames [][]byte) *Server {
	t.Helper()
	ring := capturering.New(capturering.Limits{Frames: 16, Bytes: 1 << 20})
	stamp := time.Unix(1_756_000_000, 0).UTC()
	for i, data := range frames {
		ring.OnPacket("rx", &protocols.Packet{
			Buffer:    data,
			Length:    len(data),
			VLAN:      -1,
			Timestamp: stamp.Add(time.Duration(i) * time.Second),
		})
	}

	return &Server{
		logger: slog.Default(),
		simulations: map[string]simulationAPIState{
			"hospital": {config: &config.Config{}, iface: "eth0", capture: ring},
		},
	}
}

func exportRequest(t *testing.T, server *Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.dispatchSessionSubpath(recorder, httptest.NewRequest(http.MethodGet, path, nil))

	return recorder
}

func readPcapng(t *testing.T, body []byte) [][]byte {
	t.Helper()
	reader, err := pcapgo.NewNgReader(bytes.NewReader(body), pcapgo.DefaultNgReaderOptions)
	if err != nil {
		t.Fatalf("NewNgReader: %v", err)
	}
	var out [][]byte
	for {
		data, _, readErr := reader.ReadPacketData()
		if errors.Is(readErr, io.EOF) {
			return out
		}
		if readErr != nil {
			t.Fatalf("ReadPacketData: %v", readErr)
		}
		out = append(out, append([]byte(nil), data...))
	}
}

func TestSessionCaptureExportWritesThePcapngTheRingHolds(t *testing.T) {
	server := serverWithCapture(t, [][]byte{
		ethFrame(0x0806, 1), ethFrame(0x0800, 2), ethFrame(0x0806, 3),
	})

	recorder := exportRequest(t, server, "/api/v1/sessions/hospital/capture/export")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/x-pcapng" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := recorder.Header().Get("Content-Disposition"); !strings.Contains(got, "hospital") {
		t.Errorf("Content-Disposition = %q, want the session name in the filename", got)
	}
	if got := len(readPcapng(t, recorder.Body.Bytes())); got != 3 {
		t.Errorf("frames = %d, want 3", got)
	}
}

func TestSessionCaptureExportAppliesTheBPFFilter(t *testing.T) {
	server := serverWithCapture(t, [][]byte{
		ethFrame(0x0806, 1), ethFrame(0x0800, 2), ethFrame(0x0806, 3),
	})

	recorder := exportRequest(t, server, "/api/v1/sessions/hospital/capture/export?filter=arp")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	frames := readPcapng(t, recorder.Body.Bytes())
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2 ARP frames", len(frames))
	}
	for _, frame := range frames {
		if frame[12] != 0x08 || frame[13] != 0x06 {
			t.Errorf("filter passed a non-ARP frame: ethertype %#x%02x", frame[12], frame[13])
		}
	}
}

func TestSessionCaptureExportLastKeepsTheNewestFrames(t *testing.T) {
	server := serverWithCapture(t, [][]byte{
		ethFrame(0x0806, 1), ethFrame(0x0806, 2), ethFrame(0x0806, 3),
	})

	recorder := exportRequest(t, server, "/api/v1/sessions/hospital/capture/export?last=2")
	frames := readPcapng(t, recorder.Body.Bytes())
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	if frames[0][14] != 2 || frames[1][14] != 3 {
		t.Errorf("last=2 returned frames %d,%d; want 2,3", frames[0][14], frames[1][14])
	}
}

func TestSessionCaptureExportRejectsABadFilterBeforeWritingAnyBody(t *testing.T) {
	server := serverWithCapture(t, [][]byte{ethFrame(0x0806, 1)})

	recorder := exportRequest(t, server, "/api/v1/sessions/hospital/capture/export?filter=nonsense%20%26%26")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Header().Get("Content-Type"), "pcapng") {
		t.Error("a rejected filter still opened a capture stream")
	}
}

func TestSessionCaptureExportRejectsANonNumericLast(t *testing.T) {
	server := serverWithCapture(t, [][]byte{ethFrame(0x0806, 1)})

	recorder := exportRequest(t, server, "/api/v1/sessions/hospital/capture/export?last=lots")
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", recorder.Code)
	}
}

func TestSessionCaptureExportRejectsAWrite(t *testing.T) {
	server := serverWithCapture(t, [][]byte{ethFrame(0x0806, 1)})
	recorder := httptest.NewRecorder()
	server.dispatchSessionSubpath(recorder,
		httptest.NewRequest(http.MethodPost, "/api/v1/sessions/hospital/capture/export", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", recorder.Code)
	}
}

func TestSessionCaptureExportOfASessionWithNoRingIsAnEmptyCapture(t *testing.T) {
	server := &Server{
		logger: slog.Default(),
		simulations: map[string]simulationAPIState{
			"hospital": {config: &config.Config{}, iface: "eth0"},
		},
	}

	recorder := exportRequest(t, server, "/api/v1/sessions/hospital/capture/export")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if got := len(readPcapng(t, recorder.Body.Bytes())); got != 0 {
		t.Errorf("frames = %d, want 0", got)
	}
}
