package interactive

import (
	"fmt"
	"strings"
	"time"

	"github.com/krisarmstrong/niac-go/internal/apperr"
	"github.com/krisarmstrong/niac-go/internal/config"
)

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % minutesPerHour
	seconds := int(d.Seconds()) % secondsPerMinute

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func padPanelLine(text string) string {
	if len(text) > panelContentWidth {
		if panelContentWidth > minEllipsisWidth {
			text = text[:panelContentWidth-minEllipsisWidth] + "..."
		} else {
			text = text[:panelContentWidth]
		}
	}

	return fmt.Sprintf("║ %-64s ║\n", text)
}

func fitColumn(text string, width int) string {
	if width <= 0 {
		return ""
	}

	if len(text) > width {
		if width > minEllipsisWidth {
			text = text[:width-minEllipsisWidth] + "..."
		} else {
			text = text[:width]
		}
	}

	return fmt.Sprintf("%-*s", width, text)
}

func formatRelativeTime(ts time.Time) string {
	if ts.IsZero() {
		return "never"
	}

	elapsed := max(time.Since(ts), 0)
	if elapsed < time.Second {
		return "now"
	}

	if elapsed < time.Minute {
		return fmt.Sprintf("%ds ago", int(elapsed.Seconds()))
	}

	if elapsed < time.Hour {
		return fmt.Sprintf(
			"%dm%ds ago",
			int(elapsed.Minutes()),
			int(elapsed.Seconds())%secondsPerMinute,
		)
	}

	hours := int(elapsed.Hours())
	minutes := int(elapsed.Minutes()) % minutesPerHour

	if hours >= maxHoursNoMinutes {
		return fmt.Sprintf("%dh", hours)
	}

	return fmt.Sprintf("%dh%dm", hours, minutes)
}

func getDebugLevelName(level int) string {
	switch level {
	case 0:
		return "QUIET"
	case 1:
		return "NORMAL"
	case debugLevelVerbose:
		return "VERBOSE"
	case debugLevelDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

func (m *model) addDebugLog(message string) {
	timestamp := time.Now().Format("15:04:05")
	logEntry := fmt.Sprintf("[%s] %s", timestamp, message)
	m.debugLogs = append(m.debugLogs, logEntry)

	// Keep only last maxDebugLogs log entries
	if len(m.debugLogs) > maxDebugLogs {
		m.debugLogs = m.debugLogs[len(m.debugLogs)-maxDebugLogs:]
	}
}

func getErrorTypeIndex(errorType apperr.ErrorType) int {
	types := apperr.AllErrorTypes()
	for i, t := range types {
		if t == errorType {
			return i
		}
	}

	return 0
}

func getErrorTypeByIndex(index int) apperr.ErrorType {
	types := apperr.AllErrorTypes()
	if index >= 0 && index < len(types) {
		return types[index]
	}

	return apperr.ErrorTypeFCS
}

// truncateName truncates a name to fit in the diagram.
func truncateName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}

	if maxLen <= minEllipsisWidth {
		return name[:maxLen]
	}

	return name[:maxLen-minEllipsisWidth] + "..."
}

// boolToEnabled returns "Enabled" or "Disabled" based on the boolean.
func boolToEnabled(b bool) string {
	if b {
		return successStyle.Render("Enabled")
	}

	return "Disabled"
}

// handleScrollInput is a shared helper for handling scroll navigation in list views.
// It updates selectedIdx and scrollY based on the key pressed.
// Returns true if a navigation key was handled, false otherwise.
// visibleRows is the number of visible rows in the viewport.
func handleScrollInput(
	key string,
	selectedIdx, scrollY, listLen, visibleRows int,
) (int, int, bool) {
	switch key {
	case "up":
		if selectedIdx > 0 {
			selectedIdx--
			if selectedIdx < scrollY {
				scrollY = selectedIdx
			}
		}

		return selectedIdx, scrollY, true
	case keyDown:
		if selectedIdx < listLen-1 {
			selectedIdx++
			if selectedIdx >= scrollY+visibleRows {
				scrollY++
			}
		}

		return selectedIdx, scrollY, true
	case "pgup":
		scrollY -= visibleRows
		if scrollY < 0 {
			scrollY = 0
		}

		return selectedIdx, scrollY, true
	case "pgdown":
		scrollY += visibleRows
		maxScroll := max(listLen-visibleRows, 0)
		scrollY = min(scrollY, maxScroll)

		return selectedIdx, scrollY, true
	}

	return selectedIdx, scrollY, false
}

// getSelectedDeviceName returns the name of the currently selected device.
func (m *model) getSelectedDeviceName() string {
	if len(m.cfg.Devices) > 0 && m.selectedDeviceIdx >= 0 &&
		m.selectedDeviceIdx < len(m.cfg.Devices) {
		return m.cfg.Devices[m.selectedDeviceIdx].Name
	}
	return "None"
}

// getDevicesSummary returns a summary of devices for export.
func (m *model) getDevicesSummary() []map[string]string {
	devices := make([]map[string]string, 0, len(m.cfg.Devices))
	for _, device := range m.cfg.Devices {
		ip := noIPPlaceholder
		if len(device.IPAddresses) > 0 {
			ip = device.IPAddresses[0].String()
		}

		devices = append(devices, map[string]string{
			"name": device.Name,
			"type": device.Type,
			"ip":   ip,
			"mac":  device.MACAddress.String(),
		})
	}

	return devices
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

// refreshStats updates the model with current stack statistics.
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

func (m *model) toggleNeighborView() {
	m.showNeighbors = !m.showNeighbors
	m.showHelp = false
	m.showLogs = false
	m.showStats = false
	m.showHexDump = false

	m.menuVisible = false
	if m.showNeighbors {
		if len(m.neighbors) == 0 {
			m.statusMessage = "Neighbor table opened - waiting for discovery packets"
			m.statusIsError = false
		} else {
			m.statusMessage = successStyle.Render(
				fmt.Sprintf("Showing %d learned neighbors", len(m.neighbors)),
			)
			m.statusIsError = false
		}
	}
}

func (m *model) promptForValue(errorType apperr.ErrorType, prompt string) {
	m.selectedErrorType = getErrorTypeIndex(errorType)
	m.valueInputPrompt = prompt
	m.valueInputBuffer = ""
	m.valueInputMode = true
	m.menuVisible = false
}

func (m *model) injectError(errorType apperr.ErrorType, value int) {
	// Inject error on currently selected device
	if len(m.cfg.Devices) == 0 {
		m.statusMessage = errorStyle.Render("No devices configured")
		m.statusIsError = true
		m.addDebugLog("ERROR: No devices configured for error injection")

		return
	}

	// Ensure selectedDeviceIdx is within bounds
	if m.selectedDeviceIdx < 0 || m.selectedDeviceIdx >= len(m.cfg.Devices) {
		m.selectedDeviceIdx = 0
	}

	device := m.cfg.Devices[m.selectedDeviceIdx]

	deviceIP := "unknown"
	if len(device.IPAddresses) > 0 {
		deviceIP = device.IPAddresses[0].String()
	}

	m.stateManager.SetError(deviceIP, "eth0", errorType, value)
	m.statusMessage = successStyle.Render(
		fmt.Sprintf("Injected %s (%d%%) on %s", errorType, value, device.Name),
	)
	m.statusIsError = false
	m.packetsInjected++
	m.errorsActive++
	m.addDebugLog(
		fmt.Sprintf("Injected %s (%d%%) on %s (%s)", errorType, value, device.Name, deviceIP),
	)
}

// setValidationStatus sets the status message based on validation results.
func (m *model) setValidationStatus() {
	switch {
	case m.validationResults.HasErrors():
		m.statusMessage = errorStyle.Render(
			fmt.Sprintf("Validation found %d error(s), %d warning(s)",
				len(m.validationResults.Errors), len(m.validationResults.Warnings)),
		)
		m.statusIsError = true
	case m.validationResults.HasWarnings():
		m.statusMessage = fmt.Sprintf(
			"Validation passed with %d warning(s)",
			len(m.validationResults.Warnings),
		)
		m.statusIsError = false
	default:
		m.statusMessage = successStyle.Render("Validation passed - no errors or warnings")
		m.statusIsError = false
	}
}

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
