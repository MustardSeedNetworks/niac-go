package api

import (
	"io/fs"
	"sync"
	"time"
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
	ipv4AddrLen         = 4 // IPv4 address length in bytes (FIX #269)
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
	// DefaultPcapCacheMemory is the default max memory for cached PCAP analyses (100MB).
	// FIX #290: Reduced from 500MB to 100MB to prevent excessive memory usage.
	DefaultPcapCacheMemory = 100 * 1024 * 1024
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

// newPcapCacheWithLimit creates a new PCAP cache with configurable memory limit.
// FIX #290: Allow configurable cache memory via ServerConfig.PcapCacheMemory.
func newPcapCacheWithLimit(maxMem int64) *pcapCache {
	if maxMem <= 0 {
		maxMem = DefaultPcapCacheMemory
	}

	return &pcapCache{
		entries:    make(map[string]*pcapCacheEntry),
		maxEntries: MaxPcapCacheEntries,
		maxMemory:  maxMem,
		currentMem: 0,
	}
}

// Set adds or updates an analysis in the cache.
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

// Get retrieves an analysis from the cache.
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
