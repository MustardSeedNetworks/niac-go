package interactive

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// renderValidationErrors renders the validation errors section.
func (m *model) renderValidationErrors(panel *strings.Builder) {
	errorCount := len(m.validationResults.Errors)
	if errorCount == 0 {
		return
	}

	panel.WriteString(padPanelLine(validationErrorStyle.Render("ERRORS:")))

	for i, err := range m.validationResults.Errors {
		if i >= maxValidationDisplay {
			remaining := errorCount - i
			panel.WriteString(
				padPanelLine(
					validationErrorStyle.Render(
						fmt.Sprintf("  ... and %d more error(s)", remaining),
					),
				),
			)
			break
		}

		errLine := fmt.Sprintf("  [%s] %s", err.Field, err.Message)
		if len(errLine) > validationTruncate {
			errLine = errLine[:validationTruncate-minEllipsisWidth] + "..."
		}

		panel.WriteString(padPanelLine(validationErrorStyle.Render(errLine)))
	}

	panel.WriteString(padPanelLine(""))
}

// renderValidationWarnings renders the validation warnings section.
func (m *model) renderValidationWarnings(panel *strings.Builder) {
	warningCount := len(m.validationResults.Warnings)
	if warningCount == 0 {
		return
	}

	panel.WriteString(padPanelLine(validationWarningStyle.Render("WARNINGS:")))

	for i, warn := range m.validationResults.Warnings {
		if i >= maxValidationDisplay {
			remaining := warningCount - i
			panel.WriteString(
				padPanelLine(
					validationWarningStyle.Render(
						fmt.Sprintf("  ... and %d more warning(s)", remaining),
					),
				),
			)
			break
		}

		warnLine := fmt.Sprintf("  [%s] %s", warn.Field, warn.Message)
		if len(warnLine) > validationTruncate {
			warnLine = warnLine[:validationTruncate-minEllipsisWidth] + "..."
		}

		panel.WriteString(padPanelLine(validationWarningStyle.Render(warnLine)))
	}
}

// renderValidationSuccess renders the success message when no validation issues exist.
func (m *model) renderValidationSuccess(panel *strings.Builder) {
	successLine := validationSuccessStyle.Render("Configuration is valid - no errors or warnings")
	panel.WriteString(padPanelLine(successLine))
	panel.WriteString(padPanelLine(""))
	panel.WriteString(padPanelLine(fmt.Sprintf("Devices configured: %d", len(m.cfg.Devices))))
}

// renderValidationIssues renders errors and warnings when present.
func (m *model) renderValidationIssues(panel *strings.Builder) {
	errorCount := len(m.validationResults.Errors)
	warningCount := len(m.validationResults.Warnings)

	summaryLine := fmt.Sprintf("Found: %s, %s",
		validationErrorStyle.Render(fmt.Sprintf("%d error(s)", errorCount)),
		validationWarningStyle.Render(fmt.Sprintf("%d warning(s)", warningCount)))
	panel.WriteString(padPanelLine(summaryLine))
	panel.WriteString("+=================================================================+\n")

	m.renderValidationErrors(panel)
	m.renderValidationWarnings(panel)
}

// renderValidation renders the configuration validation panel.
func (m *model) renderValidation() string {
	var panel strings.Builder

	panel.WriteString("+=================================================================+\n")
	panel.WriteString("|                   Configuration Validation                       |\n")
	panel.WriteString("+=================================================================+\n")

	if m.validationResults == nil {
		panel.WriteString(padPanelLine("No validation results available"))
		panel.WriteString("+=================================================================+")
		return panel.String()
	}

	errorCount := len(m.validationResults.Errors)
	warningCount := len(m.validationResults.Warnings)

	if errorCount == 0 && warningCount == 0 {
		m.renderValidationSuccess(&panel)
	} else {
		m.renderValidationIssues(&panel)
	}

	panel.WriteString("+=================================================================+\n")
	panel.WriteString(
		padPanelLine(validationInfoStyle.Render("Press [v] or [Esc] to close this view")),
	)
	panel.WriteString("+=================================================================+")

	return panel.String()
}

// renderConfigDiff renders the config diff view.
func (m *model) renderConfigDiff() string {
	var panel strings.Builder

	panel.WriteString("+=================================================================+\n")
	panel.WriteString("|                    Configuration Diff                            |\n")
	panel.WriteString("+=================================================================+\n")

	if len(m.configDiffContent) == 0 {
		panel.WriteString(padPanelLine("No diff content available"))
		panel.WriteString("+=================================================================+")

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
		panel.WriteString("+=================================================================+\n")
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

	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine("[C] or [ESC] Close  [r] Reload config to update baseline"))
	panel.WriteString("+=================================================================+")

	return panel.String()
}

// renderExport renders the export panel.
func (m *model) renderExport() string {
	var panel strings.Builder

	panel.WriteString("+=================================================================+\n")
	panel.WriteString("|                      Export Statistics                           |\n")
	panel.WriteString("+=================================================================+\n")

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
	panel.WriteString("+=================================================================+\n")
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

	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine("[Enter] Export  [j] JSON  [c] CSV  [ESC] Cancel"))
	panel.WriteString("+=================================================================+")

	return panel.String()
}

// generateConfigDiff generates a diff between current and previous config.
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
