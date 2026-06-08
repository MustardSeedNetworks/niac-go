package protocols

import (
	"context"
	"encoding/binary"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// FDP protocol constants.
const (
	// FDPMulticastMAC is the FDP multicast destination MAC address (01:E0:52:CC:CC:CC).
	FDPMulticastMAC = "01:E0:52:CC:CC:CC"

	// FDPAdvertiseInterval is the default FDP advertisement interval.
	FDPAdvertiseInterval = 60 * time.Second

	// FDPHoldtime is the FDP holdtime in seconds.
	FDPHoldtime = 180

	// FDPVersion is the FDP protocol version.
	FDPVersion = 1
)

// FDP TLV Types.
const (
	FDPTLVTypeDeviceID     = 0x0001
	FDPTLVTypePort         = 0x0002
	FDPTLVTypePlatform     = 0x0003
	FDPTLVTypeCapabilities = 0x0004
	FDPTLVTypeSoftware     = 0x0005
	FDPTLVTypeIPAddress    = 0x0006
)

// FDP Capabilities flags.
const (
	FDPCapRouter = 0x01
	FDPCapSwitch = 0x02
	FDPCapHost   = 0x04
)

// FDP protocol ID for LLC/SNAP header.
const fdpProtocolID = 0x2000

// FDP encoding constants.
const (
	fdpNullByte          = 0x00   // null byte for checksum placeholder
	fdpTLVHeaderSize     = 4      // TLV header size (Type 2 + Length 2)
	fdpMaxLen            = 65535  // max uint16 for lengths
	fdpLLCSNAPHeaderSize = 8      // LLC/SNAP header size in bytes
	fdpCapabilitiesLen   = 8      // Capabilities TLV length (Type + Length + Value)
	fdpCapValueSize      = 4      // Capabilities value size (uint32)
	fdpChecksumByteShift = 8      // Bit shift for high byte in checksum
	fdpChecksumWordShift = 16     // Bit shift for folding 32-bit to 16-bit
	fdpChecksumWordMask  = 0xffff // Mask for 16-bit value in checksum fold
)

// Device type string constant for FDP specific types.
const deviceTypeHost = "host"

// FDPHandler handles FDP advertisements.
type FDPHandler struct {
	stack           *Stack
	mu              sync.Mutex
	stopChan        chan struct{}
	advertiseTicker *time.Ticker
	running         bool
}

// NewFDPHandler creates a new FDP handler.
func NewFDPHandler(stack *Stack) *FDPHandler {
	return &FDPHandler{
		stack:    stack,
		stopChan: make(chan struct{}),
	}
}

// Start begins periodic FDP advertisements. Safe to call again after Stop.
func (h *FDPHandler) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return
	}

	h.stopChan = make(chan struct{})
	h.advertiseTicker = time.NewTicker(FDPAdvertiseInterval)
	h.running = true

	stopChan := h.stopChan
	ticker := h.advertiseTicker

	if h.stack.GetDebugLevel() >= 1 {
		logging.ProtocolLogf(
			context.Background(),
			"FDP",
			logging.LevelInfo,
			"Starting periodic advertisements (interval: %v)",
			FDPAdvertiseInterval,
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

// Stop halts FDP advertisements. Safe to call multiple times.
func (h *FDPHandler) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return
	}

	close(h.stopChan)
	h.running = false
}

// sendAdvertisements sends FDP advertisements for all devices.
func (h *FDPHandler) sendAdvertisements() {
	debugLevel := h.stack.GetDebugLevel()

	devices := h.stack.GetDevices().GetAll()
	for _, device := range devices {
		if len(device.MACAddress) == 0 {
			continue
		}

		// FDP is Foundry/Brocade-proprietary. Like EDP, real-world
		// gear from any other vendor doesn't speak it, so we require
		// explicit opt-in (FDPConfig != nil && Enabled) to avoid
		// every simulated device spamming FDP advertisements regardless
		// of vendor. Closes the "every device emits FDP" bug observed
		// via the Neighbors page in v0.67.x.
		if device.FDPConfig == nil || !device.FDPConfig.Enabled {
			continue
		}

		// Build and send FDP frame
		frame := h.buildFDPFrame(device)
		if frame != nil {
			err := h.sendFrame(device, frame)
			if err != nil && debugLevel >= DebugLevelInfo {
				logging.ProtocolLogf(
					context.Background(),
					"FDP",
					logging.LevelError,
					"Error sending advertisement for %s: %v",
					device.Name,
					err,
				)
			} else if debugLevel >= DebugLevelVerbose {
				logging.ProtocolLogf(
					context.Background(),
					"FDP",
					logging.LevelInfo,
					"Sent advertisement for %s (%d bytes)",
					device.Name,
					len(frame),
				)
			}
		}
	}
}

// buildFDPFrame constructs an FDP frame for a device.
func (h *FDPHandler) buildFDPFrame(device *config.Device) []byte {
	var payload []byte

	// Use holdtime from config if available, otherwise use default
	holdtime := byte(FDPHoldtime)
	if device.FDPConfig != nil && device.FDPConfig.Holdtime > 0 {
		holdtime = safeconv.Byte(device.FDPConfig.Holdtime)
	}

	// FDP header: Version (1 byte) + TTL/Holdtime (1 byte) + Checksum (2 bytes)
	payload = append(payload, FDPVersion)
	payload = append(payload, holdtime)
	payload = append(payload, fdpNullByte, fdpNullByte) // Checksum placeholder

	// Add TLVs
	payload = append(payload, h.buildDeviceIDTLV(device)...)
	payload = append(payload, h.buildPortTLV(device)...)
	payload = append(payload, h.buildPlatformTLV(device)...)
	payload = append(payload, h.buildCapabilitiesTLV(device)...)
	payload = append(payload, h.buildSoftwareTLV(device)...)

	if len(device.IPAddresses) > 0 {
		payload = append(payload, h.buildIPAddressTLV(device)...)
	}

	// Calculate checksum
	checksum := h.calculateChecksum(payload)
	binary.BigEndian.PutUint16(payload[2:4], checksum)

	// Build LLC/SNAP header
	llcSnap := h.buildLLCSNAPHeader()

	// Combine LLC/SNAP + FDP payload
	llcSnap = append(llcSnap, payload...)

	return llcSnap
}

// buildLLCSNAPHeader builds the LLC/SNAP header for FDP.
func (h *FDPHandler) buildLLCSNAPHeader() []byte {
	header := make([]byte, fdpLLCSNAPHeaderSize)

	// LLC header (3 bytes)
	header[0] = 0xAA // DSAP
	header[1] = 0xAA // SSAP
	header[2] = 0x03 // Control

	// SNAP header (5 bytes)
	// OUI (3 bytes): 00:E0:52 (Foundry/Brocade)
	header[3] = 0x00
	header[4] = 0xE0
	header[5] = 0x52

	// Protocol ID (2 bytes): FDP protocol (similar to CDP)
	binary.BigEndian.PutUint16(header[6:8], fdpProtocolID)

	return header
}

// buildDeviceIDTLV builds the Device ID TLV.
func (h *FDPHandler) buildDeviceIDTLV(device *config.Device) []byte {
	deviceID := []byte(device.Name)

	length := min(
		// Type (2) + Length (2) + Value
		fdpTLVHeaderSize+len(deviceID), fdpMaxLen)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], FDPTLVTypeDeviceID)
	binary.BigEndian.PutUint16(tlv[2:4], safeconv.Uint16(length))
	copy(tlv[4:], deviceID)

	return tlv
}

// buildPortTLV builds the Port TLV.
func (h *FDPHandler) buildPortTLV(device *config.Device) []byte {
	var portName []byte

	// Use port ID from config if available
	switch {
	case device.FDPConfig != nil && device.FDPConfig.PortID != "":
		portName = []byte(device.FDPConfig.PortID)
	case len(device.Interfaces) > 0 && device.Interfaces[0].Name != "":
		// Try to use first interface name if available
		portName = []byte(device.Interfaces[0].Name)
	default:
		portName = []byte("Port 1")
	}

	length := min(fdpTLVHeaderSize+len(portName), fdpMaxLen)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], FDPTLVTypePort)
	binary.BigEndian.PutUint16(tlv[2:4], safeconv.Uint16(length))
	copy(tlv[4:], portName)

	return tlv
}

// buildPlatformTLV builds the Platform TLV.
func (h *FDPHandler) buildPlatformTLV(device *config.Device) []byte {
	// Use platform from config if available, otherwise generate default
	var platform []byte
	if device.FDPConfig != nil && device.FDPConfig.Platform != "" {
		platform = []byte(device.FDPConfig.Platform)
	} else {
		platform = []byte("NIAC-Go Simulated " + device.Type)
	}

	length := min(fdpTLVHeaderSize+len(platform), fdpMaxLen)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], FDPTLVTypePlatform)
	binary.BigEndian.PutUint16(tlv[2:4], safeconv.Uint16(length))
	copy(tlv[4:], platform)

	return tlv
}

// buildCapabilitiesTLV builds the Capabilities TLV.
func (h *FDPHandler) buildCapabilitiesTLV(device *config.Device) []byte {
	// Determine capabilities based on device type
	var capabilities uint32

	switch device.Type {
	case deviceTypeRouter:
		capabilities = FDPCapRouter | FDPCapSwitch
	case deviceTypeSwitch:
		capabilities = FDPCapSwitch
	default:
		capabilities = FDPCapHost
	}

	length := fdpCapabilitiesLen // Type (2) + Length (2) + Capabilities (4)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], FDPTLVTypeCapabilities)
	binary.BigEndian.PutUint16(tlv[2:4], fdpCapabilitiesLen) // fixed capabilities TLV length
	binary.BigEndian.PutUint32(tlv[4:8], capabilities)

	return tlv
}

// buildSoftwareTLV builds the Software TLV.
func (h *FDPHandler) buildSoftwareTLV(device *config.Device) []byte {
	// Use software version from config if available, otherwise use default
	var software []byte
	if device.FDPConfig != nil && device.FDPConfig.SoftwareVersion != "" {
		software = []byte(device.FDPConfig.SoftwareVersion)
	} else {
		software = []byte("NIAC-Go v1.5.0")
	}

	length := min(fdpTLVHeaderSize+len(software), fdpMaxLen)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], FDPTLVTypeSoftware)
	binary.BigEndian.PutUint16(tlv[2:4], safeconv.Uint16(length))
	copy(tlv[4:], software)

	return tlv
}

// buildIPAddressTLV builds the IP Address TLV.
func (h *FDPHandler) buildIPAddressTLV(device *config.Device) []byte {
	if len(device.IPAddresses) == 0 {
		return nil
	}

	// Use first IP address
	ip := device.IPAddresses[0]

	var ipBytes []byte
	if ip.To4() != nil {
		ipBytes = ip.To4()
	} else {
		ipBytes = ip.To16()
	}

	length := min(fdpTLVHeaderSize+len(ipBytes), fdpMaxLen)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], FDPTLVTypeIPAddress)
	binary.BigEndian.PutUint16(tlv[2:4], safeconv.Uint16(length))
	copy(tlv[4:], ipBytes)

	return tlv
}

// calculateChecksum calculates the FDP checksum.
func (h *FDPHandler) calculateChecksum(data []byte) uint16 {
	// Standard Internet checksum
	sum := uint32(0)

	// Sum 16-bit words
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}

	// Handle odd byte
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << fdpChecksumByteShift
	}

	// Fold 32-bit sum to 16 bits
	for sum > fdpChecksumWordMask {
		sum = (sum >> fdpChecksumWordShift) + (sum & fdpChecksumWordMask)
	}

	// Return one's complement
	return ^uint16(sum)
}

// sendFrame sends an FDP frame.
func (h *FDPHandler) sendFrame(device *config.Device, fdpPayload []byte) error {
	return sendDiscoveryFrame(FDPMulticastMAC, device, fdpPayload, h.stack)
}

// fdpTLVData holds parsed TLV data from an FDP frame.
type fdpTLVData struct {
	deviceID string
	portID   string
	platform string
	software string
	mgmtIP   net.IP
	caps     []string
	ttl      int
}

// HandlePacket parses inbound FDP frames and records neighbor metadata.
func (h *FDPHandler) HandlePacket(pkt *Packet) {
	fdpData := h.extractFDPData(pkt)
	if fdpData == nil {
		return
	}

	tlvData := h.parseFDPTLVs(fdpData)

	device := h.stack.selectDiscoveryDevice(ProtocolFDP)
	if device == nil {
		return
	}

	entry := h.buildFDPNeighborRecord(device, tlvData)
	h.logFDPNeighbor(entry, pkt)
	h.stack.recordNeighbor(entry)
}

// extractFDPData extracts the FDP payload from the packet.
func (h *FDPHandler) extractFDPData(pkt *Packet) []byte {
	payload, ok := ethernetPayload(pkt.Buffer)
	if !ok || len(payload) < fdpTLVHeaderSize {
		return nil
	}

	fdpData := payload
	if len(payload) >= fdpLLCSNAPHeaderSize && payload[0] == 0xAA && payload[1] == 0xAA {
		fdpData = payload[fdpLLCSNAPHeaderSize:]
	}

	if len(fdpData) < fdpTLVHeaderSize {
		return nil
	}

	return fdpData
}

// parseFDPTLVs parses all TLVs from FDP data.
func (h *FDPHandler) parseFDPTLVs(fdpData []byte) *fdpTLVData {
	data := &fdpTLVData{
		ttl: int(fdpData[1]),
	}

	cursor := fdpTLVHeaderSize
	limit := len(fdpData)

	for cursor+fdpTLVHeaderSize <= limit {
		tlvType := binary.BigEndian.Uint16(fdpData[cursor : cursor+2])
		length := int(binary.BigEndian.Uint16(fdpData[cursor+2 : cursor+fdpTLVHeaderSize]))

		if length < fdpTLVHeaderSize || cursor+length > limit {
			break
		}

		value := fdpData[cursor+fdpTLVHeaderSize : cursor+length]
		h.processFDPTLV(data, tlvType, value)
		cursor += length
	}

	return data
}

// processFDPTLV processes a single FDP TLV.
func (h *FDPHandler) processFDPTLV(data *fdpTLVData, tlvType uint16, value []byte) {
	switch tlvType {
	case FDPTLVTypeDeviceID:
		data.deviceID = strings.TrimSpace(string(value))
	case FDPTLVTypePort:
		data.portID = strings.TrimSpace(string(value))
	case FDPTLVTypePlatform:
		data.platform = strings.TrimSpace(string(value))
	case FDPTLVTypeCapabilities:
		if len(value) >= fdpCapValueSize {
			capBits := binary.BigEndian.Uint32(value[:fdpCapValueSize])
			data.caps = fdpCapabilitiesToStrings(capBits)
		}
	case FDPTLVTypeSoftware:
		data.software = strings.TrimSpace(string(value))
	case FDPTLVTypeIPAddress:
		if len(value) == net.IPv4len || len(value) == net.IPv6len {
			ip := make(net.IP, len(value))
			copy(ip, value)
			data.mgmtIP = ip
		}
	}
}

// buildFDPNeighborRecord constructs a NeighborRecord from parsed FDP data.
func (h *FDPHandler) buildFDPNeighborRecord(device *config.Device, data *fdpTLVData) NeighborRecord {
	ttlSeconds := data.ttl
	if ttlSeconds <= 0 {
		ttlSeconds = FDPHoldtime
	}

	entry := NeighborRecord{
		Protocol:        ProtocolFDP,
		LocalDevice:     device.Name,
		RemoteDevice:    coalesceStrings(data.deviceID, data.platform),
		RemoteChassisID: coalesceStrings(data.deviceID, data.platform),
		RemotePort:      data.portID,
		Description:     joinNonEmpty(" / ", data.platform, data.software),
		TTL:             time.Duration(ttlSeconds) * time.Second,
		Capabilities:    dedupStrings(data.caps),
	}

	if data.mgmtIP != nil {
		entry.ManagementAddress = data.mgmtIP.String()
	}

	if entry.RemoteDevice == "" {
		entry.RemoteDevice = "unknown-fdp"
	}

	if entry.RemoteChassisID == "" {
		entry.RemoteChassisID = entry.RemoteDevice
	}

	return entry
}

// logFDPNeighbor logs FDP neighbor information if debug is enabled.
func (h *FDPHandler) logFDPNeighbor(entry NeighborRecord, _ *Packet) {
	debugLevel := h.stack.GetDebugLevel()
	if debugLevel >= DebugLevelInfo {
		logging.ProtocolLogf(
			context.Background(), "FDP", logging.LevelInfo,
			"Neighbor %s via %s (local %s)",
			entry.RemoteDevice, entry.RemotePort, entry.LocalDevice,
		)
	}
}

func fdpCapabilitiesToStrings(bits uint32) []string {
	var caps []string
	if bits&FDPCapRouter != 0 {
		caps = append(caps, deviceTypeRouter)
	}

	if bits&FDPCapSwitch != 0 {
		caps = append(caps, deviceTypeSwitch)
	}

	if bits&FDPCapHost != 0 {
		caps = append(caps, deviceTypeHost)
	}

	return caps
}
