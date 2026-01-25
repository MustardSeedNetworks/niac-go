package interactive

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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

// renderActiveErrors renders the active error injections section.
func (m *model) renderActiveErrors(s *strings.Builder) {
	activeStates := m.stateManager.GetAllStates()
	if len(activeStates) == 0 {
		return
	}

	s.WriteString(errorStyle.Render("Active Error Injections:"))
	s.WriteString("\n")

	for _, state := range activeStates {
		fmt.Fprintf(s, "  - %s on %s:%s (%d%%)\n",
			state.ErrorType,
			state.DeviceIP,
			state.Interface,
			state.Value,
		)
	}
	s.WriteString("\n")
}

// renderDeviceList renders the list of simulated devices.
func (m *model) renderDeviceList(s *strings.Builder) {
	s.WriteString(deviceStyle.Render("Simulated Devices:"))
	s.WriteString("\n")

	for i, device := range m.cfg.Devices {
		ip := noIPPlaceholder
		if len(device.IPAddresses) > 0 {
			ip = device.IPAddresses[0].String()
		}

		prefix := "  "
		suffix := ""

		if i == m.selectedDeviceIdx {
			prefix = selectedStyle.Render("-> ")
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
