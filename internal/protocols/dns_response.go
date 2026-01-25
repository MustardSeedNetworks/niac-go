package protocols

import (
	"fmt"
	"net"
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
)

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

func (h *DNSHandler) sendDNSPayloadV6(
	payload []byte,
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
	err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, gopacket.Payload(payload))
	if err != nil {
		return fmt.Errorf("failed to serialize DNS/IPv6 response: %w", err)
	}

	return h.stack.SendRawPacket(buf.Bytes())
}
