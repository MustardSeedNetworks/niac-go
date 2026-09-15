package api

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// unauthoredProtocols names every device capability no shipped pack authors,
// against the reason it is absent. An entry is a statement, not a suppression:
// the test fails when a capability listed here becomes authored, so the list
// can only shrink, and it fails when a capability appears that is on neither
// side, so a protocol added to the runtime cannot arrive unnoticed.
func unauthoredProtocols() map[string]string {
	return map[string]string{
		// P5-9 owns the protocol surface no pack authors. STP is authored by five
		// built-in templates and FTP by one, so those two are reachable from the
		// shipped library but not from a pack; the rest are reachable from nothing.
		"STPConfig":    "P5-9: no pack authors spanning tree (5 built-in templates do)",
		"FTPConfig":    "P5-9: no pack authors FTP (1 built-in template does)",
		"SSHConfig":    "P5-9: no pack authors the SSH command service",
		"SNMPv3Config": "P5-9: no pack authors SNMPv3 USM users alongside v2c",
		"ICMPv6Config": "P5-9: no pack authors dual-stack ICMPv6",
		"PortChannels": "P5-9: no pack authors a LAG, and IF-MIB has no aggregate for one yet",

		// Extreme and Foundry discovery. Vendor-specific, and vendor diversity is
		// out of scope for v1 while the newer models have no walks -- P5-9 records
		// the exclusion rather than leaving it silent.
		"EDPConfig": "P5-9: vendor-specific, deliberately unauthored for v1",
		"FDPConfig": "P5-9: vendor-specific, deliberately unauthored for v1",

		// Authored by no shipped content at all, and owned by no row.
		"Babble":    "#2138: authored nowhere; no row owns it",
		"MapToIP":   "#2138: authored nowhere; no row owns it",
		"TTLConfig": "#2138: authored nowhere; no row owns it",
	}
}

// unreachableFaults is the same contract for the fault catalog.
func unreachableFaults() map[string]string {
	return map[string]string{
		// The three resource alarms are demonstrated by the resource-pressure
		// built-in template, which authors all three on one device beside a healthy
		// control (internal/templates/builtin/resource-pressure.yaml). They are a
		// state a device is driven into rather than a fault a steady-state pack
		// carries, and the packs deliberately do not offer them.
		"cpu_percent":    "demonstrated by the resource-pressure built-in template",
		"memory_percent": "demonstrated by the resource-pressure built-in template",
		"disk_percent":   "demonstrated by the resource-pressure built-in template",
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
func authoredProtocols(t *testing.T) map[string][]string {
	t.Helper()
	deviceType := reflect.TypeFor[config.Device]()
	authored := map[string][]string{}
	for _, pack := range scenario.Packs() {
		cfg := packConfig(t, pack)
		seen := map[string]bool{}
		for i := range cfg.Devices {
			device := reflect.ValueOf(cfg.Devices[i])
			for f := range deviceType.NumField() {
				name := deviceType.Field(f).Name
				if seen[name] || device.Field(f).IsZero() {
					continue
				}
				seen[name] = true
				authored[name] = append(authored[name], pack.ID)
			}
		}
	}

	return authored
}

// TestEveryDeviceProtocolIsAuthoredOrExcluded is the durable half of the
// 2026-09-11 audit. That audit happened because nothing measured what the packs
// author against what the runtime can serve, so the Wi-Fi tier could be scenery
// and syslog could work while no scenario emitted any. A capability is either
// demonstrated by a pack or carries a written reason why not.
func TestEveryDeviceProtocolIsAuthoredOrExcluded(t *testing.T) {
	authored := authoredProtocols(t)

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
func reachableFaults(t *testing.T) map[string][]string {
	t.Helper()
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

	for _, pack := range scenario.Packs() {
		noteAuthoredFaults(t, pack, note)
		noteInjectableFaults(t, pack, note)
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
func noteAuthoredFaults(t *testing.T, pack scenario.Pack, note func(name, pack string)) {
	t.Helper()
	cfg := packConfig(t, pack)
	for i := range cfg.Devices {
		for _, fault := range cfg.Devices[i].Faults {
			note(fault.Type, pack.ID+" (authored)")
		}
		for _, iface := range cfg.Devices[i].Interfaces {
			for _, fault := range iface.Faults {
				note(fault.Type, pack.ID+" (authored)")
			}
		}
	}
}

// noteInjectableFaults records the faults a pack offers an operator to inject.
func noteInjectableFaults(t *testing.T, pack scenario.Pack, note func(name, pack string)) {
	t.Helper()
	stack, _ := packStack(t, pack)
	for _, target := range stack.InterfaceFaultTargets() {
		for _, labels := range target.ErrorTypes {
			for _, label := range labels {
				note(label, pack.ID+" (injectable)")
			}
		}
	}
	for _, target := range stack.DeviceFaultTargets() {
		for _, faultType := range target.Faults {
			note(string(faultType), pack.ID+" (injectable)")
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

// TestEveryFaultTypeIsReachableOrExcluded is the fault half of the matrix.
func TestEveryFaultTypeIsReachableOrExcluded(t *testing.T) {
	reachable := reachableFaults(t)

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

func packConfig(t *testing.T, pack scenario.Pack) *config.Config {
	t.Helper()
	generated, err := scenario.Generate(pack.Request)
	if err != nil {
		t.Fatalf("generate %s: %v", pack.ID, err)
	}

	return generated.Config
}
