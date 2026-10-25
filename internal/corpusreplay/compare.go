package corpusreplay

import (
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// compareField is one manifest column the replay reproduces.
type compareField struct {
	column string
	value  func(snmp.ReplayReceipt) string
	kind   fieldKind
}

type fieldKind int

const (
	kindText fieldKind = iota
	kindInt
	kindBool
	kindCounts
)

const colUDPSmoke = "udp_smoke"

// compareFields lists the receipt columns in manifest order. Wall time and
// the platform it ran on are not compared: they describe the run, not the
// recording.
func compareFields() []compareField {
	count := func(column string, get func(snmp.ReplayReceipt) int) compareField {
		return compareField{column: column, kind: kindInt, value: func(r snmp.ReplayReceipt) string {
			return strconv.Itoa(get(r))
		}}
	}

	return []compareField{
		{column: "status", kind: kindText, value: func(r snmp.ReplayReceipt) string { return r.Status }},
		count("source_entries", func(r snmp.ReplayReceipt) int { return r.SourceEntries }),
		count("rejected_lines", func(r snmp.ReplayReceipt) int { return r.RejectedLines }),
		count("invalid_oids", func(r snmp.ReplayReceipt) int { return r.InvalidOIDs }),
		count("probe_bindings", func(r snmp.ReplayReceipt) int { return r.ProbeBindings }),
		count("get_missing", func(r snmp.ReplayReceipt) int { return r.GetMissing }),
		count("contract_omitted", func(r snmp.ReplayReceipt) int { return r.ContractOmitted }),
		count("opaque_wire_preserved", func(r snmp.ReplayReceipt) int { return r.OpaqueWirePreserved }),
		count("ber_failures", func(r snmp.ReplayReceipt) int { return r.BERFailures }),
		count("ber_type_changes", func(r snmp.ReplayReceipt) int { return r.BERTypeChanges }),
		count("ber_value_changes", func(r snmp.ReplayReceipt) int { return r.BERValueChanges }),
		count("contract_kept", func(r snmp.ReplayReceipt) int { return r.ContractKept }),
		count("contract_substituted", func(r snmp.ReplayReceipt) int { return r.ContractSubstituted }),
		count("kept_type_changes", func(r snmp.ReplayReceipt) int { return r.KeptTypeChanges }),
		count("kept_value_changes", func(r snmp.ReplayReceipt) int { return r.KeptValueChanges }),
		{column: "source_identity", kind: kindBool, value: func(r snmp.ReplayReceipt) string {
			return strconv.FormatBool(r.Identity)
		}},
		count("next_checks", func(r snmp.ReplayReceipt) int { return r.NextChecks }),
		count("bulk_checks", func(r snmp.ReplayReceipt) int { return r.BulkChecks }),
		count("ordering_failures", func(r snmp.ReplayReceipt) int { return r.OrderingFailures }),
		{column: colUDPSmoke, kind: kindBool, value: func(r snmp.ReplayReceipt) string {
			return strconv.FormatBool(r.Smoke)
		}},
		{column: "failure_types", kind: kindCounts, value: func(r snmp.ReplayReceipt) string {
			return formatCounts(r.FailureTypes)
		}},
	}
}

// Compare returns every baseline column the receipt disagrees with. Columns
// the baseline does not carry are not compared, and udp_smoke is compared only
// when the smoke test ran.
func Compare(baseline map[string]string, receipt snmp.ReplayReceipt, smoke bool) []string {
	if baseline == nil {
		return nil
	}
	var drift []string
	for _, field := range compareFields() {
		want, ok := baseline[field.column]
		if !ok || (field.column == colUDPSmoke && !smoke) {
			continue
		}
		got := field.value(receipt)
		if !sameField(field.kind, want, got) {
			drift = append(drift, fmt.Sprintf("%s: baseline %s, replay %s", field.column, want, got))
		}
	}

	return drift
}

func sameField(kind fieldKind, want, got string) bool {
	switch kind {
	case kindInt:
		wantN, wantErr := strconv.Atoi(strings.TrimSpace(want))
		gotN, gotErr := strconv.Atoi(got)

		return wantErr == nil && gotErr == nil && wantN == gotN
	case kindBool:
		wantB, wantErr := strconv.ParseBool(strings.TrimSpace(want))
		gotB, gotErr := strconv.ParseBool(got)

		return wantErr == nil && gotErr == nil && wantB == gotB
	case kindCounts:
		wantCounts, ok := parseCounts(want)

		return ok && formatCounts(wantCounts) == got
	case kindText:
		return strings.TrimSpace(want) == got
	default:
		return false
	}
}

// parseCounts reads a failure_types cell. The qualification worker wrote it
// as JSON; a Python dict literal with single quotes is accepted too.
func parseCounts(cell string) (map[string]int, bool) {
	cell = strings.TrimSpace(cell)
	if cell == "" {
		return map[string]int{}, true
	}
	counts := map[string]int{}
	if json.Unmarshal([]byte(cell), &counts) == nil {
		return counts, true
	}
	if json.Unmarshal([]byte(strings.ReplaceAll(cell, "'", `"`)), &counts) == nil {
		return counts, true
	}

	return nil, false
}

// formatCounts renders counts as JSON with sorted keys.
func formatCounts(counts map[string]int) string {
	normalized := maps.Clone(counts)
	if normalized == nil {
		normalized = map[string]int{}
	}
	body, err := json.Marshal(normalized)
	if err != nil {
		return "{}"
	}

	return string(body)
}
