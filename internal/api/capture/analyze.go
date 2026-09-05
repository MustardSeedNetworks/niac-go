package capture

import (
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/niac-go/internal/packetdecode"
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

	// maxRetainedPackets bounds the per-packet rows one analysis keeps.
	//
	// An upload may be ~100MB (MaxPCAPUploadBodySize), which is roughly 1.5M
	// minimum-size frames; a Packet struct with its hex preview per frame is far
	// more memory than the capture itself, and the cache's own 500MB ceiling
	// only evicts *after* one analysis has already been built. Statistics still
	// count every packet in the file -- only the rendered rows stop here.
	maxRetainedPackets = 50000
)

// Protocol operation constants.
const ()

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
		if len(result.Packets) < maxRetainedPackets {
			result.Packets = append(result.Packets, pkt)
		} else {
			result.Truncated = true
		}
		result.Stats.TotalBytes += int64(pkt.Length)
		result.Stats.Protocols[pkt.Protocol]++

		collector.recordPacket(pkt, ts)
	}

	collector.finalizeStats(result, packetNum)

	return result, nil
}

// parsePacket turns one frame into the row the packet list renders.
//
// The decode itself is internal/packetdecode, the same call the live inspector
// makes, so a capture exported from the inspector reads back the way it was
// shown. This function only projects that decoder's map onto the analyzer's
// struct and adds what is specific to a file: a stable row number and a hex
// preview.
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

	rawLen := min(len(packet.Data()), maxRawDataBytes)
	pkt.RawData = hex.EncodeToString(packet.Data()[:rawLen])

	decoded := map[string]any{"protocol": "Unknown", "summary": ""}
	packetdecode.Enrich(decoded, packet.Data())
	applyDecoded(&pkt, decoded)

	if pkt.Info == "" {
		pkt.Info = fmt.Sprintf("%s %s -> %s", pkt.Protocol, pkt.SourceIP, pkt.DestIP)
	}

	return pkt
}

// applyDecoded copies the decoder's fields onto pkt. The decoder speaks the SSE
// wire spelling because that is the shape a browser already parses; this is the
// one place that translates.
func applyDecoded(pkt *Packet, decoded map[string]any) {
	if v, ok := decoded["protocol"].(string); ok && v != "" {
		pkt.Protocol = v
	}
	if v, ok := decoded["summary"].(string); ok && v != "" {
		pkt.Info = v
	}
	if v, ok := decoded["source_ip"].(string); ok && v != "" {
		pkt.SourceIP = v
	}
	if v, ok := decoded["dest_ip"].(string); ok && v != "" {
		pkt.DestIP = v
	}
	if v, ok := decoded["source_port"].(uint16); ok {
		port := int(v)
		pkt.SourcePort = &port
	}
	if v, ok := decoded["dest_port"].(uint16); ok {
		port := int(v)
		pkt.DestPort = &port
	}
	if v, ok := decoded["headers"].(map[string]any); ok {
		pkt.Headers = v
	}
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
	for i := range min(len(sorted), n) {
		result = append(result, IPCount{
			IP:    sorted[i].key,
			Count: sorted[i].value,
		})
	}

	return result
}
