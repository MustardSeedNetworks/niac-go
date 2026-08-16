package ipc

import (
	"os"
	"path/filepath"
	"time"
)

// Command represents an IPC command.
type Command string

const (
	// CommandStatus queries the current simulation status.
	CommandStatus Command = "status"
	// CommandReload reloads the configuration.
	CommandReload Command = "reload"
	// CommandShutdown gracefully shuts down the simulation.
	CommandShutdown Command = "shutdown"
	// CommandLogs subscribes to log stream.
	CommandLogs Command = "logs"
	// CommandTopology retrieves the current network topology.
	CommandTopology Command = "topology"
	// CommandDump returns captured packets as hex dumps.
	CommandDump Command = "dump"
	// CommandNeighbors retrieves the neighbor discovery table.
	CommandNeighbors Command = "neighbors"
)

// LogLevel represents the severity level of a log entry.
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

// LogEntry represents a single log message from the simulation.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source,omitempty"`   // e.g., "LLDP", "ARP", device name
	Device    string    `json:"device,omitempty"`   // device name if applicable
	Protocol  string    `json:"protocol,omitempty"` // protocol if applicable
}

// Request represents an IPC request.
type Request struct {
	Command Command        `json:"command"`
	Args    map[string]any `json:"args,omitempty"`
}

// Response represents an IPC response.
type Response struct {
	Success bool           `json:"success"`
	Data    map[string]any `json:"data,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// StatusData contains simulation status information.
type StatusData struct {
	Running      bool      `json:"running"`
	Interface    string    `json:"interface"`
	ConfigPath   string    `json:"config_path"`
	DeviceCount  int       `json:"device_count"`
	Uptime       float64   `json:"uptime_seconds"`
	StartedAt    time.Time `json:"started_at"`
	PacketsRX    uint64    `json:"packets_received"`
	PacketsTX    uint64    `json:"packets_sent"`
	ErrorsActive int       `json:"errors_active"`
}

// PacketData contains captured packet information for hex dump display.
type PacketData struct {
	Timestamp time.Time `json:"timestamp"`
	Length    int       `json:"length"`
	Device    string    `json:"device,omitempty"`
	Interface string    `json:"interface,omitempty"`
	Data      []byte    `json:"data"`
}

// NeighborData represents neighbor discovery information for IPC responses.
type NeighborData struct {
	Protocol          string    `json:"protocol"`
	LocalDevice       string    `json:"local_device"`
	RemoteDevice      string    `json:"remote_device"`
	RemotePort        string    `json:"remote_port"`
	RemoteChassisID   string    `json:"remote_chassis_id"`
	Description       string    `json:"description"`
	Capabilities      []string  `json:"capabilities"`
	ManagementAddress string    `json:"management_address"`
	LastSeen          time.Time `json:"last_seen"`
	ExpireAt          time.Time `json:"expire_at"`
}

// DefaultSocketPath returns the default socket path.
func DefaultSocketPath() string {
	tmpDir := os.TempDir()

	return filepath.Join(tmpDir, "niac.sock")
}
