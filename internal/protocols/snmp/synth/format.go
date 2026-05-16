package synth

import (
	"fmt"
	"io"
)

// Walk-file line emission helpers. The format the simulator's walk
// parser accepts is:
//
//   .1.3.6.1.2.1.1.5.0 = STRING: "router-1"
//
// We always include the leading dot. Whitespace + quoting match what
// `snmpwalk -OnQ` emits so the simulator's existing parser handles
// these lines without special-casing.

func emitString(w io.Writer, oid, value string) {
	fmt.Fprintf(w, ".%s = STRING: %q\n", oid, value)
}

func emitOID(w io.Writer, oid, value string) {
	fmt.Fprintf(w, ".%s = OID: .%s\n", oid, value)
}

func emitInteger(w io.Writer, oid string, value int) {
	fmt.Fprintf(w, ".%s = INTEGER: %d\n", oid, value)
}

func emitCounter32(w io.Writer, oid string, value uint32) {
	fmt.Fprintf(w, ".%s = Counter32: %d\n", oid, value)
}

func emitGauge32(w io.Writer, oid string, value uint32) {
	fmt.Fprintf(w, ".%s = Gauge32: %d\n", oid, value)
}

func emitTimeticks(w io.Writer, oid string, value uint32) {
	fmt.Fprintf(w, ".%s = Timeticks: (%d) 0:00:00.00\n", oid, value)
}

func emitIPAddress(w io.Writer, oid, value string) {
	fmt.Fprintf(w, ".%s = IpAddress: %s\n", oid, value)
}

func emitPhysAddress(w io.Writer, oid, value string) {
	// Walk format is "Hex-STRING: aa bb cc dd ee ff" but most tools
	// (including the niac validator) accept the MAC literal too. Keep
	// it simple.
	fmt.Fprintf(w, ".%s = STRING: %q\n", oid, value)
}
