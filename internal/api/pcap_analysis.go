package api

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

// dirPermSecure is the permission mode for secure directories (owner rwx only).
// This is intentionally 0700 for directories (not 0600) because directories
// require execute permission to access their contents.
const dirPermSecure fs.FileMode = 0o700

// Analysis constants.
const topNCount = 10 // top N count for sources/destinations

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
