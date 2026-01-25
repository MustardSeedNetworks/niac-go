package interactive

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) renderHexDump() string {
	var dump strings.Builder

	dump.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	dump.WriteString("║                    Packet Hex Dump Viewer                        ║\n")
	dump.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.packetBuffer) == 0 {
		dump.WriteString("║ No packets captured yet                                          ║\n")
		dump.WriteString("║ Packets will appear here as they are received                    ║\n")
		dump.WriteString("╚══════════════════════════════════════════════════════════════════╝")

		return dump.String()
	}

	// Get current packet
	if m.hexDumpPacketIndex >= len(m.packetBuffer) {
		m.hexDumpPacketIndex = len(m.packetBuffer) - 1
	}

	pkt := m.packetBuffer[m.hexDumpPacketIndex]

	// Packet metadata
	dump.WriteString(
		fmt.Sprintf("║ Packet: %d/%d                                                    ║\n",
			m.hexDumpPacketIndex+1, len(m.packetBuffer)),
	)
	dump.WriteString(fmt.Sprintf("║ Time:     %-54s ║\n", pkt.Timestamp.Format("15:04:05.000000")))
	dump.WriteString(fmt.Sprintf("║ Protocol: %-54s ║\n", pkt.Protocol))
	dump.WriteString(fmt.Sprintf("║ Source:   %-54s ║\n", pkt.SrcAddr))
	dump.WriteString(fmt.Sprintf("║ Dest:     %-54s ║\n", pkt.DstAddr))
	dump.WriteString(fmt.Sprintf("║ Length:   %-54d ║\n", pkt.Length))
	dump.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	dump.WriteString("║ Offset   Hex                                      ASCII          ║\n")
	dump.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

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
		lineStr := fmt.Sprintf("║ %04x   ", offset)

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

		lineStr += hexStr + " " + asciiStr + " ║\n"
		dump.WriteString(lineStr)
	}

	// Show scroll indicator if needed
	if totalLines > maxLines {
		dump.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
		dump.WriteString(
			fmt.Sprintf("║ Showing lines %d-%d of %d (use ↑/↓/PgUp/PgDn to scroll)        ║\n",
				startLine+1, endLine, totalLines),
		)
	}

	dump.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	dump.WriteString("║ Press [n] next packet  [p] previous packet  [x] close           ║\n")
	dump.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return dump.String()
}

// AddPacket adds a packet to the capture buffer.
func (m *model) AddPacket(protocol, srcAddr, dstAddr string, data []byte) {
	pkt := CapturedPacket{
		Timestamp: time.Now(),
		Protocol:  protocol,
		SrcAddr:   srcAddr,
		DstAddr:   dstAddr,
		Length:    len(data),
		Data:      make([]byte, len(data)),
	}
	copy(pkt.Data, data)

	m.packetBuffer = append(m.packetBuffer, pkt)

	// Keep only last maxPacketBuffer packets
	if len(m.packetBuffer) > maxPacketBuffer {
		m.packetBuffer = m.packetBuffer[len(m.packetBuffer)-maxPacketBuffer:]
		// Adjust index if needed
		if m.hexDumpPacketIndex >= len(m.packetBuffer) {
			m.hexDumpPacketIndex = len(m.packetBuffer) - 1
		}
	}
}

// handleHexDumpNextPacket navigates to the next packet in hex dump viewer.
func (m *model) handleHexDumpNextPacket() (tea.Model, tea.Cmd) {
	if !m.showHexDump || len(m.packetBuffer) == 0 {
		return m, nil
	}

	m.hexDumpPacketIndex = (m.hexDumpPacketIndex + 1) % len(m.packetBuffer)
	m.hexDumpScrollY = 0
	m.statusMessage = successStyle.Render(
		fmt.Sprintf("✓ Packet %d/%d", m.hexDumpPacketIndex+1, len(m.packetBuffer)),
	)

	return m, nil
}

// handleHexDumpPrevPacket navigates to the previous packet in hex dump viewer.
func (m *model) handleHexDumpPrevPacket() (tea.Model, tea.Cmd) {
	if !m.showHexDump || len(m.packetBuffer) == 0 {
		return m, nil
	}

	m.hexDumpPacketIndex--
	if m.hexDumpPacketIndex < 0 {
		m.hexDumpPacketIndex = len(m.packetBuffer) - 1
	}

	m.hexDumpScrollY = 0
	m.statusMessage = successStyle.Render(
		fmt.Sprintf("✓ Packet %d/%d", m.hexDumpPacketIndex+1, len(m.packetBuffer)),
	)

	return m, nil
}
