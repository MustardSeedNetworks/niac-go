package protocols

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// TestDiscoveryEnablementMatchesEmitters pins the predicate that decides who
// owns an inbound neighbour to the rules the emitters actually follow.
//
// The two drifted apart. The emitters were tightened so a device only
// announces a vendor protocol it was configured for — the change that closed
// the "every device emits CDP" bug on the Neighbors page — and this predicate
// was left reading an absent config block as enabled. selectDiscoveryDevice
// returns the first device the predicate accepts, so an inbound CDP neighbour
// was attributed to a workstation that never sends CDP rather than to the
// switch that does.
func TestDiscoveryEnablementMatchesEmitters(t *testing.T) {
	stack := &Stack{}

	tests := []struct {
		name   string
		device config.Device
		proto  string
		want   bool
	}{
		// Vendor protocols opt in. An absent block is not consent.
		{"cdp absent on a workstation", config.Device{Type: "workstation"}, ProtocolCDP, false},
		{"cdp absent on a switch", config.Device{Type: "switch"}, ProtocolCDP, false},
		{
			"cdp enabled", config.Device{
				Type: "switch", CDPConfig: &config.CDPConfig{Enabled: true},
			}, ProtocolCDP, true,
		},
		{
			"cdp present but off", config.Device{
				Type: "switch", CDPConfig: &config.CDPConfig{Enabled: false},
			}, ProtocolCDP, false,
		},
		{"edp absent", config.Device{Type: "switch"}, ProtocolEDP, false},
		{"fdp absent", config.Device{Type: "switch"}, ProtocolFDP, false},

		// LLDP is the IEEE standard and ships on for infrastructure.
		{"lldp absent on a switch", config.Device{Type: "switch"}, ProtocolLLDP, true},
		{"lldp absent on a router", config.Device{Type: "router"}, ProtocolLLDP, true},
		{"lldp absent on a firewall", config.Device{Type: "firewall"}, ProtocolLLDP, true},
		{"lldp absent on a phone", config.Device{Type: "voip-phone"}, ProtocolLLDP, true},
		{"lldp absent on a workstation", config.Device{Type: "workstation"}, ProtocolLLDP, false},
		{"lldp absent on a server", config.Device{Type: "server"}, ProtocolLLDP, false},
		{
			"lldp turned off on a switch", config.Device{
				Type: "switch", LLDPConfig: &config.LLDPConfig{Enabled: false},
			}, ProtocolLLDP, false,
		},
		{
			"lldp turned on for a server", config.Device{
				Type: "server", LLDPConfig: &config.LLDPConfig{Enabled: true},
			}, ProtocolLLDP, true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := tt.device
			if got := stack.isDeviceEnabledForProtocol(&device, tt.proto); got != tt.want {
				t.Errorf("isDeviceEnabledForProtocol(%s) = %v, want %v", tt.proto, got, tt.want)
			}
		})
	}
}

// The defect in one assertion: a device that does not emit a protocol must not
// be chosen to own a neighbour discovered over it.
func TestSilentDeviceDoesNotOwnANeighbour(t *testing.T) {
	stack := &Stack{}
	silent := config.Device{Name: "WS-01", Type: "workstation"}
	speaking := config.Device{
		Name: "SW-01", Type: "switch", CDPConfig: &config.CDPConfig{Enabled: true},
	}

	if stack.isDeviceEnabledForProtocol(&silent, ProtocolCDP) {
		t.Error("a workstation with no CDP config was offered as a CDP neighbour owner")
	}
	if !stack.isDeviceEnabledForProtocol(&speaking, ProtocolCDP) {
		t.Error("a switch running CDP was not offered as a CDP neighbour owner")
	}
}
