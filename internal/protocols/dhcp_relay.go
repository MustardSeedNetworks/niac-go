package protocols

import (
	"net"
	"slices"
)

// serveRelayedScope makes the handler answer from a relayed scope: its pool,
// its network's mask and router, and replies sent from the relay. The
// server's static leases belong to its own network and are not offered here.
func (h *DHCPHandler) serveRelayedScope(relay fabricDHCPRelay) {
	start := net.IP(relay.scope.Start.AsSlice())
	end := net.IP(relay.scope.End.AsSlice())
	h.SetPool(start, end)

	h.mu.Lock()
	defer h.mu.Unlock()
	h.subnetMask = net.IP(slices.Clone(relay.mask))
	h.gateway = net.IP(relay.scope.Router.AsSlice())
	h.staticLeases = nil
	h.relayIP = net.IP(relay.scope.Relay.Address.AsSlice())
	h.relayMAC = slices.Clone(relay.mac)
}

// replySource is the giaddr, source address and source MAC of a reply. A
// relayed reply keeps the giaddr the request carried and reaches the client
// from the relay's interface (RFC 2131 §4.1). Caller must hold h.mu.
func (h *DHCPHandler) replySource(serverIP net.IP, serverMAC net.HardwareAddr) (net.IP, net.IP, net.HardwareAddr) {
	if h.relayIP == nil {
		return net.IPv4zero, serverIP, serverMAC
	}
	return h.relayIP, h.relayIP, h.relayMAC
}
