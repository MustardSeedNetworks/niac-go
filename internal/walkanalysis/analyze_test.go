package walkanalysis

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// realisticWalk is a net-snmp-style capture of a non-Cisco device that carries
// system identity, a two-row IF-MIB + ifXTable, an LLDP neighbor (3-component
// timeMark.localPortNum.remIndex index, mirroring the simulator's synthesis in
// internal/protocols/snmp/mib_discovery.go), and a CDP neighbor.
const realisticWalk = `.1.3.6.1.2.1.1.1.0 = STRING: "Juniper Networks EX4300, JUNOS 21.4R3"
.1.3.6.1.2.1.1.2.0 = OID: .1.3.6.1.4.1.2636.1.1.1.2.30
.1.3.6.1.2.1.1.4.0 = STRING: "netops@example.net"
.1.3.6.1.2.1.1.5.0 = STRING: "dist-sw-01"
.1.3.6.1.2.1.1.6.0 = STRING: "IDF-3 Rack B12"
.1.3.6.1.2.1.2.2.1.2.1 = STRING: "ge-0/0/1"
.1.3.6.1.2.1.2.2.1.3.1 = INTEGER: 6
.1.3.6.1.2.1.2.2.1.5.1 = Gauge32: 1000000000
.1.3.6.1.2.1.2.2.1.6.1 = Hex-STRING: 00 1A 2B 3C 4D 5E
.1.3.6.1.2.1.2.2.1.7.1 = INTEGER: 1
.1.3.6.1.2.1.2.2.1.8.1 = INTEGER: 1
.1.3.6.1.2.1.2.2.1.2.2 = STRING: "lo0"
.1.3.6.1.2.1.2.2.1.3.2 = INTEGER: 24
.1.3.6.1.2.1.2.2.1.7.2 = INTEGER: 1
.1.3.6.1.2.1.2.2.1.8.2 = INTEGER: 1
.1.3.6.1.2.1.31.1.1.1.1.1 = STRING: "ge-0/0/1"
.1.3.6.1.2.1.31.1.1.1.15.1 = Gauge32: 1000
.1.3.6.1.2.1.31.1.1.1.18.1 = STRING: "uplink to core"
.1.0.8802.1.1.2.1.3.7.1.3.1 = STRING: "ge-0/0/1"
.1.0.8802.1.1.2.1.4.1.1.7.0.1.1 = STRING: "xe-1/0/0"
.1.0.8802.1.1.2.1.4.1.1.9.0.1.1 = STRING: "core-rtr-01"
.1.3.6.1.4.1.9.9.23.1.2.1.1.6.1.1 = STRING: "core-rtr-01"
.1.3.6.1.4.1.9.9.23.1.2.1.1.7.1.1 = STRING: "TenGigE0/0/0"
`

func analyzeFixture(t *testing.T, walk string) *Analysis {
	t.Helper()
	path := filepath.Join(t.TempDir(), "device.walk")
	if err := os.WriteFile(path, []byte(walk), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	analysis, err := AnalyzeFile(path)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}
	return analysis
}

func TestAnalyzeDeviceIdentity(t *testing.T) {
	got := analyzeFixture(t, realisticWalk).Device
	want := Device{
		SysName:     "dist-sw-01",
		SysDescr:    "Juniper Networks EX4300, JUNOS 21.4R3",
		SysObjectID: "1.3.6.1.4.1.2636.1.1.1.2.30",
		SysContact:  "netops@example.net",
		SysLocation: "IDF-3 Rack B12",
	}
	if got != want {
		t.Errorf("device = %+v, want %+v", got, want)
	}
}

func TestAnalyzeInterfaces(t *testing.T) {
	got := analyzeFixture(t, realisticWalk)
	if len(got.Interfaces) != 2 {
		t.Fatalf("interfaces = %d, want 2 (%+v)", len(got.Interfaces), got.Interfaces)
	}

	eth := got.Interfaces[0]
	want := Interface{
		Index:       1,
		Name:        "ge-0/0/1",
		Description: "uplink to core",
		Type:        "ethernetCsmacd",
		Speed:       1000000000,
		AdminStatus: "up",
		OperStatus:  "up",
		MACAddress:  "00:1A:2B:3C:4D:5E",
	}
	if eth != want {
		t.Errorf("interface[0] = %+v, want %+v", eth, want)
	}

	lo := got.Interfaces[1]
	if lo.Name != "lo0" || lo.Type != "softwareLoopback" || lo.MACAddress != "" {
		t.Errorf("interface[1] = %+v, want lo0/softwareLoopback/no-mac", lo)
	}
}

func TestAnalyzeStats(t *testing.T) {
	got := analyzeFixture(t, realisticWalk).Statistics
	want := Stats{
		TotalInterfaces:    2,
		PhysicalInterfaces: 1,
		LogicalInterfaces:  1,
		TotalNeighbors:     2,
	}
	if got != want {
		t.Errorf("stats = %+v, want %+v", got, want)
	}
}

func TestAnalyzeNeighbors(t *testing.T) {
	got := analyzeFixture(t, realisticWalk).Neighbors
	// Sorted by LocalInterface, then Protocol: cdp precedes lldp.
	want := []Neighbor{
		{
			LocalInterface:  "ge-0/0/1",
			RemoteDevice:    "core-rtr-01",
			RemoteInterface: "TenGigE0/0/0",
			Protocol:        "cdp",
		},
		{
			LocalInterface:  "ge-0/0/1",
			RemoteDevice:    "core-rtr-01",
			RemoteInterface: "xe-1/0/0",
			Protocol:        "lldp",
		},
	}
	if len(got) != len(want) {
		t.Fatalf("neighbors = %d, want %d (%+v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("neighbor[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestAnalyzeNonCiscoDescr guards the regression fixed by this rewrite: the old
// regex parser only matched sysDescr lines containing "Cisco".
func TestAnalyzeNonCiscoDescr(t *testing.T) {
	got := analyzeFixture(t, realisticWalk).Device
	if got.SysDescr == "" {
		t.Error("non-Cisco sysDescr was dropped")
	}
}

func TestAnalyzeEmptyWalk(t *testing.T) {
	got := Analyze(nil)
	if got == nil {
		t.Fatal("Analyze(nil) returned nil")
	}
	if len(got.Interfaces) != 0 || len(got.Neighbors) != 0 {
		t.Errorf("empty walk yielded %+v", got)
	}
	if got.Statistics != (Stats{}) {
		t.Errorf("empty stats = %+v", got.Statistics)
	}
}

// TestLLDPTwoComponentIndex covers the abbreviated LLDP remote index used by
// some captures (localPortNum.remIndex) rather than the full 3-component form.
func TestLLDPTwoComponentIndex(t *testing.T) {
	walk := `.1.3.6.1.2.1.2.2.1.2.5 = STRING: "Gi1/0/5"
.1.3.6.1.2.1.2.2.1.3.5 = INTEGER: 6
.1.0.8802.1.1.2.1.4.1.1.9.5.1 = STRING: "neighbor-x"
.1.0.8802.1.1.2.1.4.1.1.7.5.1 = STRING: "Gi0/1"
`
	got := analyzeFixture(t, walk).Neighbors
	if len(got) != 1 {
		t.Fatalf("neighbors = %d, want 1 (%+v)", len(got), got)
	}
	// index "5.1" -> localPortNum = parts[len-2] = "5" -> IF-MIB ifIndex 5 = Gi1/0/5.
	if got[0].LocalInterface != "Gi1/0/5" || got[0].RemoteDevice != "neighbor-x" {
		t.Errorf("neighbor = %+v, want local Gi1/0/5 / remote neighbor-x", got[0])
	}
}

func TestInterfaceSpeedHighSpeedFallback(t *testing.T) {
	// ifSpeed pegged at the 32-bit ceiling -> ifHighSpeed (Mbps) wins.
	entries := []snmp.WalkEntry{
		{OID: ".1.3.6.1.2.1.2.2.1.2.1", Value: "Hu0/0/0/0"},
		{OID: ".1.3.6.1.2.1.2.2.1.3.1", Value: 6},
		{OID: ".1.3.6.1.2.1.2.2.1.5.1", Value: uint(4294967295)},
		{OID: ".1.3.6.1.2.1.31.1.1.1.15.1", Value: uint(100000)},
	}
	got := Analyze(entries)
	if len(got.Interfaces) != 1 {
		t.Fatalf("interfaces = %d, want 1", len(got.Interfaces))
	}
	const want = int64(100000) * bitsPerMbit
	if got.Interfaces[0].Speed != want {
		t.Errorf("speed = %d, want %d", got.Interfaces[0].Speed, want)
	}
}

func TestInterfaceMTUIsPreserved(t *testing.T) {
	entries := []snmp.WalkEntry{
		{OID: ".1.3.6.1.2.1.2.2.1.2.1", Value: "HundredGigE0/0/0/1"},
		{OID: ".1.3.6.1.2.1.2.2.1.3.1", Value: 6},
		{OID: ".1.3.6.1.2.1.2.2.1.4.1", Value: 1500},
	}
	got := Analyze(entries)
	if len(got.Interfaces) != 1 || got.Interfaces[0].MTU != 1500 {
		t.Fatalf("interfaces = %+v, want captured MTU 1500", got.Interfaces)
	}
}

func TestFormatMAC(t *testing.T) {
	tests := map[string]string{
		"001A2B3C4D5E":          "00:1A:2B:3C:4D:5E",
		"00 1a 2b 3c 4d 5e":     "00:1A:2B:3C:4D:5E",
		"0x00112233445566":      "00:11:22:33:44:55:66",
		"00:11:22:33:44:55":     "00:11:22:33:44:55",
		"":                      "",
		"not-a-mac":             "",
		"abc":                   "", // odd length
		"switch-hostname-value": "",
	}
	for in, want := range tests {
		if got := formatMAC(in); got != want {
			t.Errorf("formatMAC(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitColIndex(t *testing.T) {
	col, idx, ok := splitColIndex("1.3.6.1.2.1.2.2.1.5.7", ifTablePrefix)
	if !ok || col != "5" || idx != "7" {
		t.Errorf("splitColIndex = (%q,%q,%v), want (5,7,true)", col, idx, ok)
	}
	if _, _, matched := splitColIndex("1.3.6.1.2.1.1.5.0", ifTablePrefix); matched {
		t.Error("splitColIndex matched an unrelated OID")
	}
	if _, _, matched := splitColIndex("1.3.6.1.2.1.2.2.1.5", ifTablePrefix); matched {
		t.Error("splitColIndex accepted a column with no index")
	}
}

func TestLLDPLocalPortNum(t *testing.T) {
	cases := map[string]string{
		"0.1.1": "1", // timeMark.localPortNum.remIndex
		"5.1":   "5", // localPortNum.remIndex
		"3":     "3",
		"":      "",
	}
	for index, want := range cases {
		if got := lldpLocalPortNum(index); got != want {
			t.Errorf("lldpLocalPortNum(%q) = %q, want %q", index, got, want)
		}
	}
}
