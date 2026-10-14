package protocols

import (
	"bytes"
	"net"
	"sync"
	"time"
)

// observedClientTTL ages an entry out when its client has been silent for
// that long. It mirrors the IEEE 802.1D default bridge ageing time, which is
// the semantic an operator already has for "this MAC is still on the wire" —
// the same 300 s the FDB tables this table sits beside use.
const observedClientTTL = 300 * time.Second

// ObservedClient is one MAC seen sourcing traffic into the simulation.
//
// The simulation used to learn a client only from a DHCP ACK or an inbound
// LLDP/CDP/EDP/FDP frame, so a tester with a static address that speaks
// neither was attached and invisible. Every received frame the stack accepts
// contributes here instead, which makes this the answer to "who is attached to
// this scenario right now".
//
// Device and Interface name the pool port the client is plugged into. Both are
// empty when the session's attachment names a network rather than a pool, or
// when every port of the pool was already taken.
type ObservedClient struct {
	MAC       string `json:"mac"`
	IP        string `json:"ip,omitempty"`
	Device    string `json:"device,omitempty"`
	Interface string `json:"interface,omitempty"`
	// VLAN is the tag the client's frames carried, absent when untagged.
	VLAN      int       `json:"vlan,omitempty"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
	ExpireAt  time.Time `json:"expireAt"`
	Frames    uint64    `json:"frames"`
}

type observedClientTable struct {
	mu      sync.RWMutex
	entries map[string]*ObservedClient
	now     func() time.Time
}

func newObservedClientTable() *observedClientTable {
	return &observedClientTable{
		entries: make(map[string]*ObservedClient),
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// observe records one frame from mac. ip is the address the frame claimed, or
// nil when it claimed none; a client that has not been given an address yet
// keeps an empty IP rather than the 0.0.0.0 its DHCP DISCOVER sourced. A
// dual-stack client sources neighbour discovery from its IPv6 link-local
// address between the frames it sends from its lease, so a claim never
// replaces a better one: an IPv4 address beats a routable IPv6 address, which
// beats a link-local one (#2651).
func (t *observedClientTable) observe(mac net.HardwareAddr, ip net.IP, vlan int) {
	key := mac.String()
	now := t.now()

	t.mu.Lock()
	defer t.mu.Unlock()

	entry, seen := t.entries[key]
	if !seen {
		entry = &ObservedClient{MAC: key, FirstSeen: now}
		t.entries[key] = entry
	}

	entry.LastSeen = now
	entry.ExpireAt = now.Add(observedClientTTL)
	entry.Frames++
	entry.VLAN = vlan

	if len(ip) > 0 && rankOf(ip) >= rankOf(net.ParseIP(entry.IP)) {
		entry.IP = ip.String()
	}
}

// addressRank orders the addresses a client can claim by how well each
// identifies it on the scenario's networks.
type addressRank int

const (
	rankNone addressRank = iota
	rankIPv6LinkLocal
	rankIPv6Routable
	rankIPv4
)

func rankOf(ip net.IP) addressRank {
	switch {
	case ip == nil:
		return rankNone
	case ip.To4() != nil:
		return rankIPv4
	case ip.IsLinkLocalUnicast():
		return rankIPv6LinkLocal
	default:
		return rankIPv6Routable
	}
}

func (t *observedClientTable) cleanupExpired() {
	now := t.now()

	t.mu.Lock()
	defer t.mu.Unlock()

	for key, entry := range t.entries {
		if now.After(entry.ExpireAt) {
			delete(t.entries, key)
		}
	}
}

func (t *observedClientTable) reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.entries = make(map[string]*ObservedClient)
}

func (t *observedClientTable) list() []ObservedClient {
	t.mu.RLock()
	defer t.mu.RUnlock()

	out := make([]ObservedClient, 0, len(t.entries))
	for _, entry := range t.entries {
		out = append(out, *entry)
	}

	return out
}

// GetObservedClients returns a snapshot of the clients seen on the wire during
// this session.
func (s *Stack) GetObservedClients() []ObservedClient {
	if s == nil || s.observedClients == nil {
		return nil
	}

	clients := s.observedClients.list()

	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()

	if s.fabric == nil || s.fabric.placement == nil {
		return clients
	}
	for i := range clients {
		if port, ok := s.fabric.placement.lookup(clients[i].MAC); ok {
			clients[i].Device = port.Device
			clients[i].Interface = port.Interface
		}
	}

	return clients
}

// recordObservedClient learns the source of one accepted frame. It runs after
// the fabric and VLAN confinement checks and before routing: a frame the
// simulation rejects as out of scope is not something attached to it, and a
// frame no handler happens to answer still came from a client that is.
func (s *Stack) recordObservedClient(pkt *Packet) {
	if s.observedClients == nil || pkt == nil {
		return
	}

	src := pkt.GetSourceMAC()
	if !isUnicastMAC(src) || isLinkControlDestination(pkt.GetDestMAC()) {
		return
	}

	// libpcap reports the simulation's own outbound frames on the same
	// handle, so without this the table fills with the scenario itself.
	if s.devicesFor(pkt.VLAN).GetByMAC(src) != nil {
		return
	}
	// Nor is the host's own NIC something plugged into the scenario.
	if s.fabric != nil && bytes.Equal(src, s.fabric.hostMAC) {
		return
	}

	// Packet.VLAN is -1 for an untagged frame, a parser sentinel rather than a
	// VLAN; reporting it would tell a consumer the client sits on VLAN -1.
	vlan := 0
	if pkt.VLANTagged {
		vlan = pkt.VLAN
	}

	s.observedClients.observe(src, sourceAddressClaimedBy(pkt), vlan)
	s.placeObservedClient(src)
}

// isLinkControlDestination reports whether a frame to dst is control traffic
// for the link itself: STP, LLDP, LACP and 802.1X to the IEEE reserved group
// 01-80-C2-00-00-00..0F, and the vendor discovery and per-VLAN spanning tree
// groups (CDP, PVST+, EDP, FDP). A switch consumes these in its control plane
// and never learns their source in its forwarding table, so a sender of only
// these is the neighbour on the far end of a trunk, such as the uplink switch
// whose BPDUs took a tester port in #2651, not a client plugged into a port.
func isLinkControlDestination(dst net.HardwareAddr) bool {
	if len(dst) != macAddrLen {
		return false
	}
	if bytes.Equal(dst[:5], []byte{0x01, 0x80, 0xc2, 0x00, 0x00}) && dst[5] <= 0x0f {
		return true
	}

	switch dst.String() {
	case "01:00:0c:cc:cc:cc", // CDP, VTP, DTP, PAgP, UDLD
		"01:00:0c:cc:cc:cd", // PVST+
		"00:e0:2b:00:00:00", // EDP
		"01:e0:52:cc:cc:cc": // FDP
		return true
	default:
		return false
	}
}

// isUnicastMAC rejects the all-zero address and any group address: neither can
// be a client's own source address.
func isUnicastMAC(mac net.HardwareAddr) bool {
	if len(mac) != macAddrLen {
		return false
	}

	if mac[0]&0x01 != 0 {
		return false
	}

	for _, b := range mac {
		if b != 0 {
			return true
		}
	}

	return false
}

// sourceAddressClaimedBy returns the address the frame's sender claimed as its
// own, or nil when it claimed none. An unspecified address (0.0.0.0 in a DHCP
// DISCOVER, :: in a duplicate-address probe) is a claim of nothing.
func sourceAddressClaimedBy(pkt *Packet) net.IP {
	offset := etherHeaderSize
	if pkt.VLANTagged {
		offset += dot1qTagLen
	}

	var ip net.IP

	switch etherTypeAfterTags(pkt) {
	case EtherTypeARP:
		if pkt.Length >= offset+arpSenderProtocolEnd {
			ip = net.IP(pkt.Buffer[offset+ARPSenderProtocolAddress : offset+arpSenderProtocolEnd])
		}
	case EtherTypeIP:
		if pkt.Length >= offset+ipv4SourceEnd {
			ip = net.IP(pkt.Buffer[offset+ipv4SourceOffset : offset+ipv4SourceEnd])
		}
	case EtherTypeIPv6:
		if pkt.Length >= offset+ipv6SourceEnd {
			ip = net.IP(pkt.Buffer[offset+ipv6SourceOffset : offset+ipv6SourceEnd])
		}
	}

	if ip == nil || ip.IsUnspecified() {
		return nil
	}

	return ip
}

func etherTypeAfterTags(pkt *Packet) uint16 {
	if pkt.VLANTagged {
		return pkt.Get16(vlanEtherTypeOffset)
	}

	return pkt.GetEtherType()
}

// Offsets into the layer-3 header of a frame, from the end of its Ethernet
// (and 802.1Q) header.
const (
	arpSenderProtocolEnd = ARPSenderProtocolAddress + SizeOfIP
	ipv4SourceOffset     = 12
	ipv4SourceEnd        = ipv4SourceOffset + SizeOfIP
	ipv6SourceOffset     = 8
	ipv6SourceEnd        = ipv6SourceOffset + SizeOfIPv6
)
