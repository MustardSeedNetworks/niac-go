package interactive

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// handleValueInput handles keyboard input during value entry mode.
func (m *model) handleValueInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEnter:
		// Process the input
		var value int

		_, err := fmt.Sscanf(m.valueInputBuffer, "%d", &value)
		if err != nil || value < 0 || value > 100 {
			m.statusMessage = errorStyle.Render("Invalid value. Must be between 0 and 100")
			m.statusIsError = true
		} else {
			errorType := getErrorTypeByIndex(m.selectedErrorType)
			m.injectError(errorType, value)
		}

		m.valueInputMode = false
		m.valueInputBuffer = ""

		return m, nil

	case keyEsc:
		// Cancel input
		m.valueInputMode = false
		m.valueInputBuffer = ""
		m.statusMessage = "Input cancelled"
		m.statusIsError = false

		return m, nil

	case "backspace":
		if len(m.valueInputBuffer) > 0 {
			m.valueInputBuffer = m.valueInputBuffer[:len(m.valueInputBuffer)-1]
		}

		return m, nil

	default:
		// Only accept digits
		if len(msg.String()) == 1 && msg.String()[0] >= '0' && msg.String()[0] <= '9' {
			if len(m.valueInputBuffer) < maxInputDigits {
				m.valueInputBuffer += msg.String()
			}
		}

		return m, nil
	}
}

// handleSearchCharInput handles character input during search.
func (m *model) handleSearchCharInput(msg tea.KeyMsg) {
	if len(msg.String()) != 1 {
		return
	}
	char := msg.String()[0]
	if char >= 32 && char <= 126 {
		m.searchQuery += msg.String()
		m.performSearch()
	}
}

// handleSearchInput handles keyboard input during search mode.
func (m *model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.cancelSearch()
	case keyEnter:
		m.selectSearchResult()
	case "up":
		if m.selectedResult > 0 {
			m.selectedResult--
		}
	case keyDown:
		if m.selectedResult < len(m.searchResults)-1 {
			m.selectedResult++
		}
	case "tab":
		m.cycleSearchCategory()
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.performSearch()
		}
	default:
		m.handleSearchCharInput(msg)
	}

	return m, nil
}

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

// handleAlertConfigInput handles keyboard input in alert config panel.
func (m *model) handleAlertConfigInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	alertTypes := []string{"CPU", "Memory", "Disk", "PacketLoss", "Latency"}

	switch msg.String() {
	case keyEsc:
		m.showAlertConfig = false
		m.statusMessage = "Alert configuration saved"
		m.statusIsError = false

		return m, nil

	case "up":
		if m.selectedAlertType > 0 {
			m.selectedAlertType--
		}

		return m, nil

	case keyDown:
		if m.selectedAlertType < len(alertTypes)-1 {
			m.selectedAlertType++
		}

		return m, nil

	case "left":
		// Decrease threshold by 5%
		alertType := alertTypes[m.selectedAlertType]
		if m.alertThresholds == nil {
			m.alertThresholds = make(map[string]int)
		}

		if m.alertThresholds[alertType] > 0 {
			m.alertThresholds[alertType] -= 5
		}

		return m, nil

	case "right":
		// Increase threshold by 5%
		alertType := alertTypes[m.selectedAlertType]
		if m.alertThresholds == nil {
			m.alertThresholds = make(map[string]int)
		}

		if m.alertThresholds[alertType] < alertThresholdMax {
			m.alertThresholds[alertType] += 5
		}

		return m, nil

	case keyEnter:
		// Toggle individual alert
		m.alertsEnabled = !m.alertsEnabled

		return m, nil

	case " ":
		// Toggle all alerts
		m.alertsEnabled = !m.alertsEnabled

		return m, nil
	}

	return m, nil
}

// handlePcapReplayInput handles keyboard input in PCAP replay panel.
func (m *model) handlePcapReplayInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.showPcapReplay = false
		m.pcapPlaying = false
		m.statusMessage = "PCAP replay closed"
		m.statusIsError = false

		return m, nil

	case " ":
		// Toggle play/pause
		m.pcapPlaying = !m.pcapPlaying
		if m.pcapPlaying {
			m.statusMessage = "PCAP playback started"
		} else {
			m.statusMessage = "PCAP playback paused"
		}

		m.statusIsError = false

		return m, nil

	case "left":
		// Step backward one packet
		if m.pcapPlaybackIndex > 0 {
			m.pcapPlaybackIndex--
		}

		return m, nil

	case "right":
		// Step forward one packet
		if m.pcapPlaybackIndex < len(m.pcapPackets)-1 {
			m.pcapPlaybackIndex++
		}

		return m, nil

	case "+", "=":
		// Increase playback speed
		if m.pcapPlaybackSpeed < maxPlaybackSpeed {
			m.pcapPlaybackSpeed *= 2
		}

		return m, nil

	case "-", "_":
		// Decrease playback speed
		if m.pcapPlaybackSpeed > minPlaybackSpeed {
			m.pcapPlaybackSpeed /= 2
		}

		return m, nil

	case "r":
		// Restart from beginning
		m.pcapPlaybackIndex = 0
		m.statusMessage = "PCAP replay restarted"
		m.statusIsError = false

		return m, nil
	}

	return m, nil
}

// handleHistoryInput handles keyboard input in history viewer.
func (m *model) handleHistoryInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == keyEsc {
		m.showHistory = false

		return m, nil
	}

	const visibleRows = 8

	newIdx, newScroll, handled := handleScrollInput(
		key,
		m.selectedHistoryIdx,
		m.historyScrollY,
		len(m.historyEntries),
		visibleRows,
	)
	if handled {
		m.selectedHistoryIdx = newIdx
		m.historyScrollY = newScroll
	}

	return m, nil
}

// handleSnmpWalkInput handles keyboard input in SNMP walk browser.
func (m *model) handleSnmpWalkInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == keyEsc {
		m.showSnmpWalk = false

		return m, nil
	}

	const visibleRows = 12

	newIdx, newScroll, handled := handleScrollInput(
		key,
		m.selectedSnmpOid,
		m.snmpScrollY,
		len(m.snmpOidTree),
		visibleRows,
	)
	if handled {
		m.selectedSnmpOid = newIdx
		m.snmpScrollY = newScroll
	}

	return m, nil
}

// handleDeviceConfigInput handles keyboard input in device config panel.
func (m *model) handleDeviceConfigInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.showDeviceConfig = false

		return m, nil
	case "tab":
		m.deviceConfigTab = (m.deviceConfigTab + 1) % deviceConfigTabCount
		m.deviceConfigScrollY = 0

		return m, nil
	case "up":
		if m.deviceConfigScrollY > 0 {
			m.deviceConfigScrollY--
		}

		return m, nil
	case keyDown:
		m.deviceConfigScrollY++

		return m, nil
	}

	return m, nil
}
