// Package ipc provides inter-process communication for NIAC remote control
package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/api"
	"github.com/krisarmstrong/niac-go/pkg/errors"
)

const (
	// DefaultTimeout is the default timeout for IPC operations
	DefaultTimeout = 5 * time.Second

	// EnvSocketPath is the environment variable for custom socket path
	EnvSocketPath = "NIAC_SOCKET_PATH"
)

// Client is an IPC client for communicating with the NIAC server
type Client struct {
	socketPath string
	timeout    time.Duration
}

// NewClient creates a new IPC client with the specified socket path.
// If socketPath is empty, DefaultSocketPath() is used.
func NewClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = DefaultSocketPath()
	}
	return &Client{
		socketPath: socketPath,
		timeout:    DefaultTimeout,
	}
}

// SetTimeout sets the timeout for IPC operations.
func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// SocketPath returns the socket path this client connects to.
func (c *Client) SocketPath() string {
	return c.socketPath
}

// SendCommand sends a command to the IPC server and returns the response.
// This is the core method that all convenience methods use.
func (c *Client) SendCommand(cmd Command, args map[string]interface{}) (*Response, error) {
	// Connect to socket
	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IPC socket: %w", err)
	}
	defer conn.Close()

	// Set deadline for the entire operation
	if err := conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, fmt.Errorf("failed to set connection deadline: %w", err)
	}

	// Create and send request
	req := Request{
		Command: cmd,
		Args:    args,
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(&req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	var resp Response
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &resp, nil
}

// GetStatus retrieves the current simulation status from the server.
func (c *Client) GetStatus() (*StatusData, error) {
	resp, err := c.SendCommand(CommandStatus, nil)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("status command failed: %s", resp.Error)
	}

	// Extract status from response data
	statusData, ok := resp.Data["status"]
	if !ok {
		return nil, fmt.Errorf("response missing status data")
	}

	// Convert map to StatusData struct
	statusBytes, err := json.Marshal(statusData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status data: %w", err)
	}

	var status StatusData
	if err := json.Unmarshal(statusBytes, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status data: %w", err)
	}

	return &status, nil
}

// Reload triggers a configuration reload on the server.
func (c *Client) Reload() error {
	resp, err := c.SendCommand(CommandReload, nil)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("reload command failed: %s", resp.Error)
	}

	return nil
}

// InjectError injects an error on the specified device.
// The errorType should be one of the ErrorType constants (e.g., "FCS Errors", "Packet Discards").
// The value represents the error rate or percentage depending on the error type.
func (c *Client) InjectError(device, errorType string, value int) error {
	args := map[string]interface{}{
		"device":     device,
		"error_type": errorType,
		"value":      value,
	}

	resp, err := c.SendCommand(CommandInject, args)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("inject command failed: %s", resp.Error)
	}

	return nil
}

// InjectErrorType is a convenience method that accepts an errors.ErrorType.
func (c *Client) InjectErrorType(device string, errorType errors.ErrorType, value int) error {
	return c.InjectError(device, string(errorType), value)
}

// ListInjections returns a list of all active error injections.
func (c *Client) ListInjections() ([]ErrorInjectionData, error) {
	resp, err := c.SendCommand(CommandList, nil)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("list command failed: %s", resp.Error)
	}

	// Extract injections from response data
	injectionsData, ok := resp.Data["injections"]
	if !ok {
		return nil, fmt.Errorf("response missing injections data")
	}

	// Convert to slice of ErrorInjectionData
	injectionsBytes, err := json.Marshal(injectionsData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal injections data: %w", err)
	}

	var injections []ErrorInjectionData
	if err := json.Unmarshal(injectionsBytes, &injections); err != nil {
		return nil, fmt.Errorf("failed to unmarshal injections data: %w", err)
	}

	return injections, nil
}

// ClearInjections clears error injections.
// If device is empty, all injections are cleared.
// If device is specified, only injections for that device are cleared.
func (c *Client) ClearInjections(device string) error {
	var args map[string]interface{}
	if device != "" {
		args = map[string]interface{}{
			"device": device,
		}
	}

	resp, err := c.SendCommand(CommandClear, args)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("clear command failed: %s", resp.Error)
	}

	return nil
}

// Shutdown requests a graceful shutdown of the NIAC server.
func (c *Client) Shutdown() error {
	resp, err := c.SendCommand(CommandShutdown, nil)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("shutdown command failed: %s", resp.Error)
	}

	return nil
}

// Ping checks if the server is reachable and responding.
// It does this by sending a status command and checking for a valid response.
func (c *Client) Ping() error {
	_, err := c.GetStatus()
	return err
}

// IsRunning returns true if the NIAC server is running and reachable.
func (c *Client) IsRunning() bool {
	status, err := c.GetStatus()
	if err != nil {
		return false
	}
	return status.Running
}

// GetDefaultSocketPath returns the default socket path, checking the
// NIAC_SOCKET_PATH environment variable first, then falling back to
// the temp directory.
func GetDefaultSocketPath() string {
	if path := os.Getenv(EnvSocketPath); path != "" {
		return path
	}
	return filepath.Join(os.TempDir(), "niac.sock")
}

// DefaultClient creates a new client with the default socket path.
func DefaultClient() *Client {
	return NewClient(GetDefaultSocketPath())
}

// DumpPackets retrieves captured packets from the server.
// Packets can be filtered by device name, interface name, and limited by count.
// Pass empty strings or 0 to skip filters.
func (c *Client) DumpPackets(device, iface string, count int) ([]PacketData, error) {
	args := make(map[string]interface{})
	if device != "" {
		args["device"] = device
	}
	if iface != "" {
		args["interface"] = iface
	}
	if count > 0 {
		args["count"] = count
	}

	resp, err := c.SendCommand(CommandDump, args)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("dump command failed: %s", resp.Error)
	}

	// Extract packets from response data
	packetsData, ok := resp.Data["packets"]
	if !ok {
		return nil, fmt.Errorf("response missing packets data")
	}

	// Handle nil packets (no packets captured)
	if packetsData == nil {
		return []PacketData{}, nil
	}

	// Convert to slice of PacketData
	packetsBytes, err := json.Marshal(packetsData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal packets data: %w", err)
	}

	var packets []PacketData
	if err := json.Unmarshal(packetsBytes, &packets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal packets data: %w", err)
	}

	return packets, nil
}

// GetTopology retrieves the current network topology from the server.
// Returns the topology graph with nodes (devices) and links (connections).
func (c *Client) GetTopology() (*api.Topology, error) {
	resp, err := c.SendCommand(CommandTopology, nil)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("topology command failed: %s", resp.Error)
	}

	// Extract topology from response data
	topologyData, ok := resp.Data["topology"]
	if !ok {
		return nil, fmt.Errorf("response missing topology data")
	}

	// Convert map to Topology struct
	topologyBytes, err := json.Marshal(topologyData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal topology data: %w", err)
	}

	var topology api.Topology
	if err := json.Unmarshal(topologyBytes, &topology); err != nil {
		return nil, fmt.Errorf("failed to unmarshal topology data: %w", err)
	}

	return &topology, nil
}

// GetLogs retrieves log entries from the server.
// The level parameter filters logs by minimum severity (debug, info, warn, error).
// The count parameter limits the number of logs returned.
func (c *Client) GetLogs(level string, count int) ([]LogEntry, error) {
	args := make(map[string]interface{})
	if level != "" {
		args["level"] = level
	}
	if count > 0 {
		args["count"] = count
	}

	resp, err := c.SendCommand(CommandLogs, args)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("logs command failed: %s", resp.Error)
	}

	// Extract logs from response data
	logsData, ok := resp.Data["logs"]
	if !ok {
		return nil, fmt.Errorf("response missing logs data")
	}

	// Handle nil logs
	if logsData == nil {
		return []LogEntry{}, nil
	}

	// Convert to slice of LogEntry
	logsBytes, err := json.Marshal(logsData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal logs data: %w", err)
	}

	var logs []LogEntry
	if err := json.Unmarshal(logsBytes, &logs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal logs data: %w", err)
	}

	return logs, nil
}

// LogSubscription represents an active log subscription for streaming logs
type LogSubscription struct {
	client   *Client
	level    string
	filter   string
	interval time.Duration
	stopCh   chan struct{}
	logCh    chan LogEntry
	errCh    chan error
}

// SubscribeLogs creates a subscription that polls for new log entries.
// The subscription runs in a background goroutine and sends logs to the returned channel.
// Call Stop() on the subscription to terminate it.
func (c *Client) SubscribeLogs(level, filter string, interval time.Duration) *LogSubscription {
	if interval < 100*time.Millisecond {
		interval = 500 * time.Millisecond
	}

	sub := &LogSubscription{
		client:   c,
		level:    level,
		filter:   filter,
		interval: interval,
		stopCh:   make(chan struct{}),
		logCh:    make(chan LogEntry, 100),
		errCh:    make(chan error, 1),
	}

	go sub.run()

	return sub
}

// Logs returns the channel for receiving log entries
func (s *LogSubscription) Logs() <-chan LogEntry {
	return s.logCh
}

// Errors returns the channel for receiving errors
func (s *LogSubscription) Errors() <-chan error {
	return s.errCh
}

// Stop terminates the log subscription
func (s *LogSubscription) Stop() {
	close(s.stopCh)
}

// run is the background goroutine that polls for logs
func (s *LogSubscription) run() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	defer close(s.logCh)

	// Track seen logs to avoid duplicates
	seenLogs := make(map[string]bool)

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			logs, err := s.client.GetLogs(s.level, 100)
			if err != nil {
				select {
				case s.errCh <- err:
				default:
				}
				continue
			}

			for _, log := range logs {
				// Create a unique key for this log entry
				key := fmt.Sprintf("%s:%s:%s", log.Timestamp.Format(time.RFC3339Nano), log.Level, log.Message)

				// Skip if we've seen this log before
				if seenLogs[key] {
					continue
				}

				// Apply text filter if specified
				if s.filter != "" && !matchesFilter(log.Message, s.filter) {
					continue
				}

				seenLogs[key] = true

				select {
				case s.logCh <- log:
				case <-s.stopCh:
					return
				}
			}

			// Clean up old entries from the seen map to prevent unbounded growth
			if len(seenLogs) > 1000 {
				seenLogs = make(map[string]bool)
			}
		}
	}
}

// matchesFilter checks if a message matches the given filter pattern
func matchesFilter(message, filter string) bool {
	// Simple substring match for now
	return len(filter) == 0 || containsIgnoreCase(message, filter)
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	sLower := make([]byte, len(s))
	substrLower := make([]byte, len(substr))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			sLower[i] = s[i] + 32
		} else {
			sLower[i] = s[i]
		}
	}
	for i := 0; i < len(substr); i++ {
		if substr[i] >= 'A' && substr[i] <= 'Z' {
			substrLower[i] = substr[i] + 32
		} else {
			substrLower[i] = substr[i]
		}
	}
	return bytesContains(sLower, substrLower)
}

// bytesContains checks if b contains sub
func bytesContains(b, sub []byte) bool {
	if len(sub) == 0 {
		return true
	}
	if len(b) < len(sub) {
		return false
	}
	for i := 0; i <= len(b)-len(sub); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			if b[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// GetNeighbors retrieves the neighbor discovery table from the server.
// This returns all discovered neighbors from LLDP, CDP, EDP, and FDP protocols.
func (c *Client) GetNeighbors() ([]NeighborData, error) {
	resp, err := c.SendCommand(CommandNeighbors, nil)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("neighbors command failed: %s", resp.Error)
	}

	// Extract neighbors from response data
	neighborsData, ok := resp.Data["neighbors"]
	if !ok {
		return nil, fmt.Errorf("response missing neighbors data")
	}

	// Handle nil neighbors (no neighbors discovered)
	if neighborsData == nil {
		return []NeighborData{}, nil
	}

	// Convert to slice of NeighborData
	neighborsBytes, err := json.Marshal(neighborsData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal neighbors data: %w", err)
	}

	var neighbors []NeighborData
	if err := json.Unmarshal(neighborsBytes, &neighbors); err != nil {
		return nil, fmt.Errorf("failed to unmarshal neighbors data: %w", err)
	}

	return neighbors, nil
}
