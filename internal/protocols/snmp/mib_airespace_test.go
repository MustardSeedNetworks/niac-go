package snmp

import (
	"net"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// The OIDs below are read off the controller captures in the walk corpus
// (walks/raw/cisco/cisco-controller-03.walk), not off AIRESPACE-WIRELESS-MIB
// from memory. One associated station there is
//
//	.1.3.6.1.4.1.14179.2.1.4.1.4.0.33.106.197.225.52 = Hex-STRING: 00 17 DF A1 0F D0
//	.1.3.6.1.4.1.14179.2.2.1.1.3.0.23.223.161.15.208 = STRING: "Cisco1252"
//
// so a station row is keyed by its own MAC as six bare octets, and its AP
// column is the six-octet key of the AP's row. Columns 5, 7 and 9 of the same
// station answer 0, "Cisco4400" and 3: the radio slot, the SSID and
// associated(3).
const (
	oracleStationMAC    = "1.3.6.1.4.1.14179.2.1.4.1.1"
	oracleStationIP     = "1.3.6.1.4.1.14179.2.1.4.1.2"
	oracleStationAPMAC  = "1.3.6.1.4.1.14179.2.1.4.1.4"
	oracleStationSlot   = "1.3.6.1.4.1.14179.2.1.4.1.5"
	oracleStationSSID   = "1.3.6.1.4.1.14179.2.1.4.1.7"
	oracleStationStatus = "1.3.6.1.4.1.14179.2.1.4.1.9"

	oracleAPMAC      = "1.3.6.1.4.1.14179.2.2.1.1.1"
	oracleAPSlots    = "1.3.6.1.4.1.14179.2.2.1.1.2"
	oracleAPName     = "1.3.6.1.4.1.14179.2.2.1.1.3"
	oracleAPLocation = "1.3.6.1.4.1.14179.2.2.1.1.4"
	oracleAPStatus   = "1.3.6.1.4.1.14179.2.2.1.1.6"
	oracleAPModel    = "1.3.6.1.4.1.14179.2.2.1.1.16"
	oracleAPAddress  = "1.3.6.1.4.1.14179.2.2.1.1.19"

	// createTestDevice's MAC, 00:11:22:33:44:55, as six bare octets.
	oracleAPIndex = "0.17.34.51.68.85"
)

func wirelessController() *config.Device {
	device := createTestDevice()
	device.Name = "MED-WLC01"
	device.Type = "server"
	device.MACAddress = net.HardwareAddr{0x00, 0x1d, 0x45, 0x91, 0x11, 0xd0}

	return device
}

func joinedAP() *config.Device {
	device := wifiAPWithClients()
	device.WiFiConfig.Controller = "MED-WLC01"
	device.SNMPConfig.SysLocation = "Building A, floor 2"
	device.Properties["model"] = "C9120AXI"
	// One station on the 5 GHz radio as well, so the slot column has to
	// follow the radio rather than default to the first.
	device.WiFiConfig.Radios[1].Clients = []config.WiFiClient{{
		MAC: "02:00:00:00:00:05", IPAddress: "10.250.8.221",
		AssociatedSeconds: 7, SignalDBM: -61, SignalQualityPct: 70,
	}}

	return device
}

// TestControllerReportsTheAPsThatJoinedIt is the AP-table clause: a controller
// names each joined AP in a row keyed by the AP's MAC, and says what it is and
// where.
func TestControllerReportsTheAPsThatJoinedIt(t *testing.T) {
	agent := NewAgent(wirelessController(), 0)
	agent.SynthesizeWirelessController([]*config.Device{joinedAP()})
	row := "." + oracleAPIndex

	wantMAC(t, agent, oracleAPMAC+row, "00:11:22:33:44:55")
	wantInt(t, agent.mib.Get(oracleAPSlots+row), 2, "bsnAPNumOfSlots")
	wantString(t, agent.mib.Get(oracleAPName+row), "MED-AP-01", "bsnAPName")
	wantString(t, agent.mib.Get(oracleAPLocation+row), "Building A, floor 2", "bsnAPLocation")
	wantInt(t, agent.mib.Get(oracleAPStatus+row), 1, "bsnAPOperationStatus associated(1)")
	wantString(t, agent.mib.Get(oracleAPModel+row), "C9120AXI", "bsnAPModel")
	wantString(t, agent.mib.Get(oracleAPAddress+row), "192.168.1.1", "bsnApIpAddress")
}

// TestControllerReportsEachStationOnTheAPAndRadioItIsOn is the station clause:
// every client of every joined AP is one row, and its AP column is that AP's
// row key -- the join an NMS makes to say which AP a station is on.
func TestControllerReportsEachStationOnTheAPAndRadioItIsOn(t *testing.T) {
	agent := NewAgent(wirelessController(), 0)
	agent.SynthesizeWirelessController([]*config.Device{joinedAP()})

	tests := []struct {
		mac, index, address string
		slot                int
	}{
		{"02:c0:17:a4:03:6b", oracleFirstClientMAC, "10.250.8.219", 0},
		{"aa:bb:cc:dd:ee:ff", oracleSecondClientMAC, "10.250.8.220", 0},
		{"02:00:00:00:00:05", "2.0.0.0.0.5", "10.250.8.221", 1},
	}
	for _, tt := range tests {
		t.Run(tt.mac, func(t *testing.T) {
			row := "." + tt.index
			wantMAC(t, agent, oracleStationMAC+row, tt.mac)
			wantString(t, agent.mib.Get(oracleStationIP+row), tt.address, "bsnMobileStationIpAddress")
			wantMAC(t, agent, oracleStationAPMAC+row, "00:11:22:33:44:55")
			wantInt(t, agent.mib.Get(oracleStationSlot+row), tt.slot, "bsnMobileStationAPIfSlotId")
			wantString(t, agent.mib.Get(oracleStationSSID+row), "corp-wifi", "bsnMobileStationSsid")
			wantInt(t, agent.mib.Get(oracleStationStatus+row), 3, "bsnMobileStationStatus associated(3)")
		})
	}
}

// TestControllerWithNoJoinedAPsServesNoAirespaceTree: a device nobody joined
// is not a controller to an NMS, so it must not answer the tree at all.
func TestControllerWithNoJoinedAPsServesNoAirespaceTree(t *testing.T) {
	agent := NewAgent(wirelessController(), 0)
	agent.SynthesizeWirelessController(nil)

	if next, _ := agent.mib.GetNext("1.3.6.1.4.1.14179"); strings.HasPrefix(
		strings.TrimPrefix(next, "."), "1.3.6.1.4.1.14179.") {
		t.Errorf("GETNEXT into AIRESPACE-WIRELESS-MIB = %s, want nothing under it", next)
	}
}

// TestControllerCaptureKeepsItsOwnAirespaceTree: a captured controller is the
// authority on which APs joined it, so authored APs do not overwrite it.
func TestControllerCaptureKeepsItsOwnAirespaceTree(t *testing.T) {
	device := wirelessController()
	device.SNMPConfig.WalkFile = writeWalk(t, ""+
		".1.3.6.1.2.1.1.1.0 = STRING: \"Cisco Controller\"\n"+
		".1.3.6.1.4.1.14179.2.2.1.1.3.0.23.223.161.15.208 = STRING: \"Cisco1252\"\n")
	agent := NewAgent(device, 0)
	if err := agent.LoadWalkFile(device.SNMPConfig.WalkFile); err != nil {
		t.Fatalf("LoadWalkFile() error = %v", err)
	}
	agent.SynthesizeWirelessController([]*config.Device{joinedAP()})

	if row := agent.mib.Get(oracleAPName + "." + oracleAPIndex); row != nil {
		t.Errorf("authored AP replaced the captured AP table: %s", oidValueString(row))
	}
	wantString(t, agent.mib.Get(oracleAPName+".0.23.223.161.15.208"), "Cisco1252", "captured bsnAPName")
}
