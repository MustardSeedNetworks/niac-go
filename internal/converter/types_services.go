package converter

import "github.com/invopop/jsonschema"

// SSHConfig enables authenticated vendor-like command sessions.
type SSHConfig struct {
	// Enabled serves the SSH listener. When true, username and password_env
	// are both required.
	Enabled bool `yaml:"enabled"`

	// Username is the account the simulated CLI accepts.
	Username string `yaml:"username,omitempty"`

	// PasswordEnv names the environment variable holding the password — the
	// password itself is never written in the config. That variable must be
	// set in the daemon's environment or the device fails to start.
	PasswordEnv string `yaml:"password_env,omitempty" jsonschema:"pattern=^[A-Za-z_][A-Za-z0-9_]*$"`
}

// JSONSchemaExtend requires SSH credentials only when the service is enabled.
func (SSHConfig) JSONSchemaExtend(schema *jsonschema.Schema) {
	addEnabledRequirements(schema, "username", "password_env")
	username, found := schema.Properties.Get("username")
	if found {
		username.Pattern = `.*\S.*`
	}
}

// SyslogConfig sends configuration-state messages to RFC 5424 collectors.
type SyslogConfig struct {
	// Enabled turns on syslog emission for this device. When true, at least
	// one receiver is required.
	Enabled bool `yaml:"enabled"`

	// Receivers are collector endpoints as "host:port", for example
	// 10.10.0.9:514.
	Receivers []string `yaml:"receivers,omitempty"`
}

// JSONSchemaExtend requires collectors only when SYSLOG is enabled.
func (SyslogConfig) JSONSchemaExtend(schema *jsonschema.Schema) {
	addEnabledRequirements(schema, "receivers")
	receivers, found := schema.Properties.Get("receivers")
	if found {
		minimum := uint64(1)
		receivers.MinItems = &minimum
		octet := `(?:25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])`
		port := `(?:[1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])`
		receivers.Items.Pattern = `^(?:` + octet + `\.){3}` + octet + `:` + port + `$`
	}
}

func addEnabledRequirements(schema *jsonschema.Schema, required ...string) {
	properties := jsonschema.NewProperties()
	properties.Set("enabled", &jsonschema.Schema{Const: true})
	schema.AllOf = append(schema.AllOf, &jsonschema.Schema{
		If:   &jsonschema.Schema{Properties: properties, Required: []string{"enabled"}},
		Then: &jsonschema.Schema{Required: required},
	})
}

// IPerf3Config represents iPerf3 server emulation configuration.
type IPerf3Config struct {
	// Enabled serves the iperf3 listener.
	Enabled bool `yaml:"enabled,omitempty"`

	// Port is the TCP port iperf3 listens on. Omit for the default, 5201.
	Port uint16 `yaml:"port,omitempty" validate:"omitempty,gte=1"`

	// MaxBandwidthMbps caps the throughput the device will report.
	MaxBandwidthMbps float64 `yaml:"max_bandwidth_mbps,omitempty"`

	// TypicalLatencyMs is the round-trip latency reported to the tester.
	TypicalLatencyMs float64 `yaml:"typical_latency_ms,omitempty"`

	// JitterMs is the variation reported around the typical latency.
	JitterMs float64 `yaml:"jitter_ms,omitempty"`

	// PacketLossPercent is the loss rate reported to the tester, 0..100.
	PacketLossPercent float64 `yaml:"packet_loss_percent,omitempty"`

	// UploadMbps is the reported client-to-server rate.
	UploadMbps float64 `yaml:"upload_mbps,omitempty"`

	// DownloadMbps is the reported server-to-client rate.
	DownloadMbps float64 `yaml:"download_mbps,omitempty"`
}

// ReflectorConfig represents a NetAlly-style UDP reflector endpoint. When
// present, the device echoes signed reflector probes (TrueSpeed / performance
// tests) back to the sender with source/destination swapped, matching the
// niac-java Reflector() section. The presence of this block enables the
// reflector; there is no separate enable flag (Java sets isReflector on the
// section being parsed).
type ReflectorConfig struct {
	// LatencyMs delays the reflected packet by this many milliseconds,
	// simulating one-way path latency (Java Latency()). 0 = reflect
	// immediately.
	LatencyMs int `yaml:"latency_ms,omitempty" validate:"omitempty,gte=0,lte=60000"`

	// JitterMs randomises the delay by +/- this many milliseconds around
	// LatencyMs (Java Jitter()). 0 = no jitter.
	JitterMs int `yaml:"jitter_ms,omitempty" validate:"omitempty,gte=0,lte=60000"`

	// DSCP selects which ToS bits are toggled on the reflected packet:
	// true wiggles the bottom two DSCP bits (0x03, Java Dscp), false
	// wiggles the IP-precedence bit (0x01, Java IpPrecedence — the default).
	DSCP bool `yaml:"dscp,omitempty"`
}

// OSFingerprintConfig represents OS fingerprinting configuration for device simulation.
type OSFingerprintConfig struct {
	// OSType is the operating system to imitate, for example "linux",
	// "windows", "cisco-ios" or "juniper-junos".
	OSType string `yaml:"os_type,omitempty"`

	// TTL is the default IP TTL stamped on outbound packets, which is a
	// primary fingerprinting signal (Linux 64, Windows 128, Cisco 255).
	TTL uint8 `yaml:"ttl,omitempty"`

	// WindowSize is the advertised TCP window size.
	WindowSize uint16 `yaml:"window_size,omitempty"`

	// WindowScale is the TCP window scale option.
	WindowScale uint8 `yaml:"window_scale,omitempty"`

	// MSS is the TCP maximum segment size.
	MSS uint16 `yaml:"mss,omitempty"`

	// SSHBanner is the SSH version banner presented on connect.
	SSHBanner string `yaml:"ssh_banner,omitempty"`

	// HTTPServer is the HTTP Server response header.
	HTTPServer string `yaml:"http_server,omitempty"`

	// FTPBanner is the FTP welcome banner.
	FTPBanner string `yaml:"ftp_banner,omitempty"`

	// SMTPBanner is the SMTP greeting banner.
	SMTPBanner string `yaml:"smtp_banner,omitempty"`

	// TelnetBanner is the Telnet greeting banner.
	TelnetBanner string `yaml:"telnet_banner,omitempty"`

	// DontFragment sets the IP DF bit, another fingerprinting signal
	// (typically true on Linux, false on Windows).
	DontFragment bool `yaml:"dont_fragment,omitempty"`
}

// HTTPConfig represents HTTP server configuration.
type HTTPConfig struct {
	// Enabled serves the HTTP listener.
	Enabled bool `yaml:"enabled,omitempty"`

	// ServerName is the Server response header, a signal a scanner uses to
	// identify the platform.
	ServerName string `yaml:"server_name,omitempty"`

	// Endpoints are the paths this server answers. A request to any other
	// path gets 404.
	Endpoints []HTTPEndpoint `yaml:"endpoints,omitempty"`
}

// HTTPEndpoint represents an HTTP endpoint configuration.
type HTTPEndpoint struct {
	// Path is the request path, for example /status.
	Path string `yaml:"path,omitempty"`

	// Method is the HTTP method matched. Omit for GET.
	Method string `yaml:"method,omitempty"`

	// StatusCode is the response status. Omit for 200.
	StatusCode int `yaml:"status_code,omitempty"`

	// ContentType is the response Content-Type header.
	ContentType string `yaml:"content_type,omitempty"`

	// Body is the response body served verbatim.
	Body string `yaml:"body,omitempty"`
}

// FtpConfig represents FTP server configuration.
type FtpConfig struct {
	// Enabled serves the FTP listener.
	Enabled bool `yaml:"enabled,omitempty"`

	// WelcomeBanner is the 220 greeting, a primary identification signal.
	WelcomeBanner string `yaml:"welcome_banner,omitempty"`

	// SystemType is the SYST response, for example "UNIX Type: L8".
	SystemType string `yaml:"system_type,omitempty"`

	// AllowAnonymous accepts the anonymous account without a password.
	AllowAnonymous bool `yaml:"allow_anonymous,omitempty"`

	// Users are the accounts the server accepts.
	Users []FtpUser `yaml:"users,omitempty"`
}

// FtpUser represents an FTP user account.
type FtpUser struct {
	// Username is the account name.
	Username string `yaml:"username,omitempty"`

	// Password is the account password. Unlike SSH, this is written in the
	// config — use only simulated credentials, never a real one.
	Password string `yaml:"password,omitempty"`

	// HomeDir is the directory the account lands in.
	HomeDir string `yaml:"home_dir,omitempty"`
}

// MdnsConfig publishes a device on multicast DNS, the way Bonjour and Avahi
// announce a host and its services on the local link.
type MdnsConfig struct {
	// Enabled publishes this device over mDNS.
	Enabled bool `yaml:"enabled,omitempty"`

	// Hostname is the advertised name, without the .local suffix.
	Hostname string `yaml:"hostname,omitempty"`

	// Services are the DNS-SD services advertised, which is what makes a
	// printer or a camera discoverable as such.
	Services []MdnsService `yaml:"services,omitempty"`

	// TTL is the seconds a resolver should cache these records.
	TTL uint32 `yaml:"ttl,omitempty"`
}

// MdnsService is one advertised DNS-SD service, such as _ipp._tcp.
type MdnsService struct {
	// Type is the DNS-SD service type, for example _ipp._tcp or _rtsp._tcp.
	Type string `yaml:"type"`

	// Port is the TCP or UDP port the service is advertised on.
	Port uint16 `yaml:"port"`

	// TXT are the service's TXT record key=value strings.
	TXT []string `yaml:"txt,omitempty"`
}

// NetbiosConfig represents NetBIOS service configuration.
type NetbiosConfig struct {
	// Enabled answers NetBIOS name queries.
	Enabled bool `yaml:"enabled,omitempty"`

	// Name is the NetBIOS name, at most 15 characters — longer names are
	// truncated on the wire, so authored names should already be short.
	Name string `yaml:"name,omitempty"`

	// Workgroup is the advertised workgroup or domain.
	Workgroup string `yaml:"workgroup,omitempty"`

	// NodeType is the NetBIOS node type, for example B, P, M or H.
	NodeType string `yaml:"node_type,omitempty"`

	// Services are the advertised service suffixes.
	Services []string `yaml:"services,omitempty"`

	// TTL is the seconds a resolver should cache the name.
	TTL uint32 `yaml:"ttl,omitempty"`

	// Names are additional NetBIOS name entries served by this device.
	Names []NetbiosName `yaml:"names,omitempty"`

	// MsBrowse advertises the __MSBROWSE__ entry, marking the device as a
	// master browser.
	MsBrowse bool `yaml:"msbrowse,omitempty"`
}

// NetbiosName represents a NetBIOS name entry.
type NetbiosName struct {
	// Name is the entry's NetBIOS name, at most 15 characters.
	Name string `yaml:"name,omitempty"`

	// Suffix is the two-hex-digit name suffix, for example 00 (workstation)
	// or 20 (file server).
	Suffix string `yaml:"suffix,omitempty"`

	// Group marks the entry as a group name rather than a unique name.
	Group bool `yaml:"group,omitempty"`
}
