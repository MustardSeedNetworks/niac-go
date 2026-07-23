package devicestate

import (
	"errors"
	"slices"
	"strconv"
)

// ErrVLANNotFound indicates that a device has no VLAN with the requested ID.
var ErrVLANNotFound = errors.New("vlan not found")

// EnsureVLAN creates an active VLAN when it is not already present.
func (s *Store) EnsureVLAN(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, vlan := range s.running.network.VLANs {
		if vlan.ID == id {
			return
		}
	}
	s.running.network.VLANs = append(s.running.network.VLANs, VLAN{ID: id, Active: true})
	s.version++
	s.recordEvent(EventVLANUpdated, strconv.Itoa(id))
}

// UpdateVLAN applies one version-checked transaction to a configured VLAN.
func (s *Store) UpdateVLAN(id int, update func(VLAN) VLAN) error {
	s.mu.RLock()
	for index, vlan := range s.running.network.VLANs {
		if vlan.ID == id {
			version := s.version
			s.mu.RUnlock()
			next := update(vlan)
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.version != version {
				return ErrConcurrentUpdate
			}
			s.running.network.VLANs[index] = next
			s.version++
			s.recordEvent(EventVLANUpdated, strconv.Itoa(id))
			return nil
		}
	}
	s.mu.RUnlock()
	return ErrVLANNotFound
}

// EnsureRouter creates a routing process when it is not already present.
func (s *Store) EnsureRouter(protocol, processID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, router := range s.running.network.Routers {
		if router.Protocol == protocol && router.ProcessID == processID {
			return
		}
	}
	s.running.network.Routers = append(s.running.network.Routers, Router{
		Protocol: protocol, ProcessID: processID,
	})
	s.version++
	s.recordEvent(EventRouterUpdated, protocol+":"+processID)
}

// AddRouterNetwork adds one unique network statement to a routing process.
func (s *Store) AddRouterNetwork(protocol, processID string, network RouterNetwork) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for index := range s.running.network.Routers {
		router := &s.running.network.Routers[index]
		if router.Protocol != protocol || router.ProcessID != processID {
			continue
		}
		if slices.Contains(router.Networks, network) {
			return
		}
		router.Networks = append(router.Networks, network)
		s.version++
		s.recordEvent(EventRouterUpdated, protocol+":"+processID)
		return
	}
}
