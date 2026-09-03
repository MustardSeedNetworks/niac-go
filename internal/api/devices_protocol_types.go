package api

// This file declares the editable-request shape for every per-device protocol
// block the UI's Device type exposes (ui/src/api/device-config-types.ts),
// following the SNMPAgentRequest/SSHConfigRequest pattern already used by
// DeviceCreateRequest/DeviceUpdateRequest: one request struct per config.*
// type, applied wholesale when the pointer is non-nil so a partial update
// never has to guess which sub-fields the caller meant to leave alone.
//
// `protocols` is deliberately absent — it is server-computed
// (collectDeviceProtocols) and must never become a settable field.

// LLDPRequest is the editable LLDP subset accepted by device CRUD.
type LLDPRequest struct {
	Enabled           bool   `json:"enabled"`
	AdvertiseInterval int    `json:"advertiseInterval,omitempty"`
	TTL               int    `json:"ttl,omitempty"`
	ChassisIDType     string `json:"chassisIdType,omitempty"`
	SystemDescription string `json:"systemDescription,omitempty"`
	PortDescription   string `json:"portDescription,omitempty"`
}

// CDPRequest is the editable CDP subset accepted by device CRUD.
type CDPRequest struct {
	Enabled           bool   `json:"enabled"`
	AdvertiseInterval int    `json:"advertiseInterval,omitempty"`
	Holdtime          int    `json:"holdtime,omitempty"`
	Version           int    `json:"version,omitempty"`
	Platform          string `json:"platform,omitempty"`
	SoftwareVersion   string `json:"softwareVersion,omitempty"`
	PortID            string `json:"portId,omitempty"`
}

// EDPRequest is the editable EDP subset accepted by device CRUD.
type EDPRequest struct {
	Enabled           bool   `json:"enabled"`
	AdvertiseInterval int    `json:"advertiseInterval,omitempty"`
	VersionString     string `json:"versionString,omitempty"`
	DisplayString     string `json:"displayString,omitempty"`
}

// FDPRequest is the editable FDP subset accepted by device CRUD.
type FDPRequest struct {
	Enabled           bool   `json:"enabled"`
	AdvertiseInterval int    `json:"advertiseInterval,omitempty"`
	Holdtime          int    `json:"holdtime,omitempty"`
	SoftwareVersion   string `json:"softwareVersion,omitempty"`
	Platform          string `json:"platform,omitempty"`
	PortID            string `json:"portId,omitempty"`
}

// STPRequest is the editable STP subset accepted by device CRUD.
type STPRequest struct {
	Enabled        bool   `json:"enabled"`
	Version        string `json:"version,omitempty"`
	BridgePriority uint16 `json:"bridgePriority,omitempty"`
	HelloTime      uint16 `json:"helloTime,omitempty"`
	MaxAge         uint16 `json:"maxAge,omitempty"`
	ForwardDelay   uint16 `json:"forwardDelay,omitempty"`
}

// DHCPLeaseRequest is one static DHCP lease assignment.
type DHCPLeaseRequest struct {
	ClientIP   string `json:"clientIp"`
	MACAddress string `json:"macAddress"`
	MACMask    string `json:"macMask,omitempty"`
}

// DHCPRequest is the editable DHCP subset accepted by device CRUD.
// config.DHCPConfig has no Enabled field of its own — Enabled here only
// gates whether the pointer is created, same as TTLRequest/DNSRequest below.
type DHCPRequest struct {
	Enabled          bool               `json:"enabled"`
	SubnetMask       string             `json:"subnetMask,omitempty"`
	Router           string             `json:"router,omitempty"`
	DomainNameServer string             `json:"domainNameServer,omitempty"`
	ServerIdentifier string             `json:"serverIdentifier,omitempty"`
	NextServerIP     string             `json:"nextServerIp,omitempty"`
	PoolStart        string             `json:"poolStart,omitempty"`
	PoolEnd          string             `json:"poolEnd,omitempty"`
	DomainName       string             `json:"domainName,omitempty"`
	NTPServers       []string           `json:"ntpServers,omitempty"`
	DomainSearch     []string           `json:"domainSearch,omitempty"`
	TFTPServerName   string             `json:"tftpServerName,omitempty"`
	BootfileName     string             `json:"bootfileName,omitempty"`
	VendorSpecific   string             `json:"vendorSpecific,omitempty"` // hex-encoded
	ClientLeases     []DHCPLeaseRequest `json:"clientLeases,omitempty"`
}

// DHCPv6PoolRequest is one IPv6 address pool.
type DHCPv6PoolRequest struct {
	Network    string `json:"network"`
	RangeStart string `json:"rangeStart,omitempty"`
	RangeEnd   string `json:"rangeEnd,omitempty"`
}

// DHCPv6Request is the editable DHCPv6 subset accepted by device CRUD.
type DHCPv6Request struct {
	Enabled           bool                `json:"enabled"`
	Pools             []DHCPv6PoolRequest `json:"pools,omitempty"`
	PreferredLifetime uint32              `json:"preferredLifetime,omitempty"`
	ValidLifetime     uint32              `json:"validLifetime,omitempty"`
	Preference        uint8               `json:"preference,omitempty"`
	DNSServers        []string            `json:"dnsServers,omitempty"`
	DomainList        []string            `json:"domainList,omitempty"`
	SNTPServers       []string            `json:"sntpServers,omitempty"`
	NTPServers        []string            `json:"ntpServers,omitempty"`
	SIPServers        []string            `json:"sipServers,omitempty"`
	SIPDomains        []string            `json:"sipDomains,omitempty"`
}

// DNSRecordRequest is one DNS A or PTR record.
type DNSRecordRequest struct {
	Name  string `json:"name"`
	IP    string `json:"ip"`
	TTL   uint32 `json:"ttl,omitempty"`
	RCode int    `json:"rcode,omitempty"`
}

// DNSRequest is the editable DNS subset accepted by device CRUD.
// config.DNSConfig has no Enabled field of its own — see DHCPRequest.
type DNSRequest struct {
	Enabled        bool               `json:"enabled"`
	ForwardRecords []DNSRecordRequest `json:"forwardRecords,omitempty"`
	ReverseRecords []DNSRecordRequest `json:"reverseRecords,omitempty"`
}

// HTTPEndpointRequest is one custom HTTP endpoint definition.
type HTTPEndpointRequest struct {
	Path        string `json:"path"`
	Method      string `json:"method,omitempty"`
	StatusCode  int    `json:"statusCode,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Body        string `json:"body,omitempty"`
}

// HTTPRequest is the editable HTTP subset accepted by device CRUD.
type HTTPRequest struct {
	Enabled    bool                  `json:"enabled"`
	ServerName string                `json:"serverName,omitempty"`
	Endpoints  []HTTPEndpointRequest `json:"endpoints,omitempty"`
}

// FTPUserRequest is one FTP user account.
type FTPUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	HomeDir  string `json:"homeDir,omitempty"`
}

// FTPRequest is the editable FTP subset accepted by device CRUD.
type FTPRequest struct {
	Enabled        bool             `json:"enabled"`
	WelcomeBanner  string           `json:"welcomeBanner,omitempty"`
	SystemType     string           `json:"systemType,omitempty"`
	AllowAnonymous bool             `json:"allowAnonymous,omitempty"`
	Users          []FTPUserRequest `json:"users,omitempty"`
}

// NetBIOSRequest is the editable NetBIOS subset accepted by device CRUD.
// The optional per-name status table (config.NetBIOSConfig.Names) is not
// exposed here: it is a rarely-used advanced field and its suffix wire
// format is not settled in the UI type, so wiring it now would be guessing.
type NetBIOSRequest struct {
	Enabled   bool     `json:"enabled"`
	Name      string   `json:"name,omitempty"`
	Workgroup string   `json:"workgroup,omitempty"`
	NodeType  string   `json:"nodeType,omitempty"`
	Services  []string `json:"services,omitempty"`
	TTL       uint32   `json:"ttl,omitempty"`
	MsBrowse  bool     `json:"msbrowse,omitempty"`
}

// ICMPRequest is the editable ICMP (v4) subset accepted by device CRUD.
// RouterAdvertisement is not exposed here: it is a nested, rarely-edited
// sub-object and no failing subtest requires it.
type ICMPRequest struct {
	Enabled          bool   `json:"enabled"`
	TTL              uint8  `json:"ttl,omitempty"`
	RateLimit        int    `json:"rateLimit,omitempty"`
	AddressMaskReply string `json:"addressMaskReply,omitempty"`
}

// ICMPv6Request is the editable ICMPv6 subset accepted by device CRUD.
// RouterAdvertisement is not exposed here for the same reason as ICMPRequest.
type ICMPv6Request struct {
	Enabled   bool  `json:"enabled"`
	HopLimit  uint8 `json:"hopLimit,omitempty"`
	RateLimit int   `json:"rateLimit,omitempty"`
}

// TTLRequest is the editable device-level TTL (traceroute simulation)
// subset accepted by device CRUD. config.TTLConfig has no Enabled field of
// its own — see DHCPRequest.
type TTLRequest struct {
	Enabled bool   `json:"enabled"`
	TTL     int    `json:"ttl,omitempty"`
	IP      string `json:"ip,omitempty"`
	Mask    string `json:"mask,omitempty"`
}

// OSFingerprintRequest is the editable OS fingerprint subset accepted by
// device CRUD. config.OSFingerprintConfig has no Enabled field of its own —
// see DHCPRequest.
type OSFingerprintRequest struct {
	Enabled      bool   `json:"enabled"`
	OSType       string `json:"osType,omitempty"`
	TTL          uint8  `json:"ttl,omitempty"`
	WindowSize   uint16 `json:"windowSize,omitempty"`
	WindowScale  uint8  `json:"windowScale,omitempty"`
	MSS          uint16 `json:"mss,omitempty"`
	SSHBanner    string `json:"sshBanner,omitempty"`
	HTTPServer   string `json:"httpServer,omitempty"`
	FTPBanner    string `json:"ftpBanner,omitempty"`
	SMTPBanner   string `json:"smtpBanner,omitempty"`
	TelnetBanner string `json:"telnetBanner,omitempty"`
	DontFragment bool   `json:"dontFragment,omitempty"`
}

// IPerf3Request is the editable iPerf3 server subset accepted by device CRUD.
type IPerf3Request struct {
	Enabled           bool    `json:"enabled"`
	Port              uint16  `json:"port,omitempty"`
	MaxBandwidthMbps  float64 `json:"maxBandwidthMbps,omitempty"`
	TypicalLatencyMs  float64 `json:"typicalLatencyMs,omitempty"`
	JitterMs          float64 `json:"jitterMs,omitempty"`
	PacketLossPercent float64 `json:"packetLossPercent,omitempty"`
	UploadMbps        float64 `json:"uploadMbps,omitempty"`
	DownloadMbps      float64 `json:"downloadMbps,omitempty"`
}
