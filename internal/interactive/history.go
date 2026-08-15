package interactive

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// renderHistory renders the run history viewer panel.
func (m *model) renderHistory() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString(padPanelLine(diffHeaderStyle.Render("Run History")))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.historyEntries) == 0 {
		panel.WriteString(padPanelLine("No history entries recorded yet"))
		panel.WriteString(padPanelLine("History is recorded when simulations start/stop"))
	} else {
		// Header row
		panel.WriteString(
			padPanelLine(
				fmt.Sprintf(
					"%-20s %-10s %-8s %-12s %-12s",
					"Date/Time",
					"Duration",
					"Devices",
					"Packets RX",
					"Packets TX",
				),
			),
		)
		panel.WriteString(padPanelLine(strings.Repeat("-", historyRowWidth)))

		// History entries
		startIdx := m.historyScrollY
		endIdx := min(startIdx+historyVisibleRows, len(m.historyEntries))

		for i := startIdx; i < endIdx; i++ {
			entry := m.historyEntries[i]
			duration := entry.EndTime.Sub(entry.StartTime)

			prefix := "  "
			if i == m.selectedHistoryIdx {
				prefix = selectedStyle.Render("→ ")
			}

			line := fmt.Sprintf("%s%-20s %-10s %-8d %-12d %-12d",
				prefix,
				entry.StartTime.Format("2006-01-02 15:04"),
				formatDuration(duration),
				entry.DeviceCount,
				entry.PacketsReceived,
				entry.PacketsSent,
			)
			panel.WriteString(padPanelLine(line))
		}
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine(fmt.Sprintf("Total runs: %d", len(m.historyEntries))))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(
		padPanelLine("[↑↓] Navigate  [Enter] Details  [PgUp/PgDn] Scroll  [ESC] Close"),
	)
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// handleHistoryInput handles keyboard input in history viewer.
func (m *model) handleHistoryInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	m.statusMessage = "Run History - [↑↓] navigate, [Enter] details"
	m.statusIsError = false

	return m, nil
}
