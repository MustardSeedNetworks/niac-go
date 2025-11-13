package interactive

import (
	"net"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/errors"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// TestFormatDuration tests duration formatting
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "00:00:00"},
		{"one second", 1 * time.Second, "00:00:01"},
		{"one minute", 1 * time.Minute, "00:01:00"},
		{"one hour", 1 * time.Hour, "01:00:00"},
		{"complex", 2*time.Hour + 34*time.Minute + 56*time.Second, "02:34:56"},
		{"max seconds", 59 * time.Second, "00:00:59"},
		{"max minutes", 59 * time.Minute, "00:59:00"},
		{"24 hours", 24 * time.Hour, "24:00:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %s, expected %s", tt.duration, result, tt.expected)
			}
		})
	}
}

// TestGetDebugLevelName tests debug level name mapping
func TestGetDebugLevelName(t *testing.T) {
	tests := []struct {
		level    int
		expected string
	}{
		{0, "QUIET"},
		{1, "NORMAL"},
		{2, "VERBOSE"},
		{3, "DEBUG"},
		{4, "UNKNOWN"},
		{-1, "UNKNOWN"},
		{99, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := getDebugLevelName(tt.level)
			if result != tt.expected {
				t.Errorf("getDebugLevelName(%d) = %s, expected %s", tt.level, result, tt.expected)
			}
		})
	}
}

// createTestModel creates a test model with basic config
func createTestModel() model {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "test-device",
				Type:        "router",
				MACAddress:  mac,
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
		},
	}

	sm := errors.NewStateManager()

	return model{
		cfg:           cfg,
		stateManager:  sm,
		interfaceName: "eth0",
		debugLevel:    0,
		menuItems: []string{
			"1. Inject FCS Errors (50%)",
			"2. Inject Packet Discards (25%)",
			"3. Clear All Errors",
		},
		startTime: time.Now(),
		debugLogs: make([]string, 0, 100),
	}
}

// TestModel_Init tests model initialization
func TestModel_Init(t *testing.T) {
	m := createTestModel()

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() returned nil command")
	}
}

// TestModel_DebugLog tests debug log functionality
func TestModel_DebugLog(t *testing.T) {
	m := createTestModel()

	// Add a log entry
	m.addDebugLog("test message")

	if len(m.debugLogs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(m.debugLogs))
	}

	if !strings.Contains(m.debugLogs[0], "test message") {
		t.Errorf("Log entry doesn't contain message: %s", m.debugLogs[0])
	}

	// Verify timestamp is included
	if !strings.Contains(m.debugLogs[0], "[") || !strings.Contains(m.debugLogs[0], "]") {
		t.Error("Log entry should contain timestamp in brackets")
	}
}

// TestModel_DebugLog_MaxSize tests log size limit
func TestModel_DebugLog_MaxSize(t *testing.T) {
	m := createTestModel()

	// Add 150 log entries (more than the 100 limit)
	for i := 0; i < 150; i++ {
		m.addDebugLog("test message")
	}

	if len(m.debugLogs) != 100 {
		t.Errorf("Expected log limit of 100, got %d", len(m.debugLogs))
	}
}

// TestModel_Update_QuitKey tests quit key handling
func TestModel_Update_QuitKey(t *testing.T) {
	m := createTestModel()

	// Test 'q' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("Quit command should not be nil")
	}
}

// TestModel_Update_DebugCycle tests debug level cycling
func TestModel_Update_DebugCycle(t *testing.T) {
	m := createTestModel()
	m.debugLevel = 0

	// Press 'd' four times to cycle through all levels
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}

	for i := 1; i <= 4; i++ {
		result, _ := m.Update(msg)
		m = result.(model)
		expected := i % 4
		if m.debugLevel != expected {
			t.Errorf("After %d presses, expected debug level %d, got %d", i, expected, m.debugLevel)
		}
	}
}

// TestModel_Update_MenuToggle tests menu toggle
func TestModel_Update_MenuToggle(t *testing.T) {
	m := createTestModel()
	m.menuVisible = false

	// Press 'i' to open menu
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	result, _ := m.Update(msg)
	m = result.(model)

	if !m.menuVisible {
		t.Error("Menu should be visible after pressing 'i'")
	}

	// Press 'i' again to close menu
	result, _ = m.Update(msg)
	m = result.(model)

	if m.menuVisible {
		t.Error("Menu should be hidden after pressing 'i' again")
	}
}

// TestModel_Update_HelpToggle tests help screen toggle
func TestModel_Update_HelpToggle(t *testing.T) {
	m := createTestModel()

	// Press 'h' to open help
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	result, _ := m.Update(msg)
	m = result.(model)

	if !m.showHelp {
		t.Error("Help should be visible after pressing 'h'")
	}

	// Other overlays should be closed
	if m.showLogs || m.showStats || m.menuVisible {
		t.Error("Other overlays should be closed when help is shown")
	}
}

// TestModel_Update_LogsToggle tests logs viewer toggle
func TestModel_Update_LogsToggle(t *testing.T) {
	m := createTestModel()

	// Press 'l' to open logs
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	result, _ := m.Update(msg)
	m = result.(model)

	if !m.showLogs {
		t.Error("Logs should be visible after pressing 'l'")
	}

	// Other overlays should be closed
	if m.showHelp || m.showStats || m.menuVisible {
		t.Error("Other overlays should be closed when logs is shown")
	}
}

// TestModel_Update_StatsToggle tests statistics viewer toggle
func TestModel_Update_StatsToggle(t *testing.T) {
	m := createTestModel()

	// Press 's' to open stats
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	result, _ := m.Update(msg)
	m = result.(model)

	if !m.showStats {
		t.Error("Stats should be visible after pressing 's'")
	}

	// Other overlays should be closed
	if m.showHelp || m.showLogs || m.menuVisible {
		t.Error("Other overlays should be closed when stats is shown")
	}
}

// TestModel_Update_ClearErrors tests clear all errors command
func TestModel_Update_ClearErrors(t *testing.T) {
	m := createTestModel()

	// Inject some errors first
	m.stateManager.SetError("192.168.1.1", "eth0", errors.ErrorTypeFCS, 50)
	m.errorsActive = 1

	// Press 'c' to clear errors
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	result, _ := m.Update(msg)
	m = result.(model)

	if m.errorsActive != 0 {
		t.Errorf("Expected 0 active errors, got %d", m.errorsActive)
	}

	// Verify state manager is cleared
	states := m.stateManager.GetAllStates()
	if len(states) != 0 {
		t.Errorf("Expected 0 states in state manager, got %d", len(states))
	}
}

// TestModel_Update_MenuNavigation tests menu navigation
func TestModel_Update_MenuNavigation(t *testing.T) {
	m := createTestModel()
	m.menuVisible = true
	m.selectedItem = 0

	// Press down arrow
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	result, _ := m.Update(downMsg)
	m = result.(model)

	if m.selectedItem != 1 {
		t.Errorf("Expected selected item 1, got %d", m.selectedItem)
	}

	// Press down arrow again
	result, _ = m.Update(downMsg)
	m = result.(model)

	if m.selectedItem != 2 {
		t.Errorf("Expected selected item 2, got %d", m.selectedItem)
	}

	// Try to go past end (should stay at 2)
	result, _ = m.Update(downMsg)
	m = result.(model)

	if m.selectedItem != 2 {
		t.Errorf("Expected selected item to stay at 2, got %d", m.selectedItem)
	}

	// Press up arrow
	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	result, _ = m.Update(upMsg)
	m = result.(model)

	if m.selectedItem != 1 {
		t.Errorf("Expected selected item 1 after up, got %d", m.selectedItem)
	}
}

// TestModel_InjectError tests error injection
func TestModel_InjectError(t *testing.T) {
	m := createTestModel()

	initialInjected := m.packetsInjected
	initialActive := m.errorsActive

	// Inject FCS error
	m.injectError(errors.ErrorTypeFCS, 50)

	// Verify counters updated
	if m.packetsInjected != initialInjected+1 {
		t.Errorf("Expected packetsInjected to increase by 1, got %d", m.packetsInjected)
	}

	if m.errorsActive != initialActive+1 {
		t.Errorf("Expected errorsActive to increase by 1, got %d", m.errorsActive)
	}

	// Verify error was set in state manager
	state := m.stateManager.GetError("192.168.1.1", "eth0")
	if state == nil {
		t.Fatal("Error state not set in state manager")
	}

	if state.ErrorType != errors.ErrorTypeFCS {
		t.Errorf("Expected error type %s, got %s", errors.ErrorTypeFCS, state.ErrorType)
	}

	if state.Value != 50 {
		t.Errorf("Expected error value 50, got %d", state.Value)
	}
}

// TestModel_InjectError_NoDevices tests error injection with no devices
func TestModel_InjectError_NoDevices(t *testing.T) {
	m := createTestModel()
	m.cfg.Devices = []config.Device{} // Clear devices

	// Try to inject error
	m.injectError(errors.ErrorTypeFCS, 50)

	// Should set error status
	if !m.statusIsError {
		t.Error("Status should be error when no devices configured")
	}

	// Verify no error was injected
	states := m.stateManager.GetAllStates()
	if len(states) != 0 {
		t.Error("No errors should be injected when no devices configured")
	}
}

// TestModel_HandleMenuSelection tests menu selection handling
func TestModel_HandleMenuSelection(t *testing.T) {
	m := createTestModel()

	tests := []struct {
		name          string
		selectedItem  int
		expectedError errors.ErrorType
	}{
		{"FCS Errors", 0, errors.ErrorTypeFCS},
		{"Packet Discards", 1, errors.ErrorTypeDiscards},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m = createTestModel() // Fresh model
			m.selectedItem = tt.selectedItem

			m.handleMenuSelection()

			// New behavior: menu selection enters value input mode
			if !m.valueInputMode {
				t.Fatal("Value input mode should be enabled after menu selection")
			}

			if m.valueInputPrompt == "" {
				t.Error("Value input prompt should be set")
			}

			// Verify menu is now hidden
			if m.menuVisible {
				t.Error("Menu should be hidden when entering value input mode")
			}
		})
	}
}

// TestModel_HandleMenuSelection_ClearAll tests clear all menu option
func TestModel_HandleMenuSelection_ClearAll(t *testing.T) {
	m := createTestModel()

	// Inject an error first
	m.stateManager.SetError("192.168.1.1", "eth0", errors.ErrorTypeFCS, 50)
	m.errorsActive = 1

	// Select "Clear All Errors" (index 2 in our test menu)
	m.selectedItem = 2
	m.handleMenuSelection()

	// Verify errors cleared
	if m.errorsActive != 0 {
		t.Errorf("Expected 0 active errors after clear, got %d", m.errorsActive)
	}

	states := m.stateManager.GetAllStates()
	if len(states) != 0 {
		t.Errorf("Expected 0 states after clear, got %d", len(states))
	}
}

// TestModel_View tests view rendering
func TestModel_View(t *testing.T) {
	m := createTestModel()

	view := m.View()

	// Verify basic elements are present
	if !strings.Contains(view, "NIAC-Go Interactive Mode") {
		t.Error("View should contain title")
	}

	if !strings.Contains(view, "eth0") {
		t.Error("View should contain interface name")
	}

	if !strings.Contains(view, "test-device") {
		t.Error("View should contain device name")
	}

	if !strings.Contains(view, "Controls:") {
		t.Error("View should contain controls section")
	}
}

// TestModel_RenderMenu tests menu rendering
func TestModel_RenderMenu(t *testing.T) {
	m := createTestModel()
	m.selectedItem = 0

	menu := m.renderMenu()

	// Verify menu structure
	if !strings.Contains(menu, "Interactive Error Injection Menu") {
		t.Error("Menu should contain title")
	}

	if !strings.Contains(menu, "Inject FCS Errors") {
		t.Error("Menu should contain FCS Errors option")
	}

	// Verify selection indicator
	if !strings.Contains(menu, "→") {
		t.Error("Menu should show selection indicator")
	}
}

// TestModel_RenderHelp tests help screen rendering
func TestModel_RenderHelp(t *testing.T) {
	m := createTestModel()

	help := m.renderHelp()

	// Verify help content
	if !strings.Contains(help, "NIAC-Go Help") {
		t.Error("Help should contain title")
	}

	if !strings.Contains(help, "Keyboard Shortcuts") {
		t.Error("Help should contain keyboard shortcuts section")
	}

	if !strings.Contains(help, "Debug Levels") {
		t.Error("Help should contain debug levels section")
	}

	if !strings.Contains(help, "Error Injection Types") {
		t.Error("Help should contain error injection types")
	}

	// Verify all debug levels are documented
	for level := 0; level <= 3; level++ {
		if !strings.Contains(help, getDebugLevelName(level)) {
			t.Errorf("Help should document debug level %d (%s)", level, getDebugLevelName(level))
		}
	}
}

// TestModel_RenderLogs tests log viewer rendering
func TestModel_RenderLogs(t *testing.T) {
	m := createTestModel()

	// Test with no logs
	logs := m.renderLogs()
	if !strings.Contains(logs, "No debug logs yet") {
		t.Error("Should show 'no logs' message when empty")
	}

	// Add some logs
	m.addDebugLog("test log 1")
	m.addDebugLog("test log 2")

	logs = m.renderLogs()
	if !strings.Contains(logs, "test log 1") || !strings.Contains(logs, "test log 2") {
		t.Error("Logs should contain added log messages")
	}
}

// TestModel_RenderStatistics tests statistics rendering
func TestModel_RenderStatistics(t *testing.T) {
	m := createTestModel()
	m.stackStats = stackStatsSnapshot{
		PacketsReceived: 60,
		PacketsSent:     40,
		ARPRequests:     5,
		ARPReplies:      6,
		ICMPRequests:    7,
		ICMPReplies:     8,
		DNSQueries:      9,
		DHCPRequests:    10,
	}
	m.packetsInjected = 10
	m.errorsActive = 2

	stats := m.renderStatistics()

	// Verify statistics content
	if !strings.Contains(stats, "Detailed Statistics") {
		t.Error("Stats should contain title")
	}

	if !strings.Contains(stats, "Uptime") {
		t.Error("Stats should show uptime")
	}

	if !strings.Contains(stats, "Debug Level") {
		t.Error("Stats should show debug level")
	}

	if !strings.Contains(stats, "Total Packets") {
		t.Error("Stats should show total packets")
	}

	if !strings.Contains(stats, "eth0") {
		t.Error("Stats should show interface name")
	}
}

// TestModel_TickUpdate tests tick message handling
func TestModel_TickUpdate(t *testing.T) {
	m := createTestModel()

	// Inject an error
	m.stateManager.SetError("192.168.1.1", "eth0", errors.ErrorTypeFCS, 50)

	// Set start time to past so uptime is measurable
	m.startTime = time.Now().Add(-5 * time.Second)
	initialUptime := m.uptime

	// Send tick message
	tickMsg := tickMsg(time.Now())
	result, cmd := m.Update(tickMsg)
	m = result.(model)

	// Verify uptime updated
	if m.uptime <= initialUptime {
		t.Error("Uptime should increase after tick")
	}

	// Verify errorsActive updated
	if m.errorsActive != 1 {
		t.Errorf("Expected 1 active error, got %d", m.errorsActive)
	}

	// Verify tick command continues
	if cmd == nil {
		t.Error("Tick command should continue")
	}
}

// TestTickCmd tests tick command creation
func TestTickCmd(t *testing.T) {
	cmd := tickCmd()

	if cmd == nil {
		t.Fatal("tickCmd() returned nil")
	}

	// Execute the command to get the message
	msg := cmd()

	// Should return tickMsg
	if _, ok := msg.(tickMsg); !ok {
		t.Error("tickCmd should return tickMsg type")
	}
}

// TestRun_NilConfig tests Run with nil config
func TestRun_NilConfig(t *testing.T) {
	// Create minimal debug config
	debugConfig := logging.NewDebugConfig(0)

	// Run with nil config should handle gracefully
	// Note: This will actually start the TUI, so we can't easily test in unit tests
	// This is a placeholder for integration testing
	_ = debugConfig
}

// BenchmarkFormatDuration benchmarks duration formatting
func BenchmarkFormatDuration(b *testing.B) {
	d := 2*time.Hour + 34*time.Minute + 56*time.Second

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatDuration(d)
	}
}

// BenchmarkGetDebugLevelName benchmarks debug level name lookup
func BenchmarkGetDebugLevelName(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getDebugLevelName(i % 4)
	}
}

// BenchmarkModel_View benchmarks view rendering
func BenchmarkModel_View(b *testing.B) {
	m := createTestModel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

// BenchmarkModel_AddDebugLog benchmarks log addition
func BenchmarkModel_AddDebugLog(b *testing.B) {
	m := createTestModel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.addDebugLog("test message")
	}
}
