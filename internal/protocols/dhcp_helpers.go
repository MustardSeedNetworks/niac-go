package protocols

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// MaxPoolSize is the maximum number of IPs allowed in a DHCP pool.
const MaxPoolSize = 65536 // 2^16 IPs (reasonable for simulation)

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

// findServerDevice finds a suitable server device from the device list.
func findServerDevice(devices []*config.Device) *config.Device {
	for _, dev := range devices {
		if len(dev.IPAddresses) > 0 {
			return dev
		}
	}

	return nil
}

// getRequestedIP extracts the requested IP from DHCP options.
func getRequestedIP(dhcp *layers.DHCPv4) net.IP {
	for _, opt := range dhcp.Options {
		if opt.Type == layers.DHCPOptRequestIP && len(opt.Data) == 4 {
			return net.IP(opt.Data)
		}
	}

	return nil
}

func (h *DHCPHandler) updateFDBTables(mac net.HardwareAddr) {
	if h == nil || h.stack == nil {
		return
	}

	h.stack.updateFDBTables(mac)
}

// SendDHCPOffer sends a DHCP Offer message.
func (h *DHCPHandler) SendDHCPOffer(
	xid uint32,
	clientMAC net.HardwareAddr,
	offeredIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
) error {
	return h.sendDHCPResponse(xid, clientMAC, offeredIP, serverIP, serverMAC, DHCPOffer)
}

// SendDHCPAck sends a DHCP Ack message.
func (h *DHCPHandler) SendDHCPAck(
	xid uint32,
	clientMAC net.HardwareAddr,
	assignedIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
) error {
	return h.sendDHCPResponse(xid, clientMAC, assignedIP, serverIP, serverMAC, DHCPAck)
}

// sendDHCPResponse sends a DHCP Offer or Ack response.
func (h *DHCPHandler) sendDHCPResponse(
	xid uint32,
	clientMAC net.HardwareAddr,
	assignedIP, serverIP net.IP,
	serverMAC net.HardwareAddr,
	msgType uint8,
) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpReply,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  dhcpHWAddrLen,
		HardwareOpts: 0,
		Xid:          xid,
		Secs:         0,
		Flags:        dhcpBroadcastFlag,
		ClientIP:     net.IPv4zero,
		YourClientIP: assignedIP,
		NextServerIP: net.IPv4zero,
		RelayAgentIP: net.IPv4zero,
		ClientHWAddr: clientMAC,
	}
	dhcp.Options = h.buildDHCPOptions(msgType, serverIP.To4(), clientMAC)

	return h.serializeAndSendDHCP(dhcp, serverIP, serverMAC)
}

// serializeAndSendDHCP serializes and sends a DHCP packet.
func (h *DHCPHandler) serializeAndSendDHCP(
	dhcp *layers.DHCPv4,
	serverIP net.IP,
	serverMAC net.HardwareAddr,
) error {
	udp := &layers.UDP{SrcPort: dhcpServerPort, DstPort: dhcpClientPort}
	ip := &layers.IPv4{
		Version: ipv4Version, TTL: ipv4DefaultTTL, Protocol: layers.IPProtocolUDP,
		SrcIP: serverIP, DstIP: net.IPv4bcast,
	}
	eth := &layers.Ethernet{
		SrcMAC: serverMAC, DstMAC: net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}

	_ = udp.SetNetworkLayerForChecksum(ip)

	if err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, dhcp); err != nil {
		return fmt.Errorf("failed to serialize DHCP response: %w", err)
	}

	return h.stack.SendRawPacket(buf.Bytes())
}
