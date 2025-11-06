package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// LLDP protocol constants
const (
	// LLDP multicast destination MAC address (01:80:c2:00:00:0e)
	LLDPMulticastMAC = "\x01\x80\xc2\x00\x00\x0e"

	// LLDP advertisement interval (default 30 seconds per IEEE 802.1AB)
	LLDPAdvertiseInterval = 30 * time.Second

	// LLDP TTL (Time To Live) - typically 4x advertisement interval
	LLDPTTL = 120 // seconds
)

// LLDP TLV Types (per IEEE 802.1AB)
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
	// 9-126 reserved for future standardization
	LLDPTLVTypeOrganizationSpecific = 127
)

// LLDP Chassis ID Subtypes
const (
	LLDPChassisIDSubtypeChassisComponent = 1
	LLDPChassisIDSubtypeInterfaceAlias   = 2
	LLDPChassisIDSubtypePortComponent    = 3
	LLDPChassisIDSubtypeMACAddress       = 4
	LLDPChassisIDSubtypeNetworkAddress   = 5
	LLDPChassisIDSubtypeInterfaceName    = 6
	LLDPChassisIDSubtypeLocal            = 7
)

// LLDP Port ID Subtypes
const (
	LLDPPortIDSubtypeInterfaceAlias = 1
	LLDPPortIDSubtypePortComponent  = 2
	LLDPPortIDSubtypeMACAddress     = 3
	LLDPPortIDSubtypeNetworkAddress = 4
	LLDPPortIDSubtypeInterfaceName  = 5
	LLDPPortIDSubtypeAgentCircuitID = 6
	LLDPPortIDSubtypeLocal          = 7
)

// LLDP System Capabilities
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

// LLDPHandler handles LLDP advertisements
type LLDPHandler struct {
	stack           *Stack
	stopChan        chan struct{}
	advertiseTicker *time.Ticker
}

// NewLLDPHandler creates a new LLDP handler
func NewLLDPHandler(stack *Stack) *LLDPHandler {
	return &LLDPHandler{
		stack:    stack,
		stopChan: make(chan struct{}),
	}
}

// Start begins periodic LLDP advertisements
func (h *LLDPHandler) Start() {
	debugLevel := h.stack.GetDebugLevel()

	if debugLevel >= 1 {
		fmt.Printf("LLDP: Starting periodic advertisements (interval: %v)\n", LLDPAdvertiseInterval)
	}

	h.advertiseTicker = time.NewTicker(LLDPAdvertiseInterval)

	go func() {
		// Send initial advertisement immediately
		h.sendAdvertisements()

		for {
			select {
			case <-h.advertiseTicker.C:
				h.sendAdvertisements()
			case <-h.stopChan:
				h.advertiseTicker.Stop()
				return
			}
		}
	}()
}

// Stop halts LLDP advertisements
func (h *LLDPHandler) Stop() {
	close(h.stopChan)
}

// sendAdvertisements sends LLDP advertisements for all devices
func (h *LLDPHandler) sendAdvertisements() {
	debugLevel := h.stack.GetDebugLevel()

	devices := h.stack.GetDevices().GetAll()
	for _, device := range devices {
		if len(device.MACAddress) == 0 {
			continue
		}

		// Skip if LLDP is explicitly disabled for this device
		if device.LLDPConfig != nil && !device.LLDPConfig.Enabled {
			continue
		}

		// Build and send LLDP frame
		frame := h.buildLLDPFrame(device)
		if frame != nil {
			err := h.sendFrame(device, frame)
			if err != nil && debugLevel >= 2 {
				fmt.Printf("LLDP: Error sending advertisement for %s: %v\n", device.Name, err)
			} else if debugLevel >= 3 {
				fmt.Printf("LLDP: Sent advertisement for %s (%d bytes)\n", device.Name, len(frame))
			}
		}
	}
}

// buildLLDPFrame constructs an LLDP frame for a device
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

// buildChassisIDTLV builds the Chassis ID TLV
func (h *LLDPHandler) buildChassisIDTLV(device *config.Device) []byte {
	// Determine chassis ID type and value from config or use default
	subtype := byte(LLDPChassisIDSubtypeMACAddress)
	var chassisID []byte

	if device.LLDPConfig != nil && device.LLDPConfig.ChassisIDType != "" {
		switch device.LLDPConfig.ChassisIDType {
		case "mac":
			subtype = byte(LLDPChassisIDSubtypeMACAddress)
			chassisID = device.MACAddress
		case "local":
			subtype = byte(LLDPChassisIDSubtypeLocal)
			chassisID = []byte(device.Name)
		case "network_address":
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
	length := 1 + len(chassisID) // subtype + chassis ID

	tlv := make([]byte, 2+length)
	tlv[0] = byte(LLDPTLVTypeChassisID<<1) | byte((length>>8)&0x01)
	tlv[1] = byte(length & 0xff)
	tlv[2] = subtype
	copy(tlv[3:], chassisID)

	return tlv
}

// buildPortIDTLV builds the Port ID TLV
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

	length := 1 + len(portID) // subtype + port ID

	tlv := make([]byte, 2+length)
	tlv[0] = byte(LLDPTLVTypePortID<<1) | byte((length>>8)&0x01)
	tlv[1] = byte(length & 0xff)
	tlv[2] = subtype
	copy(tlv[3:], portID)

	return tlv
}

// buildTTLTLV builds the TTL TLV
func (h *LLDPHandler) buildTTLTLV(device *config.Device) []byte {
	// Use TTL from config if available, otherwise use default
	ttl := uint16(LLDPTTL)
	if device.LLDPConfig != nil && device.LLDPConfig.TTL > 0 {
		ttl = uint16(device.LLDPConfig.TTL)
	}

	length := 2 // TTL is 2 bytes

	tlv := make([]byte, 2+length)
	tlv[0] = byte(LLDPTLVTypeTTL << 1)
	tlv[1] = byte(length)
	binary.BigEndian.PutUint16(tlv[2:4], ttl)

	return tlv
}

// buildPortDescriptionTLV builds the Port Description TLV
func (h *LLDPHandler) buildPortDescriptionTLV(device *config.Device) []byte {
	// Use port description from config if available, otherwise generate default
	var description []byte
	if device.LLDPConfig != nil && device.LLDPConfig.PortDescription != "" {
		description = []byte(device.LLDPConfig.PortDescription)
	} else {
		description = []byte(fmt.Sprintf("%s interface", device.Type))
	}

	length := len(description)
	if length == 0 {
		return nil
	}

	tlv := make([]byte, 2+length)
	tlv[0] = byte(LLDPTLVTypePortDescription<<1) | byte((length>>8)&0x01)
	tlv[1] = byte(length & 0xff)
	copy(tlv[2:], description)

	return tlv
}

// buildSystemNameTLV builds the System Name TLV
func (h *LLDPHandler) buildSystemNameTLV(device *config.Device) []byte {
	name := []byte(device.Name)

	length := len(name)
	if length == 0 {
		return nil
	}

	tlv := make([]byte, 2+length)
	tlv[0] = byte(LLDPTLVTypeSystemName<<1) | byte((length>>8)&0x01)
	tlv[1] = byte(length & 0xff)
	copy(tlv[2:], name)

	return tlv
}

// buildSystemDescriptionTLV builds the System Description TLV
func (h *LLDPHandler) buildSystemDescriptionTLV(device *config.Device) []byte {
	// Use system description from config if available, otherwise generate default
	var description []byte
	if device.LLDPConfig != nil && device.LLDPConfig.SystemDescription != "" {
		description = []byte(device.LLDPConfig.SystemDescription)
	} else {
		description = []byte(fmt.Sprintf("NIAC-Go simulated %s device", device.Type))
	}

	length := len(description)
	if length == 0 {
		return nil
	}

	tlv := make([]byte, 2+length)
	tlv[0] = byte(LLDPTLVTypeSystemDescription<<1) | byte((length>>8)&0x01)
	tlv[1] = byte(length & 0xff)
	copy(tlv[2:], description)

	return tlv
}

// buildSystemCapabilitiesTLV builds the System Capabilities TLV
func (h *LLDPHandler) buildSystemCapabilitiesTLV(device *config.Device) []byte {
	// Determine capabilities based on device type
	var capabilities uint16
	var enabled uint16

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

	tlv := make([]byte, 2+length)
	tlv[0] = byte(LLDPTLVTypeSystemCapabilities << 1)
	tlv[1] = byte(length)
	binary.BigEndian.PutUint16(tlv[2:4], capabilities)
	binary.BigEndian.PutUint16(tlv[4:6], enabled)

	return tlv
}

// buildManagementAddressTLV builds the Management Address TLV
func (h *LLDPHandler) buildManagementAddressTLV(device *config.Device) []byte {
	if len(device.IPAddresses) == 0 {
		return nil
	}

	// Use first IP address
	ip := device.IPAddresses[0]

	// Determine address subtype (IPv4 or IPv6)
	var addressSubtype byte
	var addressBytes []byte

	if len(ip) == 4 {
		addressSubtype = 1 // IPv4
		addressBytes = ip
	} else if len(ip) == 16 {
		addressSubtype = 2 // IPv6
		addressBytes = ip
	} else {
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
	interfaceSubtype := byte(2)                  // ifIndex
	interfaceNumber := uint32(1)                 // Interface index
	oidStringLength := byte(0)                   // No OID

	length := 1 + addressStringLength + 1 + 4 + 1 // total TLV value length

	tlv := make([]byte, 2+length)
	tlv[0] = byte(LLDPTLVTypeManagementAddress<<1) | byte((length>>8)&0x01)
	tlv[1] = byte(length & 0xff)

	offset := 2
	tlv[offset] = byte(addressStringLength)
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

// buildEndTLV builds the End TLV
func (h *LLDPHandler) buildEndTLV() []byte {
	return []byte{0x00, 0x00} // Type=0, Length=0
}

// sendFrame sends an LLDP frame
func (h *LLDPHandler) sendFrame(device *config.Device, lldpPayload []byte) error {
	// Build Ethernet header
	dstMAC, _ := net.ParseMAC(LLDPMulticastMAC)

	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr(device.MACAddress),
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
		return fmt.Errorf("error serializing LLDP frame: %v", err)
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

// HandlePacket processes an incoming LLDP packet (for future use)
func (h *LLDPHandler) HandlePacket(pkt *Packet) {
	debugLevel := h.stack.GetDebugLevel()

	if debugLevel >= 2 {
		fmt.Printf("LLDP: Received LLDP frame sn=%d (parsing not yet implemented)\n", pkt.SerialNumber)
	}

	// TODO: Parse incoming LLDP frames and store neighbor information
	// This would be useful for:
	// - Discovering real devices on the network
	// - Responding to LLDP queries
	// - Building network topology
}
