package api

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// unauthoredProtocols names every device capability no shipped pack authors,
// against the reason it is absent. An entry is a statement, not a suppression:
// the test fails when a capability listed here becomes authored, so the list
// can only shrink, and it fails when a capability appears that is on neither
// side, so a protocol added to the runtime cannot arrive unnoticed.
func unauthoredProtocols() map[string]string {
	return map[string]string{
		// P5-9 owns the protocol surface no pack authors. FTP is authored by one
		// built-in scenario, so it is reachable from the shipped library but not
		// from a pack; the rest are reachable from nothing.
		"FTPConfig":    "P5-9: no pack authors FTP (1 built-in scenario does)",
		"ICMPv6Config": "P5-9: no pack authors dual-stack ICMPv6",

		// Extreme and Foundry discovery. Vendor-specific, and vendor diversity is
		// out of scope for v1 while the newer models have no walks -- P5-9 records
		// the exclusion rather than leaving it silent.
		"EDPConfig": "P5-9: vendor-specific, deliberately unauthored for v1",
		"FDPConfig": "P5-9: vendor-specific, deliberately unauthored for v1",

		// Hand-authoring only. Each is a knob an operator can set in a config by
		// hand, and no consumer of a pack needs it: no collector or
		// troubleshooting flow reads junk traffic, a UDP remap or a shaped
		// traceroute TTL as evidence of anything.
		"Babble":    "hand-authoring only: no consumer needs a pack to emit junk traffic",
		"MapToIP":   "hand-authoring only: no consumer needs a pack to remap UDP traffic",
		"TTLConfig": "hand-authoring only: no consumer needs a pack to shape traceroute TTLs",
	}
}

// unreachableFaults is the same contract for the fault catalog.
func unreachableFaults() map[string]string {
	return map[string]string{
		// The three resource alarms are demonstrated by the resource-pressure
		// built-in scenario, which authors all three on one device beside a healthy
		// control (internal/templates/builtin/resource-pressure.yaml). They are a
		// state a device is driven into rather than a fault a steady-state pack
		// carries, and the packs deliberately do not offer them.
		"cpu_percent":    "demonstrated by the resource-pressure built-in scenario",
		"memory_percent": "demonstrated by the resource-pressure built-in scenario",
		"disk_percent":   "demonstrated by the resource-pressure built-in scenario",
	}
}

// deviceProtocolCapabilities lists the fields of config.Device that represent a
// protocol service a device can run, derived rather than written out: every
// pointer-to-struct field, plus the few carriers that are not pointers. A new
// *FooConfig therefore enters the matrix by existing.
func deviceProtocolCapabilities() []string {
	// Not pointer-to-struct, and each a capability the runtime serves rather
	// than a fact about topology or identity.
	extra := map[string]bool{
		"SNMPConfig": true, "PortChannels": true, "TrunkPorts": true,
		"Babble": true, "MapToIP": true,
	}
	deviceType := reflect.TypeFor[config.Device]()
	names := make([]string, 0, deviceType.NumField())
	for field := range deviceType.Fields() {
		isConfigPointer := field.Type.Kind() == reflect.Pointer &&
			field.Type.Elem().Kind() == reflect.Struct
		if isConfigPointer || extra[field.Name] {
			names = append(names, field.Name)
		}
	}
	sort.Strings(names)

	return names
}

// authoredProtocols reports, for each capability, the packs that set it.
func authoredProtocols(packs []generatedPack) map[string][]string {
	deviceType := reflect.TypeFor[config.Device]()
	authored := map[string][]string{}
	for _, pack := range packs {
		seen := map[string]bool{}
		for i := range pack.cfg.Devices {
			device := reflect.ValueOf(pack.cfg.Devices[i])
			for f := range deviceType.NumField() {
				name := deviceType.Field(f).Name
				if seen[name] || device.Field(f).IsZero() {
					continue
				}
				seen[name] = true
				authored[name] = append(authored[name], pack.id)
			}
		}
	}

	return authored
}

// TestShippedPacks runs every check that walks all the shipped packs over one
// generation of each. Generation grows with pack size and is about ten times
// slower under the race detector, and each check generating the packs again
// (the fault check twice) took the package's -race run from 375 to 468 of
// CI's 600 s as P-PACK-1 resized them. The packs are shared, so the checks
// only read them.
func TestShippedPacks(t *testing.T) {
	packs := generateShippedPacks(t)

	t.Run("every device protocol is authored or excluded", func(t *testing.T) {
		checkEveryDeviceProtocolIsAuthoredOrExcluded(t, packs)
	})
	t.Run("every fault type is reachable or excluded", func(t *testing.T) {
		checkEveryFaultTypeIsReachableOrExcluded(t, packs)
	})
	t.Run("every pack offers something to inject", func(t *testing.T) {
		checkEveryPackOffersSomethingToInject(t, packs)
	})
}

// generatedPack is one shipped pack's config and the runtime stack over it.
type generatedPack struct {
	id    string
	cfg   *config.Config
	stack *protocols.Stack
}

func generateShippedPacks(t *testing.T) []generatedPack {
	t.Helper()
	packs := make([]generatedPack, 0, len(scenario.Packs()))
	for _, pack := range scenario.Packs() {
		generated, err := scenario.Generate(pack.Request)
		if err != nil {
			t.Fatalf("generate %s: %v", pack.ID, err)
		}
		packs = append(packs, generatedPack{
			id:    pack.ID,
			cfg:   generated.Config,
			stack: protocols.NewStack(nil, generated.Config, logging.NewDebugConfig(0)),
		})
	}

	return packs
}

// checkEveryDeviceProtocolIsAuthoredOrExcluded is the durable half of the
// 2026-09-11 audit. That audit happened because nothing measured what the packs
// author against what the runtime can serve, so the Wi-Fi tier could be scenery
// and syslog could work while no scenario emitted any. A capability is either
// demonstrated by a pack or carries a written reason why not.
func checkEveryDeviceProtocolIsAuthoredOrExcluded(t *testing.T, packs []generatedPack) {
	t.Helper()
	authored := authoredProtocols(packs)

	var missing, stale []string
	for _, capability := range deviceProtocolCapabilities() {
		_, excluded := unauthoredProtocols()[capability]
		switch {
		case len(authored[capability]) == 0 && !excluded:
			missing = append(missing, capability)
		case len(authored[capability]) > 0 && excluded:
			stale = append(stale, capability+" (now authored by "+
				strings.Join(authored[capability], ", ")+")")
		}
	}
	if len(missing) > 0 {
		t.Errorf("no pack authors these, and no reason is recorded:\n  %s",
			strings.Join(missing, "\n  "))
	}
	if len(stale) > 0 {
		t.Errorf("recorded as unauthored, but a pack now authors them -- drop the entry:\n  %s",
			strings.Join(stale, "\n  "))
	}
}

// reachableFaults reports, for each fault type, how it is reachable from a
// pack: authored into the scenario, or offered to the operator to inject.
//
// Both count, because a pack is a healthy network carrying one finding (P2-7):
// every other fault is meant to be injected during a demo, not baked in. A
// catalog entry that is neither authored nor offered anywhere is one an
// operator has no way to produce.
func reachableFaults(packs []generatedPack) map[string][]string {
	canonical := faultTypesByLabel()
	reachable := map[string]map[string]bool{}
	note := func(name, pack string) {
		if resolved, ok := canonical[name]; ok {
			name = resolved
		}
		if reachable[name] == nil {
			reachable[name] = map[string]bool{}
		}
		reachable[name][pack] = true
	}

	for _, pack := range packs {
		noteAuthoredFaults(pack, note)
		noteInjectableFaults(pack, note)
	}

	result := map[string][]string{}
	for faultType, packs := range reachable {
		for pack := range packs {
			result[faultType] = append(result[faultType], pack)
		}
		sort.Strings(result[faultType])
	}

	return result
}

// noteAuthoredFaults records the faults a pack writes into its own scenario.
func noteAuthoredFaults(pack generatedPack, note func(name, pack string)) {
	for i := range pack.cfg.Devices {
		for _, fault := range pack.cfg.Devices[i].Faults {
			note(fault.Type, pack.id+" (authored)")
		}
		for _, iface := range pack.cfg.Devices[i].Interfaces {
			for _, fault := range iface.Faults {
				note(fault.Type, pack.id+" (authored)")
			}
		}
	}
}

// noteInjectableFaults records the faults a pack offers an operator to inject.
func noteInjectableFaults(pack generatedPack, note func(name, pack string)) {
	for _, target := range pack.stack.InterfaceFaultTargets() {
		for _, labels := range target.ErrorTypes {
			for _, label := range labels {
				note(label, pack.id+" (injectable)")
			}
		}
	}
	for _, target := range pack.stack.DeviceFaultTargets() {
		for _, faultType := range target.Faults {
			note(string(faultType), pack.id+" (injectable)")
		}
	}
}

// faultTypesByLabel maps every operator-facing fault name back to its catalog
// type. The runtime advertises interface faults by label and device faults by
// type, so the two axes have to be reduced to one vocabulary before they can be
// counted together.
func faultTypesByLabel() map[string]string {
	byLabel := map[string]string{
		devicestate.FaultDuplicateIP.Label(): string(devicestate.FaultDuplicateIP),
		devicestate.FaultBadMask.Label():     string(devicestate.FaultBadMask),
	}
	for _, definition := range devicestate.InterfaceFaultDefinitions() {
		byLabel[definition.Label] = string(definition.Type)
	}
	for _, definition := range devicestate.DeviceFaultDefinitions() {
		byLabel[definition.Label] = string(definition.Type)
	}

	return byLabel
}

// checkEveryFaultTypeIsReachableOrExcluded is the fault half of the matrix.
func checkEveryFaultTypeIsReachableOrExcluded(t *testing.T, packs []generatedPack) {
	t.Helper()
	reachable := reachableFaults(packs)

	var missing, stale []string
	for _, faultType := range devicestate.AuthorableFaultTypes() {
		_, excluded := unreachableFaults()[faultType]
		switch {
		case len(reachable[faultType]) == 0 && !excluded:
			missing = append(missing, faultType)
		case len(reachable[faultType]) > 0 && excluded:
			stale = append(stale, faultType+" (now reachable via "+
				strings.Join(reachable[faultType], ", ")+")")
		}
	}
	if len(missing) > 0 {
		t.Errorf("no pack authors or offers these, and no reason is recorded:\n  %s",
			strings.Join(missing, "\n  "))
	}
	if len(stale) > 0 {
		t.Errorf("recorded as unreachable, but a pack now reaches them -- drop the entry:\n  %s",
			strings.Join(stale, "\n  "))
	}
}
