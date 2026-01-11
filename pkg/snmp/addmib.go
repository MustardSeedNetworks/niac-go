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

// AddMib applies an AddMib directive to the agent.
func (a *Agent) AddMib(oid, typeStr, valueStr string) error {
	if a == nil || a.mib == nil {
		return fmt.Errorf("agent not initialized")
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
	case strings.HasPrefix(handler, "fixed"):
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
		return fmt.Errorf("invalid varimib value")
	}

	totalInterval := 0
	for _, p := range pairs {
		totalInterval += p.interval
	}
	if totalInterval <= 0 {
		return fmt.Errorf("invalid varimib intervals")
	}

	switch asnType {
	case gosnmp.OctetString, gosnmp.IPAddress, gosnmp.ObjectIdentifier, gosnmp.Opaque:
		// String/IP variants
		a.mib.SetDynamic(oid, func() *OIDValue {
			ticks := int(a.sysUpTimeTicks())
			mod := ticks % totalInterval
			value := pairs[len(pairs)-1].value
			for _, p := range pairs {
				if mod < p.interval {
					break
				}
				value = p.value
				mod -= p.interval
			}
			parsed, _ := parseValue(typeStr, asnType, value)
			return &OIDValue{Type: asnType, Value: parsed}
		})
		return nil
	default:
		// Integral variants
		totalValue := int64(0)
		for _, p := range pairs {
			inc, _ := strconv.ParseInt(p.value, 10, 64)
			totalValue += inc
		}
		a.mib.SetDynamic(oid, func() *OIDValue {
			ticks := int(a.sysUpTimeTicks())
			multiple := ticks / totalInterval
			leftOver := ticks % totalInterval
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
			return &OIDValue{Type: asnType, Value: castIntegral(asnType, nowValue)}
		})
		return nil
	}
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
		return nil, fmt.Errorf("no varimib pairs")
	}

	pairs := make([]varimibPair, 0, len(matches))
	for _, match := range matches {
		fields := strings.Fields(match[1])
		if len(fields) < 2 {
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
		return gosnmp.OctetString, fmt.Errorf("unsupported SNMP type: %s", typeStr)
	}
}

func parseValue(typeStr string, asnType gosnmp.Asn1BER, value string) (interface{}, error) {
	switch asnType {
	case gosnmp.OctetString:
		if strings.EqualFold(typeStr, "Hex-STRING") {
			clean := strings.ReplaceAll(value, " ", "")
			bytes, err := hex.DecodeString(clean)
			if err != nil {
				return nil, err
			}
			return bytes, nil
		}
		return []byte(value), nil
	case gosnmp.IPAddress:
		ip := net.ParseIP(value)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP address %s", value)
		}
		return ip.To4(), nil
	case gosnmp.ObjectIdentifier:
		return value, nil
	case gosnmp.Opaque:
		return []byte(value), nil
	case gosnmp.Integer:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}
		return int(i), nil
	case gosnmp.Counter32, gosnmp.Gauge32, gosnmp.TimeTicks:
		u, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return nil, err
		}
		return uint32(u), nil
	case gosnmp.Counter64:
		u, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, err
		}
		return uint64(u), nil
	case gosnmp.Null:
		return nil, nil
	default:
		return []byte(value), nil
	}
}

func castIntegral(asnType gosnmp.Asn1BER, value int64) interface{} {
	switch asnType {
	case gosnmp.Counter32, gosnmp.Gauge32, gosnmp.TimeTicks:
		if value < 0 {
			value = 0
		}
		return uint32(value)
	case gosnmp.Counter64:
		if value < 0 {
			value = 0
		}
		return uint64(value)
	default:
		return int(value)
	}
}

func (a *Agent) sysUpTimeTicks() uint32 {
	uptime := time.Since(a.startTime)
	ms := uptime.Milliseconds() / 10
	if ms > 0xFFFFFFFF {
		ms = 0xFFFFFFFF
	}
	return uint32(ms)
}
