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

// EDP TLV Types. Values verified against Wireshark's packet-extreme.c EDP
// dissector (edp_type_vals) — the only public reference for this
// Extreme-proprietary protocol, since Extreme's own spec is not published.
const (
	EDPTLVTypeNull    = 0x00 // End marker
	EDPTLVTypeDisplay = 0x01 // Device display string (MIB-II sysName-style)
	EDPTLVTypeInfo    = 0x02 // Info TLV (slot/port/version)
	EDPTLVTypeVlan    = 0x05 // VLAN info TLV
	EDPTLVTypeESRP    = 0x08 // Extreme Standby Router Protocol TLV

	// edpTLVMarker prefixes every TLV (marker + type + length), verified
	// against tcpdump/Wireshark decode — it is not itself a TLV type.
	edpTLVMarker = 0x99
)

// EDP encoding constants. The wire layout below (LLC/SNAP + 16-byte EDP
// header + marker-prefixed TLVs) is taken from Wireshark's packet-extreme.c
// dissector, the only public description of this Extreme-proprietary
// protocol. tcpdump has no EDP or Extreme-OUI decoder at all (verified
// against tcpdump/oui.c and print-llc.c upstream), so it can only confirm
// the 802.3 + LLC/SNAP framing (DSAP/SSAP/Control/OUI/PID at the right
// offsets) — not the EDP layer itself. tshark, which links Wireshark's
// packet-extreme.c, decodes a built frame end to end as "eth:llc:edp" with
// a verified-good checksum and the correct Display string.
const (
	// edpLLCSNAPHeaderSize is the LLC/SNAP header size in bytes: DSAP(1) +
	// SSAP(1) + Control(1) + OUI(3) + Protocol Type(2).
	edpLLCSNAPHeaderSize = 8

	// EDPOrgCode is the Extreme Networks OUI carried in the SNAP header.
	EDPOrgCode = 0x00E02B

	// edpProtocolType is the SNAP protocol type identifying EDP (Wireshark:
	// dissector_add_uint("llc.extreme_pid", 0x00bb, edp_handle)).
	edpProtocolType = 0x00BB

	// edpHeaderSize is the fixed EDP header size in bytes: Version(1) +
	// Reserved(1) + Length(2) + Checksum(2) + Sequence(2) + Machine ID
	// Type(2) + Machine ID MAC(6).
	edpHeaderSize = 16

	edpReservedByte      = 0x00   // reserved/null byte
	edpSeqNumHigh        = 0x00   // sequence number high byte (initial value)
	edpSeqNumLow         = 0x01   // sequence number low byte (initial value)
	edpMachineIDTypeMAC  = 0x0000 // machine ID type: MAC address follows
	edpMachineIDTypeSize = 2      // machine ID type field size in bytes
	edpMaxLen            = 65535  // max uint16 for lengths
	edpTLVHeaderSize     = 4      // TLV header size (Marker 1 + Type 1 + Length 2), length field includes this header
	edpChecksumByteShift = 8      // Bit shift for high byte in checksum
	edpChecksumWordShift = 16     // Bit shift for folding 32-bit to 16-bit
	edpChecksumWordMask  = 0xffff // Mask for 16-bit value in checksum fold
	edpChecksumSize      = 2      // Checksum size in bytes
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

// buildEDPFrame constructs an EDP frame for a device: LLC/SNAP header
// followed by the 16-byte EDP header and marker-prefixed TLVs. Layout
// verified against Wireshark's packet-extreme.c EDP dissector and confirmed
// by decoding a built frame with tcpdump -v.
func (h *EDPHandler) buildEDPFrame(device *config.Device) []byte {
	var payload []byte

	// Version (1 byte)
	payload = append(payload, EDPVersion)

	// Reserved (1 byte)
	payload = append(payload, edpReservedByte)

	// Length (2 bytes) placeholder, filled in once the TLVs are appended below.
	payload = append(payload, edpReservedByte, edpReservedByte)

	// Checksum (2 bytes) placeholder, filled in last, over the whole
	// header+TLVs with this field zeroed.
	payload = append(payload, edpReservedByte, edpReservedByte)

	// Sequence number (2 bytes) - could be incremented, using initial value
	payload = append(payload, edpSeqNumHigh, edpSeqNumLow)

	// Machine ID: 2-byte type (0x0000 = MAC follows) + 6-byte system MAC.
	midType := make([]byte, edpMachineIDTypeSize)
	binary.BigEndian.PutUint16(midType, edpMachineIDTypeMAC)
	payload = append(payload, midType...)
	payload = append(payload, device.MACAddress[:min(len(device.MACAddress), SizeOfMac)]...)

	for len(payload) < edpHeaderSize {
		payload = append(payload, edpReservedByte)
	}

	// Add TLVs
	payload = append(payload, h.buildDisplayTLV(device)...)
	payload = append(payload, h.buildInfoTLV(device)...)

	// Add NULL TLV (end marker)
	payload = append(payload, h.buildNullTLV()...)

	// Length (2 bytes): total size of header + TLVs.
	length := min(len(payload), edpMaxLen)
	binary.BigEndian.PutUint16(payload[2:4], safeconv.Uint16(length))

	// Checksum (2 bytes): standard Internet checksum over the whole
	// header+TLVs with the checksum field itself zeroed (still zero here).
	checksum := h.calculateChecksum(payload)
	binary.BigEndian.PutUint16(payload[4:6], checksum)

	// Prepend the LLC/SNAP header (DSAP/SSAP 0xAA, OUI 00:E0:2B, protocol
	// type 0x00BB) — EDP is SNAP-encapsulated, not carried by a custom
	// EtherType.
	frame := make([]byte, 0, edpLLCSNAPHeaderSize+len(payload))
	frame = append(frame, h.buildLLCSNAPHeader()...)
	frame = append(frame, payload...)

	return frame
}

// buildLLCSNAPHeader builds the LLC/SNAP header for EDP.
func (h *EDPHandler) buildLLCSNAPHeader() []byte {
	header := make([]byte, edpLLCSNAPHeaderSize)

	// LLC header (3 bytes)
	header[0] = 0xAA // DSAP
	header[1] = 0xAA // SSAP
	header[2] = 0x03 // Control (UI)

	// SNAP header (5 bytes): OUI (3 bytes) 00:E0:2B (Extreme Networks)
	header[3] = 0x00
	header[4] = 0xE0
	header[5] = 0x2B

	// Protocol Type (2 bytes): 0x00BB (EDP)
	binary.BigEndian.PutUint16(header[6:8], edpProtocolType)

	return header
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

	// TLV: Marker (1 byte) + Type (1 byte) + Length (2 bytes, includes
	// header) + Value.
	tlvLen := min(edpTLVHeaderSize+len(display), edpMaxLen)
	displayLen := tlvLen - edpTLVHeaderSize

	tlv := make([]byte, tlvLen)
	tlv[0] = edpTLVMarker
	tlv[1] = EDPTLVTypeDisplay
	binary.BigEndian.PutUint16(tlv[2:4], safeconv.Uint16(tlvLen))
	copy(tlv[4:], display[:displayLen])

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
		if address := h.stack.firstStateIPAddress(device); address != nil {
			info += fmt.Sprintf("IP:%s ", address.String())
		}

		// Add device type
		info += fmt.Sprintf("Type:%s ", device.Type)

		// Add NIAC-Go identifier
		info += "NIAC-Go:v1.5.0"
	}

	infoBytes := []byte(info)

	// TLV: Marker (1 byte) + Type (1 byte) + Length (2 bytes, includes
	// header) + Value.
	tlvLen := min(edpTLVHeaderSize+len(infoBytes), edpMaxLen)
	infoLen := tlvLen - edpTLVHeaderSize

	tlv := make([]byte, tlvLen)
	tlv[0] = edpTLVMarker
	tlv[1] = EDPTLVTypeInfo
	binary.BigEndian.PutUint16(tlv[2:4], safeconv.Uint16(tlvLen))
	copy(tlv[4:], infoBytes[:infoLen])

	return tlv
}

// buildNullTLV builds the NULL TLV (end marker).
func (h *EDPHandler) buildNullTLV() []byte {
	// NULL TLV: Marker (1 byte) + Type (1 byte, 0x00) + Length (2 bytes,
	// value 4 — the header size, since Null carries no value).
	return []byte{edpTLVMarker, EDPTLVTypeNull, 0x00, edpTLVHeaderSize}
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

// sendFrame sends an EDP frame (LLC/SNAP header + EDP header + TLVs) inside
// an 802.3 Ethernet frame with a length field, matching real EDP encapsulation.
func (h *EDPHandler) sendFrame(device *config.Device, edpFrame []byte) {
	_ = sendDiscoveryFrame(EDPMulticastMAC, device, edpFrame, h.stack)
}

// edpParsedData holds the parsed data from an EDP packet.
type edpParsedData struct {
	deviceID string
	display  string
	info     string
}

// parseEDPHeader parses the fixed 16-byte EDP header and returns the machine
// ID (system MAC, formatted) and the cursor to the first TLV. Returns empty
// string and -1 if parsing fails.
func parseEDPHeader(payload []byte) (string, int) {
	if len(payload) < edpHeaderSize {
		return "", -1
	}

	dataLength := int(binary.BigEndian.Uint16(payload[2:4]))
	if dataLength < edpHeaderSize || dataLength > len(payload) {
		return "", -1
	}

	machineID := net.HardwareAddr(payload[10:16]).String()

	return machineID, edpHeaderSize
}

// parseEDPTLVs parses marker-prefixed TLVs from the payload starting at
// cursor, up to boundary (the EDP header's Length field).
func parseEDPTLVs(payload []byte, cursor int, boundary int) (string, string) {
	if boundary > len(payload) {
		boundary = len(payload)
	}

	var display, info string
	for cursor < boundary {
		if cursor+edpTLVHeaderSize > boundary || payload[cursor] != edpTLVMarker {
			break
		}

		tlvType := payload[cursor+1]
		tlvLen := int(binary.BigEndian.Uint16(payload[cursor+2 : cursor+4]))

		if tlvLen < edpTLVHeaderSize || cursor+tlvLen > boundary {
			break
		}

		value := payload[cursor+edpTLVHeaderSize : cursor+tlvLen]
		cursor += tlvLen

		switch tlvType {
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

	// EDP is LLC/SNAP encapsulated; strip the 8-byte LLC/SNAP header if present.
	if len(payload) >= edpLLCSNAPHeaderSize && payload[0] == 0xAA && payload[1] == 0xAA {
		payload = payload[edpLLCSNAPHeaderSize:]
	}

	deviceID, cursor := parseEDPHeader(payload)
	if cursor < 0 {
		return
	}

	dataLength := int(binary.BigEndian.Uint16(payload[2:4]))
	display, info := parseEDPTLVs(payload, cursor, dataLength)
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
