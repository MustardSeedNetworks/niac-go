package scenario

import "github.com/MustardSeedNetworks/niac-go/internal/converter"

// imagingModalityRole is the catalog role of the packs' imaging modality, the
// MRI that sends its studies to the hospital's PACS archive.
const imagingModalityRole = "mr-system"

// packDICOM opens DICOM on an imaging modality. Its peer, the PACS archive,
// gets it from the hospital's workload service, so both ends of the
// conversation answer C-ECHO.
func packDICOM(role string) *converter.DICOMConfig {
	if role != imagingModalityRole {
		return nil
	}

	return &converter.DICOMConfig{Enabled: true}
}
