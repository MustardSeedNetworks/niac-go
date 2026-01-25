package interactive

import (
	tea "github.com/charmbracelet/bubbletea"
)

// keyAction holds mappings for keys to their handler functions.
type keyAction func() (tea.Model, tea.Cmd)

// handleKeyMsg routes key message to appropriate handler.
func (m *model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle modal input modes first
	if m.searchMode {
		return m.handleSearchInput(msg)
	}

	if m.showExport {
		return m.handleExportInput(msg)
	}

	if m.showAlertConfig {
		return m.handleAlertConfigInput(msg)
	}

	if m.showPcapReplay {
		return m.handlePcapReplayInput(msg)
	}

	if m.showHistory {
		return m.handleHistoryInput(msg)
	}

	if m.showSnmpWalk {
		return m.handleSnmpWalkInput(msg)
	}

	if m.showDeviceConfig {
		return m.handleDeviceConfigInput(msg)
	}

	if m.valueInputMode {
		return m.handleValueInput(msg)
	}

	// Handle normal key input
	return m.handleNormalKeyInput(msg)
}

// handleNKeyInput handles the 'n' key based on current context.
func (m *model) handleNKeyInput() (tea.Model, tea.Cmd) {
	if m.showHexDump && len(m.packetBuffer) > 0 {
		return m.handleHexDumpNextPacket()
	}
	m.toggleNeighborView()
	return m, nil
}

// handlePKeyInput handles the 'p' key based on current context.
func (m *model) handlePKeyInput() (tea.Model, tea.Cmd) {
	if m.showHexDump && len(m.packetBuffer) > 0 {
		return m.handleHexDumpPrevPacket()
	}
	return m, nil
}

// handleCKeyInput handles the 'c' key based on current context.
func (m *model) handleCKeyInput() (tea.Model, tea.Cmd) {
	if m.showTemplates && !m.showTemplatePreview {
		return m.handleTemplateAction()
	}
	return m.handleClearErrors()
}

// handleNormalKeyInput handles keyboard input in normal mode.
func (m *model) handleNormalKeyInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Quick dispatch for keys with simple handlers
	simpleHandlers := map[string]keyAction{
		"v": m.handleValidationToggle,
		"t": m.handleTemplateToggle,
		"C": m.handleConfigDiffToggle,
		"e": m.handleConfigEdit,
		"E": m.handleExportToggle,
		"a": m.handleAlertConfigToggle,
		"T": m.handleTopologyToggle,
		"P": m.handlePcapReplayToggle,
		"H": m.handleHistoryToggle,
		"W": m.handleSnmpWalkToggle,
		"F": m.handleDeviceConfigToggle,
		"/": m.handleSearchToggle,
		"i": m.handleMenuToggle,
		"D": m.handleDeviceCycle,
		"d": m.handleDebugLevelCycle,
		"r": m.handleReload,
		"l": m.handleLogsToggle,
		"s": m.handleStatsToggle,
		"x": m.handleHexDumpToggle,
	}

	if handler, ok := simpleHandlers[key]; ok {
		return handler()
	}

	// Keys with more complex logic
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case keyEsc:
		return m.handleEscapeKey()
	case "h", "?":
		return m.handleHelpToggle()
	case "N":
		m.toggleNeighborView()
		return m, nil
	case "n":
		return m.handleNKeyInput()
	case "p":
		return m.handlePKeyInput()
	case "c":
		return m.handleCKeyInput()
	case "up":
		return m.handleUpKey()
	case keyDown:
		return m.handleDownKey()
	case "pgup":
		return m.handlePageUp()
	case "pgdown":
		return m.handlePageDown()
	case keyEnter:
		return m.handleEnterKey()
	case "1", "2", "3", "4", "5", "6", "7":
		return m.handleQuickErrorInjection(key)
	}

	return m, nil
}
