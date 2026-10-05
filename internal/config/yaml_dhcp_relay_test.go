package config

import (
	"net"
	"strings"
	"testing"
)

const relayedDHCPYAML = `
networks:
  - name: data
    subnet: 10.1.10.0/24
  - name: wifi-corp
    subnet: 10.1.220.0/24
attachments:
  - name: tester
    connect: data
devices:
  - name: CORE-SW01
    type: router
    mac: "02:00:00:00:01:01"
    interfaces:
      - name: Vlan10
        network: data
        address: 10.1.10.1/24
      - name: Vlan220
        network: wifi-corp
        address: 10.1.220.1/24
        dhcp_relay: 10.1.10.5
  - name: DHCP01
    type: server
    mac: "02:00:00:00:01:05"
    interfaces:
      - name: eth0
        network: data
        address: 10.1.10.5/24
    dhcp:
      server_identifier: 10.1.10.5
      router: 10.1.10.1
      pool_start: 10.1.10.100
      pool_end: 10.1.10.199
      scopes:
        - pool_start: 10.1.220.100
          pool_end: 10.1.220.199
          router: 10.1.220.1
`

func TestRelayedDHCPSurvivesAYAMLRoundTrip(t *testing.T) {
	cfg, err := LoadYAMLBytes([]byte(relayedDHCPYAML))
	if err != nil {
		t.Fatalf("LoadYAMLBytes() error = %v", err)
	}
	assertRelayedDHCP(t, cfg)

	data, err := MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatalf("MarshalConfigYAML() error = %v", err)
	}
	roundTrip, err := LoadYAMLBytes(data)
	if err != nil {
		t.Fatalf("reload error = %v\n%s", err, data)
	}
	assertRelayedDHCP(t, roundTrip)
}

func assertRelayedDHCP(t *testing.T, cfg *Config) {
	t.Helper()
	if relay := cfg.Devices[0].Interfaces[1].DHCPRelay; relay != "10.1.10.5" {
		t.Errorf("dhcp_relay = %q, want 10.1.10.5", relay)
	}
	scopes := cfg.Devices[1].DHCPConfig.Scopes
	if len(scopes) != 1 || !scopes[0].PoolStart.Equal(net.ParseIP("10.1.220.100")) ||
		!scopes[0].PoolEnd.Equal(net.ParseIP("10.1.220.199")) ||
		!scopes[0].Router.Equal(net.ParseIP("10.1.220.1")) {
		t.Errorf("scopes = %#v, want the wifi-corp pool", scopes)
	}
}

func TestRelayedDHCPScopeRejectsAReversedPool(t *testing.T) {
	data := strings.Replace(relayedDHCPYAML, "pool_end: 10.1.220.199", "pool_end: 10.1.220.50", 1)
	if _, err := LoadYAMLBytes([]byte(data)); err == nil || !strings.Contains(err.Error(), "pool_end") {
		t.Fatalf("LoadYAMLBytes() error = %v, want a pool_end finding", err)
	}
}

func TestRelayedDHCPNeedsRoutedNetworks(t *testing.T) {
	cfg, err := LoadYAMLBytes([]byte(relayedDHCPYAML))
	if err != nil {
		t.Fatalf("LoadYAMLBytes() error = %v", err)
	}
	cfg.Networks, cfg.Attachments = nil, nil

	result := NewValidator("flat.yaml").Validate(cfg)
	for _, field := range []string{"devices[0].interfaces[1].dhcp_relay", "devices[1].dhcp.scopes"} {
		found := false
		for _, finding := range result.Errors {
			found = found || finding.Field == field
		}
		if !found {
			t.Errorf("errors = %#v, want one on %s", result.Errors, field)
		}
	}
}
