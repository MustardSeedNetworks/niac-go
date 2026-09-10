package protocols

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gosnmp/gosnmp"
)

func TestDeviceFaultMIBComparisonRejectsPersistentCorruption(t *testing.T) {
	for _, tc := range []struct {
		name    string
		corrupt gosnmp.SnmpPDU
	}{
		{"value", gosnmp.SnmpPDU{Name: "1.3.6.1.2.1.1.1.0", Type: gosnmp.OctetString, Value: "corrupted"}},
		{"type", gosnmp.SnmpPDU{Name: "1.3.6.1.2.1.1.1.0", Type: gosnmp.Integer, Value: "original"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			baseline := []gosnmp.SnmpPDU{{Name: tc.corrupt.Name, Type: gosnmp.OctetString, Value: "original"}}
			corrupted := []gosnmp.SnmpPDU{tc.corrupt}
			if err := compareDeviceFaultMIBRows(baseline, corrupted, corrupted, nil); err == nil {
				t.Fatal("persistent corruption was accepted as normal counter drift")
			}
		})
	}
}

func TestDeviceFaultMIBComparisonExemptsOnlyKnownLiveValues(t *testing.T) {
	baseline := []gosnmp.SnmpPDU{{Name: "1.3.6.1.2.1.11.16.0", Type: gosnmp.Counter32, Value: uint32(1)}}
	faulted := []gosnmp.SnmpPDU{{Name: baseline[0].Name, Type: gosnmp.Counter32, Value: uint32(2)}}
	settled := []gosnmp.SnmpPDU{{Name: baseline[0].Name, Type: gosnmp.Counter32, Value: uint32(3)}}
	if err := compareDeviceFaultMIBRows(baseline, faulted, settled, nil); err != nil {
		t.Fatalf("known request counter drift rejected: %v", err)
	}
	faulted[0].Type = gosnmp.Integer
	if err := compareDeviceFaultMIBRows(baseline, faulted, settled, nil); err == nil {
		t.Fatal("live-counter type corruption was accepted")
	}
	for _, rows := range [][]gosnmp.SnmpPDU{baseline, faulted, settled} {
		rows[0].Name = "1.3.6.1.2.1.11.30.0" // snmpEnableAuthenTraps is not a counter.
		rows[0].Type = gosnmp.Integer
	}
	if err := compareDeviceFaultMIBRows(baseline, faulted, settled, nil); err == nil {
		t.Fatal("unrelated SNMP-group state was exempted as a live counter")
	}
}

func TestDeviceFaultMIBComparisonRequiresResourceRestoration(t *testing.T) {
	const oid = "1.3.6.1.2.1.25.3.3.1.2.1"
	baseline := []gosnmp.SnmpPDU{{Name: oid, Type: gosnmp.Integer, Value: 18}}
	faulted := []gosnmp.SnmpPDU{{Name: oid, Type: gosnmp.Integer, Value: 90}}
	resources := map[string]string{oid: "90"}
	if err := compareDeviceFaultMIBRows(baseline, faulted, baseline, resources); err != nil {
		t.Fatal(err)
	}
	if err := compareDeviceFaultMIBRows(baseline, faulted, faulted, resources); err == nil {
		t.Fatal("armed resource whitelist exempted an unrestored resource")
	}
	if err := compareDeviceFaultMIBRows(baseline, faulted, baseline, nil); err == nil {
		t.Fatal("unlisted resource substitution was accepted")
	}
}

func compareDeviceFaultMIBRows(
	baseline, faulted, settled []gosnmp.SnmpPDU,
	resources map[string]string,
) error {
	if len(baseline) != len(faulted) || len(baseline) != len(settled) {
		return fmt.Errorf(
			"MIB row counts differ: baseline=%d faulted=%d cleared=%d",
			len(baseline),
			len(faulted),
			len(settled),
		)
	}
	matched := 0
	for index, row := range baseline {
		armed, cleared := faulted[index], settled[index]
		if row.Name != armed.Name || row.Name != cleared.Name || row.Type != armed.Type || row.Type != cleared.Type {
			return fmt.Errorf("row %s changed OID or type", row.Name)
		}
		if want, resource := resources[row.Name]; resource {
			matched++
			if armed.Type != gosnmp.Integer || fmt.Sprint(armed.Value) != want ||
				!reflect.DeepEqual(row.Value, cleared.Value) {
				return fmt.Errorf("resource %s was not substituted or restored", row.Name)
			}
			continue
		}
		// SweepGetNext calls ProcessPDU directly: only sysUpTime,
		// snmpInTotalReqVars and snmpInGetNexts advance in this fixture.
		switch row.Name {
		case "1.3.6.1.2.1.1.3.0", "1.3.6.1.2.1.11.13.0", "1.3.6.1.2.1.11.16.0":
			continue
		}
		if !reflect.DeepEqual(row.Value, armed.Value) || !reflect.DeepEqual(row.Value, cleared.Value) {
			return fmt.Errorf("unaffected row %s changed value", row.Name)
		}
	}
	if matched != len(resources) {
		return fmt.Errorf("resource rows found=%d, want=%d", matched, len(resources))
	}
	return nil
}

// changedOIDs is retained for the separate PoE comparison tests.
func changedOIDs(first, second []gosnmp.SnmpPDU) map[string]struct{} {
	changed := make(map[string]struct{})
	values := make(map[string]any, len(first))
	for _, row := range first {
		values[row.Name] = row.Value
	}
	for _, row := range second {
		if previous, seen := values[row.Name]; !seen || !reflect.DeepEqual(previous, row.Value) {
			changed[row.Name] = struct{}{}
		}
	}
	return changed
}
