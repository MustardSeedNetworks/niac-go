package devicestate

import (
	"errors"
	"slices"
)

var (
	// ErrInterfaceNotFound indicates that a device has no interface by that name.
	ErrInterfaceNotFound = errors.New("interface not found")
	// ErrInterfaceRename indicates that interface identity cannot change through an update.
	ErrInterfaceRename = errors.New("interface rename is not supported")
)

// UpdateInterface applies one atomic transaction to a named interface.
func (s *Store) UpdateInterface(name string, update func(Interface) (Interface, error)) error {
	s.mu.RLock()
	for index, current := range s.running.network.Interfaces {
		if current.Name != name {
			continue
		}
		version := s.version
		s.mu.RUnlock()
		next, err := update(cloneInterface(current))
		if err != nil {
			return err
		}
		if next.Name != current.Name {
			return ErrInterfaceRename
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.version != version {
			return ErrConcurrentUpdate
		}
		s.running.network.Interfaces[index] = cloneInterface(next)
		if current.Address != next.Address {
			s.running.network.Routes = reconcileConnectedRoute(s.running.network.Routes, next)
		}
		s.version++
		s.recordInterfaceEvent(index+1, current, next)
		return nil
	}
	s.mu.RUnlock()
	return ErrInterfaceNotFound
}

func reconcileConnectedRoute(routes []Route, iface Interface) []Route {
	result := slices.DeleteFunc(append([]Route(nil), routes...), func(route Route) bool {
		return route.Connected && route.Via == iface.Name
	})
	if iface.Address.IsValid() {
		result = append(result, Route{
			Destination: iface.Address.Masked(), Via: iface.Name, Connected: true,
		})
	}
	return result
}

func cloneInterface(iface Interface) Interface {
	iface.VLANs = append([]int(nil), iface.VLANs...)
	return iface
}
