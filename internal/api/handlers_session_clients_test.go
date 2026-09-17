package api

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// GET /api/v1/sessions/{id}/clients is the read surface for AP-1b's
// observed-client table. It is session-scoped because the table is: two
// scenarios running at once each answer for their own wire.
//
// The stacks here run on a real transport carrying real frames rather than on
// a test-only setter, so what the endpoint serves is what the wire produced.

var errNoMoreFrames = errors.New("replay transport: no more frames")

// replayTransport hands a stack a fixed list of frames and then goes quiet.
type replayTransport struct {
	mu     sync.Mutex
	frames [][]byte
}

func (t *replayTransport) ReadPacket(buf []byte) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.frames) == 0 {
		// A busy receive loop would spin a core for the whole test.
		time.Sleep(time.Millisecond)
		return nil, errNoMoreFrames
	}
	frame := t.frames[0]
	t.frames = t.frames[1:]
	return append(buf[:0], frame...), nil
}

func (t *replayTransport) SendPacket([]byte) error { return nil }
func (t *replayTransport) SetFilter(string) error  { return nil }
func (t *replayTransport) Filter() string          { return "" }

// arpFrom builds a minimal ARP request announcing mac at ip.
func arpFrom(mac net.HardwareAddr, ip string) []byte {
	frame := make([]byte, 14+28)
	copy(frame[0:6], net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	copy(frame[6:12], mac)
	frame[12], frame[13] = 0x08, 0x06
	arp := frame[14:]
	arp[1], arp[3] = 0x01, 0x00
	arp[2] = 0x08
	arp[4], arp[5] = 6, 4
	arp[7] = 0x01
	copy(arp[8:14], mac)
	copy(arp[14:18], net.ParseIP(ip).To4())
	copy(arp[24:28], net.ParseIP("192.0.2.1").To4())
	return frame
}

// runningSession starts a session whose wire carries frames and registers it
// on server under id. The stack is stopped when the test ends.
func runningSession(t *testing.T, server *Server, id string, frames [][]byte) {
	t.Helper()
	cfg := &config.Config{Devices: []config.Device{{
		Name:        "MED-ACC-SW01",
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.0.2.1")},
	}}}
	stack := protocols.NewStackWithTransport(&replayTransport{frames: frames}, cfg, logging.NewDebugConfig(0))
	if err := stack.Start(); err != nil {
		t.Fatalf("%s: Start() = %v", id, err)
	}
	t.Cleanup(stack.Stop)
	server.simulations[id] = simulationAPIState{config: cfg, iface: "eth0", stack: stack}
}

// clientsAfter polls the endpoint until it reports want entries, so the test
// does not race the stack's decode thread. The condition is checked before the
// deadline is consulted, so a fast result is never reported as a timeout.
func clientsAfter(t *testing.T, server *Server, id string, want int) []protocols.ObservedClient {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		recorder := sessionRequest(t, server, "/api/v1/sessions/"+id+"/clients")
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", id, recorder.Code)
		}
		var clients []protocols.ObservedClient
		if err := json.Unmarshal(recorder.Body.Bytes(), &clients); err != nil {
			t.Fatalf("%s: decode %q: %v", id, recorder.Body.String(), err)
		}
		if len(clients) == want {
			return clients
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: %d clients after 5s, want %d", id, len(clients), want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestSessionClientsReportsWhatTheWireShowed(t *testing.T) {
	server := &Server{simulations: map[string]simulationAPIState{}}
	runningSession(t, server, "hospital", [][]byte{
		arpFrom(net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x22}, "192.0.2.60"),
		arpFrom(net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x11}, "192.0.2.50"),
	})

	clients := clientsAfter(t, server, "hospital", 2)

	// Sorted by MAC, so a poller does not see the map's iteration order.
	if clients[0].MAC != "02:00:00:00:00:11" || clients[1].MAC != "02:00:00:00:00:22" {
		t.Fatalf("clients = %q, %q; want them ordered by MAC", clients[0].MAC, clients[1].MAC)
	}
	if clients[0].IP != "192.0.2.50" || clients[1].IP != "192.0.2.60" {
		t.Errorf("IPs = %q, %q; want 192.0.2.50, 192.0.2.60", clients[0].IP, clients[1].IP)
	}
	if clients[0].Frames != 1 {
		t.Errorf("Frames = %d, want 1", clients[0].Frames)
	}
	if clients[0].FirstSeen.IsZero() || clients[0].LastSeen.IsZero() || clients[0].ExpireAt.IsZero() {
		t.Errorf("timestamps not reported: %+v", clients[0])
	}
}

func TestSessionClientsAreEmptyBeforeTheSessionServesTraffic(t *testing.T) {
	server := serverWithSessions(map[string][]string{"hospital": {"MED-CORE-SW01"}})

	recorder := sessionRequest(t, server, "/api/v1/sessions/hospital/clients")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	// An absent table is an empty list, never JSON null, so a consumer can
	// iterate the response without a nil check.
	if body := recorder.Body.String(); body != "[]\n" && body != "[]" {
		t.Errorf("body = %q, want an empty JSON array", body)
	}
}

func TestSessionClientsAnswerPerSession(t *testing.T) {
	server := &Server{simulations: map[string]simulationAPIState{}}
	runningSession(t, server, "hospital", [][]byte{
		arpFrom(net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x33}, "192.0.2.70"),
	})
	runningSession(t, server, "warehouse", nil)

	if clients := clientsAfter(t, server, "hospital", 1); clients[0].MAC != "02:00:00:00:00:33" {
		t.Errorf("hospital client = %q", clients[0].MAC)
	}
	clientsAfter(t, server, "warehouse", 0)
}

func TestSessionClientsIsReadOnly(t *testing.T) {
	// The table is a wire observation; nothing may post to it.
	server := serverWithSessions(map[string][]string{"hospital": {"MED-CORE-SW01"}})

	recorder := httptest.NewRecorder()
	server.dispatchSessionSubpath(
		recorder,
		httptest.NewRequest(http.MethodPost, "/api/v1/sessions/hospital/clients", nil),
	)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", recorder.Code)
	}
	if allow := recorder.Header().Get("Allow"); allow != http.MethodGet {
		t.Errorf("Allow = %q, want GET", allow)
	}
}
