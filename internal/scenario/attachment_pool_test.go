package scenario_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// AP-3. Every pack used to plant the tester on LAB-EDGE-R1's transit link --
// nobody plugs a CyberScope into a lab edge router's uplink, and with several
// testers on one switch they all appeared in that same non-place. The tester
// now arrives on a pool of free access ports, which is where a technician
// actually plugs in.

// freePorts are the ports a device declares that carry no link, no learned
// client and no port-channel membership: exactly what a pool may draw from.
func freePorts(device *config.Device) []string {
	var free []string
	for i := range device.Interfaces {
		iface := &device.Interfaces[i]
		if iface.Type != "ethernet" || len(iface.VLANs) != 1 {
			continue
		}
		occupied := slices.ContainsFunc(device.TrunkPorts, func(trunk config.TrunkPort) bool {
			return trunk.Interface == iface.Name
		})
		if occupied {
			continue
		}
		if slices.ContainsFunc(device.PortChannels, func(channel config.PortChannel) bool {
			return slices.Contains(channel.Members, iface.Name)
		}) {
			continue
		}
		free = append(free, iface.Name)
	}
	return free
}

func TestEveryAccessSwitchOffersASparePortPool(t *testing.T) {
	const wantFree = 4
	for _, pack := range scenario.Packs() {
		cfg := generatedPack(t, pack).Config
		switches := 0
		for i := range cfg.Devices {
			device := &cfg.Devices[i]
			if !strings.Contains(device.Name, "-ACC-SW") &&
				!strings.Contains(device.Name, "-SRV-SW") {
				continue
			}
			switches++
			if free := freePorts(device); len(free) < wantFree {
				t.Errorf("%s %s offers %d free ports %v, want at least %d",
					pack.ID, device.Name, len(free), free, wantFree)
			}
		}
		if switches == 0 {
			t.Errorf("%s: no access or server switch found", pack.ID)
		}
	}
}

func TestTesterAttachesToAnAccessPortPool(t *testing.T) {
	for _, pack := range scenario.Packs() {
		cfg := generatedPack(t, pack).Config
		if len(cfg.Attachments) != 1 {
			t.Fatalf("%s: attachments = %#v", pack.ID, cfg.Attachments)
		}
		attachment := cfg.Attachments[0]
		if attachment.Network != "" {
			t.Errorf("%s: tester still attaches to network %q instead of a port pool",
				pack.ID, attachment.Network)
			continue
		}
		if attachment.At == nil {
			t.Errorf("%s: attachment declares no pool", pack.ID)
			continue
		}
		if !strings.Contains(attachment.At.Device, "-ACC-SW") {
			t.Errorf("%s: pool sits on %s, want an access switch",
				pack.ID, attachment.At.Device)
		}
		if len(attachment.At.Ports) < 4 {
			t.Errorf("%s: pool has %d ports, want at least 4",
				pack.ID, len(attachment.At.Ports))
		}
	}
}

// The pool has to compile, not merely parse: each port must resolve to the one
// network a client on it lands on. This is the clause the fabric compiler
// could not satisfy before the VLAN-scoped broadcast domain landed with it.
func TestGeneratedPoolCompilesToOneNetwork(t *testing.T) {
	for _, pack := range scenario.Packs() {
		cfg := generatedPack(t, pack).Config
		report := fabric.CompileConfig(cfg)
		for _, diagnostic := range report.Diagnostics {
			if strings.Contains(string(diagnostic.Code), "attachment") {
				t.Errorf("%s: %s %s: %s",
					pack.ID, diagnostic.Code, diagnostic.Field, diagnostic.Message)
			}
		}
		if len(report.Topology.Attachments) != 1 {
			t.Errorf("%s: compiled attachments = %#v", pack.ID, report.Topology.Attachments)
			continue
		}
		compiled := report.Topology.Attachments[0]
		if compiled.Network == "" {
			t.Errorf("%s: pool resolved no network", pack.ID)
		}
		// Every port on one network is what resolvePoolNetwork enforces; assert
		// the ports themselves agree, so a future pool spanning VLANs is caught
		// here rather than by a runtime that derives one gateway from it.
		for _, port := range compiled.Ports {
			if port.Network != compiled.Network {
				t.Errorf("%s: port %s lands on %s, pool network is %s",
					pack.ID, port.Interface, port.Network, compiled.Network)
			}
		}
	}
}

// The tester's old home. The row deletes the connect and the description that
// named the instrument; the transit network and the port stay, because the lab
// edge still has an uplink.
func TestNoPackNamesTheTesterInAnInterfaceDescription(t *testing.T) {
	for _, pack := range scenario.Packs() {
		yaml := string(generatedPack(t, pack).YAML)
		if strings.Contains(yaml, "CyberScope VLAN 200 attachment") {
			t.Errorf("%s: an interface description still names the tester", pack.ID)
		}
	}
}
