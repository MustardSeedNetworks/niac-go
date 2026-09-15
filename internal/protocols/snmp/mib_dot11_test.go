package snmp

import (
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// The OIDs below are written as literals on purpose. They are read off the
// Cisco AIR-AP1200 capture in the walk corpus
// (walks/raw/cisco/cisco-c1200-01.walk), which answered dot11StationID and
// dot11MACAddress as the radio MAC, dot11PHYType as dsss(2) on its 2.4 GHz
// radio and ofdm(4) on its 5 GHz one, and dot11CurrentChannel only for the
// DSSS radio. A test that used the constants under test could not tell a wrong
// subtree from a right one -- the lesson POWER-ETHERNET-MIB taught when two of
// its three tables shipped one arc short.
const (
	oracleStationID       = "1.2.840.10036.1.1.1.1"
	oracleDesiredSSID     = "1.2.840.10036.1.1.1.9"
	oracleMACAddress      = "1.2.840.10036.2.1.1.1"
	oraclePHYType         = "1.2.840.10036.4.1.1.1"
	oraclePowerLevels     = "1.2.840.10036.4.3.1.1"
	oracleTxPowerLevel1   = "1.2.840.10036.4.3.1.2"
	oracleCurrentTxPower  = "1.2.840.10036.4.3.1.10"
	oracleCurrentChannel  = "1.2.840.10036.4.5.1.1"
	oracleCurrentFrequenc = "1.2.840.10036.4.11.1.1"
)

// wifiAP is a two-radio access point: 2.4 GHz on channel 6 and 5 GHz on
// channel 149, the shape every pack's Wi-Fi tier is authored in.
func wifiAP() *config.Device {
	device := createTestDevice()
	device.Name = "MED-AP-01"
	device.Type = "ap"
	device.Interfaces = []config.Interface{
		{Name: "Dot11Radio0", Type: "ieee80211", AdminStatus: "up"},
		{Name: "Dot11Radio1", Type: "ieee80211", AdminStatus: "up"},
		{Name: "GigabitEthernet0", Type: "ethernet", AdminStatus: "up"},
	}
	device.WiFiConfig = &config.WiFiConfig{Radios: []config.WiFiRadio{
		{
			Interface:  "Dot11Radio0",
			SSID:       "corp-wifi",
			BSSID:      "00:0c:ce:88:23:c7",
			Band:       "2.4GHz",
			Channel:    6,
			TxPowerDBM: 17,
		},
		{
			Interface:  "Dot11Radio1",
			SSID:       "corp-wifi",
			BSSID:      "00:0b:fd:d4:6f:de",
			Band:       "5GHz",
			Channel:    149,
			TxPowerDBM: 20,
		},
	}}

	return device
}

func radioIndex(t *testing.T, agent *Agent, name string) string {
	t.Helper()
	index, ok := agent.ifIndexForInterface(name)
	if !ok {
		t.Fatalf("no ifIndex for %s", name)
	}

	return index
}

func TestDot11IdentityFollowsTheAuthoredRadio(t *testing.T) {
	agent := NewAgent(wifiAP(), 0)
	first := radioIndex(t, agent, "Dot11Radio0")
	second := radioIndex(t, agent, "Dot11Radio1")

	wantString(t, agent.mib.Get(oracleDesiredSSID+"."+first), "corp-wifi", "dot11DesiredSSID")
	wantMAC(t, agent, oracleStationID+"."+first, "00:0c:ce:88:23:c7")
	wantMAC(t, agent, oracleMACAddress+"."+first, "00:0c:ce:88:23:c7")
	wantMAC(t, agent, oracleMACAddress+"."+second, "00:0b:fd:d4:6f:de")

	// The BSSID is the radio's address, so IF-MIB must report the same one:
	// two addresses for one radio is a finding a tester raises against us.
	wantMAC(t, agent, ifPhysAddress+"."+first, "00:0c:ce:88:23:c7")
}

func TestDot11PhyFollowsTheAuthoredBand(t *testing.T) {
	agent := NewAgent(wifiAP(), 0)
	first := radioIndex(t, agent, "Dot11Radio0")
	second := radioIndex(t, agent, "Dot11Radio1")

	wantInt(t, agent.mib.Get(oraclePHYType+"."+first), 2, "dot11PHYType 2.4 GHz")
	wantInt(t, agent.mib.Get(oraclePHYType+"."+second), 4, "dot11PHYType 5 GHz")

	// A 2.4 GHz channel is reported by the DSSS table and a 5 GHz one by the
	// OFDM table; the capture's own 5 GHz radio answers no DSSS row at all.
	wantInt(t, agent.mib.Get(oracleCurrentChannel+"."+first), 6, "dot11CurrentChannel")
	if row := agent.mib.Get(oracleCurrentChannel + "." + second); row != nil {
		t.Errorf("5 GHz radio answers dot11CurrentChannel = %s", oidValueString(row))
	}
	wantInt(t, agent.mib.Get(oracleCurrentFrequenc+"."+second), 149, "dot11CurrentFrequency")
	if row := agent.mib.Get(oracleCurrentFrequenc + "." + first); row != nil {
		t.Errorf("2.4 GHz radio answers dot11CurrentFrequency = %s", oidValueString(row))
	}
}

// TestDot11TxPowerIsReportedInMilliwatts: the MIB's unit is mW and the author
// writes dBm, so the conversion is the whole content of these three rows.
func TestDot11TxPowerIsReportedInMilliwatts(t *testing.T) {
	agent := NewAgent(wifiAP(), 0)
	first := radioIndex(t, agent, "Dot11Radio0")
	second := radioIndex(t, agent, "Dot11Radio1")

	wantInt(t, agent.mib.Get(oraclePowerLevels+"."+first), 1, "dot11NumberSupportedPowerLevels")
	wantInt(t, agent.mib.Get(oracleCurrentTxPower+"."+first), 1, "dot11CurrentTxPowerLevel")
	wantInt(t, agent.mib.Get(oracleTxPowerLevel1+"."+first), 50, "17 dBm in mW")
	wantInt(t, agent.mib.Get(oracleTxPowerLevel1+"."+second), 100, "20 dBm in mW")
}

// TestDot11IsAbsentWithoutAnAuthoredRadio: a device with no wifi block is not
// an access point, and an empty dot11 table would make it look like one.
func TestDot11IsAbsentWithoutAnAuthoredRadio(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)

	if oids := dot11OIDs(agent); len(oids) != 0 {
		t.Errorf("a router answers %d dot11 objects: %v", len(oids), oids)
	}
}

// TestDot11ServesOnlyTheObjectsTheCaptureProves is the positive clause: the
// exact set, so an object nobody verified cannot land silently.
func TestDot11ServesOnlyTheObjectsTheCaptureProves(t *testing.T) {
	agent := NewAgent(wifiAP(), 0)
	first := radioIndex(t, agent, "Dot11Radio0")
	second := radioIndex(t, agent, "Dot11Radio1")

	want := []string{
		oracleStationID + "." + first,
		oracleStationID + "." + second,
		oracleDesiredSSID + "." + first,
		oracleDesiredSSID + "." + second,
		oracleMACAddress + "." + first,
		oracleMACAddress + "." + second,
		oraclePHYType + "." + first,
		oraclePHYType + "." + second,
		oraclePowerLevels + "." + first,
		oraclePowerLevels + "." + second,
		oracleTxPowerLevel1 + "." + first,
		oracleTxPowerLevel1 + "." + second,
		oracleCurrentTxPower + "." + first,
		oracleCurrentTxPower + "." + second,
		oracleCurrentChannel + "." + first,
		oracleCurrentFrequenc + "." + second,
	}
	sort.Strings(want)

	if got := dot11OIDs(agent); !equalStrings(got, want) {
		t.Errorf("dot11 objects served:\n got %v\nwant %v", got, want)
	}
}

// TestDot11SynthesizedForAWalkWithoutIt: a walk-backed AP gets nothing at
// construction, because only the loaded capture can say whether it carries
// radios of its own -- and the capture also owns the ifIndexes these rows are
// keyed by, which is why the synthesized rows land at the walk's numbering
// rather than at one this agent invented.
func TestDot11SynthesizedForAWalkWithoutIt(t *testing.T) {
	device := wifiAP()
	device.SNMPConfig.WalkFile = writeWalk(t, ""+
		".1.3.6.1.2.1.1.5.0 = STRING: MED-AP-01\r\n"+
		".1.3.6.1.2.1.2.2.1.2.11 = STRING: Dot11Radio0\r\n"+
		".1.3.6.1.2.1.2.2.1.3.11 = INTEGER: 71\r\n")

	agent := NewAgent(device, 0)
	if oids := dot11OIDs(agent); len(oids) != 0 {
		t.Fatalf("a walk-backed AP was given dot11 rows before its walk loaded: %v", oids)
	}
	if err := agent.LoadWalkFile(device.SNMPConfig.WalkFile); err != nil {
		t.Fatalf("LoadWalkFile() error = %v", err)
	}

	wantString(t, agent.mib.Get(oracleDesiredSSID+".11"), "corp-wifi",
		"dot11DesiredSSID at the capture's own ifIndex")
	wantInt(t, agent.mib.Get(oracleCurrentChannel+".11"), 6, "dot11CurrentChannel")
	// The walk path runs several more refreshes after the dot11 one, and one of
	// them rewrites ifPhysAddress: the BSSID has to survive all of them, or a
	// walk-backed AP reports two addresses for one radio.
	wantMAC(t, agent, ifPhysAddress+".11", "00:0c:ce:88:23:c7")
}

// The other half: a capture that already carries IEEE802dot11-MIB keeps it. The
// rows below are the AIR-AP1200's own, and a real AP is the authority on its
// radios -- overwriting them would register as an unclassified substitution in
// the replay-fidelity contract.
func TestDot11NotSynthesizedOverACaptureThatHasIt(t *testing.T) {
	device := wifiAP()
	device.SNMPConfig.WalkFile = writeWalk(t, ""+
		".1.3.6.1.2.1.1.5.0 = STRING: MED-AP-01\r\n"+
		".1.3.6.1.2.1.2.2.1.2.1 = STRING: Dot11Radio0\r\n"+
		".1.3.6.1.2.1.2.2.1.3.1 = INTEGER: 71\r\n"+
		".1.2.840.10036.1.1.1.9.1 = STRING: captured-ssid\r\n"+
		".1.2.840.10036.4.5.1.1.1 = INTEGER: 11\r\n")

	agent := NewAgent(device, 0)
	if err := agent.LoadWalkFile(device.SNMPConfig.WalkFile); err != nil {
		t.Fatalf("LoadWalkFile() error = %v", err)
	}

	wantString(t, agent.mib.Get(oracleDesiredSSID+".1"), "captured-ssid",
		"dot11DesiredSSID — the capture's own SSID, not the authored one")
	wantInt(t, agent.mib.Get(oracleCurrentChannel+".1"), 11,
		"dot11CurrentChannel — the capture's own channel")
}

func TestWalkOwnsDot11(t *testing.T) {
	captured := []WalkEntry{
		{OID: oracleDesiredSSID + ".1", Type: gosnmp.OctetString, Value: "captured-ssid"},
	}
	if !walkOwnsDot11(captured) {
		t.Error("walkOwnsDot11() missed a dot11 object in the capture")
	}
	if walkOwnsDot11([]WalkEntry{{OID: "1.3.6.1.2.1.1.1.0"}}) {
		t.Error("walkOwnsDot11() claimed a capture with no dot11 object")
	}
}

func writeWalk(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ap.walk")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func dot11OIDs(agent *Agent) []string {
	var found []string
	for _, oid := range agent.mib.AllOIDs() {
		if strings.HasPrefix(oid, dot11Root+".") {
			found = append(found, oid)
		}
	}
	sort.Strings(found)

	return found
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}

func wantString(t *testing.T, value *OIDValue, want, what string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s is absent", what)
	}
	if got := oidValueString(value); got != want {
		t.Errorf("%s = %q, want %q", what, got, want)
	}
}

func wantMAC(t *testing.T, agent *Agent, oid, want string) {
	t.Helper()
	value := agent.mib.Get(oid)
	if value == nil {
		t.Fatalf("%s is absent", oid)
	}
	bytes, ok := value.Value.([]byte)
	if !ok {
		t.Fatalf("%s = %T, want the six octets of a MAC", oid, value.Value)
	}
	if got := net.HardwareAddr(bytes).String(); got != want {
		t.Errorf("%s = %s, want %s", oid, got, want)
	}
}
