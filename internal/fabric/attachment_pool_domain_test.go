package fabric_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// twoSitePoolConfig is the shape every multi-site generated pack has: two
// sites that reuse the same VLAN ids on their own networks, joined by a
// *routed* WAN link -- no VLANs, no native VLAN -- through an edge router.
//
// A tester on a free access port in site A lands on site A's data network and
// on nothing else. Site B's data network carries the same VLAN id but is a
// different broadcast domain: the router between them terminates layer 2.
func twoSitePoolConfig() *config.Config {
	site := func(code, thirdOctet, wanAddress string) []config.Device {
		return []config.Device{
			{
				Name: code + "-ACC-SW01",
				Type: "switch",
				Interfaces: []config.Interface{
					{
						Name: "Vlan200", Network: code + "-mgmt",
						Address: "10." + thirdOctet + ".200.21/24",
					},
					{Name: "HundredGigabitEthernet1/0/49", VLANs: []int{200, 210}},
					{Name: "GigabitEthernet1/0/45", VLANs: []int{210}},
				},
				TrunkPorts: []config.TrunkPort{{
					Interface:    "HundredGigabitEthernet1/0/49",
					VLANs:        []int{200, 210},
					NativeVLAN:   200,
					RemoteDevice: code + "-CORE-SW01",
				}},
			},
			{
				Name: code + "-CORE-SW01",
				Type: "layer3-switch",
				Interfaces: []config.Interface{
					{
						Name: "Vlan200", Network: code + "-mgmt",
						Address: "10." + thirdOctet + ".200.2/24",
					},
					{
						Name: "Vlan210", Network: code + "-data",
						Address: "10." + thirdOctet + ".210.2/24",
					},
					{Name: "HundredGigabitEthernet1/0/1", VLANs: []int{200, 210}},
					{
						Name: "HundredGigabitEthernet0/0/1", Network: "wan-" + code,
						Address: wanAddress,
					},
				},
				TrunkPorts: []config.TrunkPort{
					{
						Interface:    "HundredGigabitEthernet1/0/1",
						VLANs:        []int{200, 210},
						NativeVLAN:   200,
						RemoteDevice: code + "-ACC-SW01",
					},
					// The routed uplink: no VLANs, no native VLAN.
					{
						Interface:    "HundredGigabitEthernet0/0/1",
						RemoteDevice: "EDGE-R1",
					},
				},
			},
		}
	}

	devices := append(
		site("aaa", "51", "203.0.113.2/29"),
		site("bbb", "52", "203.0.113.10/29")...,
	)
	devices = append(devices, config.Device{
		Name: "EDGE-R1",
		Type: "router",
		Interfaces: []config.Interface{
			{Name: "HundredGigabitEthernet0/0/1", Network: "wan-aaa", Address: "203.0.113.1/29"},
			{Name: "HundredGigabitEthernet0/0/2", Network: "wan-bbb", Address: "203.0.113.9/29"},
		},
		TrunkPorts: []config.TrunkPort{
			{Interface: "HundredGigabitEthernet0/0/1", RemoteDevice: "aaa-CORE-SW01"},
			{Interface: "HundredGigabitEthernet0/0/2", RemoteDevice: "bbb-CORE-SW01"},
		},
	})

	return &config.Config{
		Networks: []config.Network{
			{Name: "aaa-mgmt", Subnet: "10.51.200.0/24", VirtualVLAN: 200},
			{Name: "aaa-data", Subnet: "10.51.210.0/24", VirtualVLAN: 210},
			{Name: "bbb-mgmt", Subnet: "10.52.200.0/24", VirtualVLAN: 200},
			{Name: "bbb-data", Subnet: "10.52.210.0/24", VirtualVLAN: 210},
			{Name: "wan-aaa", Subnet: "203.0.113.0/29"},
			{Name: "wan-bbb", Subnet: "203.0.113.8/29"},
		},
		Attachments: []config.LogicalAttachment{{
			Name: "cyberscope",
			At: &config.AttachmentPort{
				Device: "aaa-ACC-SW01",
				Ports:  []string{"GigabitEthernet1/0/45"},
			},
		}},
		Devices: devices,
	}
}

// TestCompilePoolAttachmentIgnoresRoutedLinks is the regression for the defect
// AP-3 found in AP-1's compiler: broadcastDomain walked every authored link,
// including routed ones, so the "domain" from an access switch was the whole
// scenario. Two sites reusing a VLAN id -- which every generated multi-site
// pack does -- then made the port's network ambiguous and the pool unusable.
func TestCompilePoolAttachmentIgnoresRoutedLinks(t *testing.T) {
	report := fabric.CompileConfig(twoSitePoolConfig())

	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code == fabric.CodeAttachmentPortNetworkAmbiguous {
			t.Fatalf("routed link merged two sites' layer-2 domains: %s", diagnostic.Message)
		}
	}
	if len(report.Topology.Attachments) != 1 {
		t.Fatalf("attachments = %#v, diagnostics = %#v",
			report.Topology.Attachments, report.Diagnostics)
	}
	attachment := report.Topology.Attachments[0]
	if attachment.Network != "aaa-data" {
		t.Fatalf("pool network = %q, want aaa-data", attachment.Network)
	}
	if len(attachment.Ports) != 1 || attachment.Ports[0].VLAN != 210 {
		t.Fatalf("ports = %#v", attachment.Ports)
	}
}

// A VLAN the trunk between the port and the SVI owner does not carry is not in
// reach, however many links away the SVI is. Pruning on the VLAN rather than
// only on "is this link routed" is what makes that true.
func TestCompilePoolAttachmentRequiresTheTrunkToCarryThePortVLAN(t *testing.T) {
	cfg := twoSitePoolConfig()
	for i := range cfg.Devices {
		if cfg.Devices[i].Name != "aaa-ACC-SW01" {
			continue
		}
		for j := range cfg.Devices[i].TrunkPorts {
			trunk := &cfg.Devices[i].TrunkPorts[j]
			if trunk.Interface == "HundredGigabitEthernet1/0/49" {
				trunk.VLANs = []int{200}
			}
		}
	}

	report := fabric.CompileConfig(cfg)

	found := false
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code == fabric.CodeAttachmentPortNetworkUnresolved {
			found = true
		}
	}
	if !found {
		t.Fatalf("a VLAN the uplink does not carry resolved anyway: %#v", report.Diagnostics)
	}
}
