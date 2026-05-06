package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

// Sentinel errors for API devices.
var (
	ErrInvalidMACAddress = errors.New("invalid MAC address")
	ErrInvalidIPAddress  = errors.New("invalid IP address")
	errValidationFailed  = errors.New("validation failed")
)

const (
	// MaxDeviceCount is the maximum number of devices allowed.
	// SECURITY FIX #173: Prevents resource exhaustion via unbounded device creation.
	MaxDeviceCount = 1000

	// maxLabelLen is the max length of a hostname label (RFC 1123).
	maxLabelLen = 63
)

// validateHostname validates a device hostname per RFC 1123.
// SECURITY FIX #169: Prevents invalid or dangerous hostname values.
func validateHostname(hostname string) *ErrorDetail {
	if hostname == "" {
		return &ErrorDetail{Field: "hostname", Issue: "hostname is required"}
	}

	if len(hostname) > maxHostnameLen {
		return &ErrorDetail{
			Field: "hostname",
			Issue: fmt.Sprintf("hostname exceeds maximum length of %d characters", maxHostnameLen),
			Value: hostname[:min(truncateErrorValue, len(hostname))],
		}
	}

	if net.ParseIP(hostname) != nil {
		return &ErrorDetail{Field: "hostname", Issue: "hostname must not be an IP address", Value: hostname}
	}

	for label := range strings.SplitSeq(hostname, ".") {
		if issue := validateHostnameLabel(label); issue != "" {
			return &ErrorDetail{
				Field: "hostname",
				Issue: issue,
				Value: hostname[:min(truncateErrorValue, len(hostname))],
			}
		}
	}

	return nil
}

// validateHostnameLabel validates a single hostname label per RFC 1123.
func validateHostnameLabel(label string) string {
	if len(label) == 0 {
		return "hostname contains empty label (consecutive dots)"
	}
	if len(label) > maxLabelLen {
		return fmt.Sprintf("hostname label exceeds maximum length of %d characters", maxLabelLen)
	}
	if !isAlphanumeric(label[0]) {
		return "hostname labels must start with an alphanumeric character"
	}
	if !isAlphanumeric(label[len(label)-1]) {
		return "hostname labels must not end with a hyphen"
	}
	for _, c := range label {
		if !isAlphanumeric(safeconv.ByteFromRune(c)) && c != '-' {
			return "hostname contains invalid characters (only alphanumeric and hyphens allowed)"
		}
	}
	return ""
}

// isAlphanumeric checks if a byte is alphanumeric.
func isAlphanumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// YAML validation constants to prevent resource exhaustion attacks.
// SECURITY FIX #153: Limit YAML input size and complexity.
const (
	MaxYAMLSize  = 1024 * 1024 // 1MB max YAML input
	MaxYAMLDepth = 20          // Maximum nesting depth
)

// Device protocol collection constant.
const deviceProtocolCapacity = 10 // Expected max protocols per device

// validateYAMLInput checks YAML input for size and complexity limits.
// SECURITY FIX #153: Prevents memory exhaustion and DoS attacks.
func validateYAMLInput(input string) error {
	// Check size
	if len(input) > MaxYAMLSize {
		return fmt.Errorf("YAML input too large: %d bytes (max %d)", len(input), MaxYAMLSize)
	}

	// Check for empty input
	if strings.TrimSpace(input) == "" {
		return errors.New("YAML input is empty")
	}

	return nil
}

// checkYAMLDepth verifies parsed YAML doesn't exceed maximum nesting depth.
// SECURITY FIX #153: Prevents billion laughs and similar attacks.
func checkYAMLDepth(data any, currentDepth int) error {
	if currentDepth > MaxYAMLDepth {
		return fmt.Errorf("YAML nesting depth exceeds maximum of %d", MaxYAMLDepth)
	}

	switch v := data.(type) {
	case map[string]any:
		for _, val := range v {
			err := checkYAMLDepth(val, currentDepth+1)
			if err != nil {
				return err
			}
		}
	case []any:
		for _, item := range v {
			err := checkYAMLDepth(item, currentDepth+1)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DeviceResponse represents a device in API responses with full details.
type DeviceResponse struct {
	// Basic info
	Hostname   string   `json:"hostname"`
	Type       string   `json:"type"`
	MAC        string   `json:"mac,omitempty"`
	IPs        []string `json:"ips,omitempty"`
	VLAN       int      `json:"vlan,omitempty"`
	Interfaces []string `json:"interfaces,omitempty"`
	Protocols  []string `json:"protocols"`

	// Protocol configurations (optional, included when requested)
	SNMPAgent     *SNMPAgentResponse     `json:"snmp_agent,omitempty"`
	CDP           *CDPResponse           `json:"cdp,omitempty"`
	LLDP          *LLDPResponse          `json:"lldp,omitempty"`
	DHCP          *DHCPResponse          `json:"dhcp,omitempty"`
	DNS           *DNSResponse           `json:"dns,omitempty"`
	HTTP          *HTTPResponse          `json:"http,omitempty"`
	FTP           *FTPResponse           `json:"ftp,omitempty"`
	NetBIOS       *NetBIOSResponse       `json:"netbios,omitempty"`
	STP           *STPResponse           `json:"stp,omitempty"`
	TrafficConfig *TrafficConfigResponse `json:"traffic_config,omitempty"`

	// Raw YAML for advanced editing
	RawYAML string `json:"raw_yaml,omitempty"`
}

// SNMPAgentResponse represents SNMP agent configuration.
type SNMPAgentResponse struct {
	Enabled     bool             `json:"enabled"`
	Community   string           `json:"community,omitempty"`
	SysName     string           `json:"sysname,omitempty"`
	SysLocation string           `json:"syslocation,omitempty"`
	SysDescr    string           `json:"sysdescr,omitempty"`
	SysContact  string           `json:"syscontact,omitempty"`
	WalkFile    string           `json:"walk_file,omitempty"`
	AddMibs     []AddMibResponse `json:"add_mibs,omitempty"`
}

// AddMibResponse represents an additional MIB entry.
type AddMibResponse struct {
	OID   string `json:"oid"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// CDPResponse represents CDP configuration.
type CDPResponse struct {
	Enabled         bool   `json:"enabled"`
	Platform        string `json:"platform,omitempty"`
	SoftwareVersion string `json:"software_version,omitempty"`
	PortID          string `json:"port_id,omitempty"`
	Version         int    `json:"version,omitempty"`
	Holdtime        int    `json:"holdtime,omitempty"`
}

// LLDPResponse represents LLDP configuration.
type LLDPResponse struct {
	Enabled           bool   `json:"enabled"`
	ChassisIDType     string `json:"chassis_id_type,omitempty"`
	PortDescription   string `json:"port_description,omitempty"`
	SystemDescription string `json:"system_description,omitempty"`
	TTL               int    `json:"ttl,omitempty"`
}

// DHCPResponse represents DHCP server configuration.
type DHCPResponse struct {
	Enabled    bool     `json:"enabled"`
	PoolStart  string   `json:"pool_start,omitempty"`
	PoolEnd    string   `json:"pool_end,omitempty"`
	SubnetMask string   `json:"subnet_mask,omitempty"`
	Router     string   `json:"router,omitempty"`
	DNS        []string `json:"dns,omitempty"`
}

// DNSResponse represents DNS server configuration.
type DNSResponse struct {
	Enabled bool `json:"enabled"`
	Records int  `json:"record_count,omitempty"`
}

// HTTPResponse represents HTTP server configuration.
type HTTPResponse struct {
	Enabled       bool   `json:"enabled"`
	ServerName    string `json:"server_name,omitempty"`
	EndpointCount int    `json:"endpoint_count,omitempty"`
}

// FTPResponse represents FTP server configuration.
type FTPResponse struct {
	Enabled        bool   `json:"enabled"`
	WelcomeBanner  string `json:"welcome_banner,omitempty"`
	AllowAnonymous bool   `json:"allow_anonymous,omitempty"`
}

// NetBIOSResponse represents NetBIOS configuration.
type NetBIOSResponse struct {
	Enabled   bool   `json:"enabled"`
	Name      string `json:"name,omitempty"`
	Workgroup string `json:"workgroup,omitempty"`
}

// STPResponse represents STP configuration.
type STPResponse struct {
	Enabled  bool   `json:"enabled"`
	Priority uint16 `json:"priority,omitempty"`
}

// TrafficConfigResponse represents traffic pattern configuration.
type TrafficConfigResponse struct {
	Enabled         bool `json:"enabled"`
	ARPEnabled      bool `json:"arp_enabled,omitempty"`
	ARPInterval     int  `json:"arp_interval,omitempty"`
	PingEnabled     bool `json:"ping_enabled,omitempty"`
	PingInterval    int  `json:"ping_interval,omitempty"`
	PingPayloadSize int  `json:"ping_payload_size,omitempty"`
}

// DeviceCreateRequest represents a request to create a device.
type DeviceCreateRequest struct {
	Hostname string `json:"hostname"`
	Type     string `json:"type"`
	MAC      string `json:"mac,omitempty"`
	IP       string `json:"ip,omitempty"`
	Template string `json:"template,omitempty"` // Use template as base
	RawYAML  string `json:"raw_yaml,omitempty"` // Advanced: full YAML
}

// DeviceUpdateRequest represents a request to update a device.
type DeviceUpdateRequest struct {
	Type    string `json:"type,omitempty"`
	MAC     string `json:"mac,omitempty"`
	IP      string `json:"ip,omitempty"`
	RawYAML string `json:"raw_yaml,omitempty"` // Full YAML for the device
}

// DeviceCloneRequest represents a request to clone a device.
type DeviceCloneRequest struct {
	NewHostname string `json:"new_hostname"`
	NewIP       string `json:"new_ip,omitempty"`
	NewMAC      string `json:"new_mac,omitempty"`
}

// DeviceListResponse represents the response for listing devices.
type DeviceListResponse struct {
	Devices    []DeviceResponse `json:"devices"`
	TotalCount int              `json:"total_count"`
}

// handleDevicesV2 handles device CRUD operations
// Routes:
//
//	GET    /api/v1/config/devices           - List all devices
//	GET    /api/v1/config/devices/:id       - Get device by hostname
//	POST   /api/v1/config/devices           - Create new device
//	PUT    /api/v1/config/devices/:id       - Update device
//	DELETE /api/v1/config/devices/:id       - Delete device
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
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	var req DeviceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Failed to parse request", nil)
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

// handleDeviceUpdate updates an existing device.
func (s *Server) handleDeviceUpdate(w http.ResponseWriter, r *http.Request, hostname string) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	var req DeviceUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Failed to parse request", nil)
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

	newCfg := *cfg

	if req.RawYAML != "" {
		updatedDevice, parseErr := updateDeviceFromYAML(req.RawYAML, hostname)
		if parseErr != nil {
			writeError(w, r, http.StatusBadRequest, "parse_failed", parseErr.Error(), nil)
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

// handleDeviceDelete deletes a device.
func (s *Server) handleDeviceDelete(w http.ResponseWriter, r *http.Request, hostname string) {
	cfg := s.currentConfig()
	if cfg == nil {
		writeError(w, r, http.StatusNotFound, "config_not_found", "No configuration loaded", nil)

		return
	}

	// Find and remove device
	newCfg := *cfg
	found := false

	newDevices := make([]config.Device, 0, len(cfg.Devices)-1)

	for _, dev := range cfg.Devices {
		if dev.Name == hostname {
			found = true

			continue
		}

		newDevices = append(newDevices, dev)
	}

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

// handleDeviceClone clones an existing device.
func (s *Server) handleDeviceClone(w http.ResponseWriter, r *http.Request, hostname string) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	var req DeviceCloneRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Failed to parse request", nil)

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
	newCfg := *cfg
	newCfg.Devices = append(newCfg.Devices, *clonedDevice)

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

	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, resp)
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

// collectDeviceProtocols returns a list of enabled protocols for a device.
func collectDeviceProtocols(dev *config.Device) []string {
	protocols := make([]string, 0, deviceProtocolCapacity)

	if dev.SNMPConfig.Community != "" || dev.SNMPConfig.WalkFile != "" {
		protocols = append(protocols, "SNMP")
	}

	if dev.DHCPConfig != nil {
		protocols = append(protocols, "DHCP")
	}

	if dev.DNSConfig != nil {
		protocols = append(protocols, "DNS")
	}

	if dev.HTTPConfig != nil {
		protocols = append(protocols, "HTTP")
	}

	if dev.FTPConfig != nil {
		protocols = append(protocols, "FTP")
	}

	if dev.LLDPConfig != nil && dev.LLDPConfig.Enabled {
		protocols = append(protocols, "LLDP")
	}

	if dev.CDPConfig != nil && dev.CDPConfig.Enabled {
		protocols = append(protocols, "CDP")
	}

	if dev.NetBIOSConfig != nil && dev.NetBIOSConfig.Enabled {
		protocols = append(protocols, "NetBIOS")
	}

	if dev.STPConfig != nil && dev.STPConfig.Enabled {
		protocols = append(protocols, "STP")
	}

	return protocols
}

// buildSNMPAgentResponse creates an SNMPAgentResponse from device config.
func buildSNMPAgentResponse(dev *config.Device) *SNMPAgentResponse {
	if dev.SNMPConfig.Community == "" && dev.SNMPConfig.WalkFile == "" {
		return nil
	}

	resp := &SNMPAgentResponse{
		Enabled:     true,
		Community:   dev.SNMPConfig.Community,
		SysName:     dev.SNMPConfig.SysName,
		SysLocation: dev.SNMPConfig.SysLocation,
		SysDescr:    dev.SNMPConfig.SysDescr,
		SysContact:  dev.SNMPConfig.SysContact,
		WalkFile:    dev.SNMPConfig.WalkFile,
	}

	if len(dev.SNMPConfig.AddMibs) > 0 {
		resp.AddMibs = make([]AddMibResponse, 0, len(dev.SNMPConfig.AddMibs))
		for _, mib := range dev.SNMPConfig.AddMibs {
			resp.AddMibs = append(resp.AddMibs, AddMibResponse{
				OID:   mib.OID,
				Type:  mib.Type,
				Value: mib.Value,
			})
		}
	}

	return resp
}

// buildDHCPResponse creates a DHCPResponse from device config.
func buildDHCPResponse(dev *config.Device) *DHCPResponse {
	if dev.DHCPConfig == nil {
		return nil
	}

	resp := &DHCPResponse{Enabled: true}

	if dev.DHCPConfig.PoolStart != nil {
		resp.PoolStart = dev.DHCPConfig.PoolStart.String()
	}

	if dev.DHCPConfig.PoolEnd != nil {
		resp.PoolEnd = dev.DHCPConfig.PoolEnd.String()
	}

	if dev.DHCPConfig.SubnetMask != nil {
		resp.SubnetMask = net.IP(dev.DHCPConfig.SubnetMask).String()
	}

	if dev.DHCPConfig.Router != nil {
		resp.Router = dev.DHCPConfig.Router.String()
	}

	resp.DNS = make([]string, 0, len(dev.DHCPConfig.DomainNameServer))
	for _, dns := range dev.DHCPConfig.DomainNameServer {
		resp.DNS = append(resp.DNS, dns.String())
	}

	return resp
}

// buildTrafficConfigResponse creates a TrafficConfigResponse from device config.
func buildTrafficConfigResponse(dev *config.Device) *TrafficConfigResponse {
	if dev.TrafficConfig == nil || !dev.TrafficConfig.Enabled {
		return nil
	}

	tc := &TrafficConfigResponse{Enabled: true}

	if dev.TrafficConfig.ARPAnnouncements != nil {
		tc.ARPEnabled = dev.TrafficConfig.ARPAnnouncements.Enabled
		tc.ARPInterval = dev.TrafficConfig.ARPAnnouncements.Interval
	}

	if dev.TrafficConfig.PeriodicPings != nil {
		tc.PingEnabled = dev.TrafficConfig.PeriodicPings.Enabled
		tc.PingInterval = dev.TrafficConfig.PeriodicPings.Interval
		tc.PingPayloadSize = dev.TrafficConfig.PeriodicPings.PayloadSize
	}

	return tc
}

// populateProtocolDetails fills in protocol detail fields on DeviceResponse.
func populateProtocolDetails(resp *DeviceResponse, dev *config.Device) {
	resp.SNMPAgent = buildSNMPAgentResponse(dev)
	resp.CDP = buildCDPResponse(dev)
	resp.LLDP = buildLLDPResponse(dev)
	resp.DHCP = buildDHCPResponse(dev)
	resp.DNS = buildDNSResponse(dev)
	resp.HTTP = buildHTTPResponse(dev)
	resp.FTP = buildFTPResponse(dev)
	resp.NetBIOS = buildNetBIOSResponse(dev)
	resp.STP = buildSTPResponse(dev)
	resp.TrafficConfig = buildTrafficConfigResponse(dev)
}

func buildCDPResponse(dev *config.Device) *CDPResponse {
	if dev.CDPConfig == nil || !dev.CDPConfig.Enabled {
		return nil
	}
	return &CDPResponse{
		Enabled:         true,
		Platform:        dev.CDPConfig.Platform,
		SoftwareVersion: dev.CDPConfig.SoftwareVersion,
		PortID:          dev.CDPConfig.PortID,
		Version:         dev.CDPConfig.Version,
		Holdtime:        dev.CDPConfig.Holdtime,
	}
}

func buildLLDPResponse(dev *config.Device) *LLDPResponse {
	if dev.LLDPConfig == nil || !dev.LLDPConfig.Enabled {
		return nil
	}
	return &LLDPResponse{
		Enabled:           true,
		ChassisIDType:     dev.LLDPConfig.ChassisIDType,
		PortDescription:   dev.LLDPConfig.PortDescription,
		SystemDescription: dev.LLDPConfig.SystemDescription,
		TTL:               dev.LLDPConfig.TTL,
	}
}

func buildDNSResponse(dev *config.Device) *DNSResponse {
	if dev.DNSConfig == nil {
		return nil
	}
	return &DNSResponse{
		Enabled: true,
		Records: len(dev.DNSConfig.ForwardRecords) + len(dev.DNSConfig.ReverseRecords),
	}
}

func buildHTTPResponse(dev *config.Device) *HTTPResponse {
	if dev.HTTPConfig == nil || !dev.HTTPConfig.Enabled {
		return nil
	}
	return &HTTPResponse{
		Enabled:       true,
		ServerName:    dev.HTTPConfig.ServerName,
		EndpointCount: len(dev.HTTPConfig.Endpoints),
	}
}

func buildFTPResponse(dev *config.Device) *FTPResponse {
	if dev.FTPConfig == nil || !dev.FTPConfig.Enabled {
		return nil
	}
	return &FTPResponse{
		Enabled:        true,
		WelcomeBanner:  dev.FTPConfig.WelcomeBanner,
		AllowAnonymous: dev.FTPConfig.AllowAnonymous,
	}
}

func buildNetBIOSResponse(dev *config.Device) *NetBIOSResponse {
	if dev.NetBIOSConfig == nil || !dev.NetBIOSConfig.Enabled {
		return nil
	}
	return &NetBIOSResponse{
		Enabled:   true,
		Name:      dev.NetBIOSConfig.Name,
		Workgroup: dev.NetBIOSConfig.Workgroup,
	}
}

func buildSTPResponse(dev *config.Device) *STPResponse {
	if dev.STPConfig == nil || !dev.STPConfig.Enabled {
		return nil
	}
	return &STPResponse{
		Enabled:  true,
		Priority: dev.STPConfig.BridgePriority,
	}
}

// deviceToResponse converts a Device to DeviceResponse.
func deviceToResponse(dev *config.Device, includeDetails, includeYAML bool) DeviceResponse {
	resp := DeviceResponse{
		Hostname:  dev.Name,
		Type:      dev.Type,
		VLAN:      dev.VLAN,
		Protocols: collectDeviceProtocols(dev),
	}

	if dev.MACAddress != nil {
		resp.MAC = dev.MACAddress.String()
	}

	resp.IPs = make([]string, 0, len(dev.IPAddresses))
	for _, ip := range dev.IPAddresses {
		resp.IPs = append(resp.IPs, ip.String())
	}

	resp.Interfaces = make([]string, 0, len(dev.Interfaces))
	for _, iface := range dev.Interfaces {
		resp.Interfaces = append(resp.Interfaces, iface.Name)
	}

	if includeDetails {
		populateProtocolDetails(&resp, dev)
	}

	if includeYAML {
		yamlBytes, err := serializeDeviceToYAML(dev)
		if err == nil {
			resp.RawYAML = string(yamlBytes)
		}
	}

	return resp
}

// Helper functions.
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
