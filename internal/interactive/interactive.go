// Package interactive provides a terminal user interface for network simulation control and monitoring
package interactive

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/krisarmstrong/niac-go/internal/apperr"
	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/protocols"
)

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		tea.EnterAltScreen,
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func reloadCmd(fn func() (*config.Config, error)) tea.Cmd {
	if fn == nil {
		return nil
	}

	return func() tea.Msg {
		cfg, err := fn()

		return reloadMsg{cfg: cfg, err: err}
	}
}

// Update handles messages and updates the TUI model state.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case reloadMsg:
		return m.handleReloadMsg(msg)
	case editorFinishedMsg:
		return m.handleEditorFinishedMsg(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tickMsg:
		m.uptime = time.Since(m.startTime)
		m.errorsActive = len(m.stateManager.GetAllStates())
		m.refreshStats()

		return m, tickCmd()
	}

	return m, nil
}

func (m *model) refreshStats() {
	if m.stack == nil {
		return
	}

	stats := m.stack.GetStats()
	m.stackStats = stackStatsSnapshot{
		PacketsReceived: stats.PacketsReceived,
		PacketsSent:     stats.PacketsSent,
		ARPRequests:     stats.ARPRequests,
		ARPReplies:      stats.ARPReplies,
		ICMPRequests:    stats.ICMPRequests,
		ICMPReplies:     stats.ICMPReplies,
		DNSQueries:      stats.DNSQueries,
		DHCPRequests:    stats.DHCPRequests,
	}
	m.neighbors = m.stack.GetNeighbors()
}

// View renders the TUI display to the terminal.
func (m *model) View() string {
	var s strings.Builder

	// Title
	s.WriteString(
		titleStyle.Render(fmt.Sprintf(" NIAC-Go Interactive Mode - %s ", m.interfaceName)),
	)
	s.WriteString("\n\n")

	// Status bar with selected device
	stats := fmt.Sprintf(
		"Uptime: %s  |  Debug: %d (%s)  |  Selected Device: %s  |  Errors Active: %d  |  Injected: %d",
		formatDuration(m.uptime),
		m.debugLevel,
		getDebugLevelName(m.debugLevel),
		m.getSelectedDeviceName(),
		m.errorsActive,
		m.packetsInjected,
	)
	s.WriteString(statsStyle.Render(stats))
	s.WriteString("\n\n")

	// Devices
	m.renderDeviceList(&s)

	// Active errors
	m.renderActiveErrors(&s)

	// Status message
	m.renderStatusMessage(&s)

	// Value input prompt
	if m.valueInputMode {
		s.WriteString(m.renderValueInput())
		s.WriteString("\n")
	}

	// Menu
	if m.menuVisible && !m.valueInputMode {
		s.WriteString(m.renderMenu())
		s.WriteString("\n")
	}

	// Render all overlays
	m.renderOverlays(&s)

	// Controls
	m.renderControls(&s)

	return s.String()
}

// renderOverlays renders all overlay panels and returns the combined output.
func (m *model) renderOverlays(s *strings.Builder) {
	type overlayConfig struct {
		show   bool
		render func() string
	}

	overlays := []overlayConfig{
		{m.showHelp, m.renderHelp},
		{m.showLogs, m.renderLogs},
		{m.showStats, m.renderStatistics},
		{m.showNeighbors, m.renderNeighbors},
		{m.showHexDump, m.renderHexDump},
		{m.showTemplates, m.renderTemplateBrowser},
		{m.showValidation, m.renderValidation},
		{m.showConfigDiff, m.renderConfigDiff},
		{m.showSearch, m.renderSearch},
		{m.showExport, m.renderExport},
		{m.showTopology, m.renderTopology},
		{m.showAlertConfig, m.renderAlertConfig},
		{m.showPcapReplay, m.renderPcapReplay},
		{m.showHistory, m.renderHistory},
		{m.showSnmpWalk, m.renderSnmpWalk},
		{m.showDeviceConfig, m.renderDeviceConfig},
	}

	for _, overlay := range overlays {
		if overlay.show {
			s.WriteString(overlay.render())
			s.WriteString("\n")
		}
	}
}

// renderControls renders the control key hints at the bottom of the screen.
func (m *model) renderControls(s *strings.Builder) {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	quitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	// Controls - first row
	s.WriteString("Controls: ")
	s.WriteString(keyStyle.Render("[i]"))
	s.WriteString(" Menu  ")
	s.WriteString(keyStyle.Render("[D]"))
	s.WriteString(" Device  ")
	s.WriteString(keyStyle.Render("[d]"))
	s.WriteString(" Debug  ")
	s.WriteString(keyStyle.Render("[h]"))
	s.WriteString(" Help  ")
	s.WriteString(keyStyle.Render("[l]"))
	s.WriteString(" Logs  ")
	s.WriteString(keyStyle.Render("[s]"))
	s.WriteString(" Stats  ")
	s.WriteString(keyStyle.Render("[N]"))
	s.WriteString(" Neighbors  ")
	s.WriteString(keyStyle.Render("[x]"))
	s.WriteString(" Hex\n")

	// Controls - second row
	s.WriteString("          ")
	s.WriteString(keyStyle.Render("[t]"))
	s.WriteString(" Templates  ")
	s.WriteString(keyStyle.Render("[v]"))
	s.WriteString(" Validate  ")
	s.WriteString(keyStyle.Render("[C]"))
	s.WriteString(" Diff  ")
	s.WriteString(keyStyle.Render("[e]"))
	s.WriteString(" Edit  ")
	s.WriteString(keyStyle.Render("[E]"))
	s.WriteString(" Export  ")
	s.WriteString(keyStyle.Render("[a]"))
	s.WriteString(" Alerts\n")

	// Controls - third row
	s.WriteString("          ")
	s.WriteString(keyStyle.Render("[T]"))
	s.WriteString(" Topology  ")
	s.WriteString(keyStyle.Render("[/]"))
	s.WriteString(" Search  ")
	s.WriteString(keyStyle.Render("[c]"))
	s.WriteString(" Clear  ")
	s.WriteString(quitStyle.Render("[q]"))
	s.WriteString(" Quit")
}

// renderStatusMessage renders the status message if present.
func (m *model) renderStatusMessage(s *strings.Builder) {
	if m.statusMessage != "" {
		if m.statusIsError {
			s.WriteString(errorStyle.Render(m.statusMessage))
		} else {
			s.WriteString(m.statusMessage)
		}
		s.WriteString("\n\n")
	}
}

// renderDeviceList renders the list of simulated devices.
func (m *model) renderDeviceList(s *strings.Builder) {
	s.WriteString(deviceStyle.Render("📡 Simulated Devices:"))
	s.WriteString("\n")

	for i, device := range m.cfg.Devices {
		ip := noIPPlaceholder
		if len(device.IPAddresses) > 0 {
			ip = device.IPAddresses[0].String()
		}

		prefix := "  "
		suffix := ""

		if i == m.selectedDeviceIdx {
			prefix = selectedStyle.Render("→ ")
			suffix = selectedStyle.Render(" [SELECTED]")
		}

		fmt.Fprintf(s, "%s%d. %s (%s) - %s - %s%s\n",
			prefix,
			i+1,
			device.Name,
			device.Type,
			ip,
			device.MACAddress.String(),
			suffix,
		)
	}
	s.WriteString("\n")
}

// closeAllOverlays closes all overlay panels.
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
}

// Run starts the interactive mode with the specified configuration.
func Run(
	interfaceName string,
	cfg *config.Config,
	debugConfig *logging.DebugConfig,
	stack *protocols.Stack,
	startTime time.Time,
	reloadFunc func() (*config.Config, error),
) error {
	return RunWithConfigPath(interfaceName, cfg, debugConfig, stack, startTime, reloadFunc, "")
}

// RunWithConfigPath starts the interactive mode with a config file path for editing.
func RunWithConfigPath(
	interfaceName string,
	cfg *config.Config,
	debugConfig *logging.DebugConfig,
	stack *protocols.Stack,
	startTime time.Time,
	reloadFunc func() (*config.Config, error),
	configFilePath string,
) error {
	if debugConfig == nil {
		debugConfig = logging.NewDebugConfig(1)
	}

	debugLevel := debugConfig.GetGlobal()

	if startTime.IsZero() {
		startTime = time.Now()
	}

	// Initialize state manager
	stateManager := apperr.NewStateManager()

	// Create menu items
	menuItems := []string{
		"1. Inject FCS Errors (custom value)",
		"2. Inject Packet Discards (custom value)",
		"3. Inject Interface Errors (custom value)",
		"4. Inject High Utilization (custom value)",
		"5. Inject High CPU (custom value)",
		"6. Inject High Memory (custom value)",
		"7. Inject High Disk (custom value)",
		"8. Clear All Errors",
		"9. Exit Menu",
	}

	// Create model
	m := model{
		cfg:            cfg,
		stateManager:   stateManager,
		interfaceName:  interfaceName,
		debugLevel:     debugLevel,
		stack:          stack,
		reloadFunc:     reloadFunc,
		menuItems:      menuItems,
		startTime:      startTime,
		configFilePath: configFilePath,
		exportFormat:   formatJSON,
		searchCategory: searchCategoryAll,
		statusMessage:  "Press 'i' for menu, 'r' to reload config, 'h' for help",
		debugLogs:      make([]string, 0, maxDebugLogs),
	}

	if stack != nil {
		m.refreshStats()
	}

	// Add initial log entry
	m.addDebugLog("Started NIAC-Go interactive mode on " + interfaceName)
	m.addDebugLog(fmt.Sprintf("Debug level: %d (%s)", debugLevel, getDebugLevelName(debugLevel)))
	m.addDebugLog(fmt.Sprintf("Simulating %d device(s)", len(cfg.Devices)))

	// Start TUI
	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	return nil
}
