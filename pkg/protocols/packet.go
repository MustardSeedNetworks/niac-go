// Package protocols implements network protocol handlers for NIAC
package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// Packet represents a network packet with metadata
type Packet struct {
	Buffer       []byte
	Length       int
	SerialNumber int
	Timestamp    time.Time
	LoopTime     time.Duration // For periodic packets
	Device       interface{}   // Associated device
	VLAN         int          // -1 if no VLAN
}

// Constants for packet parsing
const (
	SizeOfMac     = 6
	SizeOfIP      = 4
	SizeOfIPv6    = 16
	EtherTypeIP   = 0x0800
	EtherTypeARP  = 0x0806
	EtherTypeIPv6 = 0x86dd
	EtherTypeVLAN = 0x8100
	EtherTypeLLDP = 0x88cc
)

// NewPacket creates a new packet with a buffer
func NewPacket(size int) *Packet {
	return &Packet{
		Buffer:    make([]byte, size),
		Length:    0,
		VLAN:      -1,
		Timestamp: time.Now(),
	}
}

// Clone creates a deep copy of the packet
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

// Get16 reads a 16-bit value at offset
func (p *Packet) Get16(offset int) uint16 {
	if offset+2 > len(p.Buffer) {
		return 0
	}
	return binary.BigEndian.Uint16(p.Buffer[offset:])
}

// Put16 writes a 16-bit value at offset
func (p *Packet) Put16(value uint16, offset int) {
	if offset+2 <= len(p.Buffer) {
		binary.BigEndian.PutUint16(p.Buffer[offset:], value)
	}
}

// Get32 reads a 32-bit value at offset
func (p *Packet) Get32(offset int) uint32 {
	if offset+4 > len(p.Buffer) {
		return 0
	}
	return binary.BigEndian.Uint32(p.Buffer[offset:])
}

// Put32 writes a 32-bit value at offset
func (p *Packet) Put32(value uint32, offset int) {
	if offset+4 <= len(p.Buffer) {
		binary.BigEndian.PutUint32(p.Buffer[offset:], value)
	}
}

// GetMAC reads a MAC address at offset
func (p *Packet) GetMAC(offset int) net.HardwareAddr {
	if offset+SizeOfMac > len(p.Buffer) {
		return nil
	}
	mac := make(net.HardwareAddr, SizeOfMac)
	copy(mac, p.Buffer[offset:offset+SizeOfMac])
	return mac
}

// PutMAC writes a MAC address at offset
func (p *Packet) PutMAC(mac net.HardwareAddr, offset int) {
	if offset+SizeOfMac <= len(p.Buffer) && len(mac) == SizeOfMac {
		copy(p.Buffer[offset:], mac)
	}
}

// GetIP reads an IPv4 address at offset
func (p *Packet) GetIP(offset int) net.IP {
	if offset+SizeOfIP > len(p.Buffer) {
		return nil
	}
	ip := make(net.IP, SizeOfIP)
	copy(ip, p.Buffer[offset:offset+SizeOfIP])
	return ip
}

// PutIP writes an IPv4 address at offset
func (p *Packet) PutIP(ip net.IP, offset int) {
	if offset+SizeOfIP <= len(p.Buffer) {
		copy(p.Buffer[offset:], ip.To4())
	}
}

// GetSourceMAC returns the source MAC address
func (p *Packet) GetSourceMAC() net.HardwareAddr {
	return p.GetMAC(SizeOfMac)
}

// GetDestMAC returns the destination MAC address
func (p *Packet) GetDestMAC() net.HardwareAddr {
	return p.GetMAC(0)
}

// PutSourceMAC sets the source MAC address
func (p *Packet) PutSourceMAC(mac net.HardwareAddr) {
	p.PutMAC(mac, SizeOfMac)
}

// PutDestMAC sets the destination MAC address
func (p *Packet) PutDestMAC(mac net.HardwareAddr) {
	p.PutMAC(mac, 0)
}

// CopySourceMACToDest copies source MAC to destination
func (p *Packet) CopySourceMACToDest() {
	copy(p.Buffer[0:SizeOfMac], p.Buffer[SizeOfMac:SizeOfMac*2])
}

// GetEtherType returns the EtherType field
func (p *Packet) GetEtherType() uint16 {
	return p.Get16(SizeOfMac * 2)
}

// ParsePacket parses raw bytes into a Packet
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
		vlanInfo := pkt.Get16(SizeOfMac*2 + 2)
		pkt.VLAN = int(vlanInfo & 0x0FFF)
	}

	return pkt, nil
}

// BuildEthernetHeader builds an Ethernet II header
func BuildEthernetHeader(dst, src net.HardwareAddr, etherType uint16) []byte {
	header := make([]byte, 14)
	copy(header[0:6], dst)
	copy(header[6:12], src)
	binary.BigEndian.PutUint16(header[12:14], etherType)
	return header
}

// CalculateIPChecksum calculates IP header checksum
func CalculateIPChecksum(header []byte) uint16 {
	sum := uint32(0)
	for i := 0; i < len(header); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(header[i:]))
	}
	// Add carry
	for sum > 0xFFFF {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}
	return ^uint16(sum)
}

// DecodePacket uses gopacket to decode packet layers
func DecodePacket(data []byte) (gopacket.Packet, error) {
	packet := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.Default)
	if packet.ErrorLayer() != nil {
		return nil, fmt.Errorf("error decoding packet: %v", packet.ErrorLayer().Error())
	}
	return packet, nil
}
