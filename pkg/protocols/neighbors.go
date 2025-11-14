package protocols

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	ProtocolLLDP = "LLDP"
	ProtocolCDP  = "CDP"
	ProtocolEDP  = "EDP"
	ProtocolFDP  = "FDP"
)

type NeighborRecord struct {
	Protocol          string
	LocalDevice       string
	RemoteDevice      string
	RemotePort        string
	RemoteChassisID   string
	Description       string
	Capabilities      []string
	ManagementAddress string
	LastSeen          time.Time
	ExpireAt          time.Time
	TTL               time.Duration
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
		entry.TTL = 180 * time.Second
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

func capabilitiesToStrings(caps []string) []string {
	if len(caps) == 0 {
		return nil
	}
	out := make([]string, len(caps))
	copy(out, caps)
	return out
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
