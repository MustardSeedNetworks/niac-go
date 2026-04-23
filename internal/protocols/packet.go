package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// Packet represents a network packet with metadata.
type Packet struct {
	Buffer       []byte
	Length       int
	SerialNumber int
	Timestamp    time.Time
	LoopTime     time.Duration // For periodic packets
	Device       any           // Associated device
	VLAN         int           // -1 if no VLAN
}

// Constants for packet parsing.
const (
	SizeOfMac         = 6
	SizeOfIP          = 4
	SizeOfIPv6        = 16
	EtherTypeIP       = 0x0800
	EtherTypeARP      = 0x0806
	EtherTypeIPv6     = 0x86dd
	EtherTypeVLAN     = 0x8100
	EtherTypeLLDP     = 0x88cc
	EtherTypeEDP      = 0x00E02B
	EtherTypeFDP      = 0x8037
	etherHeaderSize   = 14     // Ethernet header size
	vlanIDMask        = 0x0FFF // VLAN ID mask (12 bits)
	ethTypeOffsetMult = 2      // Multiplier for EtherType offset (src+dst MACs)
	checksumWordMask  = 0xFFFF // Mask for 16-bit word in checksum calculation
	checksumWordShift = 16     // Bit shift for folding 32-bit to 16-bit checksum
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

// Clone creates a deep copy of the packet.
func (p *Packet) Clone() *Packet {
	clone := &Packet{
		Buffer:       make([]byte, len(p.Buffer)),
		Length:       p.Length,
		SerialNumber: p.SerialNumber,
		Timestamp:    p.Timestamp,
		LoopTime:     p.LoopTime,
		Device:       p.Device,
		VLAN:         p.VLAN,
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
		// VLAN tag present
		vlanInfo := pkt.Get16(SizeOfMac*ethTypeOffsetMult + ethTypeOffsetMult)
		pkt.VLAN = int(vlanInfo & vlanIDMask)
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
	// Walk pairs of bytes. An odd-length header would panic the previous
	// loop on the trailing byte (binary.BigEndian.Uint16 reads 2 bytes);
	// per RFC 1071 a lone trailing octet is promoted to the high byte of
	// a notional 16-bit word padded with zeros.
	n := len(header) &^ 1
	for i := 0; i < n; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(header[i:]))
	}
	if len(header)%2 == 1 {
		sum += uint32(header[len(header)-1]) << 8
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
