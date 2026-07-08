package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// handleDevicesV2 handles device CRUD operations
// Routes:
//
//	GET    /api/v1/config/devices           - List all devices
//	GET    /api/v1/config/devices/:id       - Get device by hostname
//	POST   /api/v1/config/devices           - Create new device
//	PUT    /api/v1/config/devices/:id       - Update device
//	DELETE /api/v1/config/devices/:id       - Delete device
//	DELETE /api/v1/config/devices           - Batch delete devices
//	POST   /api/v1/config/devices/:id/clone - Clone device
func (s *Server) handleDevicesV2(w http.ResponseWriter, r *http.Request) {
	// Parse the path to determine action
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/config/devices")
	path = strings.TrimPrefix(path, "/")

	// Split path into segments
	segments := strings.Split(path, "/")
	deviceID := ""
	action := ""

	if len(segments) > 0 && segments[0] != "" {
		deviceID = segments[0]
	}

	if len(segments) > 1 {
		action = segments[1]
	}

	switch {
	case r.Method == http.MethodGet && deviceID == "":
		// List all devices
		s.handleDeviceList(w, r)
	case r.Method == http.MethodGet && deviceID != "":
		// Get single device
		s.handleDeviceGet(w, r, deviceID)
	case r.Method == http.MethodPost && deviceID == "":
		// Create new device
		s.handleDeviceCreate(w, r)
	case r.Method == http.MethodPut && deviceID != "":
		// Update device
		s.handleDeviceUpdate(w, r, deviceID)
	case r.Method == http.MethodDelete && deviceID != "":
		// Delete device
		s.handleDeviceDelete(w, r, deviceID)
	case r.Method == http.MethodDelete && deviceID == "":
		// Batch delete devices
		s.handleDeviceBatchDelete(w, r)
	case r.Method == http.MethodPost && action == "clone":
		// Clone device
		s.handleDeviceClone(w, r, deviceID)
	default:
		w.Header().Set("Allow", "GET, POST, PUT, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDeviceList returns all devices with full details.
func (s *Server) handleDeviceList(w http.ResponseWriter, r *http.Request) {
	cfg := s.currentConfig()
	if cfg == nil {
		s.writeJSON(w, DeviceListResponse{Devices: []DeviceResponse{}, TotalCount: 0})

		return
	}

	// Check if details are requested
	includeDetails := r.URL.Query().Get("details") == "true"
	includeYAML := r.URL.Query().Get("yaml") == "true"

	devices := make([]DeviceResponse, 0, len(cfg.Devices))

	for _, dev := range cfg.Devices {
		resp := deviceToResponse(&dev, includeDetails, includeYAML)
		devices = append(devices, resp)
	}

	s.writeJSON(w, DeviceListResponse{
		Devices:    devices,
		TotalCount: len(devices),
	})
}

// handleDeviceGet returns a single device by hostname.
func (s *Server) handleDeviceGet(w http.ResponseWriter, r *http.Request, hostname string) {
	cfg := s.currentConfig()
	if cfg == nil {
		writeError(w, r, http.StatusNotFound, "config_not_found", "No configuration loaded", nil)

		return
	}

	for _, dev := range cfg.Devices {
		if dev.Name == hostname {
			resp := deviceToResponse(&dev, true, true)
			s.writeJSON(w, resp)

			return
		}
	}

	writeError(w, r, http.StatusNotFound, "device_not_found",
		fmt.Sprintf("Device '%s' not found", hostname), nil)
}

// handleDeviceCreate creates a new device.
func (s *Server) handleDeviceCreate(w http.ResponseWriter, r *http.Request) {
	var req DeviceCreateRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}

	if detail := validateHostname(req.Hostname); detail != nil {
		writeError(w, r, http.StatusBadRequest, "validation_failed", detail.Issue, []ErrorDetail{*detail})
		return
	}

	cfg, err := s.validateDeviceCreatePreconditions(w, r, req.Hostname)
	if err != nil {
		return // Error already written
	}

	newDevice, err := s.createAndSaveDevice(w, r, cfg, req)
	if err != nil {
		return // Error already written
	}

	if s.sseHub != nil {
		s.sseHub.BroadcastLog("info", "Device created: "+req.Hostname)
	}

	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, deviceToResponse(newDevice, true, false))
}

// validateDeviceCreatePreconditions checks config and device limits.

func (s *Server) handleDeviceUpdate(w http.ResponseWriter, r *http.Request, hostname string) {
	var req DeviceUpdateRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}

	cfg := s.currentConfig()
	if cfg == nil {
		writeError(w, r, http.StatusNotFound, "config_not_found", "No configuration loaded", nil)
		return
	}

	deviceIdx := findDeviceIndex(cfg.Devices, hostname)
	if deviceIdx == -1 {
		writeError(w, r, http.StatusNotFound, "device_not_found",
			fmt.Sprintf("Device '%s' not found", hostname), nil)
		return
	}

	newCfg := *deepCopyConfig(cfg)

	if req.RawYAML != "" {
		updatedDevice, parseErr := updateDeviceFromYAML(req.RawYAML, hostname)
		if parseErr != nil {
			line, msg := parseYAMLError(parseErr)
			writeError(w, r, http.StatusBadRequest, "parse_failed", parseErr.Error(),
				[]ErrorDetail{{Issue: msg, Line: line}})

			return
		}

		if !s.requireDeviceProtocolFeatures(w, r, updatedDevice) {
			return
		}

		newCfg.Devices[deviceIdx] = *updatedDevice
	} else {
		if err := applyPartialDeviceUpdate(&newCfg.Devices[deviceIdx], req); err != nil {
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

	if err := s.saveConfig(&newCfg); err != nil {
		writeError(w, r, http.StatusInternalServerError, "save_failed", "Failed to save configuration", nil)
		return
	}

	if s.sseHub != nil {
		s.sseHub.BroadcastLog("info", "Device updated: "+hostname)
	}

	resp := deviceToResponse(&newCfg.Devices[deviceIdx], true, false)
	s.writeJSON(w, resp)
}

// removeDeviceByHostname returns devices with the named device removed and
// whether it was found. It is the single code path shared by the single- and
// batch-delete handlers so both apply identical removal semantics.
func removeDeviceByHostname(devices []config.Device, hostname string) ([]config.Device, bool) {
	found := false

	newDevices := make([]config.Device, 0, len(devices))

	for _, dev := range devices {
		if dev.Name == hostname {
			found = true

			continue
		}

		newDevices = append(newDevices, dev)
	}

	return newDevices, found
}

// handleDeviceDelete deletes a single device.
func (s *Server) handleDeviceDelete(w http.ResponseWriter, r *http.Request, hostname string) {
	cfg := s.currentConfig()
	if cfg == nil {
		writeError(w, r, http.StatusNotFound, "config_not_found", "No configuration loaded", nil)

		return
	}

	newCfg := *deepCopyConfig(cfg)

	newDevices, found := removeDeviceByHostname(newCfg.Devices, hostname)
	if !found {
		writeError(w, r, http.StatusNotFound, "device_not_found",
			fmt.Sprintf("Device '%s' not found", hostname), nil)

		return
	}

	newCfg.Devices = newDevices

	err := s.saveConfig(&newCfg)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "save_failed", "Failed to save configuration", nil)

		return
	}

	// Broadcast change via SSE
	if s.sseHub != nil {
		s.sseHub.BroadcastLog("info", "Device deleted: "+hostname)
	}

	s.writeJSON(w, map[string]string{
		"status":   "deleted",
		"hostname": hostname,
	})
}

// handleDeviceBatchDelete deletes multiple devices in a single request,
// reporting a per-hostname result rather than failing the whole request when
// some hostnames don't exist. The config is persisted once at the end so the
// batch is a single atomic write instead of N sequential saves.
func (s *Server) handleDeviceBatchDelete(w http.ResponseWriter, r *http.Request) {
	var req DeviceBatchDeleteRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}

	if len(req.Hostnames) == 0 {
		writeError(w, r, http.StatusBadRequest, "validation_failed", "hostnames must not be empty", nil)

		return
	}

	if len(req.Hostnames) > MaxDeviceCount {
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			fmt.Sprintf("hostnames must not exceed %d entries", MaxDeviceCount), nil)

		return
	}

	cfg := s.currentConfig()
	if cfg == nil {
		writeError(w, r, http.StatusNotFound, "config_not_found", "No configuration loaded", nil)

		return
	}

	newCfg := *deepCopyConfig(cfg)

	results := make([]DeviceBatchDeleteResult, 0, len(req.Hostnames))
	deletedHostnames := make([]string, 0, len(req.Hostnames))

	deleted, failed := 0, 0

	for _, hostname := range req.Hostnames {
		newDevices, found := removeDeviceByHostname(newCfg.Devices, hostname)
		if !found {
			failed++

			results = append(results, DeviceBatchDeleteResult{
				Hostname: hostname,
				Success:  false,
				Error:    fmt.Sprintf("Device '%s' not found", hostname),
			})

			continue
		}

		newCfg.Devices = newDevices
		deleted++

		deletedHostnames = append(deletedHostnames, hostname)
		results = append(results, DeviceBatchDeleteResult{
			Hostname: hostname,
			Success:  true,
		})
	}

	if deleted > 0 {
		if err := s.saveConfig(&newCfg); err != nil {
			writeError(w, r, http.StatusInternalServerError, "save_failed", "Failed to save configuration", nil)

			return
		}

		if s.sseHub != nil {
			s.sseHub.BroadcastLog("info", "Devices deleted: "+strings.Join(deletedHostnames, ", "))
		}
	}

	s.writeJSON(w, DeviceBatchDeleteResponse{
		Results: results,
		Deleted: deleted,
		Failed:  failed,
	})
}

// handleDeviceClone clones an existing device.
func (s *Server) handleDeviceClone(w http.ResponseWriter, r *http.Request, hostname string) {
	var req DeviceCloneRequest
	if !decodeJSONStrict(w, r, &req, MaxRequestBodySize) {
		return
	}

	// SECURITY FIX #169: Validate new hostname format
	if detail := validateHostname(req.NewHostname); detail != nil {
		detail.Field = "new_hostname"
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			detail.Issue, []ErrorDetail{*detail})

		return
	}

	cfg := s.currentConfig()
	if cfg == nil {
		writeError(w, r, http.StatusNotFound, "config_not_found", "No configuration loaded", nil)

		return
	}

	// SECURITY FIX #173: Enforce device count limit
	if len(cfg.Devices) >= MaxDeviceCount {
		writeError(w, r, http.StatusTooManyRequests, "device_limit_reached",
			fmt.Sprintf("Maximum device count of %d reached", MaxDeviceCount), nil)

		return
	}

	// Find source device
	var sourceDevice *config.Device

	for i := range cfg.Devices {
		if cfg.Devices[i].Name == hostname {
			sourceDevice = &cfg.Devices[i]

			break
		}
	}

	if sourceDevice == nil {
		writeError(w, r, http.StatusNotFound, "device_not_found",
			fmt.Sprintf("Device '%s' not found", hostname), nil)

		return
	}

	// Check if new hostname already exists
	for _, dev := range cfg.Devices {
		if dev.Name == req.NewHostname {
			writeError(w, r, http.StatusConflict, "device_exists",
				fmt.Sprintf("Device '%s' already exists", req.NewHostname), nil)

			return
		}
	}

	// Clone device
	clonedDevice := cloneDevice(sourceDevice, req.NewHostname, req.NewIP, req.NewMAC)

	// Add to config and save
	newCfg := *deepCopyConfig(cfg)
	newCfg.Devices = append(newCfg.Devices, *clonedDevice)

	if err := s.saveConfig(&newCfg); err != nil {
		writeError(w, r, http.StatusInternalServerError, "save_failed", "Failed to save configuration", nil)

		return
	}

	// Broadcast change via SSE
	if s.sseHub != nil {
		s.sseHub.BroadcastLog("info", fmt.Sprintf("Device cloned: %s -> %s", hostname, req.NewHostname))
	}

	resp := deviceToResponse(clonedDevice, true, false)

	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, resp)
}

// saveConfig saves the config to file and updates the server state.
