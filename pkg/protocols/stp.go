package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
)

// STP constants
const (
	STPMulticastMAC = "01:80:C2:00:00:00"
	STPProtocolID   = 0x0000
	STPVersion      = 0x00
	STPVersionRSTP  = 0x02
	STPVersionMSTP  = 0x03
)

// BPDU types
const (
	BPDUTypeConfig = 0x00
	BPDUTypeTCN    = 0x80 // Topology Change Notification
)

// STP port states
const (
	STPStateDisabled   = 0
	STPStateBlocking   = 1
	STPStateListening  = 2
	STPStateLearning   = 3
	STPStateForwarding = 4
)

// STP port roles (RSTP)
const (
	STPRoleUnknown    = 0
	STPRoleAlternate  = 1
	STPRoleBackup     = 2
	STPRoleRoot       = 3
	STPRoleDesignated = 4
)

// BPDU flags
const (
	BPDUFlagTopologyChange    = 0x01
	BPDUFlagProposal          = 0x02
	BPDUFlagPortRoleShift     = 2 // 2 bits for port role
	BPDUFlagLearning          = 0x10
	BPDUFlagForwarding        = 0x20
	BPDUFlagAgreement         = 0x40
	BPDUFlagTopologyChangeAck = 0x80
)

// Default STP timers (in seconds)
const (
	DefaultHelloTime    = 2
	DefaultMaxAge       = 20
	DefaultForwardDelay = 15
)

// STPHandler handles Spanning Tree Protocol packets
type STPHandler struct {
	stack      *Stack
	debugLevel int

	// Bridge configuration
	bridgeMAC      net.HardwareAddr
	bridgePriority uint16

	// Root bridge info
	rootID       uint64 // Priority + MAC
	rootPathCost uint32

	// Timers
	helloTime    uint16
	maxAge       uint16
	forwardDelay uint16

	lastBPDUTime time.Time
}

// NewSTPHandler creates a new STP handler
func NewSTPHandler(stack *Stack, debugLevel int) *STPHandler {
	return &STPHandler{
		stack:          stack,
		debugLevel:     debugLevel,
		bridgePriority: 32768, // Default priority
		helloTime:      DefaultHelloTime,
		maxAge:         DefaultMaxAge,
		forwardDelay:   DefaultForwardDelay,
		lastBPDUTime:   time.Now(),
	}
}

// HandlePacket processes an STP/RSTP BPDU packet
func (h *STPHandler) HandlePacket(pkt *Packet) {
	// Check minimum packet size (Ethernet header + LLC + BPDU)
	if len(pkt.Buffer) < 38 {
		if h.debugLevel >= 2 {
			fmt.Printf("STP: Packet too short sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	// Skip Ethernet header (14 bytes)
	offset := 14

	// Parse LLC header (3 bytes)
	// DSAP=0x42, SSAP=0x42, Control=0x03
	dsap := pkt.Buffer[offset]
	ssap := pkt.Buffer[offset+1]

	if dsap != 0x42 || ssap != 0x42 {
		if h.debugLevel >= 2 {
			fmt.Printf("STP: Invalid LLC header sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	offset += 3

	// Parse BPDU header
	protocolID := binary.BigEndian.Uint16(pkt.Buffer[offset : offset+2])
	version := pkt.Buffer[offset+2]
	bpduType := pkt.Buffer[offset+3]

	if protocolID != STPProtocolID {
		if h.debugLevel >= 2 {
			fmt.Printf("STP: Invalid protocol ID 0x%04x sn=%d\n", protocolID, pkt.SerialNumber)
		}
		return
	}

	if h.debugLevel >= 3 {
		fmt.Printf("STP: Received BPDU version=%d type=0x%02x sn=%d\n",
			version, bpduType, pkt.SerialNumber)
	}

	h.lastBPDUTime = time.Now()

	switch bpduType {
	case BPDUTypeConfig:
		h.handleConfigBPDU(pkt, offset)
	case BPDUTypeTCN:
		h.handleTCN(pkt)
	default:
		if h.debugLevel >= 2 {
			fmt.Printf("STP: Unknown BPDU type 0x%02x sn=%d\n", bpduType, pkt.SerialNumber)
		}
	}
}

// handleConfigBPDU processes a Configuration BPDU
func (h *STPHandler) handleConfigBPDU(pkt *Packet, offset int) {
	data := pkt.Buffer[offset:]

	// Parse Configuration BPDU fields
	if len(data) < 35 {
		if h.debugLevel >= 2 {
			fmt.Printf("STP: Config BPDU too short sn=%d\n", pkt.SerialNumber)
		}
		return
	}

	flags := data[4]
	rootID := binary.BigEndian.Uint64(data[5:13])
	rootPathCost := binary.BigEndian.Uint32(data[13:17])
	bridgeID := binary.BigEndian.Uint64(data[17:25])
	portID := binary.BigEndian.Uint16(data[25:27])
	messageAge := binary.BigEndian.Uint16(data[27:29])
	maxAge := binary.BigEndian.Uint16(data[29:31])
	helloTime := binary.BigEndian.Uint16(data[31:33])
	forwardDelay := binary.BigEndian.Uint16(data[33:35])

	if h.debugLevel >= 2 {
		tcFlag := (flags & BPDUFlagTopologyChange) != 0
		tcAckFlag := (flags & BPDUFlagTopologyChangeAck) != 0

		fmt.Printf("STP: Config BPDU - Root=0x%016x Cost=%d Bridge=0x%016x Port=%d TC=%v TCAck=%v sn=%d\n",
			rootID, rootPathCost, bridgeID, portID, tcFlag, tcAckFlag, pkt.SerialNumber)
	}

	// Store information for potential response generation
	h.rootID = rootID
	h.rootPathCost = rootPathCost
	h.helloTime = helloTime
	h.maxAge = maxAge
	h.forwardDelay = forwardDelay

	// Update message age tracking
	_ = messageAge // Store if needed for aging
}

// handleTCN processes a Topology Change Notification BPDU
func (h *STPHandler) handleTCN(pkt *Packet) {
	if h.debugLevel >= 2 {
		fmt.Printf("STP: Topology Change Notification received sn=%d\n", pkt.SerialNumber)
	}

	// In a real implementation, this would trigger topology change procedures
	// For simulation, we just log it
}

// SendConfigBPDU sends a Configuration BPDU for a device
func (h *STPHandler) SendConfigBPDU(device *config.Device) error {
	if len(device.MACAddress) == 0 {
		return fmt.Errorf("device has no MAC address")
	}

	// Skip if STP is explicitly disabled for this device
	if device.STPConfig != nil && !device.STPConfig.Enabled {
		return nil
	}

	// Get STP parameters from device config or use defaults
	bridgePriority := h.bridgePriority
	helloTime := h.helloTime
	maxAge := h.maxAge
	forwardDelay := h.forwardDelay

	if device.STPConfig != nil {
		if device.STPConfig.BridgePriority > 0 {
			bridgePriority = device.STPConfig.BridgePriority
		}
		if device.STPConfig.HelloTime > 0 {
			helloTime = device.STPConfig.HelloTime
		}
		if device.STPConfig.MaxAge > 0 {
			maxAge = device.STPConfig.MaxAge
		}
		if device.STPConfig.ForwardDelay > 0 {
			forwardDelay = device.STPConfig.ForwardDelay
		}
	}

	// Parse STP multicast MAC
	dstMAC, err := net.ParseMAC(STPMulticastMAC)
	if err != nil {
		return fmt.Errorf("failed to parse STP multicast MAC: %w", err)
	}

	// Build BPDU packet
	buf := make([]byte, 0, 64)

	// Ethernet header (14 bytes)
	buf = append(buf, dstMAC...)            // Destination MAC
	buf = append(buf, device.MACAddress...) // Source MAC
	buf = append(buf, 0x00, 0x26)           // Length = 38 bytes (LLC + BPDU)

	// LLC header (3 bytes)
	buf = append(buf, 0x42) // DSAP
	buf = append(buf, 0x42) // SSAP
	buf = append(buf, 0x03) // Control

	// BPDU header (4 bytes)
	buf = append(buf, 0x00, 0x00)     // Protocol ID
	buf = append(buf, STPVersion)     // Version
	buf = append(buf, BPDUTypeConfig) // BPDU Type

	// Configuration BPDU fields (31 bytes)
	flags := uint8(0)
	if h.debugLevel >= 3 {
		// Set topology change flag for testing
		flags |= BPDUFlagTopologyChange
	}
	buf = append(buf, flags) // Flags

	// Root ID (8 bytes) = Priority (2) + MAC (6)
	bridgeID := h.makeBridgeID(bridgePriority, device.MACAddress)
	rootID := bridgeID // We are root

	buf = append(buf, byte(rootID>>56), byte(rootID>>48), byte(rootID>>40), byte(rootID>>32),
		byte(rootID>>24), byte(rootID>>16), byte(rootID>>8), byte(rootID))

	// Root Path Cost (4 bytes)
	buf = append(buf, 0x00, 0x00, 0x00, 0x00)

	// Bridge ID (8 bytes)
	buf = append(buf, byte(bridgeID>>56), byte(bridgeID>>48), byte(bridgeID>>40), byte(bridgeID>>32),
		byte(bridgeID>>24), byte(bridgeID>>16), byte(bridgeID>>8), byte(bridgeID))

	// Port ID (2 bytes) = Priority (4 bits) + Port Number (12 bits)
	portID := uint16(0x8001) // Priority 128, Port 1
	buf = append(buf, byte(portID>>8), byte(portID))

	// Message Age (2 bytes) in 1/256ths of a second
	buf = append(buf, 0x00, 0x00)

	// Max Age (2 bytes) in 1/256ths of a second
	maxAgeScaled := maxAge * 256
	buf = append(buf, byte(maxAgeScaled>>8), byte(maxAgeScaled))

	// Hello Time (2 bytes) in 1/256ths of a second
	helloTimeScaled := helloTime * 256
	buf = append(buf, byte(helloTimeScaled>>8), byte(helloTimeScaled))

	// Forward Delay (2 bytes) in 1/256ths of a second
	forwardDelayScaled := forwardDelay * 256
	buf = append(buf, byte(forwardDelayScaled>>8), byte(forwardDelayScaled))

	// Pad to minimum Ethernet frame size if needed
	for len(buf) < 64 {
		buf = append(buf, 0x00)
	}

	// Send packet
	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	pkt := &Packet{
		Buffer:       buf,
		Length:       len(buf),
		SerialNumber: serialNum,
	}

	h.stack.Send(pkt)

	if h.debugLevel >= 2 {
		fmt.Printf("STP: Sent Config BPDU from %s sn=%d\n", device.Name, serialNum)
	}

	return nil
}

// makeBridgeID creates a bridge ID from priority and MAC address
func (h *STPHandler) makeBridgeID(priority uint16, mac net.HardwareAddr) uint64 {
	bridgeID := uint64(priority) << 48
	for i := 0; i < 6 && i < len(mac); i++ {
		bridgeID |= uint64(mac[i]) << uint(40-i*8)
	}
	return bridgeID
}

// SetBridgePriority sets the bridge priority
func (h *STPHandler) SetBridgePriority(priority uint16) {
	h.bridgePriority = priority
}

// GetPortState returns the current STP port state
func (h *STPHandler) GetPortState() int {
	// For simulation, we assume ports are always forwarding
	return STPStateForwarding
}

// SetDebugLevel updates the debug level
func (h *STPHandler) SetDebugLevel(level int) {
	h.debugLevel = level
}
