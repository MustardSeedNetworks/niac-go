package scenario_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// Every pack phone answers SIP OPTIONS as the handset it models (AI-1), and
// nothing else in a pack does: a scanner that files a camera or a switch as a
// SIP endpoint has been told something false.
func TestPacksServeSIPOnThePhones(t *testing.T) {
	t.Parallel()

	for _, pack := range scenario.Packs() {
		cfg := generatePack(t, pack.ID)
		if result := config.NewValidator(pack.ID).Validate(cfg); !result.Valid {
			t.Errorf("%s: %#v", pack.ID, result.Errors)
		}
		phones := 0
		for index := range cfg.Devices {
			device := &cfg.Devices[index]
			isPhone := device.Properties["role"] == "voip-phone"
			serving := device.SIPConfig != nil && device.SIPConfig.Enabled
			switch {
			case isPhone && !serving:
				t.Errorf("%s: phone %s serves no SIP", pack.ID, device.Name)
			case !isPhone && serving:
				t.Errorf("%s: %s is no phone and serves SIP", pack.ID, device.Name)
			case isPhone && device.SIPConfig.UserAgent != "Cisco-CP8841/12.8.1":
				t.Errorf("%s: %s User-Agent = %q, want the CP-8841 on load 12.8.1",
					pack.ID, device.Name, device.SIPConfig.UserAgent)
			}
			if isPhone {
				phones++
			}
		}
		if phones == 0 {
			t.Errorf("%s: no phones", pack.ID)
		}
	}
}
