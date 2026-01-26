package api

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// Protocol operation constants.
const (
	arpOperationRequest = 1 // ARP Request operation
	arpOperationReply   = 2 // ARP Reply operation
	icmpTypeEchoRequest = 8 // ICMP Echo Request type
	icmpTypeEchoReply   = 0 // ICMP Echo Reply type
)

// Packet parsing constants.
const (
	idUniqueMask    = 0xFFFF // mask for ID uniqueness
	maxRawDataBytes = 128    // max raw data bytes to include
)

// ============================================================================
// Packet Layer Parsing
// ============================================================================

// parseEthernetLayer extracts Ethernet header information from a packet.
func parseEthernetLayer(packet gopacket.Packet, pkt *PcapPacket) {
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		return
	}

	eth, ok := ethLayer.(*layers.Ethernet)
	if !ok {
		return
	}

	pkt.Headers["ethernet"] = map[string]string{
		"srcMAC": eth.SrcMAC.String(),
		"dstMAC": eth.DstMAC.String(),
		"type":   eth.EthernetType.String(),
	}
}

// parseIPv4Layer extracts IPv4 header information from a packet.
func parseIPv4Layer(packet gopacket.Packet, pkt *PcapPacket) {
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return
	}

	ip, ok := ipLayer.(*layers.IPv4)
	if !ok {
		return
	}

	pkt.SourceIP = ip.SrcIP.String()
	pkt.DestIP = ip.DstIP.String()
	pkt.Protocol = ip.Protocol.String()
	pkt.Headers["ipv4"] = map[string]any{
		"version": ip.Version,
		"ttl":     ip.TTL,
		"id":      ip.Id,
	}
}

// parseIPv6Layer extracts IPv6 header information from a packet.
func parseIPv6Layer(packet gopacket.Packet, pkt *PcapPacket) {
	ipLayer := packet.Layer(layers.LayerTypeIPv6)
	if ipLayer == nil {
		return
	}

	ip, ok := ipLayer.(*layers.IPv6)
	if !ok {
		return
	}

	pkt.SourceIP = ip.SrcIP.String()
	pkt.DestIP = ip.DstIP.String()
	pkt.Protocol = ip.NextHeader.String()
	pkt.Headers["ipv6"] = map[string]any{
		"version":    ip.Version,
		"hopLimit":   ip.HopLimit,
		"nextHeader": ip.NextHeader.String(),
	}
}

// parseARPLayer extracts ARP information from a packet.
func parseARPLayer(packet gopacket.Packet, pkt *PcapPacket) {
	arpLayer := packet.Layer(layers.LayerTypeARP)
	if arpLayer == nil {
		return
	}

	arp, ok := arpLayer.(*layers.ARP)
	if !ok {
		return
	}

	pkt.Protocol = "ARP"
	pkt.SourceIP = fmt.Sprintf(
		"%d.%d.%d.%d",
		arp.SourceProtAddress[0],
		arp.SourceProtAddress[1],
		arp.SourceProtAddress[2],
		arp.SourceProtAddress[3],
	)
	pkt.DestIP = fmt.Sprintf(
		"%d.%d.%d.%d",
		arp.DstProtAddress[0],
		arp.DstProtAddress[1],
		arp.DstProtAddress[2],
		arp.DstProtAddress[3],
	)

	opStr := "Unknown"
	switch arp.Operation {
	case arpOperationRequest:
		opStr = "Request"
	case arpOperationReply:
		opStr = "Reply"
	}

	pkt.Info = fmt.Sprintf("ARP %s: Who has %s? Tell %s", opStr, pkt.DestIP, pkt.SourceIP)
}

// parseTCPLayer extracts TCP information from a packet.
func parseTCPLayer(packet gopacket.Packet, pkt *PcapPacket) {
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		return
	}

	tcp, ok := tcpLayer.(*layers.TCP)
	if !ok {
		return
	}

	pkt.Protocol = "TCP"
	srcPort := int(tcp.SrcPort)
	dstPort := int(tcp.DstPort)
	pkt.SourcePort = &srcPort
	pkt.DestPort = &dstPort

	flags := getTCPFlags(tcp)
	pkt.Info = fmt.Sprintf("%d -> %d [%s] Seq=%d Ack=%d Win=%d",
		srcPort, dstPort, flags, tcp.Seq, tcp.Ack, tcp.Window)

	pkt.Headers["tcp"] = map[string]any{
		"srcPort": srcPort,
		"dstPort": dstPort,
		"seq":     tcp.Seq,
		"ack":     tcp.Ack,
		"flags":   flags,
		"window":  tcp.Window,
	}

	pkt.Protocol = getProtocolByPort(srcPort, dstPort, "TCP")
}

// parseUDPLayer extracts UDP information from a packet.
func parseUDPLayer(packet gopacket.Packet, pkt *PcapPacket) {
	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		return
	}

	udp, ok := udpLayer.(*layers.UDP)
	if !ok {
		return
	}

	pkt.Protocol = "UDP"
	srcPort := int(udp.SrcPort)
	dstPort := int(udp.DstPort)
	pkt.SourcePort = &srcPort
	pkt.DestPort = &dstPort

	pkt.Info = fmt.Sprintf("%d -> %d Len=%d", srcPort, dstPort, udp.Length)
	pkt.Headers["udp"] = map[string]any{
		"srcPort": srcPort,
		"dstPort": dstPort,
		"length":  udp.Length,
	}

	pkt.Protocol = getProtocolByPort(srcPort, dstPort, "UDP")
}

// parseICMPLayer extracts ICMP information from a packet.
func parseICMPLayer(packet gopacket.Packet, pkt *PcapPacket) {
	icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
	if icmpLayer == nil {
		return
	}

	icmp, ok := icmpLayer.(*layers.ICMPv4)
	if !ok {
		return
	}

	pkt.Protocol = "ICMP"
	pkt.Info = fmt.Sprintf("Type=%d Code=%d", icmp.TypeCode.Type(), icmp.TypeCode.Code())

	if icmp.TypeCode.Type() == icmpTypeEchoRequest {
		pkt.Info = "Echo Request (ping)"
	} else if icmp.TypeCode.Type() == icmpTypeEchoReply {
		pkt.Info = "Echo Reply (pong)"
	}
}

// parseDNSLayer extracts DNS information from a packet.
func parseDNSLayer(packet gopacket.Packet, pkt *PcapPacket) {
	dnsLayer := packet.Layer(layers.LayerTypeDNS)
	if dnsLayer == nil {
		return
	}

	dns, ok := dnsLayer.(*layers.DNS)
	if !ok {
		return
	}

	pkt.Protocol = "DNS"
	if dns.QR {
		pkt.Info = fmt.Sprintf("Response: %d answers", len(dns.Answers))
		return
	}

	if len(dns.Questions) > 0 {
		pkt.Info = "Query: " + string(dns.Questions[0].Name)
	} else {
		pkt.Info = "Query"
	}
}

func parsePacket(packet gopacket.Packet, num int) PcapPacket {
	pkt := PcapPacket{
		ID:        fmt.Sprintf("%d-%x", num, time.Now().UnixNano()&idUniqueMask),
		Number:    num,
		Timestamp: packet.Metadata().Timestamp.UTC().Format(time.RFC3339Nano),
		Length:    packet.Metadata().Length,
		Protocol:  "Unknown",
		SourceIP:  "N/A",
		DestIP:    "N/A",
		Headers:   make(map[string]any),
	}

	// Get raw hex data (first 128 bytes)
	rawLen := min(len(packet.Data()), maxRawDataBytes)
	pkt.RawData = hex.EncodeToString(packet.Data()[:rawLen])

	// Parse each layer
	parseEthernetLayer(packet, &pkt)
	parseIPv4Layer(packet, &pkt)
	parseIPv6Layer(packet, &pkt)
	parseARPLayer(packet, &pkt)
	parseTCPLayer(packet, &pkt)
	parseUDPLayer(packet, &pkt)
	parseICMPLayer(packet, &pkt)
	parseDNSLayer(packet, &pkt)

	// Set default info if not set
	if pkt.Info == "" {
		pkt.Info = fmt.Sprintf("%s %s -> %s", pkt.Protocol, pkt.SourceIP, pkt.DestIP)
	}

	return pkt
}

func getTCPFlags(tcp *layers.TCP) string {
	var flags []string
	if tcp.SYN {
		flags = append(flags, "SYN")
	}

	if tcp.ACK {
		flags = append(flags, "ACK")
	}

	if tcp.FIN {
		flags = append(flags, "FIN")
	}

	if tcp.RST {
		flags = append(flags, "RST")
	}

	if tcp.PSH {
		flags = append(flags, "PSH")
	}

	if tcp.URG {
		flags = append(flags, "URG")
	}

	if len(flags) == 0 {
		return "none"
	}

	return strings.Join(flags, ",")
}

func getProtocolByPort(srcPort, dstPort int, baseProto string) string {
	ports := map[int]string{
		20:   "FTP-DATA",
		21:   "FTP",
		22:   "SSH",
		23:   "Telnet",
		25:   "SMTP",
		53:   "DNS",
		67:   "DHCP",
		68:   "DHCP",
		80:   "HTTP",
		110:  "POP3",
		123:  "NTP",
		137:  "NetBIOS",
		138:  "NetBIOS",
		139:  "NetBIOS",
		143:  "IMAP",
		161:  "SNMP",
		162:  "SNMP-Trap",
		443:  "HTTPS",
		445:  "SMB",
		514:  "Syslog",
		993:  "IMAPS",
		995:  "POP3S",
		3306: "MySQL",
		3389: "RDP",
		5432: "PostgreSQL",
		6379: "Redis",
		8080: "HTTP-Alt",
		8443: "HTTPS-Alt",
	}

	if name, ok := ports[dstPort]; ok {
		return name
	}

	if name, ok := ports[srcPort]; ok {
		return name
	}

	return baseProto
}
