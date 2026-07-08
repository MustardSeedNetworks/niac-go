package snmp

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
)

// VariMibType represents the type of varimib handler.
type VariMibType string

const (
	VariMibFixed     VariMibType = "fixed"
	VariMibIntegral  VariMibType = "integral"
	VariMibString    VariMibType = "string"
	VariMibSysUpTime VariMibType = "sysuptime"
)

// VariMibHandler represents a handler for time-varying MIB values.
type VariMibHandler interface {
	// GetValue returns the current value based on elapsed time
	GetValue() any
	// GetType returns the ASN.1 type of the value
	GetType() gosnmp.Asn1BER
}

// VariMibIntegralHandler handles counters that increment/decrement over time
// Format: varimib((centiseconds delta) (centiseconds delta) ...)
// Example: varimib((200 10) (400 -10)) - increment 10 every 2sec, decrement 10 every 4sec.
type VariMibIntegralHandler struct {
	mu        sync.RWMutex
	startTime time.Time
	baseValue int64
	asnType   gosnmp.Asn1BER
	intervals []integralInterval
}

type integralInterval struct {
	centiSeconds int64 // interval in centiseconds (1/100 sec)
	delta        int64 // value change per interval
}

// VariMibStringHandler handles strings that cycle through values over time
// Format: varimib((centiseconds "value1") (centiseconds "value2") ...)
// Example: varimib((42000 10.250.0.3) (42000 10.250.0.2)).
type VariMibStringHandler struct {
	mu        sync.RWMutex
	startTime time.Time
	asnType   gosnmp.Asn1BER
	intervals []stringInterval
}

type stringInterval struct {
	centiSeconds int64
	value        string
}

// VariMibFixedHandler returns a fixed value (no time variation).
type VariMibFixedHandler struct {
	value   any
	asnType gosnmp.Asn1BER
}

// ParseVariMibValue parses a varimib value string and returns the appropriate handler
// Formats supported:
//   - fixed(value) - returns constant value
//   - varimib((centisecs delta) ...) - integral counter
//   - varimib((centisecs "string") ...) - cycling string
func ParseVariMibValue(value string, asnType gosnmp.Asn1BER, baseValue any) (VariMibHandler, error) {
	value = strings.TrimSpace(value)

	// Check for fixed value
	if strings.HasPrefix(value, "fixed(") && strings.HasSuffix(value, ")") {
		inner := value[6 : len(value)-1]

		return NewVariMibFixedHandler(inner, asnType)
	}

	// Check for varimib pattern
	if strings.HasPrefix(value, "varimib") {
		return parseVariMibIntervals(value, asnType, baseValue)
	}

	// Default: treat as fixed value
	return NewVariMibFixedHandler(value, asnType)
}

// parseVariMibIntervals parses the varimib((t1 v1) (t2 v2) ...) format.
func parseVariMibIntervals(value string, asnType gosnmp.Asn1BER, baseValue any) (VariMibHandler, error) {
	value = cleanVariMibValue(value)

	// Try numeric pattern first (integral handler)
	if handler := tryParseIntegralHandler(value, asnType, baseValue); handler != nil {
		return handler, nil
	}

	// Try string pattern (string handler)
	handler, err := tryParseStringHandler(value, asnType)
	if err != nil && !errors.Is(err, ErrNoPatternMatch) {
		return nil, err
	}

	if handler != nil {
		return handler, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrInvalidVarimibFormat, value)
}

// cleanVariMibValue removes the varimib prefix and surrounding brackets.
func cleanVariMibValue(value string) string {
	value = strings.TrimPrefix(value, "varimib")
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	value = strings.TrimPrefix(value, "(")
	value = strings.TrimSuffix(value, ")")
	return value
}

// tryParseIntegralHandler attempts to parse value as an integral varimib handler.
// Returns nil if the value doesn't match the numeric pattern.
func tryParseIntegralHandler(value string, asnType gosnmp.Asn1BER, baseValue any) *VariMibIntegralHandler {
	numericRe := regexp.MustCompile(`\((\d+)\s+(-?\d+)\)`)
	numMatches := numericRe.FindAllStringSubmatch(value, -1)

	if len(numMatches) == 0 || !allMatchesNumeric(numMatches) {
		return nil
	}

	base := extractBaseValue(baseValue)
	intervals := parseIntegralIntervals(numMatches)

	return &VariMibIntegralHandler{
		startTime: time.Now(),
		baseValue: base,
		asnType:   asnType,
		intervals: intervals,
	}
}

// allMatchesNumeric checks if all regex matches contain valid numeric deltas.
func allMatchesNumeric(matches [][]string) bool {
	for _, match := range matches {
		if _, err := strconv.ParseInt(match[2], 10, 64); err != nil {
			return false
		}
	}
	return true
}

// extractBaseValue converts various numeric types to int64.
func extractBaseValue(baseValue any) int64 {
	switch v := baseValue.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		if v <= math.MaxInt64 {
			return int64(v)
		}
	case uint32:
		return int64(v)
	case uint64:
		if v <= math.MaxInt64 {
			return int64(v)
		}
	}
	return 0
}

// parseIntegralIntervals converts regex matches to integralInterval slice.
func parseIntegralIntervals(matches [][]string) []integralInterval {
	intervals := make([]integralInterval, len(matches))
	for i, match := range matches {
		centisecs, _ := strconv.ParseInt(match[1], 10, 64)
		delta, _ := strconv.ParseInt(match[2], 10, 64)
		intervals[i] = integralInterval{centiSeconds: centisecs, delta: delta}
	}
	return intervals
}

// tryParseStringHandler attempts to parse value as a string varimib handler.
// Returns ErrNoPatternMatch if the value doesn't match the string pattern.
func tryParseStringHandler(value string, asnType gosnmp.Asn1BER) (*VariMibStringHandler, error) {
	stringRe := regexp.MustCompile(`\((\d+)\s+([^)]+)\)`)
	strMatches := stringRe.FindAllStringSubmatch(value, -1)

	if len(strMatches) == 0 {
		return nil, ErrNoPatternMatch
	}

	intervals, err := parseStringIntervals(strMatches)
	if err != nil {
		return nil, err
	}

	return &VariMibStringHandler{
		startTime: time.Now(),
		asnType:   asnType,
		intervals: intervals,
	}, nil
}

// parseStringIntervals converts regex matches to stringInterval slice.
func parseStringIntervals(matches [][]string) ([]stringInterval, error) {
	intervals := make([]stringInterval, len(matches))
	for i, match := range matches {
		centisecs, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidCentiseconds, match[1])
		}

		strVal := strings.TrimSpace(match[2])
		strVal = strings.Trim(strVal, "\"'")
		intervals[i] = stringInterval{centiSeconds: centisecs, value: strVal}
	}
	return intervals, nil
}

// NewVariMibFixedHandler creates a fixed value handler.
func NewVariMibFixedHandler(value string, asnType gosnmp.Asn1BER) (*VariMibFixedHandler, error) {
	var parsedValue any

	switch asnType {
	case gosnmp.Integer:
		v, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidInteger, value)
		}

		parsedValue = int(v)

	case gosnmp.Counter32, gosnmp.Gauge32, gosnmp.Uinteger32:
		v, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidUnsignedInt, value)
		}

		parsedValue = uint32(v)

	case gosnmp.Counter64:
		v, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidCounter64, value)
		}

		parsedValue = v

	case gosnmp.TimeTicks:
		v, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidTimeticks, value)
		}

		parsedValue = uint32(v)

	case gosnmp.IPAddress:
		parsedValue = value

	case gosnmp.ObjectIdentifier:
		parsedValue = value

	case gosnmp.OctetString:
		parsedValue = value

	case gosnmp.EndOfContents, // Also covers UnknownType (same value)
		gosnmp.Boolean,
		gosnmp.BitString,
		gosnmp.Null,
		gosnmp.ObjectDescription,
		gosnmp.Opaque,
		gosnmp.NsapAddress,
		gosnmp.OpaqueFloat,
		gosnmp.OpaqueDouble,
		gosnmp.NoSuchObject,
		gosnmp.NoSuchInstance,
		gosnmp.EndOfMibView:
		parsedValue = value
	}

	return &VariMibFixedHandler{
		value:   parsedValue,
		asnType: asnType,
	}, nil
}

// GetValue implements VariMibHandler.
func (h *VariMibFixedHandler) GetValue() any {
	return h.value
}

// GetType implements VariMibHandler.
func (h *VariMibFixedHandler) GetType() gosnmp.Asn1BER {
	return h.asnType
}

// GetValue implements VariMibHandler for integral counters.
func (h *VariMibIntegralHandler) GetValue() any {
	h.mu.RLock()
	defer h.mu.RUnlock()

	elapsed := time.Since(h.startTime)
	centisecs := elapsed.Milliseconds() / CentisecsPerSec // Convert to centiseconds

	value := h.baseValue

	// Calculate accumulated changes from all intervals
	for _, interval := range h.intervals {
		if interval.centiSeconds <= 0 {
			continue
		}
		// Number of complete intervals that have passed
		count := centisecs / interval.centiSeconds
		value += count * interval.delta
	}

	// Ensure non-negative for counter types
	value = max(value, 0)

	// Return appropriate type based on ASN type
	switch h.asnType {
	case gosnmp.Counter32, gosnmp.Gauge32, gosnmp.Uinteger32:
		if value > math.MaxUint32 {
			value %= (math.MaxUint32 + 1) // Wrap around for 32-bit counters
		}

		return safeUint32(value)
	case gosnmp.Counter64:
		return safeUint64(value)
	case gosnmp.TimeTicks:
		return safeUint32(value)
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
		gosnmp.OpaqueFloat,
		gosnmp.OpaqueDouble,
		gosnmp.NoSuchObject,
		gosnmp.NoSuchInstance,
		gosnmp.EndOfMibView:
		return int(value)
	}

	// Unreachable - all cases handled above
	return int(value)
}

// GetType implements VariMibHandler.
func (h *VariMibIntegralHandler) GetType() gosnmp.Asn1BER {
	return h.asnType
}

// GetValue implements VariMibHandler for cycling strings.
func (h *VariMibStringHandler) GetValue() any {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.intervals) == 0 {
		return ""
	}

	elapsed := time.Since(h.startTime)
	centisecs := elapsed.Milliseconds() / CentisecsPerSec

	// Calculate total cycle time
	var totalCycle int64
	for _, interval := range h.intervals {
		totalCycle += interval.centiSeconds
	}

	if totalCycle == 0 {
		return h.intervals[0].value
	}

	// Find position in cycle
	pos := centisecs % totalCycle

	var accumulated int64
	for _, interval := range h.intervals {
		accumulated += interval.centiSeconds
		if pos < accumulated {
			return interval.value
		}
	}

	return h.intervals[len(h.intervals)-1].value
}

// GetType implements VariMibHandler.
func (h *VariMibStringHandler) GetType() gosnmp.Asn1BER {
	return h.asnType
}

// IsVariMibValue checks if a value string represents a varimib pattern.
func IsVariMibValue(value string) bool {
	value = strings.TrimSpace(value)

	return strings.HasPrefix(value, "varimib") || strings.HasPrefix(value, "fixed(")
}

// VariMibRegistry stores active varimib handlers for a device.
type VariMibRegistry struct {
	mu       sync.RWMutex
	handlers map[string]VariMibHandler // OID -> handler
}

// NewVariMibRegistry creates a new registry.
func NewVariMibRegistry() *VariMibRegistry {
	return &VariMibRegistry{
		handlers: make(map[string]VariMibHandler),
	}
}

// Register adds a handler for an OID.
func (r *VariMibRegistry) Register(oid string, handler VariMibHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	oid = strings.TrimPrefix(oid, ".")
	r.handlers[oid] = handler
}

// Get retrieves a handler for an OID.
func (r *VariMibRegistry) Get(oid string) VariMibHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	oid = strings.TrimPrefix(oid, ".")

	return r.handlers[oid]
}

// GetOIDValue returns the current value for an OID as an OIDValue.
func (r *VariMibRegistry) GetOIDValue(oid string) *OIDValue {
	handler := r.Get(oid)
	if handler == nil {
		return nil
	}

	return &OIDValue{
		Type:  handler.GetType(),
		Value: handler.GetValue(),
	}
}
