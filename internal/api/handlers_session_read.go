package api

import (
	"net/http"
	"path/filepath"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// Read surfaces for one named session. Each takes the session resolved by the
// dispatcher, so none of them can reach process-wide runtime state.
//
// A session that exists but has no stack yet is reported as empty rather than
// as an error: the session is real, it simply is not serving anything yet.

func (s *Server) handleSessionTopology(w http.ResponseWriter, r *http.Request, session sessionRuntime) {
	if !requireGet(w, r) {
		return
	}
	s.writeJSON(w, session.topology())
}

func (s *Server) handleSessionDevices(w http.ResponseWriter, r *http.Request, session sessionRuntime) {
	if !requireGet(w, r) {
		return
	}
	cfg := session.config()
	if cfg == nil {
		s.writeJSON(w, []map[string]any{})
		return
	}
	devices := make([]map[string]any, 0, len(cfg.Devices))
	for index := range cfg.Devices {
		devices = append(devices, deviceSummary(&cfg.Devices[index]))
	}
	s.writeJSON(w, devices)
}

func (s *Server) handleSessionSegments(w http.ResponseWriter, r *http.Request, session sessionRuntime) {
	if !requireGet(w, r) {
		return
	}
	cfg := session.config()
	if cfg == nil {
		s.writeJSON(w, []segmentResponse{})
		return
	}
	s.writeJSON(w, buildSegmentResponses(cfg))
}

func (s *Server) handleSessionNeighbors(w http.ResponseWriter, r *http.Request, session sessionRuntime) {
	if !requireGet(w, r) {
		return
	}
	stack := session.stack()
	if stack == nil {
		s.writeJSON(w, []protocols.NeighborRecord{})
		return
	}
	neighbors := stack.GetNeighbors()
	if neighbors == nil {
		neighbors = []protocols.NeighborRecord{}
	}
	s.writeJSON(w, neighbors)
}

func (s *Server) handleSessionBehaviors(w http.ResponseWriter, r *http.Request, session sessionRuntime) {
	if !requireGet(w, r) {
		return
	}
	stack := session.stack()
	if stack == nil {
		s.writeJSON(w, behavior.Status{State: "idle"})
		return
	}
	s.writeJSON(w, stack.BehaviorStatus())
}

// sessionInterfaceResponse names the device an interface belongs to, which the
// per-device response does not need but a whole-session listing does.
type sessionInterfaceResponse struct {
	Device      string `json:"device"`
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	AdminStatus string `json:"adminStatus,omitempty"`
	OperStatus  string `json:"operStatus,omitempty"`
}

// handleSessionInterfaces reports the interfaces of the session's simulated
// devices. The unscoped /api/v1/interfaces reports the host's capture NICs,
// which is a different thing entirely and stays where it is.
func (s *Server) handleSessionInterfaces(w http.ResponseWriter, r *http.Request, session sessionRuntime) {
	if !requireGet(w, r) {
		return
	}
	cfg := session.config()
	if cfg == nil {
		s.writeJSON(w, []sessionInterfaceResponse{})
		return
	}
	interfaces := make([]sessionInterfaceResponse, 0)
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		for _, iface := range device.Interfaces {
			interfaces = append(interfaces, sessionInterfaceResponse{
				Device:      device.Name,
				Name:        iface.Name,
				Type:        iface.Type,
				AdminStatus: iface.AdminStatus,
				OperStatus:  iface.OperStatus,
			})
		}
	}
	s.writeJSON(w, interfaces)
}

func (s *Server) handleSessionStats(w http.ResponseWriter, r *http.Request, session sessionRuntime) {
	if !requireGet(w, r) {
		return
	}
	payload, ok := s.sessionStatsPayload(session)
	if !ok {
		writeError(w, r, http.StatusServiceUnavailable, "simulation_not_running",
			"This session is not serving traffic", nil)
		return
	}
	s.writeJSON(w, payload)
}

func (s *Server) handleSessionRuntime(w http.ResponseWriter, r *http.Request, session sessionRuntime) {
	if !requireGet(w, r) {
		return
	}
	stack := session.stack()
	if stack == nil {
		s.writeJSON(w, map[string]any{
			"running":        false,
			"sessionId":      session.id,
			"version":        s.cfg.Version,
			"uptime_seconds": time.Since(s.startTime).Seconds(),
		})
		return
	}
	stats := stack.GetStats()
	deviceCount := 0
	if cfg := session.config(); cfg != nil {
		deviceCount = cfg.DeviceCount()
	}
	runtimeInfo := map[string]any{
		"running":          true,
		"sessionId":        session.id,
		"version":          s.cfg.Version,
		"interface":        session.iface(),
		"config_path":      session.configPath(),
		"device_count":     deviceCount,
		"packets_sent":     stats.PacketsSent,
		"packets_received": stats.PacketsReceived,
		"uptime_seconds":   time.Since(s.startTime).Seconds(),
	}
	if path := session.configPath(); path != "" {
		runtimeInfo["config_name"] = filepath.Base(path)
	}
	s.writeJSON(w, runtimeInfo)
}
