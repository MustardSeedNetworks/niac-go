package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
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

// stpOriginationTick is how often origination checks for due BPDUs; hello
// times are whole seconds.
const stpOriginationTick = time.Second

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

	// Defaults for a device that authors no value of its own.
	bridgePriority uint16
	helloTime      uint16
	maxAge         uint16
	forwardDelay   uint16

	running  bool
	stopChan chan struct{}
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
	}
}

// Start begins originating Configuration BPDUs from every STP-enabled device,
// each at its own hello time. Safe to call again after Stop.
func (h *STPHandler) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return
	}
	h.stopChan = make(chan struct{})
	h.running = true
	stop := h.stopChan

	go func() {
		due := h.sendDueBPDUs(time.Now(), nil)
		ticker := time.NewTicker(stpOriginationTick)
		defer ticker.Stop()
		for {
			select {
			case now := <-ticker.C:
				due = h.sendDueBPDUs(now, due)
			case <-stop:
				return
			}
		}
	}()
}

// Stop halts BPDU origination. Safe to call multiple times.
func (h *STPHandler) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return
	}
	close(h.stopChan)
	h.running = false
}

// sendDueBPDUs sends a BPDU from each STP-enabled device that is due by now and
// returns when each is next due. The next time is the last one plus the hello
// time, not now plus it, so tick jitter cannot stretch a 2 s hello to 3 s. A
// device that cannot advertise at the client is skipped here rather than
// refused at egress, which would count every hello as a fabric drop.
func (h *STPHandler) sendDueBPDUs(now time.Time, due map[*config.Device]time.Time) map[*config.Device]time.Time {
	h.stack.reloadMu.RLock()
	defer h.stack.reloadMu.RUnlock()

	next := make(map[*config.Device]time.Time, len(due))
	for _, device := range h.stack.AllDevices() {
		if !stpEnabled(device) || (h.stack.fabric != nil && !h.stack.fabric.advertisesAtClient(device)) {
			continue
		}
		at, scheduled := due[device]
		if scheduled && now.Before(at) {
			next[device] = at
			continue
		}
		if err := h.SendConfigBPDU(device); err != nil && h.debugLevel >= DebugLevelInfo {
			logging.Debugf("STP: BPDU from %s not sent: %v", device.Name, err)
		}
		hello := time.Duration(h.getSTPParams(device).helloTime) * time.Second
		if !scheduled || now.Sub(at) >= hello {
			at = now
		}
		next[device] = at.Add(hello)
	}
	return next
}

// HandlePacket processes an STP/RSTP BPDU packet.
func (h *STPHandler) HandlePacket(pkt *Packet) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check minimum packet size (Ethernet header + LLC + BPDU)
	if len(pkt.Buffer) < stpMinPacketSize {
		if h.debugLevel >= DebugLevelInfo {
			logging.Debugf("STP: Packet too short sn=%d", pkt.SerialNumber)
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
			logging.Debugf("STP: Invalid LLC header sn=%d", pkt.SerialNumber)
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
			logging.Debugf("STP: Invalid protocol ID 0x%04x sn=%d", protocolID, pkt.SerialNumber)
		}

		return
	}

	if h.debugLevel >= DebugLevelVerbose {
		logging.Debugf("STP: Received BPDU version=%d type=0x%02x sn=%d",
			version, bpduType, pkt.SerialNumber)
	}

	switch bpduType {
	case BPDUTypeConfig:
		h.handleConfigBPDU(pkt, offset)
	case BPDUTypeTCN:
		h.handleTCN(pkt)
	default:
		if h.debugLevel >= DebugLevelInfo {
			logging.Debugf("STP: Unknown BPDU type 0x%02x sn=%d", bpduType, pkt.SerialNumber)
		}
	}
}

// handleConfigBPDU processes a Configuration BPDU.
func (h *STPHandler) handleConfigBPDU(pkt *Packet, offset int) {
	data := pkt.Buffer[offset:]

	// Parse Configuration BPDU fields
	if len(data) < stpMinConfigBPDU {
		if h.debugLevel >= DebugLevelInfo {
			logging.Debugf("STP: Config BPDU too short sn=%d", pkt.SerialNumber)
		}

		return
	}

	// Only logged: the simulated bridges' positions come from the authored
	// topology (electSpanningTree), and libpcap hands back the stack's own
	// BPDUs too.
	flags := data[4]
	rootID := binary.BigEndian.Uint64(data[5:13])
	rootPathCost := binary.BigEndian.Uint32(data[13:17])
	bridgeID := binary.BigEndian.Uint64(data[17:25])
	portID := binary.BigEndian.Uint16(data[25:27])

	if h.debugLevel >= DebugLevelInfo {
		tcFlag := (flags & BPDUFlagTopologyChange) != 0
		tcAckFlag := (flags & BPDUFlagTopologyChangeAck) != 0

		logging.Debugf(
			"STP: Config BPDU - Root=0x%016x Cost=%d Bridge=0x%016x Port=%d TC=%v TCAck=%v sn=%d",
			rootID,
			rootPathCost,
			bridgeID,
			portID,
			tcFlag,
			tcAckFlag,
			pkt.SerialNumber,
		)
	}
}

// handleTCN processes a Topology Change Notification BPDU.
func (h *STPHandler) handleTCN(pkt *Packet) {
	if h.debugLevel >= DebugLevelInfo {
		logging.Debugf("STP: Topology Change Notification received sn=%d", pkt.SerialNumber)
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
	h.mu.RLock()
	p := stpParams{
		bridgePriority: h.bridgePriority,
		helloTime:      h.helloTime,
		maxAge:         h.maxAge,
		forwardDelay:   h.forwardDelay,
	}
	h.mu.RUnlock()

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
	buf = append(
		buf,
		byte(helloTimeScaled>>stpBridgeIDShift8),
		safeconv.ByteFromUint16(helloTimeScaled),
	)

	forwardDelayScaled := params.forwardDelay * stpTimerScale
	buf = append(
		buf,
		byte(forwardDelayScaled>>stpBridgeIDShift8),
		safeconv.ByteFromUint16(forwardDelayScaled),
	)

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

	// A device outside the elected tree (STP not enabled) speaks as a root.
	bridgeID := makeBridgeID(params.bridgePriority, device.MACAddress)
	position, elected := h.stack.spanningTree[device]
	if !elected {
		position = stpPosition{root: bridgeID}
	}
	buf = appendBridgeID(buf, position.root)
	buf = binary.BigEndian.AppendUint32(buf, position.cost)
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

	pkt := &Packet{
		Buffer:       buf,
		Length:       len(buf),
		SerialNumber: serialNum,
		Device:       device,
		VLAN:         h.stack.discoveryVLAN(device),
	}
	h.stack.Send(pkt)

	if h.debugLevel >= DebugLevelInfo {
		logging.Debugf("STP: Sent Config BPDU from %s sn=%d", device.Name, serialNum)
	}

	return nil
}

// makeBridgeID creates a bridge ID from priority and MAC address.
func makeBridgeID(priority uint16, mac net.HardwareAddr) uint64 {
	bridgeID := uint64(priority) << stpBridgeIDShift48
	for i := range min(SizeOfMac, len(mac)) {
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
