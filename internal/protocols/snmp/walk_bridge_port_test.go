package snmp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// bridgePortWalk carries a two-port BRIDGE-MIB whose dot1dTpPortTable holds
// values the agent's own defaults disagree with, so an overwrite is visible.
const bridgePortWalk = `.1.3.6.1.2.1.2.2.1.2.1 = STRING: "GigabitEthernet0/1"
.1.3.6.1.2.1.2.2.1.2.2 = STRING: "GigabitEthernet0/2"
.1.3.6.1.2.1.17.1.1.0 = Hex-STRING: 00 0B FD D4 6F DE
.1.3.6.1.2.1.17.1.2.0 = INTEGER: 2
.1.3.6.1.2.1.17.1.4.1.2.1 = INTEGER: 1
.1.3.6.1.2.1.17.1.4.1.2.2 = INTEGER: 2
.1.3.6.1.2.1.17.4.4.1.1.1 = INTEGER: 1
.1.3.6.1.2.1.17.4.4.1.2.1 = INTEGER: 9999
.1.3.6.1.2.1.17.4.4.1.3.1 = Counter32: 123456
.1.3.6.1.2.1.17.4.4.1.4.1 = Counter32: 654321
.1.3.6.1.2.1.17.4.4.1.5.1 = Counter32: 77
`

// loadBridgePortWalk loads bridgePortWalk into an agent for a device the
// caller has shaped, so each test decides whether port 1 is one the scenario
// authored or one only the capture knows about.
func loadBridgePortWalk(t *testing.T, shape func(*config.Device)) *Agent {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bridge.walk")
	if err := os.WriteFile(path, []byte(bridgePortWalk), 0o600); err != nil {
		t.Fatal(err)
	}
	device := createTestDevice()
	shape(device)
	agent := NewAgent(device, 0)
	if err := agent.LoadWalkFile(path); err != nil {
		t.Fatal(err)
	}
	return agent
}

// TestBridgePortTableSurvivesAnUndeclaredPort is finding 1 of the F0 audit,
// narrowed by the owner on 2026-09-08: refreshBridgePortCounters overwrote
// dot1dTpPortTable for every bridge port the walk defines, whether or not the
// scenario authored the interface behind it and whether or not the device
// declares trunk_ports. A port the scenario says nothing about must arrive
// byte-identical.
func TestBridgePortTableSurvivesAnUndeclaredPort(t *testing.T) {
	agent := loadBridgePortWalk(t, func(*config.Device) {})

	assertBridgePortColumn(t, agent, dot1dTpPortMaxInfo+".1", "9999")
	assertBridgePortColumn(t, agent, dot1dTpPortInFrames+".1", "123456")
	assertBridgePortColumn(t, agent, dot1dTpPortOutFrames+".1", "654321")
	assertBridgePortColumn(t, agent, dot1dTpPortInDiscards+".1", "77")
}

// TestBridgePortTableIsSubstitutedOnAnAuthoredPort: the narrowing keeps the
// substitution where substitution 5 already reaches — a bridge port whose
// ifIndex the scenario authored reports this run, not the capture's.
func TestBridgePortTableIsSubstitutedOnAnAuthoredPort(t *testing.T) {
	agent := loadBridgePortWalk(t, func(device *config.Device) {
		device.Interfaces = []config.Interface{{Name: "GigabitEthernet0/1"}}
	})

	assertBridgePortColumn(t, agent, dot1dTpPortMaxInfo+".1", "1500")
	assertBridgePortColumn(t, agent, dot1dTpPortInDiscards+".1", "0")
	if got := oidValueString(agent.mib.Get(dot1dTpPortInFrames + ".1")); got == "123456" {
		t.Error("dot1dTpPortInFrames still serves the capture's frozen counter on an authored port")
	}
}

// TestBridgePortTableIsSubstitutedUnderTrunkPorts: trunk_ports hands the
// device's forwarding topology to the scenario, so the bridge port table
// follows the neighbour tables substitution 4 already replaces.
func TestBridgePortTableIsSubstitutedUnderTrunkPorts(t *testing.T) {
	agent := loadBridgePortWalk(t, func(device *config.Device) {
		device.TrunkPorts = []config.TrunkPort{{Interface: "GigabitEthernet0/1"}}
	})

	assertBridgePortColumn(t, agent, dot1dTpPortMaxInfo+".1", "1500")
}

// TestBridgePortTableIsAddedWhereTheWalkHasNone: the narrowing must not stop
// the agent giving a walk-supplied bridge the dot1dTpPortTable it lacks —
// that is the agent_added bucket, not an overwrite. Port 2 has a
// dot1dBasePortIfIndex row and no dot1dTpPort row of its own.
func TestBridgePortTableIsAddedWhereTheWalkHasNone(t *testing.T) {
	agent := loadBridgePortWalk(t, func(*config.Device) {})

	assertBridgePortColumn(t, agent, dot1dTpPortMaxInfo+".2", "1500")
	if agent.mib.Get(dot1dTpPortInFrames+".2") == nil {
		t.Error("dot1dTpPortInFrames.2 absent: a bridge port the walk left bare gained no counters")
	}
}

// assertBridgePortColumn reads the served value whatever SNMP type carries it:
// the columns under test cross Integer and Counter32, and a substituted
// counter is dynamic.
func assertBridgePortColumn(t *testing.T, agent *Agent, oid, want string) {
	t.Helper()
	entry := agent.mib.Get(oid)
	if entry == nil {
		t.Fatalf("%s absent", oid)
	}
	if got := oidValueString(entry); got != want {
		t.Errorf("%s = %s, want %s", oid, got, want)
	}
}
