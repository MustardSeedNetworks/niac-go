package fabric_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// poolConfig is one access switch with two free ports on the user VLAN, its
// uplink to a core switch that owns the VLAN's SVI, and one occupied port.
//
// The shape matters: an access switch owns only its own management SVI, so the
// network a free port lands a client on is never readable from the switch
// itself -- it is on whichever device owns that VLAN's layer-3 interface,
// reached over the trunk. Every generated pack is built this way.
func poolConfig() *config.Config {
	return &config.Config{
		Networks: []config.Network{
			{Name: "med-mgmt", Subnet: "10.51.200.0/24", VirtualVLAN: 200},
			{Name: "med-data", Subnet: "10.51.210.0/24", VirtualVLAN: 210},
		},
		Attachments: []config.LogicalAttachment{{
			Name: "cyberscope",
			At: &config.AttachmentPort{
				Device: "MED-ACC-SW01",
				Ports:  []string{"GigabitEthernet1/0/20", "GigabitEthernet1/0/21"},
			},
		}},
		Devices: []config.Device{accessSwitch(), coreSwitch()},
	}
}

func accessSwitch() config.Device {
	return config.Device{
		Name: "MED-ACC-SW01",
		Type: "switch",
		Interfaces: []config.Interface{
			{Name: "Vlan200", Network: "med-mgmt", Address: "10.51.200.21/24"},
			{Name: "HundredGigabitEthernet1/0/49", VLANs: []int{200, 210}},
			{Name: "GigabitEthernet1/0/10", VLANs: []int{210}},
			{Name: "GigabitEthernet1/0/20", VLANs: []int{210}},
			{Name: "GigabitEthernet1/0/21", VLANs: []int{210}},
		},
		TrunkPorts: []config.TrunkPort{
			{
				Interface:    "HundredGigabitEthernet1/0/49",
				VLANs:        []int{200, 210},
				NativeVLAN:   200,
				RemoteDevice: "MED-CORE-SW01",
			},
			{
				Interface:    "GigabitEthernet1/0/10",
				VLANs:        []int{210},
				NativeVLAN:   210,
				RemoteDevice: "MED-NURSE-1101",
				FDBOnly:      true,
			},
		},
	}
}

func coreSwitch() config.Device {
	return config.Device{
		Name: "MED-CORE-SW01",
		Type: "layer3-switch",
		Interfaces: []config.Interface{
			{Name: "Vlan200", Network: "med-mgmt", Address: "10.51.200.2/24"},
			{Name: "Vlan210", Network: "med-data", Address: "10.51.210.2/24"},
			{Name: "HundredGigabitEthernet1/0/1", VLANs: []int{200, 210}},
		},
		TrunkPorts: []config.TrunkPort{{
			Interface:    "HundredGigabitEthernet1/0/1",
			VLANs:        []int{200, 210},
			NativeVLAN:   200,
			RemoteDevice: "MED-ACC-SW01",
		}},
	}
}

func poolBinding() fabric.Binding {
	return fabric.Binding{
		Attachment:     "cyberscope",
		Interface:      "eth0",
		Mode:           fabric.ModeAccess,
		AccessVLAN:     200,
		PolicyApproved: true,
	}
}

func TestCompilePoolAttachmentResolvesEveryPortToItsNetwork(t *testing.T) {
	report := fabric.Compile(poolConfig(), poolBinding())

	if !report.Safe {
		t.Fatalf("diagnostics = %#v", report.Diagnostics)
	}
	if len(report.Topology.Attachments) != 1 {
		t.Fatalf("attachments = %#v", report.Topology.Attachments)
	}
	attachment := report.Topology.Attachments[0]
	if attachment.Name != "cyberscope" || attachment.Device != "MED-ACC-SW01" {
		t.Fatalf("attachment = %#v", attachment)
	}
	want := []fabric.AttachmentPort{
		{Device: "MED-ACC-SW01", Interface: "GigabitEthernet1/0/20", VLAN: 210, Network: "med-data"},
		{Device: "MED-ACC-SW01", Interface: "GigabitEthernet1/0/21", VLAN: 210, Network: "med-data"},
	}
	if !slices.Equal(attachment.Ports, want) {
		t.Fatalf("ports = %#v, want %#v", attachment.Ports, want)
	}
	// The pool's network is what a client on any of its ports lands on; the
	// binding still has to report one, because the runtime derives the DHCP
	// server and gateway from it.
	if report.Topology.Binding.Network != "med-data" {
		t.Fatalf("binding network = %q, want med-data", report.Topology.Binding.Network)
	}
}

// The wire VLAN and the port VLAN are separate namespaces: the host may be
// cabled on VLAN 200 while the tester lands inside the scenario on VLAN 210.
// A check that they agree would reject valid moves, so there must not be one.
func TestCompilePoolAttachmentDoesNotCompareWireVLANToPortVLAN(t *testing.T) {
	binding := poolBinding()
	binding.AccessVLAN = 4094

	report := fabric.Compile(poolConfig(), binding)

	if !report.Safe {
		t.Fatalf("a wire VLAN unrelated to the port VLAN was refused: %#v", report.Diagnostics)
	}
}

func TestCompilePoolAttachmentDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		want fabric.DiagnosticCode
		edit func(cfg *config.Config)
	}{
		{
			name: "unknown device",
			want: fabric.CodeUnknownAttachmentDevice,
			edit: func(cfg *config.Config) { cfg.Attachments[0].At.Device = "NOPE-SW01" },
		},
		{
			name: "unknown interface",
			want: fabric.CodeUnknownAttachmentPort,
			edit: func(cfg *config.Config) {
				cfg.Attachments[0].At.Ports = []string{"GigabitEthernet1/0/99"}
			},
		},
		{
			name: "port already carries a topology edge",
			want: fabric.CodeAttachmentPortOccupied,
			edit: func(cfg *config.Config) {
				cfg.Attachments[0].At.Ports = []string{"HundredGigabitEthernet1/0/49"}
			},
		},
		{
			name: "port learns a client through the forwarding database",
			want: fabric.CodeAttachmentPortOccupied,
			edit: func(cfg *config.Config) {
				cfg.Attachments[0].At.Ports = []string{"GigabitEthernet1/0/10"}
			},
		},
		{
			name: "port carries no single access VLAN",
			want: fabric.CodeAttachmentPortVLANUnresolved,
			edit: func(cfg *config.Config) {
				cfg.Devices[0].Interfaces[3].VLANs = []int{210, 250}
			},
		},
		{
			name: "port VLAN has no network",
			want: fabric.CodeAttachmentPortNetworkUnresolved,
			edit: func(cfg *config.Config) {
				cfg.Networks = cfg.Networks[:1]
				cfg.Devices[1].Interfaces = cfg.Devices[1].Interfaces[:1]
			},
		},
		{
			name: "port VLAN serves more than one network in reach",
			want: fabric.CodeAttachmentPortNetworkAmbiguous,
			edit: func(cfg *config.Config) {
				cfg.Networks = append(cfg.Networks, config.Network{
					Name: "amb-data", Subnet: "10.52.210.0/24", VirtualVLAN: 210,
				})
				cfg.Devices[1].Interfaces = append(cfg.Devices[1].Interfaces, config.Interface{
					Name: "Vlan210-2", Network: "amb-data", Address: "10.52.210.2/24",
				})
			},
		},
		{
			name: "pool ports land on different networks",
			want: fabric.CodeAttachmentPoolNetworksDiffer,
			edit: func(cfg *config.Config) {
				cfg.Devices[0].Interfaces[4].VLANs = []int{200}
			},
		},
		{
			name: "one port listed twice",
			want: fabric.CodeDuplicateAttachmentPort,
			edit: func(cfg *config.Config) {
				cfg.Attachments[0].At.Ports = []string{
					"GigabitEthernet1/0/20", "GigabitEthernet1/0/20",
				}
			},
		},
		{
			name: "empty pool",
			want: fabric.CodeAttachmentPoolEmpty,
			edit: func(cfg *config.Config) { cfg.Attachments[0].At.Ports = nil },
		},
		{
			name: "both attachment forms",
			want: fabric.CodeAttachmentFormAmbiguous,
			edit: func(cfg *config.Config) { cfg.Attachments[0].Network = "med-data" },
		},
		{
			name: "neither attachment form",
			want: fabric.CodeAttachmentFormAmbiguous,
			edit: func(cfg *config.Config) { cfg.Attachments[0].At = nil },
		},
		{
			name: "pin outside its pool",
			want: fabric.CodeAttachmentPinOutsidePool,
			edit: func(cfg *config.Config) {
				cfg.Attachments[0].Pins = []config.AttachmentPin{{
					MAC:       "00:c0:17:aa:bb:cc",
					Device:    "MED-ACC-SW01",
					Interface: "GigabitEthernet1/0/10",
				}}
			},
		},
		{
			name: "two pins on one port",
			want: fabric.CodeAttachmentPinDuplicate,
			edit: func(cfg *config.Config) {
				cfg.Attachments[0].Pins = []config.AttachmentPin{
					{
						MAC:       "00:c0:17:aa:bb:cc",
						Device:    "MED-ACC-SW01",
						Interface: "GigabitEthernet1/0/20",
					},
					{
						MAC:       "00:c0:17:aa:bb:dd",
						Device:    "MED-ACC-SW01",
						Interface: "GigabitEthernet1/0/20",
					},
				}
			},
		},
		{
			name: "unparseable pin MAC",
			want: fabric.CodeInvalidAttachmentPinMAC,
			edit: func(cfg *config.Config) {
				cfg.Attachments[0].Pins = []config.AttachmentPin{{
					MAC:       "not-a-mac",
					Device:    "MED-ACC-SW01",
					Interface: "GigabitEthernet1/0/20",
				}}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := poolConfig()
			test.edit(cfg)

			report := fabric.Compile(cfg, poolBinding())

			if report.Safe {
				t.Fatalf("compile accepted %s", test.name)
			}
			if !slices.ContainsFunc(report.Diagnostics, func(d fabric.Diagnostic) bool {
				return d.Code == test.want
			}) {
				t.Fatalf("diagnostics = %#v, want %s", report.Diagnostics, test.want)
			}
			// Every pool finding describes the scenario file, not the physical
			// deployment, so an authoring surface with no binding reports it too.
			if test.want.IsBinding() {
				t.Fatalf("%s is classified as a binding diagnostic", test.want)
			}
			if !slices.ContainsFunc(
				fabric.CompileConfig(cfg).Diagnostics,
				func(d fabric.Diagnostic) bool { return d.Code == test.want },
			) {
				t.Fatalf("CompileConfig did not report %s", test.want)
			}
		})
	}
}

// A pinned MAC is what makes "this CyberScope is always sw2 Gi1/0/1" express
// ible; it has to survive the compile in a form the runtime can key on.
func TestCompilePoolAttachmentKeepsPins(t *testing.T) {
	cfg := poolConfig()
	cfg.Attachments[0].Pins = []config.AttachmentPin{{
		MAC:       "00:C0:17:AA:BB:CC",
		Device:    "MED-ACC-SW01",
		Interface: "GigabitEthernet1/0/21",
	}}

	report := fabric.Compile(cfg, poolBinding())

	if !report.Safe {
		t.Fatalf("diagnostics = %#v", report.Diagnostics)
	}
	pins := report.Topology.Attachments[0].Pins
	if len(pins) != 1 || pins[0].Interface != "GigabitEthernet1/0/21" {
		t.Fatalf("pins = %#v", pins)
	}
	// Canonicalised, so a runtime lookup by MAC does not depend on how the
	// author typed it.
	if pins[0].MAC != "00:c0:17:aa:bb:cc" {
		t.Fatalf("pin MAC = %q, want lower-case canonical form", pins[0].MAC)
	}
}

// The network-scoped form is what every scenario uses today and must keep
// compiling unchanged.
func TestCompileNetworkAttachmentStillCompiles(t *testing.T) {
	report := fabric.Compile(referenceConfig(), fabric.Binding{
		Attachment:     "tester",
		Interface:      "eth0",
		Mode:           fabric.ModeAccess,
		AccessVLAN:     200,
		PolicyApproved: true,
	})

	if !report.Safe {
		t.Fatalf("diagnostics = %#v", report.Diagnostics)
	}
	if report.Topology.Binding.Network != "lab-access" {
		t.Fatalf("binding network = %q", report.Topology.Binding.Network)
	}
	if len(report.Topology.Attachments) != 0 {
		t.Fatalf("a network attachment declared ports: %#v", report.Topology.Attachments)
	}
}
