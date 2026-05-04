package interactive

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// openEditor opens the config file in the user's preferred editor.
func (m *model) openEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}

	if editor == "" {
		editor = "vi" // Default fallback
	}

	c := exec.CommandContext( //nolint:gosec // G702: editor is from trusted EDITOR/VISUAL env var, not user-controlled input
		context.Background(),
		editor,
		m.configFilePath,
	)

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// appendDeviceCountDiff adds device count change to diff content.
func (m *model) appendDeviceCountDiff(prevCount, currCount int) {
	if prevCount == currCount {
		m.configDiffContent = append(
			m.configDiffContent,
			fmt.Sprintf("  Device count: %d (unchanged)", currCount),
		)
		return
	}

	if currCount > prevCount {
		m.configDiffContent = append(
			m.configDiffContent,
			diffAddedStyle.Render(
				fmt.Sprintf(
					"+ Device count: %d -> %d (+%d)",
					prevCount,
					currCount,
					currCount-prevCount,
				),
			),
		)
	} else {
		m.configDiffContent = append(
			m.configDiffContent,
			diffRemovedStyle.Render(
				fmt.Sprintf(
					"- Device count: %d -> %d (-%d)",
					prevCount,
					currCount,
					prevCount-currCount,
				),
			),
		)
	}
}

// appendAddedDevices adds newly added devices to diff content.
func (m *model) appendAddedDevices(currDevices, prevDevices map[string]*config.Device) {
	for name, device := range currDevices {
		if _, exists := prevDevices[name]; exists {
			continue
		}

		ip := noIPPlaceholder
		if len(device.IPAddresses) > 0 {
			ip = device.IPAddresses[0].String()
		}

		m.configDiffContent = append(
			m.configDiffContent,
			diffAddedStyle.Render(fmt.Sprintf("+ %s (%s, %s)", name, device.Type, ip)),
		)
	}
}

// appendRemovedDevices adds removed devices to diff content.
func (m *model) appendRemovedDevices(currDevices, prevDevices map[string]*config.Device) {
	for name, device := range prevDevices {
		if _, exists := currDevices[name]; exists {
			continue
		}

		ip := noIPPlaceholder
		if len(device.IPAddresses) > 0 {
			ip = device.IPAddresses[0].String()
		}

		m.configDiffContent = append(
			m.configDiffContent,
			diffRemovedStyle.Render(fmt.Sprintf("- %s (%s, %s)", name, device.Type, ip)),
		)
	}
}

// appendModifiedDevices adds modified devices to diff content.
func (m *model) appendModifiedDevices(currDevices, prevDevices map[string]*config.Device) {
	for name, currDev := range currDevices {
		prevDev, exists := prevDevices[name]
		if !exists {
			continue
		}

		changes := m.compareDevices(prevDev, currDev)
		for _, change := range changes {
			m.configDiffContent = append(m.configDiffContent, fmt.Sprintf("  %s: %s", name, change))
		}
	}
}

// buildDeviceMap creates a map of device names to device pointers.
func buildDeviceMap(devices []config.Device) map[string]*config.Device {
	result := make(map[string]*config.Device)
	for i := range devices {
		result[devices[i].Name] = &devices[i]
	}
	return result
}

func (m *model) generateConfigDiff() {
	m.configDiffContent = nil

	if m.previousConfig == nil {
		m.configDiffContent = append(m.configDiffContent, "No previous configuration to compare.")
		m.configDiffContent = append(
			m.configDiffContent,
			"Reload config with [r] to create a comparison baseline.",
		)
		return
	}

	prevCount := len(m.previousConfig.Devices)
	currCount := len(m.cfg.Devices)

	m.configDiffContent = append(
		m.configDiffContent,
		diffHeaderStyle.Render("=== Configuration Diff ==="),
	)
	m.configDiffContent = append(m.configDiffContent, "")

	m.appendDeviceCountDiff(prevCount, currCount)
	m.configDiffContent = append(m.configDiffContent, "")

	prevDevices := buildDeviceMap(m.previousConfig.Devices)
	currDevices := buildDeviceMap(m.cfg.Devices)

	m.configDiffContent = append(m.configDiffContent, diffHeaderStyle.Render("--- Devices ---"))
	m.appendAddedDevices(currDevices, prevDevices)
	m.appendRemovedDevices(currDevices, prevDevices)
	m.appendModifiedDevices(currDevices, prevDevices)

	if len(m.configDiffContent) <= minDiffLinesForNoOp {
		m.configDiffContent = append(m.configDiffContent, "  No device changes detected")
	}
}

// compareDevices compares two devices and returns a list of changes.
func (m *model) compareDevices(prev, curr *config.Device) []string {
	var changes []string

	// Compare type
	if prev.Type != curr.Type {
		changes = append(changes, fmt.Sprintf("type: %s -> %s", prev.Type, curr.Type))
	}

	// Compare MAC
	if prev.MACAddress.String() != curr.MACAddress.String() {
		changes = append(changes, fmt.Sprintf("MAC: %s -> %s", prev.MACAddress, curr.MACAddress))
	}

	// Compare IPs
	prevIPs := make(map[string]bool)
	for _, ip := range prev.IPAddresses {
		prevIPs[ip.String()] = true
	}

	currIPs := make(map[string]bool)
	for _, ip := range curr.IPAddresses {
		currIPs[ip.String()] = true
	}

	for ip := range currIPs {
		if !prevIPs[ip] {
			changes = append(changes, diffAddedStyle.Render("+ IP: "+ip))
		}
	}

	for ip := range prevIPs {
		if !currIPs[ip] {
			changes = append(changes, diffRemovedStyle.Render("- IP: "+ip))
		}
	}

	return changes
}

// renderConfigDiff renders the config diff view.
func (m *model) renderConfigDiff() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString("║                    Configuration Diff                            ║\n")
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.configDiffContent) == 0 {
		panel.WriteString(padPanelLine("No diff content available"))
		panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

		return panel.String()
	}

	// Calculate visible lines
	maxLines := 15
	totalLines := len(m.configDiffContent)

	startLine := m.configDiffScrollY
	if startLine >= totalLines {
		startLine = max(totalLines-1, 0)
	}

	endLine := min(startLine+maxLines, totalLines)

	for i := startLine; i < endLine; i++ {
		panel.WriteString(padPanelLine(m.configDiffContent[i]))
	}

	if totalLines > maxLines {
		panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
		panel.WriteString(
			padPanelLine(
				fmt.Sprintf(
					"Lines %d-%d of %d (use arrows/PgUp/PgDn to scroll)",
					startLine+1,
					endLine,
					totalLines,
				),
			),
		)
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[C] or [ESC] Close  [r] Reload config to update baseline"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
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
