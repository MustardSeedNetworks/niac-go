package scenario_test

import (
	"bytes"
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// A site's DHCP server leases from the same data network its wired endpoints
// are statically addressed on, and nothing at runtime checks one against the
// other: a tester would be offered a simulated endpoint's address and answer
// ARP for it alongside the device. The pool has to sit clear of every static
// address the generator places, at the per-site endpoint ceiling as well as in
// the shipped packs.
func TestNoStaticAddressFallsInADHCPPool(t *testing.T) {
	requests := map[string]scenario.Request{}
	for _, pack := range scenario.Packs() {
		requests[pack.ID] = pack.Request
	}
	requests["site endpoint ceiling"] = siteEndpointCeilingRequest()

	for label, request := range requests {
		cfg := generatedRequest(t, label, request).Config
		pools := dhcpPools(cfg)
		if len(pools) == 0 {
			t.Errorf("%s: no DHCP pool, so this test asserts nothing", label)
		}
		for _, device := range cfg.Devices {
			for _, address := range staticAddresses(device) {
				for _, pool := range pools {
					if pool.contains(address) {
						t.Errorf("%s: %s %s is inside %s's pool %s-%s",
							label, device.Name, address, pool.server, pool.start, pool.end)
					}
				}
			}
		}
	}
}

// siteEndpointCeilingRequest is a single site at the generator's per-site
// endpoint ceiling, so the pool is checked against the highest static address
// a request can place, not only against the packs.
func siteEndpointCeilingRequest() scenario.Request {
	const (
		accessSwitches = 20
		perAccess      = 9
	)
	request := scenario.EnterpriseReferenceRequest()
	request.Sites = request.Sites[:1]
	request.Counts.AccessSwitches = accessSwitches
	request.Counts.WorkstationsPerAccess = perAccess
	return request
}

type dhcpPool struct {
	server     string
	start, end net.IP
}

func (pool dhcpPool) contains(address net.IP) bool {
	address = address.To4()

	return address != nil &&
		bytes.Compare(address, pool.start.To4()) >= 0 &&
		bytes.Compare(address, pool.end.To4()) <= 0
}

func dhcpPools(cfg *config.Config) []dhcpPool {
	var pools []dhcpPool
	for _, device := range cfg.Devices {
		if device.DHCPConfig == nil || device.DHCPConfig.PoolStart == nil {
			continue
		}
		pools = append(pools, dhcpPool{
			server: device.Name, start: device.DHCPConfig.PoolStart, end: device.DHCPConfig.PoolEnd,
		})
	}

	return pools
}

// staticAddresses is every address device holds, each once: a single-homed
// endpoint carries its address both as a device IP and on its interface.
func staticAddresses(device config.Device) []net.IP {
	seen := map[string]bool{}
	var addresses []net.IP
	add := func(address net.IP) {
		if !seen[address.String()] {
			seen[address.String()] = true
			addresses = append(addresses, address)
		}
	}
	for _, address := range device.IPAddresses {
		add(address)
	}
	for _, iface := range device.Interfaces {
		if address, _, err := net.ParseCIDR(iface.Address); err == nil {
			add(address)
		}
	}

	return addresses
}
