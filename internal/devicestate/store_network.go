package devicestate

// ReplaceNetwork atomically installs compiled interface and route state.
func (s *Store) ReplaceNetwork(network Network) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for index := range network.Interfaces {
		if network.Interfaces[index].OperUp {
			network.Interfaces[index].CarrierUp = true
		}
	}
	s.running.network = cloneNetwork(network)
	s.startup.network = cloneNetwork(network)
	s.authored.network = cloneNetwork(network)
	s.version++
	s.recordEvent(EventNetworkInstalled, "")
}
