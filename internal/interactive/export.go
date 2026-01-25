package interactive

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// handleExportInput handles keyboard input during export panel.
func (m *model) handleExportInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.showExport = false
		m.statusMessage = "Export cancelled"
		m.statusIsError = false

		return m, nil

	case "j":
		m.exportFormat = formatJSON
		m.statusMessage = "Export format: JSON"
		m.statusIsError = false

		return m, nil

	case "c":
		m.exportFormat = "csv"
		m.statusMessage = "Export format: CSV"
		m.statusIsError = false

		return m, nil

	case keyEnter:
		// Perform export
		err := m.exportStats()
		if err != nil {
			m.statusMessage = errorStyle.Render(fmt.Sprintf("Export failed: %v", err))
			m.statusIsError = true
		} else {
			m.statusMessage = successStyle.Render("Exported to: " + m.lastExportPath)
			m.statusIsError = false
			m.showExport = false
		}

		return m, nil
	}

	return m, nil
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
	// SECURITY FIX #163: Create file with restricted permissions (owner-only)
	file, err := os.OpenFile(
		filename,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o600,
	) // #nosec G304 -- user-initiated
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}

	defer func() { _ = file.Close() }()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	_ = writer.Write(
		[]string{"Metric", "Value", "Category"},
	) // CSV write errors handled by writer.Error()

	// General stats
	_ = writer.Write(
		[]string{"Timestamp", time.Now().Format(time.RFC3339), "General"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"Interface", m.interfaceName, "General"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"Uptime (seconds)", fmt.Sprintf("%.0f", m.uptime.Seconds()), "General"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"Debug Level", strconv.Itoa(m.debugLevel), "General"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"Device Count", strconv.Itoa(len(m.cfg.Devices)), "General"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"Errors Active", strconv.Itoa(m.errorsActive), "General"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"Packets Injected", strconv.Itoa(m.packetsInjected), "General"},
	) // CSV write errors handled by writer.Error()

	// Stack stats
	_ = writer.Write(
		[]string{
			"Packets Received",
			strconv.FormatUint(m.stackStats.PacketsReceived, 10),
			"Network",
		},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"Packets Sent", strconv.FormatUint(m.stackStats.PacketsSent, 10), "Network"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"ARP Requests", strconv.FormatUint(m.stackStats.ARPRequests, 10), "Protocol"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"ARP Replies", strconv.FormatUint(m.stackStats.ARPReplies, 10), "Protocol"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"ICMP Requests", strconv.FormatUint(m.stackStats.ICMPRequests, 10), "Protocol"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"ICMP Replies", strconv.FormatUint(m.stackStats.ICMPReplies, 10), "Protocol"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"DNS Queries", strconv.FormatUint(m.stackStats.DNSQueries, 10), "Protocol"},
	) // CSV write errors handled by writer.Error()
	_ = writer.Write(
		[]string{"DHCP Requests", strconv.FormatUint(m.stackStats.DHCPRequests, 10), "Protocol"},
	) // CSV write errors handled by writer.Error()

	// Devices
	for _, device := range m.cfg.Devices {
		ip := noIPPlaceholder
		if len(device.IPAddresses) > 0 {
			ip = device.IPAddresses[0].String()
		}

		_ = writer.Write(
			[]string{
				device.Name,
				fmt.Sprintf("%s,%s,%s", device.Type, ip, device.MACAddress.String()),
				"Device",
			},
		) // CSV write errors handled by writer.Error()
	}

	return nil
}

// getDevicesSummary returns a summary of devices for export.
func (m *model) getDevicesSummary() []map[string]string {
	devices := make([]map[string]string, 0, len(m.cfg.Devices))
	for _, device := range m.cfg.Devices {
		ip := noIPPlaceholder
		if len(device.IPAddresses) > 0 {
			ip = device.IPAddresses[0].String()
		}

		devices = append(devices, map[string]string{
			"name": device.Name,
			"type": device.Type,
			"ip":   ip,
			"mac":  device.MACAddress.String(),
		})
	}

	return devices
}

// renderExport renders the export panel.
func (m *model) renderExport() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString("║                      Export Statistics                           ║\n")
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	panel.WriteString(padPanelLine("Export current simulation statistics to a file."))
	panel.WriteString(padPanelLine(""))

	// Format selection
	jsonSelected := "  "
	csvSelected := "  "

	if m.exportFormat == formatJSON {
		jsonSelected = selectedStyle.Render("->")
	} else {
		csvSelected = selectedStyle.Render("->")
	}

	panel.WriteString(padPanelLine(jsonSelected + " [j] JSON - Structured data format"))
	panel.WriteString(padPanelLine(csvSelected + " [c] CSV  - Spreadsheet compatible"))
	panel.WriteString(padPanelLine(""))

	// Stats preview
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine(diffHeaderStyle.Render("Data to export:")))
	panel.WriteString(padPanelLine("  Devices:         " + strconv.Itoa(len(m.cfg.Devices))))
	panel.WriteString(padPanelLine("  Active Errors:   " + strconv.Itoa(m.errorsActive)))
	panel.WriteString(
		padPanelLine(
			fmt.Sprintf(
				"  Packets RX/TX:   %d / %d",
				m.stackStats.PacketsReceived,
				m.stackStats.PacketsSent,
			),
		),
	)
	panel.WriteString(padPanelLine("  Uptime:          " + formatDuration(m.uptime)))

	// Last export info
	if !m.lastExportTime.IsZero() {
		panel.WriteString(padPanelLine(""))
		panel.WriteString(padPanelLine(successStyle.Render("Last export: " + m.lastExportPath)))
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[Enter] Export  [j] JSON  [c] CSV  [ESC] Cancel"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// handleExportToggle toggles the export panel.
func (m *model) handleExportToggle() (tea.Model, tea.Cmd) {
	if m.showExport {
		m.showExport = false

		return m, nil
	}

	m.showExport = true
	m.exportFormat = formatJSON
	m.closeAllOverlays()
	m.statusMessage = "Export Stats - Press [j] for JSON, [c] for CSV, [Enter] to save"
	m.statusIsError = false

	return m, nil
}
