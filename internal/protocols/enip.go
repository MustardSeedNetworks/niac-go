package protocols

import (
	"encoding/binary"
	"net"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// PortEtherNetIP is the registered EtherNet/IP encapsulation port, served
// over both TCP and UDP.
const PortEtherNetIP = 44818

// The encapsulation header in front of every EtherNet/IP message (CIP Vol. 2
// §2-3): command, length, session handle, status, sender context and options,
// little-endian. The length counts the data after the header, at most 65511
// bytes.
const (
	enipHeaderLen      = 24
	enipLengthOffset   = 2
	enipStatusOffset   = 8
	enipMaxDataLen     = 65511
	enipCommandNOP     = 0x0000
	enipListIdentity   = 0x0063
	enipUnsupportedCmd = 0x0001
)

// The List Identity reply (CIP Vol. 2 §2-4.2.3): one CIP Identity item
// carrying the encapsulation version, the device's socket address and its
// Identity object.
const (
	enipItemIdentity       = 0x000c
	enipProtocolVersion    = 1
	enipSocketFamilyINET   = 2
	enipSocketZeroLen      = 8  // sin_zero
	enipItemListHeaderLen  = 6  // item count, then the item's type and length
	enipIdentityFixedLen   = 34 // the item from version to state, less the product name's characters
	enipIdentityStatus     = 0
	enipIdentityStateReady = 3 // operational
)

// handleEtherNetIP serves TCP 44818 on a device with EtherNet/IP enabled.
// Every other device keeps the closed-port RST.
func (h *TCPHandler) handleEtherNetIP(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	devices []*config.Device,
) {
	device := h.serviceDevice(devices, ipLayer.DstIP, servesEtherNetIP)
	address := ipLayer.DstIP.To4()
	h.serveStream(pkt, ipLayer, tcp, devices, device, h.enip, func(stream []byte) ([]byte, int, bool) {
		return enipAnswer(stream, device.EtherNetIPConfig, address, true)
	}, "EtherNet/IP")
}

// handleEtherNetIP answers a datagram to UDP 44818 on a device with
// EtherNet/IP enabled and reports whether one owned the destination. A false
// return leaves the datagram to the closed-port path.
func (h *UDPHandler) handleEtherNetIP(
	pkt *Packet,
	ipLayer *layers.IPv4,
	udp *layers.UDP,
	devices []*config.Device,
) bool {
	var device *config.Device
	for _, candidate := range devices {
		if servesEtherNetIP(candidate) && h.stack.deviceOwnsIPv4(candidate, ipLayer.DstIP) {
			device = candidate
			break
		}
	}
	if device == nil {
		return false
	}

	response, _, _ := enipAnswer(udp.Payload, device.EtherNetIPConfig, ipLayer.DstIP.To4(), false)
	if len(response) == 0 {
		return true
	}
	identity, ok := h.stack.replyEthernet(pkt, device)
	if !ok {
		return true
	}
	err := h.SendUDP(
		device,
		ipLayer.DstIP.To4(), ipLayer.SrcIP.To4(),
		PortEtherNetIP, uint16(udp.SrcPort),
		response,
		identity.source, identity.destination, identity.vlan,
	)
	if err != nil && h.stack.GetDebugLevel() >= DebugLevelInfo {
		logging.Debugf("EtherNet/IP: List Identity from %s failed sn=%d err=%v", device.Name, pkt.SerialNumber, err)
	}

	return true
}

func servesEtherNetIP(device *config.Device) bool {
	return device.EtherNetIPConfig != nil && device.EtherNetIPConfig.Enabled
}

// enipAnswer answers the complete messages at the front of stream from the
// device at address. List Identity is answered; NOP never is. Over TCP every
// other command gets the reply with status 0x0001, invalid or unsupported
// command; over UDP it is dropped, as the specification requires of a
// datagram. A length no message can carry means the stream is not
// EtherNet/IP, and nothing after it is answered.
func enipAnswer(stream []byte, server *config.EtherNetIPConfig, address net.IP, overTCP bool) ([]byte, int, bool) {
	var answer []byte
	consumed := 0
	for len(stream)-consumed >= enipHeaderLen {
		request := stream[consumed:]
		length := int(binary.LittleEndian.Uint16(request[enipLengthOffset:]))
		if length > enipMaxDataLen {
			return answer, consumed, true
		}
		end := enipHeaderLen + length
		if len(request) < end {
			break
		}
		consumed += end
		switch command := binary.LittleEndian.Uint16(request); {
		case command == enipListIdentity:
			answer = append(answer, enipListIdentityReply(request[:enipHeaderLen], server, address)...)
		case command != enipCommandNOP && overTCP:
			reply := enipReplyHeader(request[:enipHeaderLen], 0)
			binary.LittleEndian.PutUint32(reply[enipStatusOffset:], enipUnsupportedCmd)
			answer = append(answer, reply...)
		}
	}

	return answer, consumed, false
}

// enipReplyHeader is the header answering request with dataLen bytes of
// data: the command, session handle and sender context are echoed, and status
// and options are zero.
func enipReplyHeader(request []byte, dataLen int) []byte {
	reply := make([]byte, enipHeaderLen, enipHeaderLen+dataLen)
	copy(reply[:8], request[:8])
	copy(reply[12:20], request[12:20])
	binary.LittleEndian.PutUint16(reply[enipLengthOffset:], safeconv.Uint16(dataLen))

	return reply
}

// enipListIdentityReply is the List Identity reply naming server at address.
// The socket address is big-endian, as in a sockaddr_in; the Identity object
// is little-endian like the rest of the message.
func enipListIdentityReply(request []byte, server *config.EtherNetIPConfig, address net.IP) []byte {
	itemLen := enipIdentityFixedLen + len(server.ProductName)
	item := make([]byte, 0, enipItemListHeaderLen+itemLen)
	item = binary.LittleEndian.AppendUint16(item, 1)
	item = binary.LittleEndian.AppendUint16(item, enipItemIdentity)
	item = binary.LittleEndian.AppendUint16(item, safeconv.Uint16(itemLen))
	item = binary.LittleEndian.AppendUint16(item, enipProtocolVersion)
	item = binary.BigEndian.AppendUint16(item, enipSocketFamilyINET)
	item = binary.BigEndian.AppendUint16(item, PortEtherNetIP)
	item = append(item, address.To4()...)
	item = append(item, make([]byte, enipSocketZeroLen)...)
	item = binary.LittleEndian.AppendUint16(item, server.VendorID)
	item = binary.LittleEndian.AppendUint16(item, server.DeviceType)
	item = binary.LittleEndian.AppendUint16(item, server.ProductCode)
	item = append(item, server.RevisionMajor, server.RevisionMinor)
	item = binary.LittleEndian.AppendUint16(item, enipIdentityStatus)
	item = binary.LittleEndian.AppendUint32(item, server.SerialNumber)
	item = append(item, safeconv.Byte(len(server.ProductName)))
	item = append(item, server.ProductName...)
	item = append(item, enipIdentityStateReady)

	return append(enipReplyHeader(request, len(item)), item...)
}
