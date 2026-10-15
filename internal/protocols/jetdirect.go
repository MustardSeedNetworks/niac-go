package protocols

import (
	"net"
	"strings"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
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
	device := h.jetDirectDevice(devices, ipLayer.DstIP)
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
		h.sendJetDirectSegment(pkt, ipLayer, tcp, device, answer)
	}
}

func (h *TCPHandler) jetDirectDevice(devices []*config.Device, ip net.IP) *config.Device {
	for _, device := range devices {
		if device.JetDirectConfig != nil && device.JetDirectConfig.Enabled && h.stack.deviceOwnsIPv4(device, ip) {
			return device
		}
	}

	return nil
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

// sendJetDirectSegment acknowledges the client's segment, carrying answer
// when there is one and closing our side when the client closed its own.
func (h *TCPHandler) sendJetDirectSegment(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	device *config.Device,
	answer string,
) {
	identity, ok := h.stack.replyEthernet(pkt, device)
	if !ok {
		return
	}
	consumed := safeconv.Uint32(min(len(tcp.Payload), maxUint32Payload))
	if tcp.FIN {
		consumed++ // a FIN occupies one sequence number
	}
	ipReply := &layers.IPv4{
		Version: tcpIPv4Version, IHL: tcpIPv4IHL, TTL: tcpIPv4TTL,
		Protocol: layers.IPProtocolTCP, SrcIP: ipLayer.DstIP, DstIP: ipLayer.SrcIP,
	}
	tcpReply := &layers.TCP{
		SrcPort: tcp.DstPort, DstPort: tcp.SrcPort,
		Seq: tcp.Ack, Ack: tcp.Seq + consumed,
		ACK: true, PSH: answer != "", FIN: tcp.FIN, Window: tcpWindowSize,
	}
	_ = tcpReply.SetNetworkLayerForChecksum(ipReply)

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	eth := &layers.Ethernet{
		SrcMAC: identity.source, DstMAC: identity.destination, EthernetType: layers.EthernetTypeIPv4,
	}
	if err := gopacket.SerializeLayers(buffer, opts, eth, ipReply, tcpReply, gopacket.Payload(answer)); err != nil {
		if h.stack.GetDebugLevel() >= DebugLevelInfo {
			logging.Debugf("JetDirect: reply from %s failed sn=%d err=%v", device.Name, pkt.SerialNumber, err)
		}
		return
	}

	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()
	h.stack.Send(&Packet{
		Buffer: buffer.Bytes(), Length: len(buffer.Bytes()), SerialNumber: serialNum,
		Device: device, VLAN: identity.vlan, generatedHost: device,
	})
}
