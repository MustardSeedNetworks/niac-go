package interactive

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// renderSnmpWalk renders the SNMP walk browser panel.
func (m *model) renderSnmpWalk() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString(padPanelLine(diffHeaderStyle.Render("SNMP Walk Browser")))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.snmpOidTree) == 0 {
		panel.WriteString(padPanelLine("No SNMP data available"))
		panel.WriteString(padPanelLine("SNMP must be enabled in the simulation"))
	} else {
		// Display OID tree
		startIdx := m.snmpScrollY
		endIdx := min(startIdx+snmpVisibleRows, len(m.snmpOidTree))

		for i := startIdx; i < endIdx; i++ {
			oid := m.snmpOidTree[i]

			prefix := "  "
			if i == m.selectedSnmpOid {
				prefix = selectedStyle.Render("→ ")
			}

			// Truncate long values
			value := oid.Value
			if len(value) > snmpValueTruncateLen {
				value = value[:snmpValueDisplayWidth] + "..."
			}

			line := fmt.Sprintf("%s%-25s %-8s %s", prefix, oid.Name, oid.Type, value)
			panel.WriteString(padPanelLine(line))
		}
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine(fmt.Sprintf("OIDs: %d", len(m.snmpOidTree))))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[↑↓] Navigate  [Enter] Expand  [/] Search  [ESC] Close"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// handleSnmpWalkInput handles keyboard input in SNMP walk browser.
func (m *model) handleSnmpWalkInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == keyEsc {
		m.showSnmpWalk = false

		return m, nil
	}

	const visibleRows = 12

	newIdx, newScroll, handled := handleScrollInput(
		key,
		m.selectedSnmpOid,
		m.snmpScrollY,
		len(m.snmpOidTree),
		visibleRows,
	)
	if handled {
		m.selectedSnmpOid = newIdx
		m.snmpScrollY = newScroll
	}

	return m, nil
}

// handleSnmpWalkToggle toggles the SNMP walk browser.
func (m *model) handleSnmpWalkToggle() (tea.Model, tea.Cmd) {
	if m.showSnmpWalk {
		m.showSnmpWalk = false

		return m, nil
	}

	m.snmpOidTree = []SnmpOidEntry{
		{OID: ".1.3.6.1.2.1.1.1.0", Name: "sysDescr", Value: "Network Simulator", Type: "STRING"},
		{OID: ".1.3.6.1.2.1.1.2.0", Name: "sysObjectID", Value: ".1.3.6.1.4.1.99999", Type: "OID"},
		{
			OID:   ".1.3.6.1.2.1.1.3.0",
			Name:  "sysUpTime",
			Value: strconv.Itoa(int(m.uptime.Seconds() * uptimeTicksMultiplier)),
			Type:  "TIMETICKS",
		},
		{OID: ".1.3.6.1.2.1.1.4.0", Name: "sysContact", Value: "admin@niac.local", Type: "STRING"},
		{OID: ".1.3.6.1.2.1.1.5.0", Name: "sysName", Value: m.interfaceName, Type: "STRING"},
		{OID: ".1.3.6.1.2.1.1.6.0", Name: "sysLocation", Value: "Network Lab", Type: "STRING"},
	}
	m.selectedSnmpOid = 0
	m.snmpScrollY = 0
	m.showSnmpWalk = true
	m.closeAllOverlays()
	m.statusMessage = "SNMP Walk Browser - [↑↓] navigate, [Enter] expand"
	m.statusIsError = false

	return m, nil
}
