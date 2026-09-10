package config_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestAddressFaultRequiresEffectiveFlatHost(t *testing.T) {
	for _, tc := range []struct {
		prefix, address string
		want            bool
	}{
		{"192.0.2.20/24", "192.0.2.20", true},
		{"192.0.2.30/24", "192.0.2.20", false},
		{"192.0.2.0/24", "192.0.2.0", false},
		{"192.0.2.255/24", "192.0.2.255", false},
		{"192.0.2.20/31", "192.0.2.20", true},
		{"192.0.2.20/32", "192.0.2.20", true},
	} {
		devices := strings.Replace(addressFaultDevices, "ips: [192.0.2.20]",
			"ips: [192.0.2.20]\n    interfaces: [{name: eth0, address: "+tc.prefix+"}]", 1)
		_, err := config.LoadYAMLBytes(
			[]byte(devices + addressFaultTimeline("type: duplicate_dhcp_offer\naddress: "+tc.address)),
		)
		if tc.want && err != nil || !tc.want && !errors.Is(err, config.ErrBehaviorAddressTarget) {
			t.Errorf("prefix %s address %s: error = %v, want accepted=%v", tc.prefix, tc.address, err, tc.want)
		}
	}
}

func TestAddressFaultHonorsExplicitSegments(t *testing.T) {
	devices := strings.TrimPrefix(addressFaultDevices, "devices:\n")
	for _, separate := range []bool{false, true} {
		body := "segments:\n  - tag: 200\n    devices:\n    " +
			strings.ReplaceAll(strings.TrimSuffix(devices, "\n"), "\n", "\n    ") + "\n"
		if separate {
			body = strings.Replace(body, "      - name: peer", "  - tag: 300\n    devices:\n      - name: peer", 1)
		}
		_, err := config.LoadYAMLBytes(
			[]byte(body + addressFaultTimeline("type: duplicate_dhcp_offer\naddress: 192.0.2.20")),
		)
		if separate && !errors.Is(err, config.ErrBehaviorAddressTarget) || !separate && err != nil {
			t.Fatalf("separate=%v: %v\n%s", separate, err, body)
		}
	}
}

func TestAddressFaultRequiresSharedRoutedNetwork(t *testing.T) {
	for _, separate := range []bool{false, true} {
		body := `networks:
  - {name: access, subnet: 192.0.2.0/24, virtual_vlan: 200}
  - {name: remote, subnet: 198.51.100.0/24, virtual_vlan: 300}
attachments:
  - {name: tester, connect: access}
` + addressFaultDevices
		body = strings.Replace(
			body,
			"ips: [192.0.2.1]",
			"interfaces: [{name: eth0, network: access, address: 192.0.2.1/24}]",
			1,
		)
		body = strings.Replace(
			body,
			"ips: [192.0.2.20]",
			"interfaces: [{name: eth0, network: access, address: 192.0.2.20/24}]",
			1,
		)
		address := "192.0.2.20"
		if separate {
			body = strings.Replace(
				body,
				"network: access, address: 192.0.2.20/24",
				"network: remote, address: 198.51.100.20/24",
				1,
			)
			address = "198.51.100.20"
		}
		_, err := config.LoadYAMLBytes(
			[]byte(body + addressFaultTimeline("type: duplicate_dhcp_offer\naddress: "+address)),
		)
		if separate && !errors.Is(err, config.ErrBehaviorAddressTarget) || !separate && err != nil {
			t.Fatalf("separate=%v: %v", separate, err)
		}
	}
}
