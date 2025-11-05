package snmp

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

// WalkEntry represents a single entry from an SNMP walk file
type WalkEntry struct {
	OID   string
	Type  gosnmp.Asn1BER
	Value interface{}
}

// ParseWalkFile parses an SNMP walk file
// Walk files are typically in the format:
// OID = TYPE: VALUE
// For example:
// .1.3.6.1.2.1.1.1.0 = STRING: "Cisco IOS Software"
// .1.3.6.1.2.1.1.3.0 = Timeticks: (12345) 0:02:03.45
func ParseWalkFile(filename string) ([]WalkEntry, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open walk file: %v", err)
	}
	defer file.Close()

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

		entry, err := parseWalkLine(line)
		if err != nil {
			// Log error but continue parsing
			fmt.Printf("Warning: line %d: %v\n", lineNum, err)
			continue
		}

		entries = append(entries, *entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading walk file: %v", err)
	}

	return entries, nil
}

// parseWalkLine parses a single line from a walk file
func parseWalkLine(line string) (*WalkEntry, error) {
	// Match pattern: OID = TYPE: VALUE
	// Example: .1.3.6.1.2.1.1.1.0 = STRING: "Cisco IOS"
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid format: missing '='")
	}

	oid := strings.TrimSpace(parts[0])
	rest := strings.TrimSpace(parts[1])

	// Parse TYPE: VALUE
	typeParts := strings.SplitN(rest, ":", 2)
	if len(typeParts) != 2 {
		return nil, fmt.Errorf("invalid format: missing ':'")
	}

	typeStr := strings.ToUpper(strings.TrimSpace(typeParts[0]))
	valueStr := strings.TrimSpace(typeParts[1])

	// Determine SNMP type and parse value
	asnType, value, err := parseTypeAndValue(typeStr, valueStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse value: %v", err)
	}

	return &WalkEntry{
		OID:   oid,
		Type:  asnType,
		Value: value,
	}, nil
}

// parseTypeAndValue parses the type and value from a walk file entry
func parseTypeAndValue(typeStr, valueStr string) (gosnmp.Asn1BER, interface{}, error) {
	switch typeStr {
	case "STRING", "OCTET STRING":
		// Remove quotes if present
		value := strings.Trim(valueStr, "\"")
		return gosnmp.OctetString, value, nil

	case "INTEGER", "INT":
		value, err := strconv.ParseInt(valueStr, 10, 32)
		if err != nil {
			return 0, nil, err
		}
		return gosnmp.Integer, int(value), nil

	case "GAUGE", "GAUGE32":
		value, err := strconv.ParseUint(valueStr, 10, 32)
		if err != nil {
			return 0, nil, err
		}
		return gosnmp.Gauge32, uint(value), nil

	case "COUNTER", "COUNTER32":
		value, err := strconv.ParseUint(valueStr, 10, 32)
		if err != nil {
			return 0, nil, err
		}
		return gosnmp.Counter32, uint(value), nil

	case "COUNTER64":
		value, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			return 0, nil, err
		}
		return gosnmp.Counter64, value, nil

	case "TIMETICKS":
		// Parse format: (12345) or just 12345
		re := regexp.MustCompile(`\((\d+)\)|(\d+)`)
		matches := re.FindStringSubmatch(valueStr)
		if len(matches) == 0 {
			return 0, nil, fmt.Errorf("invalid Timeticks format: %s", valueStr)
		}
		var numStr string
		if matches[1] != "" {
			numStr = matches[1]
		} else {
			numStr = matches[2]
		}
		value, err := strconv.ParseUint(numStr, 10, 32)
		if err != nil {
			return 0, nil, err
		}
		return gosnmp.TimeTicks, uint32(value), nil

	case "OID", "OBJECT IDENTIFIER":
		// Remove leading dot if present for consistency
		value := strings.TrimPrefix(valueStr, ".")
		return gosnmp.ObjectIdentifier, value, nil

	case "IPADDRESS", "IP ADDRESS", "IPADDR":
		// Parse IP address
		return gosnmp.IPAddress, valueStr, nil

	case "BITS":
		// BITS type - store as hex string
		value := strings.TrimPrefix(valueStr, "0x")
		return gosnmp.OctetString, value, nil

	case "HEX-STRING", "HEX":
		// Hex string - parse to bytes
		value := strings.ReplaceAll(valueStr, " ", "")
		value = strings.TrimPrefix(value, "0x")
		return gosnmp.OctetString, value, nil

	case "OPAQUE":
		return gosnmp.Opaque, valueStr, nil

	case "NULL":
		return gosnmp.Null, nil, nil

	default:
		// Unknown type - treat as string
		return gosnmp.OctetString, valueStr, nil
	}
}

// ExportToWalkFile exports MIB entries to a walk file format
func ExportToWalkFile(filename string, mib *MIB) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create walk file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Get all OIDs in sorted order
	oids := mib.AllOIDs()

	for _, oid := range oids {
		value := mib.Get(oid)
		if value == nil {
			continue
		}

		// Format as OID = TYPE: VALUE
		line := formatWalkEntry(oid, value)
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return fmt.Errorf("failed to write entry: %v", err)
		}
	}

	return nil
}

// formatWalkEntry formats a MIB entry as a walk file line
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

// formatTypeName returns the walk file type name for an ASN.1 type
func formatTypeName(asnType gosnmp.Asn1BER) string {
	switch asnType {
	case gosnmp.OctetString:
		return "STRING"
	case gosnmp.Integer:
		return "INTEGER"
	case gosnmp.Gauge32:
		return "Gauge32"
	case gosnmp.Counter32:
		return "Counter32"
	case gosnmp.Counter64:
		return "Counter64"
	case gosnmp.TimeTicks:
		return "Timeticks"
	case gosnmp.ObjectIdentifier:
		return "OID"
	case gosnmp.IPAddress:
		return "IpAddress"
	case gosnmp.Opaque:
		return "Opaque"
	case gosnmp.Null:
		return "NULL"
	default:
		return "Unknown"
	}
}

// formatValue formats a value for walk file output
func formatValue(asnType gosnmp.Asn1BER, value interface{}) string {
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
	default:
		return fmt.Sprintf("%v", value)
	}
}
