package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// Packet represents a network packet with metadata.
type Packet struct {
	Buffer               []byte
	Length               int
	SerialNumber         int
	Timestamp            time.Time
	LoopTime             time.Duration // For periodic packets
	Device               any           // Associated device
	VLAN                 int           // -1 if no VLAN
	VLANTagged           bool          // true when the received wire frame carried 802.1Q
	fabricReplySourceMAC net.HardwareAddr
	fabricFirstHopIP     net.IP
	fabricFirstHopMAC    net.HardwareAddr
	fabricFirstHopDevice *config.Device
	fabricTrace          FabricTrace
}

// FabricTrace describes the routed-lab decision made for one packet.
type FabricTrace struct {
	IngressNetwork  string
	PhysicalVLAN    uint16
	RouteDecision   string
	Hop             string
	EgressNetwork   string
	RejectionReason string
}

// Constants for packet parsing.
const (
	SizeOfMac           = 6
	SizeOfIP            = 4
	SizeOfIPv6          = 16
	EtherTypeIP         = 0x0800
	EtherTypeARP        = 0x0806
	EtherTypeIPv6       = 0x86dd
	EtherTypeVLAN       = 0x8100
	EtherTypeLLDP       = 0x88cc
	EtherTypeEDP        = 0x00E02B
	EtherTypeFDP        = 0x8037
	etherHeaderSize     = 14     // Ethernet header size
	vlanIDMask          = 0x0FFF // VLAN ID mask (12 bits)
	ethTypeOffsetMult   = 2      // Multiplier for EtherType offset (src+dst MACs)
	checksumWordMask    = 0xFFFF // Mask for 16-bit word in checksum calculation
	checksumWordShift   = 16     // Bit shift for folding 32-bit to 16-bit checksum
	checksumByteShift   = 8      // Bit shift for padding odd-length byte in checksum
	vlanEtherTypeOffset = etherHeaderSize + 2
)

// MaxPacketSize is the maximum IP packet size (IPv4/IPv6)
// SECURITY FIX MEDIUM-4: Prevent memory exhaustion from large buffer requests.
const MaxPacketSize = 65535

// NewPacket creates a new packet with a buffer
// SECURITY FIX MEDIUM-4: Validates size to prevent memory exhaustion.
func NewPacket(size int) *Packet {
	// Validate packet size bounds
	if size < 0 || size > MaxPacketSize {
		size = 1514 // Standard Ethernet MTU (defense in depth)
	}

	return &Packet{
		Buffer:    make([]byte, size),
		Length:    0,
		VLAN:      -1,
		Timestamp: time.Now(),
	}
}

// FabricTrace returns a copy of the packet's routed-lab decision metadata.
func (p *Packet) FabricTrace() FabricTrace {
	if p == nil {
		return FabricTrace{}
	}
	return p.fabricTrace
}

// Clone creates a deep copy of the packet.
func (p *Packet) Clone() *Packet {
	clone := &Packet{
		Buffer:               make([]byte, len(p.Buffer)),
		Length:               p.Length,
		SerialNumber:         p.SerialNumber,
		Timestamp:            p.Timestamp,
		LoopTime:             p.LoopTime,
		Device:               p.Device,
		VLAN:                 p.VLAN,
		VLANTagged:           p.VLANTagged,
		fabricReplySourceMAC: cloneMAC(p.fabricReplySourceMAC),
		fabricFirstHopIP:     append(net.IP(nil), p.fabricFirstHopIP...),
		fabricFirstHopMAC:    cloneMAC(p.fabricFirstHopMAC),
		fabricFirstHopDevice: p.fabricFirstHopDevice,
		fabricTrace:          p.fabricTrace,
	}
	copy(clone.Buffer, p.Buffer)

	return clone
}

// Get16 reads a 16-bit value at offset.
func (p *Packet) Get16(offset int) uint16 {
	if offset+2 > len(p.Buffer) {
		return 0
	}

	return binary.BigEndian.Uint16(p.Buffer[offset:])
}

// Put16 writes a 16-bit value at offset.
func (p *Packet) Put16(value uint16, offset int) {
	if offset+2 <= len(p.Buffer) {
		binary.BigEndian.PutUint16(p.Buffer[offset:], value)
	}
}

// Get32 reads a 32-bit value at offset.
func (p *Packet) Get32(offset int) uint32 {
	if offset+4 > len(p.Buffer) {
		return 0
	}

	return binary.BigEndian.Uint32(p.Buffer[offset:])
}

// Put32 writes a 32-bit value at offset.
func (p *Packet) Put32(value uint32, offset int) {
	if offset+4 <= len(p.Buffer) {
		binary.BigEndian.PutUint32(p.Buffer[offset:], value)
	}
}

// GetMAC reads a MAC address at offset.
func (p *Packet) GetMAC(offset int) net.HardwareAddr {
	if offset+SizeOfMac > len(p.Buffer) {
		return nil
	}

	mac := make(net.HardwareAddr, SizeOfMac)
	copy(mac, p.Buffer[offset:offset+SizeOfMac])

	return mac
}

// PutMAC writes a MAC address at offset.
func (p *Packet) PutMAC(mac net.HardwareAddr, offset int) {
	if offset+SizeOfMac <= len(p.Buffer) && len(mac) == SizeOfMac {
		copy(p.Buffer[offset:], mac)
	}
}

// GetIP reads an IPv4 address at offset.
func (p *Packet) GetIP(offset int) net.IP {
	if offset+SizeOfIP > len(p.Buffer) {
		return nil
	}

	ip := make(net.IP, SizeOfIP)
	copy(ip, p.Buffer[offset:offset+SizeOfIP])

	return ip
}

// PutIP writes an IPv4 address at offset.
func (p *Packet) PutIP(ip net.IP, offset int) {
	if offset+SizeOfIP <= len(p.Buffer) {
		copy(p.Buffer[offset:], ip.To4())
	}
}

// GetSourceMAC returns the source MAC address.
func (p *Packet) GetSourceMAC() net.HardwareAddr {
	return p.GetMAC(SizeOfMac)
}

// GetDestMAC returns the destination MAC address.
func (p *Packet) GetDestMAC() net.HardwareAddr {
	return p.GetMAC(0)
}

// PutSourceMAC sets the source MAC address.
func (p *Packet) PutSourceMAC(mac net.HardwareAddr) {
	p.PutMAC(mac, SizeOfMac)
}

// PutDestMAC sets the destination MAC address.
func (p *Packet) PutDestMAC(mac net.HardwareAddr) {
	p.PutMAC(mac, 0)
}

// CopySourceMACToDest copies source MAC to destination.
func (p *Packet) CopySourceMACToDest() {
	copy(p.Buffer[0:SizeOfMac], p.Buffer[SizeOfMac:SizeOfMac*2])
}

// GetEtherType returns the EtherType field.
func (p *Packet) GetEtherType() uint16 {
	return p.Get16(SizeOfMac * ethTypeOffsetMult)
}

// ParsePacket parses raw bytes into a Packet.
func ParsePacket(data []byte, serialNum int) (*Packet, error) {
	if len(data) < etherHeaderSize {
		return nil, fmt.Errorf("%w: %w", ErrDecodingPacket, ErrEthernetFrameTooShort)
	}
	pkt := &Packet{
		Buffer:       data,
		Length:       len(data),
		SerialNumber: serialNum,
		Timestamp:    time.Now(),
		VLAN:         -1,
	}

	// Check for VLAN tag
	etherType := pkt.GetEtherType()
	if etherType == EtherTypeVLAN {
		if len(data) < etherHeaderSize+dot1qTagLen {
			return nil, fmt.Errorf("%w: %w", ErrDecodingPacket, ErrVLANHeaderTruncated)
		}
		if innerType := pkt.Get16(vlanEtherTypeOffset); innerType == EtherTypeVLAN {
			return nil, fmt.Errorf("%w: %w", ErrDecodingPacket, ErrVLANStackUnsupported)
		}
		vlanInfo := pkt.Get16(SizeOfMac*ethTypeOffsetMult + ethTypeOffsetMult)
		pkt.VLAN = int(vlanInfo & vlanIDMask)
		pkt.VLANTagged = true
	}

	return pkt, nil
}

// BuildEthernetHeader builds an Ethernet II header.
func BuildEthernetHeader(dst, src net.HardwareAddr, etherType uint16) []byte {
	header := make([]byte, etherHeaderSize)
	copy(header[0:6], dst)
	copy(header[6:12], src)
	binary.BigEndian.PutUint16(header[12:14], etherType)

	return header
}

// CalculateIPChecksum calculates IP header checksum.
func CalculateIPChecksum(header []byte) uint16 {
	sum := uint32(0)
	// Sum 16-bit words, guarding against odd-length input which would slice
	// out-of-bounds on the last iteration.
	end := len(header) &^ 1

	for i := 0; i < end; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(header[i:]))
	}

	if len(header)%2 == 1 {
		sum += uint32(header[len(header)-1]) << checksumByteShift
	}
	// Add carry
	for sum > checksumWordMask {
		sum = (sum & checksumWordMask) + (sum >> checksumWordShift)
	}

	return ^uint16(sum)
}

// DecodePacket uses gopacket to decode packet layers.
func DecodePacket(data []byte) (gopacket.Packet, error) {
	packet := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.Default)
	if packet.ErrorLayer() != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecodingPacket, packet.ErrorLayer().Error())
	}

	return packet, nil
}
