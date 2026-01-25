package ipc

import (
	"fmt"
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

// getRecentLogs returns recent log entries based on current simulation state.
func (s *Server) getRecentLogs(count int, minLevel LogLevel) []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs := make([]LogEntry, 0)

	// Add a log entry showing current status
	if compareLevels(LogLevelInfo, minLevel) >= 0 {
		logs = append(logs, LogEntry{
			Timestamp: time.Now(),
			Level:     LogLevelInfo,
			Message:   fmt.Sprintf("Simulation running on %s with %d devices", s.interfaceName, len(s.cfg.Devices)),
			Source:    "system",
		})
	}

	// Add device-related log entries
	for _, device := range s.cfg.Devices {
		if len(logs) >= count {
			break
		}

		if compareLevels(LogLevelInfo, minLevel) >= 0 {
			logs = append(logs, LogEntry{
				Timestamp: time.Now(),
				Level:     LogLevelInfo,
				Message:   fmt.Sprintf("Device %s active (%s)", device.Name, device.Type),
				Source:    "device",
				Device:    device.Name,
			})
		}
	}

	// Add error injection logs
	states := s.stateManager.GetAllStates()
	for _, state := range states {
		if len(logs) >= count {
			break
		}

		if compareLevels(LogLevelWarn, minLevel) >= 0 {
			logs = append(logs, LogEntry{
				Timestamp: time.Now(),
				Level:     LogLevelWarn,
				Message: fmt.Sprintf(
					"Error injection active: %s on %s (value: %d)",
					state.ErrorType,
					state.DeviceIP,
					state.Value,
				),
				Source: "error-injection",
				Device: state.DeviceIP,
			})
		}
	}

	return logs
}

// compareLevels returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareLevels(a, b LogLevel) int {
	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  logLevelWarnPriority,
		LogLevelError: logLevelErrPriority,
	}

	aVal, bVal := levels[a], levels[b]
	if aVal < bVal {
		return -1
	}

	if aVal > bVal {
		return 1
	}

	return 0
}
