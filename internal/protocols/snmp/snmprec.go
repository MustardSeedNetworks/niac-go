package snmp

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

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

// bitsPerOctet is declared in mib_if.go; decimalBase names the base
// strconv.FormatUint renders the recovered number in.
const decimalBase = 10

// snmprecSniffWindow is how far into a file the format check reads. It needs
// only the first line that is neither blank nor a comment, and only that
// line's leading fields, so a few KiB is ample even where the first value is
// enormous.
const snmprecSniffWindow = 8192

// snmprecOctetStringTag and snmprecOpaqueTag are the tags whose `x` form
// carries raw bytes; every other tag's hex form is the big-endian encoding of
// a number. snmprecIPAddressTag is the exception to both: four bytes that must
// read back as a dotted quad.
const (
	snmprecOctetStringTag = "4"
	snmprecIPAddressTag   = "64"
	snmprecOpaqueTag      = "68"
)

// snmprecTypeName maps a recorded ASN.1 tag onto the token parseTypeAndValue
// takes, so both readers share one set of value parsers instead of growing a
// second. It switches on the recorded string rather than gosnmp.Asn1BER
// because parseTypeAndValue does the same, and because the tag reaches us as
// text anyway.
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
	if _, known := snmprecTypeName(strings.TrimSuffix(fields[1], "x")); !known {
		return false
	}

	return isNumericOIDCandidate(fields[0])
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

		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
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

// snmprecTypeAndValue resolves a recorded tag and value through the same
// parsers the net-snmp path uses, so both readers produce identical Go types
// for a given ASN.1 type and everything downstream stays single-path.
func snmprecTypeAndValue(tag, value string) (gosnmp.Asn1BER, any, error) {
	base := strings.TrimSuffix(tag, "x")

	name, known := snmprecTypeName(base)
	if !known {
		return 0, nil, fmt.Errorf("%w: unknown type tag %q", ErrInvalidSnmprecLine, tag)
	}

	if !strings.HasSuffix(tag, "x") {
		return parseTypeAndValue(name, value)
	}

	// decodeHexOctets, reached through the HEX-STRING branch, already accepts
	// the contiguous form a recording writes.
	if base == snmprecOctetStringTag || base == snmprecOpaqueTag {
		return parseTypeAndValue("HEX-STRING", value)
	}

	if base == snmprecIPAddressTag {
		dotted, err := snmprecHexToDotted(value)
		if err != nil {
			return 0, nil, err
		}

		return parseTypeAndValue(name, dotted)
	}

	decimal, err := snmprecHexToDecimal(value)
	if err != nil {
		return 0, nil, err
	}

	return parseTypeAndValue(name, decimal)
}

// snmprecHexToDotted renders a hex-encoded IpAddress as the dotted quad the
// shared parser stores, rather than the integer the generic numeric path
// would produce.
func snmprecHexToDotted(value string) (string, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("invalid hex address: %w", err)
	}

	octets := make([]string, len(raw))
	for i, octet := range raw {
		octets[i] = strconv.Itoa(int(octet))
	}

	return strings.Join(octets, "."), nil
}

// snmprecHexToDecimal renders a hex-encoded numeric value as the decimal
// string the shared value parsers expect.
func snmprecHexToDecimal(value string) (string, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("invalid hex value: %w", err)
	}

	var number uint64
	for _, octet := range raw {
		number = number<<bitsPerOctet | uint64(octet)
	}

	return strconv.FormatUint(number, decimalBase), nil
}
