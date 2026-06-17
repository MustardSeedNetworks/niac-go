// Package templates holds the on-disk configuration-template discovery,
// parsing, and loading engine extracted from the internal/api monolith
// (ADR-0006). It walks the filesystem template directories, parses YAML
// front-matter into structured metadata, and materialises a chosen
// template into a saved config — all with no dependency on the api
// transport layer, which composes it inward.
//
// This is distinct from the sibling internal/templates package, which is
// the embedded builtin template library compiled into the binary. This
// leaf is the runtime on-disk engine (development trees, system install
// paths, and the NIAC_TEMPLATES_DIR override) used by the
// /api/v1/templates HTTP surface and the daemon's StartSimulation path.
package templates

import "os"

// Template type constants matching the UI interface.
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

// Template represents a configuration template.
//
// Name is the stable filename-derived identifier ("minimal", "router")
// used by the /api/v1/templates/{name} and /api/v1/templates/use APIs.
// DisplayName is the human-readable label optionally provided via the
// front-matter "# Display: ..." line; clients should prefer it for UI
// rendering and fall back to Name when absent.
type Template struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description"`
	DeviceCount int    `json:"deviceCount"`
	Type        string `json:"type"`
	// Vendor is populated from the template's "# Vendor: ..."
	// front-matter, lowercased. When non-empty the UI groups the
	// template under a vendor heading instead of (or alongside) the
	// generic type-based grouping — useful for the vendor template
	// pack shipped under cmd/niac/templates/vendor-templates/.
	Vendor string   `json:"vendor,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

// Content represents the full content of a template.
type Content struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Format  string `json:"format"`
}

// Dirs returns the directories to scan for templates. It checks multiple
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
