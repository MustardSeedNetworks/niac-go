package capture

import (
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

// dirPermSecure is the permission mode for secure directories (owner rwx only).
// This is intentionally 0700 for directories (not 0600) because directories
// require execute permission to access their contents.
const dirPermSecure fs.FileMode = 0o700

// Packet-shaping constants for the analysis engine.
const (
	topNCount       = 10     // top N count for sources/destinations
	idUniqueMask    = 0xFFFF // mask for ID uniqueness
	maxRawDataBytes = 128    // max raw data bytes to include
)

// Protocol operation constants.
const (
	arpOperationRequest = 1 // ARP Request operation
	arpOperationReply   = 2 // ARP Reply operation
	icmpTypeEchoRequest = 8 // ICMP Echo Request type
	icmpTypeEchoReply   = 0 // ICMP Echo Reply type
	pcapMinHeaderLen    = 4 // Minimum header length for pcap protocol address
)

// writeToSecureTempFile creates a secure temp directory and writes pcap data to
// a temp file. Returns the temp file path and a cleanup function. Caller must
// call cleanup when done.
func writeToSecureTempFile(data []byte) (string, func(), error) {
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

// packetCollector accumulates packet statistics during pcap analysis.
type packetCollector struct {
	sourceCounts map[string]int
	destCounts   map[string]int
	firstTime    time.Time
	lastTime     time.Time
}

// newPacketCollector creates a new packet collector.
func newPacketCollector() *packetCollector {
	return &packetCollector{
		sourceCounts: make(map[string]int),
		destCounts:   make(map[string]int),
	}
}

// recordPacket updates collector statistics for a single packet.
func (c *packetCollector) recordPacket(pkt Packet, ts time.Time) {
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
func (c *packetCollector) finalizeStats(result *AnalysisResult, packetNum int) {
	result.Stats.TotalPackets = packetNum

	if !c.firstTime.IsZero() {
		result.Stats.TimeRange.Start = c.firstTime.UTC().Format(time.RFC3339Nano)
		result.Stats.TimeRange.End = c.lastTime.UTC().Format(time.RFC3339Nano)
		result.Stats.TimeRange.DurationMs = c.lastTime.Sub(c.firstTime).Milliseconds()
	}

	result.Stats.TopSources = topNFromMap(c.sourceCounts, topNCount)
	result.Stats.TopDestinations = topNFromMap(c.destCounts, topNCount)
}

// Analyze parses a libpcap/pcapng byte buffer into a structured AnalysisResult.
// The caller assigns ID and CreatedAt; filename is recorded as-is.
func Analyze(data []byte, filename string) (*AnalysisResult, error) {
	tmpPath, cleanup, err := writeToSecureTempFile(data)
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

	result := &AnalysisResult{
		Filename: filename,
		FileSize: int64(len(data)),
		Packets:  make([]Packet, 0),
		Stats:    Stats{Protocols: make(map[string]int)},
	}

	collector := newPacketCollector()
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

// parseEthernetLayer extracts Ethernet header information from a packet.
func parseEthernetLayer(packet gopacket.Packet, pkt *Packet) {
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
func parseIPv4Layer(packet gopacket.Packet, pkt *Packet) {
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
func parseIPv6Layer(packet gopacket.Packet, pkt *Packet) {
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
func parseARPLayer(packet gopacket.Packet, pkt *Packet) {
	arpLayer := packet.Layer(layers.LayerTypeARP)
	if arpLayer == nil {
		return
	}

	arp, ok := arpLayer.(*layers.ARP)
	if !ok {
		return
	}

	pkt.Protocol = "ARP"
	// Guard against non-IPv4 ARP or truncated address fields.
	if len(arp.SourceProtAddress) >= pcapMinHeaderLen {
		pkt.SourceIP = fmt.Sprintf(
			"%d.%d.%d.%d",
			arp.SourceProtAddress[0],
			arp.SourceProtAddress[1],
			arp.SourceProtAddress[2],
			arp.SourceProtAddress[3],
		)
	}

	if len(arp.DstProtAddress) >= pcapMinHeaderLen {
		pkt.DestIP = fmt.Sprintf(
			"%d.%d.%d.%d",
			arp.DstProtAddress[0],
			arp.DstProtAddress[1],
			arp.DstProtAddress[2],
			arp.DstProtAddress[3],
		)
	}

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
func parseTCPLayer(packet gopacket.Packet, pkt *Packet) {
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
func parseUDPLayer(packet gopacket.Packet, pkt *Packet) {
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
func parseICMPLayer(packet gopacket.Packet, pkt *Packet) {
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
func parseDNSLayer(packet gopacket.Packet, pkt *Packet) {
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

func parsePacket(packet gopacket.Packet, num int) Packet {
	pkt := Packet{
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

func topNFromMap(m map[string]int, n int) []IPCount {
	type kv struct {
		key   string
		value int
	}

	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].value > sorted[j].value
	})

	result := make([]IPCount, 0, n)
	for i := 0; i < len(sorted) && i < n; i++ {
		result = append(result, IPCount{
			IP:    sorted[i].key,
			Count: sorted[i].value,
		})
	}

	return result
}
