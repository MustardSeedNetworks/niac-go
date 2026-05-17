package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

// LLDP protocol constants.
const (
	// LLDPMulticastMAC is the LLDP multicast destination MAC address (01:80:c2:00:00:0e).
	LLDPMulticastMAC = "01:80:c2:00:00:0e"

	// LLDPAdvertiseInterval is the default LLDP advertisement interval per IEEE 802.1AB.
	LLDPAdvertiseInterval = 30 * time.Second

	// LLDPTTL is the LLDP time-to-live in seconds.
	LLDPTTL = 120
)

// LLDP TLV Types (per IEEE 802.1AB).
const (
	LLDPTLVTypeEnd                = 0
	LLDPTLVTypeChassisID          = 1
	LLDPTLVTypePortID             = 2
	LLDPTLVTypeTTL                = 3
	LLDPTLVTypePortDescription    = 4
	LLDPTLVTypeSystemName         = 5
	LLDPTLVTypeSystemDescription  = 6
	LLDPTLVTypeSystemCapabilities = 7
	LLDPTLVTypeManagementAddress  = 8
	// LLDPTLVTypeOrganizationSpecific is reserved for future standardization (9-126).
	LLDPTLVTypeOrganizationSpecific = 127
)

// LLDP Chassis ID Subtypes.
const (
	LLDPChassisIDSubtypeChassisComponent = 1
	LLDPChassisIDSubtypeInterfaceAlias   = 2
	LLDPChassisIDSubtypePortComponent    = 3
	LLDPChassisIDSubtypeMACAddress       = 4
	LLDPChassisIDSubtypeNetworkAddress   = 5
	LLDPChassisIDSubtypeInterfaceName    = 6
	LLDPChassisIDSubtypeLocal            = 7
)

// LLDP Port ID Subtypes.
const (
	LLDPPortIDSubtypeInterfaceAlias = 1
	LLDPPortIDSubtypePortComponent  = 2
	LLDPPortIDSubtypeMACAddress     = 3
	LLDPPortIDSubtypeNetworkAddress = 4
	LLDPPortIDSubtypeInterfaceName  = 5
	LLDPPortIDSubtypeAgentCircuitID = 6
	LLDPPortIDSubtypeLocal          = 7
)

// LLDP System Capabilities.
const (
	LLDPCapOther       = 1 << 0
	LLDPCapRepeater    = 1 << 1
	LLDPCapBridge      = 1 << 2
	LLDPCapWLANAP      = 1 << 3
	LLDPCapRouter      = 1 << 4
	LLDPCapTelephone   = 1 << 5
	LLDPCapDOCSIS      = 1 << 6
	LLDPCapStationOnly = 1 << 7
)

// LLDP encoding constants.
const (
	lldpTLVHeaderSize       = 2     // TLV header size (Type+Length fields)
	lldpTTLFieldSize        = 2     // TTL field size in bytes
	lldpLengthHighBit       = 0x01  // Mask for extracting length high bit from Type|Length byte
	lldpLengthLowMask       = 0xff  // Mask for extracting length low byte
	lldpLengthHighByteShift = 8     // Bit shift to extract high byte of length
	lldpMaxTLVLength        = 511   // Maximum TLV length (9-bit field)
	lldpMaxTTL              = 65535 // Maximum TTL value (uint16 max)
	lldpIfIndexSubtype      = 2     // Interface numbering subtype: ifIndex
	lldpIfIndexSize         = 4     // Interface index size (uint32)
	lldpIPv4AddrLen         = 4     // IPv4 address length in bytes
	lldpIPv6AddrLen         = 16    // IPv6 address length in bytes
	lldpMACAddrLen          = 6     // MAC address length in bytes
)

// LLDPHandler handles LLDP advertisements.
type LLDPHandler struct {
	stack           *Stack
	mu              sync.Mutex
	stopChan        chan struct{}
	advertiseTicker *time.Ticker
	running         bool
}

// NewLLDPHandler creates a new LLDP handler.
// lldpDefaultEnabledForType reports whether LLDP advertisements
// should be emitted by default when LLDPConfig is nil. Switches,
// routers, APs, firewalls, and VoIP phones all run LLDP out of the
// box on real hardware; hosts and servers don't unless explicitly
// configured.
func lldpDefaultEnabledForType(deviceType string) bool {
	switch strings.ToLower(deviceType) {
	case "switch", "router", "access_point", "firewall", "voip_phone":
		return true
	default:
		return false
	}
}

func NewLLDPHandler(stack *Stack) *LLDPHandler {
	return &LLDPHandler{
		stack:    stack,
		stopChan: make(chan struct{}),
	}
}

// Start begins periodic LLDP advertisements. Safe to call again after Stop.
func (h *LLDPHandler) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return
	}

	h.stopChan = make(chan struct{})
	h.advertiseTicker = time.NewTicker(LLDPAdvertiseInterval)
	h.running = true

	stopChan := h.stopChan
	ticker := h.advertiseTicker

	if h.stack.GetDebugLevel() >= 1 {
		_, _ = fmt.Fprintf(os.Stdout, "LLDP: Starting periodic advertisements (interval: %v)\n", LLDPAdvertiseInterval)
	}

	go func() {
		h.sendAdvertisements()

		for {
			select {
			case <-ticker.C:
				h.sendAdvertisements()
			case <-stopChan:
				ticker.Stop()

				return
			}
		}
	}()
}

// Stop halts LLDP advertisements. Safe to call multiple times.
func (h *LLDPHandler) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return
	}

	close(h.stopChan)
	h.running = false
}

// sendAdvertisements sends LLDP advertisements for all devices.
func (h *LLDPHandler) sendAdvertisements() {
	debugLevel := h.stack.GetDebugLevel()

	devices := h.stack.GetDevices().GetAll()
	for _, device := range devices {
		if len(device.MACAddress) == 0 {
			continue
		}

		// LLDP is the IEEE 802.1AB standard — every modern
		// network-infra device (switch / router / AP / firewall /
		// VoIP phone) speaks it. Hosts and servers generally don't
		// run it absent explicit opt-in, so default to the
		// infra-device subset and let everything else require
		// LLDPConfig.Enabled=true to participate.
		//
		// Companion to the CDP/EDP/FDP opt-in tightening: keeps the
		// Neighbors page honest by not announcing LLDP from a generic
		// Linux server that real-world wouldn't speak it.
		if !lldpDefaultEnabledForType(device.Type) {
			if device.LLDPConfig == nil || !device.LLDPConfig.Enabled {
				continue
			}
		} else if device.LLDPConfig != nil && !device.LLDPConfig.Enabled {
			continue
		}

		// Build and send LLDP frame
		frame := h.buildLLDPFrame(device)
		if frame != nil {
			err := h.sendFrame(device, frame)
			if err != nil && debugLevel >= DebugLevelInfo {
				_, _ = fmt.Fprintf(os.Stdout, "LLDP: Error sending advertisement for %s: %v\n", device.Name, err)
			} else if debugLevel >= DebugLevelVerbose {
				_, _ = fmt.Fprintf(os.Stdout, "LLDP: Sent advertisement for %s (%d bytes)\n", device.Name, len(frame))
			}
		}
	}
}

// buildLLDPFrame constructs an LLDP frame for a device.
func (h *LLDPHandler) buildLLDPFrame(device *config.Device) []byte {
	var frame []byte

	// Mandatory TLVs (Chassis ID, Port ID, TTL)
	frame = append(frame, h.buildChassisIDTLV(device)...)
	frame = append(frame, h.buildPortIDTLV(device)...)
	frame = append(frame, h.buildTTLTLV(device)...)

	// Optional TLVs
	frame = append(frame, h.buildPortDescriptionTLV(device)...)
	frame = append(frame, h.buildSystemNameTLV(device)...)
	frame = append(frame, h.buildSystemDescriptionTLV(device)...)
	frame = append(frame, h.buildSystemCapabilitiesTLV(device)...)

	// Management Address TLV (if device has IP address)
	if len(device.IPAddresses) > 0 {
		frame = append(frame, h.buildManagementAddressTLV(device)...)
	}

	// End TLV (mandatory)
	frame = append(frame, h.buildEndTLV()...)

	return frame
}

// buildChassisIDTLV builds the Chassis ID TLV.
func (h *LLDPHandler) buildChassisIDTLV(device *config.Device) []byte {
	// Determine chassis ID type and value from config or use default
	subtype := byte(LLDPChassisIDSubtypeMACAddress)

	var chassisID []byte

	if device.LLDPConfig != nil && device.LLDPConfig.ChassisIDType != "" {
		switch device.LLDPConfig.ChassisIDType {
		case config.ChassisIDTypeMAC:
			subtype = byte(LLDPChassisIDSubtypeMACAddress)
			chassisID = device.MACAddress
		case config.ChassisIDTypeLocal:
			subtype = byte(LLDPChassisIDSubtypeLocal)
			chassisID = []byte(device.Name)
		case config.ChassisIDTypeNetworkAddress:
			subtype = byte(LLDPChassisIDSubtypeNetworkAddress)

			if len(device.IPAddresses) > 0 {
				chassisID = device.IPAddresses[0]
			} else {
				chassisID = device.MACAddress
			}
		default:
			// Fall back to MAC address
			chassisID = device.MACAddress
		}
	} else {
		// Default to MAC address
		chassisID = device.MACAddress
	}

	// TLV: Type(7 bits) | Length(9 bits) | Subtype(1 byte) | Chassis ID
	length := min(1+len(chassisID), lldpMaxTLVLength) // subtype + chassis ID, clamped to 9-bit max

	tlv := make([]byte, lldpTLVHeaderSize+length)
	tlv[0] = byte(LLDPTLVTypeChassisID<<1) | byte((length>>lldpLengthHighByteShift)&lldpLengthHighBit)
	tlv[1] = byte(length & lldpLengthLowMask)
	tlv[2] = subtype
	copy(tlv[3:], chassisID)

	return tlv
}

// buildPortIDTLV builds the Port ID TLV.
func (h *LLDPHandler) buildPortIDTLV(device *config.Device) []byte {
	// Use interface name or device name as port ID
	subtype := byte(LLDPPortIDSubtypeInterfaceName)

	var portID []byte

	// Try to use first interface name if available
	if len(device.Interfaces) > 0 && device.Interfaces[0].Name != "" {
		portID = []byte(device.Interfaces[0].Name)
	} else {
		// Fall back to device name
		portID = []byte(device.Name)
	}

	length := min(1+len(portID), lldpMaxTLVLength) // subtype + port ID, clamped to 9-bit max

	tlv := make([]byte, lldpTLVHeaderSize+length)
	tlv[0] = byte(LLDPTLVTypePortID<<1) | byte((length>>lldpLengthHighByteShift)&lldpLengthHighBit)
	tlv[1] = byte(length & lldpLengthLowMask)
	tlv[2] = subtype
	copy(tlv[3:], portID)

	return tlv
}

// buildTTLTLV builds the TTL TLV.
func (h *LLDPHandler) buildTTLTLV(device *config.Device) []byte {
	// Use TTL from config if available, otherwise use default
	ttl := uint16(LLDPTTL)

	if device.LLDPConfig != nil && device.LLDPConfig.TTL > 0 {
		ttlVal := min(device.LLDPConfig.TTL, lldpMaxTTL)

		ttl = safeconv.Uint16(ttlVal)
	}

	length := lldpTTLFieldSize // TTL is 2 bytes

	tlv := make([]byte, lldpTLVHeaderSize+length)
	tlv[0] = byte(LLDPTLVTypeTTL << 1)
	tlv[1] = byte(length)
	binary.BigEndian.PutUint16(tlv[2:4], ttl)

	return tlv
}

// buildPortDescriptionTLV builds the Port Description TLV.
func (h *LLDPHandler) buildPortDescriptionTLV(device *config.Device) []byte {
	// Use port description from config if available, otherwise generate default
	var description []byte
	if device.LLDPConfig != nil && device.LLDPConfig.PortDescription != "" {
		description = []byte(device.LLDPConfig.PortDescription)
	} else {
		description = []byte(device.Type + " interface")
	}

	length := min(len(description), lldpMaxTLVLength) // clamped to 9-bit max
	if length == 0 {
		return nil
	}

	tlv := make([]byte, lldpTLVHeaderSize+length)
	tlv[0] = byte(LLDPTLVTypePortDescription<<1) | byte((length>>lldpLengthHighByteShift)&lldpLengthHighBit)
	tlv[1] = byte(length & lldpLengthLowMask)
	copy(tlv[2:], description)

	return tlv
}

// buildSystemNameTLV builds the System Name TLV.
func (h *LLDPHandler) buildSystemNameTLV(device *config.Device) []byte {
	name := []byte(device.Name)

	length := min(len(name), lldpMaxTLVLength) // clamped to 9-bit max
	if length == 0 {
		return nil
	}

	tlv := make([]byte, lldpTLVHeaderSize+length)
	tlv[0] = byte(LLDPTLVTypeSystemName<<1) | byte((length>>lldpLengthHighByteShift)&lldpLengthHighBit)
	tlv[1] = byte(length & lldpLengthLowMask)
	copy(tlv[2:], name)

	return tlv
}

// buildSystemDescriptionTLV builds the System Description TLV.
func (h *LLDPHandler) buildSystemDescriptionTLV(device *config.Device) []byte {
	// Use system description from config if available, otherwise generate default
	var description []byte
	if device.LLDPConfig != nil && device.LLDPConfig.SystemDescription != "" {
		description = []byte(device.LLDPConfig.SystemDescription)
	} else {
		description = fmt.Appendf(nil, "NIAC-Go simulated %s device", device.Type)
	}

	length := min(len(description), lldpMaxTLVLength) // clamped to 9-bit max
	if length == 0 {
		return nil
	}

	tlv := make([]byte, lldpTLVHeaderSize+length)
	tlv[0] = byte(LLDPTLVTypeSystemDescription<<1) | byte((length>>lldpLengthHighByteShift)&lldpLengthHighBit)
	tlv[1] = byte(length & lldpLengthLowMask)
	copy(tlv[2:], description)

	return tlv
}

// buildSystemCapabilitiesTLV builds the System Capabilities TLV.
func (h *LLDPHandler) buildSystemCapabilitiesTLV(device *config.Device) []byte {
	// Determine capabilities based on device type
	var (
		capabilities uint16
		enabled      uint16
	)

	switch device.Type {
	case "router":
		capabilities = LLDPCapRouter | LLDPCapBridge
		enabled = LLDPCapRouter
	case "switch":
		capabilities = LLDPCapBridge | LLDPCapRouter
		enabled = LLDPCapBridge
	case "ap", "wireless-ap":
		capabilities = LLDPCapWLANAP | LLDPCapBridge
		enabled = LLDPCapWLANAP
	case "phone", "voip-phone":
		capabilities = LLDPCapTelephone | LLDPCapStationOnly
		enabled = LLDPCapTelephone
	default:
		capabilities = LLDPCapStationOnly
		enabled = LLDPCapStationOnly
	}

	length := 4 // 2 bytes capabilities + 2 bytes enabled

	tlv := make([]byte, lldpTLVHeaderSize+length)
	tlv[0] = byte(LLDPTLVTypeSystemCapabilities << 1)
	tlv[1] = byte(length)
	binary.BigEndian.PutUint16(tlv[2:4], capabilities)
	binary.BigEndian.PutUint16(tlv[4:6], enabled)

	return tlv
}

// buildManagementAddressTLV builds the Management Address TLV.
func (h *LLDPHandler) buildManagementAddressTLV(device *config.Device) []byte {
	if len(device.IPAddresses) == 0 {
		return nil
	}

	// Use first IP address
	ip := device.IPAddresses[0]

	// Determine address subtype (IPv4 or IPv6)
	var (
		addressSubtype byte
		addressBytes   []byte
	)

	switch len(ip) {
	case lldpIPv4AddrLen:
		addressSubtype = 1 // IPv4
		addressBytes = ip
	case lldpIPv6AddrLen:
		addressSubtype = 2 // IPv6
		addressBytes = ip
	default:
		return nil
	}

	// Management Address TLV format:
	// - Address String Length (1 byte)
	// - Address Subtype (1 byte)
	// - Management Address (4 or 16 bytes)
	// - Interface Numbering Subtype (1 byte) - use ifIndex (2)
	// - Interface Number (4 bytes)
	// - OID String Length (1 byte)

	addressStringLength := 1 + len(addressBytes) // subtype + address
	interfaceSubtype := byte(lldpIfIndexSubtype) // ifIndex
	interfaceNumber := uint32(1)                 // Interface index
	oidStringLength := byte(0)                   // No OID

	length := min(
		1+addressStringLength+1+lldpIfIndexSize+1,
		lldpMaxTLVLength,
	) // total TLV value length, clamped to 9-bit max

	tlv := make([]byte, lldpTLVHeaderSize+length)
	tlv[0] = byte(LLDPTLVTypeManagementAddress<<1) | byte((length>>lldpLengthHighByteShift)&lldpLengthHighBit)
	tlv[1] = byte(length & lldpLengthLowMask)

	offset := 2
	tlv[offset] = safeconv.Byte(addressStringLength)
	offset++
	tlv[offset] = addressSubtype
	offset++
	copy(tlv[offset:], addressBytes)
	offset += len(addressBytes)
	tlv[offset] = interfaceSubtype
	offset++
	binary.BigEndian.PutUint32(tlv[offset:offset+4], interfaceNumber)
	offset += 4
	tlv[offset] = oidStringLength

	return tlv
}

// buildEndTLV builds the End TLV.
func (h *LLDPHandler) buildEndTLV() []byte {
	return []byte{0x00, 0x00} // Type=0, Length=0
}

// sendFrame sends an LLDP frame.
func (h *LLDPHandler) sendFrame(device *config.Device, lldpPayload []byte) error {
	// Build Ethernet header
	dstMAC, _ := net.ParseMAC(LLDPMulticastMAC)

	eth := &layers.Ethernet{
		SrcMAC:       device.MACAddress,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetType(EtherTypeLLDP),
	}

	// Serialize
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: false,
	}

	err := gopacket.SerializeLayers(buffer, opts,
		eth,
		gopacket.Payload(lldpPayload),
	)
	if err != nil {
		return fmt.Errorf("error serializing LLDP frame: %w", err)
	}

	// Get serial number
	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	// Create and send packet
	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: serialNum,
		Device:       device,
	}

	h.stack.Send(pkt)

	return nil
}

// HandlePacket processes an incoming LLDP packet (for future use).
func (h *LLDPHandler) HandlePacket(pkt *Packet) {
	debugLevel := h.stack.GetDebugLevel()

	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	lldpLayer := packet.Layer(layers.LayerTypeLinkLayerDiscovery)
	if lldpLayer == nil {
		return
	}

	lldp, ok := lldpLayer.(*layers.LinkLayerDiscovery)
	if !ok {
		return
	}

	device := h.stack.selectDiscoveryDevice(ProtocolLLDP)
	if device == nil {
		return
	}

	entry := NeighborRecord{
		Protocol:        ProtocolLLDP,
		LocalDevice:     device.Name,
		RemoteChassisID: lldpChassisIDToString(lldp.ChassisID),
		RemotePort:      lldpPortIDToString(lldp.PortID),
		TTL:             time.Duration(lldp.TTL) * time.Second,
	}
	if entry.TTL <= 0 {
		entry.TTL = time.Duration(LLDPTTL) * time.Second
	}

	if infoLayer := packet.Layer(layers.LayerTypeLinkLayerDiscoveryInfo); infoLayer != nil {
		if info, infoOk := infoLayer.(*layers.LinkLayerDiscoveryInfo); infoOk {
			entry.RemoteDevice = strings.TrimSpace(info.SysName)
			entry.Description = strings.TrimSpace(info.SysDescription)

			entry.Capabilities = lldpCapabilitiesToStrings(info.SysCapabilities.EnabledCap)
			if len(info.MgmtAddress.Address) > 0 {
				entry.ManagementAddress = net.IP(info.MgmtAddress.Address).String()
			}
		}
	}

	if entry.RemoteDevice == "" {
		entry.RemoteDevice = entry.RemoteChassisID
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"LLDP: Neighbor %s via %s (local %s)\n",
			entry.RemoteDevice,
			entry.RemotePort,
			entry.LocalDevice,
		)
	}

	h.stack.recordNeighbor(entry)
}

func lldpChassisIDToString(id layers.LLDPChassisID) string {
	//exhaustive:ignore
	switch id.Subtype {
	case layers.LLDPChassisIDSubTypeMACAddr:
		if len(id.ID) == lldpMACAddrLen {
			return net.HardwareAddr(id.ID).String()
		}
	default:
	}

	return string(id.ID)
}

func lldpPortIDToString(id layers.LLDPPortID) string {
	//exhaustive:ignore
	switch id.Subtype {
	case layers.LLDPPortIDSubtypeMACAddr:
		if len(id.ID) == lldpMACAddrLen {
			return net.HardwareAddr(id.ID).String()
		}
	default:
	}

	return string(id.ID)
}

func lldpCapabilitiesToStrings(caps layers.LLDPCapabilities) []string {
	var out []string
	if caps.Bridge {
		out = append(out, "bridge")
	}

	if caps.Router {
		out = append(out, "router")
	}

	if caps.WLANAP {
		out = append(out, "wlan-ap")
	}

	if caps.Repeater {
		out = append(out, "repeater")
	}

	if caps.StationOnly {
		out = append(out, "station")
	}

	if caps.Phone {
		out = append(out, "phone")
	}

	if caps.CVLAN {
		out = append(out, "cvlan")
	}

	if caps.SVLAN {
		out = append(out, "svlan")
	}

	return dedupStrings(out)
}
