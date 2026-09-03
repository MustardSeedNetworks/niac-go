package ipc

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/topology"
)

// newFakeServer stands up a Unix socket that answers each connection with one
// JSON response, so the client can be exercised end to end without a daemon.
//
// The socket lives in a deliberately short directory: sun_path is capped near
// 104 bytes on macOS, and t.TempDir()'s path plus a long test name overruns it,
// which surfaces as a bind failure rather than anything about the client.
func newFakeServer(t *testing.T, handler func(Request) Response) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "n")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	socketPath := filepath.Join(dir, "s")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen on %s: %v", socketPath, err)
	}

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return // listener closed by cleanup
			}

			go func() {
				defer func() { _ = conn.Close() }()

				var req Request
				if decodeErr := json.NewDecoder(conn).Decode(&req); decodeErr != nil {
					return
				}

				_ = json.NewEncoder(conn).Encode(handler(req))
			}()
		}
	}()

	t.Cleanup(func() {
		_ = listener.Close()
		wg.Wait()
	})

	return socketPath
}

// okResponse builds a successful response carrying one keyed payload, which is
// the shape every client getter unwraps.
func okResponse(key string, value any) Response {
	return Response{Success: true, Data: map[string]any{key: value}}
}

func TestNewClientUsesTheGivenSocketAndDefaultTimeout(t *testing.T) {
	client := NewClient("/tmp/example.sock")

	if got := client.SocketPath(); got != "/tmp/example.sock" {
		t.Errorf("SocketPath() = %q, want /tmp/example.sock", got)
	}

	client.SetTimeout(3 * time.Second)

	if client.timeout != 3*time.Second {
		t.Errorf("timeout after SetTimeout = %v, want 3s", client.timeout)
	}
}

func TestDefaultClientTargetsTheDefaultSocketPath(t *testing.T) {
	if got, want := DefaultClient().SocketPath(), GetDefaultSocketPath(); got != want {
		t.Errorf("DefaultClient socket = %q, want %q", got, want)
	}

	if GetDefaultSocketPath() != DefaultSocketPath() {
		t.Error("GetDefaultSocketPath and DefaultSocketPath disagree")
	}
}

// A missing socket is the normal case when no daemon is running, so it must
// surface as an error rather than a panic or a hang.
func TestSendCommandFailsWhenNoDaemonIsListening(t *testing.T) {
	client := NewClient(filepath.Join(t.TempDir(), "absent.sock"))
	client.SetTimeout(200 * time.Millisecond)

	if _, err := client.SendCommand(CommandStatus, nil); err == nil {
		t.Fatal("SendCommand against a missing socket = nil error, want a connection failure")
	}
}

func TestSendCommandRoundTripsCommandAndArgs(t *testing.T) {
	var (
		mu   sync.Mutex
		seen Request
	)

	path := newFakeServer(t, func(req Request) Response {
		mu.Lock()
		seen = req
		mu.Unlock()

		return Response{Success: true}
	})

	client := NewClient(path)

	resp, err := client.SendCommand(CommandDump, map[string]any{"count": 5})
	if err != nil {
		t.Fatalf("SendCommand: %v", err)
	}

	if !resp.Success {
		t.Error("Success = false, want true")
	}

	mu.Lock()
	defer mu.Unlock()

	if seen.Command != CommandDump {
		t.Errorf("server saw command %q, want %q", seen.Command, CommandDump)
	}

	if seen.Args["count"] != float64(5) {
		t.Errorf("server saw args %v, want count=5", seen.Args)
	}
}

func TestGetStatusDecodesTheStatusPayload(t *testing.T) {
	path := newFakeServer(t, func(Request) Response {
		return okResponse("status", StatusData{
			Running:     true,
			Interface:   "eth0",
			DeviceCount: 75,
			PacketsRX:   1234,
		})
	})

	status, err := NewClient(path).GetStatus()
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if !status.Running || status.Interface != "eth0" || status.DeviceCount != 75 || status.PacketsRX != 1234 {
		t.Errorf("GetStatus() = %+v, want running eth0 with 75 devices and 1234 packets", status)
	}
}

// An unsuccessful response must not be reported as an empty-but-valid status:
// a caller checking only err would otherwise render a stopped simulation as
// running with zero devices.
func TestGetStatusReportsAServerSideFailure(t *testing.T) {
	path := newFakeServer(t, func(Request) Response {
		return Response{Success: false, Error: "no simulation"}
	})

	if _, err := NewClient(path).GetStatus(); !errors.Is(err, ErrStatusCommandFailed) {
		t.Fatalf("GetStatus error = %v, want ErrStatusCommandFailed", err)
	}
}

func TestGetStatusRejectsAResponseMissingItsPayload(t *testing.T) {
	path := newFakeServer(t, func(Request) Response {
		return Response{Success: true, Data: map[string]any{}}
	})

	if _, err := NewClient(path).GetStatus(); !errors.Is(err, ErrMissingStatusData) {
		t.Fatalf("GetStatus error = %v, want ErrMissingStatusData", err)
	}
}

func TestReloadShutdownAndPingSucceedOnAnOKResponse(t *testing.T) {
	// Ping and IsRunning both go through GetStatus, so the status command has to
	// carry a payload even though reload and shutdown do not.
	path := newFakeServer(t, func(req Request) Response {
		if req.Command == CommandStatus {
			return okResponse("status", StatusData{Running: true})
		}

		return Response{Success: true}
	})

	client := NewClient(path)

	if err := client.Reload(); err != nil {
		t.Errorf("Reload: %v", err)
	}

	if err := client.Shutdown(); err != nil {
		t.Errorf("Shutdown: %v", err)
	}

	if err := client.Ping(); err != nil {
		t.Errorf("Ping: %v", err)
	}

	if !client.IsRunning() {
		t.Error("IsRunning() = false while the server is answering")
	}
}

func TestIsRunningIsFalseWithoutADaemon(t *testing.T) {
	client := NewClient(filepath.Join(t.TempDir(), "absent.sock"))
	client.SetTimeout(200 * time.Millisecond)

	if client.IsRunning() {
		t.Error("IsRunning() = true with no socket, want false")
	}
}

func TestDumpPacketsDecodesCapturedFrames(t *testing.T) {
	path := newFakeServer(t, func(Request) Response {
		return okResponse("packets", []PacketData{
			{Length: 64, Device: "SW1", Interface: "eth0", Data: []byte{0xde, 0xad}},
		})
	})

	packets, err := NewClient(path).DumpPackets("SW1", "eth0", 1)
	if err != nil {
		t.Fatalf("DumpPackets: %v", err)
	}

	if len(packets) != 1 {
		t.Fatalf("got %d packets, want 1", len(packets))
	}

	if packets[0].Length != 64 || packets[0].Device != "SW1" {
		t.Errorf("packet = %+v, want length 64 from SW1", packets[0])
	}
}

func TestGetNeighborsDecodesDiscoveryEntries(t *testing.T) {
	path := newFakeServer(t, func(Request) Response {
		return okResponse("neighbors", []NeighborData{
			{Protocol: "LLDP", LocalDevice: "SW1", RemoteDevice: "R1", RemotePort: "Gi0/1"},
		})
	})

	neighbors, err := NewClient(path).GetNeighbors()
	if err != nil {
		t.Fatalf("GetNeighbors: %v", err)
	}

	if len(neighbors) != 1 || neighbors[0].RemoteDevice != "R1" {
		t.Fatalf("neighbors = %+v, want one entry for R1", neighbors)
	}
}

// A daemon that has discovered nothing sends a null rather than an empty list.
// Returning an error there would make "no neighbours yet" indistinguishable
// from a broken daemon.
func TestGetNeighborsTreatsNullAsEmpty(t *testing.T) {
	path := newFakeServer(t, func(Request) Response {
		return okResponse("neighbors", nil)
	})

	neighbors, err := NewClient(path).GetNeighbors()
	if err != nil {
		t.Fatalf("GetNeighbors: %v", err)
	}

	if len(neighbors) != 0 {
		t.Errorf("neighbors = %+v, want empty", neighbors)
	}
}

func TestGetLogsPassesLevelAndCountAndDecodesEntries(t *testing.T) {
	var (
		mu   sync.Mutex
		seen Request
	)

	path := newFakeServer(t, func(req Request) Response {
		mu.Lock()
		seen = req
		mu.Unlock()

		return okResponse("logs", []LogEntry{
			{Level: LogLevelWarn, Message: "link down", Source: "LLDP"},
		})
	})

	logs, err := NewClient(path).GetLogs("warn", 10)
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}

	if len(logs) != 1 || logs[0].Message != "link down" {
		t.Fatalf("logs = %+v, want one 'link down' entry", logs)
	}

	mu.Lock()
	defer mu.Unlock()

	if seen.Args["level"] != "warn" || seen.Args["count"] != float64(10) {
		t.Errorf("server saw args %v, want level=warn count=10", seen.Args)
	}
}

func TestGetLogsReportsAServerSideFailure(t *testing.T) {
	path := newFakeServer(t, func(Request) Response {
		return Response{Success: false, Error: "unavailable"}
	})

	if _, err := NewClient(path).GetLogs("", 0); !errors.Is(err, ErrLogsCommandFailed) {
		t.Fatalf("GetLogs error = %v, want ErrLogsCommandFailed", err)
	}
}

func TestSubscribeLogsDeliversEntriesAndStops(t *testing.T) {
	path := newFakeServer(t, func(Request) Response {
		return okResponse("logs", []LogEntry{
			{Timestamp: time.Unix(1, 0), Level: LogLevelInfo, Message: "first"},
		})
	})

	subscription := NewClient(path).SubscribeLogs("", "", 10*time.Millisecond)
	defer subscription.Stop()

	select {
	case entry := <-subscription.Logs():
		if entry.Message != "first" {
			t.Errorf("delivered %q, want \"first\"", entry.Message)
		}
	case err := <-subscription.Errors():
		t.Fatalf("subscription error: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("no log delivered within 3s")
	}
}

// The poller must not re-deliver an entry it has already sent, or a UI tailing
// the stream shows every line once per interval.
func TestSubscribeLogsDoesNotRedeliverTheSameEntry(t *testing.T) {
	path := newFakeServer(t, func(Request) Response {
		return okResponse("logs", []LogEntry{
			{Timestamp: time.Unix(2, 0), Level: LogLevelInfo, Message: "repeat"},
		})
	})

	subscription := NewClient(path).SubscribeLogs("", "", 10*time.Millisecond)
	defer subscription.Stop()

	select {
	case <-subscription.Logs():
	case <-time.After(3 * time.Second):
		t.Fatal("no first delivery")
	}

	select {
	case entry := <-subscription.Logs():
		t.Fatalf("same entry delivered twice: %+v", entry)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestSubscribeLogsSurfacesPollingErrors(t *testing.T) {
	client := NewClient(filepath.Join(t.TempDir(), "absent.sock"))
	client.SetTimeout(100 * time.Millisecond)

	subscription := client.SubscribeLogs("", "", 10*time.Millisecond)
	defer subscription.Stop()

	select {
	case err := <-subscription.Errors():
		if err == nil {
			t.Error("received a nil error")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no error reported for an unreachable daemon")
	}
}

func TestShouldSkipLogFiltersSeenEntriesAndNonMatches(t *testing.T) {
	subscription := &LogSubscription{filter: "lldp"}
	entry := LogEntry{Timestamp: time.Unix(3, 0), Level: LogLevelInfo, Message: "LLDP advertisement"}

	seen := map[string]bool{}
	if subscription.shouldSkipLog(entry, seen) {
		t.Error("first matching entry was skipped")
	}

	seen[subscription.logKey(entry)] = true
	if !subscription.shouldSkipLog(entry, seen) {
		t.Error("already-seen entry was not skipped")
	}

	other := LogEntry{Timestamp: time.Unix(4, 0), Level: LogLevelInfo, Message: "ARP request"}
	if !subscription.shouldSkipLog(other, map[string]bool{}) {
		t.Error("entry not matching the filter was not skipped")
	}
}

// The seen-set is trimmed so a long-running tail does not grow without bound.
func TestCleanupSeenLogsTrimsTheSet(t *testing.T) {
	subscription := &LogSubscription{}

	seen := make(map[string]bool)
	for index := range 1200 {
		seen[string(rune(index))+"k"] = true
	}

	if trimmed := subscription.cleanupSeenLogs(seen); len(trimmed) >= len(seen) {
		t.Errorf("cleanupSeenLogs kept %d of %d entries, want fewer", len(trimmed), len(seen))
	}
}

func TestGetTopologyDecodesNodesAndLinks(t *testing.T) {
	path := newFakeServer(t, func(Request) Response {
		return okResponse("topology", topology.Graph{
			Nodes: []topology.Node{{Name: "SW1", Type: "switch"}, {Name: "R1", Type: "router"}},
			Links: []topology.Link{{}},
		})
	})

	graph, err := NewClient(path).GetTopology()
	if err != nil {
		t.Fatalf("GetTopology: %v", err)
	}

	if len(graph.Nodes) != 2 || graph.Nodes[0].Name != "SW1" || graph.Nodes[1].Type != "router" {
		t.Errorf("nodes = %+v, want SW1 (switch) and R1 (router)", graph.Nodes)
	}

	if len(graph.Links) != 1 {
		t.Errorf("links = %d, want 1", len(graph.Links))
	}
}

func TestGetTopologyReportsAServerSideFailure(t *testing.T) {
	path := newFakeServer(t, func(Request) Response {
		return Response{Success: false, Error: "no topology"}
	})

	if _, err := NewClient(path).GetTopology(); !errors.Is(err, ErrTopologyCommandFailed) {
		t.Fatalf("GetTopology error = %v, want ErrTopologyCommandFailed", err)
	}
}
