package scenario_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// A plant runs its cells off a fibre ring rather than dual-homing every closet
// to a distribution pair, and Link-Live draws that ring as a ring: a hand-closed
// six-node ring was discovered intact on analysis 6a774b2f9dc61ad4327b182a,
// closing edge included. Manufacturing is the pack that reads as a plant, so it
// is the one that gets the shape.
func TestManufacturingAccessLayerIsARing(t *testing.T) {
	cfg := generatePack(t, "manufacturing")
	access := []string{
		"PLT-ACC-SW01", "PLT-ACC-SW02", "PLT-ACC-SW03",
		"PLT-ACC-SW04", "PLT-ACC-SW05", "PLT-ACC-SW06",
	}

	for index, name := range access {
		next := access[(index+1)%len(access)]
		if !adjacent(cfg, name, next) {
			t.Errorf("%s is not linked to its ring neighbour %s", name, next)
		}
	}
}

// The ring is only a ring if it hangs off the distribution tier at a couple of
// points. Dual-homing every node as well would leave the same star underneath.
func TestManufacturingRingJoinsDistributionTwice(t *testing.T) {
	cfg := generatePack(t, "manufacturing")

	if uplinks := countUplinks(cfg, "PLT-ACC-SW", "PLT-DIST-SW"); uplinks != 2 {
		t.Errorf("access-to-distribution uplinks = %d, want 2", uplinks)
	}
}

// Hospital's shape is the one already validated against Link-Live, so every
// access switch there still dual-homes into the distribution pair.
func TestHospitalAccessLayerStaysDualHomed(t *testing.T) {
	cfg := generatePack(t, "hospital")

	if uplinks := countUplinks(cfg, "MED-ACC-SW", "MED-DIST-SW"); uplinks != 12 {
		t.Errorf("access-to-distribution uplinks = %d, want 12", uplinks)
	}
	if adjacent(cfg, "MED-ACC-SW01", "MED-ACC-SW02") {
		t.Error("hospital access switches are linked to each other")
	}
}

func generatePack(t *testing.T, id string) *config.Config {
	t.Helper()
	for _, pack := range scenario.Packs() {
		if pack.ID == id {
			return packDevices(t, pack)
		}
	}
	t.Fatalf("no pack %q", id)

	return nil
}

func adjacent(cfg *config.Config, name, peer string) bool {
	device := findDevice(cfg, name)
	if device == nil {
		return false
	}
	for _, port := range device.TrunkPorts {
		if port.RemoteDevice == peer {
			return true
		}
	}

	return false
}

func countUplinks(cfg *config.Config, accessPrefix, distributionPrefix string) int {
	total := 0
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		if !strings.HasPrefix(device.Name, accessPrefix) {
			continue
		}
		for _, port := range device.TrunkPorts {
			if strings.HasPrefix(port.RemoteDevice, distributionPrefix) {
				total++
			}
		}
	}

	return total
}

// A shape the generator cannot build has to be refused, not silently turned
// into the default one — that is indistinguishable from a typo in the request.
func TestUnknownAccessLayerIsRejected(t *testing.T) {
	request := packRequest(t, "manufacturing")
	request.AccessLayer = "mesh"

	if _, err := scenario.Generate(request); err == nil {
		t.Error("an unknown access layer generated a fleet")
	}
}

func TestRingNeedsEnoughNodes(t *testing.T) {
	request := packRequest(t, "manufacturing")
	request.Counts.AccessSwitches = 2

	if _, err := scenario.Generate(request); err == nil {
		t.Error("a two-node ring generated a fleet")
	}
}

func packRequest(t *testing.T, id string) scenario.Request {
	t.Helper()
	for _, pack := range scenario.Packs() {
		if pack.ID == id {
			return pack.Request
		}
	}
	t.Fatalf("no pack %q", id)

	return scenario.Request{}
}

// A campus is wide and shallow: closets land straight on a collapsed core, with
// no distribution tier between them. That is what makes its map read as a campus
// rather than a smaller copy of the hospital.
func TestCampusCollapsesTheCore(t *testing.T) {
	cfg := generatePack(t, "campus")

	if findDevice(cfg, "NTH-DIST-SW01") != nil {
		t.Error("campus still generates a distribution tier")
	}
	if uplinks := countUplinks(cfg, "NTH-ACC-SW", "NTH-CORE-SW"); uplinks != 8 {
		t.Errorf("access-to-core uplinks = %d, want 8", uplinks)
	}
}

// A store runs its lanes off one another rather than home-running each till to
// the back office, so the access tier is a chain with a single uplink.
func TestRetailStoreChainsItsLanes(t *testing.T) {
	cfg := generatePack(t, "retail")

	for index := 1; index < 4; index++ {
		name := numbered("STR-ACC-SW", index)
		next := numbered("STR-ACC-SW", index+1)
		if !adjacent(cfg, name, next) {
			t.Errorf("%s is not chained to %s", name, next)
		}
	}
	if adjacent(cfg, "STR-ACC-SW04", "STR-ACC-SW01") {
		t.Error("the lane chain closes into a ring")
	}
	if uplinks := countUplinks(cfg, "STR-ACC-SW", "STR-DIST-SW"); uplinks != 2 {
		t.Errorf("chain uplinks = %d, want 2", uplinks)
	}
}

// A metro POP hands its access nodes off a ring, and every pack keeps the full
// spine including radios and a controller pair - service provider was the one
// pack generating neither.
func TestServiceProviderRingsItsPOPAndKeepsTheSpine(t *testing.T) {
	cfg := generatePack(t, "service-provider")

	for index := 1; index <= 4; index++ {
		next := index%4 + 1
		if !adjacent(cfg, numbered("NYC-ACC-SW", index), numbered("NYC-ACC-SW", next)) {
			t.Errorf("NYC-ACC-SW%02d is not linked to its ring neighbour", index)
		}
	}
	if findDevice(cfg, "NYC-WLC01") == nil || findDevice(cfg, "NYC-WAP-B01-F01-01") == nil {
		t.Error("service provider still generates no radios and no controller")
	}
}

func numbered(prefix string, index int) string {
	return fmt.Sprintf("%s%02d", prefix, index)
}
