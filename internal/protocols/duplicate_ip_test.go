package protocols

import (
	"net"
	"net/netip"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestDuplicateIPAddsOnlyExplicitConflictResponder(t *testing.T) {
	_, pair := isolationPair(t)
	cfg := &config.Config{Segments: []config.Segment{{Tag: 200, Devices: pair.Devices}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	address := netip.MustParseAddr("10.0.0.3")
	store := stack.deviceStates[&cfg.Segments[0].Devices[0]]
	iface := store.Snapshot().Network.Interfaces[0].Name
	fault := devicestate.InterfaceAddressFault{Interface: iface, Type: devicestate.FaultDuplicateIP, Address: address}
	handler := NewARPHandler(stack)
	if got := handler.targetDevices(net.IP(address.AsSlice()), 200); len(got) != 1 {
		t.Fatal("invalid healthy baseline")
	}
	if err := stack.SetInterfaceAddressFault("a", fault); err != nil {
		t.Fatal(err)
	}
	if got := handler.targetDevices(net.IP(address.AsSlice()), 200); len(got) != 2 {
		t.Fatalf("want two ARP identities, got %d", len(got))
	}
	assertDuplicateIPReplies(t, handler, cfg.Segments[0].Devices)
	if got := stack.devicesForStateIPv4(
		200,
		net.IP(address.AsSlice()),
	); len(got) != 1 ||
		got[0] != &cfg.Segments[0].Devices[1] {
		t.Fatal("canonical ownership changed")
	}
	if got := handler.targetDevices(net.IP(address.AsSlice()), 300); len(got) != 0 {
		t.Fatal("conflict escaped segment")
	}
	if err := store.SetInterfaceFault(iface, devicestate.FaultLinkDown, 1); err != nil {
		t.Fatal(err)
	}
	if got := handler.targetDevices(net.IP(address.AsSlice()), 200); len(got) != 1 {
		t.Fatal("down interface remained conflict responder")
	}
	if err := store.SetInterfaceFault(iface, devicestate.FaultLinkDown, 0); err != nil {
		t.Fatal(err)
	}
	if err := stack.ClearInterfaceAddressFault("a", iface, devicestate.FaultDuplicateIP); err != nil {
		t.Fatal(err)
	}
	if got := handler.targetDevices(net.IP(address.AsSlice()), 200); len(got) != 1 {
		t.Fatal("clear retained conflict responder")
	}
}

func assertDuplicateIPReplies(t *testing.T, handler *ARPHandler, devices []config.Device) {
	t.Helper()
	client := net.HardwareAddr{2, 0, 0, 0, 0, 99}
	handler.handleARPRequest(&Packet{VLAN: 200}, &layers.ARP{
		SourceHwAddress: client, SourceProtAddress: net.ParseIP("10.0.0.99").To4(),
		DstProtAddress: net.ParseIP("10.0.0.3").To4(),
	})
	seen := make(map[string]bool)
	for _, reply := range isolationReplies(handler.stack) {
		packet := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		arp, ok := packet.Layer(layers.LayerTypeARP).(*layers.ARP)
		if !ok || arp.Operation != layers.ARPReply || reply.VLAN != 200 ||
			!net.IP(arp.SourceProtAddress).Equal(net.ParseIP("10.0.0.3")) ||
			net.HardwareAddr(arp.DstHwAddress).String() != client.String() {
			t.Fatal("wrong ARP reply identity or VLAN")
		}
		seen[net.HardwareAddr(arp.SourceHwAddress).String()] = true
	}
	if len(seen) != 2 || !seen[devices[0].MACAddress.String()] || !seen[devices[1].MACAddress.String()] {
		t.Fatalf("conflict reply identities = %v", seen)
	}
}
