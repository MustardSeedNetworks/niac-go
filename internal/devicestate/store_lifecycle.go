package devicestate

// SaveStartup copies the running configuration to startup configuration.
func (s *Store) SaveStartup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.startup = cloneConfiguration(s.running)
	s.version++
	s.recordEvent(EventStartupSaved, "")
}

// ReloadStartup restores running configuration from startup configuration.
func (s *Store) ReloadStartup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = cloneConfiguration(s.startup)
	s.version++
	s.recordEvent(EventStartupReloaded, "")
}

// ResetAuthored restores authored configuration as both running and startup.
func (s *Store) ResetAuthored() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.running = cloneConfiguration(s.authored)
	s.startup = cloneConfiguration(s.authored)
	s.version++
	s.recordEvent(EventAuthoredReset, "")
}

// EraseStartup restores authored configuration as startup without changing running state.
func (s *Store) EraseStartup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.startup = cloneConfiguration(s.authored)
	s.version++
	s.recordEvent(EventStartupErased, "")
}

func cloneConfiguration(source configuration) configuration {
	return configuration{identity: source.identity, network: cloneNetwork(source.network)}
}
