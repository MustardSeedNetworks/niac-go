package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
)

// EDP protocol constants
const (
	// EDP multicast destination MAC address (00:E0:2B:00:00:00)
	EDPMulticastMAC = "\x00\xE0\x2B\x00\x00\x00"

	// EDP advertisement interval (default 30 seconds)
	EDPAdvertiseInterval = 30 * time.Second

	// EDP version
	EDPVersion = 1
)

// EDP TLV Types
const (
	EDPTLVTypeDisplay     = 0x01 // Device display string
	EDPTLVTypeInfo        = 0x02 // Info TLV
	EDPTLVTypeWarning     = 0x03 // Warning TLV
	EDPTLVTypeNull        = 0x99 // End marker
)

// EDPHandler handles EDP advertisements
type EDPHandler struct {
	stack           *Stack
	stopChan        chan struct{}
	advertiseTicker *time.Ticker
}

// NewEDPHandler creates a new EDP handler
func NewEDPHandler(stack *Stack) *EDPHandler {
	return &EDPHandler{
		stack:    stack,
		stopChan: make(chan struct{}),
	}
}

// Start begins periodic EDP advertisements
func (h *EDPHandler) Start() {
	debugLevel := h.stack.GetDebugLevel()

	if debugLevel >= 1 {
		fmt.Printf("EDP: Starting periodic advertisements (interval: %v)\n", EDPAdvertiseInterval)
	}

	h.advertiseTicker = time.NewTicker(EDPAdvertiseInterval)

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

// Stop halts EDP advertisements
func (h *EDPHandler) Stop() {
	close(h.stopChan)
}

// sendAdvertisements sends EDP advertisements for all devices
func (h *EDPHandler) sendAdvertisements() {
	debugLevel := h.stack.GetDebugLevel()

	devices := h.stack.GetDevices().GetAll()
	for _, device := range devices {
		if len(device.MACAddress) == 0 {
			continue
		}

		// Build and send EDP frame
		frame := h.buildEDPFrame(device)
		if frame != nil {
			err := h.sendFrame(device, frame)
			if err != nil && debugLevel >= 2 {
				fmt.Printf("EDP: Error sending advertisement for %s: %v\n", device.Name, err)
			} else if debugLevel >= 3 {
				fmt.Printf("EDP: Sent advertisement for %s (%d bytes)\n", device.Name, len(frame))
			}
		}
	}
}

// buildEDPFrame constructs an EDP frame for a device
func (h *EDPHandler) buildEDPFrame(device *config.Device) []byte {
	var payload []byte

	// EDP Header
	// Version (1 byte)
	payload = append(payload, EDPVersion)

	// Reserved (1 byte)
	payload = append(payload, 0x00)

	// Sequence number (2 bytes) - could be incremented, using 0 for simplicity
	payload = append(payload, 0x00, 0x01)

	// ID Length (2 bytes) - length of device ID
	deviceID := []byte(device.Name)
	binary.BigEndian.PutUint16(payload[len(payload):len(payload)+2], uint16(len(deviceID)))
	payload = append(payload, make([]byte, 2)...)
	binary.BigEndian.PutUint16(payload[4:6], uint16(len(deviceID)))

	// Device ID
	payload = append(payload, deviceID...)

	// Add TLVs
	payload = append(payload, h.buildDisplayTLV(device)...)
	payload = append(payload, h.buildInfoTLV(device)...)

	// Add NULL TLV (end marker)
	payload = append(payload, h.buildNullTLV()...)

	// Checksum (2 bytes) at the end
	checksum := h.calculateChecksum(payload)
	checksumBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(checksumBytes, checksum)
	payload = append(payload, checksumBytes...)

	return payload
}

// buildDisplayTLV builds the Display TLV
func (h *EDPHandler) buildDisplayTLV(device *config.Device) []byte {
	display := []byte(fmt.Sprintf("%s (%s)", device.Name, device.Type))

	// TLV: Type (1 byte) + Length (2 bytes) + Value
	tlv := make([]byte, 3+len(display))
	tlv[0] = EDPTLVTypeDisplay
	binary.BigEndian.PutUint16(tlv[1:3], uint16(len(display)))
	copy(tlv[3:], display)

	return tlv
}

// buildInfoTLV builds the Info TLV
func (h *EDPHandler) buildInfoTLV(device *config.Device) []byte {
	// Info string includes various device information
	var info string

	// Add MAC address
	info += fmt.Sprintf("MAC:%s ", device.MACAddress.String())

	// Add IP addresses
	if len(device.IPAddresses) > 0 {
		info += fmt.Sprintf("IP:%s ", device.IPAddresses[0].String())
	}

	// Add device type
	info += fmt.Sprintf("Type:%s ", device.Type)

	// Add NIAC-Go identifier
	info += "NIAC-Go:v1.3.0"

	infoBytes := []byte(info)

	// TLV: Type (1 byte) + Length (2 bytes) + Value
	tlv := make([]byte, 3+len(infoBytes))
	tlv[0] = EDPTLVTypeInfo
	binary.BigEndian.PutUint16(tlv[1:3], uint16(len(infoBytes)))
	copy(tlv[3:], infoBytes)

	return tlv
}

// buildNullTLV builds the NULL TLV (end marker)
func (h *EDPHandler) buildNullTLV() []byte {
	// NULL TLV: Type (1 byte) + Length (2 bytes, value 0)
	return []byte{EDPTLVTypeNull, 0x00, 0x00}
}

// calculateChecksum calculates the EDP checksum
func (h *EDPHandler) calculateChecksum(data []byte) uint16 {
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

// sendFrame sends an EDP frame
func (h *EDPHandler) sendFrame(device *config.Device, edpPayload []byte) error {
	// Build Ethernet header
	dstMAC, _ := net.ParseMAC(EDPMulticastMAC)

	// Build raw Ethernet frame
	frame := make([]byte, 14+len(edpPayload))

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
	}

	h.stack.Send(pkt)

	return nil
}

// HandlePacket processes an incoming EDP packet (for future use)
func (h *EDPHandler) HandlePacket(pkt *Packet) {
	debugLevel := h.stack.GetDebugLevel()

	if debugLevel >= 2 {
		fmt.Printf("EDP: Received EDP frame sn=%d (parsing not yet implemented)\n", pkt.SerialNumber)
	}

	// TODO: Parse incoming EDP frames and store neighbor information
	// This would be useful for:
	// - Discovering real Extreme devices on the network
	// - Responding to EDP queries
	// - Building network topology
}
