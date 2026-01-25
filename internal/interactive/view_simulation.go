package interactive

import (
	"fmt"
	"strings"
)

// renderHexDump renders the packet hex dump viewer.
func (m *model) renderHexDump() string {
	var dump strings.Builder

	dump.WriteString("+=================================================================+\n")
	dump.WriteString("|                    Packet Hex Dump Viewer                        |\n")
	dump.WriteString("+=================================================================+\n")

	if len(m.packetBuffer) == 0 {
		dump.WriteString("| No packets captured yet                                          |\n")
		dump.WriteString("| Packets will appear here as they are received                    |\n")
		dump.WriteString("+=================================================================+")

		return dump.String()
	}

	// Get current packet
	if m.hexDumpPacketIndex >= len(m.packetBuffer) {
		m.hexDumpPacketIndex = len(m.packetBuffer) - 1
	}

	pkt := m.packetBuffer[m.hexDumpPacketIndex]

	// Packet metadata
	dump.WriteString(
		fmt.Sprintf("| Packet: %d/%d                                                    |\n",
			m.hexDumpPacketIndex+1, len(m.packetBuffer)),
	)
	dump.WriteString(fmt.Sprintf("| Time:     %-54s |\n", pkt.Timestamp.Format("15:04:05.000000")))
	dump.WriteString(fmt.Sprintf("| Protocol: %-54s |\n", pkt.Protocol))
	dump.WriteString(fmt.Sprintf("| Source:   %-54s |\n", pkt.SrcAddr))
	dump.WriteString(fmt.Sprintf("| Dest:     %-54s |\n", pkt.DstAddr))
	dump.WriteString(fmt.Sprintf("| Length:   %-54d |\n", pkt.Length))
	dump.WriteString("+=================================================================+\n")
	dump.WriteString("| Offset   Hex                                      ASCII          |\n")
	dump.WriteString("+=================================================================+\n")

	// Calculate number of lines to display (hexBytesPerLine bytes per line)
	maxLines := 15 // Display max 15 lines
	totalLines := (len(pkt.Data) + hexBytesPerLine - 1) / hexBytesPerLine

	startLine := m.hexDumpScrollY
	if startLine >= totalLines {
		startLine = max(totalLines-1, 0)
	}

	endLine := min(startLine+maxLines, totalLines)

	// Render hex dump lines
	for line := startLine; line < endLine; line++ {
		offset := line * hexBytesPerLine
		end := min(offset+hexBytesPerLine, len(pkt.Data))

		// Offset
		lineStr := fmt.Sprintf("| %04x   ", offset)

		// Hex bytes
		var hexBuilder, asciiBuilder strings.Builder

		for i := offset; i < end; i++ {
			b := pkt.Data[i]
			fmt.Fprintf(&hexBuilder, "%02x ", b)

			if b >= 32 && b <= 126 {
				asciiBuilder.WriteByte(b)
			} else {
				asciiBuilder.WriteByte('.')
			}
		}

		// Pad hex to align ASCII column (48 chars for 16 bytes)
		hexStr := fmt.Sprintf("%-48s", hexBuilder.String())
		asciiStr := fmt.Sprintf("%-16s", asciiBuilder.String())

		lineStr += hexStr + " " + asciiStr + " |\n"
		dump.WriteString(lineStr)
	}

	// Show scroll indicator if needed
	if totalLines > maxLines {
		dump.WriteString("+=================================================================+\n")
		dump.WriteString(
			fmt.Sprintf("| Showing lines %d-%d of %d (use up/dn/PgUp/PgDn to scroll)        |\n",
				startLine+1, endLine, totalLines),
		)
	}

	dump.WriteString("+=================================================================+\n")
	dump.WriteString("| Press [n] next packet  [p] previous packet  [x] close           |\n")
	dump.WriteString("+=================================================================+")

	return dump.String()
}

// renderPcapReplay renders the PCAP replay control panel.
func (m *model) renderPcapReplay() string {
	var panel strings.Builder

	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine(diffHeaderStyle.Render("PCAP Replay Control")))
	panel.WriteString("+=================================================================+\n")

	if len(m.pcapPackets) == 0 {
		panel.WriteString(padPanelLine("No PCAP file loaded"))
		panel.WriteString(padPanelLine("Use the hex dump viewer to capture packets"))
	} else {
		// Playback status
		status := "PAUSED"
		if m.pcapPlaying {
			status = successStyle.Render("PLAYING")
		}

		panel.WriteString(
			padPanelLine(fmt.Sprintf("Status: %s  Speed: %.2fx", status, m.pcapPlaybackSpeed)),
		)
		panel.WriteString(
			padPanelLine(fmt.Sprintf("Packet: %d / %d", m.pcapPlaybackIndex+1, len(m.pcapPackets))),
		)

		// Progress bar
		barWidth := 50

		progress := 0
		if len(m.pcapPackets) > 0 {
			progress = (m.pcapPlaybackIndex * barWidth) / len(m.pcapPackets)
		}

		progressBar := "[" + strings.Repeat(
			"=",
			progress,
		) + strings.Repeat(
			"-",
			barWidth-progress,
		) + "]"
		panel.WriteString(padPanelLine(progressBar))

		panel.WriteString("+=================================================================+\n")

		// Current packet info
		if m.pcapPlaybackIndex >= 0 && m.pcapPlaybackIndex < len(m.pcapPackets) {
			pkt := m.pcapPackets[m.pcapPlaybackIndex]
			panel.WriteString(padPanelLine("Time: " + pkt.Timestamp.Format("15:04:05.000")))
			panel.WriteString(
				padPanelLine(
					fmt.Sprintf("Protocol: %s  Length: %d bytes", pkt.Protocol, pkt.Length),
				),
			)
			panel.WriteString(padPanelLine("Src: " + pkt.SrcAddr + " -> Dst: " + pkt.DstAddr))
		}
	}

	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine("[Space] Play/Pause  [left/right] Step  [+/-] Speed  [r] Restart"))
	panel.WriteString(padPanelLine("[ESC] Close"))
	panel.WriteString("+=================================================================+")

	return panel.String()
}

// renderHistory renders the run history viewer panel.
func (m *model) renderHistory() string {
	var panel strings.Builder

	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine(diffHeaderStyle.Render("Run History")))
	panel.WriteString("+=================================================================+\n")

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
				prefix = selectedStyle.Render("-> ")
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

	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine(fmt.Sprintf("Total runs: %d", len(m.historyEntries))))
	panel.WriteString("+=================================================================+\n")
	panel.WriteString(
		padPanelLine("[up/dn] Navigate  [Enter] Details  [PgUp/PgDn] Scroll  [ESC] Close"),
	)
	panel.WriteString("+=================================================================+")

	return panel.String()
}

// renderSnmpWalk renders the SNMP walk browser panel.
func (m *model) renderSnmpWalk() string {
	var panel strings.Builder

	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine(diffHeaderStyle.Render("SNMP Walk Browser")))
	panel.WriteString("+=================================================================+\n")

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
				prefix = selectedStyle.Render("-> ")
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

	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine(fmt.Sprintf("OIDs: %d", len(m.snmpOidTree))))
	panel.WriteString("+=================================================================+\n")
	panel.WriteString(padPanelLine("[up/dn] Navigate  [Enter] Expand  [/] Search  [ESC] Close"))
	panel.WriteString("+=================================================================+")

	return panel.String()
}
