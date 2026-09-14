package config

import (
	"strings"
	"testing"
)

const poolScenario = `networks:
  - name: med-data
    subnet: 10.51.210.0/24
    virtual_vlan: 210
attachments:
  - name: cyberscope
    at:
      device: MED-ACC-SW01
      ports:
        - GigabitEthernet1/0/20
        - GigabitEthernet1/0/21
    pins:
      - mac: "00:c0:17:aa:bb:cc"
        device: MED-ACC-SW01
        interface: GigabitEthernet1/0/21
devices:
  - name: MED-ACC-SW01
    type: switch
    mac: 02:00:00:00:00:01
    interfaces:
      - name: Vlan210
        network: med-data
        address: 10.51.210.21/24
      - name: GigabitEthernet1/0/20
        vlans: [210]
      - name: GigabitEthernet1/0/21
        vlans: [210]
`

func TestLoadYAMLReadsAnAttachmentPool(t *testing.T) {
	cfg, err := LoadYAMLBytes([]byte(poolScenario))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Attachments) != 1 {
		t.Fatalf("attachments = %#v", cfg.Attachments)
	}
	attachment := cfg.Attachments[0]
	if attachment.Network != "" {
		t.Errorf("network = %q, want empty for a port-scoped attachment", attachment.Network)
	}
	if attachment.At == nil || attachment.At.Device != "MED-ACC-SW01" ||
		len(attachment.At.Ports) != 2 {
		t.Fatalf("at = %#v", attachment.At)
	}
	if len(attachment.Pins) != 1 || attachment.Pins[0].Interface != "GigabitEthernet1/0/21" {
		t.Fatalf("pins = %#v", attachment.Pins)
	}
}

// A scenario the daemon loads is a scenario the editor has to be able to write
// back. Dropping the pool on the way out would silently move every tester back
// to wherever the network-scoped form put it.
func TestAttachmentPoolSurvivesAYAMLRoundTrip(t *testing.T) {
	cfg, err := LoadYAMLBytes([]byte(poolScenario))
	if err != nil {
		t.Fatal(err)
	}
	written, err := MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"device: MED-ACC-SW01",
		"GigabitEthernet1/0/20",
		"00:c0:17:aa:bb:cc",
	} {
		if !strings.Contains(string(written), want) {
			t.Errorf("round-tripped YAML lost %q:\n%s", want, written)
		}
	}
	reloaded, err := LoadYAMLBytes(written)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Attachments[0].At == nil ||
		len(reloaded.Attachments[0].At.Ports) != 2 ||
		len(reloaded.Attachments[0].Pins) != 1 {
		t.Fatalf("reloaded attachment = %#v", reloaded.Attachments[0])
	}
}

// The network-scoped form is what every scenario on disk uses today.
func TestNetworkAttachmentStillRoundTrips(t *testing.T) {
	cfg, err := LoadYAMLBytes([]byte(`networks:
  - name: lab-transit
    subnet: 10.254.200.0/24
attachments:
  - name: cyberscope
    connect: lab-transit
devices:
  - name: LAB-EDGE-R1
    type: router
    mac: 02:00:00:00:00:01
    interfaces:
      - name: outside
        network: lab-transit
        address: 10.254.200.1/24
`))
	if err != nil {
		t.Fatal(err)
	}
	written, err := MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), "connect: lab-transit") {
		t.Fatalf("round-trip lost connect:\n%s", written)
	}
	if strings.Contains(string(written), "at:") {
		t.Fatalf("round-trip invented a pool:\n%s", written)
	}
}
