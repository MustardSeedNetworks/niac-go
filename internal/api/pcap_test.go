package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPcapCacheSetAndGet(t *testing.T) {
	cache := &pcapCache{
		entries:    make(map[string]*pcapCacheEntry),
		maxEntries: 10,
		maxMemory:  MaxPcapCacheMemory,
		currentMem: 0,
	}

	result := &PcapAnalysisResult{
		ID:       "test-id",
		Filename: "test.pcap",
		FileSize: 1024,
		Packets:  []PcapPacket{},
		Stats:    PcapStats{Protocols: make(map[string]int)},
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

func TestPcapCacheGetMiss(t *testing.T) {
	cache := &pcapCache{
		entries:    make(map[string]*pcapCacheEntry),
		maxEntries: 10,
		maxMemory:  MaxPcapCacheMemory,
		currentMem: 0,
	}

	_, ok := cache.Get("nonexistent")
	if ok {
		t.Error("should not find nonexistent entry")
	}
}

func TestPcapCacheEviction(t *testing.T) {
	cache := &pcapCache{
		entries:    make(map[string]*pcapCacheEntry),
		maxEntries: 2,
		maxMemory:  MaxPcapCacheMemory,
		currentMem: 0,
	}

	for i := range 3 {
		result := &PcapAnalysisResult{
			ID:       strings.Repeat("x", i+1),
			Filename: "test.pcap",
			Packets:  []PcapPacket{},
			Stats:    PcapStats{Protocols: make(map[string]int)},
		}
		cache.Set(result.ID, result)
	}

	if cache.EntryCount() > 2 {
		t.Errorf("cache should have at most 2 entries, got %d", cache.EntryCount())
	}
}

func TestPcapCacheReplaceExisting(t *testing.T) {
	cache := &pcapCache{
		entries:    make(map[string]*pcapCacheEntry),
		maxEntries: 10,
		maxMemory:  MaxPcapCacheMemory,
		currentMem: 0,
	}

	result1 := &PcapAnalysisResult{
		ID:       "same-id",
		Filename: "first.pcap",
		Packets:  []PcapPacket{},
		Stats:    PcapStats{Protocols: make(map[string]int)},
	}
	cache.Set("same-id", result1)

	result2 := &PcapAnalysisResult{
		ID:       "same-id",
		Filename: "second.pcap",
		Packets:  []PcapPacket{},
		Stats:    PcapStats{Protocols: make(map[string]int)},
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

func TestPcapCacheMemoryUsage(t *testing.T) {
	cache := &pcapCache{
		entries:    make(map[string]*pcapCacheEntry),
		maxEntries: 10,
		maxMemory:  MaxPcapCacheMemory,
		currentMem: 0,
	}

	if cache.MemoryUsage() != 0 {
		t.Errorf("initial memory usage = %d, want 0", cache.MemoryUsage())
	}

	result := &PcapAnalysisResult{
		ID:       "test",
		Filename: "test.pcap",
		Packets:  []PcapPacket{},
		Stats:    PcapStats{Protocols: make(map[string]int)},
	}
	cache.Set("test", result)

	if cache.MemoryUsage() == 0 {
		t.Error("memory usage should be > 0 after adding entry")
	}
}

func TestEstimateResultSize(t *testing.T) {
	t.Run("nil result", func(t *testing.T) {
		if size := estimateResultSize(nil); size != 0 {
			t.Errorf("estimateResultSize(nil) = %d, want 0", size)
		}
	})

	t.Run("empty result", func(t *testing.T) {
		result := &PcapAnalysisResult{
			Packets: []PcapPacket{},
			Stats:   PcapStats{Protocols: make(map[string]int)},
		}
		size := estimateResultSize(result)
		if size <= 0 {
			t.Errorf("estimateResultSize(empty) = %d, want > 0", size)
		}
	})

	t.Run("result with packets", func(t *testing.T) {
		result := &PcapAnalysisResult{
			ID:       "test-id",
			Filename: "test.pcap",
			Packets: []PcapPacket{
				{
					ID:       "1-abc",
					SourceIP: "192.168.1.1",
					DestIP:   "192.168.1.2",
					Protocol: "TCP",
					RawData:  strings.Repeat("ab", 64),
					Headers:  map[string]any{"key": "value"},
				},
			},
			Stats: PcapStats{Protocols: make(map[string]int)},
		}
		size := estimateResultSize(result)
		emptyResult := &PcapAnalysisResult{
			Packets: []PcapPacket{},
			Stats:   PcapStats{Protocols: make(map[string]int)},
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

func TestPcapPacketCollector(t *testing.T) {
	collector := newPcapPacketCollector()

	if collector == nil {
		t.Fatal("newPcapPacketCollector returned nil")
	}

	if len(collector.sourceCounts) != 0 {
		t.Errorf("initial sourceCounts = %d, want 0", len(collector.sourceCounts))
	}
}

func TestHandlePcapUploadMethodNotAllowed(t *testing.T) {
	server, _ := newTestServer(t)

	// Method gating moved from the handler to the route registry (ADR-0002).
	// Exercise the same methodGate wrapper register() composes for the
	// POST-only /api/v1/pcap/upload route.
	gated := server.methodGate([]string{http.MethodPost}, server.handlePcapUpload)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pcap/upload", nil)
	gated(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /pcap/upload: status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodPost {
		t.Errorf("Allow header = %q, want %q", allow, http.MethodPost)
	}
}

func TestHandlePcapUploadMissingData(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pcap/upload",
		strings.NewReader(`{"filename":"test.pcap","data":""}`))
	server.handlePcapUpload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /pcap/upload missing data: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlePcapUploadInvalidBase64(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pcap/upload",
		strings.NewReader(`{"filename":"test.pcap","data":"!!!invalid!!!"}`))
	server.handlePcapUpload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /pcap/upload invalid base64: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlePcapUploadBadJSON(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pcap/upload",
		strings.NewReader("not valid json"))
	server.handlePcapUpload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /pcap/upload bad json: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlePcapAnalysisMethodNotAllowed(t *testing.T) {
	server, _ := newTestServer(t)

	// Method gating moved from the handler to the route registry (ADR-0002).
	// Exercise the methodGate wrapper register() composes for the GET-only
	// /api/v1/pcap/ route.
	gated := server.methodGate([]string{http.MethodGet}, server.handlePcapAnalysis)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pcap/test-id", nil)
	gated(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /pcap/{id}: status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodGet {
		t.Errorf("Allow header = %q, want %q", allow, http.MethodGet)
	}
}

func TestHandlePcapAnalysisMissingID(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pcap/", nil)
	server.handlePcapAnalysis(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET /pcap/ missing ID: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandlePcapAnalysisNotFound(t *testing.T) {
	server, _ := newTestServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pcap/nonexistent-id", nil)
	server.handlePcapAnalysis(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /pcap/nonexistent: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetTCPFlags(_ *testing.T) {
	// This uses gopacket types, so test via string matching on output
	// The function is indirectly tested through parsePacket tests
}

func TestPcapUploadResponseJSON(t *testing.T) {
	resp := PcapUploadResponse{
		Success:    true,
		AnalysisID: "abc123",
		Message:    "test message",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded PcapUploadResponse
	if err = json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.AnalysisID != "abc123" {
		t.Errorf("AnalysisID = %q, want %q", decoded.AnalysisID, "abc123")
	}
}
