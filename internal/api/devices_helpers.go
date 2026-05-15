package api

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/krisarmstrong/niac-go/internal/config"
)

func (s *Server) validateDeviceCreatePreconditions(
	w http.ResponseWriter, r *http.Request, hostname string,
) (*config.Config, error) {
	cfg := s.currentConfig()
	if cfg == nil {
		writeError(w, r, http.StatusBadRequest, "config_not_found", "No configuration loaded", nil)
		return nil, errValidationFailed
	}

	if len(cfg.Devices) >= MaxDeviceCount {
		writeError(w, r, http.StatusTooManyRequests, "device_limit_reached",
			fmt.Sprintf("Maximum device count of %d reached", MaxDeviceCount), nil)
		return nil, errValidationFailed
	}

	if deviceExists(cfg.Devices, hostname) {
		writeError(w, r, http.StatusConflict, "device_exists",
			fmt.Sprintf("Device '%s' already exists", hostname), nil)
		return nil, errValidationFailed
	}

	return cfg, nil
}

// deviceExists checks if a device with the given hostname exists.
func deviceExists(devices []config.Device, hostname string) bool {
	for _, dev := range devices {
		if dev.Name == hostname {
			return true
		}
	}
	return false
}

func deepCopyConfig(cfg *config.Config) *config.Config {
	if cfg == nil {
		return nil
	}

	copied := *cfg
	copied.Devices = make([]config.Device, len(cfg.Devices))
	for i := range cfg.Devices {
		copied.Devices[i] = *deepCopyDevice(&cfg.Devices[i])
	}

	return &copied
}

func deepCopyDevice(dev *config.Device) *config.Device {
	if dev == nil {
		return nil
	}

	copied := *dev
	copied.MACAddress = copyHardwareAddr(dev.MACAddress)
	copied.MapToIP = copyIP(dev.MapToIP)
	copied.IPAddresses = copyIPSlice(dev.IPAddresses)
	copied.Interfaces = append([]config.Interface(nil), dev.Interfaces...)
	copied.PortChannels = append([]config.PortChannel(nil), dev.PortChannels...)
	copied.TrunkPorts = append([]config.TrunkPort(nil), dev.TrunkPorts...)
	if dev.Properties != nil {
		copied.Properties = make(map[string]string, len(dev.Properties))
		for key, value := range dev.Properties {
			copied.Properties[key] = value
		}
	}

	return &copied
}

func copyHardwareAddr(addr net.HardwareAddr) net.HardwareAddr {
	if addr == nil {
		return nil
	}

	return append(net.HardwareAddr(nil), addr...)
}

func copyIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}

	return append(net.IP(nil), ip...)
}

func copyIPSlice(ips []net.IP) []net.IP {
	if ips == nil {
		return nil
	}

	copied := make([]net.IP, len(ips))
	for i := range ips {
		copied[i] = copyIP(ips[i])
	}

	return copied
}

// createAndSaveDevice creates a device from request and saves the config.
func (s *Server) createAndSaveDevice(
	w http.ResponseWriter, r *http.Request, cfg *config.Config, req DeviceCreateRequest,
) (*config.Device, error) {
	newDevice, err := createDeviceFromRequest(req)
	if err != nil {
		s.logger.Error("[API] Device creation failed", "error", err, "hostname", req.Hostname)
		writeError(w, r, http.StatusBadRequest, "device_creation_failed",
			"Failed to create device from request", nil)
		return nil, err
	}

	newCfg := *deepCopyConfig(cfg)
	newCfg.Devices = append(newCfg.Devices, *newDevice)

	if saveErr := s.saveConfig(&newCfg); saveErr != nil {
		writeError(w, r, http.StatusInternalServerError, "save_failed", "Failed to save configuration", nil)
		return nil, saveErr
	}

	return newDevice, nil
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

func parseMAC(s string) (net.HardwareAddr, error) {
	mac, err := net.ParseMAC(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMACAddress, s)
	}

	return mac, nil
}

func parseIP(s string) (net.IP, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidIPAddress, s)
	}

	return ip, nil
}

func parseDeviceFromYAML(yamlStr, hostname string) (*config.Device, error) {
	// SECURITY FIX #153: Validate YAML input before parsing
	if validateErr := validateYAMLInput(yamlStr); validateErr != nil {
		return nil, fmt.Errorf("YAML validation failed: %w", validateErr)
	}

	// This is a simplified parser - in production, use the full config loader
	// For now, return a basic device
	dev := &config.Device{
		Name: hostname,
	}

	// Parse YAML into map for basic fields
	var data map[string]any
	if unmarshalErr := yaml.Unmarshal([]byte(yamlStr), &data); unmarshalErr != nil {
		return nil, fmt.Errorf("invalid YAML: %w", unmarshalErr)
	}

	// SECURITY FIX #153: Check YAML depth to prevent DoS attacks
	if depthErr := checkYAMLDepth(data, 0); depthErr != nil {
		return nil, depthErr
	}

	if t, ok := data["type"].(string); ok {
		dev.Type = t
	}

	if mac, ok := data["mac"].(string); ok {
		if parsed, parseErr := parseMAC(mac); parseErr == nil {
			dev.MACAddress = parsed
		}
	}

	return dev, nil
}

func cloneDevice(src *config.Device, newHostname, newIP, newMAC string) *config.Device {
	// Deep copy the device
	cloned := *src
	cloned.Name = newHostname

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
	// CDP/LLDP derive device ID/system name from hostname automatically
	if cloned.NetBIOSConfig != nil && cloned.NetBIOSConfig.Name != "" {
		cloned.NetBIOSConfig.Name = strings.ToUpper(newHostname)
	}

	return &cloned
}

func serializeDeviceToYAML(dev *config.Device) ([]byte, error) {
	// Build a map representation for YAML serialization
	data := map[string]any{
		"name": dev.Name,
		"type": dev.Type,
	}

	if dev.MACAddress != nil {
		data["mac"] = dev.MACAddress.String()
	}

	if len(dev.IPAddresses) > 0 {
		if len(dev.IPAddresses) == 1 {
			data["ip"] = dev.IPAddresses[0].String()
		} else {
			ips := make([]string, 0, len(dev.IPAddresses))
			for _, ip := range dev.IPAddresses {
				ips = append(ips, ip.String())
			}

			data["ips"] = ips
		}
	}

	// Add SNMP config if present
	if dev.SNMPConfig.Community != "" || dev.SNMPConfig.WalkFile != "" {
		snmp := map[string]any{
			"enabled": true,
		}
		if dev.SNMPConfig.Community != "" {
			snmp["community"] = dev.SNMPConfig.Community
		}

		if dev.SNMPConfig.SysName != "" {
			snmp["sysname"] = dev.SNMPConfig.SysName
		}

		if dev.SNMPConfig.WalkFile != "" {
			snmp["walk_file"] = dev.SNMPConfig.WalkFile
		}

		data["snmp_agent"] = snmp
	}

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal device YAML: %w", err)
	}
	return yamlData, nil
}

func serializeConfigToYAML(cfg *config.Config) (string, error) {
	yamlBytes, err := config.MarshalConfigYAML(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config YAML: %w", err)
	}

	return string(yamlBytes), nil
}
