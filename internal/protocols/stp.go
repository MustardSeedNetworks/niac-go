package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// STP constants.
const (
	STPMulticastMAC = "01:80:C2:00:00:00"
	STPProtocolID   = 0x0000
	STPVersion      = 0x00
	STPVersionRSTP  = 0x02
	STPVersionMSTP  = 0x03
)

// BPDU types.
const (
	BPDUTypeConfig = 0x00
	BPDUTypeTCN    = 0x80 // Topology Change Notification
)

// STP port states.
const (
	STPStateDisabled   = 0
	STPStateBlocking   = 1
	STPStateListening  = 2
	STPStateLearning   = 3
	STPStateForwarding = 4
)

// STP port roles (RSTP).
const (
	STPRoleUnknown    = 0
	STPRoleAlternate  = 1
	STPRoleBackup     = 2
	STPRoleRoot       = 3
	STPRoleDesignated = 4
)

// BPDU flags.
const (
	BPDUFlagTopologyChange    = 0x01
	BPDUFlagProposal          = 0x02
	BPDUFlagPortRoleShift     = 2 // 2 bits for port role
	BPDUFlagLearning          = 0x10
	BPDUFlagForwarding        = 0x20
	BPDUFlagAgreement         = 0x40
	BPDUFlagTopologyChangeAck = 0x80
)

// Default STP timers (in seconds).
const (
	DefaultHelloTime    = 2
	DefaultMaxAge       = 20
	DefaultForwardDelay = 15
)

// STP encoding constants.
const (
	stpMinPacketSize    = 38     // Minimum STP packet size (Ethernet + LLC + BPDU)
	stpMinConfigBPDU    = 35     // Minimum Configuration BPDU size
	stpBPDUBufCap       = 64     // BPDU buffer capacity
	stpLLCDSAP          = 0x42   // LLC DSAP for STP
	stpLLCSSAP          = 0x42   // LLC SSAP for STP
	stpLLCControl       = 0x03   // LLC Control field
	stpDefaultPortID    = 0x8001 // Default port ID (priority 128, port 1)
	stpTimerScale       = 256    // Timer scaling factor (1/256ths of second)
	stpPaddingByte      = 0x00   // Padding byte
	stpBridgeIDShift56  = 56     // Bridge ID bit shift
	stpBridgeIDShift48  = 48     // Bridge ID bit shift
	stpBridgeIDShift40  = 40     // Bridge ID bit shift
	stpBridgeIDShift32  = 32     // Bridge ID bit shift
	stpBridgeIDShift24  = 24     // Bridge ID bit shift
	stpBridgeIDShift16  = 16     // Bridge ID bit shift
	stpBridgeIDShift8   = 8      // Bridge ID bit shift
	stpDefaultBridgePri = 32768  // Default STP bridge priority
	stpMACBytesShift    = 8      // Bit shift multiplier for MAC byte positions
	stpLLCBPDULength    = 0x26   // LLC + BPDU length (38 bytes)
)

// STPHandler handles Spanning Tree Protocol packets.
type STPHandler struct {
	mu    sync.RWMutex
	stack *Stack

	debugLevel int

	// Bridge configuration
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

// NewSTPHandler creates a new STP handler.
func NewSTPHandler(stack *Stack, debugLevel int) *STPHandler {
	return &STPHandler{
		stack:          stack,
		debugLevel:     debugLevel,
		bridgePriority: stpDefaultBridgePri, // Default priority
		helloTime:      DefaultHelloTime,
		maxAge:         DefaultMaxAge,
		forwardDelay:   DefaultForwardDelay,
		lastBPDUTime:   time.Now(),
	}
}

// HandlePacket processes an STP/RSTP BPDU packet.
func (h *STPHandler) HandlePacket(pkt *Packet) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check minimum packet size (Ethernet header + LLC + BPDU)
	if len(pkt.Buffer) < stpMinPacketSize {
		if h.debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "STP: Packet too short sn=%d\n", pkt.SerialNumber)
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
		if h.debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "STP: Invalid LLC header sn=%d\n", pkt.SerialNumber)
		}

		return
	}

	offset += 3

	// Parse BPDU header
	protocolID := binary.BigEndian.Uint16(pkt.Buffer[offset : offset+2])
	version := pkt.Buffer[offset+2]
	bpduType := pkt.Buffer[offset+3]

	if protocolID != STPProtocolID {
		if h.debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "STP: Invalid protocol ID 0x%04x sn=%d\n", protocolID, pkt.SerialNumber)
		}

		return
	}

	if h.debugLevel >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "STP: Received BPDU version=%d type=0x%02x sn=%d\n",
			version, bpduType, pkt.SerialNumber)
	}

	h.lastBPDUTime = time.Now()

	switch bpduType {
	case BPDUTypeConfig:
		h.handleConfigBPDU(pkt, offset)
	case BPDUTypeTCN:
		h.handleTCN(pkt)
	default:
		if h.debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "STP: Unknown BPDU type 0x%02x sn=%d\n", bpduType, pkt.SerialNumber)
		}
	}
}

// handleConfigBPDU processes a Configuration BPDU.
func (h *STPHandler) handleConfigBPDU(pkt *Packet, offset int) {
	data := pkt.Buffer[offset:]

	// Parse Configuration BPDU fields
	if len(data) < stpMinConfigBPDU {
		if h.debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "STP: Config BPDU too short sn=%d\n", pkt.SerialNumber)
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

	if h.debugLevel >= DebugLevelInfo {
		tcFlag := (flags & BPDUFlagTopologyChange) != 0
		tcAckFlag := (flags & BPDUFlagTopologyChangeAck) != 0

		_, _ = fmt.Fprintf(
			os.Stdout,
			"STP: Config BPDU - Root=0x%016x Cost=%d Bridge=0x%016x Port=%d TC=%v TCAck=%v sn=%d\n",
			rootID,
			rootPathCost,
			bridgeID,
			portID,
			tcFlag,
			tcAckFlag,
			pkt.SerialNumber,
		)
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

// handleTCN processes a Topology Change Notification BPDU.
func (h *STPHandler) handleTCN(pkt *Packet) {
	if h.debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "STP: Topology Change Notification received sn=%d\n", pkt.SerialNumber)
	}
	// In a real implementation, this would trigger topology change procedures
	// For simulation, we just log it
}

// stpParams holds the STP configuration parameters for a BPDU.
type stpParams struct {
	bridgePriority uint16
	helloTime      uint16
	maxAge         uint16
	forwardDelay   uint16
}

// getSTPParams extracts STP parameters from device config with defaults from handler.
func (h *STPHandler) getSTPParams(device *config.Device) stpParams {
	p := stpParams{
		bridgePriority: h.bridgePriority,
		helloTime:      h.helloTime,
		maxAge:         h.maxAge,
		forwardDelay:   h.forwardDelay,
	}

	if device.STPConfig == nil {
		return p
	}

	if device.STPConfig.BridgePriority > 0 {
		p.bridgePriority = device.STPConfig.BridgePriority
	}

	if device.STPConfig.HelloTime > 0 {
		p.helloTime = device.STPConfig.HelloTime
	}

	if device.STPConfig.MaxAge > 0 {
		p.maxAge = device.STPConfig.MaxAge
	}

	if device.STPConfig.ForwardDelay > 0 {
		p.forwardDelay = device.STPConfig.ForwardDelay
	}

	return p
}

// appendBridgeID appends a bridge ID (8 bytes) to the buffer.
func appendBridgeID(buf []byte, bridgeID uint64) []byte {
	return append(buf,
		safeconv.ByteFromUint64(bridgeID>>stpBridgeIDShift56),
		safeconv.ByteFromUint64(bridgeID>>stpBridgeIDShift48),
		safeconv.ByteFromUint64(bridgeID>>stpBridgeIDShift40),
		safeconv.ByteFromUint64(bridgeID>>stpBridgeIDShift32),
		safeconv.ByteFromUint64(bridgeID>>stpBridgeIDShift24),
		safeconv.ByteFromUint64(bridgeID>>stpBridgeIDShift16),
		safeconv.ByteFromUint64(bridgeID>>stpBridgeIDShift8),
		safeconv.ByteFromUint64(bridgeID),
	)
}

// buildBPDUHeader builds the Ethernet, LLC, and BPDU headers.
func buildBPDUHeader(dstMAC, srcMAC net.HardwareAddr, flags uint8) []byte {
	buf := make([]byte, 0, stpBPDUBufCap)

	// Ethernet header (14 bytes)
	buf = append(buf, dstMAC...)
	buf = append(buf, srcMAC...)
	buf = append(buf, stpPaddingByte, stpLLCBPDULength)

	// LLC header (3 bytes)
	buf = append(buf, stpLLCDSAP, stpLLCSSAP, stpLLCControl)

	// BPDU header (4 bytes)
	buf = append(buf, stpPaddingByte, stpPaddingByte, STPVersion, BPDUTypeConfig)
	buf = append(buf, flags)

	return buf
}

// appendBPDUTimers appends the timer fields to the BPDU buffer.
func appendBPDUTimers(buf []byte, params stpParams) []byte {
	// Message Age (2 bytes) - always 0 for originating bridge
	buf = append(buf, stpPaddingByte, stpPaddingByte)

	// Max Age, Hello Time, Forward Delay (each 2 bytes, scaled by 256)
	maxAgeScaled := params.maxAge * stpTimerScale
	buf = append(buf, byte(maxAgeScaled>>stpBridgeIDShift8), safeconv.ByteFromUint16(maxAgeScaled))

	helloTimeScaled := params.helloTime * stpTimerScale
	buf = append(buf, byte(helloTimeScaled>>stpBridgeIDShift8), safeconv.ByteFromUint16(helloTimeScaled))

	forwardDelayScaled := params.forwardDelay * stpTimerScale
	buf = append(buf, byte(forwardDelayScaled>>stpBridgeIDShift8), safeconv.ByteFromUint16(forwardDelayScaled))

	return buf
}

// SendConfigBPDU sends a Configuration BPDU for a device.
func (h *STPHandler) SendConfigBPDU(device *config.Device) error {
	if len(device.MACAddress) == 0 {
		return ErrDeviceNoMACAddress
	}

	if device.STPConfig != nil && !device.STPConfig.Enabled {
		return nil
	}

	dstMAC, err := net.ParseMAC(STPMulticastMAC)
	if err != nil {
		return fmt.Errorf("failed to parse STP multicast MAC: %w", err)
	}

	params := h.getSTPParams(device)

	// BPDU flags are driven by simulated topology state, not by debug verbosity.
	flags := uint8(0)

	buf := buildBPDUHeader(dstMAC, device.MACAddress, flags)

	// Root ID and Bridge ID (we are root)
	bridgeID := h.makeBridgeID(params.bridgePriority, device.MACAddress)
	buf = appendBridgeID(buf, bridgeID)

	// Root Path Cost (4 bytes of zero)
	buf = append(buf, stpPaddingByte, stpPaddingByte, stpPaddingByte, stpPaddingByte)

	// Bridge ID
	buf = appendBridgeID(buf, bridgeID)

	// Port ID (2 bytes)
	portID := uint16(stpDefaultPortID)
	buf = append(buf, byte(portID>>stpBridgeIDShift8), safeconv.ByteFromUint16(portID))

	buf = appendBPDUTimers(buf, params)

	// Pad to minimum Ethernet frame size
	for len(buf) < 64 {
		buf = append(buf, stpPaddingByte)
	}

	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	pkt := &Packet{Buffer: buf, Length: len(buf), SerialNumber: serialNum}
	h.stack.Send(pkt)

	if h.debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "STP: Sent Config BPDU from %s sn=%d\n", device.Name, serialNum)
	}

	return nil
}

// makeBridgeID creates a bridge ID from priority and MAC address.
func (h *STPHandler) makeBridgeID(priority uint16, mac net.HardwareAddr) uint64 {
	bridgeID := uint64(priority) << stpBridgeIDShift48
	for i := 0; i < SizeOfMac && i < len(mac); i++ {
		shift := stpBridgeIDShift40 - i*stpMACBytesShift
		if shift >= 0 {
			bridgeID |= uint64(mac[i]) << uint(shift)
		}
	}

	return bridgeID
}

// SetBridgePriority sets the bridge priority.
func (h *STPHandler) SetBridgePriority(priority uint16) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.bridgePriority = priority
}

// GetPortState returns the current STP port state.
func (h *STPHandler) GetPortState() int {
	// For simulation, we assume ports are always forwarding
	return STPStateForwarding
}

// SetDebugLevel updates the debug level.
func (h *STPHandler) SetDebugLevel(level int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.debugLevel = level
}
