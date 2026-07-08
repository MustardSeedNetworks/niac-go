package protocols

import (
	"fmt"
	"net"
	"os"
	"slices"

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
func (h *SNMPHandler) HandlePacket(pkt *Packet, ip *layers.IPv4, udp *layers.UDP, devices []*config.Device) {
	if h == nil || h.stack == nil || h.stack.udpHandler == nil {
		return
	}

	if len(udp.Payload) == 0 {
		return
	}

	// SNMPv3 is handled from the raw datagram (USM engine discovery + auth +
	// privacy); v1/v2c continue through the community-keyed agent path below.
	if ver, ok := snmpPeekVersion(udp.Payload); ok && ver == gosnmp.Version3 {
		h.handleV3(pkt, ip, udp, devices)
		return
	}

	request, err := h.decodeRequest(udp.Payload)
	if err != nil {
		if h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "SNMP: decode failed for %s sn=%d err=%v\n", ip.DstIP, pkt.SerialNumber, err)
		}

		return
	}

	for _, device := range devices {
		h.processDeviceRequest(pkt, ip, udp, device, request)
	}
}

// processDeviceRequest handles SNMP request for a single device.
func (h *SNMPHandler) processDeviceRequest(
	pkt *Packet,
	ip *layers.IPv4,
	udp *layers.UDP,
	device *config.Device,
	request *gosnmp.SnmpPacket,
) {
	agent := h.findAgent(device, ip.SrcIP, request.Community)
	if agent == nil {
		return
	}

	payload, err := h.buildResponse(agent, request)
	if err != nil {
		if h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= 1 {
			_, _ = fmt.Fprintf(os.Stdout,
				"SNMP: marshal response failed for device %s sn=%d err=%v\n",
				device.Name, pkt.SerialNumber, err)
		}

		return
	}

	h.stack.stats.mu.Lock()
	h.stack.stats.SNMPQueries++
	h.stack.stats.mu.Unlock()

	h.sendResponse(pkt, ip, udp, device, payload)
}

// findAgent finds the appropriate SNMP agent for a device request.
func (h *SNMPHandler) findAgent(device *config.Device, srcIP net.IP, community string) *snmp.Agent {
	group := h.stack.getSNMPAgents(device)
	if group == nil {
		return nil
	}

	if !snmpAccessAllowed(device, srcIP) {
		return nil
	}

	agent := group.Get(community)
	if agent == nil {
		agent = group.Get("public")
	}

	return agent
}

// buildResponse creates and marshals an SNMP response.
func (h *SNMPHandler) buildResponse(agent *snmp.Agent, request *gosnmp.SnmpPacket) ([]byte, error) {
	// The simulator speaks SNMP v1/v2c only. gosnmp's MarshalMsg dispatches a
	// Version3 packet to marshalV3, which dereferences SecurityParameters we
	// never populate → nil-pointer SIGSEGV. Discovery tools such as NetAlly
	// CyberScope/AirCheck send SNMPv3 probes, so decline unsupported versions
	// here rather than letting one probe crash the whole simulator.
	if request.Version != gosnmp.Version1 && request.Version != gosnmp.Version2c {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedSNMPVersion, request.Version)
	}

	responseVars := agent.ProcessPDU(
		request.PDUType,
		request.Variables,
		int(request.NonRepeaters),
		request.MaxRepetitions,
	)

	response := &gosnmp.SnmpPacket{
		Version:    request.Version,
		Community:  request.Community,
		PDUType:    gosnmp.GetResponse,
		RequestID:  request.RequestID,
		Error:      gosnmp.NoError,
		ErrorIndex: 0,
		Variables:  responseVars,
	}

	return response.MarshalMsg()
}

// sendResponse sends the SNMP response packet.
func (h *SNMPHandler) sendResponse(
	pkt *Packet,
	ip *layers.IPv4,
	udp *layers.UDP,
	device *config.Device,
	payload []byte,
) {
	srcIP := ip.DstIP.To4()
	dstIP := ip.SrcIP.To4()

	if srcIP == nil || dstIP == nil {
		return
	}

	srcMAC := h.sourceMAC(device, pkt)
	dstMAC := pkt.GetSourceMAC()

	if len(dstMAC) == 0 || len(srcMAC) == 0 {
		return
	}

	err := h.stack.udpHandler.SendUDP(
		srcIP, dstIP,
		uint16(udp.DstPort), uint16(udp.SrcPort),
		payload,
		[]byte(srcMAC), []byte(dstMAC),
		pkt.VLAN,
	)
	if err != nil && h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= 1 {
		_, _ = fmt.Fprintf(os.Stdout,
			"SNMP: failed to emit response for device %s sn=%d err=%v\n",
			device.Name, pkt.SerialNumber, err)
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
		Community: "public",
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
func (h *SNMPHandler) handleV3(pkt *Packet, ip *layers.IPv4, udp *layers.UDP, devices []*config.Device) {
	for _, device := range devices {
		group := h.stack.getSNMPAgents(device)
		if group == nil || group.v3 == nil || group.v3Agent == nil {
			continue
		}

		if !snmpAccessAllowed(device, ip.SrcIP) {
			continue
		}

		resp, err := group.v3.Respond(udp.Payload, group.v3Agent.ProcessPDU)
		if err != nil || resp == nil {
			if err != nil && h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= DebugLevelInfo {
				_, _ = fmt.Fprintf(os.Stdout,
					"SNMP: v3 respond declined for %s sn=%d err=%v\n", device.Name, pkt.SerialNumber, err)
			}

			continue
		}

		h.stack.stats.mu.Lock()
		h.stack.stats.SNMPQueries++
		h.stack.stats.mu.Unlock()

		h.sendResponse(pkt, ip, udp, device, resp)
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
	if len(device.MACAddress) == snmpMACAddrLen {
		return device.MACAddress
	}

	return pkt.GetDestMAC()
}
