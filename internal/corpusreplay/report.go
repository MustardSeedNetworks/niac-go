package corpusreplay

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// Summary totals a corpus run.
type Summary struct {
	Recordings    int            `json:"recordings"`
	Statuses      map[string]int `json:"statuses"`
	SourceChanged []string       `json:"source_changed,omitempty"`
	Drifted       []string       `json:"drifted,omitempty"`
	Failed        []string       `json:"failed,omitempty"`

	SourceEntries    int `json:"source_entries"`
	ProbeBindings    int `json:"probe_bindings"`
	RejectedLines    int `json:"rejected_lines"`
	GetMissing       int `json:"get_missing"`
	BERFailures      int `json:"ber_failures"`
	KeptTypeChanges  int `json:"kept_type_changes"`
	KeptValueChanges int `json:"kept_value_changes"`
	OrderingFailures int `json:"ordering_failures"`
}

// Summarize totals results. The path lists keep the results' order.
func Summarize(results []Result) Summary {
	summary := Summary{Recordings: len(results), Statuses: map[string]int{}}
	for _, result := range results {
		receipt := result.Receipt
		summary.Statuses[receipt.Status]++
		if result.SourceChanged {
			summary.SourceChanged = append(summary.SourceChanged, result.Path)
		}
		if len(result.Drift) > 0 {
			summary.Drifted = append(summary.Drifted, result.Path)
		}
		if receipt.Status != snmp.ReplayCompleted {
			summary.Failed = append(summary.Failed, result.Path)
		}
		summary.SourceEntries += receipt.SourceEntries
		summary.ProbeBindings += receipt.ProbeBindings
		summary.RejectedLines += receipt.RejectedLines
		summary.GetMissing += receipt.GetMissing
		summary.BERFailures += receipt.BERFailures
		summary.KeptTypeChanges += receipt.KeptTypeChanges
		summary.KeptValueChanges += receipt.KeptValueChanges
		summary.OrderingFailures += receipt.OrderingFailures
	}

	return summary
}

// OK reports a run in which every recording completed and none drifted from
// its baseline. A changed source is reported but does not fail the run by
// itself: its replay is still compared, and fails through drift if it
// matters.
func (s Summary) OK() bool { return len(s.Failed) == 0 && len(s.Drifted) == 0 }

// WriteJSONL writes one JSON object per result, in the results' order.
func WriteJSONL(w io.Writer, results []Result) error {
	encoder := json.NewEncoder(w)
	for i := range results {
		if err := encoder.Encode(&results[i]); err != nil {
			return fmt.Errorf("write result %s: %w", results[i].Path, err)
		}
	}

	return nil
}

// WriteText writes a human-readable summary.
func (s Summary) WriteText(w io.Writer) error {
	statuses := make([]string, 0, len(s.Statuses))
	for status := range s.Statuses {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)

	lines := []string{fmt.Sprintf("recordings: %d", s.Recordings)}
	for _, status := range statuses {
		lines = append(lines, fmt.Sprintf("  %s: %d", status, s.Statuses[status]))
	}
	lines = append(lines,
		fmt.Sprintf("source entries: %d, probe bindings: %d, rejected lines: %d",
			s.SourceEntries, s.ProbeBindings, s.RejectedLines),
		fmt.Sprintf("get missing: %d, BER failures: %d, kept type/value changes: %d/%d, ordering failures: %d",
			s.GetMissing, s.BERFailures, s.KeptTypeChanges, s.KeptValueChanges, s.OrderingFailures),
		fmt.Sprintf("source changed since qualification: %d", len(s.SourceChanged)),
		fmt.Sprintf("drifted from baseline: %d", len(s.Drifted)),
	)
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return fmt.Errorf("write summary: %w", err)
		}
	}

	return nil
}
