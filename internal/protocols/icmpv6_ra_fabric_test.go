package protocols

import (
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// A router advertisement is link-scoped at the IP layer, not at the cable: it
// comes from the attachment network's router (the core SVI), which is behind
// the pool's access switch. A router with no interface on the attachment
// network is not on the tester's link and must stay silent (niac-go#2411).

const placementMgmtRouter = "MED-MGMT-RTR01"

func raOn(device *config.Device, ip string) {
	device.ICMPv6Config = &config.ICMPv6Config{
		Enabled:             true,
		RouterAdvertisement: &config.Icmpv6RouterAdvertisement{Period: 4},
	}
	device.IPAddresses = append(device.IPAddresses, net.ParseIP(ip))
}

// raPlacementStack is the placement fabric with an advertising core and a
// second advertising router that sits on the management network only.
func raPlacementStack(t *testing.T) *Stack {
	t.Helper()
	cfg := placementConfig()
	for index := range cfg.Devices {
		if cfg.Devices[index].Name == placementCore {
			raOn(&cfg.Devices[index], "fd00:51:d2::2")
		}
	}
	router := config.Device{
		Name:       placementMgmtRouter,
		Type:       "router",
		MACAddress: net.HardwareAddr{0x00, 0x51, 0x00, 0x00, 0x00, 0x03},
		Interfaces: []config.Interface{
			{Name: "GigabitEthernet0/0", Network: "med-mgmt", Address: "10.51.200.3/24"},
		},
	}
	raOn(&router, "fd00:51:c8::3")
	cfg.Devices = append(cfg.Devices, router)
	stack := NewStackWithTransport(idleTransport{}, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(compilePlacement(t, cfg))
	return stack
}

func queuedRouterAdvertisers(stack *Stack) []string {
	names := make(map[string]string)
	for _, device := range stack.AllDevices() {
		names[device.MACAddress.String()] = device.Name
	}
	var got []string
	for len(stack.sendQueue) > 0 {
		pkt := <-stack.sendQueue
		frame := gopacket.NewPacket(pkt.Buffer[:pkt.Length], layers.LayerTypeEthernet, gopacket.Default)
		if frame.Layer(layers.LayerTypeICMPv6RouterAdvertisement) == nil {
			continue
		}
		got = append(got, names[net.HardwareAddr(pkt.Buffer[6:12]).String()])
	}
	return got
}

func TestFabricRouterAdvertisementComesFromAttachmentRouters(t *testing.T) {
	stack := raPlacementStack(t)

	stack.icmpv6Handler.sendDueRouterAdvertisements(time.Now(), nil)

	if got, want := queuedRouterAdvertisers(stack), []string{placementCore}; !sameNames(got, want) {
		t.Errorf("unsolicited RAs from %v, want %v", got, want)
	}
}

func TestFabricRouterSolicitationIsAnsweredByAttachmentRouters(t *testing.T) {
	stack := raPlacementStack(t)
	client := placementClient(1)

	stack.decodePacket(routerSolicitationFrame(t, client))

	if got, want := queuedRouterAdvertisers(stack), []string{placementCore}; !sameNames(got, want) {
		t.Errorf("RS answered by %v, want %v", got, want)
	}
}

func routerSolicitationFrame(t *testing.T, src net.HardwareAddr) *Packet {
	t.Helper()
	eth := &layers.Ethernet{
		SrcMAC:       src,
		DstMAC:       net.HardwareAddr{0x33, 0x33, 0x00, 0x00, 0x00, 0x02},
		EthernetType: layers.EthernetTypeIPv6,
	}
	ip := &layers.IPv6{
		Version:    6,
		NextHeader: layers.IPProtocolICMPv6,
		HopLimit:   icmpv6NDPHopLimit,
		SrcIP:      net.ParseIP("fe80::ff:fe00:201"),
		DstIP:      net.ParseIP("ff02::2"),
	}
	icmp := &layers.ICMPv6{
		TypeCode: layers.CreateICMPv6TypeCode(layers.ICMPv6TypeRouterSolicitation, 0),
	}
	if err := icmp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("checksum layer: %v", err)
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, icmp, &layers.ICMPv6RouterSolicitation{}); err != nil {
		t.Fatalf("serialize RS: %v", err)
	}
	pkt, err := ParsePacket(buf.Bytes(), 1)
	if err != nil {
		t.Fatalf("ParsePacket: %v", err)
	}
	return pkt
}

func TestFabricRouterAdvertisementStopsWithEveryPoolPortShut(t *testing.T) {
	stack := raPlacementStack(t)
	for _, port := range placementPool() {
		shutPoolPort(t, stack, port)
	}

	stack.icmpv6Handler.sendDueRouterAdvertisements(time.Now(), nil)

	if got := queuedRouterAdvertisers(stack); len(got) != 0 {
		t.Errorf("unsolicited RAs from %v with no pool port left for the client", got)
	}
}
