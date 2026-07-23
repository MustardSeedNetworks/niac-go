package interactive

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func (m *model) handleMenuSelection() {
	if !m.requireFaultInjection() {
		return
	}
	if m.selectedItem < 0 || m.selectedItem >= len(m.menuItems) {
		return
	}

	selection := m.menuItems[m.selectedItem]

	// Handle menu selections - now with custom value input
	switch {
	case strings.Contains(selection, "FCS Errors"):
		m.promptForValue(devicestate.FaultFCS, "Enter FCS error count (0-100): ")
	case strings.Contains(selection, "Packet Discards"):
		m.promptForValue(devicestate.FaultDiscards, "Enter packet discard rate (0-100): ")
	case strings.Contains(selection, "Interface Errors"):
		m.promptForValue(devicestate.FaultInterface, "Enter interface error count (0-100): ")
	case strings.Contains(selection, "High Utilization"):
		m.promptForValue(devicestate.FaultUtilization, "Enter utilization percentage (0-100): ")
	case strings.Contains(selection, "Clear All"):
		m.stack.ClearAllInterfaceFaults()
		m.statusMessage = successStyle.Render("✓ All errors cleared")
		m.statusIsError = false
		m.errorsActive = 0
		m.addDebugLog("All error injections cleared")
	case strings.Contains(selection, "Exit"):
		m.menuVisible = false
	}
}

func (m *model) promptForValue(errorType devicestate.FaultType, prompt string) {
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

func getErrorTypeIndex(errorType devicestate.FaultType) int {
	types := interfaceFaultTypes()
	for i, t := range types {
		if t == errorType {
			return i
		}
	}

	return 0
}

func getErrorTypeByIndex(index int) devicestate.FaultType {
	types := interfaceFaultTypes()
	if index >= 0 && index < len(types) {
		return types[index]
	}

	return devicestate.FaultFCS
}

func (m *model) injectError(faultType devicestate.FaultType, value int) {
	if !m.requireFaultInjection() {
		return
	}
	if len(m.cfg.Devices) == 0 || m.stack == nil {
		m.statusMessage = errorStyle.Render("✗ No devices configured")
		m.statusIsError = true
		m.addDebugLog("ERROR: No devices configured for error injection")

		return
	}

	// Ensure selectedDeviceIdx is within bounds
	if m.selectedDeviceIdx < 0 || m.selectedDeviceIdx >= len(m.cfg.Devices) {
		m.selectedDeviceIdx = 0
	}

	device := &m.cfg.Devices[m.selectedDeviceIdx]

	_, interfaceName, ok := m.stack.InterfaceFaultTarget(device)
	if !ok {
		m.statusMessage = errorStyle.Render("✗ Selected device has no fault target")
		m.statusIsError = true
		return
	}
	if err := m.stack.SetInterfaceFault(device.Name, interfaceName, faultType, value); err != nil {
		m.statusMessage = errorStyle.Render("✗ " + err.Error())
		m.statusIsError = true
		return
	}
	m.statusMessage = successStyle.Render(
		fmt.Sprintf(
			"✓ Injected %s (%s) on %s",
			faultLabel(faultType),
			faultValue(faultType, value),
			device.Name,
		),
	)
	m.statusIsError = false
	m.packetsInjected++
	m.errorsActive = activeFaultCount(m.stack)
	m.addDebugLog(
		fmt.Sprintf(
			"Injected %s (%s) on %s",
			faultLabel(faultType),
			faultValue(faultType, value),
			device.Name,
		),
	)
}

// renderActiveErrors renders the active error injections section.
func (m *model) renderActiveErrors(s *strings.Builder) {
	if m.stack == nil {
		return
	}
	active := m.stack.ActiveInterfaceFaults()
	if len(active) == 0 {
		return
	}

	s.WriteString(errorStyle.Render("⚠️  Active Error Injections:"))
	s.WriteString("\n")

	for deviceIP, interfaces := range active {
		for interfaceName, faults := range interfaces {
			for faultType, value := range faults {
				fmt.Fprintf(
					s,
					"  • %s on %s:%s (%s)\n",
					faultLabel(faultType),
					deviceIP,
					interfaceName,
					faultValue(faultType, value),
				)
			}
		}
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

// handleQuickErrorInjection handles number keys 1-4 for quick error injection.
func (m *model) handleQuickErrorInjection(key string) (tea.Model, tea.Cmd) {
	if !m.requireFaultInjection() {
		return m, nil
	}
	if m.menuVisible || m.showHelp || m.showLogs || m.showStats {
		return m, nil
	}

	errorTypeMap := map[string]struct {
		errorType devicestate.FaultType
		prompt    string
	}{
		"1": {devicestate.FaultFCS, "Enter FCS error count (0-100): "},
		"2": {devicestate.FaultDiscards, "Enter packet discard rate (0-100): "},
		"3": {devicestate.FaultInterface, "Enter interface error count (0-100): "},
		"4": {devicestate.FaultUtilization, "Enter utilization percentage (0-100): "},
	}

	if errInfo, ok := errorTypeMap[key]; ok {
		m.promptForValue(errInfo.errorType, errInfo.prompt)
	}

	return m, nil
}

// handleClearErrors clears all error injections.
func (m *model) handleClearErrors() (tea.Model, tea.Cmd) {
	if !m.requireFaultInjection() {
		return m, nil
	}
	if m.stack != nil {
		m.stack.ClearAllInterfaceFaults()
	}
	m.statusMessage = successStyle.Render("All error injections cleared")
	m.statusIsError = false
	m.errorsActive = 0
	m.addDebugLog("All error injections cleared")

	return m, nil
}

func (m *model) requireFaultInjection() bool {
	if m.faultInjectionEnabled {
		return true
	}
	m.menuVisible = false
	m.valueInputMode = false
	m.statusMessage = errorStyle.Render("Error injection requires NIAC Pro")
	m.statusIsError = true
	return false
}

func interfaceFaultTypes() []devicestate.FaultType {
	definitions := devicestate.InterfaceFaultDefinitions()
	result := make([]devicestate.FaultType, 0, len(definitions))
	for _, definition := range definitions {
		result = append(result, definition.Type)
	}
	return result
}

func faultLabel(faultType devicestate.FaultType) string {
	if label := faultType.Label(); label != "" {
		return label
	}
	return string(faultType)
}

func faultValue(faultType devicestate.FaultType, value int) string {
	if faultType == devicestate.FaultUtilization {
		return fmt.Sprintf("%d%%", value)
	}
	return fmt.Sprintf("%d/s", value)
}

func activeFaultCount(stack *protocols.Stack) int {
	if stack == nil {
		return 0
	}
	count := 0
	for _, interfaces := range stack.ActiveInterfaceFaults() {
		for _, faults := range interfaces {
			count += len(faults)
		}
	}
	return count
}
