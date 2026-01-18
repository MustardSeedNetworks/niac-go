package snmp

import (
	"bufio"
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
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		entry, parseErr := parseWalkLine(line)
		if parseErr != nil {
			// Log error but continue parsing
			_, _ = fmt.Fprintf(os.Stdout, "Warning: line %d: %v\n", lineNum, parseErr)

			continue
		}

		entries = append(entries, *entry)
	}

	scanErr := scanner.Err()
	if scanErr != nil {
		return nil, fmt.Errorf("error reading walk file: %w", scanErr)
	}

	return entries, nil
}

// parseWalkLine parses a single line from a walk file.
func parseWalkLine(line string) (*WalkEntry, error) {
	// Match pattern: OID = TYPE: VALUE
	// Example: .1.3.6.1.2.1.1.1.0 = STRING: "Cisco IOS"
	parts := strings.SplitN(line, "=", OIDPartsMinPDU)
	if len(parts) != OIDPartsMinPDU {
		return nil, ErrInvalidWalkFormat
	}

	oid := strings.TrimSpace(parts[0])
	rest := strings.TrimSpace(parts[1])

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
		return gosnmp.OctetString, strings.TrimPrefix(valueStr, "0x"), nil
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
	value, err := strconv.ParseInt(valueStr, 10, 32)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to parse integer: %w", err)
	}

	return gosnmp.Integer, int(value), nil
}

func parseGaugeValue(valueStr string) (gosnmp.Asn1BER, any, error) {
	value, err := strconv.ParseUint(valueStr, 10, 32)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to parse gauge: %w", err)
	}

	return gosnmp.Gauge32, uint(value), nil
}

func parseCounter32Value(valueStr string) (gosnmp.Asn1BER, any, error) {
	value, err := strconv.ParseUint(valueStr, 10, 32)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to parse counter32: %w", err)
	}

	return gosnmp.Counter32, uint(value), nil
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
	value := strings.ReplaceAll(valueStr, " ", "")
	value = strings.TrimPrefix(value, "0x")

	return gosnmp.OctetString, value, nil
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
