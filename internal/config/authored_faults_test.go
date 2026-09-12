package config_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const authoredFaultYAML = `
devices:
  - name: EDGE-SW01
    type: switch
    vendor: cisco
    mac_suffix: 1
    ips: ["192.0.2.10"]
    snmp_agent:
      community: public
      sysname: EDGE-SW01
    dns:
      forward_records:
        - name: host.example
          ip: 192.0.2.20
    faults:
      - type: dns_nxdomain
        value: 1
    interfaces:
      - name: Gi0/1
        address: 192.0.2.10/24
        speed: 1000
        faults:
          - type: fcs_errors
            value: 25
          - type: link_down
`

func TestAuthoredFaultsLoadFromYAML(t *testing.T) {
	cfg, err := config.LoadYAMLBytes([]byte(authoredFaultYAML))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	device := cfg.Devices[0]
	if len(device.Faults) != 1 || device.Faults[0].Type != "dns_nxdomain" {
		t.Fatalf("device faults = %+v, want one dns_nxdomain", device.Faults)
	}
	faults := device.Interfaces[0].Faults
	if len(faults) != 2 {
		t.Fatalf("interface faults = %+v, want two", faults)
	}
	// link_down authors no value because a dead link has no magnitude; it
	// still has to reach the store as armed rather than as a zero that clears.
	if faults[1].Type != "link_down" || faults[1].Value != 1 {
		t.Errorf("link_down = %+v, want value 1", faults[1])
	}
}

// The per-type ceiling is the one rule the field tags cannot carry. Latency is
// milliseconds and reaches 60000 while every rate stops at 100, so the tag has
// to admit the larger of the two and a rate of 250 passes it. Only the catalog
// knows that dhcp_no_offer is a rate.
func TestAuthoredFaultRejectsAValueAboveItsOwnCeiling(t *testing.T) {
	overRate := strings.Replace(authoredFaultYAML,
		"      - type: dns_nxdomain\n        value: 1",
		"      - type: dhcp_no_offer\n        value: 250", 1)
	if overRate == authoredFaultYAML {
		t.Fatal("the fixture no longer contains the device fault this test raises")
	}

	_, err := config.LoadYAMLBytes([]byte(overRate))
	if !errors.Is(err, config.ErrAuthoredFaultValue) {
		t.Fatalf("error = %v, want ErrAuthoredFaultValue", err)
	}
}

func TestAuthoredOnOffFaultRefusesAMagnitude(t *testing.T) {
	withValue := strings.Replace(authoredFaultYAML,
		"- type: link_down", "- type: link_down\n            value: 50", 1)
	if withValue == authoredFaultYAML {
		t.Fatal("the fixture no longer contains the on/off fault this test loads")
	}

	if _, err := config.LoadYAMLBytes([]byte(withValue)); err == nil {
		t.Fatal("a value on link_down was accepted; it has no magnitude to carry")
	}
}
