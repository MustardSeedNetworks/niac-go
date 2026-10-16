package protocols

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"net/netip"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// Offsets into an IPv4 header, from the end of its Ethernet header.
const (
	ipv4TTLOffset      = 8
	ipv4ChecksumOffset = 10
	ipv4MinHeaderLen   = 20
	ipv4IHLMask        = 0x0f
	ipv4IHLWordLen     = 4
)

var errFabricTTLExpired = errors.New("routed IPv4 packet TTL expired before the attachment")

// forwardIPv4 is the simulated routers forwarding an untagged IPv4 frame out
// of the fabric: a packet from a host behind them leaves with its TTL
// decremented once per router it crosses, as from real hardware. A frame that
// changes is returned as a copy, because a looping packet resends its buffer.
func (r *fabricRuntime) forwardIPv4(frame []byte) ([]byte, error) {
	if len(frame) < ethHeaderLen+ipv4MinHeaderLen ||
		binary.BigEndian.Uint16(frame[ethMACsLen:]) != EtherTypeIP {
		return frame, nil
	}
	header := frame[ethHeaderLen:]
	headerLen := int(header[0]&ipv4IHLMask) * ipv4IHLWordLen
	if headerLen < ipv4MinHeaderLen || headerLen > len(header) {
		return frame, nil
	}
	source := netip.AddrFrom4([SizeOfIP]byte(header[ipv4SourceOffset:ipv4SourceEnd]))
	hops := r.forwardingHops(source, net.HardwareAddr(frame[SizeOfMac:ethMACsLen]))
	if hops == 0 {
		return frame, nil
	}
	if int(header[ipv4TTLOffset]) <= hops {
		return nil, errFabricTTLExpired
	}
	forwarded := bytes.Clone(frame)
	header = forwarded[ethHeaderLen : ethHeaderLen+headerLen]
	header[ipv4TTLOffset] -= safeconv.Uint8(hops)
	binary.BigEndian.PutUint16(header[ipv4ChecksumOffset:], 0)
	binary.BigEndian.PutUint16(header[ipv4ChecksumOffset:], CalculateIPChecksum(header))
	return forwarded, nil
}

// forwardingHops counts the simulated routers between source and the
// attachment, entering at the attachment router whose MAC the frame carries.
// It is zero for a source on the attachment network or on that router itself.
func (r *fabricRuntime) forwardingHops(source netip.Addr, routerMAC net.HardwareAddr) int {
	origin, exists := r.endpointForAddress(source)
	if !exists || origin.network == r.attachmentNetwork {
		return 0
	}
	router, route := r.routeFor(source, routerMAC)
	if router == nil || router.device == origin.device {
		return 0
	}
	hops := 1
	// A static-route loop never reaches source; the bound ends the walk.
	for range len(r.devicesByName) {
		if route.connected {
			break
		}
		next, found := r.endpointForAddress(route.nextHop)
		if !found || next.device == origin.device {
			break
		}
		if route, found = r.bestRoute(next.device, source); !found {
			break
		}
		hops++
	}
	return hops
}

// bestRoute is device's longest available route to dst.
func (r *fabricRuntime) bestRoute(device *config.Device, dst netip.Addr) (fabricRoute, bool) {
	bestBits := -1
	var best fabricRoute
	for _, route := range r.routesFor(device) {
		if route.destination.Bits() > bestBits && route.destination.Contains(dst) &&
			r.routeAvailable(device, route) {
			bestBits = route.destination.Bits()
			best = route
		}
	}
	return best, bestBits >= 0
}
