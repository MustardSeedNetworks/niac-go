package api

import (
	"encoding/json"
	"fmt"
	"maps"
	"net"
	"net/http"
	"strings"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// handleDeviceClone clones an existing device.
func (s *Server) handleDeviceClone(w http.ResponseWriter, r *http.Request, hostname string) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	var req DeviceCloneRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Failed to parse request", nil)
		return
	}

	if req.NewHostname == "" {
		writeError(w, r, http.StatusBadRequest, "validation_failed",
			"new_hostname is required", []ErrorDetail{{Field: "new_hostname", Issue: "required"}})
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

	// Find source device
	var sourceDevice *config.Device

	for i := range cfg.Devices {
		if cfg.Devices[i].Name == hostname {
			sourceDevice = &cfg.Devices[i]
			break
		}
	}

	if sourceDevice == nil {
		s.configMu.Unlock()
		writeError(w, r, http.StatusNotFound, "device_not_found",
			fmt.Sprintf("Device '%s' not found", hostname), nil)
		return
	}

	// Check if new hostname already exists
	for _, dev := range cfg.Devices {
		if dev.Name == req.NewHostname {
			s.configMu.Unlock()
			writeError(w, r, http.StatusConflict, "device_exists",
				fmt.Sprintf("Device '%s' already exists", req.NewHostname), nil)
			return
		}
	}

	// Clone device
	clonedDevice := cloneDevice(sourceDevice, req.NewHostname, req.NewIP, req.NewMAC)

	// Add to config and save (deep copy devices slice)
	newCfg := *cfg
	newCfg.Devices = append(append([]config.Device(nil), cfg.Devices...), *clonedDevice)
	s.configMu.Unlock()

	err = s.saveConfig(&newCfg)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "save_failed", "Failed to save configuration", nil)
		return
	}

	// Broadcast change via SSE
	if s.sseHub != nil {
		s.sseHub.BroadcastLog("info", fmt.Sprintf("Device cloned: %s -> %s", hostname, req.NewHostname))
	}

	resp := deviceToResponse(clonedDevice, true, false)
	// FIX #286: Wrap response in mutation format { success, device, message }
	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, map[string]any{
		"success": true,
		"device":  resp,
		"message": fmt.Sprintf("Device '%s' cloned from '%s' successfully", req.NewHostname, hostname),
	})
}

func cloneDevice(src *config.Device, newHostname, newIP, newMAC string) *config.Device {
	// FIX #268: Deep copy the device including all pointer fields
	cloned := *src
	cloned.Name = newHostname

	cloneDeviceSlices(src, &cloned)
	cloneDevicePointers(src, &cloned)

	// Update IP if provided
	if newIP != "" {
		if ip, parseErr := parseIP(newIP); parseErr == nil {
			cloned.IPAddresses = []net.IP{ip}
		}
	}

	// Update MAC if provided
	if newMAC != "" {
		if mac, parseErr := parseMAC(newMAC); parseErr == nil {
			cloned.MACAddress = mac
		}
	}

	// Update hostname in protocol configs where applicable
	if cloned.SNMPConfig.SysName != "" {
		cloned.SNMPConfig.SysName = newHostname
	}

	if cloned.NetBIOSConfig != nil && cloned.NetBIOSConfig.Name != "" {
		cloned.NetBIOSConfig.Name = strings.ToUpper(newHostname)
	}

	return &cloned
}

// cloneDeviceSlices deep copies all slice and map fields from src to dst.
func cloneDeviceSlices(src, dst *config.Device) {
	if src.IPAddresses != nil {
		dst.IPAddresses = make([]net.IP, len(src.IPAddresses))
		for i, ip := range src.IPAddresses {
			dst.IPAddresses[i] = make(net.IP, len(ip))
			copy(dst.IPAddresses[i], ip)
		}
	}

	if src.MACAddress != nil {
		dst.MACAddress = make(net.HardwareAddr, len(src.MACAddress))
		copy(dst.MACAddress, src.MACAddress)
	}

	if src.Interfaces != nil {
		dst.Interfaces = make([]config.Interface, len(src.Interfaces))
		copy(dst.Interfaces, src.Interfaces)
	}

	if src.PortChannels != nil {
		dst.PortChannels = make([]config.PortChannel, len(src.PortChannels))
		copy(dst.PortChannels, src.PortChannels)
	}

	if src.TrunkPorts != nil {
		dst.TrunkPorts = make([]config.TrunkPort, len(src.TrunkPorts))
		copy(dst.TrunkPorts, src.TrunkPorts)
	}

	if src.Properties != nil {
		dst.Properties = make(map[string]string, len(src.Properties))
		maps.Copy(dst.Properties, src.Properties)
	}
}

// cloneDevicePointers deep copies all pointer fields from src to dst.
func cloneDevicePointers(src, dst *config.Device) {
	cloneDeviceNetworkPointers(src, dst)
	cloneDeviceServicePointers(src, dst)
}

// cloneDeviceNetworkPointers copies network protocol pointer fields.
func cloneDeviceNetworkPointers(src, dst *config.Device) {
	if src.TTLConfig != nil {
		c := *src.TTLConfig
		dst.TTLConfig = &c
	}
	if src.DHCPConfig != nil {
		c := *src.DHCPConfig
		dst.DHCPConfig = &c
	}
	if src.DNSConfig != nil {
		c := *src.DNSConfig
		dst.DNSConfig = &c
	}
	if src.LLDPConfig != nil {
		c := *src.LLDPConfig
		dst.LLDPConfig = &c
	}
	if src.CDPConfig != nil {
		c := *src.CDPConfig
		dst.CDPConfig = &c
	}
	if src.EDPConfig != nil {
		c := *src.EDPConfig
		dst.EDPConfig = &c
	}
	if src.FDPConfig != nil {
		c := *src.FDPConfig
		dst.FDPConfig = &c
	}
	if src.STPConfig != nil {
		c := *src.STPConfig
		dst.STPConfig = &c
	}
}

// cloneDeviceServicePointers copies service and traffic pointer fields.
func cloneDeviceServicePointers(src, dst *config.Device) {
	if src.HTTPConfig != nil {
		c := *src.HTTPConfig
		dst.HTTPConfig = &c
	}
	if src.FTPConfig != nil {
		c := *src.FTPConfig
		dst.FTPConfig = &c
	}
	if src.NetBIOSConfig != nil {
		c := *src.NetBIOSConfig
		dst.NetBIOSConfig = &c
	}
	if src.ICMPConfig != nil {
		c := *src.ICMPConfig
		dst.ICMPConfig = &c
	}
	if src.ICMPv6Config != nil {
		c := *src.ICMPv6Config
		dst.ICMPv6Config = &c
	}
	if src.DHCPv6Config != nil {
		c := *src.DHCPv6Config
		dst.DHCPv6Config = &c
	}
	if src.TrafficConfig != nil {
		c := *src.TrafficConfig
		dst.TrafficConfig = &c
	}
	if src.OSFingerprintConfig != nil {
		c := *src.OSFingerprintConfig
		dst.OSFingerprintConfig = &c
	}
	if src.IPerf3 != nil {
		c := *src.IPerf3
		dst.IPerf3 = &c
	}
}
