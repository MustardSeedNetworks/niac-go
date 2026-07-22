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
	InterfaceDetails []DeviceInterfaceResponse `json:"interfaceDetails,omitempty"`
	Protocols        []string                  `json:"protocols"`

	// Protocol configurations (optional, included when requested)
	SNMPAgent *SNMPAgentResponse `json:"snmpAgent,omitempty"`
	CDP       *CDPResponse       `json:"cdp,omitempty"`
	LLDP      *LLDPResponse      `json:"lldp,omitempty"`
	DHCP      *DHCPResponse      `json:"dhcp,omitempty"`
	DNS       *DNSResponse       `json:"dns,omitempty"`
	HTTP      *HTTPResponse      `json:"http,omitempty"`
	FTP       *FTPResponse       `json:"ftp,omitempty"`
	NetBIOS   *NetBIOSResponse   `json:"netbios,omitempty"`
	STP       *STPResponse       `json:"stp,omitempty"`

	// Raw YAML for advanced editing
	RawYAML string `json:"rawYaml,omitempty"`
}

// SNMPAgentResponse represents SNMP agent configuration.
type SNMPAgentResponse struct {
	Enabled     bool             `json:"enabled"`
	Community   string           `json:"community,omitempty"`
	SysName     string           `json:"sysname,omitempty"`
	SysLocation string           `json:"syslocation,omitempty"`
	SysDescr    string           `json:"sysdescr,omitempty"`
	SysContact  string           `json:"syscontact,omitempty"`
	WalkFile    string           `json:"walkFile,omitempty"`
	WalkFiles   []string         `json:"walkFiles,omitempty"`
	AddMibs     []AddMibResponse `json:"addMibs,omitempty"`
}

// AddMibResponse represents an additional MIB entry.
type AddMibResponse struct {
	OID   string `json:"oid"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// SNMPAgentRequest is the editable SNMP subset accepted by device CRUD.
type SNMPAgentRequest struct {
	Community   string          `json:"community,omitempty"`
	SysName     string          `json:"sysName,omitempty"`
	SysDescr    string          `json:"sysDescr,omitempty"`
	SysContact  string          `json:"sysContact,omitempty"`
	SysLocation string          `json:"sysLocation,omitempty"`
	WalkFile    string          `json:"walkFile,omitempty"`
	WalkFiles   []string        `json:"walkFiles,omitempty"`
	AddMibs     []AddMibRequest `json:"addMibs,omitempty"`
}

type AddMibRequest struct {
	OID   string `json:"oid"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// CDPResponse represents CDP configuration.
type CDPResponse struct {
	Enabled         bool   `json:"enabled"`
	Platform        string `json:"platform,omitempty"`
	SoftwareVersion string `json:"softwareVersion,omitempty"`
	PortID          string `json:"portId,omitempty"`
	Version         int    `json:"version,omitempty"`
	Holdtime        int    `json:"holdtime,omitempty"`
}

// LLDPResponse represents LLDP configuration.
type LLDPResponse struct {
	Enabled           bool   `json:"enabled"`
	ChassisIDType     string `json:"chassisIdType,omitempty"`
	PortDescription   string `json:"portDescription,omitempty"`
	SystemDescription string `json:"systemDescription,omitempty"`
	TTL               int    `json:"ttl,omitempty"`
}

// DHCPResponse represents DHCP server configuration.
type DHCPResponse struct {
	Enabled    bool     `json:"enabled"`
	PoolStart  string   `json:"poolStart,omitempty"`
	PoolEnd    string   `json:"poolEnd,omitempty"`
	SubnetMask string   `json:"subnetMask,omitempty"`
	Router     string   `json:"router,omitempty"`
	DNS        []string `json:"dns,omitempty"`
}

// DNSResponse represents DNS server configuration.
type DNSResponse struct {
	Enabled bool `json:"enabled"`
	Records int  `json:"recordCount,omitempty"`
}

// HTTPResponse represents HTTP server configuration.
type HTTPResponse struct {
	Enabled       bool   `json:"enabled"`
	ServerName    string `json:"serverName,omitempty"`
	EndpointCount int    `json:"endpointCount,omitempty"`
}

// FTPResponse represents FTP server configuration.
type FTPResponse struct {
	Enabled        bool   `json:"enabled"`
	WelcomeBanner  string `json:"welcomeBanner,omitempty"`
	AllowAnonymous bool   `json:"allowAnonymous,omitempty"`
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

// DeviceInterfaceResponse represents configured metadata for a device interface.
type DeviceInterfaceResponse struct {
	Name        string `json:"name"`
	Speed       int    `json:"speed,omitempty"`
	Duplex      string `json:"duplex,omitempty"`
	AdminStatus string `json:"adminStatus,omitempty"`
	OperStatus  string `json:"operStatus,omitempty"`
	Description string `json:"description,omitempty"`
	VLANs       []int  `json:"vlans,omitempty"`
}

// DeviceInterfaceUpdate represents partial update data for a device interface.
type DeviceInterfaceUpdate struct {
	Name        string `json:"name"`
	Speed       int    `json:"speed,omitempty"`
	Duplex      string `json:"duplex,omitempty"`
	AdminStatus string `json:"adminStatus,omitempty"`
	OperStatus  string `json:"operStatus,omitempty"`
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
	InterfaceDetails []DeviceInterfaceUpdate `json:"interfaceDetails,omitempty"`
	Template         string                  `json:"template,omitempty"` // Use template as base
	RawYAML          string                  `json:"rawYaml,omitempty"`  // Advanced: full YAML
	SNMPAgent        *SNMPAgentRequest       `json:"snmpAgent,omitempty"`
}

// DeviceUpdateRequest represents a request to update a device.
type DeviceUpdateRequest struct {
	Type             string                  `json:"type,omitempty"`
	MAC              string                  `json:"mac,omitempty"`
	IP               string                  `json:"ip,omitempty"`
	Interfaces       []DeviceInterfaceUpdate `json:"interfaces,omitempty"`
	InterfaceDetails []DeviceInterfaceUpdate `json:"interfaceDetails,omitempty"`
	RawYAML          string                  `json:"rawYaml,omitempty"` // Full YAML for the device
	SNMPAgent        *SNMPAgentRequest       `json:"snmpAgent,omitempty"`
}

// DeviceCloneRequest represents a request to clone a device.
type DeviceCloneRequest struct {
	NewHostname string `json:"newHostname"`
	NewIP       string `json:"newIp,omitempty"`
	NewMAC      string `json:"newMac,omitempty"`
}

// DeviceListResponse represents the response for listing devices.
type DeviceListResponse struct {
	Devices    []DeviceResponse `json:"devices"`
	TotalCount int              `json:"totalCount"`
}

// DeviceBatchDeleteRequest represents a request to delete multiple devices
// in a single call.
type DeviceBatchDeleteRequest struct {
	Hostnames []string `json:"hostnames"`
}

// DeviceBatchDeleteResult reports the outcome of deleting a single hostname
// as part of a batch delete request.
type DeviceBatchDeleteResult struct {
	Hostname string `json:"hostname"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

// DeviceBatchDeleteResponse represents the response for a batch device
// delete. The request succeeds (200) even when some hostnames fail;
// per-hostname outcomes are reported in Results.
type DeviceBatchDeleteResponse struct {
	Results []DeviceBatchDeleteResult `json:"results"`
	Deleted int                       `json:"deleted"`
	Failed  int                       `json:"failed"`
}
