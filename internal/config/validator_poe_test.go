package config

import (
	"net"
	"strings"
	"testing"
)

const poeSwitchName = "closet-1"

func poeDevice(budget int) Device {
	return Device{
		Name:        poeSwitchName,
		Type:        "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		PoEConfig:   &PoEConfig{BudgetWatts: budget},
	}
}

func poweredDevice(name string, tenthWatts int) Device {
	return Device{
		Name:        name,
		Type:        "voip-phone",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x66},
		IPAddresses: []net.IP{net.ParseIP("192.168.1.50")},
		LLDPConfig: &LLDPConfig{MED: &LLDPMEDConfig{
			Power: &LLDPMEDPower{DeviceType: "pd", ValueTenthWatts: tenthWatts},
		}},
	}
}

func errorFor(t *testing.T, cfg *Config, field string) string {
	t.Helper()
	for _, err := range NewValidator("test.yaml").Validate(cfg).Errors {
		if strings.HasSuffix(err.Field, field) {
			return err.Message
		}
	}

	return ""
}

func TestValidatePoEBudgetBounds(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		budget    int
		wantError bool
	}{
		{"unset budget on a declared PSE", 0, true},
		{"below one class-1 port", 3, true},
		{"a 48-port access switch", 370, false},
		{"a fully loaded 802.3bt chassis", 4320, false},
		{"beyond any single chassis", 4321, true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := &Config{Devices: []Device{poeDevice(testCase.budget)}}

			message := errorFor(t, cfg, "poe.budget_watts")
			if testCase.wantError && message == "" {
				t.Errorf("budget %d accepted, want rejected", testCase.budget)
			}
			if !testCase.wantError && message != "" {
				t.Errorf("budget %d rejected: %s", testCase.budget, message)
			}
		})
	}
}

func TestValidatePoEUsageThresholdBounds(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		threshold int
		wantError bool
	}{
		{"unset takes the default", 0, false},
		{"zero percent is not authorable", -1, true},
		{"a conservative closet", 65, false},
		{"a full budget is not a threshold", 100, true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			device := poeDevice(370)
			device.PoEConfig.UsageThresholdPercent = testCase.threshold
			cfg := &Config{Devices: []Device{device}}

			message := errorFor(t, cfg, "poe.usage_threshold_percent")
			if testCase.wantError && message == "" {
				t.Errorf("threshold %d accepted, want rejected", testCase.threshold)
			}
			if !testCase.wantError && message != "" {
				t.Errorf("threshold %d rejected: %s", testCase.threshold, message)
			}
		})
	}
}

// TestValidatePoEOverSubscription: a real switch answers an over-budget port
// with a power-denied counter instead of powering it, so an over-subscribed pack
// replays a device the author meant to be up as one that never comes up. Naming
// it at validate time is the difference between an authoring error and a tester
// filing a bug against the simulation.
func TestValidatePoEOverSubscription(t *testing.T) {
	switchDevice := poeDevice(12)
	switchDevice.TrunkPorts = []TrunkPort{
		{Interface: "Gi1/0/1", RemoteDevice: "phone-1"},
		{Interface: "Gi1/0/2", RemoteDevice: "phone-2"},
	}
	cfg := &Config{Devices: []Device{
		switchDevice,
		poweredDevice("phone-1", 62),
		poweredDevice("phone-2", 62),
	}}

	message := errorFor(t, cfg, "poe.budget_watts")
	if message == "" {
		t.Fatal("12 W budget with 12.4 W attached accepted, want rejected")
	}
	if !strings.Contains(message, "12.4 W") {
		t.Errorf("message = %q, want the attached draw named to a tenth of a watt", message)
	}
}

// A budget that exactly covers the attached devices is correct engineering, not
// an error: the gate must not fire on the equality boundary.
func TestValidatePoEBudgetExactlyCoveringIsValid(t *testing.T) {
	switchDevice := poeDevice(13)
	switchDevice.TrunkPorts = []TrunkPort{{Interface: "Gi1/0/1", RemoteDevice: "phone-1"}}
	cfg := &Config{Devices: []Device{switchDevice, poweredDevice("phone-1", 130)}}

	if message := errorFor(t, cfg, "poe.budget_watts"); message != "" {
		t.Errorf("13 W budget with 13.0 W attached rejected: %s", message)
	}
}

// The PSE side is authored on the switch; the draw is authored once on the
// powered device. A device that advertises power as a PSE rather than a PD is
// not drawing anything and must not count against a budget.
func TestPoEDrawTenthWattsOnlyCountsPoweredDevices(t *testing.T) {
	pse := poweredDevice("switch-1", 300)
	pse.LLDPConfig.MED.Power.DeviceType = "pse"

	if draw := PoEDrawTenthWatts(&pse); draw != 0 {
		t.Errorf("PoEDrawTenthWatts(pse) = %d, want 0", draw)
	}
	pd := poweredDevice("phone-1", 62)
	if draw := PoEDrawTenthWatts(&pd); draw != 62 {
		t.Errorf("PoEDrawTenthWatts(pd) = %d, want 62", draw)
	}
	if draw := PoEDrawTenthWatts(nil); draw != 0 {
		t.Errorf("PoEDrawTenthWatts(nil) = %d, want 0", draw)
	}
}

func TestValidatePoEOverSubscriptionAcrossSegments(t *testing.T) {
	switchDevice := poeDevice(5)
	switchDevice.TrunkPorts = []TrunkPort{{Interface: "Gi1/0/1", RemoteDevice: "camera-1"}}
	cfg := &Config{Segments: []Segment{
		{Tag: 10, Devices: []Device{switchDevice}},
		{Tag: 20, Devices: []Device{poweredDevice("camera-1", 128)}},
	}}

	if message := errorFor(t, cfg, "poe.budget_watts"); message == "" {
		t.Fatal("over-subscription across segments accepted, want rejected")
	}
}
