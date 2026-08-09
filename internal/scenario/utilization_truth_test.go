package scenario_test

import (
	"strings"
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
	// above were flagged. The generated band must stay below that line so a demo
	// map reads healthy instead of amber; the only interfaces above it are the
	// ones a pack deliberately calls out as its story.
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
	authored := make(map[string]bool, len(pack.Request.Congestion))
	for _, link := range pack.Request.Congestion {
		authored[link.Device+"|"+link.Interface] = true
	}
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		for _, iface := range device.Interfaces {
			story := authored[device.Name+"|"+iface.Name]
			for _, value := range []float64{iface.InUtilization, iface.OutUtilization} {
				total++
				if value >= utilizationWarningFloor && !story {
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

// Every device must announce its own name by some means, or it renders as a
// bare IP address on a Link-Live map. A discovery tool reads sysName first and
// only falls back to a reverse lookup, which it resolves through its own
// resolver rather than the simulated one — so correct PTR records are not
// enough on their own.
//
// Which means depends on what the device is. Infrastructure and appliances
// answer SNMP; personal computers deliberately do not, so that a discovery tool
// files them as hosts rather than as managed infrastructure, and they announce
// over NetBIOS or multicast DNS instead.
func TestEveryDeviceAnnouncesItsName(t *testing.T) {
	for _, pack := range scenario.Packs() {
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
			switch {
			case device.SNMPConfig.SysName != "":
				if got := device.SNMPConfig.SysName; got != device.Name {
					t.Errorf("%s %s sysName = %q, want %q", pack.ID, device.Name, got, device.Name)
				}
			case device.NetBIOSConfig != nil && device.NetBIOSConfig.Enabled:
				assertAnnouncedName(t, pack.ID, device.Name, device.NetBIOSConfig.Name, netbiosNameLimit)
			case device.MDNSConfig != nil && device.MDNSConfig.Enabled:
				assertAnnouncedName(t, pack.ID, device.Name, device.MDNSConfig.Hostname, 0)
			default:
				t.Errorf("%s %s announces no name at all — it renders as a bare IP", pack.ID, device.Name)
			}
		}
	}
}

// netbiosNameLimit is the wire cap on a NetBIOS name. A longer name reaches a
// discovery tool truncated and disagrees with authored truth.
const netbiosNameLimit = 15

func assertAnnouncedName(t *testing.T, pack, deviceName, announced string, limit int) {
	t.Helper()
	if !strings.EqualFold(announced, deviceName) {
		t.Errorf("%s %s announces %q, want its own name", pack, deviceName, announced)
	}
	if limit > 0 && len(announced) > limit {
		t.Errorf("%s %s announces %d characters, truncated above %d",
			pack, deviceName, len(announced), limit)
	}
}
