package snmp

import (
	"encoding/hex"
	"fmt"
	"math"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

func validWireOID(oid string) bool {
	if !isNumericOIDCandidate(oid) {
		return false
	}
	arcs := strings.Split(strings.TrimPrefix(oid, "."), ".")
	if len(arcs) < 2 || len(arcs) > 128 {
		return false
	}
	var root uint64
	for i, arc := range arcs {
		value, err := strconv.ParseUint(arc, 10, 32)
		if err != nil || i == 0 && value > 2 || i == 1 && root != 2 && value > 39 {
			return false
		}
		if i == 0 {
			root = value
		}
	}
	return true
}

func snmprecTypeAndValue(tag, value string) (gosnmp.Asn1BER, any, error) {
	base := snmprecBaseTag(tag)
	name, known := snmprecTypeName(base)
	if !known {
		return 0, nil, fmt.Errorf("%w: unknown type tag %q", ErrInvalidSnmprecLine, tag)
	}
	if tag == base {
		return snmprecPlainValue(base, name, value)
	}
	raw, err := decodeSnmprecValue(tag, value)
	if err != nil {
		return 0, nil, err
	}
	switch base {
	case snmprecOctetStringTag:
		return gosnmp.OctetString, raw, nil
	case snmprecOpaqueTag:
		return gosnmp.Opaque, raw, nil
	case snmprecIPAddressTag:
		if len(raw) != net.IPv4len {
			return 0, nil, fmt.Errorf("%w: IpAddress requires four octets", ErrInvalidSnmprecLine)
		}
		return gosnmp.IPAddress, netip.AddrFrom4([4]byte(raw)).String(), nil
	default:
		if strings.HasSuffix(tag, "e") {
			return 0, nil, fmt.Errorf("%w: escaped encoding requires octets", ErrInvalidSnmprecLine)
		}
		number, parseErr := snmprecHexNumber(base, raw)
		if parseErr != nil {
			return 0, nil, parseErr
		}
		return snmprecPlainValue(base, name, number)
	}
}

func snmprecPlainValue(base, name, value string) (gosnmp.Asn1BER, any, error) {
	switch base {
	case "6":
		if !validWireOID(value) {
			return 0, nil, fmt.Errorf("%w: invalid recorded OID value", ErrInvalidSnmprecLine)
		}
		return gosnmp.ObjectIdentifier, strings.TrimPrefix(value, "."), nil
	case snmprecOctetStringTag:
		return gosnmp.OctetString, value, nil
	case snmprecOpaqueTag:
		return gosnmp.Opaque, []byte(value), nil
	case snmprecIPAddressTag:
		return parseIPAddressValue(value)
	case "5":
		if value != "" {
			return 0, nil, fmt.Errorf("%w: NULL has a value", ErrInvalidSnmprecLine)
		}
	default:
		if err := validateSnmprecNumber(base, value); err != nil {
			return 0, nil, err
		}
	}
	return parseTypeAndValue(name, value)
}

func decodeSnmprecValue(tag, value string) ([]byte, error) {
	if strings.HasSuffix(tag, "x") {
		return hex.DecodeString(value)
	}
	var decoded []byte
	for len(value) > 0 {
		if value[0] != '\\' {
			decoded = append(decoded, value[0])
			value = value[1:]
			continue
		}
		if len(value) < 2 || !strings.ContainsRune(`\'"abfnrtvx`, rune(value[1])) {
			return nil, fmt.Errorf("%w: invalid escaped octet", ErrInvalidSnmprecLine)
		}
		quote := byte('"')
		if value[1] == '\'' {
			quote = '\''
		}
		char, _, tail, err := strconv.UnquoteChar(value, quote)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidSnmprecLine, err)
		}
		if char < 0 || char > math.MaxUint8 {
			return nil, fmt.Errorf("%w: escape exceeds one octet", ErrInvalidSnmprecLine)
		}
		decoded = append(decoded, byte(char))
		value = tail
	}
	return decoded, nil
}

func snmprecHexNumber(tag string, raw []byte) (string, error) {
	maxBytes := 4
	if tag == "70" {
		maxBytes = 8
	}
	if len(raw) == 0 || len(raw) > maxBytes {
		return "", fmt.Errorf("%w: invalid numeric octet count", ErrInvalidSnmprecLine)
	}
	var number uint64
	for _, octet := range raw {
		number = number<<bitsPerOctet | uint64(octet)
	}
	if tag == "2" {
		signed := int64(number)
		if raw[0]&0x80 != 0 {
			signed -= 1 << uint(len(raw)*bitsPerOctet)
		}
		return strconv.FormatInt(signed, 10), nil
	}
	return strconv.FormatUint(number, 10), nil
}

func validateSnmprecNumber(tag, value string) error {
	var err error
	if tag == "2" {
		_, err = strconv.ParseInt(value, 10, 32)
	} else {
		bits := 32
		if tag == "70" {
			bits = 64
		}
		_, err = strconv.ParseUint(value, 10, bits)
	}
	if err != nil {
		return fmt.Errorf("%w: invalid numeric value: %w", ErrInvalidSnmprecLine, err)
	}
	return nil
}

func snmprecBaseTag(tag string) string {
	if strings.HasSuffix(tag, "x") || strings.HasSuffix(tag, "e") {
		return tag[:len(tag)-1]
	}
	return tag
}
