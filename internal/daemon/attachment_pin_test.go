package daemon

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

const pinPoolScenario = `networks:
  - name: med-data
    subnet: 10.51.210.0/24
    virtual_vlan: 210
attachments:
  - name: cyberscope
    at:
      device: MED-ACC-SW01
      ports:
        - GigabitEthernet1/0/20
        - GigabitEthernet1/0/21
        - GigabitEthernet1/0/22
    pins:
      - mac: "00:c0:17:00:00:02"
        device: MED-ACC-SW01
        interface: GigabitEthernet1/0/22
devices:
  - name: MED-ACC-SW01
    type: switch
    mac: 02:00:00:00:00:01
    interfaces:
      - name: Vlan210
        network: med-data
        address: 10.51.210.21/24
      - name: GigabitEthernet1/0/20
        vlans: [210]
      - name: GigabitEthernet1/0/21
        vlans: [210]
      - name: GigabitEthernet1/0/22
        vlans: [210]
      - name: GigabitEthernet1/0/23
        vlans: [210]
`

const pinnedClient = "00:c0:17:00:00:01"

func startPinTestSimulation(t *testing.T, scenario string) *Daemon {
	t.Helper()
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	daemon := replacementTestDaemon(t)
	req := replacementRequest("unused")
	req.Attachment = "cyberscope"
	req.ConfigData = scenario
	if err := daemon.StartSimulation(req); err != nil {
		t.Fatalf("StartSimulation() error = %v", err)
	}
	return daemon
}

func sessionPins(t *testing.T, daemon *Daemon) []config.AttachmentPin {
	t.Helper()
	cfg, err := config.LoadYAML(daemon.simulation.ConfigPath)
	if err != nil {
		t.Fatalf("LoadYAML(session scenario) error = %v", err)
	}
	return cfg.Attachments[0].Pins
}

func poolPin(mac, port string) api.AttachmentPin {
	return api.AttachmentPin{MAC: mac, Device: "MED-ACC-SW01", Interface: port}
}

func TestPinAttachmentClientWritesThePinAndRestartsTheSession(t *testing.T) {
	daemon := startPinTestSimulation(t, pinPoolScenario)
	before := daemon.simulation

	if err := daemon.PinAttachmentClient(defaultSessionID,
		poolPin(pinnedClient, "GigabitEthernet1/0/20")); err != nil {
		t.Fatalf("PinAttachmentClient() error = %v", err)
	}
	if daemon.simulation == before {
		t.Fatal("the session was not restarted on the new pin")
	}
	if daemon.simulation.ConfigPath != before.ConfigPath {
		t.Errorf("session moved to %q, want its own scenario %q",
			daemon.simulation.ConfigPath, before.ConfigPath)
	}
	want := []config.AttachmentPin{
		config.AttachmentPin(poolPin("00:c0:17:00:00:02", "GigabitEthernet1/0/22")),
		config.AttachmentPin(poolPin(pinnedClient, "GigabitEthernet1/0/20")),
	}
	if got := sessionPins(t, daemon); !reflect.DeepEqual(got, want) {
		t.Errorf("pins on disk = %#v, want %#v", got, want)
	}
	if got := daemon.simulation.cfg.Attachments[0].Pins; !reflect.DeepEqual(got, want) {
		t.Errorf("pins the restarted session runs = %#v, want %#v", got, want)
	}
}

func TestPinAttachmentClientMovesAnAlreadyPinnedClient(t *testing.T) {
	daemon := startPinTestSimulation(t, pinPoolScenario)

	// Spelled differently from the stored pin: the same client, not a second one.
	if err := daemon.PinAttachmentClient(defaultSessionID,
		poolPin("00-C0-17-00-00-02", "GigabitEthernet1/0/21")); err != nil {
		t.Fatalf("PinAttachmentClient() error = %v", err)
	}
	want := []config.AttachmentPin{
		config.AttachmentPin(poolPin("00-C0-17-00-00-02", "GigabitEthernet1/0/21")),
	}
	if got := sessionPins(t, daemon); !reflect.DeepEqual(got, want) {
		t.Errorf("pins = %#v, want %#v", got, want)
	}
}

func TestPinAttachmentClientRefusalLeavesScenarioAndSessionIntact(t *testing.T) {
	networkScenario := strings.Replace(
		replacementRequest("router").ConfigData, "name: tester", "name: cyberscope", 1)
	tests := []struct {
		name     string
		scenario string
		session  string
		pin      api.AttachmentPin
		wantErr  error
		wantCode fabric.DiagnosticCode
	}{
		{
			name: "port outside the pool", scenario: pinPoolScenario, session: defaultSessionID,
			pin: poolPin(pinnedClient, "GigabitEthernet1/0/23"), wantCode: fabric.CodeAttachmentPinOutsidePool,
		},
		{
			name: "port another client is pinned to", scenario: pinPoolScenario, session: defaultSessionID,
			pin: poolPin(pinnedClient, "GigabitEthernet1/0/22"), wantCode: fabric.CodeAttachmentPinDuplicate,
		},
		{
			name: "attachment names a network", scenario: networkScenario, session: defaultSessionID,
			pin: poolPin(pinnedClient, "GigabitEthernet1/0/20"), wantErr: api.ErrAttachmentPoolRequired,
		},
		{
			name: "unknown session", scenario: pinPoolScenario, session: "warehouse",
			pin: poolPin(pinnedClient, "GigabitEthernet1/0/20"), wantErr: api.ErrSimulationSessionNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			daemon := startPinTestSimulation(t, tt.scenario)
			active := daemon.simulation
			original, err := os.ReadFile(active.ConfigPath)
			if err != nil {
				t.Fatal(err)
			}

			err = daemon.PinAttachmentClient(tt.session, tt.pin)

			assertPinRefusal(t, err, tt.wantErr, tt.wantCode)
			if daemon.simulation != active {
				t.Error("a refused pin restarted the session")
			}
			if after, _ := os.ReadFile(active.ConfigPath); string(after) != string(original) {
				t.Errorf("a refused pin rewrote the scenario:\n%s", after)
			}
		})
	}
}

func TestPinAttachmentClientRestoresTheScenarioWhenTheRestartFails(t *testing.T) {
	daemon := startPinTestSimulation(t, pinPoolScenario)
	active := daemon.simulation
	original, err := os.ReadFile(active.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	injected := errors.New("injected startup failure")
	daemon.startSimulation = func(
		string, *config.Config, *fabric.Topology, bool, int, restoreRuntimeState,
	) (simulationResources, error) {
		return simulationResources{cancel: func() {}}, injected
	}

	err = daemon.PinAttachmentClient(defaultSessionID, poolPin(pinnedClient, "GigabitEthernet1/0/20"))

	if !errors.Is(err, injected) {
		t.Fatalf("PinAttachmentClient() error = %v, want %v", err, injected)
	}
	if daemon.simulation != active {
		t.Error("a failed restart replaced the running session")
	}
	if after, _ := os.ReadFile(active.ConfigPath); string(after) != string(original) {
		t.Errorf("a failed restart left the new pin on disk:\n%s", after)
	}
}

func assertPinRefusal(t *testing.T, err, wantErr error, wantCode fabric.DiagnosticCode) {
	t.Helper()
	if wantErr != nil {
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
		return
	}
	var unsafe *fabric.UnsafeTopologyError
	if !errors.As(err, &unsafe) {
		t.Fatalf("error = %v, want an unsafe-topology refusal", err)
	}
	for _, diagnostic := range unsafe.Diagnostics {
		if diagnostic.Code == wantCode {
			return
		}
	}
	t.Fatalf("diagnostics = %#v, want %s", unsafe.Diagnostics, wantCode)
}

// A re-pin rewrites the operator's scenario file, so it must not reformat it:
// the file after the move is the file before plus the pin.
func TestSetAttachmentPinChangesAGeneratedPackByThePinAlone(t *testing.T) {
	generated, err := scenario.Generate(scenario.Packs()[0].Request)
	if err != nil {
		t.Fatal(err)
	}
	attachment := generated.Config.Attachments[0]
	pin := config.AttachmentPin{
		MAC: pinnedClient, Device: attachment.At.Device, Interface: attachment.At.Ports[0],
	}

	amended, err := setAttachmentPin(generated.YAML, attachment.Name, pin)
	if err != nil {
		t.Fatal(err)
	}

	block := "      pins:\n" +
		"        - mac: \"" + pin.MAC + "\"\n" +
		"          device: " + pin.Device + "\n" +
		"          interface: " + pin.Interface + "\n"
	if !strings.Contains(string(amended), block) {
		t.Fatalf("amended scenario carries no pin block %q", block)
	}
	if got := strings.Replace(string(amended), block, "", 1); got != string(generated.YAML) {
		t.Error("the re-pin changed more of the scenario than the pin")
	}
}
