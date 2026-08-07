package protocols

import (
	"context"
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// Well-known UDP ports.
const (
	UDPPortDNS    = 53
	UDPPortDHCP   = 67
	UDPPortDHCPC  = 68
	UDPPortSNMP   = 161
	UDPPortDHCPv6 = 547
)

// UDP internal constants.
const (
	udpBroadcastOctet = 255  // Broadcast IP octet (255.255.255.255)
	udpProxyTimeoutS  = 30   // Proxy timeout in seconds
	udpProxyBufSize   = 1500 // Proxy buffer size (MTU)
	// A fixed process-local limit prevents map_to_ip bursts from exhausting
	// sockets and goroutines while keeping independent devices responsive.
	udpProxyConcurrencyLimit = 64
)

// IP protocol constants for UDP handler.
const (
	udpIPv4Version = 4  // IPv4 version field
	udpIPv4IHL     = 5  // IPv4 header length (5 = 20 bytes)
	udpIPv4TTL     = 64 // IPv4 default TTL
)

// NetAlly UDP reflector constants. The reflector echoes probes whose payload
// carries one of the signatures below, starting at reflectorSigOffset bytes
// into the UDP payload (matching niac-java Udp.reflect: signatureIdx =
// udpDataIdx + 5).
const (
	reflectorSigOffset = 5 // Signature starts 5 bytes into the UDP payload

	// reflectorWiggleIPPrec toggles the IP-precedence bit; reflectorWiggleDSCP
	// toggles the bottom two DSCP bits. The reflected packet's ToS is XORed
	// with one of these so the tester can confirm the round trip preserved (or
	// deliberately altered) DiffServ marking.
	reflectorWiggleIPPrec = 0x01
	reflectorWiggleDSCP   = 0x03

	// reflectorSigData and reflectorSigProbe are the 8-byte magic strings that
	// mark a reflector probe. A payload matching either at reflectorSigOffset is
	// echoed back.
	reflectorSigData  = "DATA:OT\x00"
	reflectorSigProbe = "PROBEOT\x00"
)

// UDPHandler handles UDP packets.
type UDPHandler struct {
	stack       *Stack
	proxyCtx    context.Context
	cancelProxy context.CancelFunc
	proxySlots  chan struct{}
	proxyWG     sync.WaitGroup
	proxyMu     sync.Mutex
	proxyConns  map[net.Conn]struct{}
	stopped     bool
	dial        func(context.Context, string) (net.Conn, error)
}

// NewUDPHandler creates a new UDP handler.
func NewUDPHandler(stack *Stack) *UDPHandler {
	ctx, cancel := context.WithCancel(context.Background())
	dialer := &net.Dialer{}
	return &UDPHandler{
		stack:       stack,
		proxyCtx:    ctx,
		cancelProxy: cancel,
		proxySlots:  make(chan struct{}, udpProxyConcurrencyLimit),
		proxyConns:  make(map[net.Conn]struct{}),
		dial: func(ctx context.Context, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, "udp", address)
		},
	}
}

// HandlePacket processes a UDP packet.
func (h *UDPHandler) HandlePacket(pkt *Packet, ipLayer *layers.IPv4, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse UDP layer
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "UDP packet missing UDP layer sn=%d\n", pkt.SerialNumber)
		}

		return
	}

	udp, ok := udpLayer.(*layers.UDP)
	if !ok {
		return
	}

	if debugLevel >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "UDP packet: %s:%d -> %s:%d length=%d sn=%d\n",
			ipLayer.SrcIP, udp.SrcPort, ipLayer.DstIP, udp.DstPort, len(udp.Payload), pkt.SerialNumber)
	}

	// MapToIP handling (UDP proxy)
	if !ipLayer.DstIP.Equal(net.IPv4(udpBroadcastOctet, udpBroadcastOctet, udpBroadcastOctet, udpBroadcastOctet)) {
		for _, device := range devices {
			if device.MapToIP != nil {
				h.proxyToMap(device, ipLayer, udp, pkt)

				return
			}
		}
	}

	// Route to application handler based on port
	switch udp.DstPort {
	case UDPPortDNS:
		// DNS query
		h.stack.dnsHandler.HandleQuery(pkt, ipLayer, udp, devices)
	case UDPPortDHCP:
		// DHCP server port
		if h.stack.allowDHCP() {
			h.stack.dhcpHandler.HandlePacket(pkt, ipLayer, udp, devices)
		}
	case UDPPortSNMP:
		h.handleSNMP(pkt, ipLayer, udp, devices)
	case MDNSPort:
		// Multicast DNS (Bonjour/Avahi)
		h.stack.mdnsHandler.HandleQuery(pkt, ipLayer, udp, devices, packet)
	case NetBIOSNameServicePort:
		// NetBIOS Name Service
		h.stack.netbiosHandler.HandleNameService(pkt, packet, udp, devices)
	case NetBIOSDatagramServicePort:
		// NetBIOS Datagram Service
		h.stack.netbiosHandler.HandleDatagramService(pkt, packet, udp, devices)
	default:
		// A NetAlly reflector probe arrives on an arbitrary UDP port and is
		// identified by its payload signature, not the port — so reflection
		// is attempted here rather than via a fixed case.
		if h.tryReflect(pkt, ipLayer, udp, devices) {
			return
		}

		if debugLevel >= DebugLevelVerbose {
			_, _ = fmt.Fprintf(os.Stdout, "UDP packet to unhandled port %d sn=%d\n", udp.DstPort, pkt.SerialNumber)
		}

		h.sendPortUnreachable(pkt, ipLayer, udp, devices)
	}
}

// sendPortUnreachable replies ICMP Destination Unreachable / port unreachable
// from the addressed device — the RFC 792 behaviour of a closed UDP port. This
// is what lets a UDP path analysis / traceroute confirm it reached the target
// (its intermediate hops already answer via the TTL Time Exceeded path).
func (h *UDPHandler) sendPortUnreachable(
	pkt *Packet,
	ipLayer *layers.IPv4,
	udp *layers.UDP,
	devices []*config.Device,
) {
	h.stack.recordUDPNoPort(devices)
	if h.stack.icmpHandler == nil || len(devices) == 0 {
		return
	}

	device := devices[0]
	// A device that forwards this port (MapToIP) isn't a closed port.
	if device.MapToIP != nil || len(device.MACAddress) == 0 {
		return
	}

	identity, ok := h.stack.replyEthernet(pkt, device)
	if !ok {
		return
	}

	// RFC 792 §3.2: quote the offending IP header + 8 bytes of its datagram.
	original := append(ipLayer.LayerContents(), udp.LayerContents()...)

	_ = h.stack.icmpHandler.SendICMPUnreachable(
		ipLayer.DstIP, ipLayer.SrcIP,
		identity.source, identity.destination,
		layers.ICMPv4CodePort,
		original,
		identity.vlan,
	)
}

func (h *UDPHandler) handleSNMP(pkt *Packet, ipLayer *layers.IPv4, udp *layers.UDP, devices []*config.Device) {
	if h.stack.snmpHandler == nil {
		if h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "SNMP handler not initialised sn=%d\n", pkt.SerialNumber)
		}

		return
	}

	h.stack.snmpHandler.HandlePacket(pkt, ipLayer, udp, devices)
}

func (h *UDPHandler) proxyToMap(device *config.Device, ipLayer *layers.IPv4, udp *layers.UDP, pkt *Packet) {
	if device == nil || device.MapToIP == nil {
		return
	}
	identity, ok := h.stack.replyEthernet(pkt, device)
	if !ok {
		return
	}

	srcIP := ipLayer.DstIP.To4()
	dstIP := ipLayer.SrcIP.To4()
	if srcIP == nil || dstIP == nil {
		return
	}

	request := udpProxyRequest{
		address: net.JoinHostPort(device.MapToIP.String(), strconv.Itoa(int(udp.DstPort))),
		payload: slices.Clone(udp.Payload),
		srcIP:   slices.Clone(srcIP),
		dstIP:   slices.Clone(dstIP),
		srcPort: uint16(udp.DstPort),
		dstPort: uint16(udp.SrcPort),
		srcMAC:  slices.Clone(identity.source),
		dstMAC:  slices.Clone(identity.destination),
		vlan:    identity.vlan,
	}
	h.launchProxy(func(ctx context.Context) {
		h.runProxy(ctx, request)
	})
}

type udpProxyRequest struct {
	address string
	payload []byte
	srcIP   []byte
	dstIP   []byte
	srcPort uint16
	dstPort uint16
	srcMAC  []byte
	dstMAC  []byte
	vlan    int
}

func (h *UDPHandler) launchProxy(work func(context.Context)) bool {
	h.proxyMu.Lock()
	if h.stopped {
		h.proxyMu.Unlock()
		return false
	}
	select {
	case h.proxySlots <- struct{}{}:
		h.proxyWG.Add(1)
		h.proxyMu.Unlock()
	default:
		h.proxyMu.Unlock()
		h.stack.recordUDPProxyOverloadDrop()
		return false
	}

	go func() {
		defer func() {
			<-h.proxySlots
			h.proxyWG.Done()
		}()
		work(h.proxyCtx)
	}()
	return true
}

func (h *UDPHandler) runProxy(ctx context.Context, request udpProxyRequest) {
	conn, err := h.dial(ctx, request.address)
	if err != nil || !h.trackProxyConn(conn) {
		return
	}
	defer func() {
		h.untrackProxyConn(conn)
		_ = conn.Close()
	}()

	_ = conn.SetDeadline(time.Now().Add(udpProxyTimeoutS * time.Second))
	if _, err = conn.Write(request.payload); err != nil {
		return
	}

	buf := make([]byte, udpProxyBufSize)
	n, err := conn.Read(buf)
	if err != nil || n <= 0 {
		return
	}

	_ = h.SendUDP(
		request.srcIP,
		request.dstIP,
		request.srcPort,
		request.dstPort,
		buf[:n],
		request.srcMAC,
		request.dstMAC,
		request.vlan,
	)
}

func (h *UDPHandler) trackProxyConn(conn net.Conn) bool {
	h.proxyMu.Lock()
	defer h.proxyMu.Unlock()
	if h.stopped {
		_ = conn.Close()
		return false
	}
	h.proxyConns[conn] = struct{}{}
	return true
}

func (h *UDPHandler) untrackProxyConn(conn net.Conn) {
	h.proxyMu.Lock()
	delete(h.proxyConns, conn)
	h.proxyMu.Unlock()
}

// Stop cancels proxy dialing, closes in-flight sockets, and waits for all
// admitted work to release its slot.
func (h *UDPHandler) Stop() {
	h.proxyMu.Lock()
	if h.stopped {
		h.proxyMu.Unlock()
		return
	}
	h.stopped = true
	h.cancelProxy()
	conns := make([]net.Conn, 0, len(h.proxyConns))
	for conn := range h.proxyConns {
		conns = append(conns, conn)
	}
	h.proxyMu.Unlock()

	for _, conn := range conns {
		_ = conn.Close()
	}
	h.proxyWG.Wait()
}

// SendUDP sends a UDP packet with a default (zero) ToS byte.
func (h *UDPHandler) SendUDP(
	srcIP, dstIP []byte,
	srcPort, dstPort uint16,
	payload []byte,
	srcMAC, dstMAC []byte,
	vlan int,
) error {
	return h.sendUDPWithTOS(srcIP, dstIP, srcPort, dstPort, payload, srcMAC, dstMAC, vlan, 0)
}

// sendUDPWithTOS sends a UDP packet, stamping the IPv4 ToS byte. tos lets the
// reflector path preserve/alter DiffServ marking; other callers pass 0.
func (h *UDPHandler) sendUDPWithTOS(
	srcIP, dstIP []byte,
	srcPort, dstPort uint16,
	payload []byte,
	srcMAC, dstMAC []byte,
	vlan int,
	tos uint8,
) error {
	// Build Ethernet header
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	// Build IP header
	ipLayer := &layers.IPv4{
		Version:  udpIPv4Version,
		IHL:      udpIPv4IHL,
		TOS:      tos,
		TTL:      udpIPv4TTL,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}

	// Build UDP header
	udpLayer := &layers.UDP{
		SrcPort: layers.UDPPort(srcPort),
		DstPort: layers.UDPPort(dstPort),
	}
	_ = udpLayer.SetNetworkLayerForChecksum(ipLayer) // error is non-critical for simulation

	// Serialize
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts,
		eth,
		ipLayer,
		udpLayer,
		gopacket.Payload(payload),
	)
	if err != nil {
		return fmt.Errorf("error serializing UDP packet: %w", err)
	}

	// Get serial number
	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	// Create and send packet
	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: serialNum,
		VLAN:         vlan, // reply on the VLAN the request arrived on
	}

	h.stack.Send(pkt)

	if h.stack.GetDebugLevel() >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "Sent UDP packet: %s:%d -> %s:%d length=%d sn=%d\n",
			srcIP, srcPort, dstIP, dstPort, len(payload), serialNum)
	}

	return nil
}

// tryReflect echoes a NetAlly reflector probe back to its sender. It returns
// true when the packet was a reflector probe destined for a reflector-enabled
// device (and was reflected), false otherwise so the caller can fall through to
// normal unhandled-port logging.
//
// A reflected packet swaps source/destination MAC and IP but — matching
// niac-java Udp.reflect — leaves the UDP ports untouched; the tester matches
// the reply by its payload signature, not the 4-tuple. The IPv4 ToS byte is
// wiggled so the round trip's DiffServ handling is observable, and the reply is
// delayed by the device's configured latency +/- jitter.
func (h *UDPHandler) tryReflect(pkt *Packet, ipLayer *layers.IPv4, udp *layers.UDP, devices []*config.Device) bool {
	device := h.reflectorForIP(devices, ipLayer.DstIP)
	if device == nil {
		return false
	}

	if !reflectorSignatureMatch(udp.Payload) {
		return false
	}

	identity, ok := h.stack.replyEthernet(pkt, device)
	if !ok {
		return false
	}

	srcIP := ipLayer.DstIP.To4() // reflector answers as itself
	dstIP := ipLayer.SrcIP.To4() // back to the tester

	if srcIP == nil || dstIP == nil {
		return false
	}

	tos := wiggleTOS(ipLayer.TOS, device.ReflectorConfig.DSCP)

	// Own the payload bytes: the capture buffer is reused, and reflection may
	// be deferred onto a timer goroutine.
	payload := append([]byte(nil), udp.Payload...)
	srcPort := uint16(udp.SrcPort)
	dstPort := uint16(udp.DstPort)

	send := func() {
		_ = h.sendUDPWithTOS(
			srcIP, dstIP, srcPort, dstPort, payload,
			identity.source, identity.destination, identity.vlan, tos,
		)
	}

	if delay := reflectorDelay(device.ReflectorConfig); delay > 0 {
		time.AfterFunc(delay, send)
	} else {
		send()
	}

	if h.stack.GetDebugLevel() >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "Reflected UDP probe %s:%d -> %s (sn=%d)\n",
			ipLayer.SrcIP, udp.SrcPort, ipLayer.DstIP, pkt.SerialNumber)
	}

	return true
}

// reflectorForIP returns the first reflector-enabled device that owns ip, or
// nil if none does.
func (h *UDPHandler) reflectorForIP(devices []*config.Device, ip net.IP) *config.Device {
	for _, device := range devices {
		if device.ReflectorConfig != nil && h.stack.deviceOwnsIPv4(device, ip) {
			return device
		}
	}

	return nil
}

// reflectorSignatureMatch reports whether payload carries a reflector signature
// at reflectorSigOffset.
func reflectorSignatureMatch(payload []byte) bool {
	return slices.ContainsFunc([]string{reflectorSigData, reflectorSigProbe}, func(sig string) bool {
		return len(payload) >= reflectorSigOffset+len(sig) &&
			string(payload[reflectorSigOffset:reflectorSigOffset+len(sig)]) == sig
	})
}

// wiggleTOS toggles the IP-precedence bit (or the bottom two DSCP bits when
// dscp is set) of a ToS byte, matching niac-java's reflector wiggle.
func wiggleTOS(tos uint8, dscp bool) uint8 {
	wiggle := uint8(reflectorWiggleIPPrec)
	if dscp {
		wiggle = reflectorWiggleDSCP
	}

	return tos ^ wiggle
}

// reflectorDelay returns the send delay for a reflected packet: the configured
// latency, randomised by +/- jitter, floored at zero.
func reflectorDelay(cfg *config.ReflectorConfig) time.Duration {
	if cfg.LatencyMs <= 0 && cfg.JitterMs <= 0 {
		return 0
	}

	ms := cfg.LatencyMs
	if cfg.JitterMs > 0 {
		ms += getSimRand().IntN(2*cfg.JitterMs+1) - cfg.JitterMs
	}

	ms = max(ms, 0)

	return time.Duration(ms) * time.Millisecond
}

// HandlePacketV6 processes a UDP packet over IPv6.
func (h *UDPHandler) HandlePacketV6(pkt *Packet, packet gopacket.Packet, ipv6 *layers.IPv6, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse UDP layer
	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "UDP/IPv6 packet missing UDP layer sn=%d\n", pkt.SerialNumber)
		}

		return
	}

	udp, ok := udpLayer.(*layers.UDP)
	if !ok {
		return
	}

	if debugLevel >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "UDP/IPv6 packet: [%s]:%d -> [%s]:%d length=%d sn=%d\n",
			ipv6.SrcIP, udp.SrcPort, ipv6.DstIP, udp.DstPort, len(udp.Payload), pkt.SerialNumber)
	}

	// Route to application handler based on port
	switch udp.DstPort {
	case UDPPortDNS:
		// DNS query over IPv6
		h.stack.dnsHandler.HandleQueryV6(pkt, packet, ipv6, udp, devices)
	case UDPPortSNMP:
		if h.stack.snmpHandler != nil && h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= 2 {
			_, _ = fmt.Fprintf(os.Stdout, "SNMP/IPv6 query received (not yet implemented) sn=%d\n", pkt.SerialNumber)
		}
	case UDPPortDHCPv6:
		// DHCPv6 server port
		h.stack.dhcpv6Handler.HandlePacket(pkt, ipv6, udp, devices)
	case NetBIOSNameServicePort:
		// NetBIOS Name Service over IPv6
		h.stack.netbiosHandler.HandleNameService(pkt, packet, udp, devices)
	case NetBIOSDatagramServicePort:
		// NetBIOS Datagram Service over IPv6
		h.stack.netbiosHandler.HandleDatagramService(pkt, packet, udp, devices)
	default:
		if debugLevel >= DebugLevelVerbose {
			_, _ = fmt.Fprintf(os.Stdout, "UDP/IPv6 packet to unhandled port %d sn=%d\n", udp.DstPort, pkt.SerialNumber)
		}
	}
}
