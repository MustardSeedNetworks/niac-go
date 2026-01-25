package api

import (
	"encoding/json"
	"maps"
	"net/http"
)

// Schema validation constants.
const (
	maxHostnameLen          = 253     // RFC 1035 max domain name length
	lldpDefaultAdvertise    = 30      // LLDP default advertise interval seconds
	lldpDefaultTTL          = 120     // LLDP default TTL seconds
	cdpDefaultAdvertise     = 60      // CDP default advertise interval seconds
	cdpDefaultHoldtime      = 180     // CDP default holdtime seconds
	minHoldtime             = 10      // minimum holdtime seconds
	maxHoldtime             = 255     // maximum holdtime (byte max)
	maxAdvertiseInterval    = 3600    // max advertise interval (1 hour)
	maxTTLValue             = 65535   // max TTL value (uint16 max)
	stpDefaultHelloTime     = 2       // STP hello time default
	stpMaxHelloTime         = 10      // STP max hello time
	stpDefaultMaxAge        = 20      // STP max age default
	stpMinMaxAge            = 6       // STP min max age
	stpMaxMaxAge            = 40      // STP max max age
	stpDefaultForwardDelay  = 15      // STP forward delay default
	stpMinForwardDelay      = 4       // STP min forward delay
	stpMaxForwardDelay      = 30      // STP max forward delay
	stpDefaultPriority      = 32768   // STP default bridge priority
	httpDefaultStatusCode   = 200     // HTTP OK status
	httpMinStatusCode       = 100     // HTTP min status code
	httpMaxStatusCode       = 599     // HTTP max status code
	netbiosMaxNameLen       = 15      // NetBIOS name max length
	netbiosDefaultTTL       = 300     // NetBIOS default TTL seconds
	maxServerPreference     = 255     // max DHCPv6 server preference
	maxIPTTL                = 255     // max IP TTL value
	maxTCPWindowScale       = 14      // max TCP window scale option
	maxPacketLossPercent    = 100     // max packet loss percentage
	cdpDefaultVersion       = 2       // CDP default version
	edpDefaultAdvertise     = 30      // EDP default advertise interval
	fdpDefaultAdvertise     = 60      // FDP default advertise interval
	fdpDefaultHoldtime      = 180     // FDP default holdtime
	defaultDNSTTL           = 3600    // DNS default TTL (1 hour)
	defaultICMPTTL          = 64      // default ICMP TTL
	dhcpv6PreferredLifetime = 604800  // DHCPv6 preferred lifetime (7 days)
	dhcpv6ValidLifetime     = 2592000 // DHCPv6 valid lifetime (30 days)
	defaultPingInterval     = 120     // default periodic ping interval
	defaultPingPayload      = 32      // default ping payload size
	defaultTrafficInterval  = 180     // default random traffic interval
	defaultTrafficPacketCnt = 5       // default traffic packet count
	defaultIperfPort        = 5201    // default iperf port
	defaultIperfDuration    = 1000    // default iperf duration ms
	defaultBandwidthMbps    = 100     // default bandwidth in Mbps
)

// SchemaProperty represents a JSON Schema property definition.
type SchemaProperty struct {
	Type        string                     `json:"type"`
	Title       string                     `json:"title,omitempty"`
	Description string                     `json:"description,omitempty"`
	Default     any                        `json:"default,omitempty"`
	Enum        []any                      `json:"enum,omitempty"`
	EnumNames   []string                   `json:"enumNames,omitempty"`
	Minimum     *float64                   `json:"minimum,omitempty"`
	Maximum     *float64                   `json:"maximum,omitempty"`
	MinLength   *int                       `json:"minLength,omitempty"`
	MaxLength   *int                       `json:"maxLength,omitempty"`
	Pattern     string                     `json:"pattern,omitempty"`
	Format      string                     `json:"format,omitempty"`
	Items       *SchemaProperty            `json:"items,omitempty"`
	Properties  map[string]*SchemaProperty `json:"properties,omitempty"`
	Required    []string                   `json:"required,omitempty"`
	// UI hints for form rendering
	UIWidget      string `json:"ui:widget,omitempty"`
	UIHelp        string `json:"ui:help,omitempty"`
	UIPlaceholder string `json:"ui:placeholder,omitempty"`
}

// ConfigSchema represents the complete JSON Schema for device configuration.
type ConfigSchema struct {
	Schema      string                     `json:"$schema"`
	Type        string                     `json:"type"`
	Title       string                     `json:"title"`
	Description string                     `json:"description,omitempty"`
	Properties  map[string]*SchemaProperty `json:"properties"`
	Required    []string                   `json:"required"`
	Definitions map[string]*SchemaProperty `json:"definitions,omitempty"`
}

// handleConfigSchema returns the JSON Schema for device configuration
// GET /api/v1/config/schema.
func (s *Server) handleConfigSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	schema := buildDeviceSchema()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(schema)
	if err != nil {
		s.logger.Error("[API] Failed to encode schema response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// buildProtocolSchemaProperties returns all protocol-related schema properties.
func buildProtocolSchemaProperties() map[string]*SchemaProperty {
	minZero := 0.0
	maxPort := 65535.0
	maxPriority := 61440.0
	maxTTL := 255.0

	return map[string]*SchemaProperty{
		"snmp_agent":     buildSNMPAgentSchema(),
		"lldp":           buildLLDPSchema(),
		"cdp":            buildCDPSchema(),
		"edp":            buildEDPSchema(),
		"fdp":            buildFDPSchema(),
		"stp":            buildSTPSchema(&maxPriority),
		"dhcp":           buildDHCPSchema(),
		"dns":            buildDNSSchema(),
		"http":           buildHTTPSchema(&maxPort),
		"ftp":            buildFTPSchema(),
		"netbios":        buildNetBIOSSchema(&maxTTL),
		"icmp":           buildICMPSchema(&maxTTL),
		"icmpv6":         buildICMPv6Schema(&maxTTL),
		"dhcpv6":         buildDHCPv6Schema(),
		"traffic":        buildTrafficSchema(&minZero),
		"ttl":            buildTTLConfigSchema(&maxTTL),
		"os_fingerprint": buildOSFingerprintSchema(),
		"iperf3":         buildIPerf3Schema(&maxPort),
	}
}

// buildSchemaDefinitions returns the common type definitions for the schema.
func buildSchemaDefinitions() map[string]*SchemaProperty {
	return map[string]*SchemaProperty{
		"ipv4": {Type: "string", Format: "ipv4", Pattern: `^(\d{1,3}\.){3}\d{1,3}$`},
		"ipv6": {Type: "string", Format: "ipv6"},
		"mac":  {Type: "string", Pattern: `^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`},
	}
}

// buildDeviceSchema creates the JSON Schema for device configuration.
func buildDeviceSchema() *ConfigSchema {
	// Merge core and protocol properties
	properties := buildCoreDeviceProperties()
	maps.Copy(properties, buildProtocolSchemaProperties())

	return &ConfigSchema{
		Schema:      "http://json-schema.org/draft-07/schema#",
		Type:        "object",
		Title:       "Device Configuration",
		Description: "NIAC network device configuration schema",
		Required:    []string{"hostname", "mac"},
		Properties:  properties,
		Definitions: buildSchemaDefinitions(),
	}
}

// Helper functions for pointer values.
func floatPtr(v float64) *float64 {
	return &v
}

func intPtr(v int) *int {
	return &v
}
