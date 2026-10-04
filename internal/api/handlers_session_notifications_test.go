package api

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// runningPack starts a shipped pack the way the daemon does, bound to its
// compiled fabric, and registers it on server under id.
func runningPack(t *testing.T, server *Server, id, packID string) {
	t.Helper()
	index := slices.IndexFunc(scenario.Packs(), func(pack scenario.Pack) bool { return pack.ID == packID })
	if index < 0 {
		t.Fatalf("no pack %q", packID)
	}
	pack := scenario.Packs()[index]
	result, err := scenario.Generate(pack.Request)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadYAMLBytes(result.YAML)
	if err != nil {
		t.Fatal(err)
	}
	report := fabric.Compile(cfg, fabric.Binding{
		Attachment: pack.Request.AttachmentName, Interface: "eth0",
		Mode: fabric.ModeAccess, AccessVLAN: 200, PolicyApproved: true,
	})
	if !report.Safe {
		t.Fatalf("compile %s: %v", packID, report.Diagnostics)
	}
	stack := protocols.NewStackWithTransport(&replayTransport{}, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&report.Topology)
	if err = stack.Start(); err != nil {
		t.Fatalf("%s: Start() = %v", id, err)
	}
	t.Cleanup(stack.Stop)
	server.simulations[id] = simulationAPIState{config: cfg, iface: "eth0", stack: stack}
}

func sessionNotifications(t *testing.T, server *Server, path string) []protocols.ReceivedNotification {
	t.Helper()
	recorder := sessionRequest(t, server, path)
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s: status = %d, want 200: %s", path, recorder.Code, recorder.Body)
	}
	var notifications []protocols.ReceivedNotification
	if err := json.Unmarshal(recorder.Body.Bytes(), &notifications); err != nil {
		t.Fatalf("%s: decode %q: %v", path, recorder.Body.String(), err)
	}
	return notifications
}

// The hospital pack authors four utilization faults on its imaging path. Each
// is reported to the site collector, MED-NMS01, which is a simulated server:
// before #2410 the switches had no route to it, so all four were lost, and a
// route alone would have put them on the wire.
func TestHospitalCollectorReceivesTheAuthoredFaults(t *testing.T) {
	server := &Server{simulations: map[string]simulationAPIState{}}
	runningPack(t, server, "hospital", "hospital")

	var faults []protocols.ReceivedNotification
	deadline := time.Now().Add(10 * time.Second)
	for {
		faults = faults[:0]
		for _, received := range sessionNotifications(t, server,
			"/api/v1/sessions/hospital/notifications?device=MED-NMS01") {
			if strings.Contains(received.Message, "FAULT_UPDATED") {
				faults = append(faults, received)
			}
		}
		if len(faults) >= 4 || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	senders := make([]string, 0, len(faults))
	for _, received := range faults {
		if received.Receiver != "MED-NMS01" || received.Protocol != "syslog" ||
			!strings.HasPrefix(received.Source, "10.51.200.") {
			t.Errorf("received = %+v", received)
		}
		senders = append(senders, received.Sender)
	}
	slices.Sort(senders)
	want := []string{"MED-ACC-SW02", "MED-ACC-SW02", "MED-DIST-SW01", "MED-DIST-SW02"}
	if !slices.Equal(senders, want) {
		t.Fatalf("MED-NMS01 received FAULT_UPDATED from %q, want %q", senders, want)
	}
}

func TestSessionNotificationsRejectsAnUnknownDevice(t *testing.T) {
	server := serverWithSessions(map[string][]string{"hospital": {"MED-CORE-SW01"}})

	recorder := sessionRequest(t, server, "/api/v1/sessions/hospital/notifications?device=MED-NMS99")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "device_not_found") {
		t.Errorf("body = %q, want device_not_found", body)
	}
}

func TestSessionNotificationsAreEmptyBeforeTheSessionServesTraffic(t *testing.T) {
	server := serverWithSessions(map[string][]string{"hospital": {"MED-CORE-SW01"}})

	if got := sessionNotifications(t, server, "/api/v1/sessions/hospital/notifications"); len(got) != 0 {
		t.Errorf("notifications = %+v, want an empty list", got)
	}
	recorder := sessionRequest(t, server, "/api/v1/sessions/hospital/notifications")
	if body := strings.TrimSpace(recorder.Body.String()); body != "[]" {
		t.Errorf("body = %q, want an empty JSON array", body)
	}
}
