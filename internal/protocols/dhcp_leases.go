package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// SetPool configures the DHCP IP address pool.
func (h *DHCPHandler) SetPool(start, end net.IP) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.poolStart = start
	h.poolEnd = end

	pool, err := h.generateIPPool(start, end)
	if err != nil {
		// Log error but don't fail initialization - just use empty pool
		fmt.Fprintf(os.Stderr, "Warning: DHCP pool generation failed: %v\n", err)

		h.ipPool = []net.IP{}
	} else {
		h.ipPool = pool
	}
}

// SetServerConfig configures DHCP server parameters.
func (h *DHCPHandler) SetServerConfig(serverIP, gateway net.IP, dnsServers []net.IP, domain string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.serverIP = serverIP
	h.gateway = gateway
	h.dnsServers = dnsServers
	h.domainName = domain
}

// SetAdvancedOptions configures advanced DHCP options.
func (h *DHCPHandler) SetAdvancedOptions(
	ntpServers []net.IP,
	domainSearch []string,
	tftpServer, bootfile string,
	vendorInfo []byte,
) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.ntpServers = ntpServers
	h.domainSearch = domainSearch
	h.tftpServerName = tftpServer
	h.bootfileName = bootfile
	h.vendorSpecificInfo = vendorInfo
}

// Reset clears all DHCP server state while preserving the associated stack.
func (h *DHCPHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.leases = make(map[string]*DHCPLease)
	h.ipPool = nil
	h.poolStart = nil
	h.poolEnd = nil
	h.serverIP = nil
	h.subnetMask = getDefaultSubnetMask()
	h.gateway = nil
	h.dnsServers = nil
	h.domainName = ""
	h.ntpServers = nil
	h.domainSearch = nil
	h.tftpServerName = ""
	h.bootfileName = ""
	h.vendorSpecificInfo = nil
	h.staticLeases = nil
}

// SetStaticLeases configures static DHCP leases.
func (h *DHCPHandler) SetStaticLeases(leases []config.DHCPLease) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.staticLeases = leases
}

// generateIPPool creates a list of available IPs
// Returns error if pool size exceeds MaxPoolSize or if range is invalid
// SECURITY FIX MEDIUM-1: Validates range to prevent integer overflow.
func (h *DHCPHandler) generateIPPool(start, end net.IP) ([]net.IP, error) {
	startInt := binary.BigEndian.Uint32(start.To4())
	endInt := binary.BigEndian.Uint32(end.To4())

	// SECURITY: Validate range to prevent integer overflow
	// This check ensures endInt >= startInt before subtraction
	if endInt < startInt {
		return nil, fmt.Errorf("%w: end IP (%s) < start IP (%s)", ErrDHCPPoolInvalid, end, start)
	}

	// Calculate pool size (safe because endInt >= startInt)
	// Using uint64 to prevent overflow even for max range (2^32 - 1)
	size := uint64(endInt) - uint64(startInt) + 1
	if size > MaxPoolSize {
		return nil, fmt.Errorf("%w: size %d exceeds maximum %d (range: %s to %s)",
			ErrDHCPPoolSizeExceeded, size, MaxPoolSize, start, end)
	}

	// Pre-allocate slice with exact capacity
	pool := make([]net.IP, 0, size)

	for i := startInt; i <= endInt; i++ {
		ip := make(net.IP, dhcpIPv4Len)
		binary.BigEndian.PutUint32(ip, i)
		pool = append(pool, ip)
	}

	return pool, nil
}

// findAvailableIP finds an available IP address
// Note: Caller must hold h.mu lock.
func (h *DHCPHandler) findAvailableIP() net.IP {
	// Check each IP in pool
	for _, ip := range h.ipPool {
		inUse := false

		for _, lease := range h.leases {
			if lease.IP.Equal(ip) && time.Now().Before(lease.Expiry) {
				inUse = true

				break
			}
		}

		if !inUse {
			return ip
		}
	}

	return nil
}

// allocateLease allocates or renews a lease.
func (h *DHCPHandler) allocateLease(mac net.HardwareAddr, requestedIP net.IP, hostname string) (*DHCPLease, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	macStr := mac.String()

	// Check static leases first
	if staticIP := h.matchStaticLease(mac); staticIP != nil {
		if existing, ok := h.leases[macStr]; ok {
			existing.Expiry = time.Now().Add(DefaultLeaseTime)

			existing.IP = staticIP

			if hostname != "" {
				existing.Hostname = hostname
			}

			return existing, nil
		}

		lease := &DHCPLease{
			IP:        staticIP,
			MAC:       mac,
			Hostname:  hostname,
			Expiry:    time.Now().Add(DefaultLeaseTime),
			LeaseTime: DefaultLeaseTime,
		}
		h.leases[macStr] = lease

		return lease, nil
	}

	// Check if client already has a lease
	if existing, ok := h.leases[macStr]; ok {
		// Renew existing lease
		existing.Expiry = time.Now().Add(DefaultLeaseTime)
		// Update hostname if provided
		if hostname != "" {
			existing.Hostname = hostname
		}

		return existing, nil
	}

	// Find available IP
	var ip net.IP
	if requestedIP != nil && h.isIPInPool(requestedIP) && !h.isIPLeased(requestedIP) {
		ip = requestedIP
	} else {
		ip = h.findAvailableIP()
	}

	if ip == nil {
		return nil, ErrNoAvailableIPAddresses
	}

	// Create new lease
	lease := &DHCPLease{
		IP:        ip,
		MAC:       mac,
		Hostname:  hostname,
		Expiry:    time.Now().Add(DefaultLeaseTime),
		LeaseTime: DefaultLeaseTime,
	}

	h.leases[macStr] = lease

	return lease, nil
}

// matchStaticLease finds a matching static lease IP for the given MAC.
func (h *DHCPHandler) matchStaticLease(mac net.HardwareAddr) net.IP {
	for _, lease := range h.staticLeases {
		if lease.MACAddress == nil {
			continue
		}

		if macMatchesMask(mac, lease.MACAddress, lease.MACMask) {
			return lease.ClientIP
		}
	}

	return nil
}

func macMatchesMask(mac, match, mask net.HardwareAddr) bool {
	if len(mac) == 0 || len(match) == 0 {
		return false
	}

	if len(mask) == 0 {
		return mac.String() == match.String()
	}

	if len(mask) != len(mac) || len(match) != len(mac) {
		return false
	}

	for i := range mac {
		if (mac[i] & mask[i]) != (match[i] & mask[i]) {
			return false
		}
	}

	return true
}

// isIPInPool checks if IP is in the pool.
func (h *DHCPHandler) isIPInPool(ip net.IP) bool {
	for _, poolIP := range h.ipPool {
		if poolIP.Equal(ip) {
			return true
		}
	}

	return false
}

// isIPLeased checks if IP is currently leased.
func (h *DHCPHandler) isIPLeased(ip net.IP) bool {
	for _, lease := range h.leases {
		if lease.IP.Equal(ip) && time.Now().Before(lease.Expiry) {
			return true
		}
	}

	return false
}
