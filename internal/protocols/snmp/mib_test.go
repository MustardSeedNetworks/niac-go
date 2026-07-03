package snmp

import (
	"fmt"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"
)

// buildLargeMIB populates a MIB with a realistic large interface table so the
// sort/index path is exercised at production scale.
func buildLargeMIB(tb testing.TB, ifCount int) *MIB {
	tb.Helper()

	m := NewMIB()
	// A per-interface row across several columns mirrors ifTable/ifXTable shape.
	for _, col := range []string{"2.2.1.1", "2.2.1.2", "2.2.1.3", "2.2.1.5", "2.2.1.10", "31.1.1.1.1"} {
		for i := 1; i <= ifCount; i++ {
			oid := fmt.Sprintf("1.3.6.1.2.1.%s.%d", col, i)
			m.Set(oid, &OIDValue{Type: gosnmp.Integer, Value: i})
		}
	}

	return m
}

// TestMIBReindexIsFast guards the regression where compareOIDs used fmt.Sscanf:
// sorting a large device's OID space took seconds on the hot path and stalled
// multi-device SNMP discovery. Reindexing a ~15k-OID MIB must be quick.
func TestMIBReindexIsFast(t *testing.T) {
	m := buildLargeMIB(t, 2500) // 6 columns * 2500 = 15000 OIDs

	start := time.Now()
	m.Reindex()
	elapsed := time.Since(start)

	if elapsed > 250*time.Millisecond {
		t.Fatalf("Reindex of 15k OIDs took %v, want < 250ms (Sscanf regression?)", elapsed)
	}
}

// TestMIBGetNextWalksAllInOrder verifies a full GetNext traversal returns every
// OID exactly once in ascending numeric order — the behavior an SNMP walk relies
// on. Numeric (not lexical) ordering matters: arc 10 must sort after arc 2.
func TestMIBGetNextWalksAllInOrder(t *testing.T) {
	const ifCount = 300
	m := buildLargeMIB(t, ifCount)
	m.Reindex()

	want := 6 * ifCount

	seen := 0
	cur := "1"
	var prev []int
	for {
		next, val := m.GetNext(cur)
		if next == "" || val == nil {
			break
		}

		parts := parseOIDParts(next)
		if prev != nil && compareOIDParts(prev, parts) >= 0 {
			t.Fatalf("GetNext not strictly ascending: %v then %v", prev, parts)
		}
		prev = parts
		seen++
		cur = next

		if seen > want+10 {
			t.Fatal("GetNext walk did not terminate — index likely broken")
		}
	}

	if seen != want {
		t.Fatalf("walk returned %d OIDs, want %d", seen, want)
	}
}

// TestCompareOIDParts pins the numeric ordering contract.
func TestCompareOIDParts(t *testing.T) {
	const mib2 = "1.3.6.1.2.1" // shared prefix, kept as one literal for goconst

	cases := []struct {
		a, b string
		want int
	}{
		{mib2 + ".2.2.1.1.2", mib2 + ".2.2.1.1.10", -1}, // arc 2 < 10 numerically
		{mib2 + ".1.1.0", mib2 + ".1.1.0", 0},
		{mib2 + ".1", mib2 + ".1.0", -1},  // shorter prefix sorts first
		{"1.3.6.1.4.1.9", mib2 + ".1", 1}, // different enterprise branch sorts after mib-2
	}
	for _, c := range cases {
		if got := compareOIDs(c.a, c.b); got != c.want {
			t.Errorf("compareOIDs(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
