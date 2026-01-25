package ipc

import (
	"fmt"
	"time"

	"github.com/krisarmstrong/niac-go/internal/api"
	apperr "github.com/krisarmstrong/niac-go/internal/apperr"
	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/protocols"
)

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

// handleInject injects an error.
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
			Error:   "device not found: " + deviceName,
		}
	}

	// Parse error type
	errorType := apperr.ErrorType(errorTypeStr)

	// Get interface (use first available)
	interfaceName := s.interfaceName
	if len(device.IPAddresses) > 0 {
		// Use device's first IP as identifier
		interfaceName = device.IPAddresses[0].String()
	}

	// Inject error
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

// handleTopology returns the current network topology.
func (s *Server) handleTopology(_ *Request) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build topology from current configuration
	topology := api.BuildTopology(s.cfg)

	return &Response{
		Success: true,
		Data: map[string]any{
			"topology": topology,
		},
	}
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
