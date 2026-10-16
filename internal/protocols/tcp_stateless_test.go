package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// serviceClientAddress is the address every test segment comes from.
const serviceClientAddress = "192.0.2.9"

func buildServiceSegment(
	t *testing.T,
	device *config.Device,
	port layers.TCPPort,
	segment *layers.TCP,
) (*Packet, *layers.IPv4) {
	t.Helper()
	eth := &layers.Ethernet{
		SrcMAC: net.HardwareAddr{0xaa, 0, 0, 0, 0, 1}, DstMAC: device.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, IHL: 5, TTL: 64, Protocol: layers.IPProtocolTCP,
		SrcIP: net.ParseIP(serviceClientAddress).To4(), DstIP: device.IPAddresses[0].To4(),
	}
	if segment.SrcPort == 0 {
		segment.SrcPort = 40000
	}
	segment.DstPort = port
	if err := segment.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatal(err)
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, segment, gopacket.Payload(segment.Payload)); err != nil {
		t.Fatal(err)
	}

	return &Packet{Buffer: buf.Bytes(), Length: len(buf.Bytes()), SerialNumber: 1}, ip
}

// serviceReply hands one segment for port to a fresh TCP handler and returns
// the TCP layer it queued, or nil when it queued nothing.
func serviceReply(t *testing.T, device *config.Device, port layers.TCPPort, segment *layers.TCP) *layers.TCP {
	t.Helper()

	return newServiceConversation(t, device, port).reply(segment)
}

// serviceConversation sends segments of one connection to the same handler.
type serviceConversation struct {
	t       *testing.T
	stack   *Stack
	handler *TCPHandler
	device  *config.Device
	port    layers.TCPPort
}

func newServiceConversation(t *testing.T, device *config.Device, port layers.TCPPort) *serviceConversation {
	t.Helper()
	stack := newTestStackInternal(t)
	stack.healthCheckHandler = NewHealthCheckHandler(stack)

	return &serviceConversation{t: t, stack: stack, handler: NewTCPHandler(stack), device: device, port: port}
}

func (c *serviceConversation) reply(segment *layers.TCP) *layers.TCP {
	c.t.Helper()
	pkt, ip := buildServiceSegment(c.t, c.device, c.port, segment)
	c.handler.routeToHandler(pkt, ip, segment, []*config.Device{c.device})

	return c.next()
}

// next returns the next TCP layer the handler queued, or nil when there is
// none.
func (c *serviceConversation) next() *layers.TCP {
	t, stack, device, port := c.t, c.stack, c.device, c.port
	t.Helper()
	select {
	case reply := <-stack.sendQueue:
		decoded := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		ipOut, _ := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		tcpOut, _ := decoded.Layer(layers.LayerTypeTCP).(*layers.TCP)
		if ipOut == nil || tcpOut == nil {
			t.Fatalf("reply is not TCP/IPv4: %v", decoded)
		}
		if !ipOut.SrcIP.Equal(device.IPAddresses[0]) || ipOut.DstIP.String() != serviceClientAddress {
			t.Errorf(
				"reply %s -> %s, want %s -> %s",
				ipOut.SrcIP,
				ipOut.DstIP,
				device.IPAddresses[0],
				serviceClientAddress,
			)
		}
		if tcpOut.SrcPort != port || tcpOut.DstPort != 40000 {
			t.Errorf("reply ports %d -> %d, want %d -> 40000", tcpOut.SrcPort, tcpOut.DstPort, port)
		}
		return tcpOut
	default:
		return nil
	}
}
