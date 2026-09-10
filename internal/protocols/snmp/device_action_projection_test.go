package snmp

import (
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func actionProjectionAgent() (*Agent, *devicestate.Store) {
	device := &config.Device{Name: "switch", Interfaces: []config.Interface{{Name: "eth0"}}}
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	state.ReplaceNetwork(
		devicestate.Network{Interfaces: []devicestate.Interface{{Name: "eth0", AdminUp: true, OperUp: true}}},
	)
	agent := NewAgentWithState(device, state, AgentOptions{})
	agent.mib.Set("1.3.6.1.2.1.1.3.0", &OIDValue{Type: gosnmp.TimeTicks, Value: uint32(123456)})
	agent.mib.Set(dot1dStpTopChanges, &OIDValue{Type: gosnmp.Counter32, Value: uint32(41)})
	agent.mib.Set(dot1dStpTimeSinceTopologyChange, &OIDValue{Type: gosnmp.TimeTicks, Value: uint32(7654)})
	agent.mib.Set(ifLastChange+".1", &OIDValue{Type: gosnmp.TimeTicks, Value: uint32(321)})
	agent.Reindex()
	return agent, state
}

func executeProjectionAction(t *testing.T, state *devicestate.Store, kind devicestate.DeviceActionType) {
	t.Helper()
	if _, err := state.ExecuteDeviceAction(kind, string(kind)); err != nil {
		t.Fatal(err)
	}
}

func assertActionValue(t *testing.T, agent *Agent, oid, want string) {
	t.Helper()
	kind := gosnmp.TimeTicks
	if oid == dot1dStpTopChanges {
		kind = gosnmp.Counter32
	}
	value, err := agent.HandleGet(oid)
	if err != nil || value == nil || value.Type != kind || oidValueString(value) != want {
		t.Fatalf("%s = %#v (%v), want %s", oid, value, err, want)
	}
}

func TestDeviceActionProjectionPreservesAndRestoresCapturedRows(t *testing.T) {
	agent, state := actionProjectionAgent()
	state.SaveCheckpoint("baseline")
	assertActionValue(t, agent, dot1dStpTopChanges, "41")
	assertActionValue(t, agent, dot1dStpTimeSinceTopologyChange, "7654")
	executeProjectionAction(t, state, devicestate.ActionSTPTopologyChange)
	assertActionValue(t, agent, dot1dStpTopChanges, "42")
	if value := agent.mib.Get(dot1dStpTimeSinceTopologyChange); value.Value.(uint32) > 100 {
		t.Fatalf("STP timer did not reset: %v", value)
	}
	agent.Reindex()
	agent.Reindex()
	assertActionValue(t, agent, dot1dStpTopChanges, "42")
	if err := state.RestoreCheckpoint("baseline"); err != nil {
		t.Fatal(err)
	}
	assertActionValue(t, agent, dot1dStpTopChanges, "41")
	assertActionValue(t, agent, dot1dStpTimeSinceTopologyChange, "7654")
}

func TestDeviceRebootResetsUptimeAndPrebootInterfaceTimestamp(t *testing.T) {
	agent, state := actionProjectionAgent()
	start := agent.startTime
	state.SaveCheckpoint("baseline")
	agent.mib.Set(ifLastChange+".999", &OIDValue{Type: gosnmp.TimeTicks, Value: uint32(654)})
	agent.Reindex()
	executeProjectionAction(t, state, devicestate.ActionReboot)
	if got := agent.mib.Get("1.3.6.1.2.1.1.3.0"); got.Value.(uint32) > 100 {
		t.Fatalf("reboot uptime = %v", got)
	}
	assertLastChange(t, agent, "1", 0)
	assertLastChange(t, agent, "999", 0)
	if !agent.startTime.Equal(start) {
		t.Fatal("reboot mutated agent lifetime")
	}
	if err := state.RestoreCheckpoint("baseline"); err != nil {
		t.Fatal(err)
	}
	assertActionValue(t, agent, "1.3.6.1.2.1.1.3.0", "123456")
	assertLastChange(t, agent, "1", 321)
	assertLastChange(t, agent, "999", 654)
}

func TestDeviceRebootSharesEpochAcrossCommunities(t *testing.T) {
	agent, state := actionProjectionAgent()
	peer := NewAgentWithState(agent.device, state, AgentOptions{Community: "other"})
	executeProjectionAction(t, state, devicestate.ActionReboot)
	saved := state.ExportState()
	saved.Telemetry.RebootedAt = time.Now().Add(-5 * time.Second)
	if err := state.RestoreState(saved); err != nil {
		t.Fatal(err)
	}
	if err := state.SetInterfaceFault("eth0", devicestate.FaultLinkDown, 1); err != nil {
		t.Fatal(err)
	}
	expected := uptimeTicks(state.InterfaceLastChange("eth0").Sub(saved.Telemetry.RebootedAt))
	for _, current := range []*Agent{agent, peer} {
		current.Reindex()
		ticks := current.sysUpTimeTicks()
		if ticks < 500 || ticks > 600 {
			t.Fatalf("community uptime = %d", ticks)
		}
		assertLastChange(t, current, "1", expected)
	}
}
