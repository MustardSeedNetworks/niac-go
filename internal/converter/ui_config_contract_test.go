package converter

import (
	"os"
	"testing"
)

// The daemon half of the config-section contract.
//
// testdata/ui_config_contract.yaml is produced by the wizard's section
// composers (ui/src/components/wizard/config-sections.contract.test.ts) and
// parsed here. The wizard writes networks, attachments, discovery defaults,
// capture playback and a device address as text, and only this package's
// strict decoder says whether the daemon accepts it -- a round trip through
// the UI's own YAML library would agree with itself no matter what it spelled.
//
// One fixture, asserted from both sides: a rename here or a serializer change
// there fails a test instead of shipping a config nobody can load.
func TestUIConfigContractParses(t *testing.T) {
	data, err := os.ReadFile("testdata/ui_config_contract.yaml")
	if err != nil {
		t.Fatalf("read contract fixture: %v", err)
	}

	cfg, err := LoadYAMLConfigFromBytes(data)
	if err != nil {
		t.Fatalf("the wizard's YAML does not parse against this package's schema: %v", err)
	}

	if len(cfg.Networks) != 2 {
		t.Fatalf("networks = %d, want 2", len(cfg.Networks))
	}
	if len(cfg.Attachments) != 1 {
		t.Fatalf("attachments = %d, want 1", len(cfg.Attachments))
	}
	if len(cfg.CapturePlaybacks) != 1 {
		t.Fatalf("capture playbacks = %d, want 1", len(cfg.CapturePlaybacks))
	}
	if cfg.DiscoveryProtocols == nil {
		t.Fatal("discovery protocols missing")
	}
	if len(cfg.Devices) != 1 || len(cfg.Devices[0].Interfaces) != 1 {
		t.Fatalf("devices = %#v, want one device carrying one interface", cfg.Devices)
	}

	// Spot-check one value per section. A key that were wrong would have
	// failed the parse above; these assert the values actually land, so a key
	// that silently moved to a different field is caught too.
	checks := []struct {
		name string
		got  any
		want any
	}{
		{"networks[0].name", cfg.Networks[0].Name, "contract-lan"},
		{"networks[0].subnet", cfg.Networks[0].Subnet, "10.20.0.0/24"},
		{"networks[1].virtual_vlan", cfg.Networks[1].VirtualVLAN, 99},
		{"attachments[0].name", cfg.Attachments[0].Name, "tester"},
		{"attachments[0].connect", cfg.Attachments[0].Connect, "contract-lan"},
		{"capture_playbacks[0].file_name", cfg.CapturePlaybacks[0].FileName, "contract.pcap"},
		{"capture_playbacks[0].loop_time", cfg.CapturePlaybacks[0].LoopTime, 60},
		{"capture_playbacks[0].scale_time", cfg.CapturePlaybacks[0].ScaleTime, 0.5},
		{"discovery_protocols.lldp.enabled", cfg.DiscoveryProtocols.LLDP.Enabled, true},
		{"discovery_protocols.lldp.interval", cfg.DiscoveryProtocols.LLDP.Interval, 30},
		{"discovery_protocols.cdp.enabled", cfg.DiscoveryProtocols.CDP.Enabled, true},
		{
			"devices[0].interfaces[0].address",
			cfg.Devices[0].Interfaces[0].Address,
			"10.20.0.1/24",
		},
		{
			"devices[0].interfaces[0].network",
			cfg.Devices[0].Interfaces[0].Network,
			"contract-lan",
		},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("%s = %#v, want %#v", check.name, check.got, check.want)
		}
	}
}
