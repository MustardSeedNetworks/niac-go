package interactive

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/apperr"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// --- display_utils.go tests ---

func TestPadPanelLine(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		check func(string) bool
	}{
		{
			"short text",
			"hello",
			func(s string) bool { return strings.HasPrefix(s, "║ hello") && strings.HasSuffix(s, "║\n") },
		},
		{
			"empty text",
			"",
			func(s string) bool { return strings.HasPrefix(s, "║ ") && strings.HasSuffix(s, "║\n") },
		},
		{
			"long text gets truncated",
			strings.Repeat("A", 100),
			func(s string) bool { return strings.Contains(s, "...") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padPanelLine(tt.text)
			if !tt.check(result) {
				t.Errorf("padPanelLine(%q) = %q, failed check", tt.text, result)
			}
		})
	}
}

func TestFitColumn(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected string
	}{
		{"short text", "hi", 10, "hi        "},
		{"exact width", "hello", 5, "hello"},
		{"long text truncated", "hello world", 8, "hello..."},
		{"zero width", "test", 0, ""},
		{"negative width", "test", -1, ""},
		{"very small width", "hello", 2, "he"},
		{"width 3 truncation", "hello", 3, "hel"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fitColumn(tt.text, tt.width)
			if got != tt.expected {
				t.Errorf("fitColumn(%q, %d) = %q, want %q", tt.text, tt.width, got, tt.expected)
			}
		})
	}
}

func TestTruncateName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short", "abc", 10, "abc"},
		{"exact", "abcde", 5, "abcde"},
		{"long", "abcdefghij", 7, "abcd..."},
		{"very short max", "abcdef", 2, "ab"},
		{"max 3 exactly", "abcdef", 3, "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateName(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncateName(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestBoolToEnabled(t *testing.T) {
	tests := []struct {
		name     string
		val      bool
		contains string
	}{
		{"true", true, "Enabled"},
		{"false", false, "Disabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boolToEnabled(tt.val)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("boolToEnabled(%v) = %q, want to contain %q", tt.val, got, tt.contains)
			}
		})
	}
}

func TestHandleScrollInput(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		selectedIdx int
		scrollY     int
		listLen     int
		visibleRows int
		wantIdx     int
		wantScrollY int
		wantHandled bool
	}{
		{"up from middle", "up", 5, 0, 10, 5, 4, 0, true},
		{"up from top", "up", 0, 0, 10, 5, 0, 0, true},
		{"up scrolls viewport", "up", 3, 3, 10, 5, 2, 2, true},
		{"down from middle", "down", 3, 0, 10, 5, 4, 0, true},
		{"down at bottom", "down", 9, 5, 10, 5, 9, 5, true},
		{"down scrolls viewport", "down", 4, 0, 10, 5, 5, 1, true},
		{"pgup", "pgup", 5, 5, 20, 5, 5, 0, true},
		{"pgup at top", "pgup", 5, 0, 20, 5, 5, 0, true},
		{"pgdown", "pgdown", 5, 0, 20, 5, 5, 5, true},
		{"pgdown clamps to max", "pgdown", 5, 14, 20, 5, 5, 15, true},
		{"unrecognized key", "tab", 3, 2, 10, 5, 3, 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, sy, handled := handleScrollInput(tt.key, tt.selectedIdx, tt.scrollY, tt.listLen, tt.visibleRows)
			if handled != tt.wantHandled {
				t.Errorf("handled = %v, want %v", handled, tt.wantHandled)
			}
			if idx != tt.wantIdx {
				t.Errorf("selectedIdx = %d, want %d", idx, tt.wantIdx)
			}
			if sy != tt.wantScrollY {
				t.Errorf("scrollY = %d, want %d", sy, tt.wantScrollY)
			}
		})
	}
}

// --- config_diff.go tests ---

func createDiffTestModel() *model {
	mac1, _ := net.ParseMAC("00:11:22:33:44:55")
	mac2, _ := net.ParseMAC("AA:BB:CC:DD:EE:FF")

	return &model{
		cfg: &config.Config{
			Devices: []config.Device{
				{
					Name:        "router1",
					Type:        "router",
					MACAddress:  mac1,
					IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
				},
				{
					Name:        "switch1",
					Type:        "switch",
					MACAddress:  mac2,
					IPAddresses: []net.IP{net.ParseIP("192.168.1.2")},
				},
			},
		},
		stateManager: apperr.NewStateManager(),
		debugLogs:    make([]string, 0, 100),
		startTime:    time.Now(),
	}
}

func TestBuildDeviceMap(t *testing.T) {
	devices := []config.Device{
		{Name: "dev1"},
		{Name: "dev2"},
		{Name: "dev3"},
	}

	result := buildDeviceMap(devices)
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}
	for _, d := range devices {
		if _, ok := result[d.Name]; !ok {
			t.Errorf("missing device %q in map", d.Name)
		}
	}
}

func TestGenerateConfigDiff_NoPreviousConfig(t *testing.T) {
	m := createDiffTestModel()
	m.previousConfig = nil

	m.generateConfigDiff()

	if len(m.configDiffContent) == 0 {
		t.Fatal("expected diff content")
	}
	found := false
	for _, line := range m.configDiffContent {
		if strings.Contains(line, "No previous configuration") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'No previous configuration' message")
	}
}

func TestGenerateConfigDiff_SameConfig(t *testing.T) {
	m := createDiffTestModel()
	m.previousConfig = m.cfg

	m.generateConfigDiff()

	found := false
	for _, line := range m.configDiffContent {
		if strings.Contains(line, "unchanged") || strings.Contains(line, "No device changes") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected unchanged or 'No device changes' message in diff")
	}
}

func TestGenerateConfigDiff_AddedDevices(t *testing.T) {
	m := createDiffTestModel()
	mac1, _ := net.ParseMAC("00:11:22:33:44:55")
	m.previousConfig = &config.Config{
		Devices: []config.Device{
			{
				Name:        "router1",
				Type:        "router",
				MACAddress:  mac1,
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
		},
	}

	m.generateConfigDiff()

	found := false
	for _, line := range m.configDiffContent {
		if strings.Contains(line, "switch1") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected added device 'switch1' in diff")
	}
}

func TestGenerateConfigDiff_RemovedDevices(t *testing.T) {
	mac1, _ := net.ParseMAC("00:11:22:33:44:55")
	mac2, _ := net.ParseMAC("AA:BB:CC:DD:EE:FF")

	m := createDiffTestModel()
	m.cfg = &config.Config{
		Devices: []config.Device{
			{
				Name:        "router1",
				Type:        "router",
				MACAddress:  mac1,
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
		},
	}
	m.previousConfig = &config.Config{
		Devices: []config.Device{
			{
				Name:        "router1",
				Type:        "router",
				MACAddress:  mac1,
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
			{
				Name:        "switch1",
				Type:        "switch",
				MACAddress:  mac2,
				IPAddresses: []net.IP{net.ParseIP("192.168.1.2")},
			},
		},
	}

	m.generateConfigDiff()

	found := false
	for _, line := range m.configDiffContent {
		if strings.Contains(line, "switch1") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected removed device 'switch1' in diff")
	}
}

func TestCompareDevices_TypeChange(t *testing.T) {
	m := createDiffTestModel()
	mac, _ := net.ParseMAC("00:11:22:33:44:55")

	prev := &config.Device{
		Name:        "dev1",
		Type:        "router",
		MACAddress:  mac,
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	curr := &config.Device{
		Name:        "dev1",
		Type:        "switch",
		MACAddress:  mac,
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}

	changes := m.compareDevices(prev, curr)
	found := false
	for _, c := range changes {
		if strings.Contains(c, "type") {
			found = true
		}
	}
	if !found {
		t.Error("expected type change in comparison")
	}
}

func TestCompareDevices_IPChange(t *testing.T) {
	m := createDiffTestModel()
	mac, _ := net.ParseMAC("00:11:22:33:44:55")

	prev := &config.Device{
		Name:        "dev1",
		Type:        "router",
		MACAddress:  mac,
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}
	curr := &config.Device{
		Name:        "dev1",
		Type:        "router",
		MACAddress:  mac,
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1"), net.ParseIP("10.0.0.1")},
	}

	changes := m.compareDevices(prev, curr)
	found := false
	for _, c := range changes {
		if strings.Contains(c, "10.0.0.1") {
			found = true
		}
	}
	if !found {
		t.Error("expected IP addition in comparison")
	}
}

func TestCompareDevices_NoChanges(t *testing.T) {
	m := createDiffTestModel()
	mac, _ := net.ParseMAC("00:11:22:33:44:55")

	dev := &config.Device{
		Name:        "dev1",
		Type:        "router",
		MACAddress:  mac,
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
	}

	changes := m.compareDevices(dev, dev)
	if len(changes) != 0 {
		t.Errorf("expected no changes, got %d: %v", len(changes), changes)
	}
}

func TestAppendDeviceCountDiff(t *testing.T) {
	tests := []struct {
		name      string
		prevCount int
		currCount int
		contains  string
	}{
		{"unchanged", 5, 5, "unchanged"},
		{"increased", 3, 5, "+2"},
		{"decreased", 5, 3, "-2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createDiffTestModel()
			m.configDiffContent = nil
			m.appendDeviceCountDiff(tt.prevCount, tt.currCount)
			if len(m.configDiffContent) == 0 {
				t.Fatal("expected diff content")
			}
			found := false
			for _, line := range m.configDiffContent {
				if strings.Contains(line, tt.contains) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %q in diff content, got %v", tt.contains, m.configDiffContent)
			}
		})
	}
}

// --- export.go tests ---

func TestGetDevicesSummary(t *testing.T) {
	m := createDiffTestModel()
	summary := m.getDevicesSummary()

	if len(summary) != 2 {
		t.Fatalf("expected 2 device summaries, got %d", len(summary))
	}

	d1 := summary[0]
	if d1["name"] != "router1" {
		t.Errorf("name = %q, want %q", d1["name"], "router1")
	}
	if d1["type"] != "router" {
		t.Errorf("type = %q, want %q", d1["type"], "router")
	}
	if d1["ip"] != "192.168.1.1" {
		t.Errorf("ip = %q, want %q", d1["ip"], "192.168.1.1")
	}
}

func TestGetDevicesSummary_NoIP(t *testing.T) {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	m := &model{
		cfg: &config.Config{
			Devices: []config.Device{
				{Name: "noip", Type: "router", MACAddress: mac},
			},
		},
	}
	summary := m.getDevicesSummary()
	if len(summary) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summary))
	}
	if summary[0]["ip"] != noIPPlaceholder {
		t.Errorf("ip = %q, want %q", summary[0]["ip"], noIPPlaceholder)
	}
}

func TestRenderExport(t *testing.T) {
	m := createDiffTestModel()
	m.interfaceName = "eth0"
	m.exportFormat = formatJSON
	m.stackStats = stackStatsSnapshot{PacketsReceived: 100, PacketsSent: 50}

	output := m.renderExport()

	checks := []string{
		"Export Statistics",
		"JSON",
		"CSV",
		"Devices:",
		"Export",
		"Cancel",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("renderExport missing %q", check)
		}
	}
}

func TestRenderExport_WithLastExport(t *testing.T) {
	m := createDiffTestModel()
	m.interfaceName = "eth0"
	m.exportFormat = formatJSON
	m.lastExportPath = "niac-stats-20260415.json"
	m.lastExportTime = time.Now()

	output := m.renderExport()
	if !strings.Contains(output, "niac-stats-20260415.json") {
		t.Error("renderExport should show last export path")
	}
}

// --- renderConfigDiff tests ---

func TestRenderConfigDiff_Empty(t *testing.T) {
	m := createDiffTestModel()
	m.configDiffContent = nil

	output := m.renderConfigDiff()
	if !strings.Contains(output, "No diff content") {
		t.Error("expected 'No diff content' message")
	}
}

func TestRenderConfigDiff_WithContent(t *testing.T) {
	m := createDiffTestModel()
	m.configDiffContent = []string{
		"=== Configuration Diff ===",
		"Device count: 2 (unchanged)",
		"--- Devices ---",
	}
	m.configDiffScrollY = 0

	output := m.renderConfigDiff()
	if !strings.Contains(output, "Configuration Diff") {
		t.Error("expected header in output")
	}
}

func TestRenderConfigDiff_Scrolling(t *testing.T) {
	m := createDiffTestModel()
	// Add enough lines to trigger scrolling
	for i := range 20 {
		m.configDiffContent = append(m.configDiffContent, strings.Repeat("x", i))
	}
	m.configDiffScrollY = 0

	output := m.renderConfigDiff()
	if !strings.Contains(output, "scroll") {
		t.Error("expected scroll indicator for long content")
	}
}

// --- hexdump.go tests ---

func TestAddPacket(t *testing.T) {
	m := createTestModel()

	data := []byte{0x00, 0x01, 0x02, 0x03}
	m.AddPacket("TCP", "192.168.1.1", "192.168.1.2", data)

	if len(m.packetBuffer) != 1 {
		t.Fatalf("expected 1 packet, got %d", len(m.packetBuffer))
	}
	pkt := m.packetBuffer[0]
	if pkt.Protocol != "TCP" {
		t.Errorf("Protocol = %q, want %q", pkt.Protocol, "TCP")
	}
	if pkt.SrcAddr != "192.168.1.1" {
		t.Errorf("SrcAddr = %q", pkt.SrcAddr)
	}
	if pkt.Length != 4 {
		t.Errorf("Length = %d, want 4", pkt.Length)
	}
	// Verify data is copied, not referenced
	data[0] = 0xFF
	if m.packetBuffer[0].Data[0] != 0x00 {
		t.Error("packet data should be a copy, not a reference")
	}
}

func TestAddPacket_BufferLimit(t *testing.T) {
	m := createTestModel()

	for i := range maxPacketBuffer + 5 {
		m.AddPacket("TCP", "src", "dst", []byte{byte(i)})
	}

	if len(m.packetBuffer) != maxPacketBuffer {
		t.Errorf("expected %d packets, got %d", maxPacketBuffer, len(m.packetBuffer))
	}
	// Oldest packets should have been trimmed
	if m.packetBuffer[0].Data[0] != 5 {
		t.Errorf("first packet data = %d, want 5", m.packetBuffer[0].Data[0])
	}
}

func TestRenderHexDump_NoPackets(t *testing.T) {
	m := createTestModel()
	output := m.renderHexDump()

	if !strings.Contains(output, "No packets captured") {
		t.Error("expected 'No packets captured' message")
	}
}

func TestRenderHexDump_WithPackets(t *testing.T) {
	m := createTestModel()
	data := make([]byte, 32)
	for i := range data {
		data[i] = byte(i)
	}
	m.AddPacket("ARP", "192.168.1.1", "192.168.1.2", data)
	m.hexDumpPacketIndex = 0

	output := m.renderHexDump()
	checks := []string{"Packet: 1/1", "ARP", "192.168.1.1", "192.168.1.2", "Offset"}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("hex dump missing %q", check)
		}
	}
}

func TestHandleHexDumpNextPacket(t *testing.T) {
	m := createTestModel()
	m.showHexDump = true
	m.AddPacket("TCP", "a", "b", []byte{1})
	m.AddPacket("UDP", "c", "d", []byte{2})
	m.hexDumpPacketIndex = 0

	m.handleHexDumpNextPacket()
	if m.hexDumpPacketIndex != 1 {
		t.Errorf("expected index 1, got %d", m.hexDumpPacketIndex)
	}

	// Wrap around
	m.handleHexDumpNextPacket()
	if m.hexDumpPacketIndex != 0 {
		t.Errorf("expected index 0 (wrap), got %d", m.hexDumpPacketIndex)
	}
}

func TestHandleHexDumpPrevPacket(t *testing.T) {
	m := createTestModel()
	m.showHexDump = true
	m.AddPacket("TCP", "a", "b", []byte{1})
	m.AddPacket("UDP", "c", "d", []byte{2})
	m.hexDumpPacketIndex = 0

	// Go backwards from 0 should wrap
	m.handleHexDumpPrevPacket()
	if m.hexDumpPacketIndex != 1 {
		t.Errorf("expected index 1 (wrap), got %d", m.hexDumpPacketIndex)
	}
}

func TestHandleHexDumpNavigation_NotActive(_ *testing.T) {
	m := createTestModel()
	m.showHexDump = false
	m.AddPacket("TCP", "a", "b", []byte{1})

	m.handleHexDumpNextPacket()
	// Should not crash when hex dump not shown

	m.handleHexDumpPrevPacket()
	// Should not crash
}

func TestHandleHexDumpNavigation_NoPackets(_ *testing.T) {
	m := createTestModel()
	m.showHexDump = true

	m.handleHexDumpNextPacket()
	m.handleHexDumpPrevPacket()
	// Should not crash with empty buffer
}

// --- search.go tests ---

func TestCancelSearch(t *testing.T) {
	m := createTestModel()
	m.searchMode = true
	m.showSearch = true
	m.searchQuery = "test"
	m.searchResults = []searchResult{{Title: "result"}}

	m.cancelSearch()

	if m.searchMode {
		t.Error("searchMode should be false")
	}
	if m.showSearch {
		t.Error("showSearch should be false")
	}
	if m.searchQuery != "" {
		t.Error("searchQuery should be empty")
	}
	if m.searchResults != nil {
		t.Error("searchResults should be nil")
	}
	if m.statusMessage != "Search cancelled" {
		t.Errorf("statusMessage = %q", m.statusMessage)
	}
}

func TestCycleSearchCategory(t *testing.T) {
	m := createTestModel()
	m.searchCategory = searchCategoryAll

	m.cycleSearchCategory()
	if m.searchCategory != searchCategoryDevices {
		t.Errorf("expected %q, got %q", searchCategoryDevices, m.searchCategory)
	}

	m.cycleSearchCategory()
	if m.searchCategory != searchCategoryLogs {
		t.Errorf("expected %q, got %q", searchCategoryLogs, m.searchCategory)
	}

	m.cycleSearchCategory()
	if m.searchCategory != searchCategoryAll {
		t.Errorf("expected %q, got %q", searchCategoryAll, m.searchCategory)
	}
}

func TestPerformSearch_EmptyQuery(t *testing.T) {
	m := createTestModel()
	m.searchQuery = ""

	m.performSearch()

	if m.searchResults != nil {
		t.Error("empty query should produce nil results")
	}
}

func TestPerformSearch_DeviceByName(t *testing.T) {
	m := createTestModel()
	m.searchQuery = "test"
	m.searchCategory = searchCategoryAll

	m.performSearch()

	found := false
	for _, r := range m.searchResults {
		if r.Category == "device" && strings.Contains(r.Title, "test-device") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find test-device by name search")
	}
}

func TestPerformSearch_DeviceByIP(t *testing.T) {
	m := createTestModel()
	m.searchQuery = "192.168"
	m.searchCategory = searchCategoryDevices

	m.performSearch()

	found := false
	for _, r := range m.searchResults {
		if r.Category == "device" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find device by IP search")
	}
}

func TestPerformSearch_Logs(t *testing.T) {
	m := createTestModel()
	m.addDebugLog("error occurred on device1")
	m.searchQuery = "error"
	m.searchCategory = searchCategoryLogs

	m.performSearch()

	found := false
	for _, r := range m.searchResults {
		if r.Category == "log" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find log entry by search")
	}
}

func TestPerformSearch_ActiveErrors(t *testing.T) {
	m := createTestModel()
	m.stateManager.SetError("192.168.1.1", "eth0", apperr.ErrorTypeFCS, 50)
	m.searchQuery = "fcs"
	m.searchCategory = searchCategoryAll

	m.performSearch()

	found := false
	for _, r := range m.searchResults {
		if r.Category == "error" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find active error by search")
	}
}

func TestSelectSearchResult_DeviceSelection(t *testing.T) {
	m := createTestModel()
	m.searchResults = []searchResult{
		{Category: "device", Title: "test-device", Index: 0},
	}
	m.selectedResult = 0

	m.selectSearchResult()

	if m.selectedDeviceIdx != 0 {
		t.Errorf("selectedDeviceIdx = %d, want 0", m.selectedDeviceIdx)
	}
	if m.searchMode {
		t.Error("searchMode should be false after selection")
	}
}

func TestSelectSearchResult_LogSelection(t *testing.T) {
	m := createTestModel()
	m.searchResults = []searchResult{
		{Category: "log", Title: "some log", Index: 0},
	}
	m.selectedResult = 0

	m.selectSearchResult()

	if !m.showLogs {
		t.Error("showLogs should be true after selecting log result")
	}
}

func TestSelectSearchResult_Empty(_ *testing.T) {
	m := createTestModel()
	m.searchResults = nil
	m.selectedResult = 0

	// Should not panic
	m.selectSearchResult()
}

func TestSelectSearchResult_OutOfBounds(_ *testing.T) {
	m := createTestModel()
	m.searchResults = []searchResult{
		{Category: "device", Title: "dev", Index: 0},
	}
	m.selectedResult = 5 // out of bounds

	// Should not panic
	m.selectSearchResult()
}

func TestRenderSearch(t *testing.T) {
	m := createTestModel()
	m.searchQuery = "test"
	m.searchCategory = searchCategoryAll

	output := m.renderSearch()
	checks := []string{"Search Mode", "Query:", "Category:", "test"}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("renderSearch missing %q", check)
		}
	}
}

func TestRenderSearch_NoResults(t *testing.T) {
	m := createTestModel()
	m.searchQuery = "nonexistent"
	m.searchResults = nil

	output := m.renderSearch()
	if !strings.Contains(output, "No results") {
		t.Error("expected 'No results' message")
	}
}

func TestRenderSearch_EmptyQuery(t *testing.T) {
	m := createTestModel()
	m.searchQuery = ""
	m.searchResults = nil

	output := m.renderSearch()
	if !strings.Contains(output, "Type to search") {
		t.Error("expected 'Type to search' prompt")
	}
}

// --- handlers.go tests ---

func TestHandleEscapeKey_Panels(t *testing.T) {
	t.Run("close validation", testEscapeCloseValidation)
	t.Run("close templates", testEscapeCloseTemplates)
	t.Run("close config diff", testEscapeCloseConfigDiff)
	t.Run("close search", testEscapeCloseSearch)
	t.Run("close export", testEscapeCloseExport)
	t.Run("close topology", testEscapeCloseTopology)
	t.Run("close alert config", testEscapeCloseAlertConfig)
	t.Run("close template preview", testEscapeCloseTemplatePreview)
}

// runEscapeKeyCase executes setup, handleEscapeKey, then check.
func runEscapeKeyCase(t *testing.T, setup func(*model), check func(*testing.T, *model)) {
	t.Helper()
	m := createTestModel()
	setup(m)
	m.handleEscapeKey()
	check(t, m)
}

func testEscapeCloseValidation(t *testing.T) {
	runEscapeKeyCase(t,
		func(m *model) { m.showValidation = true },
		func(t *testing.T, m *model) {
			if m.showValidation {
				t.Error("showValidation should be false")
			}
		},
	)
}

func testEscapeCloseTemplates(t *testing.T) {
	runEscapeKeyCase(t,
		func(m *model) { m.showTemplates = true },
		func(t *testing.T, m *model) {
			if m.showTemplates {
				t.Error("showTemplates should be false")
			}
		},
	)
}

func testEscapeCloseConfigDiff(t *testing.T) {
	runEscapeKeyCase(t,
		func(m *model) { m.showConfigDiff = true },
		func(t *testing.T, m *model) {
			if m.showConfigDiff {
				t.Error("showConfigDiff should be false")
			}
		},
	)
}

func testEscapeCloseSearch(t *testing.T) {
	runEscapeKeyCase(t,
		func(m *model) {
			m.searchMode = true
			m.showSearch = true
			m.searchQuery = "test"
		},
		func(t *testing.T, m *model) {
			if m.searchMode || m.showSearch {
				t.Error("search should be closed")
			}
			if m.searchQuery != "" {
				t.Error("searchQuery should be empty")
			}
		},
	)
}

func testEscapeCloseExport(t *testing.T) {
	runEscapeKeyCase(t,
		func(m *model) { m.showExport = true },
		func(t *testing.T, m *model) {
			if m.showExport {
				t.Error("showExport should be false")
			}
		},
	)
}

func testEscapeCloseTopology(t *testing.T) {
	runEscapeKeyCase(t,
		func(m *model) { m.showTopology = true },
		func(t *testing.T, m *model) {
			if m.showTopology {
				t.Error("showTopology should be false")
			}
		},
	)
}

func testEscapeCloseAlertConfig(t *testing.T) {
	runEscapeKeyCase(t,
		func(m *model) { m.showAlertConfig = true },
		func(t *testing.T, m *model) {
			if m.showAlertConfig {
				t.Error("showAlertConfig should be false")
			}
		},
	)
}

func testEscapeCloseTemplatePreview(t *testing.T) {
	runEscapeKeyCase(t,
		func(m *model) {
			m.showTemplatePreview = true
			m.templatePreviewContent = "some content"
		},
		func(t *testing.T, m *model) {
			if m.showTemplatePreview {
				t.Error("showTemplatePreview should be false")
			}
			if m.templatePreviewContent != "" {
				t.Error("templatePreviewContent should be empty")
			}
		},
	)
}

func TestHandlePageUp(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*model)
		wantVal func(*model) int
	}{
		{
			"config diff",
			func(m *model) { m.showConfigDiff = true; m.configDiffScrollY = 15 },
			func(m *model) int { return m.configDiffScrollY },
		},
		{
			"topology",
			func(m *model) { m.showTopology = true; m.topologyScrollY = 15 },
			func(m *model) int { return m.topologyScrollY },
		},
		{
			"hex dump",
			func(m *model) { m.showHexDump = true; m.hexDumpScrollY = 15 },
			func(m *model) int { return m.hexDumpScrollY },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createTestModel()
			tt.setup(m)
			m.handlePageUp()
			val := tt.wantVal(m)
			if val != 5 {
				t.Errorf("expected 5, got %d", val)
			}
		})
	}
}

func TestHandlePageUp_ClampsToZero(t *testing.T) {
	m := createTestModel()
	m.showConfigDiff = true
	m.configDiffScrollY = 3

	m.handlePageUp()
	if m.configDiffScrollY != 0 {
		t.Errorf("expected 0, got %d", m.configDiffScrollY)
	}
}

func TestHandlePageDown(t *testing.T) {
	m := createTestModel()
	m.showConfigDiff = true
	m.configDiffScrollY = 0

	m.handlePageDown()
	if m.configDiffScrollY != 10 {
		t.Errorf("expected 10, got %d", m.configDiffScrollY)
	}
}

func TestHandleDeviceCycle_NoDevices(t *testing.T) {
	m := createTestModel()
	m.cfg.Devices = nil

	m.handleDeviceCycle()
	if !m.statusIsError {
		t.Error("expected error status when no devices")
	}
}

func TestHandleDebugLevelCycle(t *testing.T) {
	m := createTestModel()
	m.debugLevel = 0

	m.handleDebugLevelCycle()
	if m.debugLevel != 1 {
		t.Errorf("expected 1, got %d", m.debugLevel)
	}
	if m.statusIsError {
		t.Error("should not be error status")
	}
}

func TestHandleReload_NoReloadFunc(t *testing.T) {
	m := createTestModel()
	m.reloadFunc = nil

	m.handleReload()
	if !m.statusIsError {
		t.Error("expected error when no reload function")
	}
}

func TestHandleHelpToggle_ClosesOthers(t *testing.T) {
	m := createTestModel()
	m.showLogs = true
	m.showStats = true
	m.menuVisible = true

	m.handleHelpToggle()
	if !m.showHelp {
		t.Error("showHelp should be true")
	}
	if m.showLogs || m.showStats || m.menuVisible {
		t.Error("other overlays should be closed")
	}
}

func TestHandleConfigEdit_NoPath(t *testing.T) {
	m := createTestModel()
	m.configFilePath = ""

	m.handleConfigEdit()
	if !m.statusIsError {
		t.Error("expected error when no config file path")
	}
}
