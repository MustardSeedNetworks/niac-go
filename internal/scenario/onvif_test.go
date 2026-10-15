package scenario_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// Every pack camera answers WS-Discovery as the camera it models (AI-2), and
// nothing else in a pack does: a video management system that adopts a phone
// or a switch as a camera has been told something false.
func testPacksServeONVIFOnTheCameras(t *testing.T, fixture *packFixture) {
	t.Parallel()

	for _, pack := range scenario.Packs() {
		cfg := fixture.configByID(t, pack.ID)
		if result := config.NewValidator(pack.ID).Validate(cfg); !result.Valid {
			t.Errorf("%s: %#v", pack.ID, result.Errors)
		}
		cameras := 0
		for index := range cfg.Devices {
			device := &cfg.Devices[index]
			isCamera := device.Properties["role"] == "ip-camera"
			serving := device.ONVIFConfig != nil && device.ONVIFConfig.Enabled
			switch {
			case isCamera && !serving:
				t.Errorf("%s: camera %s serves no ONVIF discovery", pack.ID, device.Name)
			case !isCamera && serving:
				t.Errorf("%s: %s is no camera and serves ONVIF discovery", pack.ID, device.Name)
			case isCamera && (device.ONVIFConfig.Name != "AXIS P3265-LVE" || device.ONVIFConfig.Hardware != "P3265-LVE"):
				t.Errorf("%s: %s ONVIF identity = %+v, want the AXIS P3265-LVE",
					pack.ID, device.Name, device.ONVIFConfig)
			}
			if isCamera {
				cameras++
			}
		}
		if cameras == 0 {
			t.Errorf("%s: no cameras", pack.ID)
		}
	}
}
