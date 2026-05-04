#!/usr/bin/env python3
"""Apply history viewer changes to interactive.go"""


def apply_changes():
    with open("pkg/interactive/interactive.go", "r") as f:
        content = f.read()

    # 1. Add HistoryEntry struct after CapturedPacket
    old = """// CapturedPacket stores packet data for hex dump viewer
type CapturedPacket struct {
	Timestamp time.Time
	Protocol  string
	SrcAddr   string
	DstAddr   string
	Length    int
	Data      []byte
}

const maxPacketBuffer = 20 // Keep last 20 packets"""

    new = """// CapturedPacket stores packet data for hex dump viewer
type CapturedPacket struct {
	Timestamp time.Time
	Protocol  string
	SrcAddr   string
	DstAddr   string
	Length    int
	Data      []byte
}

// HistoryEntry stores data about a past simulation run
type HistoryEntry struct {
	StartTime       time.Time
	EndTime         time.Time
	ConfigFile      string
	DeviceCount     int
	PacketsSent     uint64
	PacketsReceived uint64
	ErrorsInjected  int
}

const maxPacketBuffer = 20 // Keep last 20 packets"""

    content = content.replace(old, new)

    # 2. Add history state fields to model struct (after alert config state)
    old = """	// Alert configuration state
	showAlertConfig   bool
	alertThresholds   map[string]int
	alertsEnabled     bool
	selectedAlertType int
}

type tickMsg time.Time"""

    new = """	// Alert configuration state
	showAlertConfig   bool
	alertThresholds   map[string]int
	alertsEnabled     bool
	selectedAlertType int

	// History view state
	showHistory        bool
	historyEntries     []HistoryEntry
	selectedHistoryIdx int
	historyScrollY     int
}

type tickMsg time.Time"""

    content = content.replace(old, new)

    # 3. Add esc handler for closing history
    old = """			// Close alert config
			if m.showAlertConfig {
				m.showAlertConfig = false
				m.statusMessage = successStyle.Render("Alert configuration saved")
				m.statusIsError = false
				return m, nil
			}
			return m, nil

		case "v":"""

    new = """			// Close alert config
			if m.showAlertConfig {
				m.showAlertConfig = false
				m.statusMessage = successStyle.Render("Alert configuration saved")
				m.statusIsError = false
				return m, nil
			}
			// Close history view
			if m.showHistory {
				m.showHistory = false
				return m, nil
			}
			return m, nil

		case "v":"""

    content = content.replace(old, new)

    # 4. Add 'H' key handler after 'T' (topology)
    old = """		case "T":
			// Toggle topology view
			if m.showTopology {
				m.showTopology = false
			} else {
				m.generateTopologyView()
				m.showTopology = true
				m.topologyScrollY = 0
				m.closeAllOverlays()
				m.statusMessage = "Topology View - ASCII network diagram"
				m.statusIsError = false
			}
			return m, nil

		case "/":"""

    new = """		case "T":
			// Toggle topology view
			if m.showTopology {
				m.showTopology = false
			} else {
				m.generateTopologyView()
				m.showTopology = true
				m.topologyScrollY = 0
				m.closeAllOverlays()
				m.statusMessage = "Topology View - ASCII network diagram"
				m.statusIsError = false
			}
			return m, nil

		case "H":
			// Toggle history view
			if m.showHistory {
				m.showHistory = false
			} else {
				m.showHistory = true
				m.selectedHistoryIdx = 0
				m.historyScrollY = 0
				m.closeAllOverlays()
				if len(m.historyEntries) == 0 {
					m.statusMessage = "Run History - No previous runs recorded yet"
				} else {
					m.statusMessage = fmt.Sprintf("Run History - %d previous run(s)", len(m.historyEntries))
				}
				m.statusIsError = false
			}
			return m, nil

		case "/":"""

    content = content.replace(old, new)

    # 5. Add history navigation to up/down handlers
    old = """		case "up":
			if m.showSearch && len(m.searchResults) > 0 && m.selectedResult > 0 {
				m.selectedResult--
			} else if m.showConfigDiff && m.configDiffScrollY > 0 {"""

    new = """		case "up":
			if m.showHistory && len(m.historyEntries) > 0 && m.selectedHistoryIdx > 0 {
				m.selectedHistoryIdx--
			} else if m.showSearch && len(m.searchResults) > 0 && m.selectedResult > 0 {
				m.selectedResult--
			} else if m.showConfigDiff && m.configDiffScrollY > 0 {"""

    content = content.replace(old, new)

    old = """		case "down":
			if m.showSearch && len(m.searchResults) > 0 && m.selectedResult < len(m.searchResults)-1 {
				m.selectedResult++
			} else if m.showConfigDiff {"""

    new = """		case "down":
			if m.showHistory && len(m.historyEntries) > 0 && m.selectedHistoryIdx < len(m.historyEntries)-1 {
				m.selectedHistoryIdx++
			} else if m.showSearch && len(m.searchResults) > 0 && m.selectedResult < len(m.searchResults)-1 {
				m.selectedResult++
			} else if m.showConfigDiff {"""

    content = content.replace(old, new)

    # 6. Add pgup/pgdown handlers for history
    old = """		case "pgup":
			if m.showConfigDiff {
				m.configDiffScrollY -= 10"""

    new = """		case "pgup":
			if m.showHistory {
				m.historyScrollY -= 10
				if m.historyScrollY < 0 {
					m.historyScrollY = 0
				}
			} else if m.showConfigDiff {
				m.configDiffScrollY -= 10"""

    content = content.replace(old, new)

    old = """		case "pgdown":
			if m.showConfigDiff {
				m.configDiffScrollY += 10
			} else if m.showTopology {"""

    new = """		case "pgdown":
			if m.showHistory {
				m.historyScrollY += 10
			} else if m.showConfigDiff {
				m.configDiffScrollY += 10
			} else if m.showTopology {"""

    content = content.replace(old, new)

    # 7. Add enter handler for history details
    old = """		case "enter":
			if m.showTemplates && !m.showTemplatePreview {"""

    new = """		case "enter":
			if m.showHistory && len(m.historyEntries) > 0 {
				// Show details of selected history entry
				entry := m.historyEntries[m.selectedHistoryIdx]
				duration := entry.EndTime.Sub(entry.StartTime)
				m.statusMessage = fmt.Sprintf("Run %d: %s | Duration: %s | Devices: %d | TX: %d RX: %d | Errors: %d",
					m.selectedHistoryIdx+1,
					entry.StartTime.Format("2006-01-02 15:04:05"),
					formatDuration(duration),
					entry.DeviceCount,
					entry.PacketsSent,
					entry.PacketsReceived,
					entry.ErrorsInjected,
				)
				m.statusIsError = false
				return m, nil
			} else if m.showTemplates && !m.showTemplatePreview {"""

    content = content.replace(old, new)

    # 8. Add history overlay to View() - after showTopology
    old = """	// Topology overlay
	if m.showTopology {
		s.WriteString(m.renderTopology())
		s.WriteString("\\n")
	}

	// Controls - first row"""

    new = """	// Topology overlay
	if m.showTopology {
		s.WriteString(m.renderTopology())
		s.WriteString("\\n")
	}

	// History overlay
	if m.showHistory {
		s.WriteString(m.renderHistory())
		s.WriteString("\\n")
	}

	// Controls - first row"""

    content = content.replace(old, new)

    # 9. Add [H] to controls bar (second or third row)
    old = """	// Controls - second row
	s.WriteString("          ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[t]"))
	s.WriteString(" Templates  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[v]"))
	s.WriteString(" Validate  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[C]"))
	s.WriteString(" Diff  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[e]"))
	s.WriteString(" Edit  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[E]"))
	s.WriteString(" Export  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[T]"))
	s.WriteString(" Topology  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[/]"))
	s.WriteString(" Search  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[c]"))
	s.WriteString(" Clear  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("[q]"))
	s.WriteString(" Quit")"""

    new = """	// Controls - second row
	s.WriteString("          ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[t]"))
	s.WriteString(" Templates  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[v]"))
	s.WriteString(" Validate  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[C]"))
	s.WriteString(" Diff  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[e]"))
	s.WriteString(" Edit  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[E]"))
	s.WriteString(" Export  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[T]"))
	s.WriteString(" Topology  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[H]"))
	s.WriteString(" History\\n")
	// Controls - third row
	s.WriteString("          ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[/]"))
	s.WriteString(" Search  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[c]"))
	s.WriteString(" Clear  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("[q]"))
	s.WriteString(" Quit")"""

    content = content.replace(old, new)

    # 10. Add [H] to help screen
    old = """	help.WriteString("║  [T]     Network topology view (ASCII diagram)                  ║\\n")
	help.WriteString("║  [/]     Search mode (filter devices/logs by pattern)           ║\\n")"""

    new = """	help.WriteString("║  [T]     Network topology view (ASCII diagram)                  ║\\n")
	help.WriteString("║  [H]     Run history viewer (past simulation sessions)          ║\\n")
	help.WriteString("║  [/]     Search mode (filter devices/logs by pattern)           ║\\n")"""

    content = content.replace(old, new)

    # 11. Add renderHistory function before the final closing functions
    # Find a good place - after renderTopology
    render_history_func = """
// renderHistory renders the run history panel
func (m model) renderHistory() string {
	var panel strings.Builder

	panel.WriteString("\\u2554\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2557\\n")
	panel.WriteString("\\u2551                       Run History                                \\u2551\\n")
	panel.WriteString("\\u2560\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2563\\n")

	if len(m.historyEntries) == 0 {
		panel.WriteString(padPanelLine("No previous runs recorded yet."))
		panel.WriteString(padPanelLine(""))
		panel.WriteString(padPanelLine("History entries are created when simulations"))
		panel.WriteString(padPanelLine("are started and stopped."))
		panel.WriteString("\\u2560\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2563\\n")
		panel.WriteString(padPanelLine("[H] or [ESC] Close"))
		panel.WriteString("\\u255a\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u255d")
		return panel.String()
	}

	// Table header
	header := fmt.Sprintf("%s %s %s %s %s",
		fitColumn("Date/Time", 19),
		fitColumn("Duration", 10),
		fitColumn("Devices", 8),
		fitColumn("RX/TX", 15),
		fitColumn("Errs", 5),
	)
	panel.WriteString(padPanelLine(diffHeaderStyle.Render(header)))
	panel.WriteString("\\u2560\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2563\\n")

	// Calculate visible entries
	maxLines := 10
	totalEntries := len(m.historyEntries)
	startIdx := m.historyScrollY
	if startIdx >= totalEntries {
		startIdx = totalEntries - 1
		if startIdx < 0 {
			startIdx = 0
		}
	}
	endIdx := startIdx + maxLines
	if endIdx > totalEntries {
		endIdx = totalEntries
	}

	// Render entries (newest first)
	for i := startIdx; i < endIdx; i++ {
		entry := m.historyEntries[len(m.historyEntries)-1-i] // Reverse order (newest first)
		duration := entry.EndTime.Sub(entry.StartTime)
		if entry.EndTime.IsZero() {
			duration = time.Since(entry.StartTime)
		}

		dateStr := entry.StartTime.Format("2006-01-02 15:04:05")
		durationStr := formatDuration(duration)
		deviceStr := fmt.Sprintf("%d", entry.DeviceCount)
		rxTxStr := fmt.Sprintf("%d/%d", entry.PacketsReceived, entry.PacketsSent)
		errStr := fmt.Sprintf("%d", entry.ErrorsInjected)

		line := fmt.Sprintf("%s %s %s %s %s",
			fitColumn(dateStr, 19),
			fitColumn(durationStr, 10),
			fitColumn(deviceStr, 8),
			fitColumn(rxTxStr, 15),
			fitColumn(errStr, 5),
		)

		// Highlight selected row
		if i == m.selectedHistoryIdx {
			line = selectedStyle.Render("-> " + line)
		} else {
			line = "   " + line
		}

		panel.WriteString(padPanelLine(line))
	}

	// Scroll indicator
	if totalEntries > maxLines {
		panel.WriteString("\\u2560\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2563\\n")
		panel.WriteString(padPanelLine(fmt.Sprintf("Showing %d-%d of %d (use arrows/PgUp/PgDn)", startIdx+1, endIdx, totalEntries)))
	}

	// Summary stats
	panel.WriteString("\\u2560\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2563\\n")

	var totalUptime time.Duration
	for _, entry := range m.historyEntries {
		if !entry.EndTime.IsZero() {
			totalUptime += entry.EndTime.Sub(entry.StartTime)
		} else {
			totalUptime += time.Since(entry.StartTime)
		}
	}
	avgSession := time.Duration(0)
	if len(m.historyEntries) > 0 {
		avgSession = totalUptime / time.Duration(len(m.historyEntries))
	}

	panel.WriteString(padPanelLine(fmt.Sprintf("Total runs: %d  |  Total uptime: %s  |  Avg session: %s",
		len(m.historyEntries),
		formatDuration(totalUptime),
		formatDuration(avgSession),
	)))

	panel.WriteString("\\u2560\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2563\\n")
	panel.WriteString(padPanelLine("[Enter] View details  [up/down] Navigate  [H]/[ESC] Close"))
	panel.WriteString("\\u255a\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u2550\\u255d")

	return panel.String()
}

// addHistoryEntry adds a new history entry when a simulation starts
func (m *model) addHistoryEntry() {
	entry := HistoryEntry{
		StartTime:       time.Now(),
		ConfigFile:      m.configFilePath,
		DeviceCount:     len(m.cfg.Devices),
		PacketsSent:     0,
		PacketsReceived: 0,
		ErrorsInjected:  0,
	}
	m.historyEntries = append(m.historyEntries, entry)
	m.addDebugLog(fmt.Sprintf("Added history entry: %d devices", entry.DeviceCount))
}

// updateCurrentHistoryEntry updates the current (last) history entry with final stats
func (m *model) updateCurrentHistoryEntry() {
	if len(m.historyEntries) == 0 {
		return
	}
	idx := len(m.historyEntries) - 1
	m.historyEntries[idx].EndTime = time.Now()
	m.historyEntries[idx].PacketsSent = m.stackStats.PacketsSent
	m.historyEntries[idx].PacketsReceived = m.stackStats.PacketsReceived
	m.historyEntries[idx].ErrorsInjected = m.packetsInjected
}
"""

    # Actually let's insert after the last function - find a good location
    # Let's find closeAllOverlays and insert after it
    old_close = """// closeAllOverlays closes all overlay panels
func (m *model) closeAllOverlays() {
	m.showHelp = false
	m.showLogs = false
	m.showStats = false
	m.showNeighbors = false
	m.showHexDump = false
	m.showValidation = false
	m.showTemplates = false
	m.showTemplatePreview = false
	m.menuVisible = false
	m.valueInputMode = false
}"""

    new_close = """// closeAllOverlays closes all overlay panels
func (m *model) closeAllOverlays() {
	m.showHelp = false
	m.showLogs = false
	m.showStats = false
	m.showNeighbors = false
	m.showHexDump = false
	m.showValidation = false
	m.showTemplates = false
	m.showTemplatePreview = false
	m.menuVisible = false
	m.valueInputMode = false
	m.showHistory = false
}""" + render_history_func

    content = content.replace(old_close, new_close)

    with open("pkg/interactive/interactive.go", "w") as f:
        f.write(content)

    print("Changes applied successfully!")


if __name__ == "__main__":
    apply_changes()
