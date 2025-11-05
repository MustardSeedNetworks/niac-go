package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
)

// FDP protocol constants
const (
	// FDP multicast destination MAC address (01:E0:52:CC:CC:CC)
	FDPMulticastMAC = "\x01\xE0\x52\xCC\xCC\xCC"

	// FDP uses SNAP encapsulation similar to CDP
	// EtherType 0x8037 or LLC/SNAP with OUI 00:E0:52

	// FDP advertisement interval (default 60 seconds)
	FDPAdvertiseInterval = 60 * time.Second

	// FDP holdtime (typically 180 seconds)
	FDPHoldtime = 180 // seconds

	// FDP version
	FDPVersion = 1
)

// FDP TLV Types
const (
	FDPTLVTypeDeviceID    = 0x0001
	FDPTLVTypePort        = 0x0002
	FDPTLVTypePlatform    = 0x0003
	FDPTLVTypeCapabilities = 0x0004
	FDPTLVTypeSoftware    = 0x0005
	FDPTLVTypeIPAddress   = 0x0006
)

// FDP Capabilities flags
const (
	FDPCapRouter = 0x01
	FDPCapSwitch = 0x02
	FDPCapHost   = 0x04
)

// FDPHandler handles FDP advertisements
type FDPHandler struct {
	stack           *Stack
	stopChan        chan struct{}
	advertiseTicker *time.Ticker
}

// NewFDPHandler creates a new FDP handler
func NewFDPHandler(stack *Stack) *FDPHandler {
	return &FDPHandler{
		stack:    stack,
		stopChan: make(chan struct{}),
	}
}

// Start begins periodic FDP advertisements
func (h *FDPHandler) Start() {
	debugLevel := h.stack.GetDebugLevel()

	if debugLevel >= 1 {
		fmt.Printf("FDP: Starting periodic advertisements (interval: %v)\n", FDPAdvertiseInterval)
	}

	h.advertiseTicker = time.NewTicker(FDPAdvertiseInterval)

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

// Stop halts FDP advertisements
func (h *FDPHandler) Stop() {
	close(h.stopChan)
}

// sendAdvertisements sends FDP advertisements for all devices
func (h *FDPHandler) sendAdvertisements() {
	debugLevel := h.stack.GetDebugLevel()

	devices := h.stack.GetDevices().GetAll()
	for _, device := range devices {
		if len(device.MACAddress) == 0 {
			continue
		}

		// Skip if FDP is explicitly disabled for this device
		if device.FDPConfig != nil && !device.FDPConfig.Enabled {
			continue
		}

		// Build and send FDP frame
		frame := h.buildFDPFrame(device)
		if frame != nil {
			err := h.sendFrame(device, frame)
			if err != nil && debugLevel >= 2 {
				fmt.Printf("FDP: Error sending advertisement for %s: %v\n", device.Name, err)
			} else if debugLevel >= 3 {
				fmt.Printf("FDP: Sent advertisement for %s (%d bytes)\n", device.Name, len(frame))
			}
		}
	}
}

// buildFDPFrame constructs an FDP frame for a device
func (h *FDPHandler) buildFDPFrame(device *config.Device) []byte {
	var payload []byte

	// Use holdtime from config if available, otherwise use default
	holdtime := byte(FDPHoldtime)
	if device.FDPConfig != nil && device.FDPConfig.Holdtime > 0 {
		holdtime = byte(device.FDPConfig.Holdtime)
	}

	// FDP header: Version (1 byte) + TTL/Holdtime (1 byte) + Checksum (2 bytes)
	payload = append(payload, FDPVersion)
	payload = append(payload, holdtime)
	payload = append(payload, 0x00, 0x00) // Checksum placeholder

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
	frame := append(llcSnap, payload...)

	return frame
}

// buildLLCSNAPHeader builds the LLC/SNAP header for FDP
func (h *FDPHandler) buildLLCSNAPHeader() []byte {
	header := make([]byte, 8)

	// LLC header (3 bytes)
	header[0] = 0xAA // DSAP
	header[1] = 0xAA // SSAP
	header[2] = 0x03 // Control

	// SNAP header (5 bytes)
	// OUI (3 bytes): 00:E0:52 (Foundry/Brocade)
	header[3] = 0x00
	header[4] = 0xE0
	header[5] = 0x52

	// Protocol ID (2 bytes): 0x2000 (similar to CDP)
	binary.BigEndian.PutUint16(header[6:8], 0x2000)

	return header
}

// buildDeviceIDTLV builds the Device ID TLV
func (h *FDPHandler) buildDeviceIDTLV(device *config.Device) []byte {
	deviceID := []byte(device.Name)
	length := 4 + len(deviceID) // Type (2) + Length (2) + Value

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], FDPTLVTypeDeviceID)
	binary.BigEndian.PutUint16(tlv[2:4], uint16(length))
	copy(tlv[4:], deviceID)

	return tlv
}

// buildPortTLV builds the Port TLV
func (h *FDPHandler) buildPortTLV(device *config.Device) []byte {
	var portName []byte

	// Use port ID from config if available
	if device.FDPConfig != nil && device.FDPConfig.PortID != "" {
		portName = []byte(device.FDPConfig.PortID)
	} else if len(device.Interfaces) > 0 && device.Interfaces[0].Name != "" {
		// Try to use first interface name if available
		portName = []byte(device.Interfaces[0].Name)
	} else {
		portName = []byte("Port 1")
	}

	length := 4 + len(portName)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], FDPTLVTypePort)
	binary.BigEndian.PutUint16(tlv[2:4], uint16(length))
	copy(tlv[4:], portName)

	return tlv
}

// buildPlatformTLV builds the Platform TLV
func (h *FDPHandler) buildPlatformTLV(device *config.Device) []byte {
	// Use platform from config if available, otherwise generate default
	var platform []byte
	if device.FDPConfig != nil && device.FDPConfig.Platform != "" {
		platform = []byte(device.FDPConfig.Platform)
	} else {
		platform = []byte(fmt.Sprintf("NIAC-Go Simulated %s", device.Type))
	}

	length := 4 + len(platform)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], FDPTLVTypePlatform)
	binary.BigEndian.PutUint16(tlv[2:4], uint16(length))
	copy(tlv[4:], platform)

	return tlv
}

// buildCapabilitiesTLV builds the Capabilities TLV
func (h *FDPHandler) buildCapabilitiesTLV(device *config.Device) []byte {
	// Determine capabilities based on device type
	var capabilities uint32

	switch device.Type {
	case "router":
		capabilities = FDPCapRouter | FDPCapSwitch
	case "switch":
		capabilities = FDPCapSwitch
	default:
		capabilities = FDPCapHost
	}

	length := 8 // Type (2) + Length (2) + Capabilities (4)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], FDPTLVTypeCapabilities)
	binary.BigEndian.PutUint16(tlv[2:4], uint16(length))
	binary.BigEndian.PutUint32(tlv[4:8], capabilities)

	return tlv
}

// buildSoftwareTLV builds the Software TLV
func (h *FDPHandler) buildSoftwareTLV(device *config.Device) []byte {
	// Use software version from config if available, otherwise use default
	var software []byte
	if device.FDPConfig != nil && device.FDPConfig.SoftwareVersion != "" {
		software = []byte(device.FDPConfig.SoftwareVersion)
	} else {
		software = []byte("NIAC-Go v1.5.0")
	}

	length := 4 + len(software)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], FDPTLVTypeSoftware)
	binary.BigEndian.PutUint16(tlv[2:4], uint16(length))
	copy(tlv[4:], software)

	return tlv
}

// buildIPAddressTLV builds the IP Address TLV
func (h *FDPHandler) buildIPAddressTLV(device *config.Device) []byte {
	// Use first IP address
	ip := device.IPAddresses[0]

	var ipBytes []byte
	if ip.To4() != nil {
		ipBytes = ip.To4()
	} else {
		ipBytes = ip.To16()
	}

	length := 4 + len(ipBytes)

	tlv := make([]byte, length)
	binary.BigEndian.PutUint16(tlv[0:2], FDPTLVTypeIPAddress)
	binary.BigEndian.PutUint16(tlv[2:4], uint16(length))
	copy(tlv[4:], ipBytes)

	return tlv
}

// calculateChecksum calculates the FDP checksum
func (h *FDPHandler) calculateChecksum(data []byte) uint16 {
	// Standard Internet checksum
	sum := uint32(0)

	// Sum 16-bit words
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}

	// Handle odd byte
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	// Fold 32-bit sum to 16 bits
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}

	// Return one's complement
	return ^uint16(sum)
}

// sendFrame sends an FDP frame
func (h *FDPHandler) sendFrame(device *config.Device, fdpPayload []byte) error {
	// Build Ethernet header
	dstMAC, _ := net.ParseMAC(FDPMulticastMAC)

	// FDP uses length field instead of EtherType (802.3 format)
	length := uint16(len(fdpPayload))

	// Build raw Ethernet frame with 802.3 format
	frame := make([]byte, 14+len(fdpPayload))

	// Destination MAC
	copy(frame[0:6], dstMAC)

	// Source MAC
	copy(frame[6:12], device.MACAddress)

	// Length field (instead of EtherType)
	binary.BigEndian.PutUint16(frame[12:14], length)

	// Payload (LLC/SNAP + FDP)
	copy(frame[14:], fdpPayload)

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
	}

	h.stack.Send(pkt)

	return nil
}

// HandlePacket processes an incoming FDP packet (for future use)
func (h *FDPHandler) HandlePacket(pkt *Packet) {
	debugLevel := h.stack.GetDebugLevel()

	if debugLevel >= 2 {
		fmt.Printf("FDP: Received FDP frame sn=%d (parsing not yet implemented)\n", pkt.SerialNumber)
	}

	// TODO: Parse incoming FDP frames and store neighbor information
	// This would be useful for:
	// - Discovering real Foundry/Brocade devices on the network
	// - Responding to FDP queries
	// - Building network topology
}
