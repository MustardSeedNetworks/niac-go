package protocols

import (
	"fmt"
	"net"
	"os"

	"github.com/google/gopacket/layers"
	"github.com/gosnmp/gosnmp"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
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
		if h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= 2 {
			_, _ = fmt.Fprintf(os.Stdout, "SNMP: decode failed for %s sn=%d err=%v\n", ip.DstIP, pkt.SerialNumber, err)
		}

		return
	}

	for _, device := range devices {
		group := h.stack.getSNMPAgents(device)
		if group == nil {
			continue
		}

		if !snmpAccessAllowed(device, ip.SrcIP) {
			continue
		}

		agent := group.Get(request.Community)
		if agent == nil {
			agent = group.Get("public")
		}

		if agent == nil {
			continue
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
				_, _ = fmt.Fprintf(os.Stdout,
					"SNMP: marshal response failed for device %s sn=%d err=%v\n",
					device.Name,
					pkt.SerialNumber,
					err,
				)
			}

			continue
		}

		h.stack.stats.mu.Lock()
		h.stack.stats.SNMPQueries++
		h.stack.stats.mu.Unlock()

		srcIP := ip.DstIP.To4()

		dstIP := ip.SrcIP.To4()

		if srcIP == nil || dstIP == nil {
			continue
		}

		srcMAC := h.sourceMAC(device, pkt)

		dstMAC := pkt.GetSourceMAC()

		if len(dstMAC) == 0 || len(srcMAC) == 0 {
			continue
		}

		err = h.stack.udpHandler.SendUDP(
			srcIP,
			dstIP,
			uint16(udp.DstPort),
			uint16(udp.SrcPort),
			payload,
			[]byte(srcMAC),
			[]byte(dstMAC),
		)
		if err != nil && h.stack.GetProtocolDebugLevel(logging.ProtocolSNMP) >= 1 {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"SNMP: failed to emit response for device %s sn=%d err=%v\n",
				device.Name,
				pkt.SerialNumber,
				err,
			)
		}
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
	if len(device.MACAddress) == 6 {
		return device.MACAddress
	}

	return pkt.GetDestMAC()
}
