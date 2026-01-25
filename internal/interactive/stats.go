package interactive

import (
	"fmt"
	"sort"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/protocols"
)

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
				fmt.Sprintf("✓ Showing %d learned neighbors", len(m.neighbors)),
			)
			m.statusIsError = false
		}
	}
}

// getSelectedDeviceName returns the name of the currently selected device.
func (m *model) getSelectedDeviceName() string {
	if len(m.cfg.Devices) > 0 && m.selectedDeviceIdx >= 0 &&
		m.selectedDeviceIdx < len(m.cfg.Devices) {
		return m.cfg.Devices[m.selectedDeviceIdx].Name
	}
	return "None"
}

// renderStatsHeader writes the statistics panel header.
func (m *model) renderStatsHeader(stats *strings.Builder) {
	stats.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	stats.WriteString("║                     Detailed Statistics                          ║\n")
	stats.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	fmt.Fprintf(
		stats,
		"║ Uptime:              %s                                    ║\n",
		formatDuration(m.uptime),
	)
	fmt.Fprintf(
		stats,
		"║ Debug Level:         %d (%s)                              ║\n",
		m.debugLevel,
		getDebugLevelName(m.debugLevel),
	)
	fmt.Fprintf(stats, "║ Interface:           %-40s ║\n", m.interfaceName)
	stats.WriteString("║                                                                  ║\n")
}

// renderPacketStats writes packet statistics.
func (m *model) renderPacketStats(stats *strings.Builder) {
	totalPackets := m.stackStats.PacketsReceived + m.stackStats.PacketsSent
	fmt.Fprintf(
		stats,
		"║ Total Packets:       %-10d                                    ║\n",
		totalPackets,
	)
	fmt.Fprintf(
		stats,
		"║ RX / TX Packets:     %-10d / %-10d                       ║\n",
		m.stackStats.PacketsReceived,
		m.stackStats.PacketsSent,
	)
	fmt.Fprintf(
		stats,
		"║ ARP Req / Rep:       %-10d / %-10d                       ║\n",
		m.stackStats.ARPRequests,
		m.stackStats.ARPReplies,
	)
	fmt.Fprintf(
		stats,
		"║ ICMP Req / Rep:      %-10d / %-10d                       ║\n",
		m.stackStats.ICMPRequests,
		m.stackStats.ICMPReplies,
	)
	fmt.Fprintf(
		stats,
		"║ DNS Queries:         %-10d                                    ║\n",
		m.stackStats.DNSQueries,
	)
	fmt.Fprintf(
		stats,
		"║ DHCP Requests:       %-10d                                    ║\n",
		m.stackStats.DHCPRequests,
	)
	fmt.Fprintf(
		stats,
		"║ Packets Injected:    %-10d                                    ║\n",
		m.packetsInjected,
	)
	fmt.Fprintf(
		stats,
		"║ Active Errors:       %-10d                                    ║\n",
		m.errorsActive,
	)
}

// renderDeviceStats writes device statistics.
func (m *model) renderDeviceStats(stats *strings.Builder) {
	stats.WriteString("║                                                                  ║\n")
	fmt.Fprintf(
		stats,
		"║ Devices Simulated:   %-10d                                    ║\n",
		len(m.cfg.Devices),
	)

	snmpCount := 0
	for _, dev := range m.cfg.Devices {
		if dev.SNMPConfig.Community != "" || dev.SNMPConfig.WalkFile != "" {
			snmpCount++
		}
	}
	fmt.Fprintf(
		stats,
		"║ SNMP Devices:        %-10d                                    ║\n",
		snmpCount,
	)
	stats.WriteString("║                                                                  ║\n")
	fmt.Fprintf(
		stats,
		"║ Start Time:          %s                                    ║\n",
		m.startTime.Format("15:04:05"),
	)
	stats.WriteString("╚══════════════════════════════════════════════════════════════════╝")
}

func (m *model) renderStatistics() string {
	var stats strings.Builder
	m.renderStatsHeader(&stats)
	m.renderPacketStats(&stats)
	m.renderDeviceStats(&stats)
	return stats.String()
}

func (m *model) renderNeighbors() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString("║                    Neighbor Discovery Table                      ║\n")
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.neighbors) == 0 {
		panel.WriteString(padPanelLine("No neighbors discovered yet"))
		panel.WriteString(padPanelLine("Advertise LLDP/CDP/EDP/FDP to populate this view"))
		panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

		return panel.String()
	}

	rows := make([]protocols.NeighborRecord, len(m.neighbors))
	copy(rows, m.neighbors)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].LocalDevice != rows[j].LocalDevice {
			return rows[i].LocalDevice < rows[j].LocalDevice
		}

		if rows[i].Protocol != rows[j].Protocol {
			return rows[i].Protocol < rows[j].Protocol
		}

		return rows[i].RemoteDevice < rows[j].RemoteDevice
	})

	header := fmt.Sprintf("%s %s %s %s %s",
		fitColumn("Proto", colWidthProto),
		fitColumn("Local Device", colWidthLocalDevice),
		fitColumn("Remote (Port)", colWidthRemote),
		fitColumn("Mgmt Address", colWidthMgmt),
		fitColumn("Seen", colWidthSeen),
	)
	panel.WriteString(padPanelLine(header))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	for _, entry := range rows {
		remote := entry.RemoteDevice
		if entry.RemotePort != "" {
			if remote == "" {
				remote = entry.RemotePort
			} else {
				remote = fmt.Sprintf("%s/%s", remote, entry.RemotePort)
			}
		}

		mgmt := entry.ManagementAddress
		if mgmt == "" {
			mgmt = "-"
		}

		line := fmt.Sprintf("%s %s %s %s %s",
			fitColumn(strings.ToUpper(entry.Protocol), colWidthProto),
			fitColumn(entry.LocalDevice, colWidthLocalDevice),
			fitColumn(remote, colWidthRemote),
			fitColumn(mgmt, colWidthMgmt),
			fitColumn(formatRelativeTime(entry.LastSeen), colWidthSeen),
		)
		panel.WriteString(padPanelLine(line))
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	summary := fmt.Sprintf("Total neighbors: %d  •  TTL refresh every 30s", len(rows))
	panel.WriteString(padPanelLine(summary))
	panel.WriteString(padPanelLine("Press [N]/[n] to close this view"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}
