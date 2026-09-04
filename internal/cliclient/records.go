package cliclient

import (
	"strings"
	"time"
)

// LogLevel is the severity of a log record.
type LogLevel string

const (
	// LogLevelDebug marks diagnostic detail not needed in normal operation.
	LogLevelDebug LogLevel = "debug"
	// LogLevelInfo marks routine operational messages.
	LogLevelInfo LogLevel = "info"
	// LogLevelWarn marks a condition worth operator attention but not failing the simulation.
	LogLevelWarn LogLevel = "warn"
	// LogLevelError marks a failure in the simulation or a protocol handler.
	LogLevelError LogLevel = "error"
)

// LogEntry is a log record in the shape the CLI prints and emits as JSON,
// decoded from the LogEvent the daemon streams.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source,omitempty"`   // e.g., "LLDP", "ARP", device name
	Device    string    `json:"device,omitempty"`   // device name if applicable
	Protocol  string    `json:"protocol,omitempty"` // protocol if applicable
}

// PacketData is a captured frame in the shape `niac dump` prints, decoded from
// the PacketEvent the daemon streams.
type PacketData struct {
	Timestamp time.Time `json:"timestamp"`
	Length    int       `json:"length"`
	Device    string    `json:"device,omitempty"`
	Interface string    `json:"interface,omitempty"`
	Data      []byte    `json:"data"`
}

// LogMatchesFilter reports whether filter appears, case-insensitively, in any
// field of the record the reader can see. An empty filter keeps everything.
func LogMatchesFilter(log LogEntry, filter string) bool {
	if filter == "" {
		return true
	}
	wanted := strings.ToLower(filter)
	for _, field := range []string{log.Message, log.Device, log.Source, log.Protocol} {
		if strings.Contains(strings.ToLower(field), wanted) {
			return true
		}
	}

	return false
}
