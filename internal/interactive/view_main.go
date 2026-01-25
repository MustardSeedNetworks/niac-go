package interactive

import (
	"fmt"
	"strings"
)

// renderValueInput renders the value input prompt overlay.
func (m *model) renderValueInput() string {
	var input strings.Builder

	input.WriteString("+=================================================================+\n")
	input.WriteString("|                    Error Value Input                            |\n")
	input.WriteString("+=================================================================+\n")
	input.WriteString(fmt.Sprintf("| %s%-60s |\n", "", m.valueInputPrompt))
	input.WriteString("|                                                                  |\n")

	// Show current input
	inputDisplay := m.valueInputBuffer
	if inputDisplay == "" {
		inputDisplay = "_"
	}

	input.WriteString(fmt.Sprintf("| Value: %-56s |\n", inputDisplay))
	input.WriteString("|                                                                  |\n")
	input.WriteString("| Press [Enter] to confirm, [Esc] to cancel                       |\n")
	input.WriteString("+=================================================================+")

	return input.String()
}

// renderMenu renders the interactive error injection menu.
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

	menu.WriteString("+=================================================================+\n")
	menu.WriteString("|         Interactive Error Injection Menu                        |\n")
	menu.WriteString("+=================================================================+\n")
	menu.WriteString(fmt.Sprintf("| Target Device: %-49s |\n", selectedDeviceInfo))
	menu.WriteString("| (Press Shift+D to change device)                                |\n")
	menu.WriteString("+=================================================================+\n")

	for i, item := range m.menuItems {
		if i == m.selectedItem {
			menu.WriteString("| " + selectedStyle.Render("-> "+item))
		} else {
			menu.WriteString("|   " + item)
		}
		// Pad to align the right border (66 chars wide)
		padding := menuPaddingWidth - len(item) - minEllipsisWidth
		menu.WriteString(strings.Repeat(" ", padding))
		menu.WriteString("|\n")
	}

	menu.WriteString("+=================================================================+")

	return menu.String()
}
