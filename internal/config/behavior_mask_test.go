package config_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestBehaviorMaskRoundTrip(t *testing.T) {
	devices := strings.Replace(addressFaultDevices,
		"    dhcp: {pool_start: 192.0.2.100, pool_end: 192.0.2.110}",
		"    interfaces: [{name: eth0, address: 192.0.2.1/24}]", 1)
	for _, prefix := range []string{"0", "16", "32"} {
		t.Run(prefix, func(t *testing.T) {
			cfg, err := config.LoadYAMLBytes([]byte(devices + addressFaultTimeline(
				"type: bad_mask\ninterface: eth0\nprefix_bits: "+prefix)))
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := config.MarshalConfigYAML(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(encoded), "prefix_bits: "+prefix) {
				t.Fatal("round trip lost explicit prefix")
			}
			if _, err = config.LoadYAMLBytes(encoded); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestBehaviorMaskRejectsInvalidTargets(t *testing.T) {
	for _, test := range []struct {
		name, kind, address, iface string
	}{
		{"router", "router", "192.0.2.1/24", "eth0"},
		{"routed switch", "layer3-switch", "192.0.2.1/24", "eth0"},
		{"firewall", "firewall", "192.0.2.1/24", "eth0"},
		{"IPv6", "server", "2001:db8::1/64", "eth0"},
		{"missing interface", "server", "192.0.2.1/24", "eth1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := "devices:\n  - name: server\n    type: " + test.kind +
				"\n    mac: '02:00:00:00:00:01'\n    interfaces: [{name: eth0, address: '" + test.address + "'}]\n"
			_, err := config.LoadYAMLBytes([]byte(body + addressFaultTimeline(
				"type: bad_mask\ninterface: "+test.iface+"\nprefix_bits: 16")))
			if err == nil {
				t.Fatal("accepted ineligible mask fault target")
			}
		})
	}
}
