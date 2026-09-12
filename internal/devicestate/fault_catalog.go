package devicestate

import "slices"

// AuthorableFaultTypes returns every fault type the runtime can arm, across all
// four families: the interface and device catalogs, plus the three
// payload-bearing faults that are declared as bare constants because their
// value is an address or a mask rather than a number.
//
// It exists because the authoring layer has to restate this set as a literal
// enum -- a `oneof=` tag, so the generated JSON schema carries the values -- and
// a restated list drifts. It had: poe_loss was a supported interface fault that
// no authored surface accepted, so the one fault a PoE demo needs could be
// injected at runtime and never written down.
func AuthorableFaultTypes() []string {
	types := []string{
		string(FaultDuplicateDHCPOffer),
		string(FaultDuplicateIP),
		string(FaultBadMask),
	}
	for _, definition := range InterfaceFaultDefinitions() {
		types = append(types, string(definition.Type))
	}
	for _, definition := range DeviceFaultDefinitions() {
		types = append(types, string(definition.Type))
	}
	slices.Sort(types)

	return types
}
