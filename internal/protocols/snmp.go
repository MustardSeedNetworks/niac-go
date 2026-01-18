package protocols

import (
	"fmt"
	"net"
	"os"

	"github.com/google/gopacket/layers"
	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/protocols/snmp"
)

// SNMP protocol constants.
const (
	snmpMACAddrLen = 6 // MAC address length in bytes
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
	responseVars := agent.ProcessPDU(request.PDUType, request.Variables, request.MaxRepetitions)

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

	for _, ip := range device.SNMPConfig.AccessList {
		if ip != nil && ip.Equal(srcIP) {
			return true
		}
	}

	return false
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

func (h *SNMPHandler) sourceMAC(device *config.Device, pkt *Packet) net.HardwareAddr {
	if len(device.MACAddress) == snmpMACAddrLen {
		return device.MACAddress
	}

	return pkt.GetDestMAC()
}
