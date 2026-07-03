package snmp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// topoWalk mixes device-content OIDs with the topology tables a capture carries
// from its *original* neighbours (LLDP remote, CDP cache, bridge FDB).
const topoWalk = `.1.3.6.1.2.1.1.5.0 = STRING: "captured-device-name"
.1.3.6.1.2.1.2.2.1.2.1 = STRING: "GigabitEthernet1/0/1"
.1.0.8802.1.1.2.1.3.7.1.3.1 = STRING: "Gi1/0/1"
.1.0.8802.1.1.2.1.4.1.1.9.1.1 = STRING: "foreign-lldp-neighbor"
.1.3.6.1.4.1.9.9.23.1.2.1.1.6.1.1 = STRING: "foreign-cdp-device"
.1.3.6.1.2.1.17.4.3.1.2.1.2.3.4.5.6 = INTEGER: 5
`

func TestIsSynthesizedTopologyOID(t *testing.T) {
	strip := []string{
		"1.0.8802.1.1.2.1.4",                  // lldpRemoteSystemsData root
		".1.0.8802.1.1.2.1.4.1.1.9.1.1",       // lldpRemTable entry (leading dot)
		".1.3.6.1.4.1.9.9.23.1.2.1.1.6.1.1",   // cdpCacheTable entry
		".1.3.6.1.2.1.17.4.3.1.2.1.2.3.4.5.6", // dot1dTpFdbTable entry
	}
	keep := []string{
		".1.3.6.1.2.1.2.2.1.2.1",      // ifDescr (device content)
		".1.0.8802.1.1.2.1.3.7.1.3.1", // LLDP *local* port — not a neighbour
		".1.3.6.1.2.1.1.5.0",          // sysName
	}
	for _, oid := range strip {
		if !isSynthesizedTopologyOID(oid) {
			t.Errorf("%s should be a synthesized-topology OID", oid)
		}
	}
	for _, oid := range keep {
		if isSynthesizedTopologyOID(oid) {
			t.Errorf("%s should NOT be stripped", oid)
		}
	}
}

// TestLoadWalkFileStripsTopologyWhenTrunkPortsDeclared: a device that authors
// its links (trunk_ports) drops the walk's foreign neighbour/FDB tables so the
// synthesized topology wins; device content survives either way.
func TestLoadWalkFileStripsTopologyWhenTrunkPortsDeclared(t *testing.T) {
	load := func(withTrunks bool) *Agent {
		t.Helper()
		p := filepath.Join(t.TempDir(), "t.walk")
		if err := os.WriteFile(p, []byte(topoWalk), 0o600); err != nil {
			t.Fatal(err)
		}
		dev := createTestDevice()
		if withTrunks {
			dev.TrunkPorts = []config.TrunkPort{
				{Interface: "Gi1/0/1", RemoteDevice: "core-01", RemoteInterface: "Gi1/0/24"},
			}
		}
		a := NewAgent(dev, 0)
		if err := a.LoadWalkFile(p); err != nil {
			t.Fatalf("LoadWalkFile: %v", err)
		}
		return a
	}

	const (
		lldpRem = ".1.0.8802.1.1.2.1.4.1.1.9.1.1"
		cdpCch  = ".1.3.6.1.4.1.9.9.23.1.2.1.1.6.1.1"
		ifDescr = ".1.3.6.1.2.1.2.2.1.2.1"
	)

	// Without trunk_ports: walk topology data is kept (backward compatible).
	noTrunks := load(false)
	if v, _ := noTrunks.HandleGet(lldpRem); v == nil {
		t.Error("without trunk_ports, the walk LLDP neighbour should be loaded")
	}

	// With trunk_ports: foreign topology tables are skipped.
	trunks := load(true)
	if v, _ := trunks.HandleGet(lldpRem); v != nil {
		t.Errorf("with trunk_ports, walk LLDP neighbour must be skipped, got %v", v.Value)
	}
	if v, _ := trunks.HandleGet(cdpCch); v != nil {
		t.Errorf("with trunk_ports, walk CDP cache must be skipped, got %v", v.Value)
	}
	// Device content survives the strip.
	if v, _ := trunks.HandleGet(ifDescr); v == nil {
		t.Error("ifDescr (device content) must survive the topology strip")
	}
}

// TestLoadWalkFileSkipsWalkSysName: the device keeps its configured identity;
// the capture's sysName never wins (else same-walk devices collide as one node).
func TestLoadWalkFileSkipsWalkSysName(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t.walk")
	if err := os.WriteFile(p, []byte(topoWalk), 0o600); err != nil {
		t.Fatal(err)
	}
	dev := createTestDevice() // has a configured Name
	agent := NewAgent(dev, 0)
	if err := agent.LoadWalkFile(p); err != nil {
		t.Fatalf("LoadWalkFile: %v", err)
	}

	got, _ := agent.HandleGet(".1.3.6.1.2.1.1.5.0")
	if got == nil {
		t.Fatal("sysName missing")
	}
	if s, _ := got.Value.(string); s == "captured-device-name" {
		t.Error("walk sysName leaked; device identity must win")
	}
	if s, _ := got.Value.(string); s != dev.Name {
		t.Errorf("sysName = %q, want configured name %q", s, dev.Name)
	}
}
