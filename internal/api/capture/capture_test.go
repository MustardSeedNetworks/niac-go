package capture

import (
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

func TestGetProtocolByPort(t *testing.T) {
	tests := []struct {
		name      string
		srcPort   int
		dstPort   int
		baseProto string
		want      string
	}{
		{"HTTP dest", 12345, 80, "TCP", "HTTP"},
		{"HTTPS dest", 12345, 443, "TCP", "HTTPS"},
		{"SSH dest", 12345, 22, "TCP", "SSH"},
		{"DNS dest", 12345, 53, "UDP", "DNS"},
		{"SNMP dest", 12345, 161, "UDP", "SNMP"},
		{"DHCP dest", 12345, 67, "UDP", "DHCP"},
		{"HTTP src", 80, 12345, "TCP", "HTTP"},
		{"unknown ports", 12345, 54321, "TCP", "TCP"},
		{"MySQL dest", 12345, 3306, "TCP", "MySQL"},
		{"PostgreSQL dest", 12345, 5432, "TCP", "PostgreSQL"},
		{"Redis dest", 12345, 6379, "TCP", "Redis"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getProtocolByPort(tt.srcPort, tt.dstPort, tt.baseProto)
			if got != tt.want {
				t.Errorf("getProtocolByPort(%d, %d, %q) = %q, want %q",
					tt.srcPort, tt.dstPort, tt.baseProto, got, tt.want)
			}
		})
	}
}

func TestTopNFromMap(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		result := topNFromMap(map[string]int{}, 5)
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})

	t.Run("less than N entries", func(t *testing.T) {
		m := map[string]int{"a": 3, "b": 1}
		result := topNFromMap(m, 5)
		if len(result) != 2 {
			t.Errorf("len = %d, want 2", len(result))
		}
		if result[0].IP != "a" || result[0].Count != 3 {
			t.Errorf("first = {%s, %d}, want {a, 3}", result[0].IP, result[0].Count)
		}
	})

	t.Run("more than N entries", func(t *testing.T) {
		m := map[string]int{
			"a": 10, "b": 5, "c": 3, "d": 1,
		}
		result := topNFromMap(m, 2)
		if len(result) != 2 {
			t.Errorf("len = %d, want 2", len(result))
		}
		if result[0].IP != "a" || result[0].Count != 10 {
			t.Errorf("first = {%s, %d}, want {a, 10}", result[0].IP, result[0].Count)
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
