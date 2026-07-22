package protocols

import (
	"fmt"
	"net"
	"os"
	"slices"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// Well-known TCP ports.
const (
	TCPPortFTP    = 21
	TCPPortSSH    = 22
	TCPPortTelnet = 23
	TCPPortHTTP   = 80
	TCPPortHTTPS  = 443
)

// TCP flag masks.
const (
	tcpFlagSYN = 0x02 // SYN flag mask
	tcpFlagACK = 0x10 // ACK flag mask
	tcpFlagFIN = 0x01 // FIN flag mask
	tcpFlagRST = 0x04 // RST flag mask
	tcpFlagPSH = 0x08 // PSH flag mask
)

// IP protocol constants for TCP handler.
const (
	tcpIPv4Version  = 4     // IPv4 version field
	tcpIPv4IHL      = 5     // IPv4 header length (5 = 20 bytes)
	tcpIPv4TTL      = 64    // IPv4 default TTL
	tcpIPv6Version  = 6     // IPv6 version field
	tcpIPv6HopLimit = 64    // IPv6 default hop limit
	tcpWindowSize   = 65535 // Default TCP window size
	tcpInitialSeq   = 1000  // Initial TCP sequence number
)

// TCPHandler handles TCP packets.
type TCPHandler struct {
	stack *Stack
	ssh   *sshTCPHandler
}

// NewTCPHandler creates a new TCP handler.
func NewTCPHandler(stack *Stack) *TCPHandler {
	handler := &TCPHandler{stack: stack}
	handler.ssh = newSSHTCPHandler(stack)
	return handler
}

// HandlePacket processes a TCP packet over IPv4.
func (h *TCPHandler) HandlePacket(pkt *Packet, ipLayer *layers.IPv4, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	tcp, ok := h.parseTCPLayer(pkt, debugLevel)
	if !ok {
		return
	}

	h.logTCPPacket(ipLayer.SrcIP, ipLayer.DstIP, tcp, debugLevel, pkt.SerialNumber)
	h.routeToHandler(pkt, ipLayer, tcp, devices)
}

// parseTCPLayer extracts the TCP layer from the packet.
func (h *TCPHandler) parseTCPLayer(pkt *Packet, debugLevel int) (*layers.TCP, bool) {
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "TCP packet missing TCP layer sn=%d\n", pkt.SerialNumber)
		}

		return nil, false
	}

	tcp, ok := tcpLayer.(*layers.TCP)

	return tcp, ok
}

// logTCPPacket logs TCP packet details at verbose debug level.
func (h *TCPHandler) logTCPPacket(
	srcIP, dstIP any,
	tcp *layers.TCP,
	debugLevel int,
	serial int,
) {
	if debugLevel < DebugLevelVerbose {
		return
	}

	flags := buildTCPFlagsString(tcp)
	_, _ = fmt.Fprintf(os.Stdout, "TCP packet: %s:%d -> %s:%d flags=[%s] seq=%d ack=%d sn=%d\n",
		srcIP, tcp.SrcPort, dstIP, tcp.DstPort,
		flags, tcp.Seq, tcp.Ack, serial)
}

// buildTCPFlagsString builds a human-readable TCP flags string.
func buildTCPFlagsString(tcp *layers.TCP) string {
	flags := ""
	if tcp.SYN {
		flags += "SYN "
	}

	if tcp.ACK {
		flags += "ACK "
	}

	if tcp.FIN {
		flags += "FIN "
	}

	if tcp.RST {
		flags += "RST "
	}

	return flags
}

// routeToHandler routes the TCP packet to the appropriate application handler.
func (h *TCPHandler) routeToHandler(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	devices []*config.Device,
) {
	isSYNOnly := tcp.SYN && !tcp.ACK
	hasPayload := len(tcp.Payload) > 0

	switch tcp.DstPort {
	case TCPPortHTTP:
		// A bare SYN (TCP path analysis / port probe) gets a SYN-ACK so the hop
		// completes; a SYN with payload is a real HTTP request.
		if isSYNOnly {
			h.stack.healthCheckHandler.HandleTCPConnect(pkt, ipLayer, tcp, devices, TCPPortHTTP)
		} else {
			h.handleHTTP(pkt, ipLayer, tcp, devices, hasPayload)
		}
	case TCPPortHTTPS:
		h.handleHealthCheckPort(pkt, ipLayer, tcp, devices, TCPPortHTTPS, isSYNOnly, nil)
	case TCPPortSSH:
		h.ssh.HandlePacket(pkt, ipLayer, tcp, devices)
	case TCPPortFTP:
		h.handleFTP(pkt, ipLayer, tcp, devices, hasPayload)
	case TCPPortLDAP, TCPPortLDAPS:
		h.handleLDAP(pkt, ipLayer, tcp, devices, isSYNOnly, hasPayload)
	case TCPPortRTSP:
		h.handleHealthCheckPort(pkt, ipLayer, tcp, devices, TCPPortRTSP, isSYNOnly,
			func() { h.stack.healthCheckHandler.HandleRTSPRequest(pkt, ipLayer, tcp, devices) })
	case TCPPortMySQL:
		h.handleHealthCheckPort(pkt, ipLayer, tcp, devices, TCPPortMySQL, isSYNOnly,
			func() { h.stack.healthCheckHandler.HandleMySQLRequest(pkt, ipLayer, tcp, devices) })
	case TCPPortPostgres:
		h.handleHealthCheckPort(pkt, ipLayer, tcp, devices, TCPPortPostgres, isSYNOnly,
			func() { h.stack.healthCheckHandler.HandlePostgresRequest(pkt, ipLayer, tcp, devices) })
	case TCPPortMSSQL:
		h.handleHealthCheckPort(pkt, ipLayer, tcp, devices, TCPPortMSSQL, isSYNOnly,
			func() { h.stack.healthCheckHandler.HandleMSSQLRequest(pkt, ipLayer, tcp, devices) })
	case TCPPortModbus:
		h.handleHealthCheckPort(pkt, ipLayer, tcp, devices, TCPPortModbus, isSYNOnly,
			func() { h.stack.healthCheckHandler.HandleModbusRequest(pkt, ipLayer, tcp, devices) })
	case TCPPortDICOM:
		h.handleHealthCheckPort(pkt, ipLayer, tcp, devices, TCPPortDICOM, isSYNOnly,
			func() { h.stack.healthCheckHandler.HandleDICOMRequest(pkt, ipLayer, tcp, devices) })
	case TCPPortHL7:
		h.handleHealthCheckPort(pkt, ipLayer, tcp, devices, TCPPortHL7, isSYNOnly,
			func() { h.stack.healthCheckHandler.HandleHL7Request(pkt, ipLayer, tcp, devices) })
	case TCPPortOPCUA:
		h.handleHealthCheckPort(pkt, ipLayer, tcp, devices, TCPPortOPCUA, isSYNOnly,
			func() { h.stack.healthCheckHandler.HandleOPCUARequest(pkt, ipLayer, tcp, devices) })
	case TCPPortSMB:
		h.handleHealthCheckPort(pkt, ipLayer, tcp, devices, TCPPortSMB, isSYNOnly,
			func() { h.stack.healthCheckHandler.HandleSMBRequest(pkt, ipLayer, tcp, devices) })
	case TCPPortIPerf3:
		h.stack.iperf3Handler.HandleIPerf3Request(pkt, ipLayer, tcp, devices)
	default:
		if isSYNOnly {
			h.sendRST(pkt, ipLayer, tcp, devices)
		}
	}
}

// handleHTTP routes HTTP traffic.
func (h *TCPHandler) handleHTTP(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	devices []*config.Device,
	hasPayload bool,
) {
	if hasPayload {
		h.stack.httpHandler.HandleRequest(pkt, ipLayer, tcp, devices)
	}
}

// handleFTP routes FTP control connection traffic.
func (h *TCPHandler) handleFTP(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	devices []*config.Device,
	hasPayload bool,
) {
	if hasPayload {
		h.stack.ftpHandler.HandleRequest(pkt, ipLayer, tcp, devices)
	}
}

// handleLDAP routes LDAP/LDAPS traffic.
func (h *TCPHandler) handleLDAP(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	devices []*config.Device,
	isSYNOnly, hasPayload bool,
) {
	if isSYNOnly {
		h.stack.healthCheckHandler.HandleTCPConnect(pkt, ipLayer, tcp, devices, uint16(tcp.DstPort))
	} else if hasPayload {
		h.stack.healthCheckHandler.HandleLDAPRequest(pkt, ipLayer, tcp, devices)
	}
}

// handleHealthCheckPort handles ports that follow the standard health check pattern.
func (h *TCPHandler) handleHealthCheckPort(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	devices []*config.Device,
	port uint16,
	isSYNOnly bool,
	payloadHandler func(),
) {
	if isSYNOnly {
		h.stack.healthCheckHandler.HandleTCPConnect(pkt, ipLayer, tcp, devices, port)
	} else if len(tcp.Payload) > 0 && payloadHandler != nil {
		payloadHandler()
	}
}

// sendRST sends a TCP RST packet.
func (h *TCPHandler) sendRST(pkt *Packet, ipLayer *layers.IPv4, tcp *layers.TCP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	if len(devices) == 0 {
		return
	}
	device := devices[0]
	identity, ok := h.stack.replyEthernet(pkt, device)
	if !ok {
		return
	}

	h.sendRSTPacket(device, identity, ipLayer, tcp, debugLevel)
}

// findDeviceWithIP finds a device that has the specified IP address and MAC.
func (h *TCPHandler) findDeviceWithIP(devices []*config.Device, targetIP any) *config.Device {
	for _, device := range devices {
		if len(device.MACAddress) == 0 {
			continue
		}

		if h.deviceHasIP(device, targetIP) {
			return device
		}
	}

	return nil
}

// deviceHasIP checks if a device has the specified IP address.
func (h *TCPHandler) deviceHasIP(device *config.Device, targetIP any) bool {
	ip, ok := targetIP.(net.IP)
	if !ok {
		return false
	}
	if ip.To4() == nil {
		return slices.ContainsFunc(device.IPAddresses, func(deviceIP net.IP) bool {
			return deviceIP.Equal(ip)
		})
	}

	return h.stack.deviceOwnsIPv4(device, ip)
}

// lookupDestinationMAC looks up the MAC address for a destination IP, scoped
// to the VLAN segment the original request arrived on.
func (h *TCPHandler) lookupDestinationMAC(srcIP any, debugLevel int, vlan int) []byte {
	ip, ok := srcIP.(net.IP)
	if !ok {
		return nil
	}

	srcDevice := h.stack.devicesForStateIPv4(vlan, ip)
	if len(srcDevice) > 0 && len(srcDevice[0].MACAddress) > 0 {
		return srcDevice[0].MACAddress
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "Cannot send RST: no MAC for %s\n", srcIP)
	}

	return nil
}

// sendRSTPacket builds and sends a TCP RST packet.
func (h *TCPHandler) sendRSTPacket(
	device *config.Device,
	identity replyEthernetIdentity,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	debugLevel int,
) {
	eth := &layers.Ethernet{
		SrcMAC:       identity.source,
		DstMAC:       identity.destination,
		EthernetType: layers.EthernetTypeIPv4,
	}

	ipReply := &layers.IPv4{
		Version:  tcpIPv4Version,
		IHL:      tcpIPv4IHL,
		TTL:      tcpIPv4TTL,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    ipLayer.DstIP,
		DstIP:    ipLayer.SrcIP,
	}

	tcpReply := &layers.TCP{
		SrcPort: tcp.DstPort,
		DstPort: tcp.SrcPort,
		Seq:     0,
		Ack:     tcp.Seq + 1,
		RST:     true,
		ACK:     true,
		Window:  0,
	}
	_ = tcpReply.SetNetworkLayerForChecksum(ipReply)

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}

	if err := gopacket.SerializeLayers(buffer, opts, eth, ipReply, tcpReply); err != nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "Error serializing TCP RST: %v\n", err)
		}

		return
	}

	h.sendSerializedPacket(buffer.Bytes(), device, ipReply, tcpReply, debugLevel, identity.vlan)
}

// sendSerializedPacket sends a serialized packet and logs the result.
func (h *TCPHandler) sendSerializedPacket(
	data []byte,
	device *config.Device,
	ipReply *layers.IPv4,
	tcpReply *layers.TCP,
	debugLevel int,
	vlan int,
) {
	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	pkt := &Packet{
		Buffer:       data,
		Length:       len(data),
		SerialNumber: serialNum,
		Device:       device,
		VLAN:         vlan, // reply on the VLAN the request arrived on (tagged or untagged)
	}

	h.stack.Send(pkt)

	if debugLevel >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "Sent TCP RST from %s:%d to %s:%d device=%s sn=%d\n",
			ipReply.SrcIP, tcpReply.SrcPort, ipReply.DstIP, tcpReply.DstPort,
			device.Name, serialNum)
	}
}

// SendTCP sends a TCP packet.
func (h *TCPHandler) SendTCP(
	srcIP, dstIP []byte,
	srcPort, dstPort uint16,
	seq, ack uint32,
	flags byte,
	payload []byte,
	srcMAC, dstMAC []byte,
) error {
	// Build Ethernet header
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	// Build IP header
	ipLayer := &layers.IPv4{
		Version:  tcpIPv4Version,
		IHL:      tcpIPv4IHL,
		TTL:      tcpIPv4TTL,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}

	// Build TCP header
	tcpLayer := &layers.TCP{
		SrcPort: layers.TCPPort(srcPort),
		DstPort: layers.TCPPort(dstPort),
		Seq:     seq,
		Ack:     ack,
		Window:  tcpWindowSize,
	}

	// Set flags
	tcpLayer.SYN = (flags & tcpFlagSYN) != 0
	tcpLayer.ACK = (flags & tcpFlagACK) != 0
	tcpLayer.FIN = (flags & tcpFlagFIN) != 0
	tcpLayer.RST = (flags & tcpFlagRST) != 0
	tcpLayer.PSH = (flags & tcpFlagPSH) != 0

	_ = tcpLayer.SetNetworkLayerForChecksum(ipLayer) // error is non-critical for simulation

	// Serialize
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts,
		eth,
		ipLayer,
		tcpLayer,
		gopacket.Payload(payload),
	)
	if err != nil {
		return fmt.Errorf("error serializing TCP packet: %w", err)
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
	}

	h.stack.Send(pkt)

	if h.stack.GetDebugLevel() >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "Sent TCP packet: %s:%d -> %s:%d length=%d sn=%d\n",
			srcIP, srcPort, dstIP, dstPort, len(payload), serialNum)
	}

	return nil
}

// HandlePacketV6 processes a TCP packet over IPv6.
func (h *TCPHandler) HandlePacketV6(
	pkt *Packet,
	packet gopacket.Packet,
	ipv6 *layers.IPv6,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	tcp, ok := h.parseTCPLayerFromPacket(packet, debugLevel, pkt.SerialNumber)
	if !ok {
		return
	}

	h.logTCPPacketV6(ipv6, tcp, debugLevel, pkt.SerialNumber)
	h.routeToHandlerV6(pkt, packet, ipv6, tcp, devices)
}

// parseTCPLayerFromPacket extracts the TCP layer from a parsed packet.
func (h *TCPHandler) parseTCPLayerFromPacket(
	packet gopacket.Packet,
	debugLevel int,
	serial int,
) (*layers.TCP, bool) {
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "TCP/IPv6 packet missing TCP layer sn=%d\n", serial)
		}

		return nil, false
	}

	tcp, ok := tcpLayer.(*layers.TCP)

	return tcp, ok
}

// logTCPPacketV6 logs IPv6 TCP packet details at verbose debug level.
func (h *TCPHandler) logTCPPacketV6(
	ipv6 *layers.IPv6,
	tcp *layers.TCP,
	debugLevel int,
	serial int,
) {
	if debugLevel < DebugLevelVerbose {
		return
	}

	flags := buildTCPFlagsString(tcp)
	_, _ = fmt.Fprintf(os.Stdout, "TCP/IPv6 packet: [%s]:%d -> [%s]:%d flags=[%s] seq=%d ack=%d sn=%d\n",
		ipv6.SrcIP, tcp.SrcPort, ipv6.DstIP, tcp.DstPort,
		flags, tcp.Seq, tcp.Ack, serial)
}

// routeToHandlerV6 routes IPv6 TCP packets to the appropriate application handler.
func (h *TCPHandler) routeToHandlerV6(
	pkt *Packet,
	packet gopacket.Packet,
	ipv6 *layers.IPv6,
	tcp *layers.TCP,
	devices []*config.Device,
) {
	hasPayload := len(tcp.Payload) > 0
	isSYNOnly := tcp.SYN && !tcp.ACK

	switch tcp.DstPort {
	case TCPPortHTTP:
		if hasPayload {
			h.stack.httpHandler.HandleRequestV6(pkt, packet, ipv6, tcp, devices)
		}
	case TCPPortFTP:
		if hasPayload {
			h.stack.ftpHandler.HandleRequestV6(pkt, packet, ipv6, tcp, devices)
		}
	default:
		if isSYNOnly {
			h.sendRSTV6(ipv6, tcp, devices, pkt.VLAN)
		}
	}
}

// sendRSTV6 sends a TCP RST packet over IPv6.
func (h *TCPHandler) sendRSTV6(ipv6 *layers.IPv6, tcp *layers.TCP, devices []*config.Device, vlan int) {
	debugLevel := h.stack.GetDebugLevel()

	device := h.findDeviceWithIP(devices, ipv6.DstIP)
	if device == nil {
		return
	}

	dstMAC := h.lookupDestinationMACV6(ipv6.SrcIP, debugLevel, vlan)
	if dstMAC == nil {
		return
	}

	h.sendRSTPacketV6(device, dstMAC, ipv6, tcp, debugLevel)
}

// lookupDestinationMACV6 looks up the MAC address for an IPv6 destination,
// scoped to the VLAN segment the original request arrived on.
func (h *TCPHandler) lookupDestinationMACV6(srcIP net.IP, debugLevel int, vlan int) []byte {
	srcDevice := h.stack.devicesFor(vlan).GetByIP(srcIP)
	if len(srcDevice) > 0 && len(srcDevice[0].MACAddress) > 0 {
		return srcDevice[0].MACAddress
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "Cannot send RST: no MAC for [%s]\n", srcIP)
	}

	return nil
}

// sendRSTPacketV6 builds and sends a TCP RST packet over IPv6.
func (h *TCPHandler) sendRSTPacketV6(
	device *config.Device,
	dstMAC []byte,
	ipv6 *layers.IPv6,
	tcp *layers.TCP,
	debugLevel int,
) {
	eth := &layers.Ethernet{
		SrcMAC:       device.MACAddress,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv6,
	}

	ipv6Reply := &layers.IPv6{
		Version:    tcpIPv6Version,
		HopLimit:   tcpIPv6HopLimit,
		NextHeader: layers.IPProtocolTCP,
		SrcIP:      ipv6.DstIP,
		DstIP:      ipv6.SrcIP,
	}

	tcpReply := &layers.TCP{
		SrcPort: tcp.DstPort,
		DstPort: tcp.SrcPort,
		Seq:     0,
		Ack:     tcp.Seq + 1,
		RST:     true,
		ACK:     true,
		Window:  0,
	}
	_ = tcpReply.SetNetworkLayerForChecksum(ipv6Reply)

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}

	if err := gopacket.SerializeLayers(buffer, opts, eth, ipv6Reply, tcpReply); err != nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "Error serializing TCP/IPv6 RST: %v\n", err)
		}

		return
	}

	h.sendSerializedPacketV6(buffer.Bytes(), device, ipv6Reply, tcpReply, debugLevel)
}

// sendSerializedPacketV6 sends a serialized IPv6 packet and logs the result.
func (h *TCPHandler) sendSerializedPacketV6(
	data []byte,
	device *config.Device,
	ipv6Reply *layers.IPv6,
	tcpReply *layers.TCP,
	debugLevel int,
) {
	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	pkt := &Packet{
		Buffer:       data,
		Length:       len(data),
		SerialNumber: serialNum,
		Device:       device,
	}

	h.stack.Send(pkt)

	if debugLevel >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "Sent TCP/IPv6 RST from [%s]:%d to [%s]:%d device=%s sn=%d\n",
			ipv6Reply.SrcIP, tcpReply.SrcPort, ipv6Reply.DstIP, tcpReply.DstPort,
			device.Name, serialNum)
	}
}
