package protocols

import (
	"net"
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestRoutedDiscoveryAdvertisementsStayOnAttachment(t *testing.T) {
	for name, advertise := range discoveryAdvertisers() {
		t.Run(name, func(t *testing.T) {
			capture, stack := routedDiscoveryStack()

			advertise(stack)
			drainDiscoveryPackets(stack)

			if len(capture.sources) != 1 || capture.sources[0] != "02:00:00:00:00:01" {
				t.Fatalf("wire sources = %v, want attachment device only", capture.sources)
			}
			stats := stack.GetStats()
			if stats.Errors != 0 || stats.FabricDrops != 1 {
				t.Fatalf("Errors = %d, FabricDrops = %d; want 0, 1", stats.Errors, stats.FabricDrops)
			}
		})
	}
}

func TestFlatDiscoveryAdvertisementsRemainVisible(t *testing.T) {
	for name, advertise := range discoveryAdvertisers() {
		t.Run(name, func(t *testing.T) {
			capture, stack := flatDiscoveryStack()

			advertise(stack)
			drainDiscoveryPackets(stack)

			if len(capture.sources) != 2 {
				t.Fatalf("wire sources = %v, want both flat devices", capture.sources)
			}
		})
	}
}

func TestRoutedDiscoveryChecksEveryAttachmentInterface(t *testing.T) {
	_, stack := routedDiscoveryStack()
	edge := &stack.config.Devices[0]
	stack.fabric.topology.Interfaces = append(
		[]fabric.Interface{{
			Device: edge.Name, Name: "unavailable", Network: "attachment",
			Address: netip.MustParsePrefix("10.10.200.2/24"),
		}},
		stack.fabric.topology.Interfaces...,
	)
	network := stack.deviceStates[edge].Snapshot().Network
	network.Interfaces = append(network.Interfaces, devicestate.Interface{
		Name: "unavailable", Address: netip.MustParsePrefix("10.10.200.2/24"),
		AdminUp: false, OperUp: false,
	})
	stack.deviceStates[edge].ReplaceNetwork(network)
	destination, _ := net.ParseMAC(LLDPMulticastMAC)
	frame := make([]byte, ethernetHeaderSize)
	copy(frame, destination)

	if err := stack.validateDiscoveryEgress(&Packet{Buffer: frame, Device: edge}); err != nil {
		t.Fatalf("attachment advertisement rejected after unavailable interface: %v", err)
	}
}

func TestRoutedDiscoveryRejectsUnknownDeviceIdentity(t *testing.T) {
	_, stack := routedDiscoveryStack()
	destination, _ := net.ParseMAC(LLDPMulticastMAC)
	frame := make([]byte, ethernetHeaderSize)
	copy(frame, destination)

	err := stack.validateDiscoveryEgress(&Packet{Buffer: frame, Device: "inside"})

	if err == nil {
		t.Fatal("unknown discovery device identity was allowed onto the attachment")
	}
}

func discoveryAdvertisers() map[string]func(*Stack) {
	return map[string]func(*Stack){
		"LLDP": func(stack *Stack) { stack.lldpHandler.sendAdvertisements() },
		"CDP":  func(stack *Stack) { stack.cdpHandler.sendAdvertisements() },
		"EDP":  func(stack *Stack) { stack.edpHandler.sendAdvertisements() },
		"FDP":  func(stack *Stack) { stack.fdpHandler.sendAdvertisements() },
	}
}

func routedDiscoveryStack() (*discoveryCapture, *Stack) {
	capture, stack := flatDiscoveryStack()
	stack.ConfigureFabric(&fabric.Topology{
		Binding: fabric.CompiledBinding{Network: "attachment"},
		Interfaces: []fabric.Interface{
			{
				Device: "edge", Name: "outside", Network: "attachment",
				Address: netip.MustParsePrefix("10.10.200.1/24"),
			},
			{
				Device: "inside", Name: "inside", Network: "internal",
				Address: netip.MustParsePrefix("10.20.0.1/24"),
			},
		},
	})
	return capture, stack
}

func flatDiscoveryStack() (*discoveryCapture, *Stack) {
	capture := &discoveryCapture{}
	stack := newStack(capture, &config.Config{Devices: discoveryDevices()}, logging.NewDebugConfig(0))
	return capture, stack
}

func discoveryDevices() []config.Device {
	return []config.Device{
		discoveryDevice("edge", "02:00:00:00:00:01"),
		discoveryDevice("inside", "02:00:00:00:00:02"),
	}
}

func discoveryDevice(name, mac string) config.Device {
	address, _ := net.ParseMAC(mac)
	return config.Device{
		Name: name, Type: "switch", MACAddress: address,
		LLDPConfig: &config.LLDPConfig{Enabled: true},
		CDPConfig:  &config.CDPConfig{Enabled: true},
		EDPConfig:  &config.EDPConfig{Enabled: true},
		FDPConfig:  &config.FDPConfig{Enabled: true},
	}
}

func drainDiscoveryPackets(stack *Stack) {
	for len(stack.sendQueue) > 0 {
		stack.sendPacket(<-stack.sendQueue)
	}
}
