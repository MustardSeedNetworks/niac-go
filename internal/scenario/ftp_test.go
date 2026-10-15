package scenario_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// The HP printers serve FTP printing with the greeting HP documents for
// Jetdirect (D-NIAC-45), and nothing else in any pack opens the port: no
// other pack printer or server has a documented greeting to send.
func TestPacksServeFTPOnTheHPPrinters(t *testing.T) {
	t.Parallel()

	servers := 0
	for _, pack := range scenario.Packs() {
		cfg := generatePack(t, pack.ID)
		for index := range cfg.Devices {
			device := &cfg.Devices[index]
			hp := device.Type == "printer" && device.Properties["model"] == hpPrinterModel
			ftp := device.FTPConfig
			serving := ftp != nil && ftp.Enabled
			if serving != hp {
				t.Errorf("%s: %s (%s) serves FTP = %v", pack.ID, device.Name, device.Properties["model"], serving)
				continue
			}
			if !serving {
				continue
			}
			servers++
			if ftp.WelcomeBanner != "220 JD FTP Server Ready" {
				t.Errorf("%s: %s greets with %q, want HP Jetdirect's", pack.ID, device.Name, ftp.WelcomeBanner)
			}
		}
	}
	if servers == 0 {
		t.Error("no pack has an FTP printer")
	}
}
