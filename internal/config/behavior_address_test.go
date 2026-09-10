package config_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const addressFaultDevices = `devices:
  - name: server
    mac: '02:00:00:00:00:01'
    ips: [192.0.2.1]
    dhcp: {pool_start: 192.0.2.100, pool_end: 192.0.2.110}
  - name: peer
    mac: '02:00:00:00:00:02'
    ips: [192.0.2.20]
`

func addressFaultTimeline(payload string) string {
	return `behavior_timelines:
  - name: conflict
    repeat_count: 1
    phases:
      - name: offer
        duration_ms: 10
        reset: true
        faults:
          - device: server
            ` + strings.ReplaceAll(payload, "\n", "\n            ") + "\n"
}

func TestAddressFaultTimelineRoundTrip(t *testing.T) {
	cfg, err := config.LoadYAMLBytes(
		[]byte(addressFaultDevices + addressFaultTimeline("type: duplicate_dhcp_offer\naddress: 192.0.2.20")),
	)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "address: 192.0.2.20") || strings.Contains(string(encoded), "value:") {
		t.Fatalf("address payload changed: %s", encoded)
	}
	reloaded, err := config.LoadYAMLBytes(encoded)
	if err != nil || !reflect.DeepEqual(cfg.BehaviorTimelines, reloaded.BehaviorTimelines) {
		t.Fatalf("address timeline changed: %v", err)
	}
	if !reflect.DeepEqual(cfg.Devices, reloaded.Devices) {
		t.Fatal("fault round trip changed authored devices")
	}
}

func TestAddressFaultPayloadValidation(t *testing.T) {
	for _, payload := range []string{
		"type: duplicate_dhcp_offer", "type: duplicate_dhcp_offer\nvalue: 0",
		"type: duplicate_dhcp_offer\naddress: 192.0.2.20\nvalue: 0",
		"type: duplicate_dhcp_offer\naddress: 192.0.2.20\nvalue: 1",
		"type: duplicate_dhcp_offer\naddress: ''", "type: duplicate_dhcp_offer\naddress: invalid",
		"type: duplicate_dhcp_offer\naddress: '2001:db8::1'",
		"type: duplicate_dhcp_offer\naddress: 224.0.0.1",
		"type: duplicate_dhcp_offer\naddress: 255.255.255.255",
		"type: duplicate_dhcp_offer\naddress: 0.0.0.0",
		"type: duplicate_dhcp_offer\naddress: 192.0.2.20\ninterface: eth0",
		"type: latency", "type: latency\nvalue: 0",
		"type: latency\nvalue: 10\naddress: 192.0.2.20",
		"type: latency\nvalue: 10\naddress: ''",
	} {
		t.Run(payload, func(t *testing.T) {
			_, err := config.LoadYAMLBytes([]byte(addressFaultDevices + addressFaultTimeline(payload)))
			if err == nil || !strings.Contains(err.Error(), "fault") {
				t.Fatalf("expected fault validation error, got %v", err)
			}
		})
	}
}

func TestAddressFaultTargetValidation(t *testing.T) {
	for _, test := range []struct{ name, devices, address string }{
		{"missing target", strings.ReplaceAll(addressFaultDevices, "name: server", "name: other"), "192.0.2.20"},
		{"no DHCP", strings.ReplaceAll(addressFaultDevices, "    dhcp: {pool_start: 192.0.2.100, pool_end: 192.0.2.110}\n", ""), "192.0.2.20"},
		{"self", addressFaultDevices, "192.0.2.1"},
		{"unowned", addressFaultDevices, "192.0.2.30"},
		{"cross VLAN", strings.ReplaceAll(addressFaultDevices, "  - name: peer", "  - vlan: 200\n    name: peer"), "192.0.2.20"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := config.LoadYAMLBytes(
				[]byte(test.devices + addressFaultTimeline("type: duplicate_dhcp_offer\naddress: "+test.address)),
			)
			if err == nil || !strings.Contains(err.Error(), "fault") {
				t.Fatalf("expected target error, got %v", err)
			}
		})
	}
}
