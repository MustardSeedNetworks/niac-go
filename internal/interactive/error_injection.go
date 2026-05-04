package interactive

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/krisarmstrong/niac-go/internal/apperr"
)

func (m *model) handleMenuSelection() {
	if m.selectedItem < 0 || m.selectedItem >= len(m.menuItems) {
		return
	}

	selection := m.menuItems[m.selectedItem]

	// Handle menu selections - now with custom value input
	switch {
	case strings.Contains(selection, "FCS Errors"):
		m.promptForValue(apperr.ErrorTypeFCS, "Enter FCS error count (0-100): ")
	case strings.Contains(selection, "Packet Discards"):
		m.promptForValue(apperr.ErrorTypeDiscards, "Enter packet discard rate (0-100): ")
	case strings.Contains(selection, "Interface Errors"):
		m.promptForValue(apperr.ErrorTypeInterface, "Enter interface error count (0-100): ")
	case strings.Contains(selection, "High Utilization"):
		m.promptForValue(apperr.ErrorTypeUtilization, "Enter utilization percentage (0-100): ")
	case strings.Contains(selection, "High CPU"):
		m.promptForValue(apperr.ErrorTypeCPU, "Enter CPU percentage (0-100): ")
	case strings.Contains(selection, "High Memory"):
		m.promptForValue(apperr.ErrorTypeMemory, "Enter memory percentage (0-100): ")
	case strings.Contains(selection, "High Disk"):
		m.promptForValue(apperr.ErrorTypeDisk, "Enter disk percentage (0-100): ")
	case strings.Contains(selection, "Clear All"):
		m.stateManager.ClearAll()
		m.statusMessage = successStyle.Render("✓ All errors cleared")
		m.statusIsError = false
		m.errorsActive = 0
		m.addDebugLog("All error injections cleared")
	case strings.Contains(selection, "Exit"):
		m.menuVisible = false
	}
}

func (m *model) promptForValue(errorType apperr.ErrorType, prompt string) {
	m.selectedErrorType = getErrorTypeIndex(errorType)
	m.valueInputPrompt = prompt
	m.valueInputBuffer = ""
	m.valueInputMode = true
	m.menuVisible = false
}

func (m *model) handleValueInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEnter:
		// Process the input
		var value int

		_, err := fmt.Sscanf(m.valueInputBuffer, "%d", &value)
		if err != nil || value < 0 || value > 100 {
			m.statusMessage = errorStyle.Render("✗ Invalid value. Must be between 0 and 100")
			m.statusIsError = true
		} else {
			errorType := getErrorTypeByIndex(m.selectedErrorType)
			m.injectError(errorType, value)
		}

		m.valueInputMode = false
		m.valueInputBuffer = ""

		return m, nil

	case keyEsc:
		// Cancel input
		m.valueInputMode = false
		m.valueInputBuffer = ""
		m.statusMessage = "Input cancelled"
		m.statusIsError = false

		return m, nil

	case "backspace":
		if len(m.valueInputBuffer) > 0 {
			m.valueInputBuffer = m.valueInputBuffer[:len(m.valueInputBuffer)-1]
		}

		return m, nil

	default:
		// Only accept digits
		if len(msg.String()) == 1 && msg.String()[0] >= '0' && msg.String()[0] <= '9' {
			if len(m.valueInputBuffer) < maxInputDigits {
				m.valueInputBuffer += msg.String()
			}
		}

		return m, nil
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

func (m *model) injectError(errorType apperr.ErrorType, value int) {
	// Inject error on currently selected device
	if len(m.cfg.Devices) == 0 {
		m.statusMessage = errorStyle.Render("✗ No devices configured")
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
		fmt.Sprintf("✓ Injected %s (%d%%) on %s", errorType, value, device.Name),
	)
	m.statusIsError = false
	m.packetsInjected++
	m.errorsActive++
	m.addDebugLog(
		fmt.Sprintf("Injected %s (%d%%) on %s (%s)", errorType, value, device.Name, deviceIP),
	)
}

// renderActiveErrors renders the active error injections section.
func (m *model) renderActiveErrors(s *strings.Builder) {
	activeStates := m.stateManager.GetAllStates()
	if len(activeStates) == 0 {
		return
	}

	s.WriteString(errorStyle.Render("⚠️  Active Error Injections:"))
	s.WriteString("\n")

	for _, state := range activeStates {
		fmt.Fprintf(s, "  • %s on %s:%s (%d%%)\n",
			state.ErrorType,
			state.DeviceIP,
			state.Interface,
			state.Value,
		)
	}
	s.WriteString("\n")
}

func (m *model) renderValueInput() string {
	var input strings.Builder

	input.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	input.WriteString("║                    Error Value Input                            ║\n")
	input.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	fmt.Fprintf(&input, "║ %s%-60s ║\n", "", m.valueInputPrompt)
	input.WriteString("║                                                                  ║\n")

	// Show current input
	inputDisplay := m.valueInputBuffer
	if inputDisplay == "" {
		inputDisplay = "_"
	}

	fmt.Fprintf(&input, "║ Value: %-56s ║\n", inputDisplay)
	input.WriteString("║                                                                  ║\n")
	input.WriteString("║ Press [Enter] to confirm, [Esc] to cancel                       ║\n")
	input.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return input.String()
}

func (m *model) renderMenu() string {
	var menu strings.Builder

	// Get selected device info
	selectedDeviceInfo := "None"

	if len(m.cfg.Devices) > 0 && m.selectedDeviceIdx >= 0 &&
		m.selectedDeviceIdx < len(m.cfg.Devices) {
		device := m.cfg.Devices[m.selectedDeviceIdx]

		deviceIP := noIPPlaceholder
		if len(device.IPAddresses) > 0 {
			deviceIP = device.IPAddresses[0].String()
		}

		selectedDeviceInfo = fmt.Sprintf("%s (%s)", device.Name, deviceIP)
	}

	menu.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	menu.WriteString("║         Interactive Error Injection Menu                        ║\n")
	menu.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	fmt.Fprintf(&menu, "║ Target Device: %-49s ║\n", selectedDeviceInfo)
	menu.WriteString("║ (Press Shift+D to change device)                                ║\n")
	menu.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	for i, item := range m.menuItems {
		if i == m.selectedItem {
			menu.WriteString("║ " + selectedStyle.Render("→ "+item))
		} else {
			menu.WriteString("║   " + item)
		}
		// Pad to align the right border (66 chars wide)
		padding := menuPaddingWidth - len(item) - minEllipsisWidth
		menu.WriteString(strings.Repeat(" ", padding))
		menu.WriteString("║\n")
	}

	menu.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return menu.String()
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

// handleClearErrors clears all error injections.
func (m *model) handleClearErrors() (tea.Model, tea.Cmd) {
	m.stateManager.ClearAll()
	m.statusMessage = successStyle.Render("All error injections cleared")
	m.statusIsError = false
	m.errorsActive = 0
	m.addDebugLog("All error injections cleared")

	return m, nil
}
