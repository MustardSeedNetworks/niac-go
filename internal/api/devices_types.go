package api

// DeviceResponse represents a device in API responses with full details.
type DeviceResponse struct {
	// Basic info
	Hostname         string                    `json:"hostname"`
	Type             string                    `json:"type"`
	MAC              string                    `json:"mac,omitempty"`
	IPs              []string                  `json:"ips,omitempty"`
	VLAN             int                       `json:"vlan,omitempty"`
	Interfaces       []string                  `json:"interfaces,omitempty"`
	InterfaceDetails []DeviceInterfaceResponse `json:"interface_details,omitempty"`
	Protocols        []string                  `json:"protocols"`

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

// DeviceInterfaceResponse represents configured metadata for a device interface.
type DeviceInterfaceResponse struct {
	Name        string `json:"name"`
	Speed       int    `json:"speed,omitempty"`
	Duplex      string `json:"duplex,omitempty"`
	AdminStatus string `json:"admin_status,omitempty"`
	OperStatus  string `json:"oper_status,omitempty"`
	Description string `json:"description,omitempty"`
	VLANs       []int  `json:"vlans,omitempty"`
}

// DeviceInterfaceUpdate represents partial update data for a device interface.
type DeviceInterfaceUpdate struct {
	Name        string `json:"name"`
	Speed       int    `json:"speed,omitempty"`
	Duplex      string `json:"duplex,omitempty"`
	AdminStatus string `json:"admin_status,omitempty"`
	OperStatus  string `json:"oper_status,omitempty"`
	Description string `json:"description,omitempty"`
	VLANs       []int  `json:"vlans,omitempty"`
}

// DeviceCreateRequest represents a request to create a device.
type DeviceCreateRequest struct {
	Hostname         string                  `json:"hostname"`
	Type             string                  `json:"type"`
	MAC              string                  `json:"mac,omitempty"`
	IP               string                  `json:"ip,omitempty"`
	Interfaces       []DeviceInterfaceUpdate `json:"interfaces,omitempty"`
	InterfaceDetails []DeviceInterfaceUpdate `json:"interface_details,omitempty"`
	Template         string                  `json:"template,omitempty"` // Use template as base
	RawYAML          string                  `json:"raw_yaml,omitempty"` // Advanced: full YAML
}

// DeviceUpdateRequest represents a request to update a device.
type DeviceUpdateRequest struct {
	Type             string                  `json:"type,omitempty"`
	MAC              string                  `json:"mac,omitempty"`
	IP               string                  `json:"ip,omitempty"`
	Interfaces       []DeviceInterfaceUpdate `json:"interfaces,omitempty"`
	InterfaceDetails []DeviceInterfaceUpdate `json:"interface_details,omitempty"`
	RawYAML          string                  `json:"raw_yaml,omitempty"` // Full YAML for the device
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
