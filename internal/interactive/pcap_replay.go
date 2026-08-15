package interactive

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m *model) handlePcapReplayInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.showPcapReplay = false
		m.pcapPlaying = false
		m.statusMessage = "PCAP replay closed"
		m.statusIsError = false

		return m, nil

	case " ":
		// Toggle play/pause
		m.pcapPlaying = !m.pcapPlaying
		if m.pcapPlaying {
			m.statusMessage = "PCAP playback started"
		} else {
			m.statusMessage = "PCAP playback paused"
		}

		m.statusIsError = false

		return m, nil

	case "left":
		// Step backward one packet
		if m.pcapPlaybackIndex > 0 {
			m.pcapPlaybackIndex--
		}

		return m, nil

	case "right":
		// Step forward one packet
		if m.pcapPlaybackIndex < len(m.pcapPackets)-1 {
			m.pcapPlaybackIndex++
		}

		return m, nil

	case "+", "=":
		// Increase playback speed
		if m.pcapPlaybackSpeed < maxPlaybackSpeed {
			m.pcapPlaybackSpeed *= 2
		}

		return m, nil

	case "-", "_":
		// Decrease playback speed
		if m.pcapPlaybackSpeed > minPlaybackSpeed {
			m.pcapPlaybackSpeed /= 2
		}

		return m, nil

	case "r":
		// Restart from beginning
		m.pcapPlaybackIndex = 0
		m.statusMessage = "PCAP replay restarted"
		m.statusIsError = false

		return m, nil
	}

	return m, nil
}

// renderPcapReplay renders the PCAP replay control panel.
func (m *model) renderPcapReplay() string {
	var panel strings.Builder

	panel.WriteString("╔══════════════════════════════════════════════════════════════════╗\n")
	panel.WriteString(padPanelLine(diffHeaderStyle.Render("PCAP Replay Control")))
	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

	if len(m.pcapPackets) == 0 {
		panel.WriteString(padPanelLine("No PCAP file loaded"))
		panel.WriteString(padPanelLine("Use the hex dump viewer to capture packets"))
	} else {
		// Playback status
		status := "⏸ PAUSED"
		if m.pcapPlaying {
			status = successStyle.Render("▶ PLAYING")
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

		panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")

		// Current packet info
		if m.pcapPlaybackIndex >= 0 && m.pcapPlaybackIndex < len(m.pcapPackets) {
			pkt := m.pcapPackets[m.pcapPlaybackIndex]
			panel.WriteString(padPanelLine("Time: " + pkt.Timestamp.Format("15:04:05.000")))
			panel.WriteString(
				padPanelLine(
					fmt.Sprintf("Protocol: %s  Length: %d bytes", pkt.Protocol, pkt.Length),
				),
			)
			panel.WriteString(padPanelLine("Src: " + pkt.SrcAddr + " → Dst: " + pkt.DstAddr))
		}
	}

	panel.WriteString("╠══════════════════════════════════════════════════════════════════╣\n")
	panel.WriteString(padPanelLine("[Space] Play/Pause  [←→] Step  [+/-] Speed  [r] Restart"))
	panel.WriteString(padPanelLine("[ESC] Close"))
	panel.WriteString("╚══════════════════════════════════════════════════════════════════╝")

	return panel.String()
}

// handlePcapReplayToggle toggles the PCAP replay panel.
func (m *model) handlePcapReplayToggle() (tea.Model, tea.Cmd) {
	if m.showPcapReplay {
		m.showPcapReplay = false
		m.pcapPlaying = false

		return m, nil
	}

	m.pcapPackets = make([]CapturedPacket, len(m.packetBuffer))
	copy(m.pcapPackets, m.packetBuffer)
	m.pcapPlaybackIndex = 0
	m.pcapPlaying = false

	if m.pcapPlaybackSpeed == 0 {
		m.pcapPlaybackSpeed = 1.0
	}

	m.showPcapReplay = true
	m.closeAllOverlays()
	m.statusMessage = "PCAP Replay - [Space] play/pause, [←→] step, [+/-] speed"
	m.statusIsError = false

	return m, nil
}
