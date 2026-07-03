package snmp

import (
	"net"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// TestSynthesizePeerTopologyLearnsHostOnPort verifies a downstream device's MAC
// lands in the bridge FDB on the port resolved through the walk's ifTable and
// dot1dBasePortTable — the chain a scanner follows for "Nearest Switch".
func TestSynthesizePeerTopologyLearnsHostOnPort(t *testing.T) {
	dev := createTestDevice()
	dev.Type = "switch"
	dev.TrunkPorts = []config.TrunkPort{
		{Interface: "FastEthernet0/5", RemoteDevice: "PC01"},
		{Interface: "FastEthernet0/6", RemoteDevice: "UNKNOWN-DEV"}, // MAC unresolved -> skipped
	}

	agent := NewAgent(dev, 0)

	// Seed the walk-provided tables: Fa0/5 is ifIndex 10005, bridge port 5.
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("SetOID: %v", err)
		}
	}
	must(agent.SetOID(ifDescr+".10005", &OIDValue{Type: gosnmp.OctetString, Value: "FastEthernet0/5"}))
	must(agent.SetOID(dot1dBasePortIfIndex+".5", &OIDValue{Type: gosnmp.Integer, Value: 10005}))

	pc1MAC, _ := net.ParseMAC("aa:bb:cc:00:00:21")
	resolve := func(name string) ([]byte, bool) {
		if name == "PC01" {
			return pc1MAC, true
		}
		return nil, false // UNKNOWN-DEV is unresolvable
	}

	agent.SynthesizePeerTopology(resolve)

	idx := macBytesToOIDIndex(pc1MAC)

	port := agent.mib.Get(dot1dTpFdbPort + "." + idx)
	if port == nil {
		t.Fatal("no FDB port entry for PC01 — host not learned")
	}
	if got, ok := port.Value.(int); !ok || got != 5 {
		t.Errorf("FDB port = %v, want bridge port 5 (Fa0/5)", port.Value)
	}

	status := agent.mib.Get(dot1dTpFdbStatus + "." + idx)
	if status == nil || status.Value.(int) != FDBStatusLearned {
		t.Errorf("FDB status = %v, want learned(%d)", status, FDBStatusLearned)
	}

	// The unresolvable neighbour must not produce a learned entry: only the self
	// entry plus PC01 should carry a learned/self status under the FDB.
	if got := learnedFDBCount(agent); got != 1 {
		t.Errorf("learned FDB entries = %d, want exactly 1 (PC01)", got)
	}
}

// learnedFDBCount counts dot1dTpFdbStatus entries marked learned(3).
func learnedFDBCount(a *Agent) int {
	n := 0

	a.mib.mu.RLock()
	defer a.mib.mu.RUnlock()

	for oid, v := range a.mib.entries {
		if strings.HasPrefix(oid, dot1dTpFdbStatus+".") {
			if iv, ok := v.Value.(int); ok && iv == FDBStatusLearned {
				n++
			}
		}
	}

	return n
}
