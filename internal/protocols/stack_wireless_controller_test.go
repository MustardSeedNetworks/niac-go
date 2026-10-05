package protocols

import (
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// TestControllerReportsAnAPJoinedFromAnotherSegment: an agent knows only its
// own device, so the stack hands a controller the APs that name it. A
// controller reaches its APs over the routed network, so an AP on another VLAN
// is still its AP.
func TestControllerReportsAnAPJoinedFromAnotherSegment(t *testing.T) {
	cfg := &config.Config{Segments: []config.Segment{
		{Tag: 30, Devices: []config.Device{{
			Name: "MED-WLC01", Type: "server",
			MACAddress:  mustTestMAC(t, "00:1d:45:91:11:d0"),
			IPAddresses: []net.IP{net.ParseIP("10.51.30.21")},
			SNMPConfig:  config.SNMPConfig{Community: "public"},
		}}},
		{Tag: 220, Devices: []config.Device{{
			Name: "MED-AP-01", Type: "ap",
			MACAddress:  mustTestMAC(t, "00:11:22:33:44:55"),
			IPAddresses: []net.IP{net.ParseIP("10.51.220.11")},
			Interfaces:  []config.Interface{{Name: "Dot11Radio0", Type: "ieee80211"}},
			WiFiConfig: &config.WiFiConfig{Controller: "MED-WLC01", Radios: []config.WiFiRadio{{
				Interface: "Dot11Radio0", SSID: "corp-wifi", BSSID: "00:11:22:33:44:56",
				Band: "5GHz", Channel: 36, TxPowerDBM: 17,
				Clients: []config.WiFiClient{{
					MAC: "02:00:00:00:00:01", IPAddress: "10.51.220.101",
					AssociatedSeconds: 60, SignalDBM: -60, SignalQualityPct: 80,
				}},
			}}},
		}}},
	}}

	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	agent := stack.snmpAgents[&cfg.Segments[0].Devices[0]].baseAgent

	name, err := agent.HandleGet("1.3.6.1.4.1.14179.2.2.1.1.3.0.17.34.51.68.85")
	if err != nil || name.Value != "MED-AP-01" {
		t.Fatalf("bsnAPName = %v (%v), want MED-AP-01", name, err)
	}
	station, err := agent.HandleGet("1.3.6.1.4.1.14179.2.1.4.1.4.2.0.0.0.0.1")
	if err != nil {
		t.Fatalf("get bsnMobileStationAPMacAddr: %v", err)
	}
	if got, ok := station.Value.([]byte); !ok || net.HardwareAddr(got).String() != "00:11:22:33:44:55" {
		t.Fatalf("bsnMobileStationAPMacAddr = %v, want the AP's MAC", station.Value)
	}
}
