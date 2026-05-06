package protocols

import (
	"encoding/binary"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

// CDP protocol constants.
const (
	// CDPMulticastMAC is the CDP multicast destination MAC address (01:00:0c:cc:cc:cc).
	CDPMulticastMAC = "01:00:0c:cc:cc:cc"

	// CDPLLCDSAP is the LLC DSAP/SSAP for SNAP encapsulation.
	CDPLLCDSAP = 0xAAAA
	// CDPOrgCode is the Cisco OUI.
	CDPOrgCode = 0x00000C
	// CDPProtocol is the CDP protocol ID.
	CDPProtocol = 0x2000

	// CDPAdvertiseInterval is the default CDP advertisement interval.
	CDPAdvertiseInterval = 60 * time.Second

	// CDPHoldtime is the CDP holdtime in seconds.
	CDPHoldtime = 180

	// CDPVersion is the CDP protocol version.
	CDPVersion = 2
)

// CDP TLV Types.
const (
	CDPTLVTypeDeviceID        = 0x0001
	CDPTLVTypeAddresses       = 0x0002
	CDPTLVTypePortID          = 0x0003
	CDPTLVTypeCapabilities    = 0x0004
	CDPTLVTypeSoftwareVersion = 0x0005
	CDPTLVTypePlatform        = 0x0006
	CDPTLVTypeIPPrefix        = 0x0007
	CDPTLVTypeVTPDomain       = 0x0009
	CDPTLVTypeNativeVLAN      = 0x000A
	CDPTLVTypeDuplex          = 0x000B
	CDPTLVTypePower           = 0x0010
	CDPTLVTypeMTU             = 0x0011
	CDPTLVTypeTrustBitmap     = 0x0012
	CDPTLVTypeUntrustedCOS    = 0x0013
	CDPTLVTypeManagementAddr  = 0x0016
)

// CDP Capabilities flags.
const (
	CDPCapRouter       = 0x01
	CDPCapTransBridge  = 0x02
	CDPCapSourceBridge = 0x04
	CDPCapSwitch       = 0x08
	CDPCapHost         = 0x10
	CDPCapIGMPCapable  = 0x20
	CDPCapRepeater     = 0x40
	CDPCapPhone        = 0x80
	CDPCapRemote       = 0x100
)

// Device type string constants.
const (
	deviceTypeRouter = "router"
	deviceTypeSwitch = "switch"
)

// CDP encoding constants.
const (
	cdpNullByte           = 0x00   // null byte for padding/checksum
	cdpBitShift           = 8      // bit shift for uint16 encoding
	cdpWordShift          = 16     // bit shift for folding 32-bit to 16-bit
	cdpWordMask           = 0xffff // mask for 16-bit value
	cdpCapBytes           = 4      // capability field size in bytes
	cdpMaxUint16          = 65535  // max uint16 value for lengths
	llcSnapHeaderSize     = 8      // LLC/SNAP header size in bytes
	cdpTLVHeaderSize      = 4      // TLV header overhead (Type 2 + Length 2)
	capabilitiesTLVLen    = 8      // fixed capabilities TLV length
	cdpAddressLenFieldLen = 2      // address length field size (2 bytes)
)

// CDPHandler handles CDP advertisements.
type CDPHandler struct {
	stack           *Stack
	stopChan        chan struct{}
	stopOnce        sync.Once
	advertiseTicker *time.Ticker
}

// NewCDPHandler creates a new CDP handler.
func NewCDPHandler(stack *Stack) *CDPHandler {
	return &CDPHandler{
		stack:    stack,
		stopChan: make(chan struct{}),
	}
}

// Start begins periodic CDP advertisements.
func (h *CDPHandler) Start() {
	logger := slog.Default()
	debugLevel := h.stack.GetDebugLevel()

	if debugLevel >= 1 {
		logger.Info("CDP: Starting periodic advertisements", "interval", CDPAdvertiseInterval)
	}

	h.advertiseTicker = time.NewTicker(CDPAdvertiseInterval)

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

// Stop halts CDP advertisements.
func (h *CDPHandler) Stop() {
	h.stopOnce.Do(func() {
		close(h.stopChan)
	})
}

// sendAdvertisements sends CDP advertisements for all devices.
func (h *CDPHandler) sendAdvertisements() {
	logger := slog.Default()
	debugLevel := h.stack.GetDebugLevel()

	devices := h.stack.GetDevices().GetAll()
	for _, device := range devices {
		if len(device.MACAddress) == 0 {
			continue
		}

		// Skip if CDP is explicitly disabled for this device
		if device.CDPConfig != nil && !device.CDPConfig.Enabled {
			continue
		}

		// Build and send CDP frame
		frame := h.buildCDPFrame(device)
		if frame != nil {
			err := h.sendFrame(device, frame)
			if err != nil && debugLevel >= DebugLevelInfo {
				logger.Debug("CDP: Error sending advertisement", "device", device.Name, "error", err)
			} else if debugLevel >= DebugLevelVerbose {
				logger.Debug("CDP: Sent advertisement", "device", device.Name, "bytes", len(frame))
			}
		}
	}
}

// buildCDPFrame constructs a CDP frame for a device.
func (h *CDPHandler) buildCDPFrame(device *config.Device) []byte {
	var payload []byte

	// Use version and holdtime from config if available, otherwise use defaults
	version := byte(CDPVersion)
	holdtime := byte(CDPHoldtime)

	if device.CDPConfig != nil {
		if device.CDPConfig.Version > 0 {
			version = safeconv.Byte(device.CDPConfig.Version)
		}

		if device.CDPConfig.Holdtime > 0 {
			h := device.CDPConfig.Holdtime
			if h > math.MaxUint8 {
				slog.Warn("CDP: holdtime exceeds uint8 max, capping to 255", "device", device.Name, "holdtime", h)
				h = math.MaxUint8
			}
			holdtime = byte(h)
		}
	}

	// CDP header: Version (1 byte) + TTL (1 byte) + Checksum (2 bytes)
	payload = append(payload, version)
	payload = append(payload, holdtime)
	payload = append(payload, cdpNullByte, cdpNullByte) // Checksum placeholder

	// Add TLVs
	payload = append(payload, h.buildDeviceIDTLV(device)...)
	payload = append(payload, h.buildAddressesTLV(device)...)
	payload = append(payload, h.buildPortIDTLV(device)...)
	payload = append(payload, h.buildCapabilitiesTLV(device)...)
	payload = append(payload, h.buildSoftwareVersionTLV(device)...)
	payload = append(payload, h.buildPlatformTLV(device)...)

	// Calculate checksum (standard Internet checksum)
	checksum := h.calculateChecksum(payload)
	binary.BigEndian.PutUint16(payload[2:4], checksum)

	// Build LLC/SNAP header
	llcSnap := h.buildLLCSNAPHeader()

	// Combine LLC/SNAP + CDP payload
	frame := make([]byte, 0, len(llcSnap)+len(payload))
	frame = append(frame, llcSnap...)
	frame = append(frame, payload...)

	return frame
}

// buildLLCSNAPHeader builds the LLC/SNAP header for CDP.
func (h *CDPHandler) buildLLCSNAPHeader() []byte {
	header := make([]byte, llcSnapHeaderSize)

	// LLC header (3 bytes)
	header[0] = 0xAA // DSAP
	header[1] = 0xAA // SSAP
	header[2] = 0x03 // Control

	// SNAP header (5 bytes)
	// OUI (3 bytes): 00:00:0C (Cisco)
	header[3] = 0x00
	header[4] = 0x00
	header[5] = 0x0C

	// Protocol ID (2 bytes): 0x2000 (CDP)
	binary.BigEndian.PutUint16(header[6:8], CDPProtocol)

	return header
}

// buildDeviceIDTLV builds the Device ID TLV.
func (h *CDPHandler) buildDeviceIDTLV(device *config.Device) []byte {
	deviceID := []byte(device.Name)
	length := min(cdpTLVHeaderSize+len(deviceID), cdpMaxUint16) // Type (2) + Length (2) + Value, capped at max uint16

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], CDPTLVTypeDeviceID)
	binary.BigEndian.PutUint16(tlv[2:4], safeconv.Uint16(length))
	copy(tlv[4:], deviceID)

	return tlv
}

// buildAddressesTLV builds the Addresses TLV.
func (h *CDPHandler) buildAddressesTLV(device *config.Device) []byte {
	if len(device.IPAddresses) == 0 {
		return nil
	}

	// For simplicity, include only the first IP address
	ip := device.IPAddresses[0]

	var (
		addrBytes []byte
		protoType byte
	)

	if ip.To4() != nil {
		// IPv4
		protoType = 0xCC // NLPID for IPv4
		addrBytes = ip.To4()
	} else {
		// IPv6
		protoType = 0x8E // NLPID for IPv6
		addrBytes = ip.To16()
	}

	// Address format:
	// Number of addresses (4 bytes)
	// For each address:
	//   Protocol Type (1 byte)
	//   Protocol Length (1 byte)
	//   Protocol (variable)
	//   Address Length (2 bytes)
	//   Address (variable)

	addrLen := 1 + 1 + 1 + cdpAddressLenFieldLen + len(addrBytes)
	length := min(4+4+addrLen, cdpMaxUint16) // Type + Length + NumAddrs + Address, capped at max uint16

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], CDPTLVTypeAddresses)
	binary.BigEndian.PutUint16(tlv[2:4], safeconv.Uint16(length))
	binary.BigEndian.PutUint32(tlv[4:8], 1) // Number of addresses

	offset := 8
	tlv[offset] = protoType
	offset++
	tlv[offset] = 1 // Protocol length
	offset++
	tlv[offset] = protoType // Protocol value
	offset++
	addrBytesLen := min(len(addrBytes), cdpMaxUint16)
	binary.BigEndian.PutUint16(
		tlv[offset:offset+2],
		safeconv.Uint16(addrBytesLen),
	)
	offset += 2
	copy(tlv[offset:], addrBytes)

	return tlv
}

// buildPortIDTLV builds the Port ID TLV.
func (h *CDPHandler) buildPortIDTLV(device *config.Device) []byte {
	var portID []byte

	// Use port ID from config if available
	switch {
	case device.CDPConfig != nil && device.CDPConfig.PortID != "":
		portID = []byte(device.CDPConfig.PortID)
	case len(device.Interfaces) > 0 && device.Interfaces[0].Name != "":
		// Try to use first interface name if available
		portID = []byte(device.Interfaces[0].Name)
	default:
		// Fall back to a generic port name
		portID = []byte("Port 1")
	}

	length := min(cdpTLVHeaderSize+len(portID), cdpMaxUint16)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], CDPTLVTypePortID)
	binary.BigEndian.PutUint16(tlv[2:4], safeconv.Uint16(length))
	copy(tlv[4:], portID)

	return tlv
}

// buildCapabilitiesTLV builds the Capabilities TLV.
func (h *CDPHandler) buildCapabilitiesTLV(device *config.Device) []byte {
	// Determine capabilities based on device type
	var capabilities uint32

	switch device.Type {
	case deviceTypeRouter:
		capabilities = CDPCapRouter | CDPCapIGMPCapable
	case deviceTypeSwitch:
		capabilities = CDPCapSwitch | CDPCapIGMPCapable
	case "ap", "wireless-ap":
		capabilities = CDPCapSwitch | CDPCapIGMPCapable
	case "phone", "voip-phone":
		capabilities = CDPCapPhone | CDPCapHost
	default:
		capabilities = CDPCapHost
	}

	length := capabilitiesTLVLen // Type (2) + Length (2) + Capabilities (4)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], CDPTLVTypeCapabilities)
	binary.BigEndian.PutUint16(tlv[2:4], capabilitiesTLVLen) // length is constant
	binary.BigEndian.PutUint32(tlv[4:8], capabilities)

	return tlv
}

// buildSoftwareVersionTLV builds the Software Version TLV.
func (h *CDPHandler) buildSoftwareVersionTLV(device *config.Device) []byte {
	// Use software version from config if available, otherwise use default
	var version []byte
	if device.CDPConfig != nil && device.CDPConfig.SoftwareVersion != "" {
		version = []byte(device.CDPConfig.SoftwareVersion)
	} else {
		version = []byte("NIAC-Go v1.5.0")
	}

	length := min(cdpTLVHeaderSize+len(version), cdpMaxUint16)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], CDPTLVTypeSoftwareVersion)
	binary.BigEndian.PutUint16(tlv[2:4], safeconv.Uint16(length))
	copy(tlv[4:], version)

	return tlv
}

// buildPlatformTLV builds the Platform TLV.
func (h *CDPHandler) buildPlatformTLV(device *config.Device) []byte {
	// Use platform from config if available, otherwise generate default
	var platform []byte
	if device.CDPConfig != nil && device.CDPConfig.Platform != "" {
		platform = []byte(device.CDPConfig.Platform)
	} else {
		platform = []byte("Simulated " + device.Type)
	}

	length := min(cdpTLVHeaderSize+len(platform), cdpMaxUint16)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], CDPTLVTypePlatform)
	binary.BigEndian.PutUint16(tlv[2:4], safeconv.Uint16(length))
	copy(tlv[4:], platform)

	return tlv
}

// calculateChecksum calculates the CDP checksum.
func (h *CDPHandler) calculateChecksum(data []byte) uint16 {
	// Standard Internet checksum
	sum := uint32(0)

	// Sum 16-bit words
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}

	// Handle odd byte
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << cdpBitShift
	}

	// Fold 32-bit sum to 16 bits
	for sum > cdpWordMask {
		sum = (sum >> cdpWordShift) + (sum & cdpWordMask)
	}

	// Return one's complement
	return ^uint16(sum)
}

// sendFrame sends a CDP frame.
func (h *CDPHandler) sendFrame(device *config.Device, cdpPayload []byte) error {
	return sendDiscoveryFrame(CDPMulticastMAC, device, cdpPayload, h.stack)
}

// HandlePacket processes an incoming CDP packet (for future use).
func (h *CDPHandler) HandlePacket(pkt *Packet) {
	logger := slog.Default()
	debugLevel := h.stack.GetDebugLevel()

	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	cdpLayer := packet.Layer(layers.LayerTypeCiscoDiscovery)

	infoLayer := packet.Layer(layers.LayerTypeCiscoDiscoveryInfo)
	if cdpLayer == nil || infoLayer == nil {
		return
	}

	cdp, ok := cdpLayer.(*layers.CiscoDiscovery)
	if !ok {
		return
	}

	info, ok := infoLayer.(*layers.CiscoDiscoveryInfo)
	if !ok {
		return
	}

	device := h.stack.selectDiscoveryDevice(ProtocolCDP)
	if device == nil {
		return
	}

	entry := NeighborRecord{
		Protocol:        ProtocolCDP,
		LocalDevice:     device.Name,
		RemoteDevice:    info.DeviceID,
		RemoteChassisID: info.DeviceID,
		RemotePort:      info.PortID,
		Description:     info.Platform,
		TTL:             time.Duration(cdp.TTL) * time.Second,
		Capabilities:    cdpCapabilitiesToStrings(info.Capabilities),
	}

	if info.SysName != "" {
		entry.RemoteDevice = info.SysName
	}

	if entry.TTL <= 0 {
		entry.TTL = time.Duration(CDPHoldtime) * time.Second
	}

	if len(info.MgmtAddresses) > 0 {
		entry.ManagementAddress = info.MgmtAddresses[0].String()
	} else if len(info.Addresses) > 0 {
		entry.ManagementAddress = info.Addresses[0].String()
	}

	if debugLevel >= DebugLevelInfo {
		logger.Debug(
			"CDP: Neighbor discovered",
			"remoteDevice",
			entry.RemoteDevice,
			"remotePort",
			entry.RemotePort,
			"localDevice",
			entry.LocalDevice,
		)
	}

	h.stack.recordNeighbor(entry)
}

func cdpCapabilitiesToStrings(caps layers.CDPCapabilities) []string {
	var out []string
	if caps.L3Router {
		out = append(out, deviceTypeRouter)
	}

	if caps.L2Switch {
		out = append(out, deviceTypeSwitch)
	}

	if caps.TBBridge || caps.SPBridge {
		out = append(out, "bridge")
	}

	if caps.IsHost {
		out = append(out, "host")
	}

	if caps.L1Repeater {
		out = append(out, "repeater")
	}

	if caps.IsPhone {
		out = append(out, "phone")
	}

	if caps.RemotelyManaged {
		out = append(out, "remote")
	}

	if caps.IGMPFilter {
		out = append(out, "igmp-filter")
	}

	return dedupStrings(out)
}
