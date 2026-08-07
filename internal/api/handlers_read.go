package api

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/csrf"

	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
	"github.com/MustardSeedNetworks/niac-go/internal/storage"
)

// handleStats serves whichever session is selected. Prefer
// GET /api/v1/sessions/{id}/stats, which names the session it means instead of
// depending on selection.
func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	payload, ok := s.selectedStatsPayload()
	if !ok {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)

		return
	}

	s.writeJSON(w, payload)
}

// getDeviceProtocols extracts the list of enabled protocols for a device.
func getDeviceProtocols(dev *config.Device) []string {
	protos := make([]string, 0, protocolCapacity)

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

	return protos
}

// ipAddressesToStrings converts a slice of [net.IP] to string representations.
func ipAddressesToStrings(ips []net.IP) []string {
	result := make([]string, 0, len(ips))
	for _, ip := range ips {
		result = append(result, ip.String())
	}

	return result
}

// deviceSummary builds the JSON view of a device shared by /api/v1/devices and
// /api/v1/segments: name/type/ips/protocols, plus MAC + vendor/model when the
// vendor-template metadata is present (otherwise that YAML would be invisible to
// operators browsing the device or segment lists).
func deviceSummary(dev *config.Device) map[string]any {
	entry := map[string]any{
		"name":      dev.Name,
		"type":      dev.Type,
		"ips":       ipAddressesToStrings(dev.IPAddresses),
		"protocols": getDeviceProtocols(dev),
	}
	if len(dev.MACAddress) > 0 {
		entry["mac"] = dev.MACAddress.String()
	}
	if len(dev.Properties) > 0 {
		entry["properties"] = dev.Properties
		if v, ok := dev.Properties["vendor"]; ok && v != "" {
			entry["vendor"] = v
		}
		if m, ok := dev.Properties["model"]; ok && m != "" {
			entry["model"] = m
		}
	}

	return entry
}

func (s *Server) handleDevices(w http.ResponseWriter, _ *http.Request) {
	cfg := s.currentConfig()
	if cfg == nil {
		s.writeJSON(w, []map[string]any{})

		return
	}

	devices := make([]map[string]any, 0, len(cfg.Devices))
	for i := range cfg.Devices {
		devices = append(devices, deviceSummary(&cfg.Devices[i]))
	}

	s.writeJSON(w, devices)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Storage == nil {
		s.writeJSON(w, []storage.RunRecord{})

		return
	}

	history, err := s.cfg.Storage.ListRuns(historyListLimit)
	if err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		s.logger.ErrorContext(r.Context(), "[API] Failed to list run history", "error", err)
		writeError(w, r, http.StatusInternalServerError, "storage_error",
			"Failed to retrieve run history", nil)

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
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		s.logger.ErrorContext(r.Context(), "[API] Failed to read config", "error", err)
		writeError(w, r, status, "config_read_failed",
			"Failed to read configuration", nil)

		return
	}

	s.writeJSON(w, doc)
}

func (s *Server) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	if s.configPath() == "" {
		http.Error(w, "config path not available", http.StatusBadRequest)
		return
	}

	content, ok := s.parseConfigUpdateRequest(w, r)
	if !ok {
		return
	}

	newCfg, valid := s.validateConfigContent(w, r, content)
	if !valid {
		return
	}

	s.configMutationMu.Lock()
	defer s.configMutationMu.Unlock()

	if !s.authorizeConfigReplacement(w, r, newCfg) {
		return
	}

	if !s.applyAndSaveConfig(w, r, newCfg, content) {
		return
	}

	doc, status, err := s.readConfigDocument()
	if err != nil {
		s.logger.ErrorContext(r.Context(), "[API] Failed to read updated config", "error", err)
		writeError(w, r, status, "config_read_failed",
			"Configuration updated but failed to retrieve", nil)
		return
	}
	s.writeJSON(w, doc)
}

func (s *Server) authorizeConfigReplacement(w http.ResponseWriter, r *http.Request, cfg *config.Config) bool {
	if !s.authorizeConfigEntitlements(w, r, cfg) {
		return false
	}
	if runtimeErr := config.ValidateRuntimeRequirements(cfg); runtimeErr != nil {
		writeError(w, r, http.StatusBadRequest, "runtime_requirements_unmet",
			"Configuration runtime requirements are not met",
			[]ErrorDetail{{Field: "ssh.passwordEnv", Issue: runtimeErr.Error()}})
		return false
	}
	return true
}

func (s *Server) authorizeConfigEntitlements(w http.ResponseWriter, r *http.Request, cfg *config.Config) bool {
	switch err := ValidateConfigDeviceCount(cfg); {
	case errors.Is(err, ErrSimulationDeviceLimitExceeded):
		writeError(w, r, http.StatusBadRequest, "device_limit_reached",
			"Configuration exceeds the maximum supported device count", nil)
	case err != nil:
		writeError(w, r, http.StatusInternalServerError, "config_authorization_failed",
			"Failed to authorize configuration", nil)
	default:
		return true
	}
	return false
}

func (s *Server) parseConfigUpdateRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	var req struct {
		Content string `json:"content"`
	}
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return "", false
	}
	if strings.TrimSpace(req.Content) == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Configuration content is required", nil)
		return "", false
	}
	return req.Content, true
}

func (s *Server) validateConfigContent(
	w http.ResponseWriter, r *http.Request, content string,
) (*config.Config, bool) {
	newCfg, err := config.LoadYAMLBytes([]byte(content))
	if err != nil {
		s.logger.ErrorContext(r.Context(), "[API] Config validation failed", "error", err)

		line, msg := parseYAMLError(err)
		writeError(w, r, http.StatusBadRequest, "config_invalid", "Configuration validation failed",
			[]ErrorDetail{{Issue: msg, Line: line}})

		return nil, false
	}
	return newCfg, true
}

func (s *Server) applyAndSaveConfig(
	w http.ResponseWriter, r *http.Request, newCfg *config.Config, content string,
) bool {
	prevCfg := s.currentConfig()
	if s.cfg.ApplyConfig != nil {
		if err := s.cfg.ApplyConfig(newCfg); err != nil {
			s.logger.ErrorContext(r.Context(), "[API] Failed to apply config", "error", err)
			writeError(w, r, http.StatusInternalServerError, "config_apply_failed",
				"Failed to apply configuration", nil)
			return false
		}
	}

	if err := s.writeConfigFile(content); err != nil {
		if s.cfg.ApplyConfig != nil && prevCfg != nil {
			_ = s.cfg.ApplyConfig(prevCfg)
		}
		s.logger.ErrorContext(r.Context(), "[API] Failed to write config file", "error", err)
		writeError(w, r, http.StatusInternalServerError, "config_write_failed",
			"Failed to save configuration", nil)
		return false
	}
	s.replaceConfig(newCfg)
	return true
}

func (s *Server) handleReplay(w http.ResponseWriter, r *http.Request) {
	// FEATURE #132: Graceful degradation when replay engine is unavailable
	if s.cfg.Replay == nil {
		writeError(
			w,
			r,
			http.StatusServiceUnavailable,
			"replay_unavailable",
			"PCAP replay functionality is not available in this mode. Start niac with a configuration to enable replay.",
			nil,
		)

		return
	}

	switch r.Method {
	case http.MethodGet:
		s.writeJSON(w, s.cfg.Replay.Status())
	case http.MethodPost:
		// SECURITY FIX #97: Enforce request body size limit for PCAP uploads.
		// Bug #1e: use MaxPCAPUploadBodySize (accounts for base64 + JSON
		// envelope overhead), not the raw-pcap MaxPCAPUploadSize — see the
		// doc comment on MaxPCAPUploadBodySize in server.go.
		var req ReplayRequest
		if !decodeJSONStrict(w, r, &req, MaxPCAPUploadBodySize) {
			return
		}

		// SECURITY FIX MEDIUM-3: Comprehensive validation
		if validationErrors := validateReplayRequest(req); len(validationErrors) > 0 {
			writeError(w, r, http.StatusBadRequest, "validation_failed",
				"Replay request validation failed", validationErrors)

			return
		}

		prepared, err := s.prepareReplayRequest(req)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "replay_preparation_failed",
				"Failed to prepare replay request", nil)

			return
		}

		state, err := s.cfg.Replay.Start(prepared)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "replay_start_failed",
				"Failed to start replay", nil)

			return
		}

		s.writeJSON(w, state)
	case http.MethodDelete:
		state, err := s.cfg.Replay.Stop()
		if err != nil {
			// SECURITY FIX MEDIUM-6: Don't expose internal error details
			s.logger.ErrorContext(r.Context(), "[API] Failed to stop replay", "error", err)
			writeError(w, r, http.StatusInternalServerError, "replay_stop_failed",
				"Failed to stop replay", nil)

			return
		}

		s.writeJSON(w, state)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")

	// SECURITY FIX MEDIUM-3: Validate query parameter
	allowedKinds := []string{"", "snmp", "config", "pcap", "walks", "pcaps"}
	if err := validateQueryParam("kind", kind, allowedKinds); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_parameter",
			"Invalid query parameter", []ErrorDetail{*err})

		return
	}

	entries, err := s.collectFiles(kind)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "file_collection_failed",
			"Failed to collect files", nil)

		return
	}

	s.writeJSON(w, entries)
}

func (s *Server) handleTopology(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, s.currentTopology())
}

func (s *Server) handleTopologyExport(w http.ResponseWriter, r *http.Request) {
	// Method gating (GET-only) is enforced declaratively by the route registry.
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	// SECURITY FIX MEDIUM-3: Validate format parameter
	allowedFormats := []string{"json", "graphml", "dot"}
	if err := validateQueryParam("format", format, allowedFormats); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_parameter",
			"Invalid format parameter", []ErrorDetail{*err})

		return
	}

	graph := s.currentTopology()

	// Note: format is validated above, so only json/graphml/dot can reach here
	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=\"topology.json\"")
		s.writeJSON(w, graph)

	case "graphml":
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Disposition", "attachment; filename=\"topology.graphml\"")
		_, _ = fmt.Fprint(w, graph.ExportGraphML())

	case "dot":
		w.Header().Set("Content-Type", "text/vnd.graphviz")
		w.Header().Set("Content-Disposition", "attachment; filename=\"topology.dot\"")
		_, _ = fmt.Fprint(w, graph.ExportDOT())
	}
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, map[string]string{"version": s.cfg.Version})
}

// handleBuildVersion returns full build information for deployment verification.
// This endpoint is intentionally public (no auth) to allow health checks and
// deployment validation scripts to verify the running version.
func (s *Server) handleBuildVersion(w http.ResponseWriter, _ *http.Request) {
	const shortCommitLen = 7
	commit := s.cfg.Commit
	commitShort := commit
	if len(commit) > shortCommitLen {
		commitShort = commit[:shortCommitLen]
	}

	s.writeJSON(w, map[string]any{
		"version":        s.cfg.Version,
		"commit":         commitShort,
		"commitFull":     commit,
		"buildTime":      s.cfg.BuildTime,
		"releaseTrain":   s.cfg.ReleaseTrain,
		"uiBuildHash":    s.cfg.UIBuildHash,
		"goVersion":      runtime.Version(),
		"platform":       runtime.GOOS + "/" + runtime.GOARCH,
		"tlsFingerprint": s.tlsFingerprintForResponse(),
	})
}

func (s *Server) handleNeighbors(w http.ResponseWriter, _ *http.Request) {
	// SECURITY FIX #161: Thread-safe access to Stack
	stack := s.currentStack()
	if stack == nil {
		http.Error(w, "no simulation running", http.StatusServiceUnavailable)

		return
	}

	neighbors := stack.GetNeighbors()
	if neighbors == nil {
		neighbors = []protocols.NeighborRecord{}
	}

	s.writeJSON(w, neighbors)
}

func (s *Server) handleInterfaces(w http.ResponseWriter, r *http.Request) {
	// Method gating (GET-only) is enforced declaratively by the route registry.
	// Check for filter parameter
	filter := r.URL.Query().Get("filter")

	// SECURITY FIX MEDIUM-3: Validate filter parameter
	allowedFilters := []string{"", "usable", "all"}
	if err := validateQueryParam("filter", filter, allowedFilters); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_parameter",
			"Invalid filter parameter", []ErrorDetail{*err})

		return
	}

	// Get available network interfaces from pcap
	var ifaces []capture.PcapInterface
	var err error

	if filter == "usable" {
		ifaces, err = capture.GetUsableInterfaces()
	} else {
		ifaces, err = capture.GetAllInterfaces()
	}

	if err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		s.logger.ErrorContext(r.Context(), "[API] Failed to list interfaces", "error", err)
		writeError(w, r, http.StatusInternalServerError, "interface_list_failed",
			"Failed to retrieve network interfaces", nil)

		return
	}

	type interfaceInfo struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Addresses   []string `json:"addresses"`
		Current     bool     `json:"current"`
	}

	// SECURITY FIX #161: Thread-safe access to Interface
	currentIface := s.currentInterface()

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
			Current:     iface.Name == currentIface,
		})
	}

	s.writeJSON(w, map[string]any{
		"interfaces":        result,
		"current_interface": currentIface,
	})
}

func (s *Server) handleRuntime(w http.ResponseWriter, _ *http.Request) {
	// Method gating (GET-only) is enforced declaratively by the route registry.
	// SECURITY FIX #161: Thread-safe access to all config fields
	s.configMu.RLock()
	stack := s.cfg.Stack
	cfg := s.cfg.Config
	iface := s.cfg.Interface
	cfgPath := s.cfg.ConfigPath
	s.configMu.RUnlock()

	if stack == nil {
		// Return a JSON envelope rather than the plain-text "no
		// simulation running" message every other endpoint formerly
		// used. Generic JSON clients (the UI, scripted curl, jq
		// pipelines) couldn't parse the text response and were
		// erroring on a valid not-running state. HTTP 200 + running:
		// false matches the contract the simulation status endpoint
		// already exposes.
		s.writeJSON(w, map[string]any{
			"running":        false,
			"version":        s.cfg.Version,
			"uptime_seconds": time.Since(s.startTime).Seconds(),
		})

		return
	}

	stats := stack.GetStats()

	runtimeInfo := map[string]any{
		"running":          true, // API server is running
		"interface":        iface,
		"config_path":      cfgPath,
		"version":          s.cfg.Version,
		"device_count":     0,
		"packets_sent":     stats.PacketsSent,
		"packets_received": stats.PacketsReceived,
		"uptime_seconds":   time.Since(s.startTime).Seconds(),
	}

	if cfg != nil {
		runtimeInfo["device_count"] = len(cfg.Devices)
		runtimeInfo["config_name"] = filepath.Base(cfgPath)
	}

	s.writeJSON(w, runtimeInfo)
}

func (s *Server) handleCSRFToken(w http.ResponseWriter, r *http.Request) {
	// Method gating (GET-only) is enforced declaratively by the route registry.
	// #1257: mint or retrieve a per-session CSRF token keyed by the
	// caller's bearer (or the loopback bypass key on no-token paths).
	// Same caller fetching twice within the 24h expiry gets the same
	// value, so the UI can cache it.
	token, err := s.csrf.GetOrCreate(csrf.SessionKeyFromRequest(r))
	if err != nil {
		s.logger.ErrorContext(r.Context(), "[API] CSRF token mint failed", "error", err)
		http.Error(w, "csrf token unavailable", http.StatusInternalServerError)

		return
	}
	s.writeJSON(w, map[string]string{
		"token": token,
	})
}
