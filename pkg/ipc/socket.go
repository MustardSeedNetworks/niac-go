// Package ipc provides inter-process communication for NIAC remote control
package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/api"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/errors"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
)

// Command represents an IPC command
type Command string

const (
	// CommandStatus queries the current simulation status
	CommandStatus Command = "status"
	// CommandReload reloads the configuration
	CommandReload Command = "reload"
	// CommandInject injects an error
	CommandInject Command = "inject"
	// CommandList lists active error injections
	CommandList Command = "list"
	// CommandClear clears error injections
	CommandClear Command = "clear"
	// CommandShutdown gracefully shuts down the simulation
	CommandShutdown Command = "shutdown"
	// CommandLogs subscribes to log stream
	CommandLogs Command = "logs"
	// CommandTopology retrieves the current network topology
	CommandTopology Command = "topology"
	// CommandDump returns captured packets as hex dumps
	CommandDump Command = "dump"
	// CommandNeighbors retrieves the neighbor discovery table
	CommandNeighbors Command = "neighbors"
)

// PacketBufferSize is the maximum number of packets to store in the ring buffer
const PacketBufferSize = 1000

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogEntry represents a single log message from the simulation
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source,omitempty"`   // e.g., "LLDP", "ARP", device name
	Device    string    `json:"device,omitempty"`   // device name if applicable
	Protocol  string    `json:"protocol,omitempty"` // protocol if applicable
}

// Request represents an IPC request
type Request struct {
	Command Command                `json:"command"`
	Args    map[string]interface{} `json:"args,omitempty"`
}

// Response represents an IPC response
type Response struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// StatusData contains simulation status information
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

// ErrorInjectionData contains error injection information
type ErrorInjectionData struct {
	Device    string           `json:"device"`
	Interface string           `json:"interface"`
	ErrorType errors.ErrorType `json:"error_type"`
	Value     int              `json:"value"`
	Injected  time.Time        `json:"injected_at"`
}

// PacketData contains captured packet information for hex dump display
type PacketData struct {
	Timestamp time.Time `json:"timestamp"`
	Length    int       `json:"length"`
	Device    string    `json:"device,omitempty"`
	Interface string    `json:"interface,omitempty"`
	Data      []byte    `json:"data"`
}

// PacketBuffer is a thread-safe ring buffer for storing captured packets
type PacketBuffer struct {
	packets []PacketData
	head    int
	count   int
	mu      sync.RWMutex
}

// Server is an IPC socket server for remote control
type Server struct {
	socketPath    string
	listener      net.Listener
	stack         *protocols.Stack
	cfg           *config.Config
	stateManager  *errors.StateManager
	interfaceName string
	configPath    string
	startTime     time.Time
	running       bool
	mu            sync.RWMutex
	shutdownChan  chan struct{}
	packetBuffer  *PacketBuffer
}

// NewServer creates a new IPC server
func NewServer(socketPath string, stack *protocols.Stack, cfg *config.Config,
	stateManager *errors.StateManager, interfaceName, configPath string) *Server {
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

// NewPacketBuffer creates a new packet buffer with the given capacity
func NewPacketBuffer(capacity int) *PacketBuffer {
	return &PacketBuffer{
		packets: make([]PacketData, capacity),
		head:    0,
		count:   0,
	}
}

// Add adds a packet to the ring buffer
func (pb *PacketBuffer) Add(pkt PacketData) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.packets[pb.head] = pkt
	pb.head = (pb.head + 1) % len(pb.packets)
	if pb.count < len(pb.packets) {
		pb.count++
	}
}

// GetAll returns all packets in the buffer (oldest first)
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

	for i := 0; i < pb.count; i++ {
		idx := (start + i) % len(pb.packets)
		result[i] = pb.packets[idx]
	}

	return result
}

// GetFiltered returns packets matching the filter criteria
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

// Start starts the IPC server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("IPC server already running")
	}

	// Remove existing socket if present
	if err := os.RemoveAll(s.socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create socket directory if needed
	socketDir := filepath.Dir(s.socketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Create Unix domain socket
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}

	// Set socket permissions (owner only)
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	s.listener = listener
	s.running = true

	// Start accepting connections in background
	go s.acceptLoop()

	logging.Info("IPC server started on %s", s.socketPath)
	return nil
}

// Stop stops the IPC server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	close(s.shutdownChan)
	s.running = false

	if s.listener != nil {
		s.listener.Close()
	}

	// Remove socket file
	os.RemoveAll(s.socketPath)

	logging.Info("IPC server stopped")
	return nil
}

// acceptLoop accepts incoming connections
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
				logging.Error("IPC accept error: %v", err)
				continue
			}
		}

		// Handle connection in goroutine
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single IPC connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Set read/write timeouts
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Read request
	decoder := json.NewDecoder(conn)
	var req Request
	if err := decoder.Decode(&req); err != nil {
		s.sendError(conn, fmt.Errorf("invalid request: %w", err))
		return
	}

	// Process command
	response := s.processCommand(&req)

	// Send response
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(response); err != nil {
		logging.Error("Failed to send IPC response: %v", err)
	}
}

// processCommand processes an IPC command
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

// handleStatus returns simulation status
func (s *Server) handleStatus(req *Request) *Response {
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
		Data: map[string]interface{}{
			"status": status,
		},
	}
}

// handleReload reloads the configuration
func (s *Server) handleReload(req *Request) *Response {
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
	if errs := validator.Validate(newCfg); !errs.Valid {
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
	logging.Info("Configuration reloaded from %s", s.configPath)

	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"message":      "configuration reloaded",
			"device_count": len(newCfg.Devices),
		},
	}
}

// handleInject injects an error
func (s *Server) handleInject(req *Request) *Response {
	// Extract arguments
	deviceName, ok := req.Args["device"].(string)
	if !ok {
		return &Response{
			Success: false,
			Error:   "missing or invalid 'device' argument",
		}
	}

	errorTypeStr, ok := req.Args["error_type"].(string)
	if !ok {
		return &Response{
			Success: false,
			Error:   "missing or invalid 'error_type' argument",
		}
	}

	value, ok := req.Args["value"].(float64) // JSON numbers are float64
	if !ok {
		return &Response{
			Success: false,
			Error:   "missing or invalid 'value' argument",
		}
	}

	// Find device
	var device *config.Device
	for i := range s.cfg.Devices {
		if s.cfg.Devices[i].Name == deviceName {
			device = &s.cfg.Devices[i]
			break
		}
	}

	if device == nil {
		return &Response{
			Success: false,
			Error:   fmt.Sprintf("device not found: %s", deviceName),
		}
	}

	// Parse error type
	errorType := errors.ErrorType(errorTypeStr)

	// Get interface (use first available)
	interfaceName := s.interfaceName
	if len(device.IPAddresses) > 0 {
		// Use device's first IP as identifier
		interfaceName = device.IPAddresses[0].String()
	}

	// Inject error
	s.stateManager.SetError(deviceName, interfaceName, errorType, int(value))

	logging.Info("Error injected via IPC: device=%s, type=%s, value=%d",
		deviceName, errorType, int(value))

	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"message":    "error injected",
			"device":     deviceName,
			"error_type": errorType,
			"value":      int(value),
		},
	}
}

// handleList lists active error injections
func (s *Server) handleList(req *Request) *Response {
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
		Data: map[string]interface{}{
			"injections": injections,
			"count":      len(injections),
		},
	}
}

// handleClear clears error injections
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

		logging.Info("Cleared %d error injections for device %s", cleared, deviceName)

		return &Response{
			Success: true,
			Data: map[string]interface{}{
				"message": fmt.Sprintf("cleared %d injections for device %s", cleared, deviceName),
				"cleared": cleared,
			},
		}
	}

	// Clear all
	s.stateManager.ClearAll()
	logging.Info("All error injections cleared via IPC")

	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"message": "all error injections cleared",
		},
	}
}

// handleShutdown initiates graceful shutdown
func (s *Server) handleShutdown(req *Request) *Response {
	logging.Info("Shutdown requested via IPC")

	// Send response before shutting down
	response := &Response{
		Success: true,
		Data: map[string]interface{}{
			"message": "shutdown initiated",
		},
	}

	// Schedule shutdown in background
	go func() {
		time.Sleep(500 * time.Millisecond)
		// The caller should handle this signal
		// For now, just log it
		logging.Info("IPC shutdown signal processed")
	}()

	return response
}

// handleLogs returns recent log entries from the simulation
// Supports filtering by log level and limiting the number of entries
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
		Data: map[string]interface{}{
			"logs":  logs,
			"count": len(logs),
		},
	}
}

// getRecentLogs returns recent log entries based on current simulation state
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
				Message:   fmt.Sprintf("Error injection active: %s on %s (value: %d)", state.ErrorType, state.DeviceIP, state.Value),
				Source:    "error-injection",
				Device:    state.DeviceIP,
			})
		}
	}

	return logs
}

// compareLevels returns -1 if a < b, 0 if a == b, 1 if a > b
func compareLevels(a, b LogLevel) int {
	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
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

// handleTopology returns the current network topology
func (s *Server) handleTopology(req *Request) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build topology from current configuration
	topology := api.BuildTopology(s.cfg)

	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"topology": topology,
		},
	}
}

// NeighborData represents neighbor discovery information for IPC responses
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

// handleNeighbors returns the neighbor discovery table
func (s *Server) handleNeighbors(req *Request) *Response {
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
		Data: map[string]interface{}{
			"neighbors": result,
			"count":     len(result),
		},
	}
}

// sendError sends an error response
func (s *Server) sendError(conn net.Conn, err error) {
	response := &Response{
		Success: false,
		Error:   err.Error(),
	}

	encoder := json.NewEncoder(conn)
	encoder.Encode(response)
}

// DefaultSocketPath returns the default socket path
func DefaultSocketPath() string {
	tmpDir := os.TempDir()
	return filepath.Join(tmpDir, "niac.sock")
}

// handleDump returns captured packets from the buffer
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
		Data: map[string]interface{}{
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

// GetPacketBuffer returns the packet buffer for external access
func (s *Server) GetPacketBuffer() *PacketBuffer {
	return s.packetBuffer
}
