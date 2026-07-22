package devicestate

// ReplaceNetwork atomically installs compiled interface and route state.
func (s *Store) ReplaceNetwork(network Network) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running.network = cloneNetwork(network)
	s.startup.network = cloneNetwork(network)
	s.authored.network = cloneNetwork(network)
	s.version++
	s.recordEvent(EventNetworkInstalled, "")
}
