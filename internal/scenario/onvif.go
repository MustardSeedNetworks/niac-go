package scenario

import (
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// packONVIF answers WS-Discovery on the cameras. Without it a video
// management system's discovery finds nothing and a scanner sees a host with
// an Axis MAC prefix; with it, the camera answers a Probe with its name and
// model.
func packONVIF(role string, profile DeviceProfile) *converter.ONVIFConfig {
	if role != "ip-camera" {
		return nil
	}

	return &converter.ONVIFConfig{
		Enabled:  true,
		Name:     strings.ToUpper(profile.Vendor) + " " + profile.Model,
		Hardware: profile.Model,
	}
}
