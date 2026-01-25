package interactive

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// renderDeviceConfigTabBar renders the tab bar for device config view.
func (m *model) renderDeviceConfigTabBar() string {
	tabs := []string{"General", "Interfaces", "Protocols", "SNMP"}
	var tabBar strings.Builder

	for i, tab := range tabs {
		if i == m.deviceConfigTab {
			tabBar.WriteString(selectedStyle.Render("[" + tab + "]"))
		} else {
			tabBar.WriteString(" " + tab + " ")
		}
		if i < len(tabs)-1 {
			tabBar.WriteString(" | ")
		}
	}
	return tabBar.String()
}

// renderDeviceGeneralTab renders the general tab content.
func renderDeviceGeneralTab(panel *strings.Builder, device *config.Device) {
	panel.WriteString(padPanelLine("Name:        " + device.Name))
	panel.WriteString(padPanelLine("Type:        " + device.Type))
	panel.WriteString(padPanelLine("MAC Address: " + device.MACAddress.String()))

	if len(device.IPAddresses) > 0 {
		panel.WriteString(padPanelLine("IP Address:  " + device.IPAddresses[0].String()))
	}
	if device.VLAN > 0 {
		panel.WriteString(padPanelLine(fmt.Sprintf("VLAN:        %d", device.VLAN)))
	}
	panel.WriteString(padPanelLine("Babble:      " + boolToEnabled(device.Babble)))
}

// renderDeviceInterfacesTab renders the interfaces tab content.
func renderDeviceInterfacesTab(panel *strings.Builder, device *config.Device) {
	if len(device.Interfaces) == 0 {
		panel.WriteString(padPanelLine("No interfaces configured"))
		return
	}

	panel.WriteString(
		padPanelLine(
			fmt.Sprintf("%-15s %-10s %-10s %-8s", "Interface", "Speed", "Duplex", "Status"),
		),
	)
	panel.WriteString(padPanelLine(strings.Repeat("-", deviceConfigTableWidth)))

	for _, iface := range device.Interfaces {
		status := iface.AdminStatus
		if status == "" {
			status = "up"
		}
		panel.WriteString(padPanelLine(fmt.Sprintf("%-15s %-10d %-10s %-8s",
			iface.Name, iface.Speed, iface.Duplex, status)))
	}
}

// renderDeviceProtocolsTab renders the protocols tab content.
func renderDeviceProtocolsTab(panel *strings.Builder, device *config.Device) {
	panel.WriteString(padPanelLine("LLDP:    " + boolToEnabled(device.LLDPConfig != nil)))
	panel.WriteString(padPanelLine("CDP:     " + boolToEnabled(device.CDPConfig != nil)))
	panel.WriteString(padPanelLine("STP:     " + boolToEnabled(device.STPConfig != nil)))
	panel.WriteString(padPanelLine("EDP:     " + boolToEnabled(device.EDPConfig != nil)))
	panel.WriteString(padPanelLine("FDP:     " + boolToEnabled(device.FDPConfig != nil)))
}

// renderDeviceSNMPTab renders the SNMP tab content.
func renderDeviceSNMPTab(panel *strings.Builder, device *config.Device) {
	if device.SNMPConfig.Community == "" {
		panel.WriteString(padPanelLine("SNMP not configured for this device"))
		return
	}
	panel.WriteString(padPanelLine("Community:  " + device.SNMPConfig.Community))
	panel.WriteString(padPanelLine("SysName:    " + device.SNMPConfig.SysName))
	panel.WriteString(padPanelLine("SysDescr:   " + device.SNMPConfig.SysDescr))
	panel.WriteString(padPanelLine("Contact:    " + device.SNMPConfig.SysContact))
	panel.WriteString(padPanelLine("Location:   " + device.SNMPConfig.SysLocation))
}

// renderDeviceConfigContent renders the content for the current tab.
func (m *model) renderDeviceConfigContent(panel *strings.Builder, device *config.Device) {
	if device == nil {
		panel.WriteString(padPanelLine("Select a device with [D] to view configuration"))
		return
	}

	switch m.deviceConfigTab {
	case deviceConfigTabGeneral:
		renderDeviceGeneralTab(panel, device)
	case deviceConfigTabInterface:
		renderDeviceInterfacesTab(panel, device)
	case deviceConfigTabProtocols:
		renderDeviceProtocolsTab(panel, device)
	case deviceConfigTabSNMP:
		renderDeviceSNMPTab(panel, device)
	}
}

func (m *model) renderDeviceConfig() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")

	var device *config.Device
	deviceName := "No Device Selected"

	if m.selectedDeviceIdx >= 0 && m.selectedDeviceIdx < len(m.cfg.Devices) {
		device = &m.cfg.Devices[m.selectedDeviceIdx]
		deviceName = device.Name
	}

	panel.WriteString(padPanelLine(diffHeaderStyle.Render("Device Configuration: " + deviceName)))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine(m.renderDeviceConfigTabBar()))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	m.renderDeviceConfigContent(&panel, device)

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[Tab] Switch Tab  [↑↓] Scroll  [ESC] Close"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// handleDeviceConfigInput handles keyboard input in device config panel.
func (m *model) handleDeviceConfigInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.showDeviceConfig = false

		return m, nil
	case "tab":
		m.deviceConfigTab = (m.deviceConfigTab + 1) % deviceConfigTabCount
		m.deviceConfigScrollY = 0

		return m, nil
	case "up":
		if m.deviceConfigScrollY > 0 {
			m.deviceConfigScrollY--
		}

		return m, nil
	case keyDown:
		m.deviceConfigScrollY++

		return m, nil
	}

	return m, nil
}

// handleDeviceConfigToggle toggles the device configuration panel.
func (m *model) handleDeviceConfigToggle() (tea.Model, tea.Cmd) {
	if m.showDeviceConfig {
		m.showDeviceConfig = false

		return m, nil
	}

	m.showDeviceConfig = true
	m.deviceConfigTab = 0
	m.deviceConfigScrollY = 0
	m.closeAllOverlays()
	m.statusMessage = "Device Config - [Tab] switch tab, [↑↓] scroll"
	m.statusIsError = false

	return m, nil
}
