package snmp

import (
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

// Null represents an SNMP Null value (distinct from Go nil).
type Null struct{}

// AddMib applies an AddMib directive to the agent.
func (a *Agent) AddMib(oid, typeStr, valueStr string) error {
	if a == nil || a.mib == nil {
		return ErrAgentNotInitialized
	}

	typeStr = strings.TrimSpace(typeStr)
	valueStr = strings.TrimSpace(valueStr)
	valueStr = strings.ReplaceAll(valueStr, "[", "(")
	valueStr = strings.ReplaceAll(valueStr, "]", ")")

	handler := strings.ToLower(valueStr)

	switch {
	case strings.HasPrefix(handler, "varimib"):
		return a.addVariMib(oid, typeStr, valueStr)
	case strings.HasPrefix(handler, "sysuptime"):
		a.mib.SetDynamic(oid, func() *OIDValue {
			return &OIDValue{
				Type:  gosnmp.TimeTicks,
				Value: a.sysUpTimeTicks(),
			}
		})

		return nil
	default:
		// fallthrough to fixed parsing
	}

	return a.addFixedMib(oid, typeStr, valueStr)
}

func (a *Agent) addFixedMib(oid, typeStr, valueStr string) error {
	asnType, err := asnTypeFromString(typeStr)
	if err != nil {
		return err
	}

	value := extractHandlerValue(valueStr)

	parsed, err := parseValue(typeStr, asnType, value)
	if err != nil {
		return err
	}

	a.mib.Set(oid, &OIDValue{Type: asnType, Value: parsed})

	return nil
}

func (a *Agent) addVariMib(oid, typeStr, valueStr string) error {
	asnType, err := asnTypeFromString(typeStr)
	if err != nil {
		return err
	}

	pairs, err := parseVarimibPairs(valueStr)
	if err != nil || len(pairs) == 0 {
		return ErrInvalidVarimibValue
	}

	totalInterval := 0
	for _, p := range pairs {
		totalInterval += p.interval
	}

	if totalInterval <= 0 {
		return ErrInvalidVarimibIntervals
	}

	if isStringVarimibType(asnType) {
		a.setStringVarimib(oid, typeStr, asnType, pairs, totalInterval)
		return nil
	}

	a.setIntegralVarimib(oid, asnType, pairs, totalInterval)
	return nil
}

// isStringVarimibType returns true if the ASN type represents a string/IP variant.
func isStringVarimibType(asnType gosnmp.Asn1BER) bool {
	switch asnType {
	case gosnmp.OctetString, gosnmp.IPAddress, gosnmp.ObjectIdentifier, gosnmp.Opaque:
		return true
	case gosnmp.EndOfContents, gosnmp.Boolean, gosnmp.Integer,
		gosnmp.BitString, gosnmp.Null, gosnmp.ObjectDescription, gosnmp.Counter32,
		gosnmp.Gauge32, gosnmp.TimeTicks, gosnmp.NsapAddress, gosnmp.Counter64,
		gosnmp.Uinteger32, gosnmp.OpaqueFloat, gosnmp.OpaqueDouble, gosnmp.NoSuchObject,
		gosnmp.NoSuchInstance, gosnmp.EndOfMibView:
		// Note: EndOfContents and UnknownType have the same value (0), so only listing one
		return false
	}

	return false
}

// setStringVarimib sets up a dynamic MIB handler for string/IP type varimib values.
func (a *Agent) setStringVarimib(oid, typeStr string, asnType gosnmp.Asn1BER, pairs []varimibPair, totalInterval int) {
	a.mib.SetDynamic(oid, func() *OIDValue {
		value := computeStringVarimibValue(a.sysUpTimeTicks(), pairs, totalInterval)
		parsed, _ := parseValue(typeStr, asnType, value)
		return &OIDValue{Type: asnType, Value: parsed}
	})
}

// computeStringVarimibValue determines the current string value based on elapsed ticks.
func computeStringVarimibValue(ticks uint32, pairs []varimibPair, totalInterval int) string {
	mod := int(ticks) % totalInterval
	value := pairs[len(pairs)-1].value

	for _, p := range pairs {
		if mod < p.interval {
			break
		}
		value = p.value
		mod -= p.interval
	}

	return value
}

// setIntegralVarimib sets up a dynamic MIB handler for integral type varimib values.
func (a *Agent) setIntegralVarimib(oid string, asnType gosnmp.Asn1BER, pairs []varimibPair, totalInterval int) {
	totalValue := computeTotalVarimibValue(pairs)

	a.mib.SetDynamic(oid, func() *OIDValue {
		nowValue := computeIntegralVarimibValue(a.sysUpTimeTicks(), pairs, totalInterval, totalValue)
		return &OIDValue{Type: asnType, Value: castIntegral(asnType, nowValue)}
	})
}

// computeTotalVarimibValue sums up all delta values from varimib pairs.
func computeTotalVarimibValue(pairs []varimibPair) int64 {
	var total int64
	for _, p := range pairs {
		inc, _ := strconv.ParseInt(p.value, 10, 64)
		total += inc
	}
	return total
}

// computeIntegralVarimibValue calculates the current integral value based on elapsed ticks.
func computeIntegralVarimibValue(ticks uint32, pairs []varimibPair, totalInterval int, totalValue int64) int64 {
	ticksInt := int(ticks)
	multiple := ticksInt / totalInterval
	leftOver := ticksInt % totalInterval
	nowValue := totalValue * int64(multiple)

	for _, p := range pairs {
		inc, _ := strconv.ParseInt(p.value, 10, 64)
		if p.interval >= leftOver {
			nowValue += inc * int64(leftOver) / int64(p.interval)
			break
		}
		nowValue += inc
		leftOver -= p.interval
	}

	return nowValue
}

type varimibPair struct {
	interval int
	value    string
}

func parseVarimibPairs(valueStr string) ([]varimibPair, error) {
	valueStr = strings.TrimSpace(valueStr)
	re := regexp.MustCompile(`\(([^()]*)\)`)

	matches := re.FindAllStringSubmatch(valueStr, -1)
	if len(matches) == 0 {
		return nil, ErrNoVarimibPairs
	}

	pairs := make([]varimibPair, 0, len(matches))

	for _, match := range matches {
		fields := strings.Fields(match[1])
		if len(fields) < OIDPartsMinPDU {
			continue
		}

		interval, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		value := strings.Join(fields[1:], " ")
		pairs = append(pairs, varimibPair{interval: interval, value: value})
	}

	return pairs, nil
}

func extractHandlerValue(valueStr string) string {
	valueStr = strings.TrimSpace(valueStr)
	if idx := strings.IndexRune(valueStr, '('); idx != -1 {
		end := strings.LastIndex(valueStr, ")")
		if end > idx {
			return strings.TrimSpace(valueStr[idx+1 : end])
		}
	}

	return strings.TrimSpace(valueStr)
}

func asnTypeFromString(typeStr string) (gosnmp.Asn1BER, error) {
	switch strings.ToLower(strings.TrimSpace(typeStr)) {
	case "string":
		return gosnmp.OctetString, nil
	case "counter32":
		return gosnmp.Counter32, nil
	case "gauge32":
		return gosnmp.Gauge32, nil
	case "integer":
		return gosnmp.Integer, nil
	case "ipaddress", "networkaddress":
		return gosnmp.IPAddress, nil
	case "oid":
		return gosnmp.ObjectIdentifier, nil
	case "opaque":
		return gosnmp.Opaque, nil
	case "hex-string":
		return gosnmp.OctetString, nil
	case "timeticks":
		return gosnmp.TimeTicks, nil
	case "counter64":
		return gosnmp.Counter64, nil
	case "null":
		return gosnmp.Null, nil
	default:
		return gosnmp.OctetString, fmt.Errorf("%w: %s", ErrUnsupportedSNMPType, typeStr)
	}
}

func parseValue(typeStr string, asnType gosnmp.Asn1BER, value string) (any, error) {
	//exhaustive:ignore
	switch asnType {
	case gosnmp.OctetString:
		if strings.EqualFold(typeStr, "Hex-STRING") {
			clean := strings.ReplaceAll(value, " ", "")

			bytes, err := hex.DecodeString(clean)
			if err != nil {
				return nil, fmt.Errorf("failed to decode hex string: %w", err)
			}

			return bytes, nil
		}

		return []byte(value), nil
	case gosnmp.IPAddress:
		ip := net.ParseIP(value)
		if ip == nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidIPAddress, value)
		}

		return ip.To4(), nil
	case gosnmp.ObjectIdentifier:
		return value, nil
	case gosnmp.Opaque:
		return []byte(value), nil
	case gosnmp.Integer:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse integer: %w", err)
		}

		return int(i), nil
	case gosnmp.Counter32, gosnmp.Gauge32, gosnmp.TimeTicks:
		u, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse uint32: %w", err)
		}

		return uint32(u), nil
	case gosnmp.Counter64:
		u, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse uint64: %w", err)
		}

		return u, nil
	case gosnmp.Null:
		return Null{}, nil
	default:
		return []byte(value), nil
	}
}

func castIntegral(asnType gosnmp.Asn1BER, value int64) any {
	switch asnType {
	case gosnmp.Counter32, gosnmp.Gauge32, gosnmp.TimeTicks:
		return safeUint32(value)
	case gosnmp.Counter64:
		return safeUint64(value)
	case gosnmp.EndOfContents, // Also covers UnknownType (same value)
		gosnmp.Boolean,
		gosnmp.Integer,
		gosnmp.BitString,
		gosnmp.OctetString,
		gosnmp.Null,
		gosnmp.ObjectIdentifier,
		gosnmp.ObjectDescription,
		gosnmp.IPAddress,
		gosnmp.Opaque,
		gosnmp.NsapAddress,
		gosnmp.Uinteger32,
		gosnmp.OpaqueFloat,
		gosnmp.OpaqueDouble,
		gosnmp.NoSuchObject,
		gosnmp.NoSuchInstance,
		gosnmp.EndOfMibView:
		return int(value)
	}

	return int(value)
}

func (a *Agent) sysUpTimeTicks() uint32 {
	uptime := time.Since(a.startTime)
	ms := uptime.Milliseconds() / MillisecsPerCentisec

	return safeUint32(ms)
}
