package ipc

import (
	"encoding/json"
	"net"
	"time"
)

// sendError sends an error response.
func (s *Server) sendError(conn net.Conn, err error) {
	response := &Response{
		Success: false,
		Error:   err.Error(),
	}

	encoder := json.NewEncoder(conn)
	_ = encoder.Encode(response) // error is non-critical, connection will be closed after
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
