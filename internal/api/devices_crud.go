package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// handleDeviceCreate creates a new device.
// FIX #272: Uses write lock for entire read-modify-write to prevent races.
func (s *Server) handleDeviceCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	var req DeviceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Failed to parse request", nil)

		return
	}

	// Validate required fields
	if req.Hostname == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			"hostname is required", []ErrorDetail{{Field: "hostname", Issue: "required"}})

		return
	}

	// FIX #272: Hold write lock for entire read-modify-write operation
	s.configMu.Lock()
	cfg := s.cfg.Config
	if cfg == nil {
		s.configMu.Unlock()
		writeError(w, r, http.StatusBadRequest, "config_not_found", "No configuration loaded", nil)

		return
	}

	// Check if device already exists
	for _, dev := range cfg.Devices {
		if dev.Name == req.Hostname {
			s.configMu.Unlock()
			writeError(w, r, http.StatusConflict, "device_exists",
				fmt.Sprintf("Device '%s' already exists", req.Hostname), nil)

			return
		}
	}

	// Create device from request
	newDevice, err := createDeviceFromRequest(req)
	if err != nil {
		s.configMu.Unlock()
		writeError(w, r, http.StatusBadRequest, "device_creation_failed", err.Error(), nil)

		return
	}

	// Add to config and save
	newCfg := *cfg
	newCfg.Devices = append(newCfg.Devices, *newDevice)
	s.configMu.Unlock()

	if saveErr := s.saveConfig(&newCfg); saveErr != nil {
		writeError(w, r, http.StatusInternalServerError, "save_failed", "Failed to save configuration", nil)

		return
	}

	// Broadcast change via SSE
	if s.sseHub != nil {
		s.sseHub.BroadcastLog("info", "Device created: "+req.Hostname)
	}

	resp := deviceToResponse(newDevice, true, false)

	// FIX #286: Wrap response in mutation format { success, device, message }
	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, map[string]any{
		"success": true,
		"device":  resp,
		"message": fmt.Sprintf("Device '%s' created successfully", req.Hostname),
	})
}

// findDeviceIndex finds the index of a device by hostname, returns -1 if not found.
func findDeviceIndex(devices []config.Device, hostname string) int {
	for i, dev := range devices {
		if dev.Name == hostname {
			return i
		}
	}

	return -1
}

// updateDeviceFromYAML updates a device from raw YAML content.
func updateDeviceFromYAML(rawYAML, hostname string) (*config.Device, error) {
	if validateErr := validateYAMLInput(rawYAML); validateErr != nil {
		return nil, validateErr
	}

	return parseDeviceFromYAML(rawYAML, hostname)
}

// applyPartialDeviceUpdate applies partial updates to a device.
func applyPartialDeviceUpdate(dev *config.Device, req DeviceUpdateRequest) error {
	if req.Type != "" {
		dev.Type = req.Type
	}

	if req.MAC != "" {
		mac, parseErr := parseMAC(req.MAC)
		if parseErr != nil {
			return fmt.Errorf("invalid_mac: %w", parseErr)
		}

		dev.MACAddress = mac
	}

	if req.IP != "" {
		ip, parseErr := parseIP(req.IP)
		if parseErr != nil {
			return fmt.Errorf("invalid_ip: %w", parseErr)
		}

		dev.IPAddresses = []net.IP{ip}
	}

	return nil
}

// handleDeviceUpdate updates an existing device.
func (s *Server) handleDeviceUpdate(w http.ResponseWriter, r *http.Request, hostname string) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	var req DeviceUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Failed to parse request", nil)
		return
	}

	// FIX: Hold write lock for entire read-modify-write to prevent TOCTOU race
	s.configMu.Lock()
	cfg := s.cfg.Config
	if cfg == nil {
		s.configMu.Unlock()
		writeError(w, r, http.StatusNotFound, "config_not_found", "No configuration loaded", nil)
		return
	}

	deviceIdx := findDeviceIndex(cfg.Devices, hostname)
	if deviceIdx == -1 {
		s.configMu.Unlock()
		writeError(w, r, http.StatusNotFound, "device_not_found",
			fmt.Sprintf("Device '%s' not found", hostname), nil)
		return
	}

	newCfg := *cfg
	newCfg.Devices = append([]config.Device(nil), cfg.Devices...)

	if req.RawYAML != "" {
		updatedDevice, parseErr := updateDeviceFromYAML(req.RawYAML, hostname)
		if parseErr != nil {
			s.configMu.Unlock()
			writeError(w, r, http.StatusBadRequest, "parse_failed", parseErr.Error(), nil)
			return
		}

		newCfg.Devices[deviceIdx] = *updatedDevice
	} else {
		if err := applyPartialDeviceUpdate(&newCfg.Devices[deviceIdx], req); err != nil {
			s.configMu.Unlock()
			errMsg := err.Error()

			switch {
			case strings.HasPrefix(errMsg, "invalid_mac:"):
				writeError(w, r, http.StatusBadRequest, "invalid_mac", strings.TrimPrefix(errMsg, "invalid_mac: "), nil)
			case strings.HasPrefix(errMsg, "invalid_ip:"):
				writeError(w, r, http.StatusBadRequest, "invalid_ip", strings.TrimPrefix(errMsg, "invalid_ip: "), nil)
			default:
				writeError(w, r, http.StatusBadRequest, "update_failed", errMsg, nil)
			}

			return
		}
	}
	s.configMu.Unlock()

	if err := s.saveConfig(&newCfg); err != nil {
		writeError(w, r, http.StatusInternalServerError, "save_failed", "Failed to save configuration", nil)
		return
	}

	if s.sseHub != nil {
		s.sseHub.BroadcastLog("info", "Device updated: "+hostname)
	}

	resp := deviceToResponse(&newCfg.Devices[deviceIdx], true, false)
	// FIX #286: Wrap response in mutation format { success, device, message }
	s.writeJSON(w, map[string]any{
		"success": true,
		"device":  resp,
		"message": fmt.Sprintf("Device '%s' updated successfully", hostname),
	})
}

// handleDeviceDelete deletes a device.
func (s *Server) handleDeviceDelete(w http.ResponseWriter, r *http.Request, hostname string) {
	// FIX: Hold write lock for entire read-modify-write to prevent TOCTOU race
	s.configMu.Lock()
	cfg := s.cfg.Config
	if cfg == nil {
		s.configMu.Unlock()
		writeError(w, r, http.StatusNotFound, "config_not_found", "No configuration loaded", nil)
		return
	}

	// Find and remove device
	newCfg := *cfg
	found := false

	newDevices := make([]config.Device, 0, len(cfg.Devices))

	for _, dev := range cfg.Devices {
		if dev.Name == hostname {
			found = true
			continue
		}
		newDevices = append(newDevices, dev)
	}

	if !found {
		s.configMu.Unlock()
		writeError(w, r, http.StatusNotFound, "device_not_found",
			fmt.Sprintf("Device '%s' not found", hostname), nil)
		return
	}

	newCfg.Devices = newDevices
	s.configMu.Unlock()

	err := s.saveConfig(&newCfg)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "save_failed", "Failed to save configuration", nil)
		return
	}

	// Broadcast change via SSE
	if s.sseHub != nil {
		s.sseHub.BroadcastLog("info", "Device deleted: "+hostname)
	}

	// FIX #286: Include success field in delete response
	s.writeJSON(w, map[string]any{
		"success":  true,
		"status":   "deleted",
		"hostname": hostname,
		"message":  fmt.Sprintf("Device '%s' deleted successfully", hostname),
	})
}

// saveConfig saves the config to file and updates the server state.
func (s *Server) saveConfig(cfg *config.Config) error {
	// Serialize to YAML
	yamlContent, err := serializeConfigToYAML(cfg)
	if err != nil {
		s.logger.Error("[API] Failed to serialize config", "error", err)

		return err
	}

	// Write to file
	if writeErr := s.writeConfigFile(yamlContent); writeErr != nil {
		s.logger.Error("[API] Failed to write config file", "error", writeErr)

		return writeErr
	}

	// Update in-memory config
	s.replaceConfig(cfg)

	// Apply config if handler is set
	if s.cfg.ApplyConfig != nil {
		applyErr := s.cfg.ApplyConfig(cfg)
		if applyErr != nil {
			s.logger.Warn("[API] Failed to apply config", "error", applyErr)
			// Don't fail - file is saved, runtime may be restarted
		}
	}

	return nil
}

// createDeviceFromRequest creates a new Device from a DeviceCreateRequest.
func createDeviceFromRequest(req DeviceCreateRequest) (*config.Device, error) {
	dev := &config.Device{
		Name: req.Hostname,
		Type: req.Type,
	}

	if req.MAC != "" {
		mac, err := parseMAC(req.MAC)
		if err != nil {
			return nil, err
		}

		dev.MACAddress = mac
	}

	if req.IP != "" {
		ip, err := parseIP(req.IP)
		if err != nil {
			return nil, err
		}

		dev.IPAddresses = []net.IP{ip}
	}

	if req.RawYAML != "" {
		parsed, err := parseDeviceFromYAML(req.RawYAML, req.Hostname)
		if err != nil {
			return nil, err
		}

		return parsed, nil
	}

	return dev, nil
}
