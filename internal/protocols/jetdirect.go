package protocols

import (
	"strings"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// TCPPortJetDirect is the raw print port, also called AppSocket or port 9100.
const TCPPortJetDirect = 9100

// pjlUEL is the Universal Exit Language sequence that opens and closes a PJL
// job. A client may put it in front of the first command on a line.
const pjlUEL = "\x1b%-12345X"

// pjlFormFeed ends every PJL INFO answer, so a client knows it has all of it.
const pjlFormFeed = "\f"

// handleJetDirect serves 9100 on a printer with JetDirect enabled. It is
// stateless, like the health-check ports: the connection is accepted, print
// data is acknowledged and discarded, PJL queries are answered, and a FIN is
// answered with a FIN. Every other device keeps the closed-port RST.
func (h *TCPHandler) handleJetDirect(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	devices []*config.Device,
) {
	device := h.serviceDevice(devices, ipLayer.DstIP, func(device *config.Device) bool {
		return device.JetDirectConfig != nil && device.JetDirectConfig.Enabled
	})
	switch {
	case device == nil:
		if tcp.SYN && !tcp.ACK {
			h.sendRST(pkt, ipLayer, tcp, devices)
		}
	case tcp.RST:
	case tcp.SYN && !tcp.ACK:
		h.stack.healthCheckHandler.HandleTCPConnect(pkt, ipLayer, tcp, []*config.Device{device}, TCPPortJetDirect)
	case len(tcp.Payload) > 0 || tcp.FIN:
		answer := pjlAnswer(tcp.Payload, device.JetDirectConfig.PJLID)
		h.sendStatelessSegment(pkt, ipLayer, tcp, device, []byte(answer), "JetDirect")
	}
}

// pjlAnswer is the reply to the PJL INFO ID and INFO STATUS commands in
// payload, in the order they arrived, echoing each command the way an HP
// printer does. A printer without a PJL ID speaks no PJL, and every other
// byte is print data, which a printer consumes without answering.
func pjlAnswer(payload []byte, pjlID string) string {
	if pjlID == "" {
		return ""
	}
	var answer strings.Builder
	for line := range strings.SplitSeq(string(payload), "\n") {
		command := strings.TrimSuffix(strings.TrimPrefix(line, pjlUEL), "\r")
		fields := strings.Fields(command)
		if len(fields) != 3 || fields[0] != "@PJL" || !strings.EqualFold(fields[1], "INFO") {
			continue
		}
		switch strings.ToUpper(fields[2]) {
		case "ID":
			answer.WriteString("@PJL INFO ID\r\n\"" + pjlID + "\"\r\n" + pjlFormFeed)
		case "STATUS":
			// 10001 is the PJL status code for "ready, online".
			answer.WriteString("@PJL INFO STATUS\r\nCODE=10001\r\nDISPLAY=\"Ready\"\r\nONLINE=TRUE\r\n" + pjlFormFeed)
		}
	}

	return answer.String()
}
