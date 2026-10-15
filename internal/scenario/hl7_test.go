package scenario_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// The hospital's EMR, laboratory and pharmacy systems answer HL7 (AI-5), and
// nothing else in any pack opens the port.
func testPacksServeHL7OnClinicalSystems(t *testing.T, fixture *packFixture) {
	t.Parallel()

	clinical := []string{"EMR01", "LIS01", "RX01"}
	for _, pack := range scenario.Packs() {
		cfg := fixture.configByID(t, pack.ID)
		serving := map[string]bool{}
		for index := range cfg.Devices {
			device := &cfg.Devices[index]
			_, system, _ := strings.Cut(device.Name, "-")
			want := pack.ID == "hospital" && slices.Contains(clinical, system)
			got := device.HL7Config != nil && device.HL7Config.Enabled
			if got != want {
				t.Errorf("%s: %s serves HL7 = %v, want %v", pack.ID, device.Name, got, want)
			}
			if got {
				serving[system] = true
			}
		}
		if pack.ID == "hospital" && len(serving) != len(clinical) {
			t.Errorf("hospital: HL7 on %v, want every one of %v", serving, clinical)
		}
	}
}
