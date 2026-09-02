//go:build linux && integration

package wiretest

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// The pack binds to a routed fabric whose transit network is the physical wire.
// Devices inside the hospital site sit behind the edge router and are only
// reachable across a route, and discovery frames from them are dropped before
// egress by validateDiscoveryEgress. LAB-EDGE-R1 is the device actually on the
// attachment network, so it is the one observable from this end of the veth.
const (
	edgeRouterName = "LAB-EDGE-R1"
	transitGateway = "10.254.200.1"
	clientAddr     = "10.254.200.50" // below the authored DHCP pool, so it collides with nothing
	accessVLAN     = 200
)

// startHospital generates the hospital pack the way the product does and starts
// it on the simulated end of the wire. The generated YAML is the authored truth
// every assertion below reads, so the thing under test and the thing compared
// against are one artifact; a hand-written config here would be an oracle, not
// the product.
func startHospital(t *testing.T) *config.Config {
	t.Helper()
	requireWire(t)

	var pack scenario.Pack
	for _, candidate := range scenario.Packs() {
		if candidate.ID == "hospital" {
			pack = candidate
			break
		}
	}
	if pack.ID == "" {
		t.Fatal("no pack with id \"hospital\"; scenario.Packs() no longer ships it")
	}

	result, err := scenario.Generate(pack.Request)
	if err != nil {
		t.Fatalf("scenario.Generate(hospital): %v", err)
	}
	authored, err := config.LoadYAMLBytes(result.YAML)
	if err != nil {
		t.Fatalf("loading the generated hospital YAML: %v", err)
	}

	// Without this the daemon persists the inline config into the invoking
	// user's real ~/.niac/configs.
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())

	d, err := daemon.NewDaemon(daemon.Config{
		StoragePath: "disabled",
		AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
			Interface: simIface, Mode: fabric.ModeAccess, AccessVLAN: accessVLAN,
		}},
	})
	if err != nil {
		t.Fatalf("daemon.NewDaemon: %v", err)
	}
	if err := d.StartSimulation(api.SimulationRequest{
		SessionID:      "wiretest",
		Interface:      simIface,
		Attachment:     pack.Request.AttachmentName,
		AttachmentMode: fabric.ModeAccess,
		AccessVLAN:     accessVLAN,
		ConfigData:     string(result.YAML),
	}); err != nil {
		t.Fatalf("StartSimulation on %s: %v", simIface, err)
	}
	t.Cleanup(func() {
		if err := d.StopSimulation(""); err != nil {
			t.Errorf("StopSimulation: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = d.Shutdown(ctx)
	})
	return authored
}

// deviceByName returns the authored device or fails; a renamed device should
// report as exactly that rather than as a nil dereference.
func deviceByName(t *testing.T, cfg *config.Config, name string) *config.Device {
	t.Helper()
	for _, device := range cfg.Devices {
		if device.Name == name {
			return device
		}
	}
	t.Fatalf("no device named %q in the generated config", name)
	return nil
}

// openClient opens libpcap on the test end with immediate delivery, so a reply
// is readable as soon as it arrives instead of sitting in a buffer.
func openClient(t *testing.T) *pcap.Handle {
	t.Helper()
	inactive, err := pcap.NewInactiveHandle(testIface)
	if err != nil {
		t.Fatalf("pcap.NewInactiveHandle(%s): %v", testIface, err)
	}
	defer inactive.CleanUp()
	_ = inactive.SetSnapLen(65536)
	_ = inactive.SetPromisc(true)
	_ = inactive.SetTimeout(pcap.BlockForever)
	_ = inactive.SetImmediateMode(true)
	handle, err := inactive.Activate()
	if err != nil {
		t.Fatalf("activating pcap on %s: %v", testIface, err)
	}
	t.Cleanup(handle.Close)
	return handle
}

// clientMAC is the hardware address the kernel gave the test end of the veth.
func clientMAC(t *testing.T) net.HardwareAddr {
	t.Helper()
	iface, err := net.InterfaceByName(testIface)
	if err != nil {
		t.Fatalf("net.InterfaceByName(%s): %v", testIface, err)
	}
	return iface.HardwareAddr
}

func serialize(t *testing.T, ls ...gopacket.SerializableLayer) []byte {
	t.Helper()
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, ls...); err != nil {
		t.Fatalf("serializing frame: %v", err)
	}
	return buf.Bytes()
}

// ARP is the cheapest end-to-end proof that the simulation is on the wire: it
// exercises capture, device lookup and frame injection, and its answer carries
// an authored field — the device's MAC — that a count-based assertion would
// miss entirely.
func TestARPAnswersWithAuthoredEdgeRouterMAC(t *testing.T) {
	authored := startHospital(t)
	edge := deviceByName(t, authored, edgeRouterName)

	handle := openClient(t)
	src := clientMAC(t)

	eth := &layers.Ethernet{
		SrcMAC:       src,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}
	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   src,
		SourceProtAddress: net.ParseIP(clientAddr).To4(),
		DstHwAddress:      net.HardwareAddr{0, 0, 0, 0, 0, 0},
		DstProtAddress:    net.ParseIP(transitGateway).To4(),
	}
	frame := serialize(t, eth, arp)

	reply := awaitARPReply(t, handle, frame, net.ParseIP(transitGateway).To4())

	if got := net.HardwareAddr(reply.SourceHwAddress).String(); got != edge.MACAddress.String() {
		t.Errorf("ARP reply for %s came from MAC %s, want the authored %s MAC %s",
			transitGateway, got, edgeRouterName, edge.MACAddress)
	}
}

// awaitARPReply retransmits the request while waiting, because the responder's
// first advertisement and the pcap handle coming up race on a fresh interface.
func awaitARPReply(t *testing.T, handle *pcap.Handle, request []byte, want net.IP) *layers.ARP {
	t.Helper()
	source := gopacket.NewPacketSource(handle, handle.LinkType())
	packets := source.Packets()
	deadline := time.After(20 * time.Second)
	retry := time.NewTicker(500 * time.Millisecond)
	defer retry.Stop()

	if err := handle.WritePacketData(request); err != nil {
		t.Fatalf("writing ARP request: %v", err)
	}
	for {
		select {
		case packet := <-packets:
			layer := packet.Layer(layers.LayerTypeARP)
			if layer == nil {
				continue
			}
			arp, ok := layer.(*layers.ARP)
			if !ok || arp.Operation != layers.ARPReply {
				continue
			}
			if net.IP(arp.SourceProtAddress).Equal(want) {
				return arp
			}
		case <-retry.C:
			if err := handle.WritePacketData(request); err != nil {
				t.Fatalf("re-writing ARP request: %v", err)
			}
		case <-deadline:
			t.Fatalf("no ARP reply for %s within 20s; the simulation is not answering on %s", want, simIface)
		}
	}
}
