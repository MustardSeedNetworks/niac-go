package devicestate

// UpsertRoute creates or replaces a route with the same destination.
func (s *Store) UpsertRoute(route Route) {
	s.mu.Lock()
	defer s.mu.Unlock()

	route.Destination = route.Destination.Masked()
	for index, current := range s.running.network.Routes {
		if current.Destination == route.Destination {
			s.running.network.Routes[index] = route
			s.version++
			s.recordEvent(EventRouteUpdated, route.Destination.String())
			return
		}
	}
	s.running.network.Routes = append(s.running.network.Routes, route)
	s.version++
	s.recordEvent(EventRouteUpdated, route.Destination.String())
}
