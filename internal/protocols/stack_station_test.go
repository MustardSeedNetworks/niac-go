package protocols

import (
	"errors"
	"net"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
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
		SNMPConfig:  config.SNMPConfig{Community: "public"},
		Interfaces:  []config.Interface{{Name: "Dot11Radio0", Type: "ieee80211"}},
		WiFiConfig: &config.WiFiConfig{Controller: "MED-WLC01", Radios: []config.WiFiRadio{{
			Interface: "Dot11Radio0", SSID: "corp-wifi", BSSID: bssid,
			Band: "5GHz", Channel: 36, TxPowerDBM: 17, Clients: clients,
		}}},
	}
}

// roamTestStack is a controller registered before the two APs that join it,
// so the controller's agent exists before either AP has state. edits change
// the first AP, the one the station starts on.
func roamTestStack(t *testing.T, edits ...func(*config.Device)) (*Stack, *config.Config) {
	t.Helper()
	first := roamTestAP(t, "MED-AP-01", "00:11:22:33:44:55", "00:11:22:33:44:56", []config.WiFiClient{{
		MAC: roamTestStation, IPAddress: "10.51.220.101",
		AssociatedSeconds: 60, SignalDBM: -60, SignalQualityPct: 80,
	}})
	for _, edit := range edits {
		edit(&first)
	}
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

	if err := stack.RoamStation(
		behavior.RoamAction{Station: roamTestStation, From: "MED-AP-01", To: "MED-AP-02"},
	); err != nil {
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

	err := stack.RoamStation(behavior.RoamAction{Station: roamTestStation, From: "MED-AP-02", To: "MED-AP-01"})

	if !errors.Is(err, devicestate.ErrStationNotFound) {
		t.Fatalf("RoamStation(from the wrong AP) error = %v, want ErrStationNotFound", err)
	}
	if after := versions(); after != before {
		t.Errorf("a refused roam changed state: versions %v -> %v", before, after)
	}
}

func lastEventKinds(store *devicestate.Store, count int) []devicestate.EventKind {
	events := store.Events()
	kinds := make([]devicestate.EventKind, 0, count)
	for _, event := range events[len(events)-count:] {
		kinds = append(kinds, event.Kind)
	}
	return kinds
}

func radioOperUp(t *testing.T, store *devicestate.Store, radio string) bool {
	t.Helper()
	for _, iface := range store.Snapshot().Network.Interfaces {
		if iface.Name == radio {
			return iface.OperUp
		}
	}
	t.Fatalf("no interface %s", radio)
	return false
}

// TestRoamCauseTakesTheOldRadioDownUntilTheStationReturns: the radio goes down
// before the station leaves it, which is the order a poller sees a radio
// failure and its clients moving in; the return brings the radio up before the
// station comes back to it.
func TestRoamCauseTakesTheOldRadioDownUntilTheStationReturns(t *testing.T) {
	stack, cfg := roamTestStack(t)
	from, to := stack.deviceStates[&cfg.Devices[1]], stack.deviceStates[&cfg.Devices[2]]
	away := behavior.RoamAction{
		Station: roamTestStation, From: "MED-AP-01", To: "MED-AP-02", Cause: devicestate.RoamCauseRadioDown,
	}

	if err := stack.RoamStation(away); err != nil {
		t.Fatalf("RoamStation(away) error = %v", err)
	}

	if radioOperUp(t, from, "Dot11Radio0") {
		t.Error("the radio the cause took down still reports up")
	}
	wantAway := []devicestate.EventKind{
		devicestate.EventFaultUpdated, devicestate.EventInterfaceUpdated, devicestate.EventStationRoamed,
	}
	if got := lastEventKinds(from, 3); !slices.Equal(got, wantAway) {
		t.Errorf("old AP events = %v, want %v", got, wantAway)
	}
	if station, _ := to.Station(roamTestStation); station.ReturnRadio != "Dot11Radio0" {
		t.Errorf("the new AP records return radio %q, want Dot11Radio0", station.ReturnRadio)
	}

	back := away
	back.From, back.To, back.Return = away.To, away.From, true
	if err := stack.RoamStation(back); err != nil {
		t.Fatalf("RoamStation(back) error = %v", err)
	}

	if !radioOperUp(t, from, "Dot11Radio0") || len(from.Snapshot().Faults) != 0 {
		t.Errorf("the return left the radio down: faults %+v", from.Snapshot().Faults)
	}
	wantBack := []devicestate.EventKind{
		devicestate.EventFaultCleared, devicestate.EventInterfaceUpdated, devicestate.EventStationAssociated,
	}
	if got := lastEventKinds(from, 3); !slices.Equal(got, wantBack) {
		t.Errorf("old AP events on the return = %v, want %v", got, wantBack)
	}
}

// TestReturnLandsOnTheRadioTheStationLeft: an AP serves the SSID on two radios
// and the station was on the second. The roam back goes to that one, not to
// the first radio that serves the SSID, or the cause would be undone on one
// radio and the station put on another.
func TestReturnLandsOnTheRadioTheStationLeft(t *testing.T) {
	stack, cfg := roamTestStack(t, func(first *config.Device) {
		first.Interfaces = append(first.Interfaces, config.Interface{Name: "Dot11Radio1", Type: "ieee80211"})
		second := first.WiFiConfig.Radios[0]
		second.Interface, second.BSSID, second.Band, second.Channel = "Dot11Radio1", "00:11:22:33:44:57", "2.4GHz", 6
		first.WiFiConfig.Radios[0].Clients = nil
		first.WiFiConfig.Radios = append(first.WiFiConfig.Radios, second)
	})
	from := stack.deviceStates[&cfg.Devices[1]]
	away := behavior.RoamAction{Station: roamTestStation, From: "MED-AP-01", To: "MED-AP-02"}
	back := behavior.RoamAction{Station: roamTestStation, From: "MED-AP-02", To: "MED-AP-01", Return: true}

	for _, roam := range []behavior.RoamAction{away, back} {
		if err := stack.RoamStation(roam); err != nil {
			t.Fatalf("RoamStation(%+v) error = %v", roam, err)
		}
	}

	if station, _ := from.Station(roamTestStation); station.Radio != "Dot11Radio1" {
		t.Errorf("the station returned to %q, want Dot11Radio1", station.Radio)
	}
}

// TestRefusedCauseLeavesTheStationWhereItWas: a cause the old AP cannot report,
// or a station the new AP already has, stops the roam before the cause takes
// the radio down or either AP changes.
func TestRefusedCauseLeavesTheStationWhereItWas(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		edit    func(*config.Device)
		onNewAP bool
		want    error
	}{
		{"no SNMP on the old AP", func(first *config.Device) { first.SNMPConfig = config.SNMPConfig{} }, false, ErrFaultUnobservable},
		{"already on the new AP", func(*config.Device) {}, true, devicestate.ErrStationAssociated},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			stack, cfg := roamTestStack(t, testCase.edit)
			from, to := stack.deviceStates[&cfg.Devices[1]], stack.deviceStates[&cfg.Devices[2]]
			if testCase.onNewAP {
				station, _ := from.Station(roamTestStation)
				if err := to.AssociateStation(station); err != nil {
					t.Fatal(err)
				}
			}
			before := [2]uint64{from.Version(), to.Version()}

			err := stack.RoamStation(behavior.RoamAction{
				Station: roamTestStation, From: "MED-AP-01", To: "MED-AP-02", Cause: devicestate.RoamCauseRadioDown,
			})

			if !errors.Is(err, testCase.want) {
				t.Fatalf("RoamStation() error = %v, want %v", err, testCase.want)
			}
			if after := [2]uint64{from.Version(), to.Version()}; after != before {
				t.Errorf("a refused cause changed state: versions %v -> %v", before, after)
			}
		})
	}
}
