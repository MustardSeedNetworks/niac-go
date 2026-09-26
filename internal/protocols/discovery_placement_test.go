package protocols

import (
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// LLDP is link-local: a tester on a pool port hears the switch at the other
// end of its cable and nothing else. Before AP-2 the speakers were every
// device with an interface on the attachment network, which on a pack is the
// core switch three tiers up and never the access switch the tester is in.

func TestPoolDiscoveryLeavesOnlyFromTheTestersSwitch(t *testing.T) {
	access := placementAccessSwitch()
	for name, advertise := range discoveryAdvertisers() {
		t.Run(name, func(t *testing.T) {
			capture := &discoveryCapture{}
			stack := placementStackOn(t, capture)
			for index := range stack.config.Devices {
				enableDiscovery(&stack.config.Devices[index])
			}

			advertise(stack)
			drainDiscoveryPackets(stack)

			if len(capture.sources) == 0 {
				t.Fatalf("nothing advertised; want %s (%s)", access.Name, access.MACAddress)
			}
			for _, source := range capture.sources {
				if source != access.MACAddress.String() {
					t.Errorf("advertisement from %s on the wire; only %s (%s) is at the tester's cable",
						source, access.Name, access.MACAddress)
				}
			}
		})
	}
}

// The port a switch names is the one the tester is plugged into. One
// advertisement reaches every client on the shared wire, so it names the
// earliest-placed client's port, and before anyone is placed the port the next
// unpinned client will take -- the port a passive listener then transmits on.
func TestPoolDiscoveryNamesTheTestersPort(t *testing.T) {
	pinned := placementClient(9)
	stack := placementStackOn(t, &discoveryCapture{}, config.AttachmentPin{
		MAC: pinned.String(), Device: placementAccess, Interface: "GigabitEthernet1/0/43",
	})
	access := stackDevice(t, stack, placementAccess)
	// An authored port ID is a static claim; the pool port is where the cable is.
	access.CDPConfig = &config.CDPConfig{Enabled: true, PortID: "GigabitEthernet1/0/1"}
	access.FDPConfig = &config.FDPConfig{Enabled: true, PortID: "GigabitEthernet1/0/1"}

	steps := []struct {
		name   string
		client net.HardwareAddr
		want   string
	}{
		{"before any client", nil, "GigabitEthernet1/0/44"},
		{"first unpinned client", placementClient(1), "GigabitEthernet1/0/44"},
		{"pinned client arrives later", pinned, "GigabitEthernet1/0/44"},
	}
	for _, step := range steps {
		if step.client != nil {
			sendFrom(stack, step.client, "10.51.210.101")
		}
		got := map[string]string{
			"LLDP": string(stack.lldpHandler.buildPortIDTLV(access)[3:]),
			"CDP":  string(stack.cdpHandler.buildPortIDTLV(access)[4:]),
			"FDP":  string(stack.fdpHandler.buildPortTLV(access)[4:]),
		}
		for protocol, port := range got {
			if port != step.want {
				t.Errorf("%s: %s port = %q, want %q", step.name, protocol, port, step.want)
			}
		}
	}

	core := stackDevice(t, stack, placementCore)
	if port := string(stack.lldpHandler.buildPortIDTLV(core)[3:]); port != core.Interfaces[0].Name {
		t.Errorf("%s LLDP port = %q; a device off the pool keeps its first interface %q",
			placementCore, port, core.Interfaces[0].Name)
	}
}

// A new session starts from an empty pool, so the port a switch names is the
// next unpinned client's again, not the one a previous session's first client
// took. The pinned client arriving first is what makes the two differ.
func TestPoolDiscoveryPortStartsOverWithTheSession(t *testing.T) {
	pinned := placementClient(9)
	stack := placementStackOn(t, idleTransport{}, config.AttachmentPin{
		MAC: pinned.String(), Device: placementAccess, Interface: "GigabitEthernet1/0/43",
	})
	if err := stack.Start(); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	sendFrom(stack, pinned, "10.51.210.109")
	access := stackDevice(t, stack, placementAccess)
	if port := string(stack.lldpHandler.buildPortIDTLV(access)[3:]); port != "GigabitEthernet1/0/43" {
		t.Fatalf("with the pinned client placed first, LLDP port = %q, want its pin", port)
	}
	stack.Stop()

	if port := string(stack.lldpHandler.buildPortIDTLV(access)[3:]); port != "GigabitEthernet1/0/44" {
		t.Errorf("after the session stopped, LLDP port = %q, want the next unpinned client's %q",
			port, "GigabitEthernet1/0/44")
	}
}

func enableDiscovery(device *config.Device) {
	device.LLDPConfig = &config.LLDPConfig{Enabled: true}
	device.CDPConfig = &config.CDPConfig{Enabled: true}
	device.EDPConfig = &config.EDPConfig{Enabled: true}
	device.FDPConfig = &config.FDPConfig{Enabled: true}
	device.STPConfig = &config.STPConfig{Enabled: true}
}

func stackDevice(t *testing.T, stack *Stack, name string) *config.Device {
	t.Helper()
	for _, device := range stack.AllDevices() {
		if device.Name == name {
			return device
		}
	}
	t.Fatalf("no device %s", name)
	return nil
}
