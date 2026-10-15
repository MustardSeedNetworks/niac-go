package scenario_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// The hospital's MRIs and its PACS archive answer DICOM (AI-4), so both ends
// of the modality-to-archive conversation verify, and nothing else in any
// pack opens the ports.
func TestPacksServeDICOMOnModalitiesAndPACS(t *testing.T) {
	t.Parallel()

	for _, pack := range scenario.Packs() {
		cfg := generatePack(t, pack.ID)
		modalities, archives := 0, 0
		for index := range cfg.Devices {
			device := &cfg.Devices[index]
			modality := device.Properties["role"] == "mr-system"
			archive := strings.HasSuffix(device.Name, "-PACS01")
			serving := device.DICOMConfig != nil && device.DICOMConfig.Enabled
			if serving != (modality || archive) {
				t.Errorf("%s: %s (role %q) serves DICOM = %v", pack.ID, device.Name, device.Properties["role"], serving)
			}
			if serving && modality {
				modalities++
			}
			if serving && archive {
				archives++
			}
		}
		if pack.ID == "hospital" && (modalities == 0 || archives == 0) {
			t.Errorf("hospital: %d modalities and %d archives answer DICOM, want both", modalities, archives)
		}
	}
}
