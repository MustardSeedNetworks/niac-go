package protocols

import (
	"net"
	"time"
)

// findAvailableIP requires h.mu to be held by the caller.
func (h *DHCPHandler) findAvailableIP() net.IP {
	for _, ip := range h.ipPool {
		if _, declined := h.declined[ip.String()]; declined {
			continue
		}

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
