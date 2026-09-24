package api_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/MustardSeedNetworks/niac-go/"

// apiDomainImports is the facade ratchet (niac-go#2256): for every non-test
// file under internal/api, the niac-go internal packages it imports outside
// internal/api itself. internal/daemon is meant to be the one service facade,
// so this list only shrinks. A new import means a new entry, reviewed in the
// PR that adds it; dropping an import means dropping its entry.
func apiDomainImports() map[string][]string {
	return map[string][]string{
		"capture/analyze.go":          {"internal/packetdecode"},
		"capture/capture.go":          {"internal/packetdecode"},
		"config_diagnostics.go":       {"internal/config", "internal/converter", "internal/fabric"},
		"config_ops.go":               {"internal/config"},
		"config_state.go":             {"internal/config", "internal/protocols", "internal/topology"},
		"configs_migrate.go":          {"internal/library"},
		"debug_level.go":              {"internal/protocols"},
		"decode.go":                   {"internal/content"},
		"devices.go":                  {"internal/config"},
		"devices_helpers.go":          {"internal/config", "internal/converter"},
		"devices_protocol_apply.go":   {"internal/config"},
		"devices_responses.go":        {"internal/config"},
		"devices_validation.go":       {"internal/safeconv"},
		"draft_walk_rebase.go":        {"internal/library"},
		"handlers_behaviors.go":       {"internal/behavior"},
		"handlers_device_actions.go":  {"internal/devicestate", "internal/protocols"},
		"handlers_draft_behaviors.go": {"internal/config", "internal/devicestate", "internal/library"},
		"handlers_draft_topology.go":  {"internal/config", "internal/drafttopology", "internal/library"},
		"handlers_draft_topology_profile.go": {
			"internal/config",
			"internal/drafttopology",
			"internal/library",
			"internal/scenario",
		},
		"handlers_drafts.go":  {"internal/config", "internal/library"},
		"handlers_errors.go":  {"internal/devicestate", "internal/protocols"},
		"handlers_library.go": {"internal/library", "internal/sanitize", "internal/walkmeta"},
		"handlers_metrics.go": {"internal/protocols"},
		"handlers_read.go": {
			"internal/capture",
			"internal/config",
			"internal/protocols",
			"internal/storage",
		},
		"handlers_scenario.go":            {"internal/scenario"},
		"handlers_segments.go":            {"internal/config"},
		"handlers_session_capture.go":     {"internal/capture", "internal/capturering"},
		"handlers_session_checkpoints.go": {"internal/devicestate", "internal/protocols"},
		"handlers_session_read.go":        {"internal/protocols"},
		"handlers_simulation.go":          {"internal/config", "internal/fabric"},
		"handlers_synthesize_walk.go":     {"internal/config", "internal/library", "internal/protocols/snmp/synth"},
		"handlers_walk_profile.go": {
			"internal/library",
			"internal/protocols/snmp",
			"internal/sanitize",
			"internal/walkcapture",
			"internal/walkprofile",
		},
		"handlers_walk_profile_create.go": {
			"internal/library",
			"internal/protocols/snmp",
			"internal/scenario",
			"internal/walkprofile",
		},
		"handlers_walk_profile_resume.go": {"internal/library", "internal/protocols/snmp", "internal/walkprofile"},
		"interface_fault_types.go":        {"internal/devicestate", "internal/protocols"},
		"library_install.go":              {"internal/content", "internal/library"},
		"server.go": {
			"internal/config",
			"internal/content",
			"internal/fabric",
			"internal/library",
			"internal/protocols",
			"internal/storage",
			"internal/topology",
		},
		"session_packet_observers.go": {"internal/capturering", "internal/protocols"},
		"session_state.go": {
			"internal/capturering",
			"internal/config",
			"internal/protocols",
			"internal/topology",
		},
		"simulation_state.go": {
			"internal/capturering",
			"internal/config",
			"internal/protocols",
			"internal/topology",
		},
		"sse/packet_observer.go": {"internal/packetdecode", "internal/protocols"},
		"stats_stream.go":        {"internal/protocols"},
		"templates.go":           {"internal/config"},
		"validation.go": {
			"internal/capture",
			"internal/config",
			"internal/fabric",
			"internal/safeconv",
		},
		"walk.go":         {"internal/config", "internal/library", "internal/protocols/snmp"},
		"walk_analyze.go": {"internal/walkanalysis"},
	}
}

// TestAPIDomainImportRatchet fails in both directions: an import missing from
// apiDomainImports, and an entry the file no longer imports. go/parser ignores
// build constraints, so platform-tagged files are checked on every host.
func TestAPIDomainImportRatchet(t *testing.T) {
	want := apiDomainImports()
	got := apiInternalImports(t)

	for _, e := range absent(got, want) {
		t.Errorf(
			"%s imports %s, which apiDomainImports does not list: route the call through internal/daemon, or add %q to the %q entry",
			e.file,
			e.pkg,
			e.pkg,
			e.file,
		)
	}
	for _, e := range absent(want, got) {
		t.Errorf(
			"%s no longer imports %s: remove %q from the %q entry in apiDomainImports",
			e.file,
			e.pkg,
			e.pkg,
			e.file,
		)
	}
}

type fileImport struct{ file, pkg string }

// absent returns the imports listed in from that in does not list.
func absent(from, in map[string][]string) []fileImport {
	var out []fileImport
	for file, pkgs := range from {
		for _, pkg := range pkgs {
			if !slices.Contains(in[file], pkg) {
				out = append(out, fileImport{file, pkg})
			}
		}
	}
	return out
}

func apiInternalImports(t *testing.T) map[string][]string {
	t.Helper()
	imports := map[string][]string{}
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "testdata" {
			return filepath.SkipDir
		}
		if d.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		pkgs, err := internalImports(path)
		if len(pkgs) > 0 {
			imports[filepath.ToSlash(path)] = pkgs
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return imports
}

// internalImports returns the niac-go internal packages outside internal/api
// that the file at path imports.
func internalImports(path string) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var pkgs []string
	for _, spec := range f.Imports {
		pkg, uerr := strconv.Unquote(spec.Path.Value)
		if uerr != nil {
			return nil, uerr
		}
		rel, ok := strings.CutPrefix(pkg, modulePath)
		if ok && strings.HasPrefix(rel, "internal/") && !strings.HasPrefix(rel, "internal/api/") {
			pkgs = append(pkgs, rel)
		}
	}
	return pkgs, nil
}
