// Package api provides PCAP analysis endpoints for the NIAC web UI.
package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// ============================================================================
// PCAP Analysis Types
// ============================================================================

// PcapPacket represents a single packet from a PCAP file
type PcapPacket struct {
	ID         string                 `json:"id"`
	Number     int                    `json:"number"`
	Timestamp  string                 `json:"timestamp"`
	SourceIP   string                 `json:"sourceIP"`
	DestIP     string                 `json:"destIP"`
	SourcePort *int                   `json:"sourcePort,omitempty"`
	DestPort   *int                   `json:"destPort,omitempty"`
	Protocol   string                 `json:"protocol"`
	Length     int                    `json:"length"`
	Info       string                 `json:"info"`
	RawData    string                 `json:"rawData,omitempty"`
	Headers    map[string]interface{} `json:"headers,omitempty"`
}

// PcapTimeRange represents the time span of a capture
type PcapTimeRange struct {
	Start      string `json:"start"`
	End        string `json:"end"`
	DurationMs int64  `json:"durationMs"`
}

// PcapIPCount represents an IP address with its packet count
type PcapIPCount struct {
	IP    string `json:"ip"`
	Count int    `json:"count"`
}

// PcapStats provides statistics about the analyzed PCAP
type PcapStats struct {
	TotalPackets    int               `json:"totalPackets"`
	TotalBytes      int64             `json:"totalBytes"`
	TimeRange       PcapTimeRange     `json:"timeRange"`
	Protocols       map[string]int    `json:"protocols"`
	TopSources      []PcapIPCount     `json:"topSources"`
	TopDestinations []PcapIPCount     `json:"topDestinations"`
}

// PcapAnalysisResult is the complete analysis of a PCAP file
type PcapAnalysisResult struct {
	ID        string       `json:"id"`
	Filename  string       `json:"filename"`
	FileSize  int64        `json:"fileSize"`
	Packets   []PcapPacket `json:"packets"`
	Stats     PcapStats    `json:"stats"`
	CreatedAt string       `json:"createdAt"`
}

// PcapUploadRequest is the request body for uploading a PCAP
type PcapUploadRequest struct {
	Filename string `json:"filename"`
	Data     string `json:"data"` // Base64 encoded PCAP data
}

// PcapUploadResponse is returned after successful upload
type PcapUploadResponse struct {
	Success    bool   `json:"success"`
	AnalysisID string `json:"analysisId"`
	Message    string `json:"message"`
}

// ============================================================================
// PCAP Analysis Cache (in-memory store for recent analyses)
// ============================================================================

type pcapCache struct {
	mu       sync.RWMutex
	analyses map[string]*PcapAnalysisResult
	maxSize  int
}

var globalPcapCache = &pcapCache{
	analyses: make(map[string]*PcapAnalysisResult),
	maxSize:  50, // Keep last 50 analyses
}

func (c *pcapCache) Set(id string, result *PcapAnalysisResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If at capacity, remove oldest entry
	if len(c.analyses) >= c.maxSize {
		var oldestID string
		var oldestTime time.Time
		for id, r := range c.analyses {
			t, _ := time.Parse(time.RFC3339, r.CreatedAt)
			if oldestID == "" || t.Before(oldestTime) {
				oldestID = id
				oldestTime = t
			}
		}
		if oldestID != "" {
			delete(c.analyses, oldestID)
		}
	}

	c.analyses[id] = result
}

func (c *pcapCache) Get(id string) (*PcapAnalysisResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result, ok := c.analyses[id]
	return result, ok
}

// ============================================================================
// PCAP Analysis Handlers
// ============================================================================

// handlePcapUpload handles POST /api/v1/pcap/upload
func (s *Server) handlePcapUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed",
			"Only POST method is allowed", nil)
		return
	}

	// Limit request size
	r.Body = http.MaxBytesReader(w, r.Body, MaxPCAPUploadSize)

	var req PcapUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == "http: request body too large" {
			writeError(w, r, http.StatusRequestEntityTooLarge, "request_too_large",
				"PCAP file too large (max 100MB)", nil)
			return
		}
		writeError(w, r, http.StatusBadRequest, "invalid_request",
			"Invalid JSON request body", nil)
		return
	}

	// Validate request
	if req.Data == "" {
		writeError(w, r, http.StatusBadRequest, "missing_data",
			"PCAP data is required", nil)
		return
	}

	// Decode base64 data
	pcapData, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_base64",
			"Invalid base64 encoded data", nil)
		return
	}

	// Validate PCAP magic number
	if err := validatePCAPMagic(pcapData); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_pcap",
			err.Error(), nil)
		return
	}

	// Generate analysis ID from content hash
	hash := sha256.Sum256(pcapData)
	analysisID := hex.EncodeToString(hash[:8])

	// Check if we already have this analysis cached
	if existing, ok := globalPcapCache.Get(analysisID); ok {
		s.writeJSON(w, PcapUploadResponse{
			Success:    true,
			AnalysisID: analysisID,
			Message:    fmt.Sprintf("Analysis retrieved from cache (%d packets)", len(existing.Packets)),
		})
		return
	}

	// Analyze the PCAP
	result, err := analyzePcapData(pcapData, req.Filename)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "analysis_failed",
			fmt.Sprintf("Failed to analyze PCAP: %v", err), nil)
		return
	}

	result.ID = analysisID
	result.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	// Cache the result
	globalPcapCache.Set(analysisID, result)

	s.writeJSON(w, PcapUploadResponse{
		Success:    true,
		AnalysisID: analysisID,
		Message:    fmt.Sprintf("Successfully analyzed %d packets", len(result.Packets)),
	})
}

// handlePcapAnalysis handles GET /api/v1/pcap/{id}
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

func analyzePcapData(data []byte, filename string) (*PcapAnalysisResult, error) {
	// Write to temp file (gopacket requires file path)
	tmpFile, err := os.CreateTemp("", "pcap-analysis-*.pcap")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

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
		Stats: PcapStats{
			Protocols: make(map[string]int),
		},
	}

	sourceCounts := make(map[string]int)
	destCounts := make(map[string]int)
	var firstTime, lastTime time.Time
	packetNum := 0

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		packetNum++

		// Get timestamp
		ts := packet.Metadata().Timestamp
		if firstTime.IsZero() || ts.Before(firstTime) {
			firstTime = ts
		}
		if ts.After(lastTime) {
			lastTime = ts
		}

		// Parse packet
		pkt := parsePacket(packet, packetNum)
		result.Packets = append(result.Packets, pkt)

		// Update stats
		result.Stats.TotalBytes += int64(pkt.Length)
		result.Stats.Protocols[pkt.Protocol]++

		if pkt.SourceIP != "" && pkt.SourceIP != "N/A" {
			sourceCounts[pkt.SourceIP]++
		}
		if pkt.DestIP != "" && pkt.DestIP != "N/A" {
			destCounts[pkt.DestIP]++
		}
	}

	result.Stats.TotalPackets = packetNum

	// Time range
	if !firstTime.IsZero() {
		result.Stats.TimeRange.Start = firstTime.UTC().Format(time.RFC3339Nano)
		result.Stats.TimeRange.End = lastTime.UTC().Format(time.RFC3339Nano)
		result.Stats.TimeRange.DurationMs = lastTime.Sub(firstTime).Milliseconds()
	}

	// Top sources
	result.Stats.TopSources = topNFromMap(sourceCounts, 10)
	result.Stats.TopDestinations = topNFromMap(destCounts, 10)

	return result, nil
}

func parsePacket(packet gopacket.Packet, num int) PcapPacket {
	pkt := PcapPacket{
		ID:        fmt.Sprintf("%d-%x", num, time.Now().UnixNano()&0xFFFF),
		Number:    num,
		Timestamp: packet.Metadata().Timestamp.UTC().Format(time.RFC3339Nano),
		Length:    packet.Metadata().Length,
		Protocol:  "Unknown",
		SourceIP:  "N/A",
		DestIP:    "N/A",
		Headers:   make(map[string]interface{}),
	}

	// Get raw hex data (first 128 bytes)
	rawLen := len(packet.Data())
	if rawLen > 128 {
		rawLen = 128
	}
	pkt.RawData = hex.EncodeToString(packet.Data()[:rawLen])

	// Parse Ethernet
	if ethLayer := packet.Layer(layers.LayerTypeEthernet); ethLayer != nil {
		eth := ethLayer.(*layers.Ethernet)
		pkt.Headers["ethernet"] = map[string]string{
			"srcMAC": eth.SrcMAC.String(),
			"dstMAC": eth.DstMAC.String(),
			"type":   eth.EthernetType.String(),
		}
	}

	// Parse IPv4
	if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		ip := ipLayer.(*layers.IPv4)
		pkt.SourceIP = ip.SrcIP.String()
		pkt.DestIP = ip.DstIP.String()
		pkt.Protocol = ip.Protocol.String()
		pkt.Headers["ipv4"] = map[string]interface{}{
			"version": ip.Version,
			"ttl":     ip.TTL,
			"id":      ip.Id,
		}
	}

	// Parse IPv6
	if ipLayer := packet.Layer(layers.LayerTypeIPv6); ipLayer != nil {
		ip := ipLayer.(*layers.IPv6)
		pkt.SourceIP = ip.SrcIP.String()
		pkt.DestIP = ip.DstIP.String()
		pkt.Protocol = ip.NextHeader.String()
		pkt.Headers["ipv6"] = map[string]interface{}{
			"version":    ip.Version,
			"hopLimit":   ip.HopLimit,
			"nextHeader": ip.NextHeader.String(),
		}
	}

	// Parse ARP
	if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
		arp := arpLayer.(*layers.ARP)
		pkt.Protocol = "ARP"
		pkt.SourceIP = fmt.Sprintf("%d.%d.%d.%d", arp.SourceProtAddress[0], arp.SourceProtAddress[1], arp.SourceProtAddress[2], arp.SourceProtAddress[3])
		pkt.DestIP = fmt.Sprintf("%d.%d.%d.%d", arp.DstProtAddress[0], arp.DstProtAddress[1], arp.DstProtAddress[2], arp.DstProtAddress[3])
		opStr := "Unknown"
		if arp.Operation == 1 {
			opStr = "Request"
		} else if arp.Operation == 2 {
			opStr = "Reply"
		}
		pkt.Info = fmt.Sprintf("ARP %s: Who has %s? Tell %s", opStr, pkt.DestIP, pkt.SourceIP)
	}

	// Parse TCP
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp := tcpLayer.(*layers.TCP)
		pkt.Protocol = "TCP"
		srcPort := int(tcp.SrcPort)
		dstPort := int(tcp.DstPort)
		pkt.SourcePort = &srcPort
		pkt.DestPort = &dstPort

		flags := getTCPFlags(tcp)
		pkt.Info = fmt.Sprintf("%d -> %d [%s] Seq=%d Ack=%d Win=%d",
			srcPort, dstPort, flags, tcp.Seq, tcp.Ack, tcp.Window)

		pkt.Headers["tcp"] = map[string]interface{}{
			"srcPort": srcPort,
			"dstPort": dstPort,
			"seq":     tcp.Seq,
			"ack":     tcp.Ack,
			"flags":   flags,
			"window":  tcp.Window,
		}

		// Check for common protocols by port
		pkt.Protocol = getProtocolByPort(srcPort, dstPort, "TCP")
	}

	// Parse UDP
	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp := udpLayer.(*layers.UDP)
		pkt.Protocol = "UDP"
		srcPort := int(udp.SrcPort)
		dstPort := int(udp.DstPort)
		pkt.SourcePort = &srcPort
		pkt.DestPort = &dstPort

		pkt.Info = fmt.Sprintf("%d -> %d Len=%d", srcPort, dstPort, udp.Length)
		pkt.Headers["udp"] = map[string]interface{}{
			"srcPort": srcPort,
			"dstPort": dstPort,
			"length":  udp.Length,
		}

		// Check for common protocols by port
		pkt.Protocol = getProtocolByPort(srcPort, dstPort, "UDP")
	}

	// Parse ICMP
	if icmpLayer := packet.Layer(layers.LayerTypeICMPv4); icmpLayer != nil {
		icmp := icmpLayer.(*layers.ICMPv4)
		pkt.Protocol = "ICMP"
		pkt.Info = fmt.Sprintf("Type=%d Code=%d", icmp.TypeCode.Type(), icmp.TypeCode.Code())
		if icmp.TypeCode.Type() == 8 {
			pkt.Info = "Echo Request (ping)"
		} else if icmp.TypeCode.Type() == 0 {
			pkt.Info = "Echo Reply (pong)"
		}
	}

	// Parse DNS
	if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
		dns := dnsLayer.(*layers.DNS)
		pkt.Protocol = "DNS"
		if dns.QR {
			pkt.Info = fmt.Sprintf("Response: %d answers", len(dns.Answers))
		} else {
			if len(dns.Questions) > 0 {
				pkt.Info = fmt.Sprintf("Query: %s", string(dns.Questions[0].Name))
			} else {
				pkt.Info = "Query"
			}
		}
	}

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

