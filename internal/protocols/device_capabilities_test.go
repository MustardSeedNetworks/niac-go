package protocols

import (
	"encoding/binary"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// TestDeviceTypeCapabilities pins what each device type announces.
//
// Both capability TLVs switch on the device type and fall to a default of
// station-only / host. That default is correct for an end station and wrong
// for anything that forwards, so a forwarding type landing in it is a silent
// wire-fidelity defect rather than a build failure — `firewall` sat there and
// told every discovery tool it was a host (#2096).
//
// Station-only is IEEE 802.1AB's own term for an end station: the bit is set
// if and only if no other capability is. So a server, host, workstation, IoT
// device or printer announcing it is right, and is asserted here as
// deliberate rather than left to look like an oversight.
func TestDeviceTypeCapabilities(t *testing.T) {
	tests := []struct {
		deviceType  string
		lldpEnabled uint16
		cdp         uint32
	}{
		{"router", LLDPCapRouter, CDPCapRouter | CDPCapIGMPCapable},
		{"layer3-switch", LLDPCapRouter | LLDPCapBridge, CDPCapRouter | CDPCapSwitch | CDPCapIGMPCapable},
		{"switch", LLDPCapBridge, CDPCapSwitch | CDPCapIGMPCapable},
		{"ap", LLDPCapWLANAP, CDPCapSwitch | CDPCapIGMPCapable},
		{"access-point", LLDPCapWLANAP, CDPCapSwitch | CDPCapIGMPCapable},
		{"firewall", LLDPCapRouter, CDPCapRouter},
		{"voip-phone", LLDPCapTelephone, CDPCapPhone | CDPCapHost},

		// End stations, deliberately.
		{"server", LLDPCapStationOnly, CDPCapHost},
		{"host", LLDPCapStationOnly, CDPCapHost},
		{"workstation", LLDPCapStationOnly, CDPCapHost},
		{"iot", LLDPCapStationOnly, CDPCapHost},
		{"printer", LLDPCapStationOnly, CDPCapHost},
	}

	for _, tc := range tests {
		t.Run(tc.deviceType, func(t *testing.T) {
			device := &config.Device{Name: "d1", Type: tc.deviceType}

			lldp := (&LLDPHandler{}).buildSystemCapabilitiesTLV(device)
			if enabled := binary.BigEndian.Uint16(lldp[4:6]); enabled != tc.lldpEnabled {
				t.Errorf("LLDP enabled capabilities = 0x%04x, want 0x%04x", enabled, tc.lldpEnabled)
			}

			cdp := (&CDPHandler{}).buildCapabilitiesTLV(device)
			if caps := binary.BigEndian.Uint32(cdp[4:8]); caps != tc.cdp {
				t.Errorf("CDP capabilities = 0x%08x, want 0x%08x", caps, tc.cdp)
			}
		})
	}
}

// TestForwardingTypesAreNotStationOnly states the rule the table above
// encodes, so a new forwarding type added without a case is caught by the rule
// rather than by whoever notices the diagram looks wrong.
func TestForwardingTypesAreNotStationOnly(t *testing.T) {
	for _, deviceType := range []string{"router", "layer3-switch", "switch", "ap", "access-point", "firewall"} {
		device := &config.Device{Name: "d1", Type: deviceType}
		lldp := (&LLDPHandler{}).buildSystemCapabilitiesTLV(device)

		if binary.BigEndian.Uint16(lldp[2:4]) == LLDPCapStationOnly {
			t.Errorf("%s advertises station-only; it forwards", deviceType)
		}
		if binary.BigEndian.Uint32((&CDPHandler{}).buildCapabilitiesTLV(device)[4:8]) == CDPCapHost {
			t.Errorf("%s advertises CDP host; it forwards", deviceType)
		}
	}
}
