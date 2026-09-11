package api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// A session running a real stack, so the checkpoint handlers act on device
// state rather than on a stub that cannot disagree with the daemon.
func serverWithRunningSession(t *testing.T) (*Server, *protocols.Stack) {
	t.Helper()
	const id = "hospital"
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1", IPAddresses: []net.IP{{192, 0, 2, 1}},
		Interfaces: []config.Interface{{Name: "Gi0/1", Address: "192.0.2.1/24", Speed: 100}},
		TrunkPorts: []config.TrunkPort{{Interface: "Gi0/1"}},
		SNMPConfig: config.SNMPConfig{Community: "public"},
	}}}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	server := &Server{simulations: map[string]simulationAPIState{
		id: {config: cfg, iface: "eth0", stack: stack},
	}}
	return server, stack
}

func checkpointCall(
	t *testing.T, server *Server, method, path, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.dispatchSessionSubpath(recorder, request)
	return recorder
}

func TestSessionCheckpointSaveListAndRestore(t *testing.T) {
	server, stack := serverWithRunningSession(t)
	base := "/api/v1/sessions/hospital/checkpoints"

	recorder := checkpointCall(t, server, http.MethodPost, base, `{"name":"healthy"}`)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("save status = %d (%s), want 201", recorder.Code, recorder.Body)
	}

	recorder = checkpointCall(t, server, http.MethodGet, base, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status = %d (%s), want 200", recorder.Code, recorder.Body)
	}
	var listed struct {
		Checkpoints []string `json:"checkpoints"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listed.Checkpoints) != 1 || listed.Checkpoints[0] != "healthy" {
		t.Fatalf("checkpoints = %v, want [healthy]", listed.Checkpoints)
	}

	if err := stack.SetDeviceFault("edge-1", devicestate.FaultLatency, 250); err != nil {
		t.Fatalf("SetDeviceFault: %v", err)
	}

	recorder = checkpointCall(t, server, http.MethodPost, base+"/restore", `{"name":"healthy"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("restore status = %d (%s), want 200", recorder.Code, recorder.Body)
	}
	if active := stack.ActiveDeviceFaults(); len(active) != 0 {
		t.Fatalf("restore left faults active: %#v", active)
	}
}

// Restoring a name nothing saved is the harness's own mistake, and answering
// 200 would let an acceptance run assert against a scenario it never reset.
func TestSessionCheckpointRestoreUnknownNameIs404(t *testing.T) {
	server, _ := serverWithRunningSession(t)

	recorder := checkpointCall(t, server,
		http.MethodPost, "/api/v1/sessions/hospital/checkpoints/restore", `{"name":"absent"}`)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d (%s), want 404", recorder.Code, recorder.Body)
	}
}

func TestSessionCheckpointRejectsAnEmptyName(t *testing.T) {
	server, _ := serverWithRunningSession(t)

	recorder := checkpointCall(t, server,
		http.MethodPost, "/api/v1/sessions/hospital/checkpoints", `{"name":""}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d (%s), want 400", recorder.Code, recorder.Body)
	}
}

// A session that exists but serves nothing cannot hold a checkpoint; saying so
// is better than a 500 from a nil stack.
func TestSessionCheckpointWithoutAStackIs409(t *testing.T) {
	server := &Server{simulations: map[string]simulationAPIState{
		"idle": {config: &config.Config{}, iface: "eth0"},
	}}

	recorder := checkpointCall(t, server,
		http.MethodPost, "/api/v1/sessions/idle/checkpoints", `{"name":"healthy"}`)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d (%s), want 409", recorder.Code, recorder.Body)
	}
}

func TestSessionCheckpointRejectsAnUnsupportedMethod(t *testing.T) {
	server, _ := serverWithRunningSession(t)

	recorder := checkpointCall(t, server,
		http.MethodPut, "/api/v1/sessions/hospital/checkpoints", `{"name":"healthy"}`)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d (%s), want 405", recorder.Code, recorder.Body)
	}
}
