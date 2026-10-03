package protocols_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// The tables SEED's collectors walk (seed internal/polling/snmp/collectors).
const (
	ifTablePrefix         = "1.3.6.1.2.1.2.2.1"
	ifXTablePrefix        = "1.3.6.1.2.1.31.1.1.1"
	ipCidrRouteTable      = "1.3.6.1.2.1.4.24.4.1"
	ipRouteTable          = "1.3.6.1.2.1.4.21.1"
	dot1qTpFdbTable       = "1.3.6.1.2.1.17.7.1.2.2.1"
	dot1qTpFdbPortColumn  = "2"
	lldpRemTable          = "1.0.8802.1.1.2.1.4.1.1"
	cdpCacheTable         = "1.3.6.1.4.1.9.9.23.1.2.1.1"
	foundryFDPCacheTable  = "1.3.6.1.4.1.1991.1.1.3.2.2.1"
	tableIndexAfterColumn = 2
)

// TestShippedPacks runs every check that reads the served MIB of a shipped
// pack over one generation of each: generating and loading the seven packs is
// most of what each check costs, and several times that under the race
// detector.
func TestShippedPacks(t *testing.T) {
	for _, pack := range scenario.Packs() {
		t.Run(pack.ID, func(t *testing.T) {
			shipped := loadShippedPack(t, pack)
			t.Run("sites elect their primary core", func(t *testing.T) {
				checkPackSitesElectTheirPrimaryCore(t, shipped)
			})
			t.Run("manifest observations match the served tables", func(t *testing.T) {
				checkManifestObservationsMatchTheServedTables(t, shipped)
			})
		})
	}
}

// shippedPack is one generated pack and the stack serving it.
type shippedPack struct {
	cfg      *config.Config
	manifest scenario.Manifest
	stack    *protocols.Stack
}

// loadShippedPack generates a pack and binds its stack to the compiled fabric,
// as the daemon runs one: an unbound stack serves routes the daemon never would.
func loadShippedPack(t *testing.T, pack scenario.Pack) shippedPack {
	t.Helper()
	result, err := scenario.Generate(pack.Request)
	if err != nil {
		t.Fatal(err)
	}
	report := fabric.CompileConfig(result.Config)
	if !report.Safe {
		t.Fatalf("compile fabric: %v", report.Diagnostics)
	}
	stack := protocols.NewStack(nil, result.Config, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&report.Topology)

	return shippedPack{cfg: result.Config, manifest: result.Manifest, stack: stack}
}

// A pack's expectedObservations is a promise to SEED's SNMP acceptance, which
// polls every agent of the running pack and fails on any count that differs.
// The manifest is built from the scenario's config, so this derives each count
// the other way -- from the tables the agents actually serve -- the way SEED's
// collectors read them (niac-go#2353).
func checkManifestObservationsMatchTheServedTables(t *testing.T, pack shippedPack) {
	t.Helper()
	served := servedObservations(pack.stack, pack.cfg)
	promised := pack.manifest.Observations
	for _, collector := range slices.Sorted(maps.Keys(served)) {
		if got, want := served[collector], promised[collector]; got != want {
			t.Errorf("%s: served %+v, manifest promises %+v", collector, got, want)
		}
	}
	for _, collector := range slices.Sorted(maps.Keys(promised)) {
		if _, ok := served[collector]; !ok {
			t.Errorf("%s: manifest promises %+v, no agent serves a row", collector, promised[collector])
		}
	}
}

// servedObservations tallies what each collector finds across every agent: a
// device counts when the collector finds at least one row there. Collectors
// that report only a device count (sys_info and the neighbour tables) carry no
// rows, matching the manifest's shape.
func servedObservations(stack *protocols.Stack, cfg *config.Config) map[string]scenario.Observation {
	observations := map[string]scenario.Observation{}
	tally := func(collector string, rows int, withRows bool) {
		if rows == 0 {
			return
		}
		observation := observations[collector]
		observation.Devices++
		if withRows {
			observation.Rows += rows
		}
		observations[collector] = observation
	}

	for i := range cfg.Devices {
		device := &cfg.Devices[i]
		walk := func(prefix string) []string {
			oids, _ := stack.SNMPWalk(device, prefix)
			return oids
		}
		if _, ok := stack.SNMPWalk(device, ifTablePrefix); !ok {
			continue
		}

		tally(scenario.CollectorSysInfo, 1, false)
		interfaces := tableRows(ifTablePrefix, walk(ifTablePrefix))
		maps.Copy(interfaces, tableRows(ifXTablePrefix, walk(ifXTablePrefix)))
		tally(scenario.CollectorIfTable, len(interfaces), true)
		routes := tableRows(ipCidrRouteTable, walk(ipCidrRouteTable))
		if len(routes) == 0 {
			routes = tableRows(ipRouteTable, walk(ipRouteTable))
		}
		tally(scenario.CollectorRouting, len(routes), true)
		tally(scenario.CollectorFDB, fdbPorts(stack, device), true)
		tally(scenario.CollectorLLDP, len(tableRows(lldpRemTable, walk(lldpRemTable))), false)
		tally(scenario.CollectorCDP, len(tableRows(cdpCacheTable, walk(cdpCacheTable))), false)
		tally(scenario.CollectorFDP, len(tableRows(foundryFDPCacheTable, walk(foundryFDPCacheTable))), false)
	}

	return observations
}

// tableRows returns the distinct row indexes a table walk covers. ifTable and
// ifXTable share an index, so merging theirs gives one row per interface, as
// SEED's iftable collector merges them.
func tableRows(prefix string, oids []string) map[string]bool {
	rows := map[string]bool{}
	for _, oid := range oids {
		parts := strings.SplitN(strings.TrimPrefix(oid, prefix+"."), ".", tableIndexAfterColumn)
		if len(parts) == tableIndexAfterColumn {
			rows[parts[1]] = true
		}
	}

	return rows
}

// fdbPorts counts the distinct bridge ports dot1qTpFdbTable places a MAC on:
// SEED counts the switch ports an endpoint is learned on, not MAC entries.
func fdbPorts(stack *protocols.Stack, device *config.Device) int {
	ports := map[any]bool{}
	column := dot1qTpFdbTable + "." + dot1qTpFdbPortColumn + "."
	oids, _ := stack.SNMPWalk(device, dot1qTpFdbTable)
	for _, oid := range oids {
		if !strings.HasPrefix(oid, column) {
			continue
		}
		value, err := stack.SNMPGet(device, oid)
		if err == nil && value != nil {
			ports[value.Value] = true
		}
	}

	return len(ports)
}
