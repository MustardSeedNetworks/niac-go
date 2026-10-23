package snmp

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// snmpsim records one varbind per line as `OID|tag|value`, where tag is the
// ASN.1 type number and a trailing `x` marks a hex-encoded value. A recording
// therefore needs no type inference at all, unlike a net-snmp walk where the
// printed name has to be mapped back and long octet strings arrive split
// across continuation lines.
const snmprecFieldSeparator = "|"

// snmprecFieldCount is the OID, tag and value a recording row carries.
const snmprecFieldCount = 3

// snmprecSniffWindow is how far into a file the format check reads. It needs
// only the first line that is neither blank nor a comment, and only that
// line's leading fields, so a few KiB is ample even where the first value is
// enormous.
const snmprecSniffWindow = 8192

// OctetString and Opaque encodings preserve raw bytes. IpAddress encodes
// four octets, while numeric tags carry big-endian integers.
const (
	snmprecOctetStringTag = "4"
	snmprecIPAddressTag   = "64"
	snmprecOpaqueTag      = "68"
)

// Numeric tags use the text reader's value types after recording-specific
// range validation; octet encodings are decoded directly to preserve bytes.
func snmprecTypeName(tag string) (string, bool) {
	switch tag {
	case "2":
		return snmpTypeINTEGER, true
	case snmprecOctetStringTag:
		return snmpTypeSTRING, true
	case "5":
		return "NULL", true
	case "6":
		return snmpTypeOID, true
	case snmprecIPAddressTag:
		return "IPADDRESS", true
	case "65":
		return "COUNTER32", true
	case "66":
		return "GAUGE32", true
	case "67":
		return "TIMETICKS", true
	case snmprecOpaqueTag:
		return "OPAQUE", true
	case "70":
		return "COUNTER64", true
	case "71":
		return "UINTEGER32", true
	default:
		return "", false
	}
}

// looksLikeSnmprec reports whether a line is a recording row. A net-snmp line
// carries " = " before its type and an OID never contains "|", so the two
// formats cannot be confused for one another.
func looksLikeSnmprec(line string) bool {
	fields := strings.SplitN(line, snmprecFieldSeparator, snmprecFieldCount)
	if len(fields) != snmprecFieldCount {
		return false
	}
	if fields[1] == "" {
		return false
	}

	return isNumericOIDCandidate(strings.TrimSpace(fields[0]))
}

func isNumericOIDCandidate(oid string) bool {
	trimmed := strings.TrimPrefix(oid, ".")
	if trimmed == "" {
		return false
	}
	for _, char := range trimmed {
		if (char < '0' || char > '9') && char != '.' {
			return false
		}
	}

	return true
}

// contentIsSnmprec decides the format from the first meaningful line without
// consuming it. The file name is not consulted: a caller may hold either
// format under any extension, and a recording's leading `#` comments must not
// decide it either.
func contentIsSnmprec(reader *bufio.Reader) bool {
	peeked, err := reader.Peek(snmprecSniffWindow)
	if len(peeked) == 0 && err != nil {
		return false
	}

	for line := range strings.SplitSeq(string(peeked), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		return looksLikeSnmprec(trimmed)
	}

	return false
}

func parseSnmprec(reader *bufio.Reader) ([]WalkEntry, error) {
	var entries []WalkEntry

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, walkScanBufInitial), walkScanBufMax)

	lineNum := 0
	for scanner.Scan() {
		lineNum++

		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		entry, err := parseSnmprecLine(line)
		if err != nil {
			logging.Debugf("Warning: line %d: %v", lineNum, err)

			continue
		}

		entries = append(entries, *entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading recording: %w", err)
	}

	return entries, nil
}

func parseSnmprecLine(line string) (*WalkEntry, error) {
	fields := strings.SplitN(line, snmprecFieldSeparator, snmprecFieldCount)
	if len(fields) != snmprecFieldCount {
		return nil, fmt.Errorf("%w: %s", ErrInvalidSnmprecLine, line)
	}

	fields[0] = strings.TrimSpace(fields[0])
	if !validWireOID(fields[0]) {
		return nil, fmt.Errorf("%w: invalid numeric OID %q", ErrInvalidSnmprecLine, fields[0])
	}

	asn1Type, value, err := snmprecTypeAndValue(fields[1], fields[2])
	if err != nil {
		return nil, err
	}

	return &WalkEntry{
		OID:   strings.TrimPrefix(fields[0], "."),
		Type:  asn1Type,
		Value: value,
	}, nil
}
