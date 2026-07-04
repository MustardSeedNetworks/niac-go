package snmp

import (
	"testing"

	"github.com/gosnmp/gosnmp"
)

// setTableColumns seeds a small conceptual table: for each column OID prefix,
// rows at indices 1, 2, 3.
func setTableColumns(t *testing.T, agent *Agent, columns []string) {
	t.Helper()

	for _, col := range columns {
		for _, idx := range []string{"1", "2", "3"} {
			if err := agent.SetOID(col+"."+idx, &OIDValue{Type: gosnmp.Integer, Value: 1}); err != nil {
				t.Fatalf("SetOID: %v", err)
			}
		}
	}
}

// TestGetBulkInterleavesRepeaters is the regression guard for the bug where a
// multi-column GET-BULK returned column-major bindings (every row of column 1,
// then column 2, …). RFC 3416 §4.2.3 requires row-major interleaving; a manager
// (e.g. a NetAlly CyberScope walking ifTable) reconstructs the table by position
// and could otherwise parse only a single row.
func TestGetBulkInterleavesRepeaters(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)

	colA := "1.3.6.1.2.1.2.2.1.2" // ifDescr-like
	colB := "1.3.6.1.2.1.2.2.1.3" // ifType-like
	colC := "1.3.6.1.2.1.2.2.1.8" // ifOperStatus-like
	setTableColumns(t, agent, []string{colA, colB, colC})

	vars := []gosnmp.SnmpPDU{{Name: colA}, {Name: colB}, {Name: colC}}
	resp := agent.ProcessPDU(gosnmp.GetBulkRequest, vars, 0, 3)

	// Expect row-major: A.1, B.1, C.1, A.2, B.2, C.2, A.3, B.3, C.3.
	want := []string{
		colA + ".1", colB + ".1", colC + ".1",
		colA + ".2", colB + ".2", colC + ".2",
		colA + ".3", colB + ".3", colC + ".3",
	}

	if len(resp) < len(want) {
		t.Fatalf("got %d bindings, want at least %d", len(resp), len(want))
	}

	for i, w := range want {
		if resp[i].Name != w {
			t.Errorf("binding %d = %q, want %q (not interleaved?)", i, resp[i].Name, w)
		}
	}
}

// TestGetBulkNonRepeaters verifies the first nonRepeaters varbinds advance once
// while the rest repeat.
func TestGetBulkNonRepeaters(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)

	col := "1.3.6.1.2.1.2.2.1.2"
	setTableColumns(t, agent, []string{col})

	// sysDescr.0 as the single non-repeater; its GetNext is sysObjectID.0.
	vars := []gosnmp.SnmpPDU{
		{Name: "1.3.6.1.2.1.1.1.0"},
		{Name: col},
	}
	resp := agent.ProcessPDU(gosnmp.GetBulkRequest, vars, 1, 3)

	if len(resp) < 4 {
		t.Fatalf("got %d bindings, want >= 4", len(resp))
	}

	// First binding is the non-repeater advanced once (GetNext of sysDescr.0).
	if resp[0].Name != "1.3.6.1.2.1.1.2.0" {
		t.Errorf("non-repeater binding = %q, want sysObjectID.0", resp[0].Name)
	}

	// The remaining bindings are the repeated column, in order.
	for i, idx := range []string{".1", ".2", ".3"} {
		if resp[1+i].Name != col+idx {
			t.Errorf("repeater binding %d = %q, want %q", i, resp[1+i].Name, col+idx)
		}
	}
}

// TestGetBulkEndOfMib returns endOfMibView once a repeater is exhausted rather
// than looping.
func TestGetBulkEndOfMib(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)

	// A single high OID with nothing after it in the MIB.
	last := "1.3.6.1.4.1.99999.9.9.9"
	if err := agent.SetOID(last, &OIDValue{Type: gosnmp.Integer, Value: 1}); err != nil {
		t.Fatalf("SetOID: %v", err)
	}

	vars := []gosnmp.SnmpPDU{{Name: last}}
	resp := agent.ProcessPDU(gosnmp.GetBulkRequest, vars, 0, 5)

	if len(resp) != 1 {
		t.Fatalf("got %d bindings, want exactly 1 (endOfMibView)", len(resp))
	}

	if resp[0].Type != gosnmp.EndOfMibView {
		t.Errorf("binding type = %v, want EndOfMibView", resp[0].Type)
	}
}
