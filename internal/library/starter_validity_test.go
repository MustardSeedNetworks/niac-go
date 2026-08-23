package library_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
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

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			tmpl, getErr := templates.Get(name)
			if getErr != nil {
				t.Fatalf("get: %v", getErr)
			}

			cfg, loadErr := config.LoadYAMLBytes([]byte(tmpl.Content))
			if loadErr != nil {
				t.Fatalf("does not load — the wizard offers this and it cannot start: %v", loadErr)
			}
			if len(cfg.Devices) == 0 && len(cfg.Segments) == 0 {
				t.Fatal("no devices")
			}
			// Validate returns a non-nil *ListError even when clean, so check
			// the list rather than the pointer.
			if valErr := config.NewValidator(name).Validate(cfg); valErr != nil && len(valErr.Errors) > 0 {
				for _, e := range valErr.Errors {
					t.Errorf("fails validation: %s", e.Message)
				}
			}
		})
	}
}
