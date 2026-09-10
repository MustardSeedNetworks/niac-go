package snmp

import (
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestDeviceActionRequiresRealScalarInventory(t *testing.T) {
	state := devicestate.NewStore(devicestate.Identity{Hostname: "host"})
	agent := NewAgentWithState(&config.Device{Name: "host"}, state, AgentOptions{})
	if !agent.DeviceActionObservable(devicestate.ActionReboot) {
		t.Fatal("generated uptime not observable")
	}
	if agent.DeviceActionObservable(devicestate.ActionSTPTopologyChange) {
		t.Fatal("invented STP inventory")
	}
	executeProjectionAction(t, state, devicestate.ActionSTPTopologyChange)
	agent.Reindex()
	if agent.mib.Get(dot1dStpTopChanges) != nil || agent.mib.Get(dot1dStpTimeSinceTopologyChange) != nil {
		t.Fatal("action invented STP rows")
	}
	for _, value := range []*OIDValue{{Type: gosnmp.OctetString, Value: "invalid"}, {Type: gosnmp.TimeTicks, Value: -1}} {
		agent.mib.Set(deviceUptimeOID, value)
		agent.Reindex()
		if agent.DeviceActionObservable(devicestate.ActionReboot) {
			t.Fatal("invalid uptime marked observable")
		}
	}
}

func TestDeviceActionContractNamesOnlyActiveSTPRows(t *testing.T) {
	agent, state := actionProjectionAgent()
	state.SaveCheckpoint("baseline")
	if agent.WalkContract().Classify(dot1dStpTopChanges) != BucketKept {
		t.Fatal("baseline STP row was not kept")
	}
	executeProjectionAction(t, state, devicestate.ActionSTPTopologyChange)
	for _, oid := range []string{dot1dStpTopChanges, dot1dStpTimeSinceTopologyChange} {
		if agent.WalkContract().Classify(oid) != BucketLive || agent.WalkContract().DropsFromWalk(oid) {
			t.Fatal("active action row is not a retained live substitution")
		}
	}
	if err := state.RestoreCheckpoint("baseline"); err != nil {
		t.Fatal(err)
	}
	if agent.WalkContract().Classify(dot1dStpTopChanges) != BucketKept {
		t.Fatal("checkpoint did not restore fidelity contract")
	}
}

func TestDeviceActionKeepsDynamicSourcesAndCurrentOverrides(t *testing.T) {
	agent, state := actionProjectionAgent()
	count := uint32(100)
	agent.mib.SetDynamic(
		dot1dStpTopChanges,
		func() *OIDValue { return &OIDValue{Type: gosnmp.Counter32, Value: count} },
	)
	agent.Reindex()
	state.SaveCheckpoint("baseline")
	executeProjectionAction(t, state, devicestate.ActionSTPTopologyChange)
	assertActionValue(t, agent, dot1dStpTopChanges, "101")
	count = 200
	agent.Reindex()
	assertActionValue(t, agent, dot1dStpTopChanges, "201")
	agent.mib.Set(dot1dStpTopChanges, &OIDValue{Type: gosnmp.Counter32, Value: uint32(300)})
	agent.Reindex()
	assertActionValue(t, agent, dot1dStpTopChanges, "301")
	if err := state.RestoreCheckpoint("baseline"); err != nil {
		t.Fatal(err)
	}
	assertActionValue(t, agent, dot1dStpTopChanges, "300")
}

func TestDeviceActionSharesSTPCommunitiesButNotPeerDevices(t *testing.T) {
	agent, state := actionProjectionAgent()
	peer, _ := actionProjectionAgent()
	community := NewAgentWithState(agent.device, state, AgentOptions{Community: "other"})
	community.mib.Set(dot1dStpTopChanges, &OIDValue{Type: gosnmp.Counter32, Value: uint32(41)})
	community.mib.Set(dot1dStpTimeSinceTopologyChange, &OIDValue{Type: gosnmp.TimeTicks, Value: uint32(7654)})
	community.Reindex()
	executeProjectionAction(t, state, devicestate.ActionSTPTopologyChange)
	for _, current := range []*Agent{agent, community} {
		assertActionValue(t, current, dot1dStpTopChanges, "42")
	}
	assertActionValue(t, peer, dot1dStpTopChanges, "41")
	assertActionValue(t, peer, dot1dStpTimeSinceTopologyChange, "7654")
}

func TestDeviceActionDoesNotChangeUSMEngineLifetime(t *testing.T) {
	agent, state := actionProjectionAgent()
	engine := engineFor(t, []config.SNMPv3User{{Username: "observer"}})
	engine.bootTime = time.Now().Add(-time.Hour)
	beforeBoot, beforeID, beforeTime := engine.boots, engine.engineID, engine.bootTime
	before := engine.engineTime()
	beforeResponse := actionV3Response(t, engine, agent)
	executeProjectionAction(t, state, devicestate.ActionReboot)
	if got := agent.UptimeTicks(); got > 100 {
		t.Fatalf("served reboot uptime=%d", got)
	}
	if engine.boots != beforeBoot || engine.engineID != beforeID || !engine.bootTime.Equal(beforeTime) ||
		engine.engineTime() < before {
		t.Fatal("management reboot rewound USM engine lifetime")
	}
	afterResponse := actionV3Response(t, engine, agent)
	if usmOf(afterResponse).AuthoritativeEngineBoots != usmOf(beforeResponse).AuthoritativeEngineBoots ||
		usmOf(afterResponse).AuthoritativeEngineTime < usmOf(beforeResponse).AuthoritativeEngineTime {
		t.Fatal("served USM security lifetime changed")
	}
	if len(afterResponse.Variables) != 1 || gosnmp.ToBigInt(afterResponse.Variables[0].Value).Uint64() > 100 {
		t.Fatalf("v3 uptime did not reset: %+v", afterResponse.Variables)
	}
}

func actionV3Response(t *testing.T, engine *V3Engine, agent *Agent) *gosnmp.SnmpPacket {
	t.Helper()
	request := &gosnmp.SnmpPacket{
		Version:            gosnmp.Version3,
		MsgID:              1,
		RequestID:          1,
		PDUType:            gosnmp.GetRequest,
		SecurityModel:      gosnmp.UserSecurityModel,
		SecurityParameters: &gosnmp.UsmSecurityParameters{UserName: "observer"},
		Variables:          []gosnmp.SnmpPDU{{Name: deviceUptimeOID, Type: gosnmp.Null}},
	}
	wire, err := engine.respondToRequest(request, agent.ProcessPDU)
	if err != nil {
		t.Fatal(err)
	}
	response, err := parseV3Header(wire)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
