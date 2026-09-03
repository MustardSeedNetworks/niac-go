package scenario_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// TestEveryPackGeneratesWithoutYAMLRepair runs the kill criterion the authoring
// plan set for itself:
//
//	"if three representative scenarios still require YAML repair after the
//	 composer ships, stop adding scenario types and repair the authoring model."
//
// The composer shipped in #1163/#1165 and the criterion had never been executed
// — a plan that names its own stopping condition and then never checks it is
// just an intention. This runs it against every built-in pack rather than three,
// because the packs ARE the representative scenarios and there is no reason to
// sample when the whole set is cheap.
//
// "Requires YAML repair" is read strictly: the generated config must load and
// validate with no errors AND no warnings. A warning is something an operator
// would be expected to go and fix by hand, which is exactly what the composer
// exists to make unnecessary.
func TestEveryPackGeneratesWithoutYAMLRepair(t *testing.T) {
	packs := scenario.Packs()
	if len(packs) < 3 {
		t.Fatalf("the criterion is about three representative scenarios; only %d packs exist", len(packs))
	}

	for _, pack := range packs {
		t.Run(pack.ID, func(t *testing.T) {
			generated, err := scenario.Generate(pack.Request)
			if err != nil {
				t.Fatalf("composer could not generate %q: %v", pack.ID, err)
			}

			cfg, err := config.LoadYAMLBytes(generated.YAML)
			if err != nil {
				t.Fatalf("generated YAML for %q does not load, so it would need hand repair: %v",
					pack.ID, err)
			}

			result := config.NewValidator(pack.ID + ".yaml").Validate(cfg)
			if !result.Valid || result.HasWarnings() {
				t.Fatalf("%q is not strict-clean, so an operator would have to edit YAML:\n%s",
					pack.ID, result.Format())
			}

			// The manifest is the pack's frozen parity contract. A pack that
			// validates but no longer matches what it promises has drifted from
			// its authored truth, which the plan treats as the same failure.
			if got := generated.Manifest.Parity(); got != pack.Manifest {
				t.Errorf("%q drifted from its manifest:\n got  %#v\n want %#v", pack.ID, got, pack.Manifest)
			}
		})
	}
}
