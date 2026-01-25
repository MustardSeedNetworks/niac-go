package interactive

import (
	"fmt"
	"strings"
)

// renderHelp renders the help screen overlay.
func (m *model) renderHelp() string {
	return `+=================================================================+
|                         NIAC-Go Help                             |
+=================================================================+
| Keyboard Shortcuts:                                              |
|                                                                  |
|  [i]     Toggle interactive error injection menu                |
|  [D]     Cycle through devices (Shift+D)                        |
|  [d]     Cycle debug level (QUIET->NORMAL->VERBOSE->DEBUG)       |
|  [h][?]  Toggle this help screen                                |
|  [l]     Toggle debug log viewer                                |
|  [s]     Toggle statistics viewer                               |
|  [N]/[n] Toggle neighbor discovery table                        |
|  [x]     Toggle packet hex dump viewer                          |
|  [t]     Toggle template browser                                |
|  [v]     Validate configuration                                 |
|  [r]     Reload configuration from disk                         |
|                                                                  |
| New Features:                                                    |
|  [C]     Config diff viewer (compare before/after reload)       |
|  [e]     Quick edit config in $EDITOR                           |
|  [E]     Export statistics to JSON/CSV file                     |
|  [T]     Network topology view (ASCII diagram)                  |
|  [/]     Search mode (filter devices/logs by pattern)           |
|                                                                  |
| Navigation:                                                      |
|  [n]/[p] Navigate packets (next/previous) in hex viewer         |
|  [up][dn] Scroll / Navigate menu items                          |
|  [PgUp]  Page up in scrollable views                            |
|  [PgDn]  Page down in scrollable views                          |
|  [c]     Clear all error injections                             |
|  [1-7]   Quick error injection (FCS/Disc/If/Util/CPU/Mem/Disk)  |
|  [q]     Quit application                                       |
|                                                                  |
| Search Mode ([/]):                                               |
|  - Type to search devices, logs, and errors                     |
|  - [Tab] to cycle category (all/devices/logs)                   |
|  - [Enter] to select, [Esc] to cancel                           |
|                                                                  |
| Export Mode ([E]):                                               |
|  - [j] for JSON format, [c] for CSV format                      |
|  - [Enter] to save file, [Esc] to cancel                        |
|                                                                  |
| Debug Levels:                                                    |
|  0 - QUIET    Only critical errors                              |
|  1 - NORMAL   Status messages (default)                         |
|  2 - VERBOSE  Protocol details                                   |
|  3 - DEBUG    Full packet details                               |
|                                                                  |
| Error Injection Types:                                           |
|  - FCS Errors        Frame Check Sequence errors (0-100)        |
|  - Packet Discards   Dropped packets rate (0-100)               |
|  - Interface Errors  General interface errors (0-100)           |
|  - High Utilization  Link utilization percentage (0-100)        |
|  - High CPU          CPU usage percentage (0-100)               |
|  - High Memory       Memory usage percentage (0-100)            |
|  - High Disk         Disk usage percentage (0-100)              |
+=================================================================+`
}

// renderSearchEmptyState renders the message when no search results exist.
func (m *model) renderSearchEmptyState(panel *strings.Builder) {
	if m.searchQuery == "" {
		panel.WriteString(padPanelLine("Type to search devices, logs, and errors"))
	} else {
		panel.WriteString(padPanelLine("No results found"))
	}
}

// renderSearchResultLine renders a single search result line.
func (m *model) renderSearchResultLine(panel *strings.Builder, idx int, result searchResult) {
	prefix := "  "
	if idx == m.selectedResult {
		prefix = selectedStyle.Render("->")
	}

	categoryTag := "[" + result.Category + "]"
	line := prefix + " " + categoryTag + " " + result.Title
	if len(line) > maxSearchResultWidth {
		line = line[:maxSearchResultWidth-minEllipsisWidth] + "..."
	}
	panel.WriteString(padPanelLine(line))

	// Show detail only for selected item
	if idx != m.selectedResult || result.Detail == "" {
		return
	}

	detailLine := "     " + result.Detail
	if len(detailLine) > maxSearchResultWidth {
		detailLine = detailLine[:maxSearchResultWidth-minEllipsisWidth] + "..."
	}
	panel.WriteString(padPanelLine(statsStyle.Render(detailLine)))
}

// renderSearchResults renders the search results list.
func (m *model) renderSearchResults(panel *strings.Builder) {
	panel.WriteString(padPanelLine(fmt.Sprintf("Results: %d found", len(m.searchResults))))
	panel.WriteString("+=================================================================+\n")

	maxResults := 10
	start := 0
	if m.selectedResult >= maxResults {
		start = m.selectedResult - maxResults + 1
	}
	end := min(start+maxResults, len(m.searchResults))

	for i := start; i < end; i++ {
		m.renderSearchResultLine(panel, i, m.searchResults[i])
	}
}

// renderSearch renders the search panel.
func (m *model) renderSearch() string {
	var panel strings.Builder

	panel.WriteString("+=================================================================+\n")
	panel.WriteString("|                         Search Mode                              |\n")
	panel.WriteString("+=================================================================+\n")

	queryDisplay := m.searchQuery
	if queryDisplay == "" {
		queryDisplay = "_"
	}

	panel.WriteString(padPanelLine("Query: " + queryDisplay))
	panel.WriteString(padPanelLine("Category: [" + m.searchCategory + "] (Tab to cycle)"))
	panel.WriteString("+=================================================================+\n")

	if len(m.searchResults) == 0 {
		m.renderSearchEmptyState(&panel)
	} else {
		m.renderSearchResults(&panel)
	}

	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine("[Enter] Select  [Tab] Category  [ESC] Close"))
	panel.WriteString("+=================================================================+")

	return panel.String()
}

// renderAlertConfig renders the alert configuration panel.
func (m *model) renderAlertConfig() string {
	var panel strings.Builder

	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine(diffHeaderStyle.Render("Alert Configuration")))
	panel.WriteString("+=================================================================+\n")

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
			bar := strings.Repeat("=", filledWidth) + strings.Repeat("-", barWidth-filledWidth)

			// Highlight selected row
			prefix := "  "
			if i == m.selectedAlertType {
				prefix = selectedStyle.Render("-> ")
			}

			line := fmt.Sprintf("%s%-12s [%s] %3d%% - %s",
				prefix, alertType, bar, threshold, alertDescriptions[alertType])
			panel.WriteString(padPanelLine(line))
		}
	}

	panel.WriteString("+=================================================================+\n")

	enabledStatus := "DISABLED"
	if m.alertsEnabled {
		enabledStatus = successStyle.Render("ENABLED")
	}

	panel.WriteString(padPanelLine(fmt.Sprintf("Alerts: %s  [Space] Toggle All", enabledStatus)))
	panel.WriteString("+=================================================================+\n")
	panel.WriteString(
		padPanelLine("[up/dn] Navigate  [left/right] Adjust  [Enter] Toggle  [ESC] Save & Close"),
	)
	panel.WriteString("+=================================================================+")

	return panel.String()
}

// renderTemplateBrowser renders the template browser panel.
func (m *model) renderTemplateBrowser() string {
	var panel strings.Builder

	if m.showTemplatePreview {
		// Show template content preview
		panel.WriteString("+=================================================================+\n")
		panel.WriteString("|                    Template Preview                              |\n")
		panel.WriteString("+=================================================================+\n")

		// Show template name
		if m.selectedTemplate >= 0 && m.selectedTemplate < len(m.templateList) {
			tmpl := m.templateList[m.selectedTemplate]
			panel.WriteString(padPanelLine("Name: " + tmpl.Name))
			panel.WriteString(padPanelLine("Description: " + tmpl.Description))
			panel.WriteString(
				"+=================================================================+\n",
			)
		}

		// Show first 15 lines of content
		lines := strings.Split(m.templatePreviewContent, "\n")

		maxPreviewLines := 15
		if len(lines) > maxPreviewLines {
			lines = lines[:maxPreviewLines]
		}

		for _, line := range lines {
			panel.WriteString(padPanelLine(line))
		}

		if len(strings.Split(m.templatePreviewContent, "\n")) > maxPreviewLines {
			panel.WriteString(padPanelLine("... (content truncated)"))
		}

		panel.WriteString("+=================================================================+\n")
		panel.WriteString(padPanelLine("[ESC] Back to list  [t] Close browser"))
		panel.WriteString("+=================================================================+")

		return panel.String()
	}

	// Template list view
	panel.WriteString("+=================================================================+\n")
	panel.WriteString("|                    Template Browser                              |\n")
	panel.WriteString("+=================================================================+\n")

	if len(m.templateList) == 0 {
		panel.WriteString(padPanelLine("No templates available"))
		panel.WriteString("+=================================================================+")

		return panel.String()
	}

	for i, tmpl := range m.templateList {
		prefix := "  "
		if i == m.selectedTemplate {
			prefix = selectedStyle.Render("->")
		}

		// Format: name (description)
		line := fmt.Sprintf("%s %-18s (%s)", prefix, tmpl.Name, tmpl.Description)
		if len(line) > panelContentWidth {
			line = line[:panelContentWidth-minEllipsisWidth] + "..."
		}

		panel.WriteString(padPanelLine(line))
	}

	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine("[up/down] Navigate  [Enter] Preview  [c] Copy  [ESC] Close"))
	panel.WriteString("+=================================================================+")

	return panel.String()
}
