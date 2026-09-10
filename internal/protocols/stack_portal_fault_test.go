package protocols

import (
	"errors"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestCaptivePortalRequiresHTTPService(t *testing.T) {
	for _, tc := range []struct {
		name string
		http *config.HTTPConfig
		want bool
	}{
		{"absent", nil, false},
		{"disabled", &config.HTTPConfig{Enabled: false}, false},
		{"enabled", &config.HTTPConfig{Enabled: true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			device := faultTestDevice("portal-host")
			device.HTTPConfig = tc.http
			stack := NewStack(nil, &config.Config{Devices: []config.Device{device}}, logging.NewDebugConfig(0))
			targets := stack.DeviceFaultTargets()
			if len(targets) != 1 || slices.Contains(targets[0].Faults, devicestate.FaultCaptivePortal) != tc.want {
				t.Fatalf("portal targets=%v, want available=%v", targets, tc.want)
			}
			err := stack.SetDeviceFault(device.Name, devicestate.FaultCaptivePortal, 1)
			if tc.want && err != nil {
				t.Fatal(err)
			}
			if !tc.want && (!errors.Is(err, ErrFaultServiceAbsent) || len(stack.ActiveDeviceFaults()) != 0) {
				t.Fatalf("unserved portal stored: %v, %v", err, stack.ActiveDeviceFaults())
			}
		})
	}
}
