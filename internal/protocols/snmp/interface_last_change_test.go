package snmp

import (
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestUnchangedInterfaceLastChangeIsZero(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)
	agent.startTime = time.Now().Add(-time.Hour)
	for range 2 {
		value := agent.mib.Get(ifLastChange + ".1")
		if value == nil || value.Type != gosnmp.TimeTicks || value.Value != uint32(0) {
			t.Fatalf("unchanged interface ifLastChange = %#v, want TimeTicks 0", value)
		}
	}
}

func TestInterfaceLastChangeTracksTransitionNotPolling(t *testing.T) {
	device := &config.Device{Name: "switch", Interfaces: []config.Interface{{Name: "eth0"}}}
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	state.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{
		{Name: "eth0", AdminUp: true, OperUp: true},
	}})
	telemetry := NewProtocolTelemetry()
	telemetry.startedAt = time.Now().Add(-time.Hour)
	agent := NewAgentWithState(device, state, AgentOptions{Telemetry: telemetry})
	peer := NewAgentWithState(device, state, AgentOptions{Telemetry: telemetry, Community: "other"})
	for _, current := range []*Agent{agent, peer} {
		current.mib.Set(ifLastChange+".1", &OIDValue{Type: gosnmp.TimeTicks, Value: uint32(321)})
		current.mib.Set(ifLastChange+".999", &OIDValue{Type: gosnmp.TimeTicks, Value: uint32(654)})
		current.Reindex()
		assertLastChange(t, current, "1", 321)
	}
	if err := state.SetInterfaceFault("eth0", devicestate.FaultLinkDown, 1); err != nil {
		t.Fatal(err)
	}
	want := uptimeTicks(agent.uptimeBase + state.InterfaceLastChange("eth0").Sub(agent.startTime))
	for _, current := range []*Agent{agent, peer} {
		for range 2 {
			current.Reindex()
			assertLastChange(t, current, "1", want)
			assertLastChange(t, current, "999", 654)
		}
		if current.WalkContract().Classify(ifLastChange+".1") != BucketLive {
			t.Fatal("transition timestamp is not a named live substitution")
		}
		if current.WalkContract().Classify(ifLastChange+".999") != BucketKept {
			t.Fatal("foreign interface timestamp is not preserved")
		}
	}
	if err := state.RestoreState(state.ExportState()); err != nil {
		t.Fatal(err)
	}
	assertLastChange(t, agent, "1", 321)
}

func assertLastChange(t *testing.T, agent *Agent, index string, want uint32) {
	t.Helper()
	value, err := agent.HandleGet(ifLastChange + "." + index)
	if err != nil || value == nil || value.Type != gosnmp.TimeTicks || value.Value != want {
		t.Fatalf("ifLastChange.%s = %#v, %v; want TimeTicks %d", index, value, err, want)
	}
}

func TestInterfaceLastChangeRemappingRestoresForeignRow(t *testing.T) {
	device := &config.Device{Name: "switch", Interfaces: []config.Interface{{Name: "eth0"}}}
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	state.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{
		{Name: "eth0", AdminUp: true, OperUp: true},
	}})
	agent := NewAgentWithState(device, state, AgentOptions{})
	agent.mib.Set(ifLastChange+".1", &OIDValue{Type: gosnmp.TimeTicks, Value: uint32(321)})
	agent.Reindex()
	for _, column := range []string{ifDescr, ifName} {
		if err := agent.AddMib(column+".1", "STRING", "foreign"); err != nil {
			t.Fatal(err)
		}
		if err := agent.AddMib(column+".2", "STRING", "eth0"); err != nil {
			t.Fatal(err)
		}
	}
	agent.mib.Set(ifLastChange+".2", &OIDValue{Type: gosnmp.TimeTicks, Value: uint32(654)})
	agent.Reindex()
	if err := state.SetInterfaceFault("eth0", devicestate.FaultLinkDown, 1); err != nil {
		t.Fatal(err)
	}
	assertLastChange(t, agent, "1", 321)
	want := uptimeTicks(agent.uptimeBase + state.InterfaceLastChange("eth0").Sub(agent.startTime))
	assertLastChange(t, agent, "2", want)
}

func TestInterfaceLastChangePreservesDynamicBaseline(t *testing.T) {
	device := &config.Device{Name: "switch", Interfaces: []config.Interface{{Name: "eth0"}}}
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	state.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{
		{Name: "eth0", AdminUp: true, OperUp: true},
	}})
	agent := NewAgentWithState(device, state, AgentOptions{})
	baseline := uint32(111)
	agent.mib.SetDynamic(ifLastChange+".1", func() *OIDValue {
		return &OIDValue{Type: gosnmp.TimeTicks, Value: baseline}
	})
	agent.Reindex()
	assertLastChange(t, agent, "1", 111)
	baseline = 222
	assertLastChange(t, agent, "1", 222)
	if err := state.SetInterfaceFault("eth0", devicestate.FaultLinkDown, 1); err != nil {
		t.Fatal(err)
	}
	// Pin the epoch relative to the recorded transition, without delaying a poll.
	agent.startTime = state.InterfaceLastChange("eth0").Add(-13 * time.Second)
	agent.uptimeBase = time.Second
	assertLastChange(t, agent, "1", 1400)
	if err := state.RestoreState(state.ExportState()); err != nil {
		t.Fatal(err)
	}
	assertLastChange(t, agent, "1", 222)
}

func TestCommunityAgentsShareManagementEpoch(t *testing.T) {
	device := createTestDevice()
	telemetry := NewProtocolTelemetry()
	first := NewAgentWithCommunityAndTelemetry(device, "public", 0, telemetry)
	second := NewAgentWithCommunityAndTelemetry(device, "resources", 0, telemetry)
	if !first.startTime.Equal(second.startTime) || first.uptimeBase != second.uptimeBase {
		t.Fatal("community agents use different management uptime origins")
	}
}
