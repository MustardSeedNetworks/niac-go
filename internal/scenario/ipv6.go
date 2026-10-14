package scenario

import (
	"fmt"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// Packs are dual-stack (niac-go#2411). Each site takes a /48 from the
// documentation range (RFC 3849) keyed by its IPv4 second octet, a /64 per VLAN
// keyed by the VLAN's third octet, and a /64 per transit. An interface's host
// ID repeats its IPv4 host number, so 10.51.210.2 is 2001:db8:51:210::2. The
// lab edge and provider-facing lab networks stay IPv4.

const (
	// coreRAPeriod and coreRALifetime are IOS XE's defaults for
	// `ipv6 nd ra interval` and `ipv6 nd ra lifetime`.
	coreRAPeriod   = 200
	coreRALifetime = 1800
)

func siteIPv6Prefix(site Site) string {
	return fmt.Sprintf("2001:db8:%d::/48", site.Octet)
}

func siteSubnetV6(site Site, thirdOctet int) string {
	return fmt.Sprintf("2001:db8:%d:%d::/64", site.Octet, thirdOctet)
}

func siteIPv6(site Site, thirdOctet, host int) string {
	return fmt.Sprintf("2001:db8:%d:%d::%d", site.Octet, thirdOctet, host)
}

// transitV6 is a site transit's /64 and the address of host on it, where host
// is the IPv4 address's offset into its /29. The subnet IDs sit above every
// VLAN's third octet.
func transitV6(site Site, kind string) (string, func(host int) string) {
	var subnetID string
	switch kind {
	case "wan":
		subnetID = "ff00"
	case "security":
		subnetID = "ff01"
	case "core":
		subnetID = "ff02"
	default:
		panic("invalid transit kind: " + kind)
	}
	prefix := fmt.Sprintf("2001:db8:%d:%s::", site.Octet, subnetID)
	return prefix + "/64", func(host int) string { return fmt.Sprintf("%s%d", prefix, host) }
}

func withAddressV6(iface converter.Interface, address string) converter.Interface {
	iface.AddressV6 = address + "/64"
	return iface
}

// coreICMPv6 makes a core advertise itself as the default router on every VLAN
// it routes; each advertisement carries the client VLAN's /64.
func coreICMPv6() *converter.Icmpv6Config {
	return &converter.Icmpv6Config{
		Enabled: true,
		RouterAdvertisement: &converter.Icmpv6RouterAdvertisement{
			Period: coreRAPeriod, Lifetime: coreRALifetime,
		},
	}
}
