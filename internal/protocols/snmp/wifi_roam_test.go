package snmp

import (
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// stationState is a device state seeded with the device's authored clients,
// as the stack seeds it.
func stationState(t *testing.T, device *config.Device) *devicestate.Store {
	t.Helper()
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	if err := state.InstallStations(config.AuthoredStations(device, time.Now())); err != nil {
		t.Fatalf("InstallStations() error = %v", err)
	}

	return state
}

// TestDot11ClientTableFollowsWhoIsAssociated: the association table is who is
// on the AP now, not who was authored on it, so a station that roams away
// leaves the table and one that roams in joins it under the radio it took.
func TestDot11ClientTableFollowsWhoIsAssociated(t *testing.T) {
	device := wifiAPWithClients()
	state := stationState(t, device)
	agent := NewAgentWithState(device, state, AgentOptions{Community: "public"})
	away := clientIndex(t, agent, "Dot11Radio0", oracleFirstClientMAC)

	roamed, err := state.RoamStation("02:c0:17:a4:03:6b")
	if err != nil {
		t.Fatalf("RoamStation() error = %v", err)
	}
	roamed.Radio = "Dot11Radio1"
	roamed.MAC = "02:00:00:00:00:09"
	roamed.AssociatedAt = time.Now()
	if err = state.AssociateStation(roamed); err != nil {
		t.Fatalf("AssociateStation() error = %v", err)
	}
	agent.syncDeviceStateMIBs()

	if row := agent.mib.Get(oracleClientSignal + "." + away); row != nil {
		t.Errorf("the station that roamed away is still reported: %s", oidValueString(row))
	}
	arrived := clientIndex(t, agent, "Dot11Radio1", "2.0.0.0.0.9")
	wantInt(t, agent.mib.Get(oracleClientSignal+"."+arrived), -55, "cDot11ClientSignalStrength of the arrival")
	wantInt(t, agent.mib.Get(oracleClientUpTime+"."+arrived), 0, "cDot11ClientUpTime restarts on association")
	wantMAC(t, agent, oracleClientParentAddress+"."+arrived, "00:0b:fd:d4:6f:de")
}

func secondJoinedAP() *config.Device {
	device := joinedAP()
	device.Name = "MED-AP-02"
	device.MACAddress = net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x66}
	for index := range device.WiFiConfig.Radios {
		device.WiFiConfig.Radios[index].Clients = nil
	}

	return device
}

// TestControllerFollowsAStationBetweenJoinedAPs: the station table is keyed by
// the station, so a roam moves its row's AP and slot columns to the AP it is
// on now -- the change an NMS polling a controller sees a roam as.
func TestControllerFollowsAStationBetweenJoinedAPs(t *testing.T) {
	first, second := joinedAP(), secondJoinedAP()
	states := map[*config.Device]*devicestate.Store{first: stationState(t, first), second: stationState(t, second)}
	agent := NewAgent(wirelessController(), 0)
	agent.SynthesizeWirelessController([]*config.Device{first, second})
	agent.BindJoinedAccessPointStates(func(device *config.Device) *devicestate.Store { return states[device] })
	row := "." + oracleFirstClientMAC
	wantMAC(t, agent, oracleStationAPMAC+row, "00:11:22:33:44:55")

	station, err := states[first].RoamStation("02:c0:17:a4:03:6b")
	if err != nil {
		t.Fatalf("RoamStation() error = %v", err)
	}
	station.Radio = "Dot11Radio1"
	if err = states[second].AssociateStation(station); err != nil {
		t.Fatalf("AssociateStation() error = %v", err)
	}
	agent.syncDeviceStateMIBs()

	wantMAC(t, agent, oracleStationAPMAC+row, "00:11:22:33:44:66")
	wantInt(t, agent.mib.Get(oracleStationSlot+row), 1, "bsnMobileStationAPIfSlotId of the new radio")
	wantString(t, agent.mib.Get(oracleStationIP+row), "10.250.8.219", "bsnMobileStationIpAddress")
}

// TestControllerReportsAuthoredStationsBeforeStateIsBound: the stack binds the
// APs' state once every device has one, and a walk before that still sees the
// authored clients rather than an empty table.
func TestControllerReportsAuthoredStationsBeforeStateIsBound(t *testing.T) {
	agent := NewAgent(wirelessController(), 0)
	agent.SynthesizeWirelessController([]*config.Device{joinedAP()})
	agent.syncDeviceStateMIBs()

	wantMAC(t, agent, oracleStationAPMAC+"."+oracleFirstClientMAC, "00:11:22:33:44:55")
	if _, err := netip.ParseAddr(
		oidValueString(agent.mib.Get(oracleStationIP + "." + oracleFirstClientMAC)),
	); err != nil {
		t.Errorf("bsnMobileStationIpAddress = %v", err)
	}
}
