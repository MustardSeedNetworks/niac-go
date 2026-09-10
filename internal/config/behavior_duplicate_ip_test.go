package config_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestDuplicateIPRequiresExplicitInterfaceAndPeer(t *testing.T) {
	devices := strings.Replace(addressFaultDevices,
		"    dhcp: {pool_start: 192.0.2.100, pool_end: 192.0.2.110}",
		"    interfaces: [{name: eth0, address: 192.0.2.1/24}]", 1)
	for _, tc := range []struct {
		name, payload string
		valid         bool
	}{
		{"valid", "type: duplicate_ip\ninterface: eth0\naddress: 192.0.2.20", true},
		{"missing interface", "type: duplicate_ip\naddress: 192.0.2.20", false},
		{"unknown interface", "type: duplicate_ip\ninterface: eth1\naddress: 192.0.2.20", false},
		{"no peer", "type: duplicate_ip\ninterface: eth0\naddress: 192.0.2.30", false},
		{"self", "type: duplicate_ip\ninterface: eth0\naddress: 192.0.2.1", false},
		{"numeric", "type: duplicate_ip\ninterface: eth0\naddress: 192.0.2.20\nvalue: 1", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.LoadYAMLBytes([]byte(devices + addressFaultTimeline(tc.payload)))
			if (err == nil) != tc.valid {
				t.Fatalf("accepted=%v, want %v: %v", err == nil, tc.valid, err)
			}
			if err != nil {
				return
			}
			encoded, err := config.MarshalConfigYAML(cfg)
			if err != nil {
				t.Fatal(err)
			}
			reloaded, err := config.LoadYAMLBytes(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(cfg.Devices, reloaded.Devices) ||
				!reflect.DeepEqual(cfg.BehaviorTimelines, reloaded.BehaviorTimelines) {
				t.Fatal("round trip changed canonical inventory or fault")
			}
		})
	}
}

func TestDuplicateIPHonorsSelectedInterfaceNetwork(t *testing.T) {
	body := `networks:
  - {name: access, subnet: 192.0.2.0/24, virtual_vlan: 200}
  - {name: remote, subnet: 198.51.100.0/24, virtual_vlan: 300}
attachments:
  - {name: tester, connect: access}
devices:
  - name: server
    mac: '02:00:00:00:00:01'
    interfaces:
      - {name: eth0, network: access, address: 192.0.2.1/24}
      - {name: eth1, network: remote, address: 198.51.100.1/24}
  - name: peer
    mac: '02:00:00:00:00:02'
    interfaces: [{name: eth0, network: remote, address: 198.51.100.20/24}]
`
	for _, iface := range []string{"eth0", "eth1"} {
		_, err := config.LoadYAMLBytes([]byte(body + addressFaultTimeline(
			"type: duplicate_ip\ninterface: "+iface+"\naddress: 198.51.100.20")))
		if iface == "eth0" && err == nil {
			t.Fatal("accepted peer on another target interface network")
		}
		if iface == "eth1" && err != nil {
			t.Fatal(err)
		}
	}
}

func TestDuplicateIPRejectsCrossSegmentPeer(t *testing.T) {
	body := `segments:
  - tag: 200
    devices:
      - name: server
        mac: '02:00:00:00:00:01'
        interfaces: [{name: eth0, address: 192.0.2.1/24}]
  - tag: 300
    devices:
      - name: peer
        mac: '02:00:00:00:00:02'
        interfaces: [{name: eth0, address: 192.0.2.20/24}]
`
	_, err := config.LoadYAMLBytes([]byte(body + addressFaultTimeline(
		"type: duplicate_ip\ninterface: eth0\naddress: 192.0.2.20")))
	if err == nil {
		t.Fatal("accepted peer on another segment")
	}
}
