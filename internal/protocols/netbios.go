package protocols

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// NetBIOS ports.
const (
	NetBIOSNameServicePort     = 137 // UDP
	NetBIOSDatagramServicePort = 138 // UDP
	NetBIOSSessionServicePort  = 139 // TCP
)

// NetBIOS Name Service opcodes.
const (
	NBNSOpQuery        = 0
	NBNSOpRegistration = 5
	NBNSOpRelease      = 6
	NBNSOpWACK         = 7
	NBNSOpRefresh      = 8
)

// NetBIOS Name Service response codes.
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

// NetBIOS name types (suffix byte).
const (
	NBNameWorkstation   = 0x00 // Workstation service
	NBNameMsBrowse      = 0x01 // Master Browser election
	NBNameMessenger     = 0x03 // Messenger service
	NBNameRASServer     = 0x06 // RAS Server
	NBNameDomainMaster  = 0x1B // Domain Master Browser
	NBNameDomainCtrl    = 0x1C // Domain Controller
	NBNameMasterBrowser = 0x1D // Master Browser for subnet
	NBNameBrowser       = 0x1E // Browser election
	NBNameNetDDE        = 0x1F // NetDDE server
	NBNameFileServer    = 0x20 // LANMAN/File server
	NBNameRASClient     = 0x21 // RAS Client
	NBNameNetMonAgent   = 0xBE // Network Monitor agent
	NBNameNetMonUtility = 0xBF // Network Monitor utility
)

// NetBIOS Name Service flags.
const (
	NBNSFlagResponse   = 0x8000
	NBNSFlagAuthAnswer = 0x0400
	NBNSFlagTruncated  = 0x0200
	NBNSFlagRecursion  = 0x0100
	NBNSFlagBroadcast  = 0x0010
)

// NetBIOS encoding constants.
const (
	nbnsMinHeader        = 12     // Minimum NBNS header size
	nbDGMMinHeader       = 10     // Minimum Datagram header size
	nbnsRecordTypeNB     = 0x0020 // NB record type
	nbnsClassIN          = 0x0001 // IN class
	nbnsDefaultTTL       = 300    // Default TTL in seconds
	nbnsDefaultNodeFlags = 0x0000 // Default B-node, unique name
	nbnsRDATALen         = 6      // RDATA length (2 bytes flags + 4 bytes IP)
	nbnsEncodedNameCap   = 34     // Encoded name capacity
	nbnsEncodedNameLen   = 0x20   // Encoded name length (32 bytes)
	nbnsTerminator       = 0x00   // Terminating byte
	nbnsDecodedNameLen   = 16     // Decoded name length
	nbnsNameMaxLen       = 15     // Maximum NetBIOS name length before padding
	nbnsNibbleMask       = 0x0F   // Mask for extracting 4-bit nibble
	nbnsNibbleShift      = 4      // Bit shift for nibble operations
	nbnsOpcodeShift      = 11     // Bit shift for opcode extraction
)

// NetBIOS Datagram message types.
const (
	nbDGMDirectUnique = 0x10 // Direct unique datagram
	nbDGMDirectGroup  = 0x11 // Direct group datagram
	nbDGMBroadcast    = 0x12 // Broadcast datagram
)

// NetBIOSHandler handles NetBIOS Name Service and Datagram Service.
type NetBIOSHandler struct {
	stack      *Stack
	debugLevel int
	mu         sync.Mutex                // protects nameTable
	nameTable  map[string]*config.Device // NetBIOS name -> device mapping
}

// NewNetBIOSHandler creates a new NetBIOS handler.
func NewNetBIOSHandler(stack *Stack, debugLevel int) *NetBIOSHandler {
	return &NetBIOSHandler{
		stack:      stack,
		debugLevel: debugLevel,
		nameTable:  make(map[string]*config.Device),
	}
}

// HandleNameService processes NetBIOS Name Service packets (UDP port 137).
func (h *NetBIOSHandler) HandleNameService(
	pkt *Packet,
	packet gopacket.Packet,
	udp *layers.UDP,
	devices []*config.Device,
) {
	if len(udp.Payload) < nbnsMinHeader {
		if h.debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "NetBIOS NS: Packet too short sn=%d\n", pkt.SerialNumber)
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
	opcode := (flags >> nbnsOpcodeShift) & nbnsNibbleMask

	if h.debugLevel >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "NetBIOS NS: TID=0x%04x Op=%d QD=%d AN=%d Response=%v sn=%d\n",
			transactionID, opcode, qdCount, anCount, isResponse, pkt.SerialNumber)
	}

	if isResponse {
		// Silently accept responses
		return
	}

	// Handle queries
	if opcode == NBNSOpQuery && qdCount > 0 {
		h.handleNameQuery(pkt, packet, transactionID, flags, payload[12:], devices)
	} else if h.debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "NetBIOS NS: Unhandled opcode %d sn=%d\n", opcode, pkt.SerialNumber)
	}
}

// handleNameQuery processes NetBIOS name queries.
func (h *NetBIOSHandler) handleNameQuery(
	pkt *Packet,
	packet gopacket.Packet,
	transactionID uint16,
	_ uint16,
	data []byte,
	devices []*config.Device,
) {
	name, nameType, _ := h.decodeNetBIOSName(data)
	if name == "" {
		if h.debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "NetBIOS NS: Failed to decode name sn=%d\n", pkt.SerialNumber)
		}

		return
	}

	if h.debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "NetBIOS NS: Query for '%s'<%02X> sn=%d\n", name, nameType, pkt.SerialNumber)
	}

	ipv4, eth := extractNameQueryLayers(packet)
	if ipv4 == nil || eth == nil {
		return
	}

	matchedDevice, matchedGroup := findMatchingNetBIOSDevice(devices, name, nameType)
	if matchedDevice == nil {
		if h.debugLevel >= DebugLevelVerbose {
			_, _ = fmt.Fprintf(os.Stdout, "NetBIOS NS: Name '%s' not found sn=%d\n", name, pkt.SerialNumber)
		}

		return
	}

	deviceIPv4 := getFirstIPv4(matchedDevice.IPAddresses)
	if deviceIPv4 == nil {
		if h.debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "NetBIOS NS: Device '%s' has no IPv4 address sn=%d\n", name, pkt.SerialNumber)
		}

		return
	}

	h.sendNameQueryResponse(pkt, transactionID, name, nameType, matchedGroup,
		deviceIPv4, ipv4.SrcIP, matchedDevice.MACAddress, eth.SrcMAC)

	if h.debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "NetBIOS NS: Sent positive response for '%s' -> %s sn=%d\n",
			name, deviceIPv4, pkt.SerialNumber)
	}
}

// extractNameQueryLayers extracts IPv4 and Ethernet layers from a packet.
func extractNameQueryLayers(packet gopacket.Packet) (*layers.IPv4, *layers.Ethernet) {
	ipv4Layer := packet.Layer(layers.LayerTypeIPv4)
	if ipv4Layer == nil {
		return nil, nil
	}

	ipv4, ok := ipv4Layer.(*layers.IPv4)
	if !ok {
		return nil, nil
	}

	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		return nil, nil
	}

	eth, ok := ethLayer.(*layers.Ethernet)
	if !ok {
		return nil, nil
	}

	return ipv4, eth
}

// findMatchingNetBIOSDevice finds a device matching the NetBIOS name query.
func findMatchingNetBIOSDevice(devices []*config.Device, name string, nameType byte) (*config.Device, bool) {
	targetName := strings.ToUpper(strings.TrimSpace(name))

	for _, device := range devices {
		if matched, group := matchNetBIOSName(device, targetName, nameType); matched {
			return device, group
		}
	}

	return nil, false
}

// getFirstIPv4 returns the first IPv4 address from the list.
func getFirstIPv4(ips []net.IP) net.IP {
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip
		}
	}

	return nil
}

// sendNameQueryResponse sends a NetBIOS name query response.
func (h *NetBIOSHandler) sendNameQueryResponse(
	reqPkt *Packet,
	transactionID uint16,
	name string,
	nameType byte,
	isGroup bool,
	deviceIP, dstIP net.IP,
	srcMAC, dstMAC net.HardwareAddr,
) {
	// Build NetBIOS Name Service response
	buf := new(bytes.Buffer)

	// Transaction ID
	_ = binary.Write(buf, binary.BigEndian, transactionID) // error is non-critical for simulation

	// Flags: Response, Authoritative Answer
	flags := uint16(NBNSFlagResponse | NBNSFlagAuthAnswer)
	_ = binary.Write(buf, binary.BigEndian, flags) // error is non-critical for simulation

	// Question count = 0, Answer count = 1
	_ = binary.Write(buf, binary.BigEndian, uint16(0)) // QDCOUNT
	_ = binary.Write(buf, binary.BigEndian, uint16(1)) // ANCOUNT
	_ = binary.Write(buf, binary.BigEndian, uint16(0)) // NSCOUNT
	_ = binary.Write(buf, binary.BigEndian, uint16(0)) // ARCOUNT

	// Encode NetBIOS name in answer
	encodedName := h.encodeNetBIOSName(name, nameType)
	buf.Write(encodedName)

	// Type = NB (0x0020), Class = IN (0x0001)
	_ = binary.Write(buf, binary.BigEndian, uint16(nbnsRecordTypeNB)) // NB record
	_ = binary.Write(buf, binary.BigEndian, uint16(nbnsClassIN))      // IN class

	// Get TTL and node flags from matched device config (default: 300 seconds)
	ttl := uint32(nbnsDefaultTTL)
	nodeFlags := uint16(nbnsDefaultNodeFlags) // Default: B-node, unique name

	if matched := h.stack.GetDevices().GetByMAC(srcMAC); matched != nil && matched.NetBIOSConfig != nil {
		if matched.NetBIOSConfig.TTL > 0 {
			ttl = matched.NetBIOSConfig.TTL
		}

		switch matched.NetBIOSConfig.NodeType {
		case "B":
			nodeFlags = 0x0000
		case "P":
			nodeFlags = 0x2000
		case "M":
			nodeFlags = 0x4000
		case "H":
			nodeFlags = 0x6000
		}
	}

	if isGroup {
		nodeFlags |= 0x8000
	}

	// TTL
	_ = binary.Write(buf, binary.BigEndian, ttl) // error is non-critical for simulation

	// RDATA length = 6 (2 bytes flags + 4 bytes IP)
	_ = binary.Write(buf, binary.BigEndian, uint16(nbnsRDATALen)) // error is non-critical for simulation

	// Name flags
	_ = binary.Write(buf, binary.BigEndian, nodeFlags) // error is non-critical for simulation

	// IP address
	buf.Write(deviceIP.To4())

	// Build and send UDP packet
	_ = h.stack.udpHandler.SendUDP( // error is non-critical for simulation
		deviceIP.To4(),
		dstIP.To4(),
		NetBIOSNameServicePort,
		NetBIOSNameServicePort,
		buf.Bytes(),
		srcMAC,
		dstMAC,
		reqPkt.VLAN,
	)
}

// HandleDatagramService processes NetBIOS Datagram Service packets (UDP port 138).
func (h *NetBIOSHandler) HandleDatagramService(
	pkt *Packet,
	_ gopacket.Packet,
	udp *layers.UDP,
	_ []*config.Device,
) {
	if len(udp.Payload) < nbDGMMinHeader {
		if h.debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "NetBIOS DGM: Packet too short sn=%d\n", pkt.SerialNumber)
		}

		return
	}

	payload := udp.Payload

	// Parse NetBIOS Datagram header
	msgType := payload[0]

	if h.debugLevel >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "NetBIOS DGM: Type=0x%02x sn=%d\n", msgType, pkt.SerialNumber)
	}

	// Types: 0x10=Direct Unique, 0x11=Direct Group, 0x12=Broadcast
	switch msgType {
	case nbDGMDirectUnique, nbDGMDirectGroup, nbDGMBroadcast:
		// Silently accept datagrams (we're simulating passive devices)
		if h.debugLevel >= DebugLevelVerbose {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"NetBIOS DGM: Received datagram type 0x%02x sn=%d\n",
				msgType,
				pkt.SerialNumber,
			)
		}
	default:
		if h.debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "NetBIOS DGM: Unknown message type 0x%02x sn=%d\n", msgType, pkt.SerialNumber)
		}
	}
}

// encodeNetBIOSName encodes a NetBIOS name using first-level encoding
// NetBIOS names are 16 bytes padded with spaces, with the 16th byte being the type.
func (h *NetBIOSHandler) encodeNetBIOSName(name string, nameType byte) []byte {
	// Pad name to 15 characters
	name = strings.ToUpper(name)
	if len(name) > nbnsNameMaxLen {
		name = name[:nbnsNameMaxLen]
	}

	for len(name) < nbnsNameMaxLen {
		name += " "
	}

	// Add type suffix
	name += string([]byte{nameType})

	// First-level encoding: each byte becomes two bytes
	// Each nibble is encoded as 'A' + nibble value
	encoded := make([]byte, 0, nbnsEncodedNameCap)

	// Length byte (32 for encoded 16-byte name)
	encoded = append(encoded, nbnsEncodedNameLen)

	// Encode each byte
	for i := range 16 {
		b := name[i]
		encoded = append(encoded, 'A'+((b>>nbnsNibbleShift)&nbnsNibbleMask))
		encoded = append(encoded, 'A'+(b&nbnsNibbleMask))
	}

	// Terminating zero
	encoded = append(encoded, nbnsTerminator)

	return encoded
}

// decodeNetBIOSName decodes a NetBIOS name from first-level encoding
// Returns: name, nameType, bytesConsumed.
func (h *NetBIOSHandler) decodeNetBIOSName(data []byte) (string, byte, int) {
	if len(data) < nbnsEncodedNameCap {
		return "", 0, 0
	}

	length := int(data[0])
	if length != nbnsEncodedNameLen {
		return "", 0, 0
	}

	// Decode 32 bytes into 16 bytes
	decoded := make([]byte, nbnsDecodedNameLen)

	for i := range 16 {
		high := data[1+i*2] - 'A'
		low := data[2+i*2] - 'A'
		decoded[i] = (high << nbnsNibbleShift) | low
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

func matchNetBIOSName(device *config.Device, targetName string, nameType byte) (bool, bool) {
	if device == nil {
		return false, false
	}

	// Check NetBIOS config if present
	if match, group, found := matchNetBIOSNameFromConfig(device, targetName, nameType); found {
		return match, group
	}

	// Fallback to device and sysName (legacy behavior)
	return matchNetBIOSNameFallback(device, targetName)
}

// matchNetBIOSNameFromConfig checks NetBIOS config for name match.
// Returns (matched, isGroup, found) where found indicates if config was checked.
func matchNetBIOSNameFromConfig(device *config.Device, targetName string, nameType byte) (bool, bool, bool) {
	if device.NetBIOSConfig == nil {
		return false, false, false
	}

	if !device.NetBIOSConfig.Enabled {
		return false, false, true // Found config, but disabled
	}

	// Check explicit NetBIOS names first
	for _, n := range device.NetBIOSConfig.Names {
		if strings.EqualFold(strings.TrimSpace(n.Name), targetName) && n.Suffix == nameType {
			return true, n.Group, true
		}
	}

	// Derive names from services if explicit list not provided
	for _, entry := range deriveNetBIOSNames(device) {
		if strings.EqualFold(strings.TrimSpace(entry.Name), targetName) && entry.Suffix == nameType {
			return true, entry.Group, true
		}
	}

	return false, false, true
}

// matchNetBIOSNameFallback checks device name and sysName (legacy behavior).
func matchNetBIOSNameFallback(device *config.Device, targetName string) (bool, bool) {
	deviceName := strings.ToUpper(strings.TrimSpace(device.Name))
	if deviceName == targetName {
		return true, false
	}

	if device.SNMPConfig.SysName != "" {
		sysName := strings.ToUpper(strings.TrimSpace(device.SNMPConfig.SysName))
		if sysName == targetName {
			return true, false
		}
	}

	return false, false
}

type netbiosNameEntry struct {
	Name   string
	Suffix byte
	Group  bool
}

func deriveNetBIOSNames(device *config.Device) []netbiosNameEntry {
	if device == nil || device.NetBIOSConfig == nil {
		return nil
	}

	cfg := device.NetBIOSConfig

	name := cfg.Name
	if name == "" {
		name = device.Name
	}

	result := make([]netbiosNameEntry, 0)

	for _, svc := range cfg.Services {
		switch strings.ToLower(svc) {
		case "workstation":
			result = append(result, netbiosNameEntry{Name: name, Suffix: NBNameWorkstation})
		case "messenger":
			result = append(result, netbiosNameEntry{Name: name, Suffix: NBNameMessenger})
		case "fileserver":
			result = append(result, netbiosNameEntry{Name: name, Suffix: NBNameFileServer})
		case "domainmaster":
			result = append(result, netbiosNameEntry{Name: name, Suffix: NBNameDomainMaster, Group: true})
		case "domaincontroller", "domainctrl":
			result = append(result, netbiosNameEntry{Name: name, Suffix: NBNameDomainCtrl, Group: true})
		case "masterbrowser":
			result = append(result, netbiosNameEntry{Name: name, Suffix: NBNameMasterBrowser, Group: true})
		case "browser":
			result = append(result, netbiosNameEntry{Name: name, Suffix: NBNameBrowser, Group: true})
		case "msbrowse":
			result = append(result, netbiosNameEntry{Name: "__MSBROWSE__", Suffix: NBNameMsBrowse, Group: true})
		case "rasserver":
			result = append(result, netbiosNameEntry{Name: name, Suffix: NBNameRASServer})
		case "rasclient":
			result = append(result, netbiosNameEntry{Name: name, Suffix: NBNameRASClient})
		case "netdde":
			result = append(result, netbiosNameEntry{Name: name, Suffix: NBNameNetDDE})
		case "netmonagent", "networkmonitoragent":
			result = append(result, netbiosNameEntry{Name: name, Suffix: NBNameNetMonAgent})
		case "netmonutility", "networkmonitorutility":
			result = append(result, netbiosNameEntry{Name: name, Suffix: NBNameNetMonUtility})
		}
	}

	if cfg.MsBrowse {
		result = append(result, netbiosNameEntry{Name: "__MSBROWSE__", Suffix: NBNameMsBrowse, Group: true})
	}

	return result
}

// RegisterName registers a NetBIOS name for a device.
func (h *NetBIOSHandler) RegisterName(name string, device *config.Device) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nameTable[strings.ToUpper(strings.TrimSpace(name))] = device
}

// SetDebugLevel updates the debug level.
func (h *NetBIOSHandler) SetDebugLevel(level int) {
	h.debugLevel = level
}
