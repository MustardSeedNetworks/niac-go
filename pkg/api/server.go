package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/capture"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/errors"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
	"github.com/krisarmstrong/niac-go/pkg/storage"
)

const (
	// MaxRequestBodySize is the maximum size for API request bodies (1MB)
	MaxRequestBodySize = 1 << 20 // 1MB
)

// logRequest logs HTTP requests for debugging
func logRequest(r *http.Request) {
	log.Printf("[API] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
}

// addSecurityHeaders adds security headers to all HTTP responses
func addSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	// Don't add HSTS as it may not be HTTPS
}

// AlertConfig controls basic threshold-based alerting.
type AlertConfig struct {
	PacketsThreshold uint64 `json:"packets_threshold"`
	WebhookURL       string `json:"webhook_url"`
}

// ReplayRequest represents a packet replay request.
type ReplayRequest struct {
	File       string  `json:"file"`
	LoopMs     int     `json:"loop_ms"`
	Scale      float64 `json:"scale"`
	InlineData string  `json:"data,omitempty"`
	Uploaded   bool    `json:"-"`
}

// ReplayState reports the current replay status.
type ReplayState struct {
	Running   bool      `json:"running"`
	File      string    `json:"file"`
	LoopMs    int       `json:"loop_ms"`
	Scale     float64   `json:"scale"`
	StartedAt time.Time `json:"started_at,omitempty"`
}

// FileEntry represents a discovered file (pcap, walk, etc.).
type FileEntry struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	SizeBytes int64     `json:"size_bytes"`
	Modified  time.Time `json:"modified_at"`
}

// ReplayManager controls PCAP playback from the API server.
type ReplayManager interface {
	Status() ReplayState
	Start(ReplayRequest) (ReplayState, error)
	Stop() (ReplayState, error)
}

// ServerConfig defines API server options.
type ServerConfig struct {
	Addr        string
	MetricsAddr string
	Token       string
	Stack       *protocols.Stack
	Config      *config.Config
	ConfigPath  string
	Storage     *storage.Storage
	Interface   string
	Version     string
	Topology    Topology
	Alert       AlertConfig
	ApplyConfig func(*config.Config) error
	Replay      ReplayManager
}

// SimulationRequest represents a request to start a simulation
type SimulationRequest struct {
	Interface  string `json:"interface"`
	ConfigPath string `json:"config_path,omitempty"`
	ConfigData string `json:"config_data,omitempty"`
}

// SimulationStatus represents the current simulation status
type SimulationStatus struct {
	Running       bool      `json:"running"`
	Interface     string    `json:"interface,omitempty"`
	ConfigPath    string    `json:"config_path,omitempty"`
	ConfigName    string    `json:"config_name,omitempty"`
	DeviceCount   int       `json:"device_count"`
	StartedAt     time.Time `json:"started_at,omitempty"`
	UptimeSeconds float64   `json:"uptime_seconds"`
}

// DaemonController interface for daemon mode operations
type DaemonController interface {
	StartSimulation(req SimulationRequest) error
	StopSimulation() error
	GetStatus() SimulationStatus
}

// Server exposes the REST API, metrics endpoint, and Web UI.
type Server struct {
	cfg           ServerConfig
	httpServer    *http.Server
	metricsServer *http.Server
	alertStop     chan struct{}
	lastAlert     uint64
	alertMu       sync.RWMutex
	configMu      sync.RWMutex
	daemon        DaemonController // Optional: only set in daemon mode
	startTime     time.Time        // Track server start time for uptime
}

// NewServer returns a configured API server.
func NewServer(cfg ServerConfig) *Server {
	return &Server{
		cfg:       cfg,
		startTime: time.Now(),
	}
}

// Start boots the HTTP listeners.
func (s *Server) Start() error {
	if s.cfg.Stack == nil || s.cfg.Config == nil {
		return fmt.Errorf("api server requires stack and config references")
	}

	if s.cfg.Addr != "" {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/stats", s.auth(s.handleStats))
		mux.HandleFunc("/api/v1/devices", s.auth(s.handleDevices))
		mux.HandleFunc("/api/v1/history", s.auth(s.handleHistory))
		mux.HandleFunc("/api/v1/config", s.auth(s.handleConfig))
		mux.HandleFunc("/api/v1/replay", s.auth(s.handleReplay))
		mux.HandleFunc("/api/v1/alerts", s.auth(s.handleAlerts))
		mux.HandleFunc("/api/v1/files", s.auth(s.handleFiles))
		mux.HandleFunc("/api/v1/topology", s.auth(s.handleTopology))
		mux.HandleFunc("/api/v1/errors", s.auth(s.handleErrors))
		mux.HandleFunc("/api/v1/interfaces", s.auth(s.handleInterfaces))
		mux.HandleFunc("/api/v1/runtime", s.auth(s.handleRuntime))
		mux.HandleFunc("/api/v1/simulation", s.auth(s.handleSimulation))
		mux.HandleFunc("/api/v1/version", s.auth(s.handleVersion))
		mux.HandleFunc("/api/v1/neighbors", s.auth(s.handleNeighbors))
		mux.HandleFunc("/metrics", s.handleMetrics)
		mux.HandleFunc("/", s.auth(s.serveSPA()))

		s.httpServer = &http.Server{
			Addr:    s.cfg.Addr,
			Handler: mux,
		}

		go func() {
			if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("api server stopped: %v", err)
			}
		}()
	}

	if s.cfg.MetricsAddr != "" && s.cfg.MetricsAddr != s.cfg.Addr {
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", s.handleMetrics)

		s.metricsServer = &http.Server{
			Addr:    s.cfg.MetricsAddr,
			Handler: mux,
		}

		go func() {
			if err := s.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("metrics server stopped: %v", err)
			}
		}()
	}

	s.updateAlertConfig(s.cfg.Alert)
	return nil
}

// Shutdown stops the HTTP listeners.
func (s *Server) Shutdown(ctx context.Context) error {
	// Acquire lock before closing channel to prevent race with updateAlertConfig
	s.alertMu.Lock()
	if s.alertStop != nil {
		close(s.alertStop)
		s.alertStop = nil
	}
	s.alertMu.Unlock()

	if s.metricsServer != nil {
		_ = s.metricsServer.Shutdown(ctx)
	}

	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// SetDaemonController sets the daemon controller (for daemon mode)
func (s *Server) SetDaemonController(daemon DaemonController) {
	s.daemon = daemon
}

// UpdateSimulation updates the server with simulation components (for daemon mode)
func (s *Server) UpdateSimulation(stack *protocols.Stack, cfg *config.Config, configPath string, iface string, replay ReplayManager) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.cfg.Stack = stack
	s.cfg.Config = cfg
	s.cfg.ConfigPath = configPath
	s.cfg.Interface = iface
	s.cfg.Replay = replay
	s.cfg.Topology = BuildTopology(cfg)
}

// ClearSimulation clears simulation components (for daemon mode)
func (s *Server) ClearSimulation() {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.cfg.Stack = nil
	s.cfg.Config = nil
	s.cfg.Replay = nil
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Add security headers to all responses
		addSecurityHeaders(w)

		if s.cfg.Token == "" {
			next(w, r)
			return
		}

		// Only accept Authorization header (not query parameters for security)
		token := r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}

		if token != s.cfg.Token {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (s *Server) serveSPA() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		requestPath := strings.TrimPrefix(r.URL.Path, "/")
		if requestPath == "" || strings.HasSuffix(r.URL.Path, "/") {
			requestPath = "index.html"
		}
		if strings.Contains(requestPath, "..") {
			http.NotFound(w, r)
			return
		}

		lookupPath := path.Join("ui", requestPath)
		data, err := fs.ReadFile(uiFS, lookupPath)
		if err != nil {
			data, err = fs.ReadFile(uiFS, path.Join("ui", "index.html"))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			requestPath = "index.html"
		}

		if ctype := mime.TypeByExtension(filepath.Ext(requestPath)); ctype != "" {
			w.Header().Set("Content-Type", ctype)
		} else if strings.HasSuffix(requestPath, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(requestPath, ".css") {
			w.Header().Set("Content-Type", "text/css")
		}

		http.ServeContent(w, r, requestPath, time.Time{}, bytes.NewReader(data))
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	stack := s.cfg.Stack
	cfg := s.cfg.Config
	s.configMu.RUnlock()

	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)
		return
	}

	stats := stack.GetStats()
	deviceCount := 0
	if cfg != nil {
		deviceCount = len(cfg.Devices)
	}
	payload := map[string]interface{}{
		"timestamp":    time.Now().UTC(),
		"interface":    s.cfg.Interface,
		"version":      s.cfg.Version,
		"device_count": deviceCount,
		"stack": map[string]uint64{
			"packets_sent":     stats.PacketsSent,
			"packets_received": stats.PacketsReceived,
			"arp_requests":     stats.ARPRequests,
			"arp_replies":      stats.ARPReplies,
			"icmp_requests":    stats.ICMPRequests,
			"icmp_replies":     stats.ICMPReplies,
			"dns_queries":      stats.DNSQueries,
			"dhcp_requests":    stats.DHCPRequests,
			"snmp_queries":     stats.SNMPQueries,
			"errors":           stats.Errors,
		},
	}
	s.writeJSON(w, payload)
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	cfg := s.currentConfig()
	if cfg == nil {
		s.writeJSON(w, []map[string]interface{}{})
		return
	}

	devices := make([]map[string]interface{}, 0, len(cfg.Devices))
	for _, dev := range cfg.Devices {
		ips := make([]string, 0, len(dev.IPAddresses))
		for _, ip := range dev.IPAddresses {
			ips = append(ips, ip.String())
		}

		protos := make([]string, 0, 8)
		if dev.SNMPConfig.Community != "" || dev.SNMPConfig.WalkFile != "" {
			protos = append(protos, "SNMP")
		}
		if dev.DHCPConfig != nil {
			protos = append(protos, "DHCP")
		}
		if dev.DNSConfig != nil {
			protos = append(protos, "DNS")
		}
		if dev.HTTPConfig != nil {
			protos = append(protos, "HTTP")
		}
		if dev.FTPConfig != nil {
			protos = append(protos, "FTP")
		}
		if dev.LLDPConfig != nil && dev.LLDPConfig.Enabled {
			protos = append(protos, "LLDP")
		}
		if dev.CDPConfig != nil && dev.CDPConfig.Enabled {
			protos = append(protos, "CDP")
		}

		devices = append(devices, map[string]interface{}{
			"name":      dev.Name,
			"type":      dev.Type,
			"ips":       ips,
			"protocols": protos,
		})
	}
	s.writeJSON(w, devices)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Storage == nil {
		s.writeJSON(w, []storage.RunRecord{})
		return
	}
	history, err := s.cfg.Storage.ListRuns(20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.writeJSON(w, history)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleConfigGet(w, r)
	case http.MethodPut, http.MethodPatch, http.MethodPost:
		s.handleConfigUpdate(w, r)
	default:
		w.Header().Set("Allow", "GET, PUT, PATCH, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	doc, status, err := s.readConfigDocument()
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	s.writeJSON(w, doc)
}

func (s *Server) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	if s.cfg.ConfigPath == "" {
		http.Error(w, "config path not available", http.StatusBadRequest)
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	newCfg, err := config.LoadYAMLBytes([]byte(req.Content))
	if err != nil {
		http.Error(w, fmt.Sprintf("config validation failed: %v", err), http.StatusBadRequest)
		return
	}

	prevCfg := s.currentConfig()
	if s.cfg.ApplyConfig != nil {
		if err := s.cfg.ApplyConfig(newCfg); err != nil {
			http.Error(w, fmt.Sprintf("failed to apply config: %v", err), http.StatusInternalServerError)
			return
		}
	}

	if err := s.writeConfigFile(req.Content); err != nil {
		if s.cfg.ApplyConfig != nil && prevCfg != nil {
			// Attempt rollback to previous config to avoid divergence.
			_ = s.cfg.ApplyConfig(prevCfg)
		}
		http.Error(w, fmt.Sprintf("failed to write config: %v", err), http.StatusInternalServerError)
		return
	}

	s.replaceConfig(newCfg)

	doc, status, err := s.readConfigDocument()
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	s.writeJSON(w, doc)
}

func (s *Server) handleReplay(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Replay == nil {
		http.Error(w, "replay control unavailable", http.StatusNotImplemented)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, s.cfg.Replay.Status())
	case http.MethodPost:
		var req ReplayRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}
		prepared, err := s.prepareReplayRequest(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid replay request: %v", err), http.StatusBadRequest)
			return
		}
		state, err := s.cfg.Replay.Start(prepared)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.writeJSON(w, state)
	case http.MethodDelete:
		state, err := s.cfg.Replay.Stop()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.writeJSON(w, state)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, s.getAlertConfig())
	case http.MethodPut, http.MethodPost:
		var req AlertConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}
		s.updateAlertConfig(req)
		s.writeJSON(w, s.getAlertConfig())
	default:
		w.Header().Set("Allow", "GET, PUT, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	entries, err := s.collectFiles(kind)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.writeJSON(w, entries)
}

func (s *Server) handleTopology(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, s.currentTopology())
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, map[string]string{"version": s.cfg.Version})
}

func (s *Server) handleNeighbors(w http.ResponseWriter, r *http.Request) {
	neighbors := s.cfg.Stack.GetNeighbors()
	if neighbors == nil {
		neighbors = []protocols.NeighborRecord{}
	}
	s.writeJSON(w, neighbors)
}

func (s *Server) handleErrors(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	stack := s.cfg.Stack
	s.configMu.RUnlock()

	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)
		return
	}

	errorMgr := stack.GetErrorManager()
	if errorMgr == nil {
		http.Error(w, "error manager not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// List available error types and current active errors
		errorTypes := []map[string]string{
			{"type": "FCS Errors", "description": "Frame Check Sequence errors (0-100)"},
			{"type": "Packet Discards", "description": "Dropped packets (0-100)"},
			{"type": "Interface Errors", "description": "Generic interface errors (0-100)"},
			{"type": "High Utilization", "description": "Interface bandwidth saturation (0-100%)"},
			{"type": "High CPU", "description": "Device CPU load (0-100%)"},
			{"type": "High Memory", "description": "Device memory usage (0-100%)"},
			{"type": "High Disk", "description": "Device disk usage (0-100%)"},
		}

		activeErrors := errorMgr.GetAllStates()
		s.writeJSON(w, map[string]interface{}{
			"available_types": errorTypes,
			"active_errors":   activeErrors,
		})

	case http.MethodPost, http.MethodPut:
		// Inject or update error
		var req struct {
			DeviceIP  string `json:"device_ip"`
			Interface string `json:"interface"`
			ErrorType string `json:"error_type"`
			Value     int    `json:"value"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}

		// Validate inputs
		if req.DeviceIP == "" {
			http.Error(w, "device_ip is required", http.StatusBadRequest)
			return
		}
		if req.Interface == "" {
			http.Error(w, "interface is required", http.StatusBadRequest)
			return
		}
		if req.ErrorType == "" {
			http.Error(w, "error_type is required", http.StatusBadRequest)
			return
		}
		if req.Value < 0 || req.Value > 100 {
			http.Error(w, "value must be between 0 and 100", http.StatusBadRequest)
			return
		}

		// Inject error
		errorMgr.SetError(req.DeviceIP, req.Interface, errors.ErrorType(req.ErrorType), req.Value)

		s.writeJSON(w, map[string]interface{}{
			"success":    true,
			"message":    "error injected successfully",
			"device_ip":  req.DeviceIP,
			"interface":  req.Interface,
			"error_type": req.ErrorType,
			"value":      req.Value,
		})

	case http.MethodDelete:
		// Clear errors
		query := r.URL.Query()
		deviceIP := query.Get("device_ip")
		iface := query.Get("interface")

		if deviceIP == "" && iface == "" {
			// Clear all errors
			errorMgr.ClearAll()
			s.writeJSON(w, map[string]interface{}{
				"success": true,
				"message": "all errors cleared",
			})
		} else if deviceIP != "" && iface != "" {
			// Clear specific device/interface error
			errorMgr.ClearError(deviceIP, iface)
			s.writeJSON(w, map[string]interface{}{
				"success":   true,
				"message":   "error cleared",
				"device_ip": deviceIP,
				"interface": iface,
			})
		} else {
			http.Error(w, "both device_ip and interface are required, or omit both to clear all", http.StatusBadRequest)
		}

	default:
		w.Header().Set("Allow", "GET, POST, PUT, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get available network interfaces from pcap
	ifaces, err := capture.GetAllInterfaces()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list interfaces: %v", err), http.StatusInternalServerError)
		return
	}

	type interfaceInfo struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Addresses   []string `json:"addresses"`
		Current     bool     `json:"current"`
	}

	result := make([]interfaceInfo, 0, len(ifaces))
	for _, iface := range ifaces {
		addrs := make([]string, 0, len(iface.Addresses))
		for _, addr := range iface.Addresses {
			addrs = append(addrs, addr.IP.String())
		}
		result = append(result, interfaceInfo{
			Name:        iface.Name,
			Description: iface.Description,
			Addresses:   addrs,
			Current:     iface.Name == s.cfg.Interface,
		})
	}

	s.writeJSON(w, map[string]interface{}{
		"interfaces":        result,
		"current_interface": s.cfg.Interface,
	})
}

func (s *Server) handleRuntime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if stack is available
	s.configMu.RLock()
	stack := s.cfg.Stack
	cfg := s.cfg.Config
	s.configMu.RUnlock()

	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)
		return
	}

	stats := stack.GetStats()

	runtime := map[string]interface{}{
		"running":          true, // API server is running
		"interface":        s.cfg.Interface,
		"config_path":      s.cfg.ConfigPath,
		"version":          s.cfg.Version,
		"device_count":     0,
		"packets_sent":     stats.PacketsSent,
		"packets_received": stats.PacketsReceived,
		"uptime_seconds":   time.Since(s.startTime).Seconds(),
	}

	if cfg != nil {
		runtime["device_count"] = len(cfg.Devices)
		runtime["config_name"] = filepath.Base(s.cfg.ConfigPath)
	}

	s.writeJSON(w, runtime)
}

func (s *Server) handleSimulation(w http.ResponseWriter, r *http.Request) {
	if s.daemon == nil {
		http.Error(w, "Simulation control is only available in daemon mode. Start NIAC with 'niac daemon' command.", http.StatusNotImplemented)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get simulation status
		status := s.daemon.GetStatus()
		s.writeJSON(w, status)

	case http.MethodPost:
		// Start simulation
		// Add request body size limit
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

		var req SimulationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}

		// Validate input
		if req.Interface == "" {
			http.Error(w, "interface is required", http.StatusBadRequest)
			return
		}

		if req.ConfigPath == "" && req.ConfigData == "" {
			http.Error(w, "either config_path or config_data must be provided", http.StatusBadRequest)
			return
		}

		if err := s.daemon.StartSimulation(req); err != nil {
			http.Error(w, fmt.Sprintf("failed to start simulation: %v", err), http.StatusInternalServerError)
			return
		}

		status := s.daemon.GetStatus()
		w.WriteHeader(http.StatusCreated)
		s.writeJSON(w, status)

	case http.MethodDelete:
		// Stop simulation
		if err := s.daemon.StopSimulation(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to stop simulation: %v", err), http.StatusInternalServerError)
			return
		}

		s.writeJSON(w, map[string]string{"status": "stopped"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	stack := s.cfg.Stack
	cfg := s.cfg.Config
	s.configMu.RUnlock()

	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)
		return
	}

	stats := stack.GetStats()
	deviceCount := 0
	if cfg != nil {
		deviceCount = len(cfg.Devices)
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "niac_packets_sent_total %d\n", stats.PacketsSent)
	fmt.Fprintf(w, "niac_packets_received_total %d\n", stats.PacketsReceived)
	fmt.Fprintf(w, "niac_snmp_queries_total %d\n", stats.SNMPQueries)
	fmt.Fprintf(w, "niac_errors_total %d\n", stats.Errors)
	fmt.Fprintf(w, "niac_devices_total %d\n", deviceCount)
}

func (s *Server) writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}

func (s *Server) alertLoop(stop <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cfg := s.getAlertConfig()
			if cfg.PacketsThreshold == 0 {
				continue
			}

			s.configMu.RLock()
			stack := s.cfg.Stack
			s.configMu.RUnlock()

			if stack == nil {
				continue
			}

			stats := stack.GetStats()
			total := stats.PacketsSent + stats.PacketsReceived
			if total >= cfg.PacketsThreshold {
				s.alertMu.Lock()
				if total != s.lastAlert {
					s.lastAlert = total
					go s.sendAlert(total)
				}
				s.alertMu.Unlock()
			}
		case <-stop:
			return
		}
	}
}

func (s *Server) sendAlert(total uint64) {
	log.Printf("alert: packet threshold exceeded (total=%d)", total)
	cfg := s.getAlertConfig()
	if cfg.WebhookURL == "" {
		return
	}

	body, _ := json.Marshal(map[string]interface{}{
		"type":        "packet_threshold",
		"threshold":   cfg.PacketsThreshold,
		"total":       total,
		"interface":   s.cfg.Interface,
		"triggeredAt": time.Now().UTC(),
	})

	req, err := http.NewRequest(http.MethodPost, cfg.WebhookURL, strings.NewReader(string(body)))
	if err != nil {
		log.Printf("alert webhook error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	if resp, err := client.Do(req); err != nil {
		log.Printf("alert webhook request failed: %v", err)
	} else {
		resp.Body.Close()
	}
}

func (s *Server) prepareReplayRequest(req ReplayRequest) (ReplayRequest, error) {
	if strings.TrimSpace(req.File) == "" && req.InlineData == "" {
		return req, fmt.Errorf("pcap file path or data is required")
	}

	if req.InlineData != "" {
		data, err := base64.StdEncoding.DecodeString(req.InlineData)
		if err != nil {
			return req, fmt.Errorf("decode replay data: %w", err)
		}
		path, err := s.writeUploadedFile(data)
		if err != nil {
			return req, err
		}
		req.File = path
		req.Uploaded = true
		req.InlineData = ""
		return req, nil
	}

	abs, err := filepath.Abs(req.File)
	if err != nil {
		return req, fmt.Errorf("resolve path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return req, fmt.Errorf("stat %s: %w", abs, err)
	}
	if info.IsDir() {
		return req, fmt.Errorf("%s is a directory", abs)
	}
	req.File = abs
	return req, nil
}

func (s *Server) writeUploadedFile(data []byte) (string, error) {
	dir := filepath.Join(os.TempDir(), "niac-replay")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create upload dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "upload-*.pcap")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tmp.Close()
	if _, err := tmp.Write(data); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("write upload: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("sync upload: %w", err)
	}
	return tmp.Name(), nil
}

func (s *Server) getAlertConfig() AlertConfig {
	s.alertMu.RLock()
	defer s.alertMu.RUnlock()
	return s.cfg.Alert
}

func (s *Server) updateAlertConfig(cfg AlertConfig) {
	s.alertMu.Lock()
	if s.alertStop != nil {
		close(s.alertStop)
		s.alertStop = nil
	}
	s.cfg.Alert = cfg
	s.lastAlert = 0
	var stopChan chan struct{}
	if cfg.PacketsThreshold > 0 {
		stopChan = make(chan struct{})
		s.alertStop = stopChan
	}
	s.alertMu.Unlock()

	if stopChan != nil {
		go s.alertLoop(stopChan)
	}
}

type configDocument struct {
	Path        string    `json:"path"`
	Filename    string    `json:"filename"`
	ModifiedAt  time.Time `json:"modified_at"`
	SizeBytes   int64     `json:"size_bytes"`
	DeviceCount int       `json:"device_count"`
	Content     string    `json:"content"`
}

func (s *Server) readConfigDocument() (*configDocument, int, error) {
	if s.cfg.ConfigPath == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("config path not available")
	}

	data, err := os.ReadFile(s.cfg.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, http.StatusNotFound, fmt.Errorf("config file %s not found", s.cfg.ConfigPath)
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("reading config: %w", err)
	}

	info, err := os.Stat(s.cfg.ConfigPath)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("stat config: %w", err)
	}

	cfg := s.currentConfig()
	deviceCount := 0
	if cfg != nil {
		deviceCount = len(cfg.Devices)
	}

	return &configDocument{
		Path:        s.cfg.ConfigPath,
		Filename:    filepath.Base(s.cfg.ConfigPath),
		ModifiedAt:  info.ModTime().UTC(),
		SizeBytes:   info.Size(),
		DeviceCount: deviceCount,
		Content:     string(data),
	}, http.StatusOK, nil
}

func (s *Server) writeConfigFile(content string) error {
	dir := filepath.Dir(s.cfg.ConfigPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".niac-config-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.cfg.ConfigPath); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

func (s *Server) currentConfig() *config.Config {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.cfg.Config
}

func (s *Server) currentTopology() Topology {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.cfg.Topology
}

func (s *Server) replaceConfig(cfg *config.Config) {
	s.configMu.Lock()
	s.cfg.Config = cfg
	s.cfg.Topology = BuildTopology(cfg)
	s.configMu.Unlock()
}

func (s *Server) collectFiles(kind string) ([]FileEntry, error) {
	var root string
	var exts []string

	switch kind {
	case "walks":
		root = s.resolveIncludePath()
		exts = []string{".walk"}
	case "pcaps":
		if s.cfg.ConfigPath != "" {
			root = filepath.Dir(s.cfg.ConfigPath)
		}
		exts = []string{".pcap", ".pcapng"}
	default:
		return nil, fmt.Errorf("unsupported file kind: %s", kind)
	}

	if root == "" {
		return []FileEntry{}, nil
	}

	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return []FileEntry{}, nil
	}

	var entries []FileEntry
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		match := false
		for _, allowed := range exts {
			if ext == allowed {
				match = true
				break
			}
		}
		if !match {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		entries = append(entries, FileEntry{
			Path:      absPath,
			Name:      filepath.Base(path),
			SizeBytes: info.Size(),
			Modified:  info.ModTime().UTC(),
		})
		if len(entries) >= 200 {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil && err != filepath.SkipDir {
		return nil, err
	}
	return entries, nil
}

func (s *Server) resolveIncludePath() string {
	cfg := s.currentConfig()
	if cfg == nil || cfg.IncludePath == "" {
		return ""
	}

	includePath := cfg.IncludePath
	if !filepath.IsAbs(includePath) && s.cfg.ConfigPath != "" {
		includePath = filepath.Join(filepath.Dir(s.cfg.ConfigPath), includePath)
	}

	if abs, err := filepath.Abs(includePath); err == nil {
		return abs
	}
	return includePath
}
