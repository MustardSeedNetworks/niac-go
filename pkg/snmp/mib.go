package snmp

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/gosnmp/gosnmp"
)

// OIDValue represents an OID value with type
type OIDValue struct {
	Type     gosnmp.Asn1BER
	Value    interface{}
	Dynamic  func() *OIDValue // For dynamic values like sysUpTime
}

// MIB represents a Management Information Base
type MIB struct {
	mu      sync.RWMutex
	entries map[string]*OIDValue // OID string -> value
	sorted  []string             // Sorted list of OIDs for GetNext
	dirty   bool                 // True if sorted list needs updating
}

// NewMIB creates a new MIB
func NewMIB() *MIB {
	return &MIB{
		entries: make(map[string]*OIDValue),
		sorted:  []string{},
		dirty:   false,
	}
}

// Set sets an OID value
func (m *MIB) Set(oid string, value *OIDValue) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Normalize OID (remove leading dot)
	oid = strings.TrimPrefix(oid, ".")

	m.entries[oid] = value
	m.dirty = true
}

// SetDynamic sets a dynamic OID value (computed on each access)
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

// Get retrieves an OID value
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

// GetNext retrieves the next OID in lexicographical order
func (m *MIB) GetNext(oid string) (string, *OIDValue) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Normalize OID
	oid = strings.TrimPrefix(oid, ".")

	// Update sorted list if needed
	if m.dirty {
		m.mu.RUnlock()
		m.mu.Lock()
		m.updateSortedList()
		m.mu.Unlock()
		m.mu.RLock()
	}

	// Find next OID in sorted list
	for _, nextOID := range m.sorted {
		if compareOIDs(nextOID, oid) > 0 {
			value := m.entries[nextOID]

			// If dynamic, call function
			if value.Dynamic != nil {
				return nextOID, value.Dynamic()
			}

			return nextOID, value
		}
	}

	// End of MIB
	return "", nil
}

// updateSortedList updates the sorted OID list (caller must hold write lock)
func (m *MIB) updateSortedList() {
	m.sorted = make([]string, 0, len(m.entries))
	for oid := range m.entries {
		m.sorted = append(m.sorted, oid)
	}

	// Sort using OID comparison
	sort.Slice(m.sorted, func(i, j int) bool {
		return compareOIDs(m.sorted[i], m.sorted[j]) < 0
	})

	m.dirty = false
}

// Delete removes an OID from the MIB
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

// Count returns the number of OIDs in the MIB
func (m *MIB) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entries)
}

// AllOIDs returns all OIDs in sorted order
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

// compareOIDs compares two OID strings lexicographically
// Returns: -1 if oid1 < oid2, 0 if equal, 1 if oid1 > oid2
func compareOIDs(oid1, oid2 string) int {
	// Parse both OIDs
	parts1 := parseOIDParts(oid1)
	parts2 := parseOIDParts(oid2)

	// Compare component by component
	minLen := len(parts1)
	if len(parts2) < minLen {
		minLen = len(parts2)
	}

	for i := 0; i < minLen; i++ {
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

// parseOIDParts parses an OID string into integer components
func parseOIDParts(oid string) []int {
	parts := strings.Split(oid, ".")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		if part == "" {
			continue
		}
		var num int
		_, err := fmt.Sscanf(part, "%d", &num)
		if err == nil {
			result = append(result, num)
		}
	}

	return result
}

// FormatOID formats an OID for display
func FormatOID(oid string) string {
	// Add leading dot if not present
	if !strings.HasPrefix(oid, ".") {
		return "." + oid
	}
	return oid
}

// IsValidOID checks if an OID string is valid
func IsValidOID(oid string) bool {
	if oid == "" {
		return false
	}

	oid = strings.TrimPrefix(oid, ".")
	parts := strings.Split(oid, ".")

	for _, part := range parts {
		if part == "" {
			return false
		}
		var num int
		_, err := fmt.Sscanf(part, "%d", &num)
		if err != nil {
			return false
		}
		if num < 0 {
			return false
		}
	}

	return true
}

// StandardOIDs contains common MIB-II OID prefixes
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

// GetStandardOID returns the OID for a standard MIB-II group
func GetStandardOID(name string) (string, bool) {
	oid, exists := StandardOIDs[name]
	return oid, exists
}
