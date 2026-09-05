package protocols

import (
	"fmt"
	"net"
	"slices"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// SNMP protocol constants.
const (
	snmpMACAddrLen = 6 // MAC address length in bytes

	// ASN.1 BER length encoding: high bit of the first length octet flags the
	// long form; the low 7 bits then hold the count of subsequent length octets.
	asn1LongFormFlag     = 0x80
	asn1LengthOctetsMask = 0x7f
)

// SNMPHandler routes SNMP queries to per-device agents.
type SNMPHandler struct {
	stack *Stack
}

// NewSNMPHandler creates an SNMP handler bound to the stack.
func NewSNMPHandler(stack *Stack) *SNMPHandler {
	return &SNMPHandler{stack: stack}
}

// HandlePacket processes an SNMP request delivered over IPv4/UDP.
func (h *SNMPHandler) HandlePacket(
	pkt *Packet,
	ip *layers.IPv4,
	udp *layers.UDP,
	devices []*config.Device,
) {
	h.handleDatagram(pkt, snmpPeer{src: ip.SrcIP, dst: ip.DstIP}, udp, devices)
}

// HandlePacketV6 serves the same request arriving over IPv6.
//
// Everything below this point is version-agnostic - the MIB, the PDU codec and
// the USM engine never see an IP layer - so the two entry points differ only in
// how the peer addresses are read and how the reply is framed.
func (h *SNMPHandler) HandlePacketV6(
	pkt *Packet,
	ip *layers.IPv6,
	udp *layers.UDP,
	devices []*config.Device,
) {
	h.handleDatagram(pkt, snmpPeer{src: ip.SrcIP, dst: ip.DstIP, v6: true}, udp, devices)
}

const (
	snmpIPv6Version  = 6
	snmpIPv6HopLimit = 64
)

// snmpPeer is the pair of addresses a request arrived between, and which IP
// version carried it. Keeping the handler on this rather than on a layer type
// is what stops the v4 and v6 paths becoming two implementations that drift.
type snmpPeer struct {
	src net.IP
	dst net.IP
	v6  bool
}

func (h *SNMPHandler) handleDatagram(
	pkt *Packet,
	peer snmpPeer,
	udp *layers.UDP,
	devices []*config.Device,
) {
	if h == nil || h.stack == nil || h.stack.udpHandler == nil {
		return
	}

	if len(udp.Payload) == 0 {
		return
	}

	// SNMPv3 is handled from the raw datagram (USM engine discovery + auth +
	// privacy); v1/v2c continue through the community-keyed agent path below.
	if ver, ok := snmpPeekVersion(udp.Payload); ok && ver == gosnmp.Version3 {
		h.handleV3(pkt, peer, udp, devices)
		return
	}
	if ver, ok := snmpPeekVersion(udp.Payload); ok && ver != gosnmp.Version1 && ver != gosnmp.Version2c {
		h.recordForDevices(devices, func(agent *snmp.Agent) { agent.RecordBadVersion() })
		return
	}

	request, err := h.decodeRequest(udp.Payload)
	if err != nil {
		h.recordForDevices(devices, func(agent *snmp.Agent) { agent.RecordASNParseError() })
		if h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= DebugLevelInfo {
			logging.Debugf("SNMP: decode failed for %s sn=%d err=%v", peer.dst, pkt.SerialNumber, err)
		}

		return
	}

	for _, device := range devices {
		h.processDeviceRequest(pkt, peer, udp, device, request)
	}
}

// processDeviceRequest handles SNMP request for a single device.
func (h *SNMPHandler) processDeviceRequest(
	pkt *Packet,
	peer snmpPeer,
	udp *layers.UDP,
	device *config.Device,
	request *gosnmp.SnmpPacket,
) {
	agent := h.findAgent(device, peer.src, request.Community)
	if agent == nil {
		return
	}
	agent.RecordInboundPacket()
	agent.RecordInboundError(request.Error)

	payload, status, err := h.buildResponse(agent, request)
	if err != nil {
		if h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= 1 {
			logging.Debugf(
				"SNMP: marshal response failed for device %s sn=%d err=%v",
				device.Name, pkt.SerialNumber, err)
		}

		return
	}

	h.stack.stats.mu.Lock()
	h.stack.stats.SNMPQueries++
	h.stack.stats.mu.Unlock()

	if h.sendResponse(pkt, peer, udp, device, payload) {
		agent.RecordResponse(status)
	}
}

// findAgent finds the appropriate SNMP agent for a device request.
func (h *SNMPHandler) findAgent(device *config.Device, srcIP net.IP, community string) *snmp.Agent {
	group := h.stack.getSNMPAgents(device)
	if group == nil {
		return nil
	}

	if !snmpAccessAllowed(device, srcIP) {
		if agent := group.Get(community); agent != nil {
			agent.RecordBadCommunityUse()
		}
		return nil
	}
	agent := group.Get(community)
	if agent == nil {
		if observer := group.observer(); observer != nil {
			observer.RecordBadCommunityName()
		}
	}
	return agent
}

// buildResponse creates and marshals an SNMP response.
func (h *SNMPHandler) buildResponse(
	agent *snmp.Agent,
	request *gosnmp.SnmpPacket,
) ([]byte, gosnmp.SNMPError, error) {
	// The simulator speaks SNMP v1/v2c only. gosnmp's MarshalMsg dispatches a
	// Version3 packet to marshalV3, which dereferences SecurityParameters we
	// never populate → nil-pointer SIGSEGV. Discovery tools such as NetAlly
	// CyberScope/AirCheck send SNMPv3 probes, so decline unsupported versions
	// here rather than letting one probe crash the whole simulator.
	if request.Version != gosnmp.Version1 && request.Version != gosnmp.Version2c {
		return nil, gosnmp.GenErr, fmt.Errorf("%w: %v", ErrUnsupportedSNMPVersion, request.Version)
	}

	responseVars, status, errorIndex := agent.ProcessRequest(request)

	response := &gosnmp.SnmpPacket{
		Version:    request.Version,
		Community:  request.Community,
		PDUType:    gosnmp.GetResponse,
		RequestID:  request.RequestID,
		Error:      status,
		ErrorIndex: errorIndex,
		Variables:  responseVars,
	}

	payload, err := response.MarshalMsg()
	return payload, status, err
}

// sendResponse sends the SNMP response packet.
func (h *SNMPHandler) sendResponse(
	pkt *Packet,
	peer snmpPeer,
	udp *layers.UDP,
	device *config.Device,
	payload []byte,
) bool {
	srcMAC := h.sourceMAC(device, pkt)
	dstMAC := pkt.GetSourceMAC()

	if len(dstMAC) == 0 || len(srcMAC) == 0 {
		return false
	}
	// The reply leaves from the address the manager asked, not from whichever
	// address the device happens to prefer: a manager only accepts an answer
	// from the address it queried, which is what makes this load-bearing for a
	// link-local one.
	if peer.v6 {
		return h.sendResponseV6(pkt, peer, udp, device.Name, srcMAC, dstMAC, payload)
	}

	srcIP := peer.dst.To4()
	dstIP := peer.src.To4()
	if srcIP == nil || dstIP == nil {
		return false
	}

	err := h.stack.udpHandler.SendUDP(
		srcIP, dstIP,
		uint16(udp.DstPort), uint16(udp.SrcPort),
		payload,
		[]byte(srcMAC), []byte(dstMAC),
		pkt.VLAN,
	)
	if err != nil && h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= 1 {
		logging.Debugf(
			"SNMP: failed to emit response for device %s sn=%d err=%v",
			device.Name, pkt.SerialNumber, err)
	}
	return err == nil
}

// sendResponseV6 frames the reply as Ethernet/IPv6/UDP. IPv6 makes the UDP
// checksum mandatory, unlike IPv4 where it may be zero, so the checksum is
// computed rather than left off.
func (h *SNMPHandler) sendResponseV6(
	pkt *Packet,
	peer snmpPeer,
	udp *layers.UDP,
	deviceName string,
	srcMAC, dstMAC net.HardwareAddr,
	payload []byte,
) bool {
	reply := &layers.UDP{SrcPort: udp.DstPort, DstPort: udp.SrcPort}
	ip := &layers.IPv6{
		Version:    snmpIPv6Version,
		NextHeader: layers.IPProtocolUDP,
		HopLimit:   snmpIPv6HopLimit,
		SrcIP:      peer.dst,
		DstIP:      peer.src,
	}
	eth := &layers.Ethernet{
		SrcMAC: srcMAC, DstMAC: dstMAC, EthernetType: layers.EthernetTypeIPv6,
	}
	_ = reply.SetNetworkLayerForChecksum(ip)

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	if err := gopacket.SerializeLayers(
		buf, opts, eth, ip, reply, gopacket.Payload(payload),
	); err != nil {
		if h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= 1 {
			logging.Debugf(
				"SNMP: failed to serialize IPv6 response for device %s sn=%d err=%v",
				deviceName, pkt.SerialNumber, err)
		}

		return false
	}

	// Reply on the VLAN the query arrived on (0 = untagged).
	return h.stack.SendRawPacketVLAN(buf.Bytes(), pkt.VLAN) == nil
}

func (h *SNMPHandler) recordForDevices(devices []*config.Device, record func(*snmp.Agent)) {
	for _, device := range devices {
		if group := h.stack.getSNMPAgents(device); group != nil {
			if agent := group.observer(); agent != nil {
				record(agent)
			}
		}
	}
}

func snmpAccessAllowed(device *config.Device, srcIP net.IP) bool {
	if device == nil || len(device.SNMPConfig.AccessList) == 0 {
		return true
	}

	return slices.ContainsFunc(device.SNMPConfig.AccessList, func(ip net.IP) bool {
		return ip != nil && ip.Equal(srcIP)
	})
}

func (h *SNMPHandler) decodeRequest(payload []byte) (*gosnmp.SnmpPacket, error) {
	decoder := gosnmp.GoSNMP{
		Transport: "udp",
		Version:   gosnmp.Version2c,
		Community: config.DefaultSNMPCommunity,
		MaxOids:   gosnmp.MaxOids,
	}

	pkt, err := decoder.SnmpDecodePacket(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SNMP packet: %w", err)
	}
	return pkt, nil
}

// handleV3 routes an SNMPv3 datagram to each matching device's USM engine,
// emitting the engine's discovery Report or authenticated response.
func (h *SNMPHandler) handleV3(pkt *Packet, peer snmpPeer, udp *layers.UDP, devices []*config.Device) {
	for _, device := range devices {
		group := h.stack.getSNMPAgents(device)
		if group == nil || group.v3 == nil || group.v3Agent == nil {
			continue
		}

		if !snmpAccessAllowed(device, peer.src) {
			continue
		}

		resp, err := group.v3.Respond(udp.Payload, group.v3Agent.ProcessPDU)
		if err != nil || resp == nil {
			if err != nil && h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= DebugLevelInfo {
				logging.Debugf(
					"SNMP: v3 respond declined for %s sn=%d err=%v", device.Name, pkt.SerialNumber, err)
			}

			continue
		}

		h.stack.stats.mu.Lock()
		h.stack.stats.SNMPQueries++
		h.stack.stats.mu.Unlock()

		if h.sendResponse(pkt, peer, udp, device, resp) {
			group.v3Agent.RecordResponse(gosnmp.NoError)
		}
	}
}

// snmpPeekVersion extracts the SNMP version from a raw datagram without a full
// decode: the outer SEQUENCE's first element is INTEGER version.
func snmpPeekVersion(payload []byte) (gosnmp.SnmpVersion, bool) {
	if len(payload) < 2 || payload[0] != byte(gosnmp.Sequence) {
		return 0, false
	}

	// Skip the SEQUENCE length octets (short or long form).
	cursor := 1
	if payload[cursor]&asn1LongFormFlag != 0 {
		cursor += 1 + int(payload[cursor]&asn1LengthOctetsMask)
	} else {
		cursor++
	}

	// Expect INTEGER (0x02), single-octet length, then the version value.
	if cursor+2 >= len(payload) || payload[cursor] != byte(gosnmp.Integer) {
		return 0, false
	}
	length := int(payload[cursor+1])
	if length != 1 || cursor+2 >= len(payload) {
		return 0, false
	}

	return gosnmp.SnmpVersion(payload[cursor+2]), true
}

func (h *SNMPHandler) sourceMAC(device *config.Device, pkt *Packet) net.HardwareAddr {
	if h.stack != nil {
		if mac := h.stack.replySourceMAC(pkt, device); len(mac) == snmpMACAddrLen {
			return mac
		}
	}
	if len(device.MACAddress) == snmpMACAddrLen {
		return device.MACAddress
	}

	return pkt.GetDestMAC()
}
