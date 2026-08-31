package converter

import (
	"os"
	"testing"
)

// The daemon half of the device-YAML contract.
//
// testdata/ui_device_contract.yaml is produced by the UI's device-yaml mapper
// (ui/src/utils/device-yaml.contract.test.ts) and parsed here. The UI used to
// build this document twice, independently, and both copies emitted keys this
// package does not declare -- with KnownFields(true) that is a hard parse
// error, so the "loadable" export could not be loaded at all.
//
// One fixture, asserted from both sides: a rename here or a mapping change
// there fails a test instead of shipping a config nobody can load.
func TestUIDeviceContractParses(t *testing.T) {
	data, err := os.ReadFile("testdata/ui_device_contract.yaml")
	if err != nil {
		t.Fatalf("read contract fixture: %v", err)
	}

	cfg, err := LoadYAMLConfigFromBytes(data)
	if err != nil {
		t.Fatalf("the UI's YAML does not parse against this package's schema: %v", err)
	}
	if len(cfg.Devices) != 1 {
		t.Fatalf("device count = %d, want 1", len(cfg.Devices))
	}
	d := cfg.Devices[0]

	// Spot-check one field per block. A block whose key were wrong would have
	// failed the parse above; these assert the values actually land, so a key
	// that silently moved to a different field is caught too.
	checks := []struct {
		name string
		got  any
		want any
	}{
		{"name", d.Name, "contract-sw1"},
		{"type", d.Type, "access-point"},
		{"mac", d.MAC, "00:11:22:33:44:55"},
		{"ips", len(d.IPs), 2},
		{"vlan", d.VLAN, 10},
		{"map_to_ip", d.MapToIP, "10.0.0.9"},
		{"snmp_agent.sysname", d.SnmpAgent.SysName, "contract-sw1"},
		{"snmp_agent.walk_file", d.SnmpAgent.WalkFile, "walks/sw1.walk"},
		{"snmp_agent.walk_files", len(d.SnmpAgent.WalkFiles), 2},
		{"snmp_agent.add_mibs", len(d.SnmpAgent.AddMibs), 1},
		{"dhcp.subnet_mask", d.Dhcp.SubnetMask, "255.255.255.0"},
		{"dns.forward_records", len(d.DNS.ForwardRecords), 1},
		{"lldp.system_description", d.Lldp.SystemDescription, "contract lldp"},
		{"cdp.platform", d.Cdp.Platform, "contract-platform"},
		{"stp.bridge_priority", d.Stp.BridgePriority, uint16(32768)},
		{"http.server_name", d.HTTP.ServerName, "contract-httpd"},
		{"ftp.welcome_banner", d.Ftp.WelcomeBanner, "line one\nline two"},
		{"netbios.name", d.Netbios.Name, "CONTRACTSW1"},
		{"interfaces", len(d.Interfaces), 1},
		{"interfaces[0].admin_status", d.Interfaces[0].AdminStatus, "up"},
		{"interfaces[0].oper_status", d.Interfaces[0].OperStatus, "up"},
		{"interfaces[0].description", d.Interfaces[0].Description, `uplink "quoted" \ and a: colon`},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}
