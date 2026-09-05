package capture

import (
	"encoding/json"
	"strings"
	"testing"
)

func newTestCache(maxEntries int) *Cache {
	return &Cache{
		entries:    make(map[string]*cacheEntry),
		maxEntries: maxEntries,
		maxMemory:  maxCacheMemory,
	}
}

func TestCacheSetAndGet(t *testing.T) {
	cache := newTestCache(10)

	result := &AnalysisResult{
		ID:       "test-id",
		Filename: "test.pcap",
		FileSize: 1024,
		Packets:  []Packet{},
		Stats:    Stats{Protocols: make(map[string]int)},
	}

	cache.Set("test-id", result)

	got, ok := cache.Get("test-id")
	if !ok {
		t.Fatal("expected to find cached result")
	}
	if got.ID != "test-id" {
		t.Errorf("cached ID = %q, want %q", got.ID, "test-id")
	}
	if got.Filename != "test.pcap" {
		t.Errorf("cached Filename = %q, want %q", got.Filename, "test.pcap")
	}
}

func TestCacheGetMiss(t *testing.T) {
	cache := newTestCache(10)

	_, ok := cache.Get("nonexistent")
	if ok {
		t.Error("should not find nonexistent entry")
	}
}

func TestCacheEviction(t *testing.T) {
	cache := newTestCache(2)

	for i := range 3 {
		result := &AnalysisResult{
			ID:       strings.Repeat("x", i+1),
			Filename: "test.pcap",
			Packets:  []Packet{},
			Stats:    Stats{Protocols: make(map[string]int)},
		}
		cache.Set(result.ID, result)
	}

	if cache.EntryCount() > 2 {
		t.Errorf("cache should have at most 2 entries, got %d", cache.EntryCount())
	}
}

func TestCacheReplaceExisting(t *testing.T) {
	cache := newTestCache(10)

	result1 := &AnalysisResult{
		ID:       "same-id",
		Filename: "first.pcap",
		Packets:  []Packet{},
		Stats:    Stats{Protocols: make(map[string]int)},
	}
	cache.Set("same-id", result1)

	result2 := &AnalysisResult{
		ID:       "same-id",
		Filename: "second.pcap",
		Packets:  []Packet{},
		Stats:    Stats{Protocols: make(map[string]int)},
	}
	cache.Set("same-id", result2)

	got, ok := cache.Get("same-id")
	if !ok {
		t.Fatal("expected to find cached result")
	}
	if got.Filename != "second.pcap" {
		t.Errorf("cached Filename = %q, want %q (should be replaced)", got.Filename, "second.pcap")
	}

	if cache.EntryCount() != 1 {
		t.Errorf("cache should have 1 entry, got %d", cache.EntryCount())
	}
}

func TestCacheMemoryUsage(t *testing.T) {
	cache := newTestCache(10)

	if cache.MemoryUsage() != 0 {
		t.Errorf("initial memory usage = %d, want 0", cache.MemoryUsage())
	}

	result := &AnalysisResult{
		ID:       "test",
		Filename: "test.pcap",
		Packets:  []Packet{},
		Stats:    Stats{Protocols: make(map[string]int)},
	}
	cache.Set("test", result)

	if cache.MemoryUsage() == 0 {
		t.Error("memory usage should be > 0 after adding entry")
	}
}

func TestNewCacheDefaults(t *testing.T) {
	cache := NewCache()
	if cache.maxEntries != maxCacheEntries {
		t.Errorf("maxEntries = %d, want %d", cache.maxEntries, maxCacheEntries)
	}
	if cache.maxMemory != maxCacheMemory {
		t.Errorf("maxMemory = %d, want %d", cache.maxMemory, maxCacheMemory)
	}
	if cache.EntryCount() != 0 {
		t.Errorf("new cache EntryCount = %d, want 0", cache.EntryCount())
	}
}

func TestEstimateResultSize(t *testing.T) {
	t.Run("nil result", func(t *testing.T) {
		if size := estimateResultSize(nil); size != 0 {
			t.Errorf("estimateResultSize(nil) = %d, want 0", size)
		}
	})

	t.Run("empty result", func(t *testing.T) {
		result := &AnalysisResult{
			Packets: []Packet{},
			Stats:   Stats{Protocols: make(map[string]int)},
		}
		size := estimateResultSize(result)
		if size <= 0 {
			t.Errorf("estimateResultSize(empty) = %d, want > 0", size)
		}
	})

	t.Run("result with packets", func(t *testing.T) {
		result := &AnalysisResult{
			ID:       "test-id",
			Filename: "test.pcap",
			Packets: []Packet{
				{
					ID:       "1-abc",
					SourceIP: "192.168.1.1",
					DestIP:   "192.168.1.2",
					Protocol: "TCP",
					RawData:  strings.Repeat("ab", 64),
					Headers:  map[string]any{"key": "value"},
				},
			},
			Stats: Stats{Protocols: make(map[string]int)},
		}
		size := estimateResultSize(result)
		emptyResult := &AnalysisResult{
			Packets: []Packet{},
			Stats:   Stats{Protocols: make(map[string]int)},
		}
		emptySize := estimateResultSize(emptyResult)
		if size <= emptySize {
			t.Errorf("result with packets should be larger: %d vs %d", size, emptySize)
		}
	})
}

func TestNewPacketCollector(t *testing.T) {
	collector := newPacketCollector()

	if collector == nil {
		t.Fatal("newPacketCollector returned nil")
	}

	if len(collector.sourceCounts) != 0 {
		t.Errorf("initial sourceCounts = %d, want 0", len(collector.sourceCounts))
	}
}

// TestPacketMarshalsIPFieldsAsCamelCase guards #1493.
//
// The struct emitted `sourceIP` and `destIP` while ui/src/api/pcap-types.ts
// declares `sourceIp` and `destIp`, so both fields arrived undefined and the
// PCAP analyzer rendered every SOURCE and DESTINATION cell empty. Ports, flags
// and lengths were correct, which is what made it look like a rendering bug
// rather than a wire mismatch.
//
// Every other field on this struct already matches its TS counterpart, and the
// convention for a trailing initialism across the API is to lowercase it —
// analysisId, sessionId, webhookUrl. These two were the only pair spelled the
// other way (ADR-0007).
func TestPacketMarshalsIPFieldsAsCamelCase(t *testing.T) {
	raw, err := json.Marshal(Packet{SourceIP: "10.99.1.10", DestIP: "10.99.1.20"})
	if err != nil {
		t.Fatalf("marshal packet: %v", err)
	}
	body := string(raw)

	for _, want := range []string{`"sourceIp":"10.99.1.10"`, `"destIp":"10.99.1.20"`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s in %s", want, body)
		}
	}
	for _, unwanted := range []string{`"sourceIP"`, `"destIP"`} {
		if strings.Contains(body, unwanted) {
			t.Errorf("%s is the pre-ADR-0007 spelling the UI cannot read\n%s", unwanted, body)
		}
	}
}
