// Package builtins discovers, parses and loads the built-in scenarios: the
// hand-authored YAML files shipped under cmd/niac/templates (and the
// installed, per-user and NIAC_TEMPLATES_DIR copies of that tree). It
// parses their front-matter into catalogue metadata and copies a chosen
// scenario into a saved config, with no dependency on the api transport
// layer, which composes it inward (ADR-0006).
//
// This is distinct from internal/templates, the embedded library compiled
// into the binary for the CLI. This leaf is the on-disk engine behind the
// /api/v1/scenario/builtins surface and the daemon's StartSimulation path.
package builtins

import "os"

// Scenario type constants matching the UI interface.
const (
	typeBasic       = "basic"
	typeRouter      = "router"
	typeSwitch      = "switch"
	typeAccessPoint = "access-point"
	typeServer      = "server"
	typeFirewall    = "firewall"
	typeComplete    = "complete"
	typeCustom      = "custom"
)

// Scenario is one built-in scenario's catalogue entry.
//
// Name is the stable filename-derived identifier ("minimal", "router")
// used by /api/v1/scenario/builtins/{name} and /api/v1/scenario/builtins/copy.
// DisplayName is the human-readable label optionally provided via the
// front-matter "# Display: ..." line; clients should prefer it for UI
// rendering and fall back to Name when absent.
type Scenario struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description"`
	DeviceCount int    `json:"deviceCount"`
	Type        string `json:"type"`
	// Vendor is populated from the scenario's "# Vendor: ..."
	// front-matter, lowercased. When non-empty the UI groups the
	// scenario under a vendor heading instead of (or alongside) the
	// generic type-based grouping — useful for the vendor scenario
	// pack shipped under cmd/niac/templates/vendor-templates/.
	Vendor string   `json:"vendor,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

// Content represents the full content of a scenario.
type Content struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Format  string `json:"format"`
}

// Dirs returns the directories to scan for scenarios. It checks multiple
// locations for compatibility with both development and installed
// deployments. A NIAC_TEMPLATES_DIR override, when set, takes precedence.
func Dirs() []string {
	dirs := []string{
		// Development paths (relative to working directory)
		"cmd/niac/templates",
		"examples",
		// System-wide installed paths
		"/usr/share/niac/templates",
		"/var/lib/niac/templates",
		// User-specific paths
		os.ExpandEnv("$HOME/.niac/templates"),
	}

	// Add custom directory from environment variable
	if customDir := os.Getenv("NIAC_TEMPLATES_DIR"); customDir != "" {
		dirs = append([]string{customDir}, dirs...)
	}

	return dirs
}
