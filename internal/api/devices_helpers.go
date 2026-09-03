package api

import (
	"errors"
	"fmt"
	"maps"
	"net"
	"net/http"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

func (s *Server) validateDeviceCreatePreconditions(
	w http.ResponseWriter, r *http.Request, hostname string,
) (*config.Config, error) {
	cfg := s.currentConfig()
	if cfg == nil {
		writeError(w, r, http.StatusBadRequest, "config_not_found", "No configuration loaded", nil)
		return nil, errValidationFailed
	}
	if err := s.validateDeviceAddition(w, r, cfg, hostname); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *Server) validateDeviceAddition(
	w http.ResponseWriter, r *http.Request, cfg *config.Config, hostname string,
) error {
	deviceCount := cfg.DeviceCount()
	if deviceCount >= MaxDeviceCount {
		writeError(w, r, http.StatusTooManyRequests, "device_limit_reached",
			fmt.Sprintf("Maximum device count of %d reached", MaxDeviceCount), nil)
		return errValidationFailed
	}

	if len(cfg.Segments) > 0 {
		writeError(
			w,
			r,
			http.StatusConflict,
			"segmented_config_requires_replacement",
			"Devices in segmented configurations must be changed through whole-config replacement",
			nil,
		)
		return errValidationFailed
	}

	if deviceExists(cfg.Devices, hostname) {
		writeError(w, r, http.StatusConflict, "device_exists",
			fmt.Sprintf("Device '%s' already exists", hostname), nil)
		return errValidationFailed
	}
	return nil
}

// deviceExists checks if a device with the given hostname exists.
func deviceExists(devices []config.Device, hostname string) bool {
	return slices.ContainsFunc(devices, func(dev config.Device) bool {
		return dev.Name == hostname
	})
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
		maps.Copy(copied.Properties, dev.Properties)
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
		s.logger.ErrorContext(
			r.Context(),
			"[API] Device creation failed",
			"error",
			err,
			"hostname",
			req.Hostname,
		)

		line, msg := parseYAMLError(err)
		writeError(w, r, http.StatusBadRequest, "device_creation_failed",
			"Failed to create device from request", []ErrorDetail{{Issue: msg, Line: line}})

		return nil, err
	}

	if validationErr := config.ValidateDeviceManagementRequirements(newDevice); validationErr != nil {
		writeError(
			w,
			r,
			http.StatusBadRequest,
			"management_config_invalid",
			"Device management configuration is invalid",
			[]ErrorDetail{{Issue: validationErr.Error()}},
		)
		return nil, validationErr
	}

	newCfg := *deepCopyConfig(cfg)
	newCfg.Devices = append(newCfg.Devices, *newDevice)

	if saveErr := s.saveConfig(&newCfg); saveErr != nil {
		writeError(
			w,
			r,
			http.StatusInternalServerError,
			"save_failed",
			"Failed to save configuration",
			nil,
		)
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

	if err := applyDeviceScalarFields(dev, req.IPs, req.VLAN, req.Babble, req.MapToIP); err != nil {
		return fmt.Errorf("invalid_ip: %w", err)
	}

	interfaceUpdates := coalesceInterfaceUpdates(req.Interfaces, req.InterfaceDetails)
	if interfaceUpdates != nil {
		interfaces, err := interfaceUpdatesToConfig(interfaceUpdates)
		if err != nil {
			return err
		}
		dev.Interfaces = interfaces
	}
	if req.SNMPAgent != nil {
		applySNMPAgentRequest(&dev.SNMPConfig, req.SNMPAgent)
	}
	applyManagementRequests(dev, req.SSH, req.Syslog)
	applyDiscoveryProtocolRequests(dev, req.LLDP, req.CDP, req.EDP, req.FDP, req.STP)

	serviceErr := applyServiceProtocolRequests(
		dev, req.DHCP, req.DHCPv6, req.DNS, req.HTTP, req.FTP, req.NetBIOS,
	)
	if serviceErr != nil {
		return serviceErr
	}

	if err := applyHostProtocolRequests(dev, req.ICMP, req.ICMPv6, req.TTL, req.OSFingerprint, req.IPerf3); err != nil {
		return err
	}

	return nil
}

func applySNMPAgentRequest(dst *config.SNMPConfig, src *SNMPAgentRequest) {
	dst.Community = strings.TrimSpace(src.Community)
	dst.SysName = strings.TrimSpace(src.SysName)
	dst.SysDescr = strings.TrimSpace(src.SysDescr)
	dst.SysContact = strings.TrimSpace(src.SysContact)
	dst.SysLocation = strings.TrimSpace(src.SysLocation)
	dst.WalkFile = strings.TrimSpace(src.WalkFile)
	dst.WalkFiles = append([]string(nil), src.WalkFiles...)
	dst.AddMibs = make([]config.AddMib, 0, len(src.AddMibs))
	for _, mib := range src.AddMibs {
		dst.AddMibs = append(
			dst.AddMibs,
			config.AddMib{OID: mib.OID, Type: mib.Type, Value: mib.Value},
		)
	}
}

func applyManagementRequests(
	dev *config.Device,
	ssh *SSHConfigRequest,
	syslog *SyslogConfigRequest,
) {
	if ssh != nil {
		dev.SSHConfig = &config.SSHConfig{
			Enabled: ssh.Enabled, Username: strings.TrimSpace(ssh.Username),
			PasswordEnv: strings.TrimSpace(ssh.PasswordEnv),
		}
	}
	if syslog != nil {
		dev.SyslogConfig = &config.SyslogConfig{
			Enabled: syslog.Enabled, Receivers: append([]string(nil), syslog.Receivers...),
		}
	}
}

func interfaceUpdatesToConfig(updates []DeviceInterfaceUpdate) ([]config.Interface, error) {
	interfaces := make([]config.Interface, 0, len(updates))
	seen := make(map[string]bool, len(updates))

	for _, update := range updates {
		name := strings.TrimSpace(update.Name)
		if name == "" {
			return nil, errors.New("invalid_interface: interface name is required")
		}
		if seen[name] {
			return nil, fmt.Errorf("invalid_interface: duplicate interface %s", name)
		}
		seen[name] = true

		if update.Speed < 0 {
			return nil, fmt.Errorf("invalid_interface: speed must be zero or greater for %s", name)
		}
		if update.Duplex != "" && !isAllowedInterfaceValue(update.Duplex, "full", "half", "auto") {
			return nil, fmt.Errorf(
				"invalid_interface: duplex must be full, half, or auto for %s",
				name,
			)
		}
		if update.AdminStatus != "" && !isAllowedInterfaceValue(update.AdminStatus, "up", "down") {
			return nil, fmt.Errorf(
				"invalid_interface: admin_status must be up or down for %s",
				name,
			)
		}
		if update.OperStatus != "" &&
			!isAllowedInterfaceValue(update.OperStatus, "up", "down", "testing") {
			return nil, fmt.Errorf(
				"invalid_interface: oper_status must be up, down, or testing for %s",
				name,
			)
		}

		interfaces = append(interfaces, config.Interface{
			Name:        name,
			Speed:       update.Speed,
			Duplex:      strings.ToLower(strings.TrimSpace(update.Duplex)),
			AdminStatus: strings.ToLower(strings.TrimSpace(update.AdminStatus)),
			OperStatus:  strings.ToLower(strings.TrimSpace(update.OperStatus)),
			Description: strings.TrimSpace(update.Description),
			VLANs:       update.VLANs,
		})
	}

	return interfaces, nil
}

func coalesceInterfaceUpdates(
	interfaces, interfaceDetails []DeviceInterfaceUpdate,
) []DeviceInterfaceUpdate {
	if interfaceDetails != nil {
		return interfaceDetails
	}
	return interfaces
}

func isAllowedInterfaceValue(value string, allowed ...string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return slices.Contains(allowed, value)
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

	if err := applyDeviceScalarFields(dev, req.IPs, req.VLAN, req.Babble, req.MapToIP); err != nil {
		return nil, err
	}

	interfaceUpdates := coalesceInterfaceUpdates(req.Interfaces, req.InterfaceDetails)
	if interfaceUpdates != nil {
		interfaces, err := interfaceUpdatesToConfig(interfaceUpdates)
		if err != nil {
			return nil, err
		}
		dev.Interfaces = interfaces
	}
	if req.SNMPAgent != nil {
		applySNMPAgentRequest(&dev.SNMPConfig, req.SNMPAgent)
	}
	applyManagementRequests(dev, req.SSH, req.Syslog)
	applyDiscoveryProtocolRequests(dev, req.LLDP, req.CDP, req.EDP, req.FDP, req.STP)

	serviceErr := applyServiceProtocolRequests(
		dev, req.DHCP, req.DHCPv6, req.DNS, req.HTTP, req.FTP, req.NetBIOS,
	)
	if serviceErr != nil {
		return nil, serviceErr
	}

	if err := applyHostProtocolRequests(dev, req.ICMP, req.ICMPv6, req.TTL, req.OSFingerprint, req.IPerf3); err != nil {
		return nil, err
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

// parseDeviceFromYAML reads one device document the way the rest of the product
// reads a config. It used to have its own reader that understood two fields, so
// an operator who edited one line in the device editor got a device with its
// addresses, agent and interfaces silently dropped - the read path serializes
// all of them and the write path threw them away.
func parseDeviceFromYAML(yamlStr, hostname string) (*config.Device, error) {
	// SECURITY FIX #153: Validate YAML input before parsing
	if validateErr := validateYAMLInput(yamlStr); validateErr != nil {
		return nil, fmt.Errorf("YAML validation failed: %w", validateErr)
	}
	var depthCheck map[string]any
	if unmarshalErr := yaml.Unmarshal([]byte(yamlStr), &depthCheck); unmarshalErr != nil {
		return nil, fmt.Errorf("invalid YAML: %w", unmarshalErr)
	}
	// SECURITY FIX #153: Check YAML depth to prevent DoS attacks
	if depthErr := checkYAMLDepth(depthCheck, 0); depthErr != nil {
		return nil, depthErr
	}

	var authored converter.Device
	if unmarshalErr := yaml.Unmarshal([]byte(yamlStr), &authored); unmarshalErr != nil {
		return nil, fmt.Errorf("invalid device: %w", unmarshalErr)
	}
	// The request names the device; that is what a rename and a clone rely on.
	authored.Name = hostname

	document, marshalErr := yaml.Marshal(converter.Config{Devices: []converter.Device{authored}})
	if marshalErr != nil {
		return nil, fmt.Errorf("re-encode device: %w", marshalErr)
	}
	loaded, loadErr := config.LoadYAMLBytes(document)
	if loadErr != nil {
		return nil, fmt.Errorf("invalid device: %w", loadErr)
	}
	if len(loaded.Devices) != 1 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidDeviceDocument, hostname)
	}

	return &loaded.Devices[0], nil
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
