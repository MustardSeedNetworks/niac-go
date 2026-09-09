package protocols

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// The positive case comes first in every test below: a suppressed answer is
// only evidence of the fault if the same request is answered without it.
func TestDHCPNoOfferFaultSuppressesTheOffer(t *testing.T) {
	stack, handler, device := newFaultedDHCPHandler(t)
	discover := dhcpDiscover(net.HardwareAddr{0x02, 0, 0, 0, 0, 0x11})

	handler.handleDHCPDiscover(discover, device, 1, 0)
	offer := drainDHCP(t, stack)
	if got := dhcpMsgType(dhcpOf(t, offer)); got != DHCPOffer {
		t.Fatalf("healthy server message type = %d, want DHCPOFFER", got)
	}

	if err := stack.SetDeviceFault(
		device.Name, devicestate.FaultDHCPNoOffer, 1,
	); err != nil {
		t.Fatalf("SetDeviceFault() error = %v", err)
	}

	handler.handleDHCPDiscover(
		dhcpDiscover(net.HardwareAddr{0x02, 0, 0, 0, 0, 0x12}), device, 2, 0)
	select {
	case pkt := <-stack.sendQueue:
		t.Fatalf("faulted server still answered: %#v", dhcpOf(t, pkt))
	default:
	}

	if err := stack.SetDeviceFault(
		device.Name, devicestate.FaultDHCPNoOffer, 0,
	); err != nil {
		t.Fatalf("SetDeviceFault(clear) error = %v", err)
	}
	handler.handleDHCPDiscover(
		dhcpDiscover(net.HardwareAddr{0x02, 0, 0, 0, 0, 0x13}), device, 3, 0)
	if got := dhcpMsgType(dhcpOf(t, drainDHCP(t, stack))); got != DHCPOffer {
		t.Fatalf("cleared fault message type = %d, want DHCPOFFER", got)
	}
}

func TestDNSFaultsChangeTheAnswer(t *testing.T) {
	stack, handler, device := newFaultedDNSHandler(t)
	query := &layers.DNS{
		ID: 0x4242, RD: true,
		Questions: []layers.DNSQuestion{{
			Name: []byte("host.example."), Type: layers.DNSTypeA, Class: layers.DNSClassIN,
		}},
	}

	healthy := handler.buildDNSResponse(query, device, 0, 1)
	if healthy.ResponseCode != layers.DNSResponseCodeNoErr || len(healthy.Answers) != 1 {
		t.Fatalf("healthy response = rcode %v, %d answers",
			healthy.ResponseCode, len(healthy.Answers))
	}

	if err := stack.SetDeviceFault(
		device.Name, devicestate.FaultDNSNXDomain, 1,
	); err != nil {
		t.Fatalf("SetDeviceFault(nxdomain) error = %v", err)
	}

	faulted := handler.buildDNSResponse(query, device, 0, 2)
	if faulted.ResponseCode != layers.DNSResponseCodeNXDomain {
		t.Fatalf("faulted rcode = %v, want NXDomain", faulted.ResponseCode)
	}
	if len(faulted.Answers) != 0 {
		t.Fatalf("faulted answers = %#v, want none", faulted.Answers)
	}
	if !faulted.AA || faulted.ID != query.ID || len(faulted.Questions) != 1 {
		t.Fatalf("faulted response stopped looking like this server: %#v", faulted)
	}

	if handler.dnsQuerySilenced(device) {
		t.Fatal("an NXDOMAIN fault also silenced the server")
	}
	if err := stack.SetDeviceFault(
		device.Name, devicestate.FaultDNSTimeout, 1,
	); err != nil {
		t.Fatalf("SetDeviceFault(timeout) error = %v", err)
	}
	if !handler.dnsQuerySilenced(device) {
		t.Fatal("a timeout fault did not silence the server")
	}
}

func newFaultedDHCPHandler(t *testing.T) (*Stack, *DHCPHandler, *config.Device) {
	t.Helper()

	cfg := &config.Config{Devices: []config.Device{{
		Name:        "dhcp-server",
		MACAddress:  net.HardwareAddr{0x02, 0x00, 0x14, 0x03, 0x00, 0x01},
		IPAddresses: []net.IP{net.ParseIP("10.20.200.2")},
		Interfaces:  []config.Interface{{Name: "Gi0/1", Address: "10.20.200.2/24", Speed: 100}},
		DHCPConfig: &config.DHCPConfig{
			PoolStart: net.ParseIP("10.20.200.100"), PoolEnd: net.ParseIP("10.20.200.199"),
		},
		SNMPConfig: config.SNMPConfig{Community: "public"},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDHCPHandler(stack)
	handler.SetPool(net.ParseIP("10.20.200.100"), net.ParseIP("10.20.200.199"))
	handler.SetServerConfig(
		net.ParseIP("10.20.200.2"), net.ParseIP("10.20.200.1"), nil, "demo.lab")
	handler.subnetMask = net.ParseIP("255.255.255.0")

	return stack, handler, &cfg.Devices[0]
}

func newFaultedDNSHandler(t *testing.T) (*Stack, *DNSHandler, *config.Device) {
	t.Helper()

	cfg := &config.Config{Devices: []config.Device{{
		Name:        "dns-server",
		MACAddress:  net.HardwareAddr{0x02, 0x00, 0x14, 0x03, 0x00, 0x02},
		IPAddresses: []net.IP{net.ParseIP("10.20.200.3")},
		Interfaces:  []config.Interface{{Name: "Gi0/1", Address: "10.20.200.3/24", Speed: 100}},
		DNSConfig: &config.DNSConfig{ForwardRecords: []config.DNSRecord{
			{Name: "host.example.", IP: net.ParseIP("10.20.200.50"), TTL: 300},
		}},
		SNMPConfig: config.SNMPConfig{Community: "public"},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)
	device := &cfg.Devices[0]
	handler.LoadDeviceDNSConfig(device)

	return stack, handler, device
}

func dhcpDiscover(clientMAC net.HardwareAddr) *dhcpPacketInfo {
	return &dhcpPacketInfo{
		dhcp: &layers.DHCPv4{
			Operation:    layers.DHCPOpRequest,
			Xid:          0x12345678,
			ClientHWAddr: clientMAC,
			Options: []layers.DHCPOption{
				{Type: layers.DHCPOptMessageType, Length: 1, Data: []byte{DHCPDiscover}},
			},
		},
		messageType: DHCPDiscover,
	}
}

// Latency is the one device fault with nothing to suppress: the device still
// answers, just late. So the assertion is a schedule, not a silence -- no
// reply the instant the handler returns, a reply after the armed delay, and
// an immediate reply again once it is cleared.
func TestLatencyFaultDefersTheEchoReply(t *testing.T) {
	// Generous because the flake budget is zero: the work between the handler
	// returning and the "not yet" check below is microseconds, so the delay
	// only has to survive a scheduler stall on a loaded runner.
	const delay = 250 * time.Millisecond

	stack, handler, device := newFaultedICMPHandler(t)
	devices := []*config.Device{device}
	request, ipLayer := echoRequest(device, 1)

	handler.HandlePacket(request, ipLayer, devices)
	assertEchoReply(t, mustEchoReply(t, stack), 1)

	if err := stack.SetDeviceFault(
		device.Name, devicestate.FaultLatency, int(delay.Milliseconds()),
	); err != nil {
		t.Fatalf("SetDeviceFault() error = %v", err)
	}

	faulted, faultedIP := echoRequest(device, 2)
	start := time.Now()
	handler.HandlePacket(faulted, faultedIP, devices)
	select {
	case pkt := <-stack.sendQueue:
		t.Fatalf("latency fault did not defer the reply (sn=%d)", pkt.SerialNumber)
	default:
	}

	// The reply is serialized before the delay starts, so the request's
	// capture buffer -- reused on the next read -- may be gone by the time it
	// is sent. Scribbling over it holds a later change to deferring the build
	// to that same contract.
	clear(faulted.Buffer)

	reply := mustEchoReply(t, stack)
	if elapsed := time.Since(start); elapsed < delay {
		t.Fatalf("reply arrived after %s, want at least %s", elapsed, delay)
	}
	assertEchoReply(t, reply, 2)

	if err := stack.SetDeviceFault(device.Name, devicestate.FaultLatency, 0); err != nil {
		t.Fatalf("SetDeviceFault(clear) error = %v", err)
	}
	cleared, clearedIP := echoRequest(device, 3)
	handler.HandlePacket(cleared, clearedIP, devices)
	select {
	case pkt := <-stack.sendQueue:
		assertEchoReply(t, pkt, 3)
	default:
		t.Fatal("cleared fault still deferred the reply")
	}
}

// echoRequestMAC is the tester's own MAC, and so the address every reply must
// be addressed back to.
func echoRequestMAC() net.HardwareAddr {
	return net.HardwareAddr{0x02, 0x00, 0x14, 0x03, 0x00, 0x50}
}

func newFaultedICMPHandler(t *testing.T) (*Stack, *ICMPHandler, *config.Device) {
	t.Helper()

	cfg := &config.Config{Devices: []config.Device{{
		Name:        "edge-1",
		MACAddress:  net.HardwareAddr{0x02, 0x00, 0x14, 0x03, 0x00, 0x03},
		IPAddresses: []net.IP{net.ParseIP("10.20.200.4")},
		Interfaces:  []config.Interface{{Name: "Gi0/1", Address: "10.20.200.4/24", Speed: 100}},
		SNMPConfig:  config.SNMPConfig{Community: "public"},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))

	return stack, NewICMPHandler(stack), &cfg.Devices[0]
}

// echoRequest builds one ICMP echo request for device, using seq as both the
// sequence number and the payload so a reply can be traced to its request.
func echoRequest(device *config.Device, seq uint16) (*Packet, *layers.IPv4) {
	ipLayer := &layers.IPv4{
		Version: 4, IHL: 5, TTL: 64,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    net.ParseIP("10.20.200.50").To4(),
		DstIP:    device.IPAddresses[0].To4(),
	}
	buffer := gopacket.NewSerializeBuffer()
	_ = gopacket.SerializeLayers(buffer,
		gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		&layers.Ethernet{
			SrcMAC:       echoRequestMAC(),
			DstMAC:       device.MACAddress,
			EthernetType: layers.EthernetTypeIPv4,
		},
		ipLayer,
		&layers.ICMPv4{
			TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
			Id:       0x4242, Seq: seq,
		},
		gopacket.Payload([]byte{byte(seq)}),
	)

	return &Packet{
		Buffer: buffer.Bytes(), Length: len(buffer.Bytes()), SerialNumber: int(seq),
	}, ipLayer
}

func mustEchoReply(t *testing.T, stack *Stack) *Packet {
	t.Helper()

	select {
	case pkt := <-stack.sendQueue:
		return pkt
	case <-time.After(5 * time.Second):
		t.Fatal("no ICMP echo reply queued")

		return nil
	}
}

func assertEchoReply(t *testing.T, pkt *Packet, seq uint16) {
	t.Helper()

	parsed := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ethernet, ok := parsed.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if !ok {
		t.Fatalf("queued packet carries no Ethernet layer: % x", pkt.Buffer)
	}
	// The addresses are read straight out of the request's capture buffer, so
	// they are what a reply that failed to own its bytes gets wrong.
	if !bytes.Equal(ethernet.DstMAC, echoRequestMAC()) {
		t.Fatalf("reply destination MAC = %s, want %s", ethernet.DstMAC, echoRequestMAC())
	}

	layer := parsed.Layer(layers.LayerTypeICMPv4)
	if layer == nil {
		t.Fatalf("queued packet carries no ICMP layer: % x", pkt.Buffer)
	}
	icmp, ok := layer.(*layers.ICMPv4)
	if !ok {
		t.Fatalf("ICMP layer type = %T", layer)
	}
	if icmp.TypeCode.Type() != layers.ICMPv4TypeEchoReply {
		t.Fatalf("ICMP type = %d, want echo reply", icmp.TypeCode.Type())
	}
	if icmp.Seq != seq || icmp.Id != 0x4242 {
		t.Fatalf("reply id=%d seq=%d, want id=16962 seq=%d", icmp.Id, icmp.Seq, seq)
	}
	if len(icmp.Payload) != 1 || icmp.Payload[0] != byte(seq) {
		t.Fatalf("reply payload = % x, want %02x", icmp.Payload, seq)
	}
}
