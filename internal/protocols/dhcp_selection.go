package protocols

import (
	"net"
	"slices"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func (h *DHCPHandler) acceptsDHCPRequest(info *dhcpPacketInfo) bool {
	var serverID net.IP
	identifiers := 0
	for _, option := range info.dhcp.Options {
		if option.Type == layers.DHCPOptServerID {
			identifiers++
			if len(option.Data) != net.IPv4len {
				return false
			}
			serverID = net.IP(option.Data)
		}
	}
	if identifiers > 0 {
		return identifiers == 1 && !serverID.IsUnspecified() && !serverID.IsMulticast() &&
			!serverID.Equal(net.IPv4bcast) && serverID.Equal(h.configuredServerIP())
	}
	if info.messageType == DHCPDecline || info.messageType == DHCPRelease {
		return false
	}
	if info.messageType != DHCPRequest {
		return true
	}
	address := getRequestedIP(info.dhcp)
	if address == nil {
		address = info.dhcp.ClientIP
	}
	if len(address) == 0 || address.IsUnspecified() {
		return false
	}
	// Without a selected server, peers must not NAK addresses outside their
	// own authority. Another server on this broadcast domain may own them.
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isIPInPool(address) || h.matchStaticLease(info.dhcp.ClientHWAddr).Equal(address)
}

func cloneDHCPLease(lease *DHCPLease) *DHCPLease {
	result := *lease
	result.IP = slices.Clone(lease.IP)
	result.MAC = slices.Clone(lease.MAC)
	return &result
}

func dhcpStateAddress(network devicestate.Network) net.IP {
	for _, iface := range network.Interfaces {
		address := iface.Address.Addr().Unmap()
		if iface.AdminUp && iface.OperUp && address.Is4() {
			return net.IP(address.AsSlice())
		}
	}
	return nil
}
