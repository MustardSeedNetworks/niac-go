package snmp

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gosnmp/gosnmp"
)

const (
	float32Bits = 32
	float64Bits = 64
)

func formatCapturedEntry(entry WalkEntry) string {
	oid := entry.OID
	if !strings.HasPrefix(oid, ".") {
		oid = "." + oid
	}
	if entry.Type == gosnmp.OctetString {
		return formatCapturedOctets(oid, entry.Value)
	}
	if entry.Type == gosnmp.BitString {
		return formatCapturedBytes(oid, "BITS", entry.Value)
	}
	if entry.Type == gosnmp.Opaque {
		return formatCapturedBytes(oid, "Opaque", entry.Value)
	}
	if entry.Type == gosnmp.OpaqueFloat {
		return formatCapturedFloat(oid, "Opaque Float", entry.Value, float32Bits)
	}
	if entry.Type == gosnmp.OpaqueDouble {
		return formatCapturedFloat(oid, "Opaque Double", entry.Value, float64Bits)
	}
	return formatWalkEntry(oid, &OIDValue{Type: entry.Type, Value: entry.Value})
}

func formatCapturedOctets(oid string, value any) string {
	octets, ok := value.([]byte)
	if !ok {
		return fmt.Sprintf("%s = %s: %q", oid, snmpTypeSTRING, fmt.Sprint(value))
	}
	if utf8.Valid(octets) && isPrintableWalkText(octets) && !bytes.ContainsAny(octets, `"\`) {
		return fmt.Sprintf("%s = %s: %q", oid, snmpTypeSTRING, string(octets))
	}
	return fmt.Sprintf("%s = Hex-STRING: %s", oid, hexOctets(octets))
}

func formatCapturedBytes(oid, typeName string, value any) string {
	octets, ok := value.([]byte)
	if !ok {
		octets = fmt.Append(nil, value)
	}
	return fmt.Sprintf("%s = %s: HEX %s", oid, typeName, hexOctets(octets))
}

func formatCapturedFloat(oid, typeName string, value any, bits int) string {
	number, ok := capturedFloat(value)
	if !ok {
		return fmt.Sprintf("%s = %s: %v", oid, typeName, value)
	}
	return fmt.Sprintf("%s = %s: %s", oid, typeName, strconv.FormatFloat(number, 'g', -1, bits))
}

func capturedFloat(value any) (float64, bool) {
	switch number := value.(type) {
	case float32:
		return float64(number), true
	case float64:
		return number, true
	default:
		return 0, false
	}
}

func hexOctets(octets []byte) string {
	encoded := strings.ToUpper(hex.EncodeToString(octets))
	parts := make([]string, 0, len(encoded)/hexOctetWidth)
	for index := 0; index < len(encoded); index += hexOctetWidth {
		parts = append(parts, encoded[index:index+hexOctetWidth])
	}
	return strings.Join(parts, " ")
}

func isPrintableWalkText(value []byte) bool {
	for _, character := range string(value) {
		if character < ' ' && character != '\t' {
			return false
		}
	}
	return true
}
