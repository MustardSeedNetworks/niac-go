package interactive

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/krisarmstrong/niac-go/internal/apperr"
)

// handleMenuSelection handles the selection of a menu item.
func (m *model) handleMenuSelection() {
	if m.selectedItem < 0 || m.selectedItem >= len(m.menuItems) {
		return
	}

	selection := m.menuItems[m.selectedItem]

	// Handle menu selections - now with custom value input
	switch {
	case contains(selection, "FCS Errors"):
		m.promptForValue(apperr.ErrorTypeFCS, "Enter FCS error count (0-100): ")
	case contains(selection, "Packet Discards"):
		m.promptForValue(apperr.ErrorTypeDiscards, "Enter packet discard rate (0-100): ")
	case contains(selection, "Interface Errors"):
		m.promptForValue(apperr.ErrorTypeInterface, "Enter interface error count (0-100): ")
	case contains(selection, "High Utilization"):
		m.promptForValue(apperr.ErrorTypeUtilization, "Enter utilization percentage (0-100): ")
	case contains(selection, "High CPU"):
		m.promptForValue(apperr.ErrorTypeCPU, "Enter CPU percentage (0-100): ")
	case contains(selection, "High Memory"):
		m.promptForValue(apperr.ErrorTypeMemory, "Enter memory percentage (0-100): ")
	case contains(selection, "High Disk"):
		m.promptForValue(apperr.ErrorTypeDisk, "Enter disk percentage (0-100): ")
	case contains(selection, "Clear All"):
		m.stateManager.ClearAll()
		m.statusMessage = successStyle.Render("All errors cleared")
		m.statusIsError = false
		m.errorsActive = 0
		m.addDebugLog("All error injections cleared")
	case contains(selection, "Exit"):
		m.menuVisible = false
	}
}

// contains is a helper that checks if a string contains a substring.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// exportStats exports current statistics to a file.
func (m *model) exportStats() error {
	timestamp := time.Now().Format("20060102-150405")

	var (
		filename string
		err      error
	)

	if m.exportFormat == formatJSON {
		filename = fmt.Sprintf("niac-stats-%s.json", timestamp)
		err = m.exportStatsJSON(filename)
	} else {
		filename = fmt.Sprintf("niac-stats-%s.csv", timestamp)
		err = m.exportStatsCSV(filename)
	}

	if err != nil {
		return err
	}

	m.lastExportPath = filename
	m.lastExportTime = time.Now()
	m.addDebugLog("Exported stats to " + filename)

	return nil
}

// exportStatsJSON exports statistics to JSON format.
func (m *model) exportStatsJSON(filename string) error {
	stats := map[string]any{
		"timestamp":        time.Now().Format(time.RFC3339),
		"interface":        m.interfaceName,
		"uptime_seconds":   m.uptime.Seconds(),
		"debug_level":      m.debugLevel,
		"device_count":     len(m.cfg.Devices),
		"errors_active":    m.errorsActive,
		"packets_injected": m.packetsInjected,
		"stack_stats": map[string]uint64{
			"packets_received": m.stackStats.PacketsReceived,
			"packets_sent":     m.stackStats.PacketsSent,
			"arp_requests":     m.stackStats.ARPRequests,
			"arp_replies":      m.stackStats.ARPReplies,
			"icmp_requests":    m.stackStats.ICMPRequests,
			"icmp_replies":     m.stackStats.ICMPReplies,
			"dns_queries":      m.stackStats.DNSQueries,
			"dhcp_requests":    m.stackStats.DHCPRequests,
		},
		"devices": m.getDevicesSummary(),
	}

	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stats JSON: %w", err)
	}

	if err = os.WriteFile(filename, data, 0o600); err != nil {
		return fmt.Errorf("failed to write stats file: %w", err)
	}
	return nil
}

// exportStatsCSV exports statistics to CSV format.
func (m *model) exportStatsCSV(filename string) error {
	// Create file with restricted permissions (owner-only)
	file, err := os.OpenFile(
		filename,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o600,
	)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}

	defer func() { _ = file.Close() }()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	_ = writer.Write([]string{"Metric", "Value", "Category"})

	// General stats
	_ = writer.Write([]string{"Timestamp", time.Now().Format(time.RFC3339), "General"})
	_ = writer.Write([]string{"Interface", m.interfaceName, "General"})
	_ = writer.Write([]string{"Uptime (seconds)", fmt.Sprintf("%.0f", m.uptime.Seconds()), "General"})
	_ = writer.Write([]string{"Debug Level", strconv.Itoa(m.debugLevel), "General"})
	_ = writer.Write([]string{"Device Count", strconv.Itoa(len(m.cfg.Devices)), "General"})
	_ = writer.Write([]string{"Errors Active", strconv.Itoa(m.errorsActive), "General"})
	_ = writer.Write([]string{"Packets Injected", strconv.Itoa(m.packetsInjected), "General"})

	// Stack stats
	_ = writer.Write([]string{"Packets Received", strconv.FormatUint(m.stackStats.PacketsReceived, 10), "Network"})
	_ = writer.Write([]string{"Packets Sent", strconv.FormatUint(m.stackStats.PacketsSent, 10), "Network"})
	_ = writer.Write([]string{"ARP Requests", strconv.FormatUint(m.stackStats.ARPRequests, 10), "Protocol"})
	_ = writer.Write([]string{"ARP Replies", strconv.FormatUint(m.stackStats.ARPReplies, 10), "Protocol"})
	_ = writer.Write([]string{"ICMP Requests", strconv.FormatUint(m.stackStats.ICMPRequests, 10), "Protocol"})
	_ = writer.Write([]string{"ICMP Replies", strconv.FormatUint(m.stackStats.ICMPReplies, 10), "Protocol"})
	_ = writer.Write([]string{"DNS Queries", strconv.FormatUint(m.stackStats.DNSQueries, 10), "Protocol"})
	_ = writer.Write([]string{"DHCP Requests", strconv.FormatUint(m.stackStats.DHCPRequests, 10), "Protocol"})

	// Devices
	for _, device := range m.cfg.Devices {
		ip := noIPPlaceholder
		if len(device.IPAddresses) > 0 {
			ip = device.IPAddresses[0].String()
		}

		_ = writer.Write([]string{
			device.Name,
			fmt.Sprintf("%s,%s,%s", device.Type, ip, device.MACAddress.String()),
			"Device",
		})
	}

	return nil
}

// openEditor opens the config file in the user's preferred editor.
func (m *model) openEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}

	if editor == "" {
		editor = "vi" // Default fallback
	}

	c := exec.CommandContext(
		context.Background(),
		editor,
		m.configFilePath,
	)

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// handleReloadMsg processes configuration reload messages.
func (m *model) handleReloadMsg(msg reloadMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.statusMessage = errorStyle.Render(fmt.Sprintf("Reload failed: %v", msg.err))
		m.statusIsError = true

		return m, nil
	}

	if msg.cfg == nil {
		return m, nil
	}

	// Store previous config for diff viewer
	if m.cfg != nil {
		m.previousConfig = m.cfg
	}

	m.cfg = msg.cfg
	m.selectedDeviceIdx = 0
	m.statusMessage = successStyle.Render(
		fmt.Sprintf("Reloaded configuration (%d devices)", len(msg.cfg.Devices)),
	)
	m.statusIsError = false
	m.addDebugLog(fmt.Sprintf("Config reloaded: %d devices", len(msg.cfg.Devices)))

	return m, nil
}

// handleEditorFinishedMsg processes editor completion messages.
func (m *model) handleEditorFinishedMsg(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.statusMessage = errorStyle.Render(fmt.Sprintf("Editor error: %v", msg.err))
		m.statusIsError = true
	} else {
		m.statusMessage = successStyle.Render("Editor closed - press [r] to reload config")
		m.statusIsError = false
	}

	return m, nil
}
