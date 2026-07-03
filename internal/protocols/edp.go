package protocols

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// EDP protocol constants.
const (
	// EDPMulticastMAC is the EDP multicast destination MAC address (00:E0:2B:00:00:00).
	EDPMulticastMAC = "00:E0:2B:00:00:00"

	// EDPAdvertiseInterval is the default EDP advertisement interval.
	EDPAdvertiseInterval = 30 * time.Second

	// EDPDefaultTTL is the default neighbor aging in seconds.
	EDPDefaultTTL = 120

	// EDPVersion is the EDP protocol version.
	EDPVersion = 1
)

// EDP TLV Types.
const (
	EDPTLVTypeDisplay = 0x01 // Device display string
	EDPTLVTypeInfo    = 0x02 // Info TLV
	EDPTLVTypeWarning = 0x03 // Warning TLV
	EDPTLVTypeNull    = 0x99 // End marker
)

// EDP encoding constants.
const (
	edpReservedByte      = 0x00   // reserved/null byte
	edpSeqNumHigh        = 0x00   // sequence number high byte (initial value)
	edpSeqNumLow         = 0x01   // sequence number low byte (initial value)
	edpMaxLen            = 65535  // max uint16 for lengths
	edpLengthFieldSize   = 2      // length field size in bytes
	edpTLVHeaderSize     = 3      // TLV header size (Type 1 + Length 2)
	edpEthHeaderSize     = 14     // Ethernet header size
	edpChecksumByteShift = 8      // Bit shift for high byte in checksum
	edpChecksumWordShift = 16     // Bit shift for folding 32-bit to 16-bit
	edpChecksumWordMask  = 0xffff // Mask for 16-bit value in checksum fold
	edpChecksumSize      = 2      // Checksum size in bytes
	edpMinHeaderSize     = 8      // Minimum EDP header size (version, reserved, length, seq, id_length)
)

// EDPHandler handles EDP advertisements.
type EDPHandler struct {
	stack           *Stack
	mu              sync.Mutex
	stopChan        chan struct{}
	advertiseTicker *time.Ticker
	running         bool
}

// NewEDPHandler creates a new EDP handler.
func NewEDPHandler(stack *Stack) *EDPHandler {
	return &EDPHandler{
		stack:    stack,
		stopChan: make(chan struct{}),
	}
}

// Start begins periodic EDP advertisements. Safe to call again after Stop.
func (h *EDPHandler) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return
	}

	h.stopChan = make(chan struct{})
	h.advertiseTicker = time.NewTicker(EDPAdvertiseInterval)
	h.running = true

	stopChan := h.stopChan
	ticker := h.advertiseTicker

	if h.stack.GetDebugLevel() >= 1 {
		logging.ProtocolLogf(
			context.Background(),
			"EDP",
			logging.LevelInfo,
			"Starting periodic advertisements (interval: %v)",
			EDPAdvertiseInterval,
		)
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

// Stop halts EDP advertisements. Safe to call multiple times.
func (h *EDPHandler) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return
	}

	close(h.stopChan)
	h.running = false
}

// sendAdvertisements sends EDP advertisements for all devices.
func (h *EDPHandler) sendAdvertisements() {
	debugLevel := h.stack.GetDebugLevel()

	devices := h.stack.GetDevices().GetAll()
	for _, device := range devices {
		if len(device.MACAddress) == 0 {
			continue
		}

		// EDP is Extreme-proprietary. Most real-world devices don't
		// emit it — only ExtremeWare/EXOS hardware does. We require
		// explicit opt-in (EDPConfig != nil && Enabled) so Cisco /
		// Juniper / generic devices in mixed-vendor configs don't
		// announce themselves on a protocol they'd never actually run.
		// Closes the "every device emits EDP" bug observed via the
		// Neighbors page in v0.67.x.
		if device.EDPConfig == nil || !device.EDPConfig.Enabled {
			continue
		}

		// Build and send EDP frame
		frame := h.buildEDPFrame(device)
		if frame != nil {
			h.sendFrame(device, frame)
			if debugLevel >= DebugLevelVerbose {
				logging.ProtocolLogf(
					context.Background(),
					"EDP",
					logging.LevelInfo,
					"Sent advertisement for %s (%d bytes)",
					device.Name,
					len(frame),
				)
			}
		}
	}
}

// buildEDPFrame constructs an EDP frame for a device.
func (h *EDPHandler) buildEDPFrame(device *config.Device) []byte {
	var payload []byte

	// EDP Header
	// Version (1 byte)
	payload = append(payload, EDPVersion)

	// Reserved (1 byte)
	payload = append(payload, edpReservedByte)

	// Sequence number (2 bytes) - could be incremented, using initial value
	payload = append(payload, edpSeqNumHigh, edpSeqNumLow)

	// ID Length (2 bytes) - length of device ID
	deviceID := []byte(device.Name)

	deviceIDLen := min(len(deviceID), edpMaxLen)

	idLenBytes := make([]byte, edpLengthFieldSize)
	binary.BigEndian.PutUint16(idLenBytes, safeconv.Uint16(deviceIDLen))
	payload = append(payload, idLenBytes...)

	// Device ID
	payload = append(payload, deviceID...)

	// Add TLVs
	payload = append(payload, h.buildDisplayTLV(device)...)
	payload = append(payload, h.buildInfoTLV(device)...)

	// Add NULL TLV (end marker)
	payload = append(payload, h.buildNullTLV()...)

	// Checksum (2 bytes) at the end
	checksum := h.calculateChecksum(payload)
	checksumBytes := make([]byte, edpLengthFieldSize)
	binary.BigEndian.PutUint16(checksumBytes, checksum)
	payload = append(payload, checksumBytes...)

	return payload
}

// buildDisplayTLV builds the Display TLV.
func (h *EDPHandler) buildDisplayTLV(device *config.Device) []byte {
	// Use display string from config if available, otherwise generate default
	var display []byte
	if device.EDPConfig != nil && device.EDPConfig.DisplayString != "" {
		display = []byte(device.EDPConfig.DisplayString)
	} else {
		display = fmt.Appendf(nil, "%s (%s)", device.Name, device.Type)
	}

	// TLV: Type (1 byte) + Length (2 bytes) + Value
	displayLen := min(len(display), edpMaxLen)

	tlv := make([]byte, edpTLVHeaderSize+displayLen)
	tlv[0] = EDPTLVTypeDisplay
	binary.BigEndian.PutUint16(
		tlv[1:3],
		safeconv.Uint16(displayLen),
	)
	copy(tlv[3:], display[:displayLen])

	return tlv
}

// buildInfoTLV builds the Info TLV.
func (h *EDPHandler) buildInfoTLV(device *config.Device) []byte {
	// Use version string from config if available, otherwise generate default
	var info string

	if device.EDPConfig != nil && device.EDPConfig.VersionString != "" {
		info = device.EDPConfig.VersionString
	} else {
		// Info string includes various device information
		// Add MAC address
		info += fmt.Sprintf("MAC:%s ", device.MACAddress.String())

		// Add IP addresses
		if len(device.IPAddresses) > 0 {
			info += fmt.Sprintf("IP:%s ", device.IPAddresses[0].String())
		}

		// Add device type
		info += fmt.Sprintf("Type:%s ", device.Type)

		// Add NIAC-Go identifier
		info += "NIAC-Go:v1.5.0"
	}

	infoBytes := []byte(info)

	infoLen := min(len(infoBytes), edpMaxLen)

	// TLV: Type (1 byte) + Length (2 bytes) + Value
	tlv := make([]byte, edpTLVHeaderSize+infoLen)
	tlv[0] = EDPTLVTypeInfo
	binary.BigEndian.PutUint16(tlv[1:3], safeconv.Uint16(infoLen))
	copy(tlv[3:], infoBytes[:infoLen])

	return tlv
}

// buildNullTLV builds the NULL TLV (end marker).
func (h *EDPHandler) buildNullTLV() []byte {
	// NULL TLV: Type (1 byte) + Length (2 bytes, value 0)
	return []byte{EDPTLVTypeNull, 0x00, 0x00}
}

// calculateChecksum calculates the EDP checksum.
func (h *EDPHandler) calculateChecksum(data []byte) uint16 {
	// Standard Internet checksum
	sum := uint32(0)

	// Sum 16-bit words
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}

	// Handle odd byte
	if len(data)%edpChecksumSize == 1 {
		sum += uint32(data[len(data)-1]) << edpChecksumByteShift
	}

	// Fold 32-bit sum to 16 bits
	for sum > edpChecksumWordMask {
		sum = (sum >> edpChecksumWordShift) + (sum & edpChecksumWordMask)
	}

	// Return one's complement
	return ^uint16(sum)
}

// sendFrame sends an EDP frame.
func (h *EDPHandler) sendFrame(device *config.Device, edpPayload []byte) {
	// Build Ethernet header
	dstMAC, _ := net.ParseMAC(EDPMulticastMAC)

	// Build raw Ethernet frame
	frame := make([]byte, edpEthHeaderSize+len(edpPayload))

	// Destination MAC
	copy(frame[0:6], dstMAC)

	// Source MAC
	copy(frame[6:12], device.MACAddress)

	// EtherType (EDP uses custom EtherType)
	binary.BigEndian.PutUint16(frame[12:14], EtherTypeEDP)

	// Payload
	copy(frame[14:], edpPayload)

	// Get serial number
	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	// Create and send packet
	pkt := &Packet{
		Buffer:       frame,
		Length:       len(frame),
		SerialNumber: serialNum,
		Device:       device,
		VLAN:         device.VLAN, // advertise on the device's access VLAN
	}

	h.stack.Send(pkt)
}

// edpParsedData holds the parsed data from an EDP packet.
type edpParsedData struct {
	deviceID string
	display  string
	info     string
}

// parseEDPHeader parses the EDP header and returns the device ID and cursor position.
// Returns empty string and -1 if parsing fails.
func parseEDPHeader(payload []byte) (string, int) {
	if len(payload) < edpMinHeaderSize {
		return "", -1
	}

	idLen := int(binary.BigEndian.Uint16(payload[4:6]))
	cursor := 6

	if cursor+idLen+edpChecksumSize > len(payload) {
		return "", -1
	}

	deviceID := strings.TrimSpace(string(payload[cursor : cursor+idLen]))
	cursor += idLen

	return deviceID, cursor
}

// parseEDPTLVs parses TLVs from the payload starting at cursor.
func parseEDPTLVs(payload []byte, cursor int) (string, string) {
	checksumBoundary := len(payload) - edpChecksumSize
	if checksumBoundary <= cursor {
		return "", ""
	}

	var display, info string
	for cursor < checksumBoundary {
		if cursor+edpTLVHeaderSize > checksumBoundary {
			break
		}

		tlvt := payload[cursor]
		tlvLen := int(binary.BigEndian.Uint16(payload[cursor+1 : cursor+3]))
		cursor += edpTLVHeaderSize

		if cursor+tlvLen > checksumBoundary {
			break
		}

		value := payload[cursor : cursor+tlvLen]
		cursor += tlvLen

		switch tlvt {
		case EDPTLVTypeDisplay:
			display = strings.TrimSpace(string(value))
		case EDPTLVTypeInfo:
			info = strings.TrimSpace(string(value))
		case EDPTLVTypeNull:
			return display, info
		}
	}

	return display, info
}

// buildEDPNeighborRecord creates a neighbor record from parsed EDP data.
func buildEDPNeighborRecord(deviceName string, parsed edpParsedData, fields map[string]string) NeighborRecord {
	entry := NeighborRecord{
		Protocol:        ProtocolEDP,
		LocalDevice:     deviceName,
		RemoteDevice:    coalesceStrings(parsed.display, parsed.deviceID, fields["MAC"]),
		RemoteChassisID: coalesceStrings(fields["MAC"], parsed.deviceID),
		RemotePort:      strings.TrimSpace(fields["PORT"]),
		Description:     strings.TrimSpace(parsed.info),
		TTL:             time.Duration(EDPDefaultTTL) * time.Second,
	}

	if ip := fields["IP"]; ip != "" {
		entry.ManagementAddress = ip
	}

	if remoteType := strings.ToLower(fields["TYPE"]); remoteType != "" {
		entry.Capabilities = []string{remoteType}
	}

	entry.Capabilities = dedupStrings(entry.Capabilities)

	if entry.RemoteDevice == "" {
		entry.RemoteDevice = entry.RemoteChassisID
	}

	if entry.RemoteChassisID == "" {
		entry.RemoteChassisID = entry.RemoteDevice
	}

	return entry
}

// HandlePacket parses incoming Extreme Discovery Protocol frames and records neighbors.
func (h *EDPHandler) HandlePacket(pkt *Packet) {
	payload, ok := ethernetPayload(pkt.Buffer)
	if !ok {
		return
	}

	deviceID, cursor := parseEDPHeader(payload)
	if cursor < 0 {
		return
	}

	display, info := parseEDPTLVs(payload, cursor)
	parsed := edpParsedData{deviceID: deviceID, display: display, info: info}

	device := h.stack.selectDiscoveryDevice(ProtocolEDP)
	if device == nil {
		return
	}

	fields := parseKeyValueFields(info)
	entry := buildEDPNeighborRecord(device.Name, parsed, fields)

	if h.stack.GetDebugLevel() >= DebugLevelInfo && entry.RemoteDevice != "" {
		logging.ProtocolLogf(context.Background(), "EDP", logging.LevelInfo, "Neighbor %s via %s (local %s)",
			entry.RemoteDevice, entry.RemotePort, entry.LocalDevice)
	}

	h.stack.recordNeighbor(entry)
}
