package protocols

import (
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// sendSYNACK sends a TCP SYN-ACK response.
func (h *HealthCheckHandler) sendSYNACK(
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
	reqPkt *Packet,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(devices) == 0 {
		return
	}

	device := devices[0]
	if len(device.MACAddress) == 0 {
		return
	}

	identity, ok := h.stack.replyEthernet(reqPkt, device)
	if !ok {
		if debugLevel >= DebugLevelInfo {
			logging.Debugf("Cannot send SYN-ACK: no source MAC for %s", ipLayer.SrcIP)
		}

		return
	}

	// Build Ethernet header
	eth := &layers.Ethernet{
		SrcMAC:       identity.source,
		DstMAC:       identity.destination,
		EthernetType: layers.EthernetTypeIPv4,
	}

	// Build IP header
	ipReply := &layers.IPv4{
		Version:  hcIPv4Version,
		IHL:      hcIPv4IHL,
		TTL:      hcIPv4TTL,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    ipLayer.DstIP,
		DstIP:    ipLayer.SrcIP,
	}

	// Build TCP SYN-ACK
	tcpReply := &layers.TCP{
		SrcPort: tcpLayer.DstPort,
		DstPort: tcpLayer.SrcPort,
		Seq:     hcTCPInitialSeq, // Initial sequence number
		Ack:     tcpLayer.Seq + 1,
		SYN:     true,
		ACK:     true,
		Window:  hcTCPWindowSize,
	}
	_ = tcpReply.SetNetworkLayerForChecksum(ipReply) // error is non-critical for simulation

	// Serialize
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipReply, tcpReply)
	if err != nil {
		if debugLevel >= DebugLevelInfo {
			logging.Debugf("Error serializing SYN-ACK: %v", err)
		}

		return
	}

	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	pkt := &Packet{
		Buffer:        buffer.Bytes(),
		Length:        len(buffer.Bytes()),
		SerialNumber:  serialNum,
		Device:        device,
		VLAN:          identity.vlan,
		generatedHost: device,
	}

	h.stack.Send(pkt)

	if debugLevel >= DebugLevelVerbose {
		logging.Debugf("Sent TCP SYN-ACK from %s:%d to %s:%d device=%s",
			ipReply.SrcIP, tcpReply.SrcPort, ipReply.DstIP, tcpReply.DstPort, device.Name)
	}
}

// sendTCPResponse sends a TCP response with payload.
func (h *HealthCheckHandler) sendTCPResponse(
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	payload []byte,
	devices []*config.Device,
	reqPkt *Packet,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(devices) == 0 || len(payload) == 0 {
		return
	}

	device := devices[0]
	if len(device.MACAddress) == 0 {
		return
	}

	identity, ok := h.stack.replyEthernet(reqPkt, device)
	if !ok {
		return
	}

	eth := &layers.Ethernet{
		SrcMAC:       identity.source,
		DstMAC:       identity.destination,
		EthernetType: layers.EthernetTypeIPv4,
	}

	ipReply := &layers.IPv4{
		Version:  hcIPv4Version,
		IHL:      hcIPv4IHL,
		TTL:      hcIPv4TTL,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    ipLayer.DstIP,
		DstIP:    ipLayer.SrcIP,
	}

	// Safe conversion: min() bounds payloadLen to 0xFFFFFFFF which fits in uint32
	payloadLen := min(len(tcpLayer.Payload), maxUint32Val)

	tcpReply := &layers.TCP{
		SrcPort: tcpLayer.DstPort,
		DstPort: tcpLayer.SrcPort,
		Seq:     tcpLayer.Ack,
		Ack:     tcpLayer.Seq + safeconv.Uint32(payloadLen),
		PSH:     true,
		ACK:     true,
		Window:  hcTCPWindowSize,
	}
	_ = tcpReply.SetNetworkLayerForChecksum(ipReply) // error is non-critical for simulation

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipReply, tcpReply, gopacket.Payload(payload))
	if err != nil {
		if debugLevel >= DebugLevelInfo {
			logging.Debugf("Error serializing TCP response: %v", err)
		}

		return
	}

	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	pkt := &Packet{
		Buffer:        buffer.Bytes(),
		Length:        len(buffer.Bytes()),
		SerialNumber:  serialNum,
		Device:        device,
		VLAN:          identity.vlan,
		generatedHost: device,
	}

	h.stack.Send(pkt)
}
