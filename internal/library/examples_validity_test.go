package library_test

import (
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

// TestTemplateExamplesAreLoadable guards D2 for the shipped templates.
//
// Examples are what the New Simulation wizard offers, so one that the real
// loader or validator rejects is a scenario nobody can start. Each template is
// duplicated into a library and loaded from there rather than from the embedded
// bytes, because that is where a copy is loaded in the product: a template that
// names a captured walk resolves it relative to its own file, and the walks sit
// beside networks/ in the same library.
func TestTemplateExamplesAreLoadable(t *testing.T) {
	names := templates.ListNames()
	if len(names) == 0 {
		t.Fatal("no templates found to validate")
	}

	lib := openBootstrappedLibrary(t)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			copyName, err := lib.DuplicateExample(name, "")
			if err != nil {
				t.Fatalf("duplicate: %v", err)
			}
			assertExampleLoads(t, name, filepath.Join(lib.SubDir(library.KindNetworks), copyName+".yaml"))
		})
	}
}

// openBootstrappedLibrary opens a library whose starter walks were unpacked,
// which the captured templates need: openTempLibrary suppresses that.
func openBootstrappedLibrary(t *testing.T) *library.Library {
	t.Helper()
	lib, err := library.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	return lib
}

func assertExampleLoads(t *testing.T, name, path string) {
	t.Helper()

	cfg, loadErr := config.LoadYAML(path)
	if loadErr != nil {
		t.Fatalf("does not load — the wizard offers this and it cannot start: %v", loadErr)
	}
	if len(cfg.Devices) == 0 && len(cfg.Segments) == 0 {
		t.Fatal("no devices")
	}
	// Validate returns a non-nil *ListError even when clean, so check the list
	// rather than the pointer.
	valErr := config.NewValidator(name).Validate(cfg)
	if valErr == nil {
		return
	}
	for _, failure := range valErr.Errors {
		t.Errorf("fails validation: %s", failure.Message)
	}
}

// TestPackExamplesGenerateTheirManifest holds the list to what the document
// will actually contain: the device count a pack row advertises comes from its
// frozen manifest, and the document is generated on demand. A pack whose
// generated document drifts from its manifest would be listed with the wrong
// size.
func TestPackExamplesGenerateTheirManifest(t *testing.T) {
	for _, pack := range scenario.Packs() {
		if testing.Short() && pack.ID != "campus" {
			continue
		}
		t.Run(pack.ID, func(t *testing.T) {
			data, err := library.ExampleContent(pack.ID)
			if err != nil {
				t.Fatalf("ExampleContent: %v", err)
			}
			cfg, err := config.LoadYAMLBytes(data)
			if err != nil {
				t.Fatalf("generated pack does not load: %v", err)
			}
			if got := cfg.DeviceCount(); got != pack.Manifest.DeviceCount {
				t.Errorf("devices = %d, manifest says %d", got, pack.Manifest.DeviceCount)
			}
		})
	}
}

// F8: at least one shipped template keeps a worked example of
// snmp_agent.walk_file and of snmp_agent.add_mibs, and no template exercises
// the include_path resolution the library layout requires unless one does.
// Because this reads the loaded config rather than the file text it also proves
// the walk resolved to a file that exists.
func TestTemplateExamplesDemonstrateCapturedWalks(t *testing.T) {
	lib := openBootstrappedLibrary(t)

	walkBacked, overridden := 0, 0
	for _, name := range templates.ListNames() {
		copyName, err := lib.DuplicateExample(name, "")
		if err != nil {
			t.Fatalf("duplicate %s: %v", name, err)
		}
		cfg, err := config.LoadYAML(filepath.Join(lib.SubDir(library.KindNetworks), copyName+".yaml"))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, device := range cfg.Devices {
			if device.SNMPConfig.WalkFile != "" {
				walkBacked++
			}
			overridden += len(device.SNMPConfig.AddMibs)
		}
	}

	if walkBacked == 0 {
		t.Error("no template replays a captured walk; snmp_agent.walk_file has no worked example")
	}
	if overridden == 0 {
		t.Error("no template overrides a captured value; snmp_agent.add_mibs has no worked example")
	}
}
