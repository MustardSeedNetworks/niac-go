package interactive

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/krisarmstrong/niac-go/internal/apperr"
	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/templates"
)

// handleValidationToggle toggles the validation view.
func (m *model) handleValidationToggle() (tea.Model, tea.Cmd) {
	if m.showValidation {
		m.showValidation = false

		return m, nil
	}

	validator := config.NewValidator("current config")
	m.validationResults = validator.Validate(m.cfg)
	m.showValidation = true
	m.showHelp = false
	m.showLogs = false
	m.showStats = false
	m.showNeighbors = false
	m.showHexDump = false
	m.menuVisible = false

	m.setValidationStatus()
	m.addDebugLog(fmt.Sprintf("Config validation: %d errors, %d warnings",
		len(m.validationResults.Errors), len(m.validationResults.Warnings)))

	return m, nil
}

// handleTemplateToggle toggles the template browser.
func (m *model) handleTemplateToggle() (tea.Model, tea.Cmd) {
	if m.showTemplates {
		m.showTemplates = false
		m.showTemplatePreview = false
		m.templatePreviewContent = ""

		return m, nil
	}

	m.templateList = templates.List()
	m.selectedTemplate = 0
	m.showTemplates = true
	m.showTemplatePreview = false
	m.showHelp = false
	m.showLogs = false
	m.showStats = false
	m.showNeighbors = false
	m.showHexDump = false
	m.showValidation = false
	m.showConfigDiff = false
	m.showTopology = false
	m.showSearch = false
	m.showExport = false
	m.menuVisible = false
	m.statusMessage = "Template Browser - use arrow keys to navigate, Enter to preview"
	m.statusIsError = false

	return m, nil
}

// handleConfigDiffToggle toggles the config diff viewer.
func (m *model) handleConfigDiffToggle() (tea.Model, tea.Cmd) {
	if m.showConfigDiff {
		m.showConfigDiff = false

		return m, nil
	}

	m.generateConfigDiff()
	m.showConfigDiff = true
	m.configDiffScrollY = 0
	m.closeAllOverlays()
	m.statusMessage = "Config Diff Viewer - showing changes since last reload"
	m.statusIsError = false

	return m, nil
}

// handleConfigEdit opens the config file in an external editor.
func (m *model) handleConfigEdit() (tea.Model, tea.Cmd) {
	if m.configFilePath == "" {
		m.statusMessage = errorStyle.Render("No config file path available")
		m.statusIsError = true

		return m, nil
	}

	return m, m.openEditor()
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

// handleAlertConfigToggle toggles the alert configuration panel.
func (m *model) handleAlertConfigToggle() (tea.Model, tea.Cmd) {
	if m.showAlertConfig {
		m.showAlertConfig = false
		m.statusMessage = successStyle.Render("Alert configuration saved")
		m.statusIsError = false

		return m, nil
	}

	if m.alertThresholds == nil {
		m.alertThresholds = map[string]int{
			"CPU":        alertDefaultCPU,
			"Memory":     alertDefaultMemory,
			"Disk":       alertDefaultDisk,
			"PacketLoss": alertDefaultPacketLoss,
			"Latency":    alertDefaultLatency,
		}
	}

	m.showAlertConfig = true
	m.selectedAlertType = 0
	m.closeAllOverlays()
	m.statusMessage = "Alert Config - [Up/Down] navigate, [Left/Right] adjust, [Enter] toggle, [Space] all"
	m.statusIsError = false

	return m, nil
}

// handleTopologyToggle toggles the topology view.
func (m *model) handleTopologyToggle() (tea.Model, tea.Cmd) {
	if m.showTopology {
		m.showTopology = false

		return m, nil
	}

	m.generateTopologyView()
	m.showTopology = true
	m.topologyScrollY = 0
	m.closeAllOverlays()
	m.statusMessage = "Topology View - ASCII network diagram"
	m.statusIsError = false

	return m, nil
}

// handlePcapReplayToggle toggles the PCAP replay control panel.
func (m *model) handlePcapReplayToggle() (tea.Model, tea.Cmd) {
	if m.showPcapReplay {
		m.showPcapReplay = false
		m.pcapPlaying = false

		return m, nil
	}

	m.pcapPackets = make([]CapturedPacket, len(m.packetBuffer))
	copy(m.pcapPackets, m.packetBuffer)
	m.pcapPlaybackIndex = 0
	m.pcapPlaying = false

	if m.pcapPlaybackSpeed == 0 {
		m.pcapPlaybackSpeed = 1.0
	}

	m.showPcapReplay = true
	m.closeAllOverlays()
	m.statusMessage = "PCAP Replay - [Space] play/pause, [left/right] step, [+/-] speed"
	m.statusIsError = false

	return m, nil
}

// handleHistoryToggle toggles the history viewer.
func (m *model) handleHistoryToggle() (tea.Model, tea.Cmd) {
	if m.showHistory {
		m.showHistory = false

		return m, nil
	}

	m.showHistory = true
	m.selectedHistoryIdx = 0
	m.historyScrollY = 0
	m.closeAllOverlays()
	m.statusMessage = "Run History - [up/dn] navigate, [Enter] details"
	m.statusIsError = false

	return m, nil
}

// handleSnmpWalkToggle toggles the SNMP walk browser.
func (m *model) handleSnmpWalkToggle() (tea.Model, tea.Cmd) {
	if m.showSnmpWalk {
		m.showSnmpWalk = false

		return m, nil
	}

	m.snmpOidTree = []SnmpOidEntry{
		{OID: ".1.3.6.1.2.1.1.1.0", Name: "sysDescr", Value: "Network Simulator", Type: "STRING"},
		{OID: ".1.3.6.1.2.1.1.2.0", Name: "sysObjectID", Value: ".1.3.6.1.4.1.99999", Type: "OID"},
		{
			OID:   ".1.3.6.1.2.1.1.3.0",
			Name:  "sysUpTime",
			Value: strconv.Itoa(int(m.uptime.Seconds() * uptimeTicksMultiplier)),
			Type:  "TIMETICKS",
		},
		{OID: ".1.3.6.1.2.1.1.4.0", Name: "sysContact", Value: "admin@niac.local", Type: "STRING"},
		{OID: ".1.3.6.1.2.1.1.5.0", Name: "sysName", Value: m.interfaceName, Type: "STRING"},
		{OID: ".1.3.6.1.2.1.1.6.0", Name: "sysLocation", Value: "Network Lab", Type: "STRING"},
	}
	m.selectedSnmpOid = 0
	m.snmpScrollY = 0
	m.showSnmpWalk = true
	m.closeAllOverlays()
	m.statusMessage = "SNMP Walk Browser - [up/dn] navigate, [Enter] expand"
	m.statusIsError = false

	return m, nil
}

// handleDeviceConfigToggle toggles the device configuration panel.
func (m *model) handleDeviceConfigToggle() (tea.Model, tea.Cmd) {
	if m.showDeviceConfig {
		m.showDeviceConfig = false

		return m, nil
	}

	m.showDeviceConfig = true
	m.deviceConfigTab = 0
	m.deviceConfigScrollY = 0
	m.closeAllOverlays()
	m.statusMessage = "Device Config - [Tab] switch tab, [up/dn] scroll"
	m.statusIsError = false

	return m, nil
}

// handleSearchToggle toggles the search mode.
func (m *model) handleSearchToggle() (tea.Model, tea.Cmd) {
	if m.searchMode {
		m.searchMode = false
		m.showSearch = false
		m.searchQuery = ""
		m.searchResults = nil

		return m, nil
	}

	m.searchMode = true
	m.showSearch = true
	m.searchQuery = ""
	m.searchResults = nil
	m.selectedResult = 0
	m.searchCategory = searchCategoryAll
	m.closeAllOverlays()
	m.statusMessage = "Search Mode - type to filter devices/logs, [Esc] to exit"
	m.statusIsError = false

	return m, nil
}

// handleMenuToggle toggles the interactive menu.
func (m *model) handleMenuToggle() (tea.Model, tea.Cmd) {
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
		m.statusMessage = errorStyle.Render("No devices configured")
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
		fmt.Sprintf("Selected device: %s (%s)", device.Name, deviceIP),
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
		fmt.Sprintf("Debug level: %d (%s)", m.debugLevel, debugLevelName),
	)
	m.statusIsError = false
	m.addDebugLog(fmt.Sprintf("Debug level changed to %d (%s)", m.debugLevel, debugLevelName))

	return m, nil
}

// handleReload initiates a configuration reload.
func (m *model) handleReload() (tea.Model, tea.Cmd) {
	if m.reloadFunc == nil {
		m.statusMessage = errorStyle.Render("Reload not available in this mode")
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

// handleTemplateAction handles 'c' key in template browser.
func (m *model) handleTemplateAction() (tea.Model, tea.Cmd) {
	if !m.showTemplates || m.showTemplatePreview {
		return m, nil
	}

	if m.selectedTemplate < 0 || m.selectedTemplate >= len(m.templateList) {
		return m, nil
	}

	templateName := m.templateList[m.selectedTemplate].Name
	m.statusMessage = successStyle.Render(
		fmt.Sprintf(
			"Template: %s - Use: niac template use %s <output.yaml>",
			templateName,
			templateName,
		),
	)
	m.statusIsError = false
	m.addDebugLog("Template path shown: " + templateName)

	return m, nil
}

// handleClearErrors clears all error injections.
func (m *model) handleClearErrors() (tea.Model, tea.Cmd) {
	m.stateManager.ClearAll()
	m.statusMessage = successStyle.Render("All error injections cleared")
	m.statusIsError = false
	m.errorsActive = 0
	m.addDebugLog("All error injections cleared")

	return m, nil
}

// handleQuickErrorInjection handles number keys 1-7 for quick error injection.
func (m *model) handleQuickErrorInjection(key string) (tea.Model, tea.Cmd) {
	if m.menuVisible || m.showHelp || m.showLogs || m.showStats {
		return m, nil
	}

	errorTypeMap := map[string]struct {
		errorType apperr.ErrorType
		prompt    string
	}{
		"1": {apperr.ErrorTypeFCS, "Enter FCS error count (0-100): "},
		"2": {apperr.ErrorTypeDiscards, "Enter packet discard rate (0-100): "},
		"3": {apperr.ErrorTypeInterface, "Enter interface error count (0-100): "},
		"4": {apperr.ErrorTypeUtilization, "Enter utilization percentage (0-100): "},
		"5": {apperr.ErrorTypeCPU, "Enter CPU percentage (0-100): "},
		"6": {apperr.ErrorTypeMemory, "Enter memory percentage (0-100): "},
		"7": {apperr.ErrorTypeDisk, "Enter disk percentage (0-100): "},
	}

	if errInfo, ok := errorTypeMap[key]; ok {
		m.promptForValue(errInfo.errorType, errInfo.prompt)
	}

	return m, nil
}
