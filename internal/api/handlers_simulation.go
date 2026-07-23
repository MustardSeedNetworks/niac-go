package api

import (
	"errors"
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

var (
	// ErrRoutedLabsLicenseRequired indicates that a routed or SSH configuration
	// was rejected inside the simulation start transaction.
	ErrRoutedLabsLicenseRequired = errors.New("routed virtual labs require a license grant")
	// ErrUnlimitedDevicesLicenseRequired indicates that a config exceeds the Free device cap.
	ErrUnlimitedDevicesLicenseRequired = errors.New("device count requires an unlimited devices grant")
	// ErrSimulationDeviceLimitExceeded indicates that a config exceeds NIAC's absolute device ceiling.
	ErrSimulationDeviceLimitExceeded = errors.New("simulation exceeds the absolute device limit")
)

// SimulationEntitlements captures paid grants evaluated for one atomic start.
type SimulationEntitlements struct {
	RoutedLabs       bool
	UnlimitedDevices bool
}

// ValidateConfigEntitlements applies the license and absolute-size policy used
// by both simulation starts and whole-config replacement.
func ValidateConfigEntitlements(cfg *config.Config, entitlements SimulationEntitlements) error {
	if cfg.DeviceCount() > MaxDeviceCount {
		return ErrSimulationDeviceLimitExceeded
	}
	if cfg.DeviceCount() > FreeTierDeviceCount && !entitlements.UnlimitedDevices {
		return ErrUnlimitedDevicesLicenseRequired
	}
	if configRequiresRoutedLabs(cfg) && !entitlements.RoutedLabs {
		return ErrRoutedLabsLicenseRequired
	}
	return nil
}

func configRequiresRoutedLabs(cfg *config.Config) bool {
	if len(cfg.Networks) > 0 || len(cfg.Attachments) > 0 {
		return true
	}
	for _, segment := range cfg.NormalizedSegments() {
		for index := range segment.Devices {
			ssh := segment.Devices[index].SSHConfig
			if ssh != nil && ssh.Enabled {
				return true
			}
		}
	}
	return false
}

func (s *Server) simulationEntitlements() SimulationEntitlements {
	entitlements := SimulationEntitlements{}
	if s.license != nil {
		entitlements.RoutedLabs = s.license.HasFeature("routed_labs")
		entitlements.UnlimitedDevices = s.license.HasFeature("unlimited_devices")
	}
	return entitlements
}

func (s *Server) handleSimulation(w http.ResponseWriter, r *http.Request) {
	if s.daemon == nil {
		http.Error(
			w,
			"Simulation control is only available in daemon mode. Start NIAC with 'niac daemon' command.",
			http.StatusNotImplemented,
		)
		return
	}

	switch r.Method {
	case http.MethodGet:
		status := s.daemon.GetStatus()
		s.writeJSON(w, status)
	case http.MethodPost:
		s.handleSimulationStart(w, r)
	case http.MethodDelete:
		s.handleSimulationStop(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSimulationPreflight compiles a simulation request without changing runtime state.
func (s *Server) handleSimulationPreflight(w http.ResponseWriter, r *http.Request) {
	if s.daemon == nil {
		writeError(w, r, http.StatusNotImplemented, "daemon_required", "Daemon mode is required", nil)
		return
	}
	var req SimulationRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}
	if validationErrors := validateSimulationForPreflight(req); len(validationErrors) > 0 {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Validation failed", validationErrors)
		return
	}
	report, err := s.daemon.PreflightSimulation(req)
	if err != nil {
		if writeManagedConfigPathError(w, r, err) {
			return
		}
		s.logger.ErrorContext(r.Context(), "[API] Simulation preflight failed", "error", err)
		writeError(w, r, http.StatusBadRequest, "preflight_failed", "Simulation preflight failed", nil)
		return
	}
	s.writeJSON(w, report)
}

// handleSimulationStart processes POST requests to start a simulation.
func (s *Server) handleSimulationStart(w http.ResponseWriter, r *http.Request) {
	var req SimulationRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}

	if err := validateSimulationStartRequest(req); err != nil {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "Validation failed", err)
		return
	}

	if validationErrors := validateSimulationRequest(req); len(validationErrors) > 0 {
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			"Simulation request validation failed", validationErrors)
		return
	}
	entitlements := s.simulationEntitlements()
	if err := s.daemon.StartSimulation(req, entitlements); err != nil {
		s.handleSimulationStartError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, s.daemon.GetStatus())
}

func (s *Server) handleSimulationStartError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrRoutedLabsLicenseRequired):
		s.writeFeatureGate(w, r, "routed_labs",
			"Routed virtual labs require the Pro tier. "+defaultUpgradeMessage)
	case errors.Is(err, ErrUnlimitedDevicesLicenseRequired):
		s.writeFeatureGate(w, r, "unlimited_devices",
			"This simulation exceeds the Free tier device cap. "+defaultUpgradeMessage)
	case errors.Is(err, ErrSimulationDeviceLimitExceeded):
		writeError(w, r, http.StatusBadRequest, "device_limit_reached",
			"Simulation exceeds the maximum supported device count", nil)
	case errors.Is(err, config.ErrSSHPasswordUnavailable):
		writeError(w, r, http.StatusBadRequest, "runtime_requirements_unmet",
			"Configuration runtime requirements are not met",
			[]ErrorDetail{{Field: "ssh.passwordEnv", Issue: err.Error()}})
	case writeManagedConfigPathError(w, r, err):
	default:
		// SECURITY: Keep internal start failures in the daemon log, not the response.
		s.logger.ErrorContext(r.Context(), "[API] Failed to start simulation", "error", err)
		writeError(w, r, http.StatusInternalServerError, "simulation_start_failed",
			"Failed to start simulation", nil)
	}
}

func writeManagedConfigPathError(w http.ResponseWriter, r *http.Request, err error) bool {
	if !errors.Is(err, config.ErrPathOutsideManagedRoots) {
		return false
	}
	writeError(w, r, http.StatusBadRequest, "validation_failed", "Validation failed", []ErrorDetail{{
		Field: "config_path", Issue: "configuration must be selected from NIAC-managed storage",
		Value: "[redacted]",
	}})
	return true
}

// handleSimulationStop processes DELETE requests to stop a simulation.
func (s *Server) handleSimulationStop(w http.ResponseWriter, r *http.Request) {
	if err := s.daemon.StopSimulation(); err != nil {
		// SECURITY FIX MEDIUM-6: Don't expose internal error details
		s.logger.ErrorContext(r.Context(), "[API] Failed to stop simulation", "error", err)
		writeError(w, r, http.StatusInternalServerError, "simulation_stop_failed",
			"Failed to stop simulation", nil)
		return
	}
	s.writeJSON(w, map[string]string{"status": "stopped"})
}
