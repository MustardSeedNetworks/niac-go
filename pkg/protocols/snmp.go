package protocols

import (
	"fmt"
	"net"

	"github.com/google/gopacket/layers"
	"github.com/gosnmp/gosnmp"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/krisarmstrong/niac-go/pkg/snmp"
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

	device, agent := h.selectAgent(devices)
	if agent == nil {
		if h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= 3 {
			fmt.Printf("SNMP: no agent mapped for %s sn=%d\n", ip.DstIP, pkt.SerialNumber)
		}
		return
	}

	request, err := h.decodeRequest(udp.Payload)
	if err != nil {
		if h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= 2 {
			fmt.Printf("SNMP: decode failed for %s sn=%d err=%v\n", ip.DstIP, pkt.SerialNumber, err)
		}
		return
	}

	if request.Community != agent.GetCommunity() {
		if h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= 2 {
			// SECURITY FIX MEDIUM-5: Redact community strings to prevent credential exposure
			fmt.Printf("SNMP: community mismatch [REDACTED] (expected [REDACTED]) for device %s sn=%d\n",
				device.Name, pkt.SerialNumber)
		}
		return
	}

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

	payload, err := response.MarshalMsg()
	if err != nil {
		if h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= 1 {
			fmt.Printf("SNMP: marshal response failed for device %s sn=%d err=%v\n", device.Name, pkt.SerialNumber, err)
		}
		return
	}

	h.stack.stats.mu.Lock()
	h.stack.stats.SNMPQueries++
	h.stack.stats.mu.Unlock()

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

	err = h.stack.udpHandler.SendUDP(srcIP, dstIP, uint16(udp.DstPort), uint16(udp.SrcPort), payload, []byte(srcMAC), []byte(dstMAC))
	if err != nil && h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= 1 {
		fmt.Printf("SNMP: failed to emit response for device %s sn=%d err=%v\n", device.Name, pkt.SerialNumber, err)
	}
}

func (h *SNMPHandler) selectAgent(devices []*config.Device) (*config.Device, *snmp.Agent) {
	for _, dev := range devices {
		if agent := h.stack.getSNMPAgent(dev); agent != nil {
			return dev, agent
		}
	}
	return nil, nil
}

func (h *SNMPHandler) decodeRequest(payload []byte) (*gosnmp.SnmpPacket, error) {
	decoder := gosnmp.GoSNMP{
		Transport: "udp",
		Version:   gosnmp.Version2c,
		Community: "public",
		MaxOids:   gosnmp.MaxOids,
	}
	return decoder.SnmpDecodePacket(payload)
}

func (h *SNMPHandler) sourceMAC(device *config.Device, pkt *Packet) net.HardwareAddr {
	if len(device.MACAddress) == 6 {
		return device.MACAddress
	}
	return pkt.GetDestMAC()
}
