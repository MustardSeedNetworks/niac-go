package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildDeviceSchema(t *testing.T) {
	schema := buildDeviceSchema()

	if schema == nil {
		t.Fatal("buildDeviceSchema returned nil")
	}

	if schema.Type != "object" {
		t.Errorf("schema type = %q, want %q", schema.Type, "object")
	}

	if schema.Title != "Device Configuration" {
		t.Errorf("schema title = %q, want %q", schema.Title, "Device Configuration")
	}

	// Check required fields
	foundHostname := false
	foundMAC := false
	for _, req := range schema.Required {
		if req == "hostname" {
			foundHostname = true
		}
		if req == "mac" {
			foundMAC = true
		}
	}
	if !foundHostname {
		t.Error("hostname should be required")
	}
	if !foundMAC {
		t.Error("mac should be required")
	}

	// Check core properties exist
	coreProps := []string{"hostname", "mac", "ip", "type", "vlan", "babble"}
	for _, prop := range coreProps {
		if _, ok := schema.Properties[prop]; !ok {
			t.Errorf("missing core property: %s", prop)
		}
	}

	// Check protocol properties exist
	protoProps := []string{"snmp_agent", "lldp", "cdp", "stp", "dhcp", "dns", "http", "ftp", "netbios"}
	for _, prop := range protoProps {
		if _, ok := schema.Properties[prop]; !ok {
			t.Errorf("missing protocol property: %s", prop)
		}
	}

	// Check definitions
	if schema.Definitions == nil {
		t.Error("definitions should not be nil")
	}
	defTypes := []string{"ipv4", "ipv6", "mac"}
	for _, def := range defTypes {
		if _, ok := schema.Definitions[def]; !ok {
			t.Errorf("missing definition: %s", def)
		}
	}
}

func TestBuildCoreDeviceProperties(t *testing.T) {
	props := buildCoreDeviceProperties()

	if props == nil {
		t.Fatal("buildCoreDeviceProperties returned nil")
	}

	hostname := props["hostname"]
	if hostname == nil {
		t.Fatal("missing hostname property")
	}
	if hostname.Type != "string" {
		t.Errorf("hostname type = %q, want %q", hostname.Type, "string")
	}
	if hostname.Pattern == "" {
		t.Error("hostname should have a pattern")
	}

	mac := props["mac"]
	if mac == nil {
		t.Fatal("missing mac property")
	}
	if mac.Pattern == "" {
		t.Error("mac should have a pattern")
	}

	deviceType := props["type"]
	if deviceType == nil {
		t.Fatal("missing type property")
	}
	if len(deviceType.Enum) == 0 {
		t.Error("type should have enum values")
	}
}

func TestBuildDeviceTypeSchema(t *testing.T) {
	schema := buildDeviceTypeSchema()

	if schema == nil {
		t.Fatal("buildDeviceTypeSchema returned nil")
	}
	if schema.Type != "string" {
		t.Errorf("type = %q, want %q", schema.Type, "string")
	}
	if len(schema.Enum) == 0 {
		t.Error("should have enum values")
	}
	if len(schema.EnumNames) == 0 {
		t.Error("should have enum names")
	}
	if len(schema.Enum) != len(schema.EnumNames) {
		t.Errorf("enum count %d != enum names count %d", len(schema.Enum), len(schema.EnumNames))
	}
}

func TestBuildProtocolSchemaProperties(t *testing.T) {
	props := buildProtocolSchemaProperties()

	expectedProtocols := []string{
		"snmp_agent", "lldp", "cdp", "edp", "fdp", "stp",
		"dhcp", "dns", "http", "ftp", "netbios",
		"icmp", "icmpv6", "dhcpv6", "traffic", "ttl",
		"os_fingerprint", "iperf3",
	}

	for _, proto := range expectedProtocols {
		if _, ok := props[proto]; !ok {
			t.Errorf("missing protocol schema: %s", proto)
		}
	}
}

func TestBuildSchemaDefinitions(t *testing.T) {
	defs := buildSchemaDefinitions()

	if defs["ipv4"] == nil {
		t.Error("missing ipv4 definition")
	}
	if defs["ipv6"] == nil {
		t.Error("missing ipv6 definition")
	}
	if defs["mac"] == nil {
		t.Error("missing mac definition")
	}
}

func TestHandleConfigSchema(t *testing.T) {
	server, _ := newTestServer(t)

	t.Run("GET returns schema", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/config/schema", nil)
		server.handleConfigSchema(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		ct := rec.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}

		var schema ConfigSchema
		if err := json.NewDecoder(rec.Body).Decode(&schema); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if schema.Type != "object" {
			t.Errorf("schema type = %q, want %q", schema.Type, "object")
		}
	})

	t.Run("POST not allowed", func(t *testing.T) {
		// Method gating moved from the handler to the route registry (ADR-0002).
		// Exercise the methodGate wrapper register() composes for the GET-only
		// /api/v1/config/schema route.
		gated := server.methodGate([]string{http.MethodGet}, server.handleConfigSchema)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/config/schema", nil)
		gated(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})
}

func TestFloatPtr(t *testing.T) {
	v := floatPtr(3.14)
	if v == nil {
		t.Fatal("floatPtr returned nil")
	}
	if *v != 3.14 {
		t.Errorf("*floatPtr = %f, want %f", *v, 3.14)
	}
}

func TestIntPtr(t *testing.T) {
	v := intPtr(42)
	if v == nil {
		t.Fatal("intPtr returned nil")
	}
	if *v != 42 {
		t.Errorf("*intPtr = %d, want %d", *v, 42)
	}
}

func TestSchemaPropertyJSONSerialization(t *testing.T) {
	prop := SchemaProperty{
		Type:        "string",
		Title:       "Test",
		Description: "A test property",
		Default:     "hello",
		Pattern:     "^[a-z]+$",
	}

	data, err := json.Marshal(prop)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded SchemaProperty
	if err = json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Type != "string" {
		t.Errorf("Type = %q, want %q", decoded.Type, "string")
	}
	if decoded.Title != "Test" {
		t.Errorf("Title = %q, want %q", decoded.Title, "Test")
	}
}

// Test discovery protocol schemas.
func TestBuildSNMPAgentSchema(t *testing.T) {
	schema := buildSNMPAgentSchema()
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	requiredProps := []string{"community", "sys_name", "walk_file", "add_mibs"}
	for _, prop := range requiredProps {
		if _, ok := schema.Properties[prop]; !ok {
			t.Errorf("missing property: %s", prop)
		}
	}
}

func TestBuildLLDPSchema(t *testing.T) {
	schema := buildLLDPSchema()
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if _, ok := schema.Properties["enabled"]; !ok {
		t.Error("missing enabled property")
	}
	if _, ok := schema.Properties["ttl"]; !ok {
		t.Error("missing ttl property")
	}
}

func TestBuildCDPSchema(t *testing.T) {
	schema := buildCDPSchema()
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if _, ok := schema.Properties["platform"]; !ok {
		t.Error("missing platform property")
	}
}

func TestBuildSTPSchema(t *testing.T) {
	maxPriority := 61440.0
	schema := buildSTPSchema(&maxPriority)
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	bp := schema.Properties["bridge_priority"]
	if bp == nil {
		t.Fatal("missing bridge_priority property")
	}
	if bp.Maximum == nil || *bp.Maximum != maxPriority {
		t.Error("bridge_priority maximum not set correctly")
	}
}

func TestBuildDHCPSchema(t *testing.T) {
	schema := buildDHCPSchema()
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	props := []string{"subnet_mask", "router", "pool_start", "pool_end"}
	for _, p := range props {
		if _, ok := schema.Properties[p]; !ok {
			t.Errorf("missing property: %s", p)
		}
	}
}

func TestBuildDNSSchema(t *testing.T) {
	schema := buildDNSSchema()
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if _, ok := schema.Properties["forward_records"]; !ok {
		t.Error("missing forward_records")
	}
	if _, ok := schema.Properties["reverse_records"]; !ok {
		t.Error("missing reverse_records")
	}
}

func TestBuildHTTPSchema(t *testing.T) {
	maxPort := 65535.0
	schema := buildHTTPSchema(&maxPort)
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if _, ok := schema.Properties["enabled"]; !ok {
		t.Error("missing enabled property")
	}
	if _, ok := schema.Properties["endpoints"]; !ok {
		t.Error("missing endpoints property")
	}
}

func TestBuildICMPSchema(t *testing.T) {
	maxTTL := 255.0
	schema := buildICMPSchema(&maxTTL)
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if _, ok := schema.Properties["ttl"]; !ok {
		t.Error("missing ttl property")
	}
}

func TestBuildTrafficSchema(t *testing.T) {
	minZero := 0.0
	schema := buildTrafficSchema(&minZero)
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	subProps := []string{"arp_announcements", "periodic_pings", "random_traffic"}
	for _, p := range subProps {
		if _, ok := schema.Properties[p]; !ok {
			t.Errorf("missing property: %s", p)
		}
	}
}

func TestBuildIPerf3Schema(t *testing.T) {
	maxPort := 65535.0
	schema := buildIPerf3Schema(&maxPort)
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if _, ok := schema.Properties["port"]; !ok {
		t.Error("missing port property")
	}
	if _, ok := schema.Properties["max_bandwidth_mbps"]; !ok {
		t.Error("missing max_bandwidth_mbps property")
	}
}

func TestBuildOSFingerprintSchema(t *testing.T) {
	schema := buildOSFingerprintSchema()
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if _, ok := schema.Properties["os_type"]; !ok {
		t.Error("missing os_type property")
	}
	osType := schema.Properties["os_type"]
	if len(osType.Enum) == 0 {
		t.Error("os_type should have enum values")
	}
}

func TestBuildEDPSchema(t *testing.T) {
	schema := buildEDPSchema()
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if _, ok := schema.Properties["enabled"]; !ok {
		t.Error("missing enabled property")
	}
}

func TestBuildFDPSchema(t *testing.T) {
	schema := buildFDPSchema()
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if _, ok := schema.Properties["holdtime"]; !ok {
		t.Error("missing holdtime property")
	}
}

func TestBuildFTPSchema(t *testing.T) {
	schema := buildFTPSchema()
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if _, ok := schema.Properties["allow_anonymous"]; !ok {
		t.Error("missing allow_anonymous property")
	}
}

func TestBuildNetBIOSSchema(t *testing.T) {
	maxTTL := 255.0
	schema := buildNetBIOSSchema(&maxTTL)
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if _, ok := schema.Properties["name"]; !ok {
		t.Error("missing name property")
	}
	if _, ok := schema.Properties["workgroup"]; !ok {
		t.Error("missing workgroup property")
	}
}

func TestBuildICMPv6Schema(t *testing.T) {
	maxTTL := 255.0
	schema := buildICMPv6Schema(&maxTTL)
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if _, ok := schema.Properties["hop_limit"]; !ok {
		t.Error("missing hop_limit property")
	}
}

func TestBuildDHCPv6Schema(t *testing.T) {
	schema := buildDHCPv6Schema()
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if _, ok := schema.Properties["pools"]; !ok {
		t.Error("missing pools property")
	}
}

func TestBuildTTLConfigSchema(t *testing.T) {
	maxTTL := 255.0
	schema := buildTTLConfigSchema(&maxTTL)
	if schema.Type != "object" {
		t.Errorf("type = %q, want %q", schema.Type, "object")
	}
	if _, ok := schema.Properties["ttl"]; !ok {
		t.Error("missing ttl property")
	}
}
