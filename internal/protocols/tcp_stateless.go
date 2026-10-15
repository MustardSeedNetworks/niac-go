package protocols

import (
	"net"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// serviceDevice is the device that owns ip and serves the port, or nil when
// none does and the port is closed.
func (h *TCPHandler) serviceDevice(
	devices []*config.Device,
	ip net.IP,
	serves func(*config.Device) bool,
) *config.Device {
	for _, device := range devices {
		if serves(device) && h.stack.deviceOwnsIPv4(device, ip) {
			return device
		}
	}

	return nil
}

// serveStream answers one segment of a connection to a service that reads a
// byte stream, served by device, or resets a SYN when device is nil.
func (h *TCPHandler) serveStream(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	devices []*config.Device,
	device *config.Device,
	streams *serviceStreams,
	answer streamAnswer,
	service string,
) {
	key := newStreamKey(ipLayer, tcp)
	switch {
	case device == nil:
		if tcp.SYN && !tcp.ACK {
			h.sendRST(pkt, ipLayer, tcp, devices)
		}
	case tcp.RST:
		streams.close(key)
	case tcp.SYN && !tcp.ACK:
		streams.open(key, tcp.Seq+1)
		h.stack.healthCheckHandler.HandleTCPConnect(pkt, ipLayer, tcp, []*config.Device{device}, uint16(tcp.DstPort))
	case len(tcp.Payload) > 0 || tcp.FIN:
		reply, inOrder := streams.receive(key, tcp.Seq, tcp.Payload, answer)
		if !inOrder {
			return // the client retransmits what is missing
		}
		if tcp.FIN {
			streams.close(key)
		}
		h.sendStatelessSegment(pkt, ipLayer, tcp, device, reply, service)
	}
}

// statelessMaxSegment is the most answer one segment carries: the IPv4 MSS
// of a 1500-byte Ethernet MTU, the default every SYN the services answer
// advertises or exceeds.
const statelessMaxSegment = 1460

// sendStatelessSegment acknowledges the client's segment for a service that
// keeps no connection state of its own, carrying answer when there is one and
// closing our side when the client closed its own. The service answers each
// request before the client sends the next, so the client's ACK is our
// sequence number. An answer longer than one segment goes out as several.
func (h *TCPHandler) sendStatelessSegment(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	device *config.Device,
	answer []byte,
	service string,
) {
	identity, ok := h.stack.replyEthernet(pkt, device)
	if !ok {
		return
	}
	consumed := safeconv.Uint32(min(len(tcp.Payload), maxUint32Payload))
	if tcp.FIN {
		consumed++ // a FIN occupies one sequence number
	}
	seq := tcp.Ack
	for {
		chunk := answer[:min(len(answer), statelessMaxSegment)]
		answer = answer[len(chunk):]
		tcpReply := &layers.TCP{
			SrcPort: tcp.DstPort, DstPort: tcp.SrcPort,
			Seq: seq, Ack: tcp.Seq + consumed,
			ACK: true, PSH: len(chunk) > 0, FIN: tcp.FIN && len(answer) == 0, Window: tcpWindowSize,
		}
		if !h.sendStatelessFrame(pkt, ipLayer, tcpReply, chunk, device, identity, service) || len(answer) == 0 {
			return
		}
		seq += safeconv.Uint32(len(chunk))
	}
}

func (h *TCPHandler) sendStatelessFrame(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcpReply *layers.TCP,
	payload []byte,
	device *config.Device,
	identity replyEthernetIdentity,
	service string,
) bool {
	ipReply := &layers.IPv4{
		Version: tcpIPv4Version, IHL: tcpIPv4IHL, TTL: tcpIPv4TTL,
		Protocol: layers.IPProtocolTCP, SrcIP: ipLayer.DstIP, DstIP: ipLayer.SrcIP,
	}
	_ = tcpReply.SetNetworkLayerForChecksum(ipReply)

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	eth := &layers.Ethernet{
		SrcMAC: identity.source, DstMAC: identity.destination, EthernetType: layers.EthernetTypeIPv4,
	}
	if err := gopacket.SerializeLayers(buffer, opts, eth, ipReply, tcpReply, gopacket.Payload(payload)); err != nil {
		if h.stack.GetDebugLevel() >= DebugLevelInfo {
			logging.Debugf("%s: reply from %s failed sn=%d err=%v", service, device.Name, pkt.SerialNumber, err)
		}
		return false
	}

	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()
	h.stack.Send(&Packet{
		Buffer: buffer.Bytes(), Length: len(buffer.Bytes()), SerialNumber: serialNum,
		Device: device, VLAN: identity.vlan, generatedHost: device,
	})

	return true
}
