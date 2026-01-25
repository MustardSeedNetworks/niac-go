package protocols

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"net"
	"time"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// generateDUID generates a DUID-LL (Link-Layer) for the server.
func generateDUID() []byte {
	// DUID-LL format: Type(2) + HW Type(2) + Link-Layer Address(variable)
	duid := make([]byte, duidLLSize)
	binary.BigEndian.PutUint16(duid[0:2], DUIDTypeLL) // DUID-LL
	binary.BigEndian.PutUint16(duid[2:4], 1)          // Ethernet

	// Generate random MAC for server DUID
	mac := make([]byte, macAddrSize)
	_, _ = rand.Read(mac)                               // crypto/rand read errors will result in zero bytes
	mac[0] = (mac[0] | macLocalBitSet) & macUnicastMask // Set local, clear multicast
	copy(duid[4:10], mac)

	return duid
}

// duidString converts DUID bytes to hex string for map key.
func duidString(duid []byte) string {
	return hex.EncodeToString(duid)
}

// findIPv6ServerDevice finds a device with an IPv6 address from the device list.
func findIPv6ServerDevice(devices []*config.Device) *config.Device {
	for _, dev := range devices {
		if hasIPv6Address(dev.IPAddresses) {
			return dev
		}
	}

	return nil
}

// hasIPv6Address checks if the IP list contains any IPv6 address.
func hasIPv6Address(ips []net.IP) bool {
	for _, ip := range ips {
		if ip.To4() == nil && ip.To16() != nil {
			return true
		}
	}

	return false
}

// getFirstIPv6Address returns the first IPv6 address from the list.
func getFirstIPv6Address(ips []net.IP) net.IP {
	for _, ip := range ips {
		if ip.To4() == nil && ip.To16() != nil {
			return ip
		}
	}

	return nil
}

// allocateLease allocates a new IPv6 address lease.
func (h *DHCPv6Handler) allocateLease(clientDUID []byte, iaid uint32) (*DHCPv6Lease, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	duidKey := duidString(clientDUID)

	// Check if client already has a lease
	if existing, ok := h.leases[duidKey]; ok {
		// Renew existing lease
		h.renewLeaseUnlocked(existing)

		return existing, nil
	}

	// Find available address
	address := h.findAvailableAddress()
	if address == nil {
		return nil, ErrNoAvailableAddresses
	}

	// Create new lease
	now := time.Now()
	lease := &DHCPv6Lease{
		Address:           address,
		DUID:              clientDUID,
		IAID:              iaid,
		PreferredLifetime: now.Add(h.preferredLifetime),
		ValidLifetime:     now.Add(h.validLifetime),
		LastRenewal:       now,
	}

	h.leases[duidKey] = lease

	return lease, nil
}

// confirmLease confirms an existing lease or allocates new one.
func (h *DHCPv6Handler) confirmLease(clientDUID []byte, iaid uint32) (*DHCPv6Lease, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	duidKey := duidString(clientDUID)

	if existing, ok := h.leases[duidKey]; ok {
		h.renewLeaseUnlocked(existing)

		return existing, nil
	}

	// Allocate new lease (unlock then relock via allocateLease)
	h.mu.Unlock()
	lease, err := h.allocateLease(clientDUID, iaid)
	h.mu.Lock()

	return lease, err
}

// findLease finds a lease by client DUID.
func (h *DHCPv6Handler) findLease(clientDUID []byte) *DHCPv6Lease {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if lease, ok := h.leases[duidString(clientDUID)]; ok {
		return lease
	}

	return nil
}

// renewLease renews a lease (with locking).
func (h *DHCPv6Handler) renewLease(lease *DHCPv6Lease) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.renewLeaseUnlocked(lease)
}

// renewLeaseUnlocked renews a lease (without locking).
func (h *DHCPv6Handler) renewLeaseUnlocked(lease *DHCPv6Lease) {
	now := time.Now()
	lease.PreferredLifetime = now.Add(h.preferredLifetime)
	lease.ValidLifetime = now.Add(h.validLifetime)
	lease.LastRenewal = now
}

// findAvailableAddress finds an available IPv6 address.
func (h *DHCPv6Handler) findAvailableAddress() net.IP {
	// Check pool for available address
	for _, addr := range h.addressPool {
		inUse := false

		for _, lease := range h.leases {
			if lease.Address.Equal(addr) && time.Now().Before(lease.ValidLifetime) {
				inUse = true

				break
			}
		}

		if !inUse {
			// Return a copy
			result := make(net.IP, len(addr))
			copy(result, addr)

			return result
		}
	}

	return nil
}

// splitDomainLabels splits a domain name into labels.
func splitDomainLabels(domain string) []string {
	if domain == "" {
		return []string{}
	}

	labels := make([]string, 0)
	start := 0

	for i := range len(domain) {
		if domain[i] == '.' {
			if i > start {
				labels = append(labels, domain[start:i])
			}

			start = i + 1
		}
	}

	if start < len(domain) {
		labels = append(labels, domain[start:])
	}

	return labels
}

// encodeIPv6List encodes a list of IPv6 addresses into bytes.
func encodeIPv6List(ips []net.IP) []byte {
	data := make([]byte, 0, len(ips)*ipv6AddrSize)
	for _, ip := range ips {
		data = append(data, ip.To16()...)
	}

	return data
}
