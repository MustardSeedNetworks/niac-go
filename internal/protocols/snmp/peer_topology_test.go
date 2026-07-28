package snmp

import (
	"bytes"
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
	must(
		agent.SetOID(
			ifDescr+".10005",
			&OIDValue{Type: gosnmp.OctetString, Value: "FastEthernet0/5"},
		),
	)
	must(agent.SetOID(dot1dBasePortIfIndex+".5", &OIDValue{Type: gosnmp.Integer, Value: 10005}))

	pc1MAC, _ := net.ParseMAC("aa:bb:cc:00:00:21")
	resolve := func(name, _ string) (PeerIdentity, bool) {
		if name == "PC01" {
			return PeerIdentity{MAC: pc1MAC}, true
		}
		return PeerIdentity{}, false // UNKNOWN-DEV is unresolvable
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
		VLANs: []int{200}, NativeVLAN: 200,
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
	agent.SynthesizePeerTopology(func(string, string) (PeerIdentity, bool) {
		return PeerIdentity{MAC: pcMAC, CDPEnabled: true}, true
	})

	assertMIBValue(t, agent, lldpLocPortTable+".1.3.10005", "FastEthernet0/5")
	assertMIBValue(t, agent, lldpRemTable+".1.9.0.10005.1", "PC01")
	assertMIBValue(t, agent, cdpCacheTable+".1.6.10005.1", "PC01")
	assertMIBValue(t, agent, dot1dBasePortIfIndex+".5", 10005)
	assertMIBValue(t, agent, dot1dTpFdbPort+"."+macBytesToOIDIndex(pcMAC), 5)
	assertMIBValue(t, agent, dot1qPVID+".5", 200)
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
		Interface: "FastEthernet0/5", RemoteDevice: "PC01", RemoteInterface: "eth0",
		VLANs: []int{210}, NativeVLAN: 210, FDBOnly: true,
	}}
	agent := NewAgent(dev, 0)
	pcMAC, _ := net.ParseMAC("aa:bb:cc:00:00:21")
	agent.SynthesizePeerTopology(func(string, string) (PeerIdentity, bool) {
		return PeerIdentity{MAC: pcMAC}, true
	})

	assertMIBValue(t, agent, lldpLocPortTable+".1.3.1", "FastEthernet0/5")
	assertMIBValue(t, agent, dot1dTpFdbPort+"."+macBytesToOIDIndex(pcMAC), 1)
	qBridgeIndex := "210." + macBytesToOIDIndex(pcMAC)
	assertMIBValue(t, agent, dot1qTpFDBAddress+"."+qBridgeIndex, []byte(pcMAC))
	assertMIBValue(t, agent, dot1qTpFDBPort+"."+qBridgeIndex, 1)
	assertMIBValue(t, agent, dot1qTpFDBStatus+"."+qBridgeIndex, FDBStatusLearned)
	assertMIBValue(t, agent, dot1qFDBDynamicCount+".210", uint32(1))
	if got := agent.mib.Get(dot1qFDBDynamicCount + ".210").Type; got != gosnmp.Counter32 {
		t.Fatalf("dot1qFdbDynamicCount type = %v, want Counter32", got)
	}
	assertMIBValue(t, agent, dot1qVlanFDBID+".0.210", uint32(210))
	assertMIBValue(t, agent, dot1qPVID+".1", 210)
	for _, prefix := range []string{lldpRemTable, cdpCacheTable} {
		for _, oid := range agent.mib.AllOIDs() {
			if strings.HasPrefix(oid, prefix+".") {
				t.Fatalf("FDB-only host produced discovery neighbor %s", oid)
			}
		}
	}
}

func TestQBridgeUsesDefaultNativeVLAN(t *testing.T) {
	dev := createTestDevice()
	dev.Type = "switch"
	dev.Interfaces = []config.Interface{{Name: "FastEthernet0/5"}}
	dev.TrunkPorts = []config.TrunkPort{{
		Interface: "FastEthernet0/5", RemoteDevice: "PC01",
		VLANs: []int{200, 220}, FDBOnly: true,
	}}
	agent := NewAgent(dev, 0)
	pcMAC, _ := net.ParseMAC("aa:bb:cc:00:00:21")
	agent.SynthesizePeerTopology(func(string, string) (PeerIdentity, bool) {
		return PeerIdentity{MAC: pcMAC}, true
	})

	qBridgeIndex := "1." + macBytesToOIDIndex(pcMAC)
	assertMIBValue(t, agent, dot1qTpFDBPort+"."+qBridgeIndex, 1)
	assertMIBValue(t, agent, dot1qVlanFDBID+".0.1", uint32(1))
	assertMIBValue(t, agent, dot1qPVID+".1", 1)
	if got := agent.mib.Get(dot1qMaxVlanID).Type; got != gosnmp.Integer {
		t.Fatalf("dot1qMaxVlanId type = %v, want Integer", got)
	}
	if got := agent.mib.Get(dot1qPVID + ".1").Type; got != gosnmp.Gauge32 {
		t.Fatalf("dot1qPVID type = %v, want Gauge32", got)
	}
}

func TestSynthesizePeerTopologyPublishesResolvedLLDPIdentity(t *testing.T) {
	dev := createTestDevice()
	dev.Type = "switch"
	dev.LLDPConfig = &config.LLDPConfig{Enabled: true}
	dev.Interfaces = []config.Interface{{Name: "TenGigabitEthernet1/0/1"}}
	dev.TrunkPorts = []config.TrunkPort{{
		Interface: "TenGigabitEthernet1/0/1", RemoteDevice: "AP01",
		RemoteInterface: "mGigabitEthernet0", VLANs: []int{200, 220}, NativeVLAN: 200,
	}}
	agent := NewAgent(dev, 0)
	apMAC, _ := net.ParseMAC("aa:bb:cc:00:00:31")
	agent.SynthesizePeerTopology(func(string, string) (PeerIdentity, bool) {
		return PeerIdentity{
			MAC: apMAC, Type: "access_point",
			SystemDescription: "Cisco Wireless CW9178I Wi-Fi 7 access point",
		}, true
	})

	row := "0.1.1"
	assertMIBValue(t, agent, lldpRemTable+".1.4."+row, ChassisIDSubtypeMAC)
	assertMIBValue(t, agent, lldpRemTable+".1.5."+row, []byte(apMAC))
	assertMIBValue(
		t,
		agent,
		lldpRemTable+".1.10."+row,
		"Cisco Wireless CW9178I Wi-Fi 7 access point",
	)
	assertMIBValue(t, agent, lldpRemTable+".1.11."+row, []byte{0x10, 0x00})
	assertMIBValue(t, agent, lldpRemTable+".1.12."+row, []byte{0x10, 0x00})
}

func TestSynthesizePeerTopologyPublishesResolvedCDPIdentity(t *testing.T) {
	dev := createTestDevice()
	dev.Type = "switch"
	dev.CDPConfig = &config.CDPConfig{
		Enabled: true, Platform: "Cisco Catalyst C9350-48HX", SoftwareVersion: "IOS XE 17.18",
	}
	dev.Interfaces = []config.Interface{{Name: "TenGigabitEthernet1/0/1"}}
	dev.TrunkPorts = []config.TrunkPort{{
		Interface: "TenGigabitEthernet1/0/1", RemoteDevice: "AP01",
		RemoteInterface: "mGigabitEthernet0", NativeVLAN: 200,
	}}
	agent := NewAgent(dev, 0)
	peerAddress := net.ParseIP("10.240.200.100")
	agent.SynthesizePeerTopology(func(string, string) (PeerIdentity, bool) {
		return PeerIdentity{
			Address: peerAddress, Type: "access-point", CDPEnabled: true,
			CDPPlatform: "Cisco Wireless CW9178I", CDPVersion: "IOS XE 17.15.3",
		}, true
	})

	row := "1.1"
	assertMIBValue(t, agent, cdpCacheTable+".1.4."+row, []byte(peerAddress.To4()))
	assertMIBValue(t, agent, cdpCacheTable+".1.5."+row, "IOS XE 17.15.3")
	assertMIBValue(t, agent, cdpCacheTable+".1.6."+row, "AP01")
	assertMIBValue(t, agent, cdpCacheTable+".1.7."+row, "mGigabitEthernet0")
	assertMIBValue(t, agent, cdpCacheTable+".1.8."+row, "Cisco Wireless CW9178I")
	assertMIBValue(t, agent, cdpCacheTable+".1.9."+row, []byte{0, 0, 0, 0x28})
}

func TestSynthesizePeerTopologyRemovesCDPForNonCDPPeer(t *testing.T) {
	dev := createTestDevice()
	dev.Type = "router"
	dev.CDPConfig = &config.CDPConfig{Enabled: true, Platform: "Cisco Catalyst C8500-12X"}
	dev.Interfaces = []config.Interface{{Name: "TenGigabitEthernet0/1/1"}}
	dev.TrunkPorts = []config.TrunkPort{{
		Interface: "TenGigabitEthernet0/1/1", RemoteDevice: "FW01",
		RemoteInterface: "ethernet1/1", NativeVLAN: 200,
	}}
	agent := NewAgent(dev, 0)
	agent.SynthesizePeerTopology(func(string, string) (PeerIdentity, bool) {
		return PeerIdentity{Type: "firewall"}, true
	})

	for _, oid := range agent.mib.AllOIDs() {
		if strings.HasPrefix(oid, cdpCacheTable+".1.") {
			t.Fatalf("non-CDP firewall published CDP cache row %s", oid)
		}
	}
}

func TestCDPPeerCapabilitiesMatchDeviceAdvertisement(t *testing.T) {
	tests := []struct {
		deviceType string
		want       []byte
	}{
		{deviceType: "router", want: []byte{0, 0, 0, 0x21}},
		{deviceType: "layer3-switch", want: []byte{0, 0, 0, 0x29}},
		{deviceType: "access-point", want: []byte{0, 0, 0, 0x28}},
		{deviceType: "firewall", want: []byte{0, 0, 0, 0x10}},
		{deviceType: "voip_phone", want: []byte{0, 0, 0, 0x10}},
		{deviceType: "custom", want: []byte{0, 0, 0, 0x10}},
	}
	for _, tt := range tests {
		t.Run(tt.deviceType, func(t *testing.T) {
			if got := cdpCapabilities(tt.deviceType); !bytes.Equal(got, tt.want) {
				t.Fatalf("capabilities = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSynthesizePeerTopologyDoesNotTurnAccessPointIntoBridge(t *testing.T) {
	dev := createTestDevice()
	dev.Type = "access-point"
	dev.LLDPConfig = &config.LLDPConfig{Enabled: true}
	dev.Interfaces = []config.Interface{{Name: "mGigabitEthernet0"}}
	dev.TrunkPorts = []config.TrunkPort{{
		Interface: "mGigabitEthernet0", RemoteDevice: "ACCESS-SW01",
		RemoteInterface: "TenGigabitEthernet1/0/1", NativeVLAN: 200,
	}}
	agent := NewAgent(dev, 0)
	switchMAC, _ := net.ParseMAC("aa:bb:cc:00:00:41")
	agent.SynthesizePeerTopology(func(string, string) (PeerIdentity, bool) {
		return PeerIdentity{MAC: switchMAC, Type: "switch"}, true
	})

	assertMIBValue(t, agent, lldpRemTable+".1.9.0.1.1", "ACCESS-SW01")
	for _, oid := range agent.mib.AllOIDs() {
		if oid == dot1dBridge || strings.HasPrefix(oid, dot1dBridge+".") {
			t.Fatalf("access point published bridge OID %s", oid)
		}
	}
}

func TestLLDPPeerCapabilitiesCoverSupportedRoles(t *testing.T) {
	tests := []struct {
		deviceType string
		want       []byte
	}{
		{deviceType: "firewall", want: []byte{0x28, 0x00}},
		{deviceType: "voip_phone", want: []byte{0x05, 0x00}},
	}
	for _, tt := range tests {
		t.Run(tt.deviceType, func(t *testing.T) {
			dev := createTestDevice()
			dev.Type = "switch"
			dev.LLDPConfig = &config.LLDPConfig{Enabled: true}
			dev.Interfaces = []config.Interface{{Name: "GigabitEthernet0/1"}}
			dev.TrunkPorts = []config.TrunkPort{{
				Interface: "GigabitEthernet0/1", RemoteDevice: "PEER",
			}}
			agent := NewAgent(dev, 0)
			mac, _ := net.ParseMAC("aa:bb:cc:00:00:31")
			agent.SynthesizePeerTopology(func(string, string) (PeerIdentity, bool) {
				return PeerIdentity{MAC: mac, Type: tt.deviceType}, true
			})
			assertMIBValue(t, agent, lldpRemTable+".1.11.0.1.1", tt.want)
		})
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
	agent.SynthesizePeerTopology(func(string, string) (PeerIdentity, bool) {
		return PeerIdentity{MAC: mac}, true
	})

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
