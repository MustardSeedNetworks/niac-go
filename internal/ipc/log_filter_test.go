package ipc

import "testing"

// `niac logs tail --filter X` and `niac logs tail --follow --filter X` are the
// same flag and must answer the same way. They did not: the follow path matched
// only the message, with a hand-rolled ASCII case fold, while the non-follow
// path also matched device, source and protocol with a proper one.
func TestFilterMatchesEveryFieldTheUserCanSee(t *testing.T) {
	entry := LogEntry{
		Message:  "link state changed",
		Device:   "MED-ACC-SW02",
		Source:   "stack",
		Protocol: "LLDP",
	}

	for _, filter := range []string{"med-acc-sw02", "STACK", "lldp", "LINK STATE"} {
		if !LogMatchesFilter(entry, filter) {
			t.Errorf("filter %q did not match a record the user can see it in", filter)
		}
	}
	if LogMatchesFilter(entry, "nothing-here") {
		t.Error("filter matched a record containing none of it")
	}
}

// An empty filter is not a filter.
func TestEmptyFilterKeepsEverything(t *testing.T) {
	if !LogMatchesFilter(LogEntry{Message: "anything"}, "") {
		t.Error("an empty filter dropped a record")
	}
}

// The follow path folded case by hand, over ASCII only, so a non-ASCII filter
// silently stopped being case-insensitive.
func TestFilterFoldsCaseBeyondASCII(t *testing.T) {
	entry := LogEntry{Message: "ÜBERTRAGUNG fehlgeschlagen"}

	if !LogMatchesFilter(entry, "übertragung") {
		t.Error("a non-ASCII filter did not fold case")
	}
}
