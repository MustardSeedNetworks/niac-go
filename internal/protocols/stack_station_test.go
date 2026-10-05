package protocols

import (
	"errors"
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

const (
	roamTestStation = "02:00:00:00:00:01"
	// bsnMobileStationAPMacAddr of roamTestStation.
	roamTestStationAP = "1.3.6.1.4.1.14179.2.1.4.1.4.2.0.0.0.0.1"
)

func roamTestAP(t *testing.T, name, mac, bssid string, clients []config.WiFiClient) config.Device {
	t.Helper()
	return config.Device{
		Name: name, Type: "ap",
		MACAddress:  mustTestMAC(t, mac),
		IPAddresses: []net.IP{net.ParseIP("10.51.220.11")},
		Interfaces:  []config.Interface{{Name: "Dot11Radio0", Type: "ieee80211"}},
		WiFiConfig: &config.WiFiConfig{Controller: "MED-WLC01", Radios: []config.WiFiRadio{{
			Interface: "Dot11Radio0", SSID: "corp-wifi", BSSID: bssid,
			Band: "5GHz", Channel: 36, TxPowerDBM: 17, Clients: clients,
		}}},
	}
}

// roamTestStack is a controller registered before the two APs that join it,
// so the controller's agent exists before either AP has state.
func roamTestStack(t *testing.T) (*Stack, *config.Config) {
	t.Helper()
	first := roamTestAP(t, "MED-AP-01", "00:11:22:33:44:55", "00:11:22:33:44:56", []config.WiFiClient{{
		MAC: roamTestStation, IPAddress: "10.51.220.101",
		AssociatedSeconds: 60, SignalDBM: -60, SignalQualityPct: 80,
	}})
	second := roamTestAP(t, "MED-AP-02", "00:11:22:33:44:65", "00:11:22:33:44:66", nil)
	second.IPAddresses = []net.IP{net.ParseIP("10.51.220.12")}
	cfg := &config.Config{Devices: []config.Device{
		{
			Name: "MED-WLC01", Type: "server",
			MACAddress:  mustTestMAC(t, "00:1d:45:91:11:d0"),
			IPAddresses: []net.IP{net.ParseIP("10.51.220.21")},
			SNMPConfig:  config.SNMPConfig{Community: "public"},
		},
		first, second,
	}}

	return NewStack(nil, cfg, logging.NewDebugConfig(0)), cfg
}

func stationAPOnController(t *testing.T, stack *Stack, cfg *config.Config) string {
	t.Helper()
	value, err := stack.snmpAgents[&cfg.Devices[0]].baseAgent.HandleGet(roamTestStationAP)
	if err != nil {
		t.Fatalf("get bsnMobileStationAPMacAddr: %v", err)
	}
	mac, ok := value.Value.([]byte)
	if !ok {
		t.Fatalf("bsnMobileStationAPMacAddr = %v", value.Value)
	}

	return net.HardwareAddr(mac).String()
}

// TestRoamMovesTheStationOnBothAPsAndTheController is the roam end to end: the
// station leaves the old AP's state, joins the new one's, each records its half,
// and the controller reports it on the new AP.
func TestRoamMovesTheStationOnBothAPsAndTheController(t *testing.T) {
	stack, cfg := roamTestStack(t)
	if got := stationAPOnController(t, stack, cfg); got != "00:11:22:33:44:55" {
		t.Fatalf("before the roam the controller reports the station on %s", got)
	}

	if err := stack.RoamStation(roamTestStation, "MED-AP-01", "MED-AP-02"); err != nil {
		t.Fatalf("RoamStation() error = %v", err)
	}

	from, to := stack.deviceStates[&cfg.Devices[1]], stack.deviceStates[&cfg.Devices[2]]
	if _, still := from.Station(roamTestStation); still {
		t.Error("the old AP still has the station")
	}
	if station, arrived := to.Station(roamTestStation); !arrived || station.Radio != "Dot11Radio0" {
		t.Errorf("the new AP has %+v (%v)", station, arrived)
	}
	for _, testCase := range []struct {
		state *devicestate.Store
		kind  devicestate.EventKind
	}{{from, devicestate.EventStationRoamed}, {to, devicestate.EventStationAssociated}} {
		events := testCase.state.Events()
		if last := events[len(events)-1]; last.Kind != testCase.kind || last.Target != roamTestStation {
			t.Errorf("last event = %s %q, want %s", last.Kind, last.Target, testCase.kind)
		}
	}
	if got := stationAPOnController(t, stack, cfg); got != "00:11:22:33:44:65" {
		t.Errorf("after the roam the controller reports the station on %s, want MED-AP-02", got)
	}
}

// TestRefusedRoamLeavesTheStationWhereItWas: a timeline that has lost track of
// where a station is fails, and changes nothing on either AP.
func TestRefusedRoamLeavesTheStationWhereItWas(t *testing.T) {
	stack, cfg := roamTestStack(t)
	versions := func() [2]uint64 {
		return [2]uint64{
			stack.deviceStates[&cfg.Devices[1]].Version(), stack.deviceStates[&cfg.Devices[2]].Version(),
		}
	}
	before := versions()

	err := stack.RoamStation(roamTestStation, "MED-AP-02", "MED-AP-01")

	if !errors.Is(err, devicestate.ErrStationNotFound) {
		t.Fatalf("RoamStation(from the wrong AP) error = %v, want ErrStationNotFound", err)
	}
	if after := versions(); after != before {
		t.Errorf("a refused roam changed state: versions %v -> %v", before, after)
	}
}
