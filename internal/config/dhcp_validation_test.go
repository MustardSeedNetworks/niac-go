package config

import (
	"fmt"
	"net"
	"strings"
	"testing"
)

func TestLoadYAMLRejectsInvalidDHCPv4(t *testing.T) {
	for _, fields := range []string{
		"subnet_mask: not-a-mask", "subnet_mask: 255.0.255.0", "subnet_mask: '2001:db8::1'",
		"router: invalid", "router: '2001:db8::1'", "domain_name_server: invalid",
		"server_identifier: invalid", "next_server_ip: invalid", "ntp_servers: [invalid]",
		"pool_start: invalid\n      pool_end: 10.0.0.109",
		"pool_start: '2001:db8::1'\n      pool_end: 10.0.0.109",
		"pool_start: 10.0.0.100", "pool_end: 10.0.0.109",
		"pool_start: 10.0.0.110\n      pool_end: 10.0.0.100",
		"pool_start: 10.0.0.0\n      pool_end: 10.1.0.0",
	} {
		t.Run(fields, func(t *testing.T) {
			data := dhcpValidationYAML(fields)
			if _, err := LoadYAMLBytes([]byte(data)); err == nil || !strings.Contains(err.Error(), "dhcp") {
				t.Fatalf("invalid DHCP accepted or missing field context: %v", err)
			}
		})
	}
}

func dhcpValidationYAML(fields string) string {
	return fmt.Sprintf(
		"devices:\n  - name: server\n    type: server\n    mac: '02:00:00:00:00:01'\n    ips: [10.0.0.2]\n    dhcp:\n      %s\n",
		fields,
	)
}

func TestLoadYAMLPreservesOmittedDHCPDefaults(t *testing.T) {
	for _, test := range []struct{ fields, mask, start, end string }{
		{"{}", "<nil>", "<nil>", "<nil>"},
		{"subnet_mask: 255.255.240.0", "255.255.240.0", "<nil>", "<nil>"},
		{"pool_start: 10.0.0.100\n      pool_end: 10.0.0.100", "<nil>", "10.0.0.100", "10.0.0.100"},
		{"pool_start: 10.0.0.0\n      pool_end: 10.0.255.255", "<nil>", "10.0.0.0", "10.0.255.255"},
	} {
		cfg, err := LoadYAMLBytes([]byte(dhcpValidationYAML(test.fields)))
		if err != nil {
			t.Fatalf("valid DHCP %q rejected: %v", test.fields, err)
		}
		dhcp := cfg.Devices[0].DHCPConfig
		if net.IP(dhcp.SubnetMask).String() != test.mask || dhcp.PoolStart.String() != test.start ||
			dhcp.PoolEnd.String() != test.end {
			t.Fatalf("parsed values changed: %+v", dhcp)
		}
		if result := NewValidator("valid.yaml").Validate(cfg); !result.Valid || len(result.Errors) != 0 {
			t.Fatalf("valid runtime baseline rejected: %+v", result.Errors)
		}
	}
}

func TestRuntimeValidatorRejectsInvalidDHCPv4(t *testing.T) {
	for _, test := range []struct {
		dhcp  DHCPConfig
		field string
	}{
		{DHCPConfig{SubnetMask: net.IPMask{255, 0, 255, 0}}, "subnet_mask"},
		{DHCPConfig{SubnetMask: net.IPMask{255, 255}}, "subnet_mask"},
		{DHCPConfig{Router: net.ParseIP("2001:db8::1")}, "router"},
		{DHCPConfig{ServerIdentifier: net.IP{1, 2}}, "server_identifier"},
		{DHCPConfig{DomainNameServer: []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("2001:db8::1")}}, "domain_name_server[1]"},
		{DHCPConfig{PoolStart: net.ParseIP("2001:db8::1"), PoolEnd: net.ParseIP("10.0.0.109")}, "pool_end"},
		{DHCPConfig{PoolStart: net.ParseIP("10.0.0.100")}, "pool_end"},
		{DHCPConfig{PoolStart: net.ParseIP("10.0.0.110"), PoolEnd: net.ParseIP("10.0.0.100")}, "pool_end"},
		{DHCPConfig{PoolStart: net.ParseIP("10.0.0.0"), PoolEnd: net.ParseIP("10.1.0.0")}, "pool_end"},
	} {
		cfg := &Config{
			Devices: []Device{
				{
					Name:       "server",
					Type:       "server",
					MACAddress: net.HardwareAddr{2, 0, 0, 0, 0, 1},
					DHCPConfig: &test.dhcp,
				},
			},
		}
		result := NewValidator("test.yaml").Validate(cfg)
		if result.Valid || len(result.Errors) != 1 || result.Errors[0].Field != "devices[0].dhcp."+test.field {
			t.Errorf("expected exact DHCP %s error, got %+v", test.field, result.Errors)
		}
	}
}
