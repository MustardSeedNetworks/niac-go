package cliclient

import (
	"strings"
	"time"
)

// The log and packet shapes the CLI renders.
//
// These lived in internal/ipc, a Unix-socket client whose server was removed in
// #1012. Nothing constructed that client afterwards -- only these four types
// were still referenced, by `niac logs` and `niac dump`, both of which read
// from the daemon's HTTP API. They are here so the CLI's own vocabulary lives
// with the CLI's client rather than in a package named after a transport that
// no longer exists.

// LogLevel represents the severity level of a log entry.
type LogLevel string

const (
	// LogLevelDebug marks diagnostic detail not needed in normal operation.
	LogLevelDebug LogLevel = "debug"
	// LogLevelInfo marks routine operational messages.
	LogLevelInfo LogLevel = "info"
	// LogLevelWarn marks a condition worth operator attention but not failing
	// the simulation.
	LogLevelWarn LogLevel = "warn"
	// LogLevelError marks a failure in the simulation or a protocol handler.
	LogLevelError LogLevel = "error"
)

// LogEntry represents a single log message from the simulation.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source,omitempty"`   // e.g., "LLDP", "ARP", device name
	Device    string    `json:"device,omitempty"`   // device name if applicable
	Protocol  string    `json:"protocol,omitempty"` // protocol if applicable
}

// PacketData contains captured packet information for hex dump display.
type PacketData struct {
	Timestamp time.Time `json:"timestamp"`
	Length    int       `json:"length"`
	Device    string    `json:"device,omitempty"`
	Interface string    `json:"interface,omitempty"`
	Data      []byte    `json:"data"`
}

// LogMatchesFilter reports whether a log record matches the operator's text
// filter. It is the one implementation: `niac logs tail --filter X` and the same
// command with --follow are the same flag and must answer the same way, so both
// the streaming and batch paths call this.
//
// The match is a case-insensitive substring across every field the operator can
// see in the output - message, device, source and protocol - which is what the
// CLI help promises and what the log viewer's single text box implies.
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
