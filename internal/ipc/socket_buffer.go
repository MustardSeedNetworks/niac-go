package ipc

import (
	"sync"
	"time"
)

// PacketBuffer is a thread-safe ring buffer for storing captured packets.
type PacketBuffer struct {
	packets []PacketData
	head    int
	count   int
	mu      sync.RWMutex
}

// NewPacketBuffer creates a new packet buffer with the given capacity.
func NewPacketBuffer(capacity int) *PacketBuffer {
	return &PacketBuffer{
		packets: make([]PacketData, capacity),
		head:    0,
		count:   0,
	}
}

// Add adds a packet to the ring buffer.
func (pb *PacketBuffer) Add(pkt PacketData) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.packets[pb.head] = pkt

	pb.head = (pb.head + 1) % len(pb.packets)

	if pb.count < len(pb.packets) {
		pb.count++
	}
}

// GetAll returns all packets in the buffer (oldest first).
func (pb *PacketBuffer) GetAll() []PacketData {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	if pb.count == 0 {
		return nil
	}

	result := make([]PacketData, pb.count)

	start := 0
	if pb.count == len(pb.packets) {
		start = pb.head
	}

	for i := range pb.count {
		idx := (start + i) % len(pb.packets)
		result[i] = pb.packets[idx]
	}

	return result
}

// GetFiltered returns packets matching the filter criteria.
func (pb *PacketBuffer) GetFiltered(device, iface string, count int) []PacketData {
	all := pb.GetAll()
	if all == nil {
		return nil
	}

	// Apply filters
	var filtered []PacketData

	for _, pkt := range all {
		if device != "" && pkt.Device != device {
			continue
		}

		if iface != "" && pkt.Interface != iface {
			continue
		}

		filtered = append(filtered, pkt)
	}

	// Apply count limit
	if count > 0 && len(filtered) > count {
		filtered = filtered[len(filtered)-count:]
	}

	return filtered
}

// AddPacket adds a packet to the capture buffer for later retrieval via the dump command.
// This should be called by the protocol stack when a packet is received.
func (s *Server) AddPacket(data []byte, device, iface string) {
	if s.packetBuffer == nil {
		return
	}

	// Make a copy of the packet data
	pktData := make([]byte, len(data))
	copy(pktData, data)

	pkt := PacketData{
		Timestamp: time.Now(),
		Length:    len(data),
		Device:    device,
		Interface: iface,
		Data:      pktData,
	}

	s.packetBuffer.Add(pkt)
}

// GetPacketBuffer returns the packet buffer for external access.
func (s *Server) GetPacketBuffer() *PacketBuffer {
	return s.packetBuffer
}
