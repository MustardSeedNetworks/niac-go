package snmp

import (
	"sort"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// The OIDs and the index encoding below are read off the Cisco AIR-AP1200
// capture in the walk corpus (walks/raw/cisco/cisco-c1200-01.walk), which
// carries one associated client, not off CISCO-DOT11-ASSOCIATION-MIB from
// memory. Its row is
//
//	.1.3.6.1.4.1.9.9.273.1.3.1.1.3.1.11.68.101.109.111.87.101.112.67.108.97.121.2.192.23.164.3.107 = INTEGER: -55
//
// and the index after the column decodes as ifIndex 1, then the SSID as a
// length-prefixed string (11, "DemoWepClay"), then the client MAC as six bare
// octets (02:c0:17:a4:03:6b) -- MacAddress is fixed-length, so it carries no
// length of its own. cDot11ClientAddress (column 1) is absent from the capture
// because it is an index object, and this agent does not serve it either.
const (
	oracleClientParentAddress = "1.3.6.1.4.1.9.9.273.1.2.1.1.2"
	oracleClientIPAddressType = "1.3.6.1.4.1.9.9.273.1.2.1.1.15"
	oracleClientIPAddress     = "1.3.6.1.4.1.9.9.273.1.2.1.1.16"
	oracleClientUpTime        = "1.3.6.1.4.1.9.9.273.1.3.1.1.2"
	oracleClientSignal        = "1.3.6.1.4.1.9.9.273.1.3.1.1.3"
	oracleClientSigQuality    = "1.3.6.1.4.1.9.9.273.1.3.1.1.4"

	// The SSID "corp-wifi" as the index carries it: its length, then its
	// octets. Written out rather than computed, for the same reason the OIDs
	// are: an index built by the code under test cannot disagree with it.
	oracleSSIDIndex = "9.99.111.114.112.45.119.105.102.105"

	// The two client MACs of wifiAPWithClients, as six bare octets each.
	oracleFirstClientMAC  = "2.192.23.164.3.107"
	oracleSecondClientMAC = "170.187.204.221.238.255"
)

// wifiAPWithClients is the two-radio AP of mib_dot11_test.go with two clients
// associated to its 2.4 GHz radio and none to its 5 GHz one, so the tests can
// tell a table keyed by the radio from one keyed by the device.
func wifiAPWithClients() *config.Device {
	device := wifiAP()
	device.WiFiConfig.Radios[0].Clients = []config.WiFiClient{
		{
			MAC:               "02:c0:17:a4:03:6b",
			IPAddress:         "10.250.8.219",
			AssociatedSeconds: 115286,
			SignalDBM:         -55,
			SignalQualityPct:  83,
		},
		{
			MAC:               "aa:bb:cc:dd:ee:ff",
			IPAddress:         "10.250.8.220",
			AssociatedSeconds: 42,
			SignalDBM:         -72,
			SignalQualityPct:  40,
		},
	}

	return device
}

// clientIndex is the index of one client row: the radio's ifIndex, the SSID it
// associated on, and its own MAC.
func clientIndex(t *testing.T, agent *Agent, radio, mac string) string {
	t.Helper()

	return radioIndex(t, agent, radio) + "." + oracleSSIDIndex + "." + mac
}

// TestDot11ClientRowsAreKeyedByRadioSSIDAndClient is the index clause: a client
// is reported under the radio it associated to, so an NMS reading two radios
// can say which one a station is on. Keying the table by the device instead
// would report both radios' clients at both radios.
func TestDot11ClientRowsAreKeyedByRadioSSIDAndClient(t *testing.T) {
	agent := NewAgent(wifiAPWithClients(), 0)
	first := clientIndex(t, agent, "Dot11Radio0", oracleFirstClientMAC)
	second := clientIndex(t, agent, "Dot11Radio1", oracleFirstClientMAC)

	wantInt(t, agent.mib.Get(oracleClientSignal+"."+first), -55,
		"cDot11ClientSignalStrength on the radio the client associated to")
	if row := agent.mib.Get(oracleClientSignal + "." + second); row != nil {
		t.Errorf("the 5 GHz radio answers for a 2.4 GHz client: %s", oidValueString(row))
	}
}

// TestDot11ClientReportsTheAuthoredAssociation: every value on the wire is one
// an author wrote, and the AP a client is on is its radio's BSSID -- the same
// address dot11MACAddress reports for that radio, so the two agree.
func TestDot11ClientReportsTheAuthoredAssociation(t *testing.T) {
	agent := NewAgent(wifiAPWithClients(), 0)
	index := clientIndex(t, agent, "Dot11Radio0", oracleFirstClientMAC)

	wantMAC(t, agent, oracleClientParentAddress+"."+index, "00:0c:ce:88:23:c7")
	wantInt(t, agent.mib.Get(oracleClientUpTime+"."+index), 115286, "cDot11ClientUpTime")
	wantInt(t, agent.mib.Get(oracleClientSignal+"."+index), -55, "cDot11ClientSignalStrength")
	wantInt(t, agent.mib.Get(oracleClientSigQuality+"."+index), 83, "cDot11ClientSigQuality")
	wantInt(t, agent.mib.Get(oracleClientIPAddressType+"."+index), 1, "cDot11ClientIpAddressType ipv4")

	// cDot11ClientIpAddress is an InetAddress: the four address octets, which
	// is how the capture carries it, not the dotted text.
	wantString(t, agent.mib.Get(oracleClientIPAddress+"."+index),
		string([]byte{10, 250, 8, 219}), "cDot11ClientIpAddress")
}

// TestDot11ClientTableIsAbsentWithoutAuthoredClients: an AP whose radios have
// no authored client answers no association table at all. An empty table and a
// radio with nobody on it read the same to an NMS, but only one of them is a
// statement this scenario is entitled to make.
func TestDot11ClientTableIsAbsentWithoutAuthoredClients(t *testing.T) {
	agent := NewAgent(wifiAP(), 0)

	if oids := dot11ClientOIDs(agent); len(oids) != 0 {
		t.Errorf("an AP with no authored client answers %d association objects: %v",
			len(oids), oids)
	}
}

// TestDot11ClientServesOnlyTheObjectsTheCaptureProves is the positive clause:
// the exact set, so an object nobody verified against the corpus cannot land
// silently. cDot11ClientAid, the rate sets and the cipher columns are in the
// capture too, but nothing authors them, and a plausible default in a
// discovery tool's report is worse than a gap.
func TestDot11ClientServesOnlyTheObjectsTheCaptureProves(t *testing.T) {
	agent := NewAgent(wifiAPWithClients(), 0)

	var want []string
	for _, mac := range []string{oracleFirstClientMAC, oracleSecondClientMAC} {
		index := clientIndex(t, agent, "Dot11Radio0", mac)
		want = append(want,
			oracleClientParentAddress+"."+index,
			oracleClientIPAddressType+"."+index,
			oracleClientIPAddress+"."+index,
			oracleClientUpTime+"."+index,
			oracleClientSignal+"."+index,
			oracleClientSigQuality+"."+index,
		)
	}
	sort.Strings(want)

	if got := dot11ClientOIDs(agent); !equalStrings(got, want) {
		t.Errorf("association objects served:\n got %v\nwant %v", got, want)
	}
}

// TestDot11ClientsNotSynthesizedOverACaptureThatHasThem: a capture that carries
// its own association table keeps it, for the reason the radios themselves do
// -- a real AP is the authority on who is associated to it, and overwriting
// that would register as an unclassified substitution in the replay-fidelity
// contract.
func TestDot11ClientsNotSynthesizedOverACaptureThatHasThem(t *testing.T) {
	device := wifiAPWithClients()
	device.SNMPConfig.WalkFile = writeWalk(t, ""+
		".1.3.6.1.2.1.1.5.0 = STRING: MED-AP-01\r\n"+
		".1.3.6.1.2.1.2.2.1.2.1 = STRING: Dot11Radio0\r\n"+
		".1.3.6.1.2.1.2.2.1.3.1 = INTEGER: 71\r\n"+
		".1.3.6.1.4.1.9.9.273.1.3.1.1.3.1.4.116.101.115.116.1.2.3.4.5.6 = INTEGER: -61\r\n")

	agent := NewAgent(device, 0)
	if err := agent.LoadWalkFile(device.SNMPConfig.WalkFile); err != nil {
		t.Fatalf("LoadWalkFile() error = %v", err)
	}

	wantInt(t, agent.mib.Get(oracleClientSignal+".1.4.116.101.115.116.1.2.3.4.5.6"), -61,
		"cDot11ClientSignalStrength — the capture's own client")
	if row := agent.mib.Get(oracleClientSignal + ".1." + oracleSSIDIndex + "." + oracleFirstClientMAC); row != nil {
		t.Errorf("an authored client overwrote a captured association table: %s",
			oidValueString(row))
	}
}

// TestDot11ClientsSynthesizedForAWalkWithoutThem is the other half: a captured
// AP that answers no association table gets the authored clients, at the
// capture's own ifIndex.
func TestDot11ClientsSynthesizedForAWalkWithoutThem(t *testing.T) {
	device := wifiAPWithClients()
	device.SNMPConfig.WalkFile = writeWalk(t, ""+
		".1.3.6.1.2.1.1.5.0 = STRING: MED-AP-01\r\n"+
		".1.3.6.1.2.1.2.2.1.2.11 = STRING: Dot11Radio0\r\n"+
		".1.3.6.1.2.1.2.2.1.3.11 = INTEGER: 71\r\n")

	agent := NewAgent(device, 0)
	if oids := dot11ClientOIDs(agent); len(oids) != 0 {
		t.Fatalf("a walk-backed AP was given client rows before its walk loaded: %v", oids)
	}
	if err := agent.LoadWalkFile(device.SNMPConfig.WalkFile); err != nil {
		t.Fatalf("LoadWalkFile() error = %v", err)
	}

	wantInt(t, agent.mib.Get(oracleClientSignal+".11."+oracleSSIDIndex+"."+oracleFirstClientMAC),
		-55, "cDot11ClientSignalStrength at the capture's own ifIndex")
}

func TestWalkOwnsDot11Clients(t *testing.T) {
	captured := []WalkEntry{{OID: oracleClientSignal + ".1.4.116.101.115.116.1.2.3.4.5.6"}}
	if !walkOwnsDot11Clients(captured) {
		t.Error("walkOwnsDot11Clients() missed an association object in the capture")
	}
	// The radios and the clients are separate decisions: an AP whose capture
	// carries IEEE802dot11-MIB but no association table still gets clients.
	if walkOwnsDot11Clients([]WalkEntry{{OID: oracleDesiredSSID + ".1"}}) {
		t.Error("walkOwnsDot11Clients() claimed a capture with only dot11 radio objects")
	}
}

func dot11ClientOIDs(agent *Agent) []string {
	var found []string
	for _, oid := range agent.mib.AllOIDs() {
		if strings.HasPrefix(oid, cDot11AssociationRoot+".") {
			found = append(found, oid)
		}
	}
	sort.Strings(found)

	return found
}
