package library_test

import (
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

// TestStarterPackTemplatesAreLoadable guards D2.
//
// The starter pack is what bootstrapStarterPack copies into the on-disk library
// on first run, so it is what the New Simulation wizard actually offers. All
// eight shipped templates failed to load: seven were rejected at strict decode
// (obsolete DhcpServer.enabled/pools, DNSServer.enabled/records, and in one case
// TrapsConfig.targets, Device.traffic, FtpConfig.username/password,
// NetbiosConfig.workstation_name), and basic-network decoded but failed semantic
// validation with "SNMPv1/v2c requires an explicit community" twice.
//
// Two other template trees — internal/templates/builtin and cmd/niac/templates —
// were already covered by full-validator tests. This one was not, which is
// exactly why the only tree users see was the broken one. It runs the real
// loader and the real validator, the same path a start goes through.
func TestStarterPackTemplatesAreLoadable(t *testing.T) {
	names := templates.ListNames()
	if len(names) == 0 {
		t.Fatal("no starter templates found to validate")
	}

	// Loaded from a bootstrapped library rather than from the embedded bytes,
	// because that is where they are loaded from in the product: a template
	// that names a captured walk resolves it relative to its own file, and the
	// walks sit beside networks/ in the same library. Reading the bytes alone
	// could not see that path at all, so a template with a walk_file looked
	// broken here and worked in the wizard, or the reverse.
	root := t.TempDir()
	if _, err := library.Open(root); err != nil {
		t.Fatalf("bootstrap library: %v", err)
	}
	networks := filepath.Join(root, string(library.KindNetworks))

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			assertTemplateLoads(t, name, filepath.Join(networks, name+".yaml"))
		})
	}
}

func assertTemplateLoads(t *testing.T, name, path string) {
	t.Helper()

	if _, getErr := templates.Get(name); getErr != nil {
		t.Fatalf("get: %v", getErr)
	}

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

// F8: before this, none of the shipped templates used snmp_agent.walk_file or
// snmp_agent.add_mibs, so the two features that turn a capture into a fleet had
// no worked example anywhere a user would look -- and no template exercised the
// include_path resolution the library layout requires. This keeps at least one
// of each, and because it reads the loaded config rather than the file text it
// also proves the walk resolved to a file that exists.
func TestStarterPackDemonstratesCapturedWalks(t *testing.T) {
	root := t.TempDir()
	if _, err := library.Open(root); err != nil {
		t.Fatalf("bootstrap library: %v", err)
	}
	networks := filepath.Join(root, string(library.KindNetworks))

	walkBacked, overridden := 0, 0
	for _, name := range templates.ListNames() {
		cfg, err := config.LoadYAML(filepath.Join(networks, name+".yaml"))
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
		t.Error("no starter template replays a captured walk; snmp_agent.walk_file has no worked example")
	}
	if overridden == 0 {
		t.Error("no starter template overrides a captured value; snmp_agent.add_mibs has no worked example")
	}
}
