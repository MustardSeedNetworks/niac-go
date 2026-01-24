package api

import (
	"encoding/json"
	"net/http"

	"github.com/krisarmstrong/niac-go/internal/logging"
)

// Debug level constants for protocol debug level management.
const (
	debugLevelOff   = "OFF"
	debugLevelError = "ERROR"
	debugLevelWarn  = "WARN"
	debugLevelInfo  = "INFO"
	debugLevelDebug = "DEBUG"
	debugLevelTrace = "TRACE"

	debugIntError = 1
	debugIntInfo  = 2
	debugIntDebug = 3
	debugIntTrace = 4
)

// ProtocolDebugConfig represents a protocol's debug configuration.
type ProtocolDebugConfig struct {
	Protocol string `json:"protocol"`
	Level    string `json:"level"`
	Category string `json:"category"`
}

// ProtocolDebugLevelsResponse is the response for GET /api/v1/debug/levels.
type ProtocolDebugLevelsResponse struct {
	Protocols    []ProtocolDebugConfig `json:"protocols"`
	DefaultLevel string                `json:"default_level"`
}

// UpdateProtocolDebugLevelRequest represents a single protocol level update.
type UpdateProtocolDebugLevelRequest struct {
	Protocol string `json:"protocol"`
	Level    string `json:"level"`
}

// UpdateProtocolDebugLevelsRequest is the request for PUT /api/v1/debug/levels.
type UpdateProtocolDebugLevelsRequest struct {
	Protocols []UpdateProtocolDebugLevelRequest `json:"protocols"`
}

// ResetProtocolDebugLevelsResponse is the response for POST /api/v1/debug/levels/reset.
type ResetProtocolDebugLevelsResponse struct {
	Success   bool                  `json:"success"`
	Message   string                `json:"message"`
	Protocols []ProtocolDebugConfig `json:"protocols"`
}

// getProtocolCategory returns the category for a protocol name.
func getProtocolCategory(proto string) string {
	categories := map[string]string{
		"SNMP":    "monitoring",
		"LLDP":    "discovery",
		"CDP":     "discovery",
		"EDP":     "discovery",
		"FDP":     "discovery",
		"STP":     "switching",
		"ARP":     "switching",
		"IP":      "routing",
		"ICMP":    "routing",
		"IPv6":    "routing",
		"ICMPv6":  "routing",
		"UDP":     "routing",
		"TCP":     "routing",
		"OSPF":    "routing",
		"BGP":     "routing",
		"EIGRP":   "routing",
		"RIP":     "routing",
		"ISIS":    "routing",
		"VRRP":    "redundancy",
		"HSRP":    "redundancy",
		"GLBP":    "redundancy",
		"BFD":     "redundancy",
		"MPLS":    "routing",
		"PIM":     "multicast",
		"IGMP":    "multicast",
		"MSDP":    "multicast",
		"NetFlow": "monitoring",
		"DNS":     "routing",
		"DHCP":    "routing",
		"DHCPv6":  "routing",
		"HTTP":    "routing",
		"FTP":     "routing",
		"NetBIOS": "discovery",
	}

	cat := categories[proto]
	if cat == "" {
		return "routing"
	}

	return cat
}

// getKnownProtocols returns the list of all known protocols for enumeration.
func getKnownProtocols() []string {
	return []string{
		logging.ProtocolSNMP,
		logging.ProtocolLLDP,
		logging.ProtocolCDP,
		logging.ProtocolEDP,
		logging.ProtocolFDP,
		logging.ProtocolSTP,
		logging.ProtocolARP,
		logging.ProtocolIP,
		logging.ProtocolICMP,
		logging.ProtocolIPv6,
		logging.ProtocolICMPv6,
		logging.ProtocolUDP,
		logging.ProtocolTCP,
		logging.ProtocolDNS,
		logging.ProtocolDHCP,
		logging.ProtocolDHCPv6,
		logging.ProtocolHTTP,
		logging.ProtocolFTP,
		logging.ProtocolNetBIOS,
	}
}

// intToLevel converts an integer debug level to a string representation.
func intToLevel(level int) string {
	switch {
	case level <= 0:
		return debugLevelOff
	case level == debugIntError:
		return debugLevelError
	case level == debugIntInfo:
		return debugLevelInfo
	case level == debugIntDebug:
		return debugLevelDebug
	case level >= debugIntTrace:
		return debugLevelTrace
	default:
		return debugLevelOff
	}
}

// levelToInt converts a string debug level to an integer.
func levelToInt(level string) int {
	switch level {
	case debugLevelOff:
		return 0
	case debugLevelError, debugLevelWarn:
		return debugIntError
	case debugLevelInfo:
		return debugIntInfo
	case debugLevelDebug:
		return debugIntDebug
	case debugLevelTrace:
		return debugIntTrace
	default:
		return 0
	}
}

// handleDebugLevels routes debug level requests.
func (s *Server) handleDebugLevels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleDebugLevelsGet(w, r)
	case http.MethodPut:
		s.handleDebugLevelsUpdate(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed",
			"Method not allowed", nil)
	}
}

// handleDebugLevelsGet returns current protocol debug levels.
func (s *Server) handleDebugLevelsGet(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	stack := s.cfg.Stack
	s.configMu.RUnlock()

	if stack == nil {
		writeError(w, r, http.StatusServiceUnavailable, "no_simulation",
			"No simulation running", nil)

		return
	}

	debugConfig := stack.GetDebugConfig()
	if debugConfig == nil {
		writeError(w, r, http.StatusServiceUnavailable, "no_debug_config",
			"Debug configuration not available", nil)

		return
	}

	known := getKnownProtocols()
	protocols := make([]ProtocolDebugConfig, 0, len(known))

	for _, proto := range known {
		level := debugConfig.GetProtocolLevel(proto)
		protocols = append(protocols, ProtocolDebugConfig{
			Protocol: proto,
			Level:    intToLevel(level),
			Category: getProtocolCategory(proto),
		})
	}

	s.writeJSON(w, ProtocolDebugLevelsResponse{
		Protocols:    protocols,
		DefaultLevel: intToLevel(debugConfig.GetGlobal()),
	})
}

// handleDebugLevelsUpdate updates protocol debug levels.
func (s *Server) handleDebugLevelsUpdate(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	stack := s.cfg.Stack
	s.configMu.RUnlock()

	if stack == nil {
		writeError(w, r, http.StatusServiceUnavailable, "no_simulation",
			"No simulation running", nil)

		return
	}

	debugConfig := stack.GetDebugConfig()
	if debugConfig == nil {
		writeError(w, r, http.StatusServiceUnavailable, "no_debug_config",
			"Debug configuration not available", nil)

		return
	}

	var req UpdateProtocolDebugLevelsRequest

	body := http.MaxBytesReader(w, r.Body, MaxRequestBodySize)
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json",
			"Invalid JSON request body", nil)

		return
	}

	if len(req.Protocols) == 0 {
		writeError(w, r, http.StatusBadRequest, "empty_protocols",
			"At least one protocol level update is required", nil)

		return
	}

	// FIX #344: Validate protocols and levels
	knownProtos := make(map[string]bool, len(getKnownProtocols()))
	for _, p := range getKnownProtocols() {
		knownProtos[p] = true
	}
	validLevels := map[string]bool{
		debugLevelOff: true, debugLevelError: true, debugLevelWarn: true,
		debugLevelInfo: true, debugLevelDebug: true, debugLevelTrace: true,
	}

	for _, update := range req.Protocols {
		if update.Protocol == "" {
			continue
		}
		if !knownProtos[update.Protocol] {
			writeError(w, r, http.StatusBadRequest, "unknown_protocol",
				"Unknown protocol: "+update.Protocol, nil)
			return
		}
		if !validLevels[update.Level] {
			writeError(w, r, http.StatusBadRequest, "invalid_level",
				"Invalid debug level: "+update.Level, nil)
			return
		}
		debugConfig.SetProtocolLevel(update.Protocol, levelToInt(update.Level))
	}

	// Return updated state
	s.handleDebugLevelsGet(w, r)
}

// handleDebugLevelsReset resets all protocol debug levels to default.
func (s *Server) handleDebugLevelsReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed",
			"Method not allowed", nil)

		return
	}

	s.configMu.RLock()
	stack := s.cfg.Stack
	s.configMu.RUnlock()

	if stack == nil {
		writeError(w, r, http.StatusServiceUnavailable, "no_simulation",
			"No simulation running", nil)

		return
	}

	debugConfig := stack.GetDebugConfig()
	if debugConfig == nil {
		writeError(w, r, http.StatusServiceUnavailable, "no_debug_config",
			"Debug configuration not available", nil)

		return
	}

	// Reset all protocol levels by setting them to the global default
	known := getKnownProtocols()
	globalLevel := debugConfig.GetGlobal()

	for _, proto := range known {
		debugConfig.SetProtocolLevel(proto, globalLevel)
	}

	// Build response
	protocols := make([]ProtocolDebugConfig, 0, len(known))
	for _, proto := range known {
		protocols = append(protocols, ProtocolDebugConfig{
			Protocol: proto,
			Level:    intToLevel(globalLevel),
			Category: getProtocolCategory(proto),
		})
	}

	s.writeJSON(w, ResetProtocolDebugLevelsResponse{
		Success:   true,
		Message:   "All protocol debug levels reset to default",
		Protocols: protocols,
	})
}
