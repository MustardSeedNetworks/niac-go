package protocols

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Neighbor discovery protocol identifiers.
const (
	ProtocolLLDP = "LLDP" // Link Layer Discovery Protocol (IEEE 802.1AB)
	ProtocolCDP  = "CDP"  // Cisco Discovery Protocol
	ProtocolEDP  = "EDP"  // Extreme Discovery Protocol
	ProtocolFDP  = "FDP"  // Foundry Discovery Protocol
)

// Neighbor table constants.
const (
	neighborDefaultTTLSeconds = 180 // Default TTL for neighbor entries in seconds
)

// NeighborRecord represents a discovered network neighbor from LLDP/CDP/EDP/FDP protocols.
// json tags are explicit so /api/v1/neighbors emits snake_case fields
// the UI's toCamelCase converter recognises (was emitting raw
// PascalCase, which the frontend silently treated as undefined).
type NeighborRecord struct {
	Protocol          string        `json:"protocol"`
	LocalDevice       string        `json:"local_device"`
	RemoteDevice      string        `json:"remote_device"`
	RemotePort        string        `json:"remote_port"`
	RemoteChassisID   string        `json:"remote_chassis_id"`
	Description       string        `json:"description"`
	Capabilities      []string      `json:"capabilities"`
	ManagementAddress string        `json:"management_address"`
	LastSeen          time.Time     `json:"last_seen"`
	ExpireAt          time.Time     `json:"expire_at"`
	TTL               time.Duration `json:"ttl"`
}

type neighborTable struct {
	mu      sync.RWMutex
	entries map[string]map[string]*NeighborRecord
}

func newNeighborTable() *neighborTable {
	return &neighborTable{
		entries: make(map[string]map[string]*NeighborRecord),
	}
}

func (t *neighborTable) reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.entries = make(map[string]map[string]*NeighborRecord)
}

func (t *neighborTable) upsert(entry NeighborRecord) {
	if entry.LocalDevice == "" || entry.RemoteChassisID == "" {
		return
	}

	if entry.TTL <= 0 {
		entry.TTL = neighborDefaultTTLSeconds * time.Second
	}

	entry.LastSeen = time.Now().UTC()
	entry.ExpireAt = entry.LastSeen.Add(entry.TTL)

	key := neighborKey(entry.Protocol, entry.RemoteChassisID, entry.RemotePort)

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.entries[entry.LocalDevice]; !ok {
		t.entries[entry.LocalDevice] = make(map[string]*NeighborRecord)
	}

	clone := entry
	t.entries[entry.LocalDevice][key] = &clone
}

func (t *neighborTable) cleanupExpired() {
	now := time.Now().UTC()

	t.mu.Lock()
	defer t.mu.Unlock()

	for local, neighbors := range t.entries {
		for key, record := range neighbors {
			if now.After(record.ExpireAt) {
				delete(neighbors, key)
			}
		}

		if len(neighbors) == 0 {
			delete(t.entries, local)
		}
	}
}

func (t *neighborTable) list() []NeighborRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var out []NeighborRecord

	for _, neighbors := range t.entries {
		for _, record := range neighbors {
			out = append(out, *record)
		}
	}

	return out
}

func neighborKey(protocol, chassis, port string) string {
	return fmt.Sprintf("%s|%s|%s", protocol, chassis, port)
}

func dedupStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}

	seen := make(map[string]struct{}, len(values))

	out := make([]string, 0, len(values))

	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}

		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}

		out = append(out, v)
	}

	return out
}
