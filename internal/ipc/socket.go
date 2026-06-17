package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	apperr "github.com/MustardSeedNetworks/niac-go/internal/apperr"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
	"github.com/MustardSeedNetworks/niac-go/internal/topology"
)

// ErrIPCServerAlreadyRunning is returned when the IPC server is already running.
var ErrIPCServerAlreadyRunning = errors.New("IPC server already running")

// Command represents an IPC command.
type Command string

const (
	// CommandStatus queries the current simulation status.
	CommandStatus Command = "status"
	// CommandReload reloads the configuration.
	CommandReload Command = "reload"
	// CommandInject injects an error.
	CommandInject Command = "inject"
	// CommandList lists active error injections.
	CommandList Command = "list"
	// CommandClear clears error injections.
	CommandClear Command = "clear"
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

// Socket configuration constants.
const (
	connectionTimeoutSec = 5   // timeout for IPC connections
	shutdownDelayMs      = 500 // delay before shutdown signal processing
	logLevelWarnPriority = 2   // log level priority for warn
	logLevelErrPriority  = 3   // log level priority for error
)

// PacketBufferSize is the maximum number of packets to store in the ring buffer.
const PacketBufferSize = 1000

// LogLevel represents the severity level of a log entry.
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
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

// ErrorInjectionData contains error injection information.
type ErrorInjectionData struct {
	Device    string           `json:"device"`
	Interface string           `json:"interface"`
	ErrorType apperr.ErrorType `json:"error_type"`
	Value     int              `json:"value"`
	Injected  time.Time        `json:"injected_at"`
}

// PacketData contains captured packet information for hex dump display.
type PacketData struct {
	Timestamp time.Time `json:"timestamp"`
	Length    int       `json:"length"`
	Device    string    `json:"device,omitempty"`
	Interface string    `json:"interface,omitempty"`
	Data      []byte    `json:"data"`
}

// PacketBuffer is a thread-safe ring buffer for storing captured packets.
type PacketBuffer struct {
	packets []PacketData
	head    int
	count   int
	mu      sync.RWMutex
}

// Server is an IPC socket server for remote control.
type Server struct {
	socketPath    string
	listener      net.Listener
	stack         *protocols.Stack
	cfg           *config.Config
	stateManager  *apperr.StateManager
	interfaceName string
	configPath    string
	startTime     time.Time
	running       bool
	mu            sync.RWMutex
	shutdownChan  chan struct{}
	packetBuffer  *PacketBuffer
}

// NewServer creates a new IPC server.
func NewServer(socketPath string, stack *protocols.Stack, cfg *config.Config,
	stateManager *apperr.StateManager, interfaceName, configPath string,
) *Server {
	return &Server{
		socketPath:    socketPath,
		stack:         stack,
		cfg:           cfg,
		stateManager:  stateManager,
		interfaceName: interfaceName,
		configPath:    configPath,
		startTime:     time.Now(),
		running:       false,
		shutdownChan:  make(chan struct{}),
		packetBuffer:  NewPacketBuffer(PacketBufferSize),
	}
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

// Start starts the IPC server.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrIPCServerAlreadyRunning
	}

	// Remove existing socket if present. ENOENT is expected on first run.
	if err := os.Remove(s.socketPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create socket directory if needed
	socketDir := filepath.Dir(s.socketPath)
	if err := os.MkdirAll(socketDir, 0o750); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Create Unix domain socket
	lc := net.ListenConfig{}

	listener, err := lc.Listen(context.Background(), "unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}

	// Set socket permissions (owner only)
	chmodErr := os.Chmod(s.socketPath, 0o600)
	if chmodErr != nil {
		_ = listener.Close()

		return fmt.Errorf("failed to set socket permissions: %w", chmodErr)
	}

	s.listener = listener
	s.running = true

	// Start accepting connections in background
	go s.acceptLoop()

	logging.Infof("IPC server started on %s", s.socketPath)

	return nil
}

// Stop stops the IPC server.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	close(s.shutdownChan)
	s.running = false

	if s.listener != nil {
		_ = s.listener.Close()
	}

	// Remove socket file
	_ = os.Remove(s.socketPath)

	logging.Infof("IPC server stopped")

	return nil
}

// acceptLoop accepts incoming connections.
func (s *Server) acceptLoop() {
	for {
		select {
		case <-s.shutdownChan:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			// Check if we're shutting down
			select {
			case <-s.shutdownChan:
				return
			default:
				logging.Errorf("IPC accept error: %v", err)

				continue
			}
		}

		// Handle connection in goroutine
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single IPC connection.
func (s *Server) handleConnection(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	// Set read/write timeouts
	_ = conn.SetDeadline(
		time.Now().Add(connectionTimeoutSec * time.Second),
	) // error is non-critical, connection will timeout naturally

	// SECURITY FIX: Limit IPC message size to prevent memory exhaustion (1MB)
	const maxIPCMessageSize = 1 << 20
	decoder := json.NewDecoder(io.LimitReader(conn, maxIPCMessageSize))

	var req Request
	err := decoder.Decode(&req)
	if err != nil {
		s.sendError(conn, fmt.Errorf("invalid request: %w", err))

		return
	}

	// Process command
	response := s.processCommand(&req)

	// Send response
	encoder := json.NewEncoder(conn)
	err = encoder.Encode(response)
	if err != nil {
		logging.Errorf("Failed to send IPC response: %v", err)
	}
}

// processCommand processes an IPC command.
func (s *Server) processCommand(req *Request) *Response {
	switch req.Command {
	case CommandStatus:
		return s.handleStatus(req)
	case CommandReload:
		return s.handleReload(req)
	case CommandInject:
		return s.handleInject(req)
	case CommandList:
		return s.handleList(req)
	case CommandClear:
		return s.handleClear(req)
	case CommandShutdown:
		return s.handleShutdown(req)
	case CommandLogs:
		return s.handleLogs(req)
	case CommandTopology:
		return s.handleTopology(req)
	case CommandDump:
		return s.handleDump(req)
	case CommandNeighbors:
		return s.handleNeighbors(req)
	default:
		return &Response{
			Success: false,
			Error:   fmt.Sprintf("unknown command: %s", req.Command),
		}
	}
}

// handleStatus returns simulation status.
func (s *Server) handleStatus(_ *Request) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := s.stack.GetStats()
	uptime := time.Since(s.startTime).Seconds()

	status := StatusData{
		Running:      true,
		Interface:    s.interfaceName,
		ConfigPath:   s.configPath,
		DeviceCount:  len(s.cfg.Devices),
		Uptime:       uptime,
		StartedAt:    s.startTime,
		PacketsRX:    stats.PacketsReceived,
		PacketsTX:    stats.PacketsSent,
		ErrorsActive: len(s.stateManager.GetAllStates()),
	}

	return &Response{
		Success: true,
		Data: map[string]any{
			"status": status,
		},
	}
}

// handleReload reloads the configuration.
func (s *Server) handleReload(_ *Request) *Response {
	// Load new config
	newCfg, err := config.Load(s.configPath)
	if err != nil {
		return &Response{
			Success: false,
			Error:   fmt.Sprintf("failed to load config: %v", err),
		}
	}

	// Validate new config
	validator := config.NewValidator(s.configPath)
	errs := validator.Validate(newCfg)
	if !errs.Valid {
		return &Response{
			Success: false,
			Error:   fmt.Sprintf("config validation failed: %v", errs),
		}
	}

	s.mu.Lock()
	s.cfg = newCfg
	s.mu.Unlock()

	// Note: Full hot-reload would require updating the stack
	// For now, we just update the config reference
	logging.Infof("Configuration reloaded from %s", s.configPath)

	return &Response{
		Success: true,
		Data: map[string]any{
			"message":      "configuration reloaded",
			"device_count": len(newCfg.Devices),
		},
	}
}

// parseInjectArgs extracts and validates device/error_type/value from a request's args.
// Returns (deviceName, errorType, value, errResp). errResp is non-nil when any arg is missing.
func parseInjectArgs(args map[string]any) (string, string, float64, *Response) {
	deviceName, ok := args["device"].(string)
	if !ok {
		return "", "", 0, &Response{
			Success: false,
			Error:   "missing or invalid 'device' argument",
		}
	}
	errorTypeStr, ok := args["error_type"].(string)
	if !ok {
		return "", "", 0, &Response{
			Success: false,
			Error:   "missing or invalid 'error_type' argument",
		}
	}
	value, ok := args["value"].(float64) // JSON numbers are float64
	if !ok {
		return "", "", 0, &Response{
			Success: false,
			Error:   "missing or invalid 'value' argument",
		}
	}
	return deviceName, errorTypeStr, value, nil
}

// findConfigDevice returns a pointer to the first device with the given name, or nil.
func findConfigDevice(devices []config.Device, name string) *config.Device {
	for i := range devices {
		if devices[i].Name == name {
			return &devices[i]
		}
	}
	return nil
}

// handleInject injects an error.
func (s *Server) handleInject(req *Request) *Response {
	deviceName, errorTypeStr, value, errResp := parseInjectArgs(req.Args)
	if errResp != nil {
		return errResp
	}

	device := findConfigDevice(s.cfg.Devices, deviceName)
	if device == nil {
		return &Response{
			Success: false,
			Error:   "device not found: " + deviceName,
		}
	}

	errorType := apperr.ErrorType(errorTypeStr)
	interfaceName := s.interfaceName
	if len(device.IPAddresses) > 0 {
		interfaceName = device.IPAddresses[0].String()
	}

	s.stateManager.SetError(deviceName, interfaceName, errorType, int(value))

	logging.Infof("Error injected via IPC: device=%s, type=%s, value=%d",
		deviceName, errorType, int(value))

	return &Response{
		Success: true,
		Data: map[string]any{
			"message":    "error injected",
			"device":     deviceName,
			"error_type": errorType,
			"value":      int(value),
		},
	}
}

// handleList lists active error injections.
func (s *Server) handleList(_ *Request) *Response {
	states := s.stateManager.GetAllStates()

	injections := make([]ErrorInjectionData, 0, len(states))
	for _, state := range states {
		injections = append(injections, ErrorInjectionData{
			Device:    state.DeviceIP,
			Interface: state.Interface,
			ErrorType: state.ErrorType,
			Value:     state.Value,
			Injected:  time.Now(), // ErrorState doesn't store injection time
		})
	}

	return &Response{
		Success: true,
		Data: map[string]any{
			"injections": injections,
			"count":      len(injections),
		},
	}
}

// handleClear clears error injections.
func (s *Server) handleClear(req *Request) *Response {
	// Check for device filter
	if deviceName, ok := req.Args["device"].(string); ok {
		// Clear for specific device
		cleared := 0

		states := s.stateManager.GetAllStates()
		for _, state := range states {
			if state.DeviceIP == deviceName {
				s.stateManager.ClearError(state.DeviceIP, state.Interface)

				cleared++
			}
		}

		logging.Infof("Cleared %d error injections for device %s", cleared, deviceName)

		return &Response{
			Success: true,
			Data: map[string]any{
				"message": fmt.Sprintf("cleared %d injections for device %s", cleared, deviceName),
				"cleared": cleared,
			},
		}
	}

	// Clear all
	s.stateManager.ClearAll()
	logging.Infof("All error injections cleared via IPC")

	return &Response{
		Success: true,
		Data: map[string]any{
			"message": "all error injections cleared",
		},
	}
}

// handleShutdown initiates graceful shutdown.
func (s *Server) handleShutdown(_ *Request) *Response {
	logging.Infof("Shutdown requested via IPC")

	// Send response before shutting down
	response := &Response{
		Success: true,
		Data: map[string]any{
			"message": "shutdown initiated",
		},
	}

	// Schedule shutdown in background
	go func() {
		time.Sleep(shutdownDelayMs * time.Millisecond)
		// The caller should handle this signal
		// For now, just log it
		logging.Infof("IPC shutdown signal processed")
	}()

	return response
}

// handleLogs returns recent log entries from the simulation
// Supports filtering by log level and limiting the number of entries.
func (s *Server) handleLogs(req *Request) *Response {
	// Extract optional parameters
	count := 100 // default number of log entries
	if countVal, ok := req.Args["count"].(float64); ok {
		count = int(countVal)
	}

	minLevel := LogLevelDebug

	if levelStr, ok := req.Args["level"].(string); ok {
		switch levelStr {
		case "debug":
			minLevel = LogLevelDebug
		case "info":
			minLevel = LogLevelInfo
		case "warn":
			minLevel = LogLevelWarn
		case "error":
			minLevel = LogLevelError
		}
	}

	// Generate log entries based on current state
	logs := s.getRecentLogs(count, minLevel)

	return &Response{
		Success: true,
		Data: map[string]any{
			"logs":  logs,
			"count": len(logs),
		},
	}
}

// getRecentLogs returns recent log entries based on current simulation state.
func (s *Server) getRecentLogs(count int, minLevel LogLevel) []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs := make([]LogEntry, 0)

	if compareLevels(LogLevelInfo, minLevel) >= 0 {
		logs = append(logs, LogEntry{
			Timestamp: time.Now(),
			Level:     LogLevelInfo,
			Message:   fmt.Sprintf("Simulation running on %s with %d devices", s.interfaceName, len(s.cfg.Devices)),
			Source:    "system",
		})
	}

	logs = appendDeviceLogs(logs, s.cfg.Devices, count, minLevel)
	logs = appendErrorInjectionLogs(logs, s.stateManager.GetAllStates(), count, minLevel)

	return logs
}

// appendDeviceLogs adds device-active entries until count or min-level filter stops it.
func appendDeviceLogs(logs []LogEntry, devices []config.Device, count int, minLevel LogLevel) []LogEntry {
	for _, device := range devices {
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
	return logs
}

// appendErrorInjectionLogs adds warn entries for each active error-injection state.
func appendErrorInjectionLogs(
	logs []LogEntry, states []*apperr.ErrorState, count int, minLevel LogLevel,
) []LogEntry {
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

// handleTopology returns the current network topology.
func (s *Server) handleTopology(_ *Request) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build topology from current configuration
	graph := topology.Build(s.cfg)

	return &Response{
		Success: true,
		Data: map[string]any{
			"topology": graph,
		},
	}
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

// handleNeighbors returns the neighbor discovery table.
func (s *Server) handleNeighbors(_ *Request) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get neighbors from the protocol stack
	neighbors := s.stack.GetNeighbors()
	if neighbors == nil {
		neighbors = []protocols.NeighborRecord{}
	}

	// Convert to IPC-friendly format
	result := make([]NeighborData, 0, len(neighbors))
	for _, n := range neighbors {
		result = append(result, NeighborData{
			Protocol:          n.Protocol,
			LocalDevice:       n.LocalDevice,
			RemoteDevice:      n.RemoteDevice,
			RemotePort:        n.RemotePort,
			RemoteChassisID:   n.RemoteChassisID,
			Description:       n.Description,
			Capabilities:      n.Capabilities,
			ManagementAddress: n.ManagementAddress,
			LastSeen:          n.LastSeen,
			ExpireAt:          n.ExpireAt,
		})
	}

	return &Response{
		Success: true,
		Data: map[string]any{
			"neighbors": result,
			"count":     len(result),
		},
	}
}

// sendError sends an error response.
func (s *Server) sendError(conn net.Conn, err error) {
	response := &Response{
		Success: false,
		Error:   err.Error(),
	}

	encoder := json.NewEncoder(conn)
	_ = encoder.Encode(response) // error is non-critical, connection will be closed after
}

// DefaultSocketPath returns the default socket path.
func DefaultSocketPath() string {
	tmpDir := os.TempDir()

	return filepath.Join(tmpDir, "niac.sock")
}

// handleDump returns captured packets from the buffer.
func (s *Server) handleDump(req *Request) *Response {
	// Extract filter arguments
	device := ""
	iface := ""
	count := 0

	if d, ok := req.Args["device"].(string); ok {
		device = d
	}

	if i, ok := req.Args["interface"].(string); ok {
		iface = i
	}

	if c, ok := req.Args["count"].(float64); ok { // JSON numbers are float64
		count = int(c)
	}

	// Get filtered packets from buffer
	packets := s.packetBuffer.GetFiltered(device, iface, count)

	return &Response{
		Success: true,
		Data: map[string]any{
			"packets": packets,
			"count":   len(packets),
		},
	}
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
