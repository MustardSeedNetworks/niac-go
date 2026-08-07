package scenario_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

const (
	steadyFloor       = 50
	peakFloor         = 70
	wantSteadyPercent = 50

	// Link-Live raises an interface Warning above 80% utilization. Measured
	// against a live discovery: interfaces up to 78.7% stayed clean, 81.8% and
	// above were flagged. Authored peaks must stay below that line so a demo
	// map reads healthy instead of amber.
	utilizationWarningFloor = 80
)

// A demo network has to look busy. Authored utilization sits mostly in the
// 50-70% band, peaks higher on a minority of interfaces, and leaves a few quiet
// — always below the Link-Live warning threshold.
func TestAuthoredUtilizationLooksBusy(t *testing.T) {
	for _, pack := range scenario.Packs() {
		steady, total := packUtilizationBands(t, pack)
		if total == 0 {
			t.Fatalf("%s authored no interface utilization", pack.ID)
		}
		if got := steady * 100 / total; got < wantSteadyPercent {
			t.Errorf("%s steady-band utilization = %d%%, want at least %d%%",
				pack.ID, got, wantSteadyPercent)
		}
	}
}

func packUtilizationBands(t *testing.T, pack scenario.Pack) (int, int) {
	t.Helper()
	var steady, total int
	result, err := scenario.Generate(pack.Request)
	if err != nil {
		t.Fatalf("generate %s: %v", pack.ID, err)
	}
	cfg, err := config.LoadYAMLBytes(result.YAML)
	if err != nil {
		t.Fatalf("load %s: %v", pack.ID, err)
	}
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		for _, iface := range device.Interfaces {
			for _, value := range []float64{iface.InUtilization, iface.OutUtilization} {
				total++
				if value >= utilizationWarningFloor {
					t.Errorf("%s %s %s utilization %.0f%% would trip the Link-Live warning at %d%%",
						pack.ID, device.Name, iface.Name, value, utilizationWarningFloor)
				}
				if value >= steadyFloor && value < peakFloor {
					steady++
				}
			}
		}
	}
	return steady, total
}
