package api

import (
	"net/http"
	"strings"
)

// DeviceEditorSchema is the per-type form schema the UI device
// editor consumes to hide irrelevant sections (#546 part 1). The
// frontend renders sections in a fixed order; this schema tells it
// which ones apply to the picked device type.
//
// Sections are referenced by the same string keys the editor uses
// internally ("snmp", "lldp", "cdp", "stp", "dhcp", "dns", "http",
// "ftp", "netbios", "ips", "basic"). "basic" is always
// visible — the device's name and type live there.
type DeviceEditorSchema struct {
	Type            string   `json:"type"`
	Label           string   `json:"label"`
	VisibleSections []string `json:"visibleSections"`
}

// deviceEditorSchemas is the canonical map from device type → which
// sections render. Picked to match common operator expectations: a
// switch shouldn't see a DNS server tab, a host shouldn't see STP,
// etc. The list is intentionally small — niac's device types stay
// in the same handful and adding a new one requires touching exactly
// this map.
//
// Per-type lists are kept in render-order so the UI iterating
// visibleSections in order Just Works.
var deviceEditorSchemas = map[string]DeviceEditorSchema{ //nolint:gochecknoglobals // static lookup table
	"switch": {
		Type:  "switch",
		Label: "Switch",
		VisibleSections: []string{
			"basic", "snmp", "lldp", "cdp", "stp", "ips",
		},
	},
	"router": {
		Type:  "router",
		Label: "Router",
		VisibleSections: []string{
			"basic", "snmp", "lldp", "cdp", "ips", "dhcp",
		},
	},
	"host": {
		Type:  "host",
		Label: "Host / Workstation",
		VisibleSections: []string{
			"basic", "snmp", "lldp", "ips", "dns", "http", "ftp", "netbios",
		},
	},
	"server": {
		Type:  "server",
		Label: "Server",
		VisibleSections: []string{
			"basic", "snmp", "lldp", "ips", "dns", "dhcp", "http", "ftp", "netbios",
		},
	},
	"firewall": {
		Type:  "firewall",
		Label: "Firewall",
		VisibleSections: []string{
			"basic", "snmp", "lldp", "ips",
		},
	},
	"access_point": {
		Type:  "access_point",
		Label: "Wireless Access Point",
		VisibleSections: []string{
			"basic", "snmp", "lldp", "ips",
		},
	},
	"printer": {
		Type:  "printer",
		Label: "Printer",
		VisibleSections: []string{
			"basic", "snmp", "ips",
		},
	},
	"voip_phone": {
		Type:  "voip_phone",
		Label: "VoIP Phone",
		VisibleSections: []string{
			"basic", "snmp", "lldp", "cdp", "ips",
		},
	},
	"unknown": {
		Type:  "unknown",
		Label: "Unknown / Custom",
		// Fallback shows everything so users with non-standard types
		// can still configure any field they need.
		VisibleSections: []string{
			"basic", "snmp", "lldp", "cdp", "stp", "ips", "dhcp", "dns", "http", "ftp", "netbios",
		},
	},
}

// handleDeviceEditorSchema handles GET /api/v1/device-schemas/{type}
// and GET /api/v1/device-schemas (list).
//
// The list endpoint returns every known schema so the frontend can
// build a type picker without round-tripping per type. Both endpoints
// always return JSON — unknown types fall back to the "unknown"
// schema rather than 404'ing, so a type new to the daemon that the UI
// hasn't been updated for still gives a usable form.
func (s *Server) handleDeviceEditorSchema(w http.ResponseWriter, r *http.Request) {
	// Method gating (GET-only) is enforced declaratively by the route registry.
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/device-schemas")
	rest = strings.TrimPrefix(rest, "/")

	if rest == "" {
		s.writeJSON(w, listDeviceEditorSchemas())
		return
	}

	schema := lookupDeviceEditorSchema(rest)
	s.writeJSON(w, schema)
}

// listDeviceEditorSchemas returns every defined schema in
// alphabetical-by-type order so the type picker in the UI has a
// stable display order.
func listDeviceEditorSchemas() []DeviceEditorSchema {
	out := make([]DeviceEditorSchema, 0, len(deviceEditorSchemas))
	for _, schema := range deviceEditorSchemas {
		out = append(out, schema)
	}
	// Stable sort by Type so list responses match between calls.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1].Type > out[j].Type; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// lookupDeviceEditorSchema returns the per-type schema, falling back
// to the "unknown" schema (show everything) when the type isn't in
// the map. Type names are normalised lowercase before lookup so the
// frontend can pass either "switch" or "Switch".
func lookupDeviceEditorSchema(deviceType string) DeviceEditorSchema {
	key := strings.ToLower(strings.TrimSpace(deviceType))
	if schema, ok := deviceEditorSchemas[key]; ok {
		return schema
	}
	return deviceEditorSchemas["unknown"]
}
