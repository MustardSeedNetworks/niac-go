package interactive

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// cancelSearch resets search state and dismisses the search overlay.
func (m *model) cancelSearch() {
	m.searchMode = false
	m.showSearch = false
	m.searchQuery = ""
	m.searchResults = nil
	m.statusMessage = "Search cancelled"
	m.statusIsError = false
}

// selectSearchResult handles selection of the current search result.
func (m *model) selectSearchResult() {
	if len(m.searchResults) == 0 || m.selectedResult < 0 ||
		m.selectedResult >= len(m.searchResults) {
		return
	}

	result := m.searchResults[m.selectedResult]
	m.searchMode = false
	m.showSearch = false

	switch result.Category {
	case "device":
		if result.Index >= 0 && result.Index < len(m.cfg.Devices) {
			m.selectedDeviceIdx = result.Index
			m.statusMessage = successStyle.Render("Selected device: " + result.Title)
		}
	case "log":
		m.showLogs = true
		m.statusMessage = successStyle.Render("Opened log viewer")
	}

	m.statusIsError = false
}

// cycleSearchCategory cycles through search categories.
func (m *model) cycleSearchCategory() {
	switch m.searchCategory {
	case searchCategoryAll:
		m.searchCategory = searchCategoryDevices
	case searchCategoryDevices:
		m.searchCategory = searchCategoryLogs
	case searchCategoryLogs:
		m.searchCategory = searchCategoryAll
	}
	m.performSearch()
}

// handleSearchCharInput handles character input during search.
func (m *model) handleSearchCharInput(msg tea.KeyMsg) {
	if len(msg.String()) != 1 {
		return
	}
	char := msg.String()[0]
	if char >= 32 && char <= 126 {
		m.searchQuery += msg.String()
		m.performSearch()
	}
}

func (m *model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.cancelSearch()
	case keyEnter:
		m.selectSearchResult()
	case "up":
		if m.selectedResult > 0 {
			m.selectedResult--
		}
	case keyDown:
		if m.selectedResult < len(m.searchResults)-1 {
			m.selectedResult++
		}
	case "tab":
		m.cycleSearchCategory()
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.performSearch()
		}
	default:
		m.handleSearchCharInput(msg)
	}

	return m, nil
}

// searchDevices searches devices by name, type, and IP address.
func (m *model) searchDevices(query string) {
	for i, device := range m.cfg.Devices {
		m.searchDeviceByNameAndType(query, i, device)
		m.searchDeviceByIP(query, i, device)
	}
}

// searchDeviceByNameAndType checks if device name or type matches query.
func (m *model) searchDeviceByNameAndType(query string, idx int, device config.Device) {
	if !strings.Contains(strings.ToLower(device.Name), query) &&
		!strings.Contains(strings.ToLower(device.Type), query) {
		return
	}

	ip := noIPPlaceholder
	if len(device.IPAddresses) > 0 {
		ip = device.IPAddresses[0].String()
	}

	m.searchResults = append(m.searchResults, searchResult{
		Category: "device",
		Title:    device.Name,
		Detail:   fmt.Sprintf("%s - %s - %s", device.Type, ip, device.MACAddress.String()),
		Index:    idx,
	})
}

// searchDeviceByIP checks if any device IP matches query.
func (m *model) searchDeviceByIP(query string, idx int, device config.Device) {
	for _, ip := range device.IPAddresses {
		if strings.Contains(ip.String(), query) {
			m.searchResults = append(m.searchResults, searchResult{
				Category: "device",
				Title:    device.Name,
				Detail:   fmt.Sprintf("%s - %s", device.Type, ip.String()),
				Index:    idx,
			})
			break
		}
	}
}

// searchLogs searches debug logs for matching entries.
func (m *model) searchLogs(query string) {
	for i, log := range m.debugLogs {
		if strings.Contains(strings.ToLower(log), query) {
			m.searchResults = append(m.searchResults, searchResult{
				Category: "log",
				Title:    log,
				Detail:   fmt.Sprintf("Log entry #%d", i+1),
				Index:    i,
			})
		}
	}
}

// searchActiveErrors searches active error injections for matches.
func (m *model) searchActiveErrors(query string) {
	for _, state := range m.stateManager.GetAllStates() {
		if strings.Contains(strings.ToLower(state.DeviceIP), query) ||
			strings.Contains(strings.ToLower(string(state.ErrorType)), query) {
			m.searchResults = append(m.searchResults, searchResult{
				Category: "error",
				Title:    fmt.Sprintf("%s on %s", state.ErrorType, state.DeviceIP),
				Detail:   fmt.Sprintf("Interface: %s, Value: %d%%", state.Interface, state.Value),
				Index:    -1,
			})
		}
	}
}

// performSearch executes the search and populates results.
func (m *model) performSearch() {
	m.searchResults = nil
	m.selectedResult = 0

	if m.searchQuery == "" {
		return
	}

	query := strings.ToLower(m.searchQuery)

	if m.searchCategory == searchCategoryAll || m.searchCategory == searchCategoryDevices {
		m.searchDevices(query)
	}

	if m.searchCategory == searchCategoryAll || m.searchCategory == searchCategoryLogs {
		m.searchLogs(query)
	}

	if m.searchCategory == searchCategoryAll {
		m.searchActiveErrors(query)
	}
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
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

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

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString("║                         Search Mode                              ║\n")
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	queryDisplay := m.searchQuery
	if queryDisplay == "" {
		queryDisplay = "_"
	}

	panel.WriteString(padPanelLine("Query: " + queryDisplay))
	panel.WriteString(padPanelLine("Category: [" + m.searchCategory + "] (Tab to cycle)"))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.searchResults) == 0 {
		m.renderSearchEmptyState(&panel)
	} else {
		m.renderSearchResults(&panel)
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[Enter] Select  [Tab] Category  [ESC] Close"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
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
