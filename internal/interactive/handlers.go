package interactive

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// handleReloadMsg processes config reload messages.
func (m *model) handleReloadMsg(msg reloadMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.statusMessage = errorStyle.Render(fmt.Sprintf("✗ Reload failed: %v", msg.err))
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
		fmt.Sprintf("✓ Reloaded configuration (%d devices)", len(msg.cfg.Devices)),
	)
	m.statusIsError = false
	m.addDebugLog(fmt.Sprintf("Config reloaded: %d devices", len(msg.cfg.Devices)))

	return m, nil
}

// handleEditorFinishedMsg processes editor completion messages.
func (m *model) handleEditorFinishedMsg(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.statusMessage = errorStyle.Render(fmt.Sprintf("✗ Editor error: %v", msg.err))
		m.statusIsError = true
	} else {
		m.statusMessage = successStyle.Render("Editor closed - press [r] to reload config")
		m.statusIsError = false
	}

	return m, nil
}

// handleEscapeKey handles escape key presses in main view.
func (m *model) handleEscapeKey() (tea.Model, tea.Cmd) {
	if m.showValidation {
		m.showValidation = false

		return m, nil
	}

	if m.showTemplatePreview {
		m.showTemplatePreview = false
		m.templatePreviewContent = ""

		return m, nil
	}

	if m.showTemplates {
		m.showTemplates = false

		return m, nil
	}

	if m.showConfigDiff {
		m.showConfigDiff = false

		return m, nil
	}

	if m.searchMode || m.showSearch {
		m.searchMode = false
		m.showSearch = false
		m.searchQuery = ""
		m.searchResults = nil
		m.statusMessage = searchCancelled
		m.statusIsError = false

		return m, nil
	}

	if m.showExport {
		m.showExport = false

		return m, nil
	}

	if m.showTopology {
		m.showTopology = false

		return m, nil
	}

	if m.showAlertConfig {
		m.showAlertConfig = false
		m.statusMessage = successStyle.Render("Alert configuration saved")
		m.statusIsError = false

		return m, nil
	}

	return m, nil
}

// handleMenuToggle toggles the interactive menu.
func (m *model) handleMenuToggle() (tea.Model, tea.Cmd) {
	if !m.requireFaultInjection() {
		return m, nil
	}
	m.menuVisible = !m.menuVisible
	if m.menuVisible {
		m.statusMessage = "Interactive menu opened - use arrow keys to navigate"
		m.statusIsError = false
	}

	return m, nil
}

// handleDeviceCycle cycles through devices.
func (m *model) handleDeviceCycle() (tea.Model, tea.Cmd) {
	if len(m.cfg.Devices) == 0 {
		m.statusMessage = errorStyle.Render("✗ No devices configured")
		m.statusIsError = true

		return m, nil
	}

	m.selectedDeviceIdx = (m.selectedDeviceIdx + 1) % len(m.cfg.Devices)
	device := m.cfg.Devices[m.selectedDeviceIdx]

	deviceIP := noIPPlaceholder
	if len(device.IPAddresses) > 0 {
		deviceIP = device.IPAddresses[0].String()
	}

	m.statusMessage = successStyle.Render(
		fmt.Sprintf("✓ Selected device: %s (%s)", device.Name, deviceIP),
	)
	m.statusIsError = false
	m.addDebugLog(fmt.Sprintf("Selected device: %s (%s)", device.Name, deviceIP))

	return m, nil
}

// handleDebugLevelCycle cycles through debug levels.
func (m *model) handleDebugLevelCycle() (tea.Model, tea.Cmd) {
	m.debugLevel = (m.debugLevel + 1) % debugLevelCount
	debugLevelName := getDebugLevelName(m.debugLevel)
	m.statusMessage = successStyle.Render(
		fmt.Sprintf("✓ Debug level: %d (%s)", m.debugLevel, debugLevelName),
	)
	m.statusIsError = false
	m.addDebugLog(fmt.Sprintf("Debug level changed to %d (%s)", m.debugLevel, debugLevelName))

	return m, nil
}

// handleReload initiates a configuration reload.
func (m *model) handleReload() (tea.Model, tea.Cmd) {
	if m.reloadFunc == nil {
		m.statusMessage = errorStyle.Render("✗ Reload not available in this mode")
		m.statusIsError = true

		return m, nil
	}

	m.statusMessage = "Reloading configuration..."
	m.statusIsError = false

	return m, reloadCmd(m.reloadFunc)
}

// handleHelpToggle toggles the help view.
func (m *model) handleHelpToggle() (tea.Model, tea.Cmd) {
	m.showHelp = !m.showHelp
	m.showLogs = false
	m.showStats = false
	m.showNeighbors = false
	m.showHexDump = false
	m.menuVisible = false

	return m, nil
}

// handleLogsToggle toggles the logs view.
func (m *model) handleLogsToggle() (tea.Model, tea.Cmd) {
	m.showLogs = !m.showLogs
	m.showHelp = false
	m.showStats = false
	m.showNeighbors = false
	m.menuVisible = false

	return m, nil
}

// handleStatsToggle toggles the statistics view.
func (m *model) handleStatsToggle() (tea.Model, tea.Cmd) {
	m.showStats = !m.showStats
	m.showHelp = false
	m.showLogs = false
	m.showHexDump = false
	m.showNeighbors = false
	m.menuVisible = false

	return m, nil
}

// handleHexDumpToggle toggles the hex dump viewer.
func (m *model) handleHexDumpToggle() (tea.Model, tea.Cmd) {
	m.showHexDump = !m.showHexDump
	m.showHelp = false
	m.showLogs = false
	m.showStats = false
	m.showNeighbors = false
	m.menuVisible = false

	if m.showHexDump {
		m.hexDumpScrollY = 0
		m.statusMessage = "Hex dump viewer opened - use arrow keys to navigate, [n]/[p] for next/prev packet"
	}

	return m, nil
}

// handleUpKey handles up arrow key navigation.
func (m *model) handleUpKey() (tea.Model, tea.Cmd) {
	switch {
	case m.showSearch && len(m.searchResults) > 0 && m.selectedResult > 0:
		m.selectedResult--
	case m.showConfigDiff && m.configDiffScrollY > 0:
		m.configDiffScrollY--
	case m.showTopology && m.topologyScrollY > 0:
		m.topologyScrollY--
	case m.showTemplates && !m.showTemplatePreview && m.selectedTemplate > 0:
		m.selectedTemplate--
	case m.menuVisible && m.selectedItem > 0:
		m.selectedItem--
	case m.showHexDump && m.hexDumpScrollY > 0:
		m.hexDumpScrollY--
	}

	return m, nil
}

// handleDownKey handles down arrow key navigation.
func (m *model) handleDownKey() (tea.Model, tea.Cmd) {
	switch {
	case m.showSearch && len(m.searchResults) > 0 && m.selectedResult < len(m.searchResults)-1:
		m.selectedResult++
	case m.showConfigDiff:
		m.configDiffScrollY++
	case m.showTopology:
		m.topologyScrollY++
	case m.showTemplates && !m.showTemplatePreview && m.selectedTemplate < len(m.templateList)-1:
		m.selectedTemplate++
	case m.menuVisible && m.selectedItem < len(m.menuItems)-1:
		m.selectedItem++
	case m.showHexDump:
		m.hexDumpScrollY++
	}

	return m, nil
}

// handlePageUp handles page up key for scrollable views.
func (m *model) handlePageUp() (tea.Model, tea.Cmd) {
	switch {
	case m.showConfigDiff:
		m.configDiffScrollY -= 10
		if m.configDiffScrollY < 0 {
			m.configDiffScrollY = 0
		}
	case m.showTopology:
		m.topologyScrollY -= 10
		if m.topologyScrollY < 0 {
			m.topologyScrollY = 0
		}
	case m.showHexDump:
		m.hexDumpScrollY -= 10
		if m.hexDumpScrollY < 0 {
			m.hexDumpScrollY = 0
		}
	}

	return m, nil
}

// handlePageDown handles page down key for scrollable views.
func (m *model) handlePageDown() (tea.Model, tea.Cmd) {
	switch {
	case m.showConfigDiff:
		m.configDiffScrollY += 10
	case m.showTopology:
		m.topologyScrollY += 10
	case m.showHexDump:
		m.hexDumpScrollY += 10
	}

	return m, nil
}

// handleEnterKey handles enter key in various contexts.
func (m *model) handleEnterKey() (tea.Model, tea.Cmd) {
	if m.showTemplates && !m.showTemplatePreview {
		return m.showTemplatePreviewAction()
	}

	if m.menuVisible {
		m.handleMenuSelection()
	}

	return m, nil
}
