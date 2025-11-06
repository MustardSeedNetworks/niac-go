package protocols

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// NetBIOS ports
const (
	NetBIOSNameServicePort     = 137 // UDP
	NetBIOSDatagramServicePort = 138 // UDP
	NetBIOSSessionServicePort  = 139 // TCP
)

// NetBIOS Name Service opcodes
const (
	NBNSOpQuery        = 0
	NBNSOpRegistration = 5
	NBNSOpRelease      = 6
	NBNSOpWACK         = 7
	NBNSOpRefresh      = 8
)

// NetBIOS Name Service response codes
const (
	NBNSRCodeSuccess        = 0
	NBNSRCodeFormatError    = 1
	NBNSRCodeServerFailure  = 2
	NBNSRCodeNameError      = 3
	NBNSRCodeNotImplemented = 4
	NBNSRCodeRefused        = 5
	NBNSRCodeActive         = 6
	NBNSRCodeConflict       = 7
)

// NetBIOS name types (suffix byte)
const (
	NBNameWorkstation   = 0x00
	NBNameMessenger     = 0x03
	NBNameFileServer    = 0x20
	NBNameDomainMaster  = 0x1B
	NBNameMasterBrowser = 0x1D
	NBNameBrowser       = 0x1E
)

// NetBIOS Name Service flags
const (
	NBNSFlagResponse   = 0x8000
	NBNSFlagAuthAnswer = 0x0400
	NBNSFlagTruncated  = 0x0200
	NBNSFlagRecursion  = 0x0100
	NBNSFlagBroadcast  = 0x0010
)

// NetBIOSHandler handles NetBIOS Name Service and Datagram Service
type NetBIOSHandler struct {
	stack      *Stack
	debugLevel int
	nameTable  map[string]*config.Device // NetBIOS name -> device mapping
}

// NewNetBIOSHandler creates a new NetBIOS handler
func NewNetBIOSHandler(stack *Stack, debugLevel int) *NetBIOSHandler {
	return &NetBIOSHandler{
		stack:      stack,
		debugLevel: debugLevel,
		nameTable:  make(map[string]*config.Device),
	}
}

// HandleNameService processes NetBIOS Name Service packets (UDP port 137)
func (h *NetBIOSHandler) HandleNameService(pkt *Packet, packet gopacket.Packet, udp *layers.UDP, devices []*config.Device) {
	if len(udp.Payload) < 12 {
		if h.debugLevel >= 2 {
			fmt.Printf("NetBIOS NS: Packet too short sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	payload := udp.Payload

	// Parse NetBIOS Name Service header
	transactionID := binary.BigEndian.Uint16(payload[0:2])
	flags := binary.BigEndian.Uint16(payload[2:4])
	qdCount := binary.BigEndian.Uint16(payload[4:6])
	anCount := binary.BigEndian.Uint16(payload[6:8])

	isResponse := (flags & NBNSFlagResponse) != 0
	opcode := (flags >> 11) & 0x0F

	if h.debugLevel >= 3 {
		fmt.Printf("NetBIOS NS: TID=0x%04x Op=%d QD=%d AN=%d Response=%v sn=%d\n",
			transactionID, opcode, qdCount, anCount, isResponse, pkt.SerialNumber)
	}

	if isResponse {
		// Silently accept responses
		return
	}

	// Handle queries
	if opcode == NBNSOpQuery && qdCount > 0 {
		h.handleNameQuery(pkt, packet, transactionID, flags, payload[12:], devices)
	} else if h.debugLevel >= 2 {
		fmt.Printf("NetBIOS NS: Unhandled opcode %d sn=%d\n", opcode, pkt.SerialNumber)
	}
}

// handleNameQuery processes NetBIOS name queries
func (h *NetBIOSHandler) handleNameQuery(pkt *Packet, packet gopacket.Packet, transactionID uint16, flags uint16, data []byte, devices []*config.Device) {
	// Decode NetBIOS name from query
	name, nameType, _ := h.decodeNetBIOSName(data)
	if name == "" {
		if h.debugLevel >= 2 {
			fmt.Printf("NetBIOS NS: Failed to decode name sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	if h.debugLevel >= 2 {
		fmt.Printf("NetBIOS NS: Query for '%s'<%02X> sn=%d\n", name, nameType, pkt.SerialNumber)
	}

	// Check if any device matches this NetBIOS name
	ipv4Layer := packet.Layer(layers.LayerTypeIPv4)
	if ipv4Layer == nil {
		return
	}
	ipv4 := ipv4Layer.(*layers.IPv4)

	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		return
	}
	eth := ethLayer.(*layers.Ethernet)

	// Look for matching device
	var matchedDevice *config.Device
	targetName := strings.ToUpper(strings.TrimSpace(name))

	for _, device := range devices {
		deviceName := strings.ToUpper(strings.TrimSpace(device.Name))
		if deviceName == targetName {
			matchedDevice = device
			break
		}
		// Also check sysName in SNMP config
		if device.SNMPConfig.SysName != "" {
			sysName := strings.ToUpper(strings.TrimSpace(device.SNMPConfig.SysName))
			if sysName == targetName {
				matchedDevice = device
				break
			}
		}
	}

	if matchedDevice == nil {
		// Name not found
		if h.debugLevel >= 3 {
			fmt.Printf("NetBIOS NS: Name '%s' not found sn=%d\n", name, pkt.SerialNumber)
		}
		return
	}

	// Get IPv4 address for response
	var deviceIPv4 net.IP
	for _, ip := range matchedDevice.IPAddresses {
		if ip.To4() != nil {
			deviceIPv4 = ip
			break
		}
	}

	if deviceIPv4 == nil {
		if h.debugLevel >= 2 {
			fmt.Printf("NetBIOS NS: Device '%s' has no IPv4 address sn=%d\n", name, pkt.SerialNumber)
		}
		return
	}

	// Send positive name query response
	h.sendNameQueryResponse(pkt, transactionID, name, nameType, deviceIPv4, ipv4.SrcIP, matchedDevice.MACAddress, eth.SrcMAC)

	if h.debugLevel >= 2 {
		fmt.Printf("NetBIOS NS: Sent positive response for '%s' -> %s sn=%d\n", name, deviceIPv4, pkt.SerialNumber)
	}
}

// sendNameQueryResponse sends a NetBIOS name query response
func (h *NetBIOSHandler) sendNameQueryResponse(pkt *Packet, transactionID uint16, name string, nameType byte, deviceIP, dstIP net.IP, srcMAC, dstMAC net.HardwareAddr) {
	// Build NetBIOS Name Service response
	buf := new(bytes.Buffer)

	// Transaction ID
	binary.Write(buf, binary.BigEndian, transactionID)

	// Flags: Response, Authoritative Answer
	flags := uint16(NBNSFlagResponse | NBNSFlagAuthAnswer)
	binary.Write(buf, binary.BigEndian, flags)

	// Question count = 0, Answer count = 1
	binary.Write(buf, binary.BigEndian, uint16(0)) // QDCOUNT
	binary.Write(buf, binary.BigEndian, uint16(1)) // ANCOUNT
	binary.Write(buf, binary.BigEndian, uint16(0)) // NSCOUNT
	binary.Write(buf, binary.BigEndian, uint16(0)) // ARCOUNT

	// Encode NetBIOS name in answer
	encodedName := h.encodeNetBIOSName(name, nameType)
	buf.Write(encodedName)

	// Type = NB (0x0020), Class = IN (0x0001)
	binary.Write(buf, binary.BigEndian, uint16(0x0020)) // NB record
	binary.Write(buf, binary.BigEndian, uint16(0x0001)) // IN class

	// Get TTL from matched device config (default: 300 seconds)
	ttl := uint32(300)
	nodeFlags := uint16(0x0000) // Default: B-node, unique name

	// Find the device to get its NetBIOS config
	for _, device := range h.stack.GetDevices().GetAll() {
		if device.MACAddress.String() == srcMAC.String() {
			if device.NetBIOSConfig != nil {
				if device.NetBIOSConfig.TTL > 0 {
					ttl = device.NetBIOSConfig.TTL
				}
				// Set node type flags based on config
				switch device.NetBIOSConfig.NodeType {
				case "B": // Broadcast
					nodeFlags = 0x0000
				case "P": // Peer-to-peer
					nodeFlags = 0x2000
				case "M": // Mixed
					nodeFlags = 0x4000
				case "H": // Hybrid
					nodeFlags = 0x6000
				}
			}
			break
		}
	}

	// TTL
	binary.Write(buf, binary.BigEndian, ttl)

	// RDATA length = 6 (2 bytes flags + 4 bytes IP)
	binary.Write(buf, binary.BigEndian, uint16(6))

	// Name flags
	binary.Write(buf, binary.BigEndian, nodeFlags)

	// IP address
	buf.Write(deviceIP.To4())

	// Build and send UDP packet
	h.stack.udpHandler.SendUDP(
		deviceIP.To4(),
		dstIP.To4(),
		NetBIOSNameServicePort,
		NetBIOSNameServicePort,
		buf.Bytes(),
		srcMAC,
		dstMAC,
	)
}

// HandleDatagramService processes NetBIOS Datagram Service packets (UDP port 138)
func (h *NetBIOSHandler) HandleDatagramService(pkt *Packet, packet gopacket.Packet, udp *layers.UDP, devices []*config.Device) {
	if len(udp.Payload) < 10 {
		if h.debugLevel >= 2 {
			fmt.Printf("NetBIOS DGM: Packet too short sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	payload := udp.Payload

	// Parse NetBIOS Datagram header
	msgType := payload[0]

	if h.debugLevel >= 3 {
		fmt.Printf("NetBIOS DGM: Type=0x%02x sn=%d\n", msgType, pkt.SerialNumber)
	}

	// Types: 0x10=Direct Unique, 0x11=Direct Group, 0x12=Broadcast
	switch msgType {
	case 0x10, 0x11, 0x12:
		// Silently accept datagrams (we're simulating passive devices)
		if h.debugLevel >= 3 {
			fmt.Printf("NetBIOS DGM: Received datagram type 0x%02x sn=%d\n", msgType, pkt.SerialNumber)
		}
	default:
		if h.debugLevel >= 2 {
			fmt.Printf("NetBIOS DGM: Unknown message type 0x%02x sn=%d\n", msgType, pkt.SerialNumber)
		}
	}
}

// encodeNetBIOSName encodes a NetBIOS name using first-level encoding
// NetBIOS names are 16 bytes padded with spaces, with the 16th byte being the type
func (h *NetBIOSHandler) encodeNetBIOSName(name string, nameType byte) []byte {
	// Pad name to 15 characters
	name = strings.ToUpper(name)
	if len(name) > 15 {
		name = name[:15]
	}
	for len(name) < 15 {
		name += " "
	}

	// Add type suffix
	name += string([]byte{nameType})

	// First-level encoding: each byte becomes two bytes
	// Each nibble is encoded as 'A' + nibble value
	encoded := make([]byte, 0, 34)

	// Length byte (32 for encoded 16-byte name)
	encoded = append(encoded, 0x20)

	// Encode each byte
	for i := 0; i < 16; i++ {
		b := name[i]
		encoded = append(encoded, 'A'+((b>>4)&0x0F))
		encoded = append(encoded, 'A'+(b&0x0F))
	}

	// Terminating zero
	encoded = append(encoded, 0x00)

	return encoded
}

// decodeNetBIOSName decodes a NetBIOS name from first-level encoding
// Returns: name, nameType, bytesConsumed
func (h *NetBIOSHandler) decodeNetBIOSName(data []byte) (string, byte, int) {
	if len(data) < 34 {
		return "", 0, 0
	}

	length := int(data[0])
	if length != 0x20 {
		return "", 0, 0
	}

	// Decode 32 bytes into 16 bytes
	decoded := make([]byte, 16)
	for i := 0; i < 16; i++ {
		high := data[1+i*2] - 'A'
		low := data[2+i*2] - 'A'
		decoded[i] = (high << 4) | low
	}

	// Extract name (first 15 bytes, trimmed) and type (16th byte)
	name := strings.TrimSpace(string(decoded[:15]))
	nameType := decoded[15]

	// Skip past name + length byte + terminator
	offset := 34

	// Skip query type and class if present
	if len(data) >= offset+4 {
		offset += 4
	}

	return name, nameType, offset
}

// RegisterName registers a NetBIOS name for a device
func (h *NetBIOSHandler) RegisterName(name string, device *config.Device) {
	h.nameTable[strings.ToUpper(strings.TrimSpace(name))] = device
}

// SetDebugLevel updates the debug level
func (h *NetBIOSHandler) SetDebugLevel(level int) {
	h.debugLevel = level
}
