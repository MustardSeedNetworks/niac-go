package api

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// writePcapToSecureTempFile creates a secure temp directory and writes pcap data to a temp file.
// Returns the temp file path and a cleanup function. Caller must call cleanup when done.
func writePcapToSecureTempFile(data []byte) (string, func(), error) {
	// SECURITY FIX #167: Use dedicated temp directory with restrictive permissions
	dir := filepath.Join(os.TempDir(), "niac-pcap-analysis")
	if err := os.MkdirAll(dir, dirPermSecure); err != nil {
		return "", nil, fmt.Errorf("create temp dir: %w", err)
	}

	// Ensure directory permissions are correct even if it already exists
	if err := os.Chmod(dir, dirPermSecure); err != nil {
		return "", nil, fmt.Errorf("secure temp dir: %w", err)
	}

	// Write to temp file (gopacket requires file path)
	tmpFile, err := os.CreateTemp(dir, "pcap-analysis-*.pcap")
	if err != nil {
		return "", nil, fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	// SECURITY FIX #167: Write data while file is still open to minimize race window
	if _, writeErr := tmpFile.Write(data); writeErr != nil {
		_ = tmpFile.Close()
		cleanup()

		return "", nil, fmt.Errorf("write temp file: %w", writeErr)
	}

	// Sync before close to ensure data is written
	if syncErr := tmpFile.Sync(); syncErr != nil {
		_ = tmpFile.Close()
		cleanup()

		return "", nil, fmt.Errorf("sync temp file: %w", syncErr)
	}

	if closeErr := tmpFile.Close(); closeErr != nil {
		cleanup()

		return "", nil, fmt.Errorf("close temp file: %w", closeErr)
	}

	return tmpPath, cleanup, nil
}

// pcapPacketCollector accumulates packet statistics during pcap analysis.
type pcapPacketCollector struct {
	sourceCounts map[string]int
	destCounts   map[string]int
	firstTime    time.Time
	lastTime     time.Time
}

// newPcapPacketCollector creates a new packet collector.
func newPcapPacketCollector() *pcapPacketCollector {
	return &pcapPacketCollector{
		sourceCounts: make(map[string]int),
		destCounts:   make(map[string]int),
	}
}

// recordPacket updates collector statistics for a single packet.
func (c *pcapPacketCollector) recordPacket(pkt PcapPacket, ts time.Time) {
	if c.firstTime.IsZero() || ts.Before(c.firstTime) {
		c.firstTime = ts
	}

	if ts.After(c.lastTime) {
		c.lastTime = ts
	}

	if pkt.SourceIP != "" && pkt.SourceIP != "N/A" {
		c.sourceCounts[pkt.SourceIP]++
	}

	if pkt.DestIP != "" && pkt.DestIP != "N/A" {
		c.destCounts[pkt.DestIP]++
	}
}

// finalizeStats populates the result stats from collected data.
func (c *pcapPacketCollector) finalizeStats(result *PcapAnalysisResult, packetNum int) {
	result.Stats.TotalPackets = packetNum

	if !c.firstTime.IsZero() {
		result.Stats.TimeRange.Start = c.firstTime.UTC().Format(time.RFC3339Nano)
		result.Stats.TimeRange.End = c.lastTime.UTC().Format(time.RFC3339Nano)
		result.Stats.TimeRange.DurationMs = c.lastTime.Sub(c.firstTime).Milliseconds()
	}

	result.Stats.TopSources = topNFromMap(c.sourceCounts, topNCount)
	result.Stats.TopDestinations = topNFromMap(c.destCounts, topNCount)
}

// analyzePcapData analyzes raw PCAP data and returns a structured result.
func analyzePcapData(data []byte, filename string) (*PcapAnalysisResult, error) {
	tmpPath, cleanup, err := writePcapToSecureTempFile(data)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Open pcap from temp file
	handle, err := pcap.OpenOffline(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("open pcap: %w", err)
	}
	defer handle.Close()

	result := &PcapAnalysisResult{
		Filename: filename,
		FileSize: int64(len(data)),
		Packets:  make([]PcapPacket, 0),
		Stats:    PcapStats{Protocols: make(map[string]int)},
	}

	collector := newPcapPacketCollector()
	packetNum := 0

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		packetNum++

		ts := packet.Metadata().Timestamp
		pkt := parsePacket(packet, packetNum)
		result.Packets = append(result.Packets, pkt)
		result.Stats.TotalBytes += int64(pkt.Length)
		result.Stats.Protocols[pkt.Protocol]++

		collector.recordPacket(pkt, ts)
	}

	collector.finalizeStats(result, packetNum)

	return result, nil
}

// parsePacket converts a gopacket.Packet to a PcapPacket struct.
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

	// FIX #269: Bounds check before accessing address bytes to prevent panic on malformed packets
	if len(arp.SourceProtAddress) < ipv4AddrLen || len(arp.DstProtAddress) < ipv4AddrLen {
		pkt.Info = "ARP: malformed packet (address too short)"
		return
	}

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
