package api

import (
	"sync"
	"time"
)

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

// Memory estimation constants for pcap cache.
const (
	resultBaseOverhead  = 200 // approximate fixed overhead for result struct
	packetBaseOverhead  = 100 // base packet struct overhead
	headerValueEstimate = 50  // approximate value size for header map
	statsOverhead       = 500 // approximate stats overhead
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
