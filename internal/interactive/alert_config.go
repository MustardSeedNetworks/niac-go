package interactive

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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

// renderAlertConfig renders the alert configuration panel.
func (m *model) renderAlertConfig() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString(padPanelLine(diffHeaderStyle.Render("Alert Configuration")))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	alertTypes := []string{"CPU", "Memory", "Disk", "PacketLoss", "Latency"}
	alertDescriptions := map[string]string{
		"CPU":        "High CPU usage alert",
		"Memory":     "High memory usage alert",
		"Disk":       "High disk usage alert",
		"PacketLoss": "Packet loss threshold",
		"Latency":    "High latency alert (ms)",
	}

	// Initialize thresholds if nil
	if m.alertThresholds == nil {
		// These would normally be initialized elsewhere
		panel.WriteString(padPanelLine("No alert thresholds configured"))
	} else {
		for i, alertType := range alertTypes {
			threshold := m.alertThresholds[alertType]
			if threshold == 0 {
				threshold = alertDefaultThresholdInDisplay
			}

			// Create visual threshold bar
			barWidth := 20
			filledWidth := (threshold * barWidth) / alertThresholdMax
			bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)

			// Highlight selected row
			prefix := "  "
			if i == m.selectedAlertType {
				prefix = selectedStyle.Render("→ ")
			}

			line := fmt.Sprintf("%s%-12s [%s] %3d%% - %s",
				prefix, alertType, bar, threshold, alertDescriptions[alertType])
			panel.WriteString(padPanelLine(line))
		}
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	enabledStatus := "DISABLED"
	if m.alertsEnabled {
		enabledStatus = successStyle.Render("ENABLED")
	}

	panel.WriteString(padPanelLine(fmt.Sprintf("Alerts: %s  [Space] Toggle All", enabledStatus)))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(
		padPanelLine("[↑↓] Navigate  [←→] Adjust  [Enter] Toggle  [ESC] Save & Close"),
	)
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
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
