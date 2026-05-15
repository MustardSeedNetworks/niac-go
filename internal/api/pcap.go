package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

// dirPermSecure is the permission mode for secure directories (owner rwx only).
// This is intentionally 0700 for directories (not 0600) because directories
// require execute permission to access their contents.
const dirPermSecure fs.FileMode = 0o700

// Memory estimation constants for pcap cache.
const (
	resultBaseOverhead  = 200    // approximate fixed overhead for result struct
	packetBaseOverhead  = 100    // base packet struct overhead
	headerValueEstimate = 50     // approximate value size for header map
	statsOverhead       = 500    // approximate stats overhead
	topNCount           = 10     // top N count for sources/destinations
	idUniqueMask        = 0xFFFF // mask for ID uniqueness
	maxRawDataBytes     = 128    // max raw data bytes to include
)

// Protocol operation constants.
const (
	arpOperationRequest = 1 // ARP Request operation
	arpOperationReply   = 2 // ARP Reply operation
	icmpTypeEchoRequest = 8 // ICMP Echo Request type
	icmpTypeEchoReply   = 0 // ICMP Echo Reply type
	pcapMinHeaderLen    = 4 // Minimum header length for pcap protocol address
)

// ============================================================================
// PCAP Analysis Types
// ============================================================================

// PcapPacket represents a single packet from a PCAP file.
type PcapPacket struct {
	ID         string         `json:"id"`
	Number     int            `json:"number"`
	Timestamp  string         `json:"timestamp"`
	SourceIP   string         `json:"sourceIP"`
	DestIP     string         `json:"destIP"`
	SourcePort *int           `json:"sourcePort,omitempty"`
	DestPort   *int           `json:"destPort,omitempty"`
	Protocol   string         `json:"protocol"`
	Length     int            `json:"length"`
	Info       string         `json:"info"`
	RawData    string         `json:"rawData,omitempty"`
	Headers    map[string]any `json:"headers,omitempty"`
}

// PcapTimeRange represents the time span of a capture.
type PcapTimeRange struct {
	Start      string `json:"start"`
	End        string `json:"end"`
	DurationMs int64  `json:"durationMs"`
}

// PcapIPCount represents an IP address with its packet count.
type PcapIPCount struct {
	IP    string `json:"ip"`
	Count int    `json:"count"`
}

// PcapStats provides statistics about the analyzed PCAP.
type PcapStats struct {
	TotalPackets    int            `json:"totalPackets"`
	TotalBytes      int64          `json:"totalBytes"`
	TimeRange       PcapTimeRange  `json:"timeRange"`
	Protocols       map[string]int `json:"protocols"`
	TopSources      []PcapIPCount  `json:"topSources"`
	TopDestinations []PcapIPCount  `json:"topDestinations"`
}

// PcapAnalysisResult is the complete analysis of a PCAP file.
type PcapAnalysisResult struct {
	ID        string       `json:"id"`
	Filename  string       `json:"filename"`
	FileSize  int64        `json:"fileSize"`
	Packets   []PcapPacket `json:"packets"`
	Stats     PcapStats    `json:"stats"`
	CreatedAt string       `json:"createdAt"`
}

// PcapUploadRequest is the request body for uploading a PCAP.
type PcapUploadRequest struct {
	Filename string `json:"filename"`
	Data     string `json:"data"` // Base64 encoded PCAP data
}

// PcapUploadResponse is returned after successful upload.
type PcapUploadResponse struct {
	Success    bool   `json:"success"`
	AnalysisID string `json:"analysisId"`
	Message    string `json:"message"`
}

// ============================================================================
// PCAP Analysis Cache (in-memory store for recent analyses)
// SECURITY FIX #155: Added memory limits to prevent cache exhaustion DoS
// ============================================================================

// Cache memory limits to prevent denial of service attacks.
const (
	// MaxPcapCacheMemory is the maximum total memory for cached PCAP analyses (500MB).
	MaxPcapCacheMemory = 500 * 1024 * 1024
	// MaxPcapCacheEntries is the maximum number of cached analyses.
	MaxPcapCacheEntries = 50
)

// pcapCacheEntry wraps an analysis with its approximate memory size.
type pcapCacheEntry struct {
	result  *PcapAnalysisResult
	memSize int64 // Approximate memory size in bytes
	addedAt time.Time
}

type pcapCache struct {
	mu         sync.RWMutex
	entries    map[string]*pcapCacheEntry
	maxEntries int
	maxMemory  int64
	currentMem int64
}

var globalPcapCache = &pcapCache{
	entries:    make(map[string]*pcapCacheEntry),
	maxEntries: MaxPcapCacheEntries,
	maxMemory:  MaxPcapCacheMemory,
	currentMem: 0,
}

// estimateResultSize calculates approximate memory size of an analysis result.
func estimateResultSize(result *PcapAnalysisResult) int64 {
	if result == nil {
		return 0
	}

	// Base struct size
	size := int64(resultBaseOverhead) // Approximate fixed overhead

	// Add filename and ID sizes
	size += int64(len(result.Filename) + len(result.ID))

	// Add packet sizes (approximate)
	for _, pkt := range result.Packets {
		size += int64(packetBaseOverhead) // Base packet struct overhead
		size += int64(len(pkt.ID) + len(pkt.Timestamp) + len(pkt.SourceIP))
		size += int64(len(pkt.DestIP) + len(pkt.Protocol) + len(pkt.Info))
		size += int64(len(pkt.RawData)) // RawData is the main contributor
		// Headers map estimate
		for k, v := range pkt.Headers {
			size += int64(len(k) + headerValueEstimate) // key + approximate value size
			if s, ok := v.(string); ok {
				size += int64(len(s))
			}
		}
	}

	// Stats size (relatively small)
	size += int64(statsOverhead) // Approximate stats overhead

	return size
}

func (c *pcapCache) Set(id string, result *PcapAnalysisResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Calculate size of new entry
	newSize := estimateResultSize(result)

	// If this single entry exceeds max memory, don't cache it
	if newSize > c.maxMemory {
		return
	}

	// Remove existing entry with same ID if present
	if existing, ok := c.entries[id]; ok {
		c.currentMem -= existing.memSize
		delete(c.entries, id)
	}

	// Evict oldest entries until we have room (both count and memory)
	for len(c.entries) >= c.maxEntries || c.currentMem+newSize > c.maxMemory {
		if len(c.entries) == 0 {
			break
		}

		var oldestID string
		var oldestTime time.Time

		for entryID, entry := range c.entries {
			if oldestID == "" || entry.addedAt.Before(oldestTime) {
				oldestID = entryID
				oldestTime = entry.addedAt
			}
		}

		if oldestID != "" {
			c.currentMem -= c.entries[oldestID].memSize
			delete(c.entries, oldestID)
		} else {
			break
		}
	}

	// Add new entry
	c.entries[id] = &pcapCacheEntry{
		result:  result,
		memSize: newSize,
		addedAt: time.Now(),
	}
	c.currentMem += newSize
}

func (c *pcapCache) Get(id string) (*PcapAnalysisResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[id]
	if !ok {
		return nil, false
	}

	return entry.result, true
}

// MemoryUsage returns the current memory usage of the cache.
func (c *pcapCache) MemoryUsage() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentMem
}

// EntryCount returns the number of entries in the cache.
func (c *pcapCache) EntryCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// ============================================================================
// PCAP Analysis Handlers
// ============================================================================

// decodePcapUpload parses the JSON body, decodes the base64 payload, and validates PCAP magic.
// Returns the raw PCAP bytes, the parsed request, and true on success.
func decodePcapUpload(w http.ResponseWriter, r *http.Request) ([]byte, PcapUploadRequest, bool) {
	var req PcapUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == ErrMsgRequestBodyTooLarge {
			writeError(w, r, http.StatusRequestEntityTooLarge, "request_too_large",
				"PCAP file too large (max 100MB)", nil)
			return nil, req, false
		}
		writeError(w, r, http.StatusBadRequest, "invalid_request",
			"Invalid JSON request body", nil)
		return nil, req, false
	}

	if req.Data == "" {
		writeError(w, r, http.StatusBadRequest, "missing_data",
			"PCAP data is required", nil)
		return nil, req, false
	}

	pcapData, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_base64",
			"Invalid base64 encoded data", nil)
		return nil, req, false
	}

	// The shipped examples/captures/*.pcap files are mislabelled snoop
	// (RFC 1761) captures. Transparently convert to classic pcap so the
	// rest of the analyser — which is built around libpcap-style records
	// via gopacket — stays unchanged.
	if hasSnoopMagic(pcapData) {
		converted, convErr := snoopToPCAP(pcapData)
		if convErr != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_snoop",
				fmt.Sprintf("snoop capture could not be converted: %v", convErr), nil)
			return nil, req, false
		}
		pcapData = converted
	}

	if validateErr := validatePCAPStructure(pcapData); validateErr != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_pcap",
			validateErr.Error(), nil)
		return nil, req, false
	}

	return pcapData, req, true
}

// handlePcapUpload handles POST /api/v1/pcap/upload.
func (s *Server) handlePcapUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed",
			"Only POST method is allowed", nil)
		return
	}

	// Limit request size
	r.Body = http.MaxBytesReader(w, r.Body, MaxPCAPUploadSize)

	pcapData, req, ok := decodePcapUpload(w, r)
	if !ok {
		return
	}

	// Generate analysis ID from content hash
	hash := sha256.Sum256(pcapData)
	analysisID := hex.EncodeToString(hash[:8])

	// Check if we already have this analysis cached
	if existing, found := globalPcapCache.Get(analysisID); found {
		s.writeJSON(w, PcapUploadResponse{
			Success:    true,
			AnalysisID: analysisID,
			Message:    fmt.Sprintf("Analysis retrieved from cache (%d packets)", len(existing.Packets)),
		})
		return
	}

	result, err := analyzePcapData(pcapData, req.Filename)
	if err != nil {
		s.logger.Error("[API] PCAP analysis failed", "error", err, "filename", req.Filename)
		writeError(w, r, http.StatusInternalServerError, "analysis_failed",
			"Failed to analyze PCAP file", nil)
		return
	}

	result.ID = analysisID
	result.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	globalPcapCache.Set(analysisID, result)

	s.writeJSON(w, PcapUploadResponse{
		Success:    true,
		AnalysisID: analysisID,
		Message:    fmt.Sprintf("Successfully analyzed %d packets", len(result.Packets)),
	})
}

// handlePcapAnalysis handles GET /api/v1/pcap/{id}.
func (s *Server) handlePcapAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed",
			"Only GET method is allowed", nil)

		return
	}

	// Extract analysis ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/pcap/")
	analysisID := strings.TrimSuffix(path, "/")

	if analysisID == "" {
		writeError(w, r, http.StatusBadRequest, "missing_id",
			"Analysis ID is required", nil)

		return
	}

	// Look up in cache
	result, ok := globalPcapCache.Get(analysisID)
	if !ok {
		writeError(w, r, http.StatusNotFound, "not_found",
			"Analysis not found. It may have expired or was never created.", nil)

		return
	}

	s.writeJSON(w, result)
}

// ============================================================================
// PCAP Analysis Logic
// ============================================================================

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

func topNFromMap(m map[string]int, n int) []PcapIPCount {
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

	result := make([]PcapIPCount, 0, n)
	for i := 0; i < len(sorted) && i < n; i++ {
		result = append(result, PcapIPCount{
			IP:    sorted[i].key,
			Count: sorted[i].value,
		})
	}

	return result
}
