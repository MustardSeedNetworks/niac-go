package snmp

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gosnmp/gosnmp"
)

// OIDValue represents an OID value with type.
type OIDValue struct {
	Type    gosnmp.Asn1BER
	Value   any
	Dynamic func() *OIDValue // For dynamic values like sysUpTime
}

// MIB represents a Management Information Base.
type MIB struct {
	mu      sync.RWMutex
	entries map[string]*OIDValue // OID string -> value
	sorted  []string             // Sorted list of OIDs for GetNext
	dirty   bool                 // True if sorted list needs updating
}

// NewMIB creates a new MIB.
func NewMIB() *MIB {
	return &MIB{
		entries: make(map[string]*OIDValue),
		sorted:  []string{},
		dirty:   false,
	}
}

// Set sets an OID value.
func (m *MIB) Set(oid string, value *OIDValue) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Normalize OID (remove leading dot)
	oid = strings.TrimPrefix(oid, ".")

	m.entries[oid] = value
	m.dirty = true
}

// SetDynamic sets a dynamic OID value (computed on each access).
func (m *MIB) SetDynamic(oid string, fn func() *OIDValue) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Normalize OID
	oid = strings.TrimPrefix(oid, ".")

	m.entries[oid] = &OIDValue{
		Dynamic: fn,
	}
	m.dirty = true
}

// Get retrieves an OID value.
func (m *MIB) Get(oid string) *OIDValue {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Normalize OID
	oid = strings.TrimPrefix(oid, ".")

	value, exists := m.entries[oid]
	if !exists {
		return nil
	}

	// If dynamic, call function to get current value
	if value.Dynamic != nil {
		return value.Dynamic()
	}

	return value
}

// IndexSuffixForValue returns the trailing index of the OID under table (a
// table-column prefix, e.g. ifDescr) whose value stringifies to want. When more
// than one entry matches it returns the numerically largest index: a device with
// both a real walk and trunk_ports carries the interface twice — the walk's real
// ifIndex (Cisco convention, 10000+) and a small sequential synthesis index —
// and the walk's is the one whose bridge port / ifName a manager renders.
// Returns the suffix after "<table>." and true on a hit.
func (m *MIB) IndexSuffixForValue(table, want string) (string, bool) {
	table = strings.TrimPrefix(table, ".")
	prefix := table + "."

	m.mu.RLock()
	defer m.mu.RUnlock()

	best := ""
	bestNum := -1
	found := false

	for oid, val := range m.entries {
		if !strings.HasPrefix(oid, prefix) {
			continue
		}

		v := val
		if v.Dynamic != nil {
			v = v.Dynamic()
		}

		if oidValueString(v) != want {
			continue
		}

		suffix := strings.TrimPrefix(oid, prefix)
		// Prefer the largest single-integer index; fall back to first match for
		// non-numeric suffixes.
		if n, err := strconv.Atoi(suffix); err == nil {
			if n > bestNum {
				bestNum, best = n, suffix
			}
		} else if !found {
			best = suffix
		}

		found = true
	}

	return best, found
}

// SampleIntEntry returns one entry under table (a table-column prefix) whose
// suffix and value are both integers: its numeric suffix and integer value. Used
// to infer a table's index convention — e.g. dot1dBasePortIfIndex is linear
// (ifIndex = bridgePort + constant), so one sample gives the offset for ports the
// capture didn't model. Returns ok=false if the table has no such entry.
func (m *MIB) SampleIntEntry(table string) (int, int, bool) {
	table = strings.TrimPrefix(table, ".")
	prefix := table + "."

	m.mu.RLock()
	defer m.mu.RUnlock()

	for oid, val := range m.entries {
		if !strings.HasPrefix(oid, prefix) {
			continue
		}

		s, err := strconv.Atoi(strings.TrimPrefix(oid, prefix))
		if err != nil {
			continue
		}

		v := val
		if v.Dynamic != nil {
			v = v.Dynamic()
		}

		iv, err2 := strconv.Atoi(oidValueString(v))
		if err2 != nil {
			continue
		}

		return s, iv, true
	}

	return 0, 0, false
}

// oidValueString renders an OIDValue's payload as a string for comparison,
// covering the string and []byte encodings a walk may load.
func oidValueString(v *OIDValue) string {
	switch t := v.Value.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprintf("%v", v.Value)
	}
}

// GetNext retrieves the next OID in lexicographical order.
func (m *MIB) GetNext(oid string) (string, *OIDValue) {
	// Normalize OID
	oid = strings.TrimPrefix(oid, ".")

	// First pass: check if update needed while holding read lock
	m.mu.RLock()
	needsUpdate := m.dirty
	m.mu.RUnlock()

	// Upgrade to write lock if needed
	if needsUpdate {
		m.mu.Lock()
		// Double-check after acquiring lock (another goroutine may have updated)
		if m.dirty {
			m.updateSortedList()
		}

		m.mu.Unlock()
	}

	// Re-acquire read lock for the actual read
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Binary search the sorted list for the first OID strictly greater than
	// the requested one. The list is kept in ascending compareOIDs order, so a
	// full snmpwalk costs O(walk_len · log n) instead of O(walk_len · n) — the
	// difference between responding and timing out on a 30k-OID device walk.
	idx := sort.Search(len(m.sorted), func(i int) bool {
		return compareOIDs(m.sorted[i], oid) > 0
	})

	if idx >= len(m.sorted) {
		// End of MIB
		return "", nil
	}

	nextOID := m.sorted[idx]
	value := m.entries[nextOID]

	// If dynamic, call function
	if value.Dynamic != nil {
		return nextOID, value.Dynamic()
	}

	return nextOID, value
}

// Reindex eagerly rebuilds the sorted OID list if it is stale.
//
// GetNext needs a sorted view of the OID space. Building it lazily on the first
// GetNext means that sort runs on the stack's single decode goroutine, so a
// fresh multi-device discovery (every device's MIB dirty at once) serializes one
// large sort per device on the hot path and stalls SNMP responses fleet-wide —
// big devices time out with "no details" while tiny APs slip through. Call this
// once after loading a device's OIDs so the cost is paid at startup, not during
// a walk.
func (m *MIB) Reindex() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.dirty {
		m.updateSortedList()
	}
}

// updateSortedList updates the sorted OID list (caller must hold write lock).
//
// Each OID is parsed into its integer arcs exactly once (decorate-sort-
// undecorate) rather than re-parsing both operands on every comparison. On a
// 15k-OID device that is the difference between O(n) and O(n log n) parses — and
// with many devices reindexing at once, the naive version pegged a core for
// minutes.
func (m *MIB) updateSortedList() {
	type keyed struct {
		oid   string
		parts []int
	}

	decorated := make([]keyed, 0, len(m.entries))
	for oid := range m.entries {
		decorated = append(decorated, keyed{oid: oid, parts: parseOIDParts(oid)})
	}

	sort.Slice(decorated, func(i, j int) bool {
		return compareOIDParts(decorated[i].parts, decorated[j].parts) < 0
	})

	m.sorted = make([]string, len(decorated))
	for i, d := range decorated {
		m.sorted[i] = d.oid
	}

	m.dirty = false
}

// Delete removes an OID from the MIB.
// FEATURE #116: Added godoc for utility functions.
func (m *MIB) Delete(oid string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Normalize OID
	oid = strings.TrimPrefix(oid, ".")

	if _, exists := m.entries[oid]; exists {
		delete(m.entries, oid)
		m.dirty = true
	}
}

// Count returns the number of OIDs in the MIB.
func (m *MIB) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.entries)
}

// AllOIDs returns all OIDs in sorted order.
func (m *MIB) AllOIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.dirty {
		m.mu.RUnlock()
		m.mu.Lock()
		m.updateSortedList()
		m.mu.Unlock()
		m.mu.RLock()
	}

	// Return copy
	result := make([]string, len(m.sorted))
	copy(result, m.sorted)

	return result
}

// compareOIDs compares two OID strings lexicographically.
// Returns -1 if oid1 < oid2, 0 if equal, 1 if oid1 > oid2.
// FEATURE #116: Added godoc for utility functions.
func compareOIDs(oid1, oid2 string) int {
	return compareOIDParts(parseOIDParts(oid1), parseOIDParts(oid2))
}

// compareOIDParts compares two OIDs already parsed into integer arcs.
// Returns -1 if parts1 < parts2, 0 if equal, 1 if parts1 > parts2.
func compareOIDParts(parts1, parts2 []int) int {
	minLen := min(len(parts1), len(parts2))

	for i := range minLen {
		if parts1[i] < parts2[i] {
			return -1
		}

		if parts1[i] > parts2[i] {
			return 1
		}
	}

	// If all components equal so far, shorter OID comes first
	if len(parts1) < len(parts2) {
		return -1
	}

	if len(parts1) > len(parts2) {
		return 1
	}

	return 0
}

// parseOIDParts parses an OID string into integer components.
// Examples: "1.3.6.1" -> [1, 3, 6, 1], ".1.3.6" -> [1, 3, 6]
// FEATURE #116: Added godoc for utility functions.
func parseOIDParts(oid string) []int {
	parts := strings.Split(oid, ".")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		if part == "" {
			continue
		}

		// strconv.Atoi rather than fmt.Sscanf: this runs for every arc of every
		// OID compared during a sort, so on a 15k-OID device it is called
		// millions of times. Sscanf's reflection made each MIB sort take seconds
		// on the single decode goroutine and stalled multi-device discovery.
		num, err := strconv.Atoi(part)
		if err == nil {
			result = append(result, num)
		}
	}

	return result
}

// FormatOID formats an OID for display.
func FormatOID(oid string) string {
	// Add leading dot if not present
	if !strings.HasPrefix(oid, ".") {
		return "." + oid
	}

	return oid
}

// IsValidOID checks if an OID string is valid.
func IsValidOID(oid string) bool {
	if oid == "" {
		return false
	}

	oid = strings.TrimPrefix(oid, ".")

	for part := range strings.SplitSeq(oid, ".") {
		if part == "" {
			return false
		}

		num, err := strconv.Atoi(part)
		if err != nil {
			return false
		}

		if num < 0 {
			return false
		}
	}

	return true
}

// StandardOIDs contains common MIB-II OID prefixes.
var StandardOIDs = map[string]string{
	"system":     "1.3.6.1.2.1.1",
	"interfaces": "1.3.6.1.2.1.2",
	"at":         "1.3.6.1.2.1.3",
	"ip":         "1.3.6.1.2.1.4",
	"icmp":       "1.3.6.1.2.1.5",
	"tcp":        "1.3.6.1.2.1.6",
	"udp":        "1.3.6.1.2.1.7",
	"snmp":       "1.3.6.1.2.1.11",
}

// GetStandardOID returns the OID for a standard MIB-II group.
func GetStandardOID(name string) (string, bool) {
	oid, exists := StandardOIDs[name]

	return oid, exists
}
