package config

import (
	"net"
	"net/netip"
	"strings"
)

// DeviceInterfaceAddress resolves an explicit interface prefix before its
// same-index device IP address. Additional device addresses become host routes.
func DeviceInterfaceAddress(device Device, index int) netip.Prefix {
	if index < 0 {
		return netip.Prefix{}
	}
	if index < len(device.Interfaces) {
		if prefix, err := netip.ParsePrefix(strings.TrimSpace(device.Interfaces[index].Address)); err == nil {
			return prefix
		}
	}
	if index < len(device.IPAddresses) {
		if address, ok := netip.AddrFromSlice(device.IPAddresses[index]); ok {
			address = address.Unmap()
			return netip.PrefixFrom(address, address.BitLen())
		}
	}
	return netip.Prefix{}
}

// ValidIPv4Host excludes subnet network and broadcast addresses, except on
// point-to-point and host prefixes where those endpoints are usable.
func ValidIPv4Host(prefix netip.Prefix, address netip.Addr) bool {
	if !prefix.IsValid() || !prefix.Addr().Is4() || !prefix.Contains(address) {
		return false
	}
	const pointToPointBoundary = 30
	if prefix.Bits() > pointToPointBoundary {
		return true
	}
	network := prefix.Masked().Addr().As4()
	host := address.As4()
	if host == network {
		return false
	}
	mask := net.CIDRMask(prefix.Bits(), address.BitLen())
	broadcast := network
	for index := range broadcast {
		broadcast[index] |= ^mask[index]
	}
	return host != broadcast
}
