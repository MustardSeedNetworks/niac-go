package capture

import (
	"sync"
	"time"
)

// Memory estimation constants for the analysis cache.
const (
	resultBaseOverhead  = 200 // approximate fixed overhead for result struct
	packetBaseOverhead  = 100 // base packet struct overhead
	headerValueEstimate = 50  // approximate value size for header map
	statsOverhead       = 500 // approximate stats overhead
)

// Cache memory limits to prevent denial-of-service attacks (SECURITY FIX #155).
const (
	// maxCacheMemory is the maximum total memory for cached PCAP analyses (500MB).
	maxCacheMemory = 500 * 1024 * 1024
	// maxCacheEntries is the maximum number of cached analyses.
	maxCacheEntries = 50
)

// cacheEntry wraps an analysis with its approximate memory size.
type cacheEntry struct {
	result  *AnalysisResult
	memSize int64 // Approximate memory size in bytes
	addedAt time.Time
}

// Cache is a bounded, concurrency-safe store of recent analyses. It evicts the
// oldest entries once either the entry-count or the total-memory cap is hit.
type Cache struct {
	mu         sync.RWMutex
	entries    map[string]*cacheEntry
	maxEntries int
	maxMemory  int64
	currentMem int64
}

// NewCache returns a Cache configured with the default DoS-protection limits.
func NewCache() *Cache {
	return &Cache{
		entries:    make(map[string]*cacheEntry),
		maxEntries: maxCacheEntries,
		maxMemory:  maxCacheMemory,
	}
}

// estimateResultSize calculates approximate memory size of an analysis result.
func estimateResultSize(result *AnalysisResult) int64 {
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

// Set stores result under id, evicting older entries to respect the caps.
func (c *Cache) Set(id string, result *AnalysisResult) {
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
	c.entries[id] = &cacheEntry{
		result:  result,
		memSize: newSize,
		addedAt: time.Now(),
	}
	c.currentMem += newSize
}

// Get returns the cached result for id, if present.
func (c *Cache) Get(id string) (*AnalysisResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[id]
	if !ok {
		return nil, false
	}

	return entry.result, true
}

// MemoryUsage returns the current memory usage of the cache.
func (c *Cache) MemoryUsage() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentMem
}

// EntryCount returns the number of entries in the cache.
func (c *Cache) EntryCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
