package snmp

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
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

func TestAuthoredDiscoveryUsesWalkInterfaceIdentityEndToEnd(t *testing.T) {
	dev := createTestDevice()
	dev.Type = "switch"
	dev.SNMPConfig.WalkFile = "switch.walk"
	dev.LLDPConfig = &config.LLDPConfig{Enabled: true}
	dev.CDPConfig = &config.CDPConfig{Enabled: true}
	dev.TrunkPorts = []config.TrunkPort{{
		Interface: "FastEthernet0/5", RemoteDevice: "PC01", RemoteInterface: "eth0",
	}}
	dev.Interfaces = []config.Interface{{
		Name: "FastEthernet0/5", Speed: 100, Duplex: "half", AdminStatus: "down",
		OperStatus: "down", Description: "intentional demo fault",
	}}
	agent := NewAgent(dev, 0)
	walkPath := filepath.Join(t.TempDir(), "switch.walk")
	walk := `.1.3.6.1.2.1.2.2.1.2.10005 = STRING: "FastEthernet0/5"
.1.3.6.1.2.1.17.1.2.0 = INTEGER: 5
.1.3.6.1.2.1.17.1.4.1.2.5 = INTEGER: 10005
`
	if err := os.WriteFile(walkPath, []byte(walk), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := agent.LoadWalkFile(walkPath); err != nil {
		t.Fatalf("LoadWalkFile: %v", err)
	}
	pcMAC, _ := net.ParseMAC("aa:bb:cc:00:00:21")
	agent.SynthesizePeerTopology(func(string) ([]byte, bool) { return pcMAC, true })

	assertMIBValue(t, agent, lldpLocPortTable+".1.3.10005", "FastEthernet0/5")
	assertMIBValue(t, agent, lldpRemTable+".1.9.0.10005.1", "PC01")
	assertMIBValue(t, agent, cdpCacheTable+".1.6.10005.1", "PC01")
	assertMIBValue(t, agent, dot1dBasePortIfIndex+".5", 10005)
	assertMIBValue(t, agent, dot1dTpFdbPort+"."+macBytesToOIDIndex(pcMAC), 5)
	assertMIBValue(t, agent, ifSpeed+".10005", 100000000)
	assertMIBValue(t, agent, ifHighSpeed+".10005", 100)
	assertMIBValue(t, agent, ifAdminStatus+".10005", 2)
	assertMIBValue(t, agent, ifOperStatus+".10005", 2)
	assertMIBValue(t, agent, ifAlias+".10005", "intentional demo fault")
	assertMIBValue(t, agent, dot3StatsDuplexStatus+".10005", 2)

	if stale := agent.mib.Get(cdpCacheTable + ".1.6.1.1"); stale != nil {
		t.Fatalf("stale ordinal CDP row remains: %v", stale.Value)
	}
}

func TestFDBOnlyAttachmentLearnsMACWithoutInventingNeighbor(t *testing.T) {
	dev := createTestDevice()
	dev.Type = "switch"
	dev.LLDPConfig = &config.LLDPConfig{Enabled: true}
	dev.CDPConfig = &config.CDPConfig{Enabled: true}
	dev.Interfaces = []config.Interface{{Name: "FastEthernet0/5"}}
	dev.TrunkPorts = []config.TrunkPort{{
		Interface: "FastEthernet0/5", RemoteDevice: "PC01", RemoteInterface: "eth0", FDBOnly: true,
	}}
	agent := NewAgent(dev, 0)
	pcMAC, _ := net.ParseMAC("aa:bb:cc:00:00:21")
	agent.SynthesizePeerTopology(func(string) ([]byte, bool) { return pcMAC, true })

	assertMIBValue(t, agent, lldpLocPortTable+".1.3.1", "FastEthernet0/5")
	assertMIBValue(t, agent, dot1dTpFdbPort+"."+macBytesToOIDIndex(pcMAC), 1)
	for _, prefix := range []string{lldpRemTable, cdpCacheTable} {
		for _, oid := range agent.mib.AllOIDs() {
			if strings.HasPrefix(oid, prefix+".") {
				t.Fatalf("FDB-only host produced discovery neighbor %s", oid)
			}
		}
	}
}

func assertMIBValue(t *testing.T, agent *Agent, oid string, want any) {
	t.Helper()
	entry := agent.mib.Get(oid)
	if entry == nil {
		t.Fatalf("%s is absent", oid)
	}
	if got := entry.Value; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("%s = %v, want %v", oid, got, want)
	}
}

// TestSynthesizePeerTopologyDerivesPortWhenWalkSparse covers the case a real
// capture models few bridge ports: the port is derived from the interface name
// (Fa0/7 -> 7), a dot1dBasePort row is added, and dot1dBaseNumPorts is raised so
// the learned FDB port stays in range (managers reject out-of-range ports).
func TestSynthesizePeerTopologyDerivesPortWhenWalkSparse(t *testing.T) {
	dev := createTestDevice()
	dev.Type = "switch"
	dev.SNMPConfig.WalkFile = "x.walk" // walk-backed: no synthesized ifTable
	dev.TrunkPorts = []config.TrunkPort{{Interface: "FastEthernet0/7", RemoteDevice: "PC07"}}

	agent := NewAgent(dev, 0)

	// Walk provides ifDescr, a small NumPorts, and a sample bridge-port row
	// (port 2 -> ifIndex 10002, i.e. offset 10000), but no row for Fa0/7.
	fa7 := &OIDValue{Type: gosnmp.OctetString, Value: "FastEthernet0/7"}
	if err := agent.SetOID(ifDescr+".10007", fa7); err != nil {
		t.Fatalf("SetOID: %v", err)
	}
	if err := agent.SetOID(dot1dBaseNumPorts, &OIDValue{Type: gosnmp.Integer, Value: 5}); err != nil {
		t.Fatalf("SetOID: %v", err)
	}
	if err := agent.SetOID(dot1dBasePortIfIndex+".2", &OIDValue{Type: gosnmp.Integer, Value: 10002}); err != nil {
		t.Fatalf("SetOID: %v", err)
	}

	mac, _ := net.ParseMAC("aa:bb:cc:00:00:77")
	agent.SynthesizePeerTopology(func(string) ([]byte, bool) { return mac, true })

	port := agent.mib.Get(dot1dTpFdbPort + "." + macBytesToOIDIndex(mac))
	if port == nil || port.Value.(int) != 7 {
		t.Fatalf("FDB port = %v, want physical port 7", port)
	}
	// dot1dBasePort row added and NumPorts raised to cover it.
	if e := agent.mib.Get(dot1dBasePortIfIndex + ".7"); e == nil || e.Value.(int) != 10007 {
		t.Errorf("dot1dBasePortIfIndex.7 = %v, want 10007", e)
	}
	if e := agent.mib.Get(dot1dBaseNumPorts); e == nil || e.Value.(int) < 7 {
		t.Errorf("dot1dBaseNumPorts = %v, want >= 7", e)
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
