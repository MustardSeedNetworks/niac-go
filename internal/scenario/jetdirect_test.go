package scenario_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// The office printer's model and the PJL ID it answers with.
const (
	hpPrinterModel = "LaserJet Enterprise MFP M634"
	hpPJLID        = "HP LaserJet MFP M634"
)

// Every pack printer accepts raw jobs on 9100 (AI-3), the HP answers PJL as
// the model it is, and nothing else in a pack opens the port: a scanner that
// files a camera or a workstation as a printer has been told something false.
func TestPacksServeJetDirectOnThePrinters(t *testing.T) {
	t.Parallel()

	pjlPrinters := 0
	for _, pack := range scenario.Packs() {
		cfg := generatePack(t, pack.ID)
		if result := config.NewValidator(pack.ID).Validate(cfg); !result.Valid {
			t.Errorf("%s: %#v", pack.ID, result.Errors)
		}
		printers := 0
		for index := range cfg.Devices {
			device := &cfg.Devices[index]
			checkPackJetDirect(t, pack.ID, device)
			if device.Type == "printer" {
				printers++
			}
			if device.JetDirectConfig != nil && device.JetDirectConfig.PJLID != "" {
				pjlPrinters++
			}
		}
		if printers == 0 {
			t.Errorf("%s: no printers", pack.ID)
		}
	}
	if pjlPrinters == 0 {
		t.Error("no pack has a PJL printer")
	}
}

func checkPackJetDirect(t *testing.T, packID string, device *config.Device) {
	t.Helper()
	isPrinter := device.Type == "printer"
	jetDirect := device.JetDirectConfig
	serving := jetDirect != nil && jetDirect.Enabled
	switch {
	case isPrinter && !serving:
		t.Errorf("%s: printer %s serves no JetDirect", packID, device.Name)
	case !isPrinter && serving:
		t.Errorf("%s: %s is no printer and serves JetDirect", packID, device.Name)
	case !isPrinter:
	case device.Properties["model"] == hpPrinterModel && jetDirect.PJLID != hpPJLID:
		t.Errorf("%s: %s PJL ID = %q, want %s", packID, device.Name, jetDirect.PJLID, hpPJLID)
	case device.Properties["model"] != hpPrinterModel && jetDirect.PJLID != "":
		t.Errorf("%s: %s (%s) speaks no PJL but answers as %q",
			packID, device.Name, device.Properties["model"], jetDirect.PJLID)
	}
}
