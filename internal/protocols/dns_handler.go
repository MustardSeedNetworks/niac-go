package protocols

import (
	"fmt"
	"net"
	"os"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
)

func (h *DNSHandler) HandleQuery(
	pkt *Packet,
	ipLayer *layers.IPv4,
	udpLayer *layers.UDP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	dns, ok := h.parseDNSLayer(packet, debugLevel, pkt.SerialNumber, "DNS")
	if !ok {
		return
	}

	h.stack.IncrementStat("dns_queries")
	h.logDNSQueries(dns.Questions, ipLayer.SrcIP, debugLevel, pkt.SerialNumber)

	serverDevice, serverIP := h.selectServerDevice(devices, false)
	if !h.validateServerDevice(serverDevice, serverIP, debugLevel, pkt.SerialNumber, "DNS") {
		return
	}

	if h.handleNBSTATIfPresent(pkt, ipLayer, udpLayer, serverDevice, dns, packet, debugLevel) {
		return
	}

	response := h.buildDNSResponse(dns, serverDevice, debugLevel, pkt.SerialNumber)
	srcMAC := h.extractSourceMAC(packet)
	h.sendAndLogResponse(response, serverIP, ipLayer.SrcIP, serverDevice.MACAddress, srcMAC,
		udpLayer.SrcPort, debugLevel, pkt.SerialNumber)
}

// parseDNSLayer extracts and validates the DNS layer from a packet.
func (h *DNSHandler) parseDNSLayer(
	packet gopacket.Packet,
	debugLevel int,
	serial int,
	prefix string,
) (*layers.DNS, bool) {
	dnsLayer := packet.Layer(layers.LayerTypeDNS)
	if dnsLayer == nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "%s packet missing DNS layer sn=%d\n", prefix, serial)
		}

		return nil, false
	}

	dns, ok := dnsLayer.(*layers.DNS)

	return dns, ok
}

// logDNSQueries logs DNS queries at verbose debug level.
func (h *DNSHandler) logDNSQueries(
	questions []layers.DNSQuestion,
	srcIP any,
	debugLevel int,
	serial int,
) {
	if debugLevel < DebugLevelVerbose {
		return
	}

	for _, q := range questions {
		_, _ = fmt.Fprintf(os.Stdout, "DNS Query: %s type=%s class=%s from %s sn=%d\n",
			string(q.Name), q.Type, q.Class, srcIP, serial)
	}
}

// validateServerDevice checks if the server device is properly configured.
func (h *DNSHandler) validateServerDevice(
	serverDevice *config.Device,
	serverIP net.IP,
	debugLevel int,
	serial int,
	prefix string,
) bool {
	if serverDevice == nil || serverIP == nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "%s: No server device/IP configured sn=%d\n", prefix, serial)
		}

		return false
	}

	if len(serverDevice.MACAddress) == 0 {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "%s: Server device %s missing MAC address sn=%d\n",
				prefix, serverDevice.Name, serial)
		}

		return false
	}

	return true
}

// handleNBSTATIfPresent checks for and handles NBSTAT queries.
// Returns true if an NBSTAT query was handled.
func (h *DNSHandler) handleNBSTATIfPresent(
	pkt *Packet,
	ipLayer *layers.IPv4,
	udpLayer *layers.UDP,
	serverDevice *config.Device,
	dns *layers.DNS,
	packet gopacket.Packet,
	debugLevel int,
) bool {
	for _, q := range dns.Questions {
		if q.Type != layers.DNSType(dnsTypeNBSTAT) {
			continue
		}

		err := h.handleNBSTATQuery(pkt, ipLayer, udpLayer, serverDevice, dns.ID, q, packet)
		if err != nil && debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "DNS: NBSTAT handling failed: %v sn=%d\n", err, pkt.SerialNumber)
		}

		return true
	}

	return false
}

// buildDNSResponse constructs a DNS response for the given query.
func (h *DNSHandler) buildDNSResponse(
	dns *layers.DNS,
	serverDevice *config.Device,
	debugLevel int,
	serial int,
) *layers.DNS {
	response := &layers.DNS{
		ID:           dns.ID,
		QR:           true,
		OpCode:       dns.OpCode,
		AA:           true,
		TC:           false,
		RD:           dns.RD,
		RA:           true,
		ResponseCode: layers.DNSResponseCodeNoErr,
		Questions:    dns.Questions,
		Answers:      []layers.DNSResourceRecord{},
	}

	recordSet := h.getRecordSetForDevice(serverDevice)
	response.Answers, response.ResponseCode = h.resolveQuestions(dns.Questions, recordSet, debugLevel, serial)

	if len(response.Answers) == 0 && debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "DNS: NXDOMAIN for queries sn=%d\n", serial)
	} else if len(response.Answers) > 0 && response.ResponseCode == 0 {
		response.ResponseCode = layers.DNSResponseCodeNoErr
	}

	return response
}

// extractSourceMAC extracts the source MAC address from the Ethernet layer.
func (h *DNSHandler) extractSourceMAC(packet gopacket.Packet) net.HardwareAddr {
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		return eth.SrcMAC
	}

	return nil
}

// sendAndLogResponse sends the DNS response and logs the result.
func (h *DNSHandler) sendAndLogResponse(
	response *layers.DNS,
	srcIP, dstIP net.IP,
	srcMAC, dstMAC net.HardwareAddr,
	dstPort layers.UDPPort,
	debugLevel int,
	serial int,
) {
	err := h.SendDNSResponse(response, srcIP, dstIP, srcMAC, dstMAC, dstPort)
	if err != nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "DNS: Failed to send response: %v sn=%d\n", err, serial)
		}

		return
	}

	if debugLevel >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "DNS: Sent response with %d answers sn=%d\n",
			len(response.Answers), serial)
	}
}

// SendDNSResponse sends a DNS response.
func (h *DNSHandler) SendDNSResponse(
	response *layers.DNS,
	srcIP, dstIP net.IP,
	srcMAC, dstMAC net.HardwareAddr,
	dstPort layers.UDPPort,
) error {
	// Build UDP layer
	udp := &layers.UDP{
		SrcPort: dnsPort,
		DstPort: dstPort,
	}

	// Build IP layer
	ip := &layers.IPv4{
		Version:  dnsIPv4Version,
		TTL:      dnsIPv4TTL,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    srcIP,
		DstIP:    dstIP,
	}

	// Build Ethernet layer
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	// Serialize packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	_ = udp.SetNetworkLayerForChecksum(ip) // error is non-critical for simulation

	err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, response)
	if err != nil {
		return fmt.Errorf("failed to serialize DNS response: %w", err)
	}

	// Send packet
	return h.stack.SendRawPacket(buf.Bytes())
}

// SendDNSResponseV6 sends a DNS response over IPv6.
func (h *DNSHandler) SendDNSResponseV6(
	response *layers.DNS,
	srcIP, dstIP net.IP,
	srcMAC, dstMAC net.HardwareAddr,
	dstPort layers.UDPPort,
) error {
	udp := &layers.UDP{
		SrcPort: dnsPort,
		DstPort: dstPort,
	}

	ip := &layers.IPv6{
		Version:      dnsIPv6Version,
		TrafficClass: 0,
		FlowLabel:    0,
		NextHeader:   layers.IPProtocolUDP,
		HopLimit:     dnsIPv6HopLimit,
		SrcIP:        srcIP,
		DstIP:        dstIP,
	}

	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv6,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	_ = udp.SetNetworkLayerForChecksum(ip) // error is non-critical for simulation

	err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, response)
	if err != nil {
		return fmt.Errorf("failed to serialize DNS/IPv6 response: %w", err)
	}

	return h.stack.SendRawPacket(buf.Bytes())
}

// HandleQueryV6 processes a DNS query over IPv6.
func (h *DNSHandler) HandleQueryV6(
	pkt *Packet,
	packet gopacket.Packet,
	ipv6 *layers.IPv6,
	udpLayer *layers.UDP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	dns, ok := h.parseDNSLayer(packet, debugLevel, pkt.SerialNumber, "DNS/IPv6")
	if !ok {
		return
	}

	h.stack.IncrementStat("dns_queries")

	serverDevice, serverIP := h.selectServerDevice(devices, true)
	if !h.validateServerDevice(serverDevice, serverIP, debugLevel, pkt.SerialNumber, "DNS/IPv6") {
		return
	}

	if h.handleNBSTATIfPresentV6(pkt, ipv6, udpLayer, serverDevice, dns, packet, debugLevel) {
		return
	}

	response := h.buildDNSResponseV6(dns, serverDevice, debugLevel, pkt.SerialNumber)

	dstMAC, ok := h.extractSourceMACWithValidation(packet, debugLevel, pkt.SerialNumber)
	if !ok {
		return
	}

	h.sendAndLogResponseV6(response, serverIP, ipv6.SrcIP, serverDevice.MACAddress, dstMAC,
		udpLayer.SrcPort, debugLevel, pkt.SerialNumber)
}

// handleNBSTATIfPresentV6 checks for and handles NBSTAT queries over IPv6.
// Returns true if an NBSTAT query was handled.
func (h *DNSHandler) handleNBSTATIfPresentV6(
	pkt *Packet,
	ipv6 *layers.IPv6,
	udpLayer *layers.UDP,
	serverDevice *config.Device,
	dns *layers.DNS,
	packet gopacket.Packet,
	debugLevel int,
) bool {
	for _, q := range dns.Questions {
		if q.Type != layers.DNSType(dnsTypeNBSTAT) {
			continue
		}

		err := h.handleNBSTATQueryV6(pkt, ipv6, udpLayer, serverDevice, dns.ID, q, packet)
		if err != nil && debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "DNS/IPv6: NBSTAT handling failed: %v sn=%d\n",
				err, pkt.SerialNumber)
		}

		return true
	}

	return false
}

// buildDNSResponseV6 constructs a DNS response for IPv6 queries.
func (h *DNSHandler) buildDNSResponseV6(
	dns *layers.DNS,
	serverDevice *config.Device,
	debugLevel int,
	serial int,
) *layers.DNS {
	response := &layers.DNS{
		ID:           dns.ID,
		QR:           true,
		OpCode:       dns.OpCode,
		AA:           true,
		TC:           false,
		RD:           dns.RD,
		RA:           true,
		ResponseCode: layers.DNSResponseCodeNoErr,
		Questions:    dns.Questions,
	}

	recordSet := h.getRecordSetForDevice(serverDevice)
	response.Answers, response.ResponseCode = h.resolveQuestions(dns.Questions, recordSet, debugLevel, serial)

	if len(response.Answers) == 0 && debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "DNS/IPv6: NXDOMAIN for queries sn=%d\n", serial)
	} else if len(response.Answers) > 0 && response.ResponseCode == 0 {
		response.ResponseCode = layers.DNSResponseCodeNoErr
	}

	return response
}

// extractSourceMACWithValidation extracts the source MAC and validates the Ethernet layer.
func (h *DNSHandler) extractSourceMACWithValidation(
	packet gopacket.Packet,
	debugLevel int,
	serial int,
) (net.HardwareAddr, bool) {
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "DNS/IPv6: Missing Ethernet layer sn=%d\n", serial)
		}

		return nil, false
	}

	eth, ok := ethLayer.(*layers.Ethernet)
	if !ok {
		return nil, false
	}

	return eth.SrcMAC, true
}

// sendAndLogResponseV6 sends the DNS response over IPv6 and logs the result.
func (h *DNSHandler) sendAndLogResponseV6(
	response *layers.DNS,
	srcIP, dstIP net.IP,
	srcMAC, dstMAC net.HardwareAddr,
	dstPort layers.UDPPort,
	debugLevel int,
	serial int,
) {
	err := h.SendDNSResponseV6(response, srcIP, dstIP, srcMAC, dstMAC, dstPort)
	if err != nil && debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "DNS/IPv6: Failed to send response: %v sn=%d\n", err, serial)
	}
}

// dnsResolveContext holds state for resolving DNS questions.
