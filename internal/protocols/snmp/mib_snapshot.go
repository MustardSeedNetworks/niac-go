package snmp

func (m *MIB) snapshotOIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	oids := make([]string, 0, len(m.entries))
	for oid := range m.entries {
		oids = append(oids, oid)
	}
	return oids
}
