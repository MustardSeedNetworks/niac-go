package api

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// sentFrames is the wire a running pack transmits on: it reads nothing and keeps
// every frame the stack sends.
type sentFrames struct {
	replayTransport

	mu   sync.Mutex
	sent [][]byte
}

func (w *sentFrames) SendPacket(frame []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.sent = append(w.sent, slices.Clone(frame))
	return nil
}

// toPort counts the sent UDP datagrams addressed to port.
func (w *sentFrames) toPort(port layers.UDPPort) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	count := 0
	for _, frame := range w.sent {
		packet := gopacket.NewPacket(frame, layers.LayerTypeEthernet, gopacket.Default)
		if udp, ok := packet.Layer(layers.LayerTypeUDP).(*layers.UDP); ok && udp.DstPort == port {
			count++
		}
	}
	return count
}

// runningPack starts a shipped pack the way the daemon does, bound to its
// compiled fabric, and registers it on server under id.
func runningPack(t *testing.T, server *Server, id, packID string) (*protocols.Stack, *sentFrames) {
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
	wire := &sentFrames{}
	stack := protocols.NewStackWithTransport(wire, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&report.Topology)
	if err = stack.Start(); err != nil {
		t.Fatalf("%s: Start() = %v", id, err)
	}
	t.Cleanup(stack.Stop)
	server.simulations[id] = simulationAPIState{config: cfg, iface: "eth0", stack: stack}
	return stack, wire
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
	_, _ = runningPack(t, server, "hospital", "hospital")

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

// A link fault on a pack device is reported to the site collector as the
// standard linkDown trap (#2472), and the collector records it decoded. Before,
// packs authored no traps at all. The core's uplink is the fault: the core
// reaches the collector over its own servers-VLAN SVI, so that path stays up.
func TestHospitalCollectorReceivesTheLinkDownTrap(t *testing.T) {
	const core, uplink = "MED-CORE-SW01", "HundredGigabitEthernet0/0/1"
	server := &Server{simulations: map[string]simulationAPIState{}}
	stack, wire := runningPack(t, server, "hospital", "hospital")

	targets := stack.InterfaceFaultTargets()
	index := slices.IndexFunc(targets, func(target protocols.InterfaceFaultTarget) bool {
		return target.Device == core
	})
	if index < 0 || !slices.Contains(targets[index].Interfaces, uplink) {
		t.Fatalf("%s %s is not a fault target: %+v", core, uplink, targets)
	}
	if err := stack.SetInterfaceFault(targets[index].Address, uplink, devicestate.FaultLinkDown, 1); err != nil {
		t.Fatalf("SetInterfaceFault(%s) = %v", uplink, err)
	}

	var traps []protocols.ReceivedNotification
	deadline := time.Now().Add(10 * time.Second)
	for len(traps) == 0 && time.Now().Before(deadline) {
		for _, received := range sessionNotifications(t, server,
			"/api/v1/sessions/hospital/notifications?device=MED-NMS01") {
			if received.Protocol == "snmp" {
				traps = append(traps, received)
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(traps) != 1 {
		t.Fatalf("MED-NMS01 received %d SNMP notifications, want the one linkDown: %+v", len(traps), traps)
	}
	trap := traps[0]
	if trap.Sender != core || trap.PDU != "trap" || trap.TrapOID != snmp.OIDLinkDown ||
		!strings.HasPrefix(trap.Source, "10.51.") {
		t.Errorf("received = %+v, want a linkDown trap from %s", trap, core)
	}
	if !slices.ContainsFunc(trap.Variables, func(variable protocols.NotificationVariable) bool {
		return strings.HasPrefix(variable.OID, ".1.3.6.1.2.1.2.2.1.2.") && variable.Value == uplink
	}) {
		t.Errorf("variables = %+v, want ifDescr %q", trap.Variables, uplink)
	}
	if leaked := wire.toPort(layers.UDPPort(snmp.DefaultSNMPTrapPort)); leaked != 0 {
		t.Errorf("%d SNMP notifications reached the wire, want none", leaked)
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
