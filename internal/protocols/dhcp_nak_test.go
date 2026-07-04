package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func drainDHCP(t *testing.T, stack *Stack) *Packet {
	t.Helper()

	select {
	case pkt := <-stack.sendQueue:
		return pkt
	default:
		t.Fatal("no DHCP packet queued")

		return nil
	}
}

func dhcpOf(t *testing.T, pkt *Packet) *layers.DHCPv4 {
	t.Helper()

	p := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	d, ok := p.Layer(layers.LayerTypeDHCPv4).(*layers.DHCPv4)
	if !ok {
		t.Fatal("packet has no DHCPv4 layer")
	}

	return d
}

func dhcpMsgType(d *layers.DHCPv4) byte {
	for _, o := range d.Options {
		if o.Type == layers.DHCPOptMessageType && len(o.Data) > 0 {
			return o.Data[0]
		}
	}

	return 0
}

func newDHCPTestHandler(t *testing.T) (*Stack, *DHCPHandler, *config.Device) {
	t.Helper()

	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)
	h.SetPool(net.ParseIP("10.20.200.100"), net.ParseIP("10.20.200.199"))
	h.SetServerConfig(net.ParseIP("10.20.200.2"), net.ParseIP("10.20.200.1"), nil, "demo.lab")
	h.subnetMask = net.ParseIP("255.255.255.0")

	device := &config.Device{
		Name:        "dhcp-server",
		MACAddress:  net.HardwareAddr{0x02, 0x00, 0x14, 0x03, 0x00, 0x01},
		IPAddresses: []net.IP{net.ParseIP("10.20.200.2")},
	}

	return stack, h, device
}

func dhcpRequest(clientMAC net.HardwareAddr, requestedIP net.IP, vlan int) *dhcpPacketInfo {
	return &dhcpPacketInfo{
		dhcp: &layers.DHCPv4{
			Operation:    layers.DHCPOpRequest,
			Xid:          0x12345678,
			ClientHWAddr: clientMAC,
			Options: []layers.DHCPOption{
				{Type: layers.DHCPOptMessageType, Length: 1, Data: []byte{DHCPRequest}},
				{Type: layers.DHCPOptRequestIP, Length: 4, Data: requestedIP.To4()},
			},
		},
		messageType: DHCPRequest,
		vlan:        vlan,
	}
}

// TestBuildDHCPOptionsNak verifies a NAK carries only message-type, server-id,
// and a message — never lease parameters or a subnet mask (RFC 2131 §4.3.2).
func TestBuildDHCPOptionsNak(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)

	opts := h.buildDHCPOptions(
		DHCPNak,
		net.ParseIP("10.20.200.2"),
		net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
	)

	present := map[layers.DHCPOpt]bool{}
	for _, o := range opts {
		present[o.Type] = true
	}

	if !present[layers.DHCPOptServerID] {
		t.Error("NAK must carry the server identifier")
	}

	for _, forbidden := range []layers.DHCPOpt{
		layers.DHCPOptSubnetMask, layers.DHCPOptRouter, layers.DHCPOptDNS, layers.DHCPOptLeaseTime,
	} {
		if present[forbidden] {
			t.Errorf("NAK must not carry option %v", forbidden)
		}
	}
}

// TestSendDHCPNakWireShape verifies the NAK is broadcast on the request VLAN
// with a zero yiaddr and message-type NAK.
func TestSendDHCPNakWireShape(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)

	err := h.SendDHCPNak(0x12345678,
		net.HardwareAddr{0xaa, 0xaa, 0xaa, 0x00, 0x00, 0x01},
		net.ParseIP("10.20.200.2"),
		net.HardwareAddr{0x02, 0x00, 0x14, 0x03, 0x00, 0x01},
		200)
	if err != nil {
		t.Fatalf("SendDHCPNak: %v", err)
	}

	pkt := drainDHCP(t, stack)
	if pkt.VLAN != 200 {
		t.Errorf("NAK VLAN = %d, want 200", pkt.VLAN)
	}

	d := dhcpOf(t, pkt)
	if mt := dhcpMsgType(d); mt != DHCPNak {
		t.Errorf("message type = %d, want %d (NAK)", mt, DHCPNak)
	}

	if !d.YourClientIP.Equal(net.IPv4zero) {
		t.Errorf("NAK yiaddr = %s, want 0.0.0.0", d.YourClientIP)
	}
}

// TestHandleDHCPRequestNakOnWrongSubnet is the core regression: a REQUEST for an
// address outside the pool is NAKed, not silently ACKed for a substitute.
func TestHandleDHCPRequestNakOnWrongSubnet(t *testing.T) {
	stack, h, device := newDHCPTestHandler(t)

	info := dhcpRequest(net.HardwareAddr{0xaa, 0xaa, 0xaa, 0x00, 0x00, 0x01}, net.ParseIP("10.99.99.99"), 200)
	h.handleDHCPRequest(info, device, 1, 0)

	d := dhcpOf(t, drainDHCP(t, stack))
	if mt := dhcpMsgType(d); mt != DHCPNak {
		t.Errorf("wrong-subnet REQUEST got message type %d, want %d (NAK)", mt, DHCPNak)
	}
}

// TestHandleDHCPRequestAckOnValidRequest verifies a REQUEST for an in-pool free
// address still ACKs (the fix must not NAK legitimate requests).
func TestHandleDHCPRequestAckOnValidRequest(t *testing.T) {
	stack, h, device := newDHCPTestHandler(t)

	info := dhcpRequest(net.HardwareAddr{0xaa, 0xaa, 0xaa, 0x00, 0x00, 0x02}, net.ParseIP("10.20.200.150"), 200)
	h.handleDHCPRequest(info, device, 1, 0)

	d := dhcpOf(t, drainDHCP(t, stack))
	if mt := dhcpMsgType(d); mt != DHCPAck {
		t.Errorf("valid REQUEST got message type %d, want %d (ACK)", mt, DHCPAck)
	}

	if !d.YourClientIP.Equal(net.ParseIP("10.20.200.150")) {
		t.Errorf("ACK yiaddr = %s, want the requested 10.20.200.150", d.YourClientIP)
	}
}

func TestCanGrantRequestedIP(t *testing.T) {
	_, h, _ := newDHCPTestHandler(t)
	mac := net.HardwareAddr{0xaa, 0xaa, 0xaa, 0x00, 0x00, 0x03}

	if h.canGrantRequestedIP(mac, net.ParseIP("10.20.200.150")) != true {
		t.Error("in-pool free address should be grantable")
	}

	if h.canGrantRequestedIP(mac, net.ParseIP("10.99.99.99")) != false {
		t.Error("out-of-pool address must not be grantable")
	}
}
