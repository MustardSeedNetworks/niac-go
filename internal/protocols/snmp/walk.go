package snmp

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

// SNMP type name constants for walk file parsing and formatting.
const (
	snmpTypeSTRING  = "STRING"
	snmpTypeINTEGER = "INTEGER"
	snmpTypeOID     = "OID"
	snmpTypeUnknown = "Unknown"
)

// Walk-file scanner buffer sizes. Long Hex-STRING and multi-line values in a
// large device walk can exceed bufio.Scanner's 64KiB default token size.
const (
	walkScanBufInitial = 64 * 1024
	walkScanBufMax     = 4 * 1024 * 1024
	endOfMIBMarker     = "No more variables left in this MIB View"
)

// isEndOfMIBLine reports whether net-snmp emitted its terminal walk marker.
// The marker is traversal metadata, not an SNMP variable binding.
func isEndOfMIBLine(line string) bool {
	_, value, ok := strings.Cut(line, "=")

	return ok && strings.HasPrefix(strings.TrimSpace(value), endOfMIBMarker)
}

// isHexStringLine reports whether a parsed walk line declared a Hex-STRING
// value, and therefore may be continued on following bare-hex lines.
func isHexStringLine(line string) bool {
	_, rest, ok := strings.Cut(line, "=")
	if !ok {
		return false
	}

	typeTok, _, ok := strings.Cut(rest, ":")
	if !ok {
		return false
	}

	switch strings.ToUpper(strings.TrimSpace(typeTok)) {
	case "HEX-STRING", "HEX":
		return true
	default:
		return false
	}
}

// isHexContinuation reports whether a line is a bare Hex-STRING continuation:
// one or more whitespace-separated pairs of hex digits and nothing else. These
// lines are emitted by net-snmp when a Hex-STRING value wraps across lines.
func isHexContinuation(line string) bool {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false
	}

	for _, f := range fields {
		if len(f) != 2 || !isHexByte(f) {
			return false
		}
	}

	return true
}

func isHexByte(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}

	return true
}

// appendHexContinuation folds a Hex-STRING continuation line's octets into the
// preceding entry. Values are kept as binary octets so SNMP marshaling emits
// the captured bytes rather than their hexadecimal text representation.
func appendHexContinuation(entry *WalkEntry, line string) {
	if entry.Type != gosnmp.OctetString {
		return
	}

	prev, ok := entry.Value.([]byte)
	if !ok {
		return
	}

	continuation, err := decodeHexOctets(line)
	if err != nil {
		return
	}
	combined := append([]byte(nil), prev...)
	combined = append(combined, continuation...)
	entry.Value = combined
}

// WalkEntry represents a single entry from an SNMP walk file.
type WalkEntry struct {
	OID   string
	Type  gosnmp.Asn1BER
	Value any
}

// ParseWalkFile parses an SNMP walk file
// Walk files are typically in the format:
// OID = TYPE: VALUE
// For example:
// .1.3.6.1.2.1.1.1.0 = STRING: "Cisco IOS Software"
// .1.3.6.1.2.1.1.3.0 = Timeticks: (12345) 0:02:03.45
//
// SECURITY: This function validates the file path to prevent directory traversal attacks.
func ParseWalkFile(filename string) ([]WalkEntry, error) {
	// SECURITY FIX #2.8.1: Prevent path traversal attacks
	// Clean the path and check for traversal sequences
	cleanPath := filepath.Clean(filename)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return nil, ErrDirectoryTraversal
	}

	// Convert to absolute path for validation
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("invalid walk file path: %w", err)
	}

	// Ensure file exists and is a regular file (not a symlink or directory)
	fileInfo, err := os.Lstat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access walk file: %w", err)
	}

	// Reject symlinks to prevent symlink attacks
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		return nil, ErrWalkFileSymlink
	}

	// Reject directories
	if fileInfo.IsDir() {
		return nil, ErrWalkFileIsDirectory
	}

	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open walk file: %w", err)
	}

	defer func() { _ = file.Close() }()

	var entries []WalkEntry

	scanner := bufio.NewScanner(file)
	// net-snmp wraps long Hex-STRING and multi-line octet-string values across
	// several lines; a 30k-OID switch walk can carry lines well over the 64KiB
	// default token size, so grow the buffer to avoid a mid-walk scan error.
	scanner.Buffer(make([]byte, 0, walkScanBufInitial), walkScanBufMax)

	lineNum := 0
	lastWasHex := false // previous entry came from a Hex-STRING (may continue)

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if isEndOfMIBLine(line) {
			continue
		}

		// net-snmp continues long Hex-STRING values on subsequent lines that
		// carry only whitespace-separated hex octets (no "OID = TYPE:" prefix).
		// Fold such a line into the previous entry instead of dropping it —
		// otherwise the parent value is silently truncated and the line lost.
		if lastWasHex && len(entries) > 0 && isHexContinuation(line) {
			appendHexContinuation(&entries[len(entries)-1], line)

			continue
		}

		entry, parseErr := parseWalkLine(line)
		if parseErr != nil {
			// Log error but continue parsing
			_, _ = fmt.Fprintf(os.Stdout, "Warning: line %d: %v\n", lineNum, parseErr)

			continue
		}

		lastWasHex = isHexStringLine(line)
		entries = append(entries, *entry)
	}

	scanErr := scanner.Err()
	if scanErr != nil {
		return nil, fmt.Errorf("error reading walk file: %w", scanErr)
	}

	return entries, nil
}

// extractLineOID extracts the OID portion of a raw walk-file line (everything
// before the first "="), trimmed of whitespace. Returns "" if the line has no
// "=" separator. Shared between ParseWalkFile's line parser and the walk
// validator so both agree on what counts as "the OID for this line".
func extractLineOID(line string) string {
	parts := strings.SplitN(strings.TrimSpace(line), "=", OIDPartsMinPDU)
	if len(parts) != OIDPartsMinPDU {
		return ""
	}

	return strings.TrimSpace(parts[0])
}

// parseWalkLine parses a single line from a walk file.
func parseWalkLine(line string) (*WalkEntry, error) {
	// Match pattern: OID = TYPE: VALUE
	// Example: .1.3.6.1.2.1.1.1.0 = STRING: "Cisco IOS"
	parts := strings.SplitN(line, "=", OIDPartsMinPDU)
	if len(parts) != OIDPartsMinPDU {
		return nil, ErrInvalidWalkFormat
	}

	oid := extractLineOID(line)
	rest := strings.TrimSpace(parts[1])

	// net-snmp emits zero-length octet strings with no type prefix, as
	// `OID = ""` (or a bare `OID = `). Treat these as an empty OctetString
	// rather than dropping the OID — a c3750 walk carries ~1,275 of them
	// (empty ifAlias, LLDP remote fields, etc.).
	if rest == "" || rest == `""` {
		return &WalkEntry{OID: oid, Type: gosnmp.OctetString, Value: ""}, nil
	}

	// Parse TYPE: VALUE
	typeParts := strings.SplitN(rest, ":", OIDPartsMinPDU)
	if len(typeParts) != OIDPartsMinPDU {
		return nil, ErrMissingColon
	}

	typeStr := strings.ToUpper(strings.TrimSpace(typeParts[0]))
	valueStr := strings.TrimSpace(typeParts[1])

	// Determine SNMP type and parse value
	asnType, value, err := parseTypeAndValue(typeStr, valueStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToParseValue, err)
	}

	return &WalkEntry{
		OID:   oid,
		Type:  asnType,
		Value: value,
	}, nil
}

// parseTypeAndValue parses the type and value from a walk file entry.
func parseTypeAndValue(typeStr, valueStr string) (gosnmp.Asn1BER, any, error) {
	switch typeStr {
	case snmpTypeSTRING, "OCTET STRING":
		return gosnmp.OctetString, strings.Trim(valueStr, "\""), nil
	case snmpTypeINTEGER, "INT":
		return parseIntegerValue(valueStr)
	case "GAUGE", "GAUGE32":
		return parseGaugeValue(valueStr)
	case "COUNTER", "COUNTER32":
		return parseCounter32Value(valueStr)
	case "COUNTER64":
		return parseCounter64Value(valueStr)
	case "TIMETICKS":
		return parseTimeticksValue(valueStr)
	case snmpTypeOID, "OBJECT IDENTIFIER":
		return gosnmp.ObjectIdentifier, strings.TrimPrefix(valueStr, "."), nil
	case "IPADDRESS", "IP ADDRESS", "IPADDR":
		return gosnmp.IPAddress, valueStr, nil
	case "BITS":
		return parseHexStringValue(valueStr)
	case "HEX-STRING", "HEX":
		return parseHexStringValue(valueStr)
	case "OPAQUE":
		return gosnmp.Opaque, valueStr, nil
	case "NULL":
		return gosnmp.Null, nil, nil
	default:
		return gosnmp.OctetString, valueStr, nil
	}
}

func parseIntegerValue(valueStr string) (gosnmp.Asn1BER, any, error) {
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to parse integer: %w", err)
	}
	bounded := safeInt32(value)

	return gosnmp.Integer, int(bounded), nil
}

func parseGaugeValue(valueStr string) (gosnmp.Asn1BER, any, error) {
	value, err := strconv.ParseUint(valueStr, 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to parse gauge: %w", err)
	}
	bounded := safeUint32FromUint64(value)

	return gosnmp.Gauge32, uint(bounded), nil
}

func parseCounter32Value(valueStr string) (gosnmp.Asn1BER, any, error) {
	value, err := strconv.ParseUint(valueStr, 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to parse counter32: %w", err)
	}
	bounded := safeUint32FromUint64(value)

	return gosnmp.Counter32, uint(bounded), nil
}

func parseCounter64Value(valueStr string) (gosnmp.Asn1BER, any, error) {
	value, err := strconv.ParseUint(valueStr, 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to parse counter64: %w", err)
	}

	return gosnmp.Counter64, value, nil
}

func parseTimeticksValue(valueStr string) (gosnmp.Asn1BER, any, error) {
	// Parse format: (12345) or just 12345
	re := regexp.MustCompile(`\((\d+)\)|(\d+)`)

	matches := re.FindStringSubmatch(valueStr)
	if len(matches) == 0 {
		return 0, nil, fmt.Errorf("%w: %s", ErrInvalidTimeticksFormat, valueStr)
	}

	var numStr string
	if matches[1] != "" {
		numStr = matches[1]
	} else {
		numStr = matches[2]
	}

	value, err := strconv.ParseUint(numStr, 10, 32)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to parse timeticks: %w", err)
	}

	return gosnmp.TimeTicks, uint32(value), nil
}

func parseHexStringValue(valueStr string) (gosnmp.Asn1BER, any, error) {
	value, err := decodeHexOctets(valueStr)
	if err != nil {
		return 0, nil, err
	}

	return gosnmp.OctetString, value, nil
}

func decodeHexOctets(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(strings.TrimPrefix(value, "0x"), "0X")
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "\t", "")

	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hexadecimal octets: %w", err)
	}

	return decoded, nil
}

// ExportToWalkFile exports MIB entries to a walk file format.
func ExportToWalkFile(filename string, mib *MIB) error {
	// SECURITY FIX #163: Create file with restricted permissions (owner-only)
	file, err := os.OpenFile(filepath.Clean(filename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToCreateWalkFile, err)
	}

	defer func() { _ = file.Close() }()

	writer := bufio.NewWriter(file)

	defer func() { _ = writer.Flush() }()

	// Get all OIDs in sorted order
	oids := mib.AllOIDs()

	for _, oid := range oids {
		value := mib.Get(oid)
		if value == nil {
			continue
		}

		// Format as OID = TYPE: VALUE
		line := formatWalkEntry(oid, value)

		_, writeErr := writer.WriteString(line + "\n")
		if writeErr != nil {
			return fmt.Errorf("%w: %w", ErrFailedToWriteEntry, writeErr)
		}
	}

	return nil
}

// formatWalkEntry formats a MIB entry as a walk file line.
func formatWalkEntry(oid string, value *OIDValue) string {
	// Ensure OID has leading dot
	if !strings.HasPrefix(oid, ".") {
		oid = "." + oid
	}

	// Format type name
	typeName := formatTypeName(value.Type)

	// Format value
	valueStr := formatValue(value.Type, value.Value)

	return fmt.Sprintf("%s = %s: %s", oid, typeName, valueStr)
}

// formatTypeName returns the walk file type name for an ASN.1 type.
func formatTypeName(asnType gosnmp.Asn1BER) string {
	switch asnType {
	case gosnmp.OctetString:
		return snmpTypeSTRING
	case gosnmp.Integer:
		return snmpTypeINTEGER
	case gosnmp.Gauge32:
		return "Gauge32"
	case gosnmp.Counter32:
		return "Counter32"
	case gosnmp.Counter64:
		return "Counter64"
	case gosnmp.TimeTicks:
		return "Timeticks"
	case gosnmp.ObjectIdentifier:
		return snmpTypeOID
	case gosnmp.IPAddress:
		return "IpAddress"
	case gosnmp.Opaque:
		return "Opaque"
	case gosnmp.Null:
		return "NULL"
	case gosnmp.EndOfContents, // Also covers UnknownType (same value)
		gosnmp.Boolean,
		gosnmp.BitString,
		gosnmp.ObjectDescription,
		gosnmp.NsapAddress,
		gosnmp.Uinteger32,
		gosnmp.OpaqueFloat,
		gosnmp.OpaqueDouble,
		gosnmp.NoSuchObject,
		gosnmp.NoSuchInstance,
		gosnmp.EndOfMibView:
		return snmpTypeUnknown
	}

	// Unreachable - all cases handled above
	return snmpTypeUnknown
}

// formatValue formats a value for walk file output.
func formatValue(asnType gosnmp.Asn1BER, value any) string {
	switch asnType {
	case gosnmp.OctetString:
		return fmt.Sprintf("\"%v\"", value)
	case gosnmp.Integer, gosnmp.Gauge32, gosnmp.Counter32, gosnmp.Counter64:
		return fmt.Sprintf("%v", value)
	case gosnmp.TimeTicks:
		return fmt.Sprintf("(%v)", value)
	case gosnmp.ObjectIdentifier:
		oid := fmt.Sprintf("%v", value)
		if !strings.HasPrefix(oid, ".") {
			oid = "." + oid
		}

		return oid
	case gosnmp.IPAddress:
		return fmt.Sprintf("%v", value)
	case gosnmp.Null:
		return ""
	case gosnmp.EndOfContents, // Also covers UnknownType (same value)
		gosnmp.Boolean,
		gosnmp.BitString,
		gosnmp.ObjectDescription,
		gosnmp.Opaque,
		gosnmp.NsapAddress,
		gosnmp.Uinteger32,
		gosnmp.OpaqueFloat,
		gosnmp.OpaqueDouble,
		gosnmp.NoSuchObject,
		gosnmp.NoSuchInstance,
		gosnmp.EndOfMibView:
		return fmt.Sprintf("%v", value)
	}

	// Unreachable - all cases handled above
	return fmt.Sprintf("%v", value)
}
