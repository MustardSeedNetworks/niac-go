package scenario

import (
	"hash/fnv"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// A pack's baseline is a healthy network: SNMP counters advance from the
// authored utilization, so a baseline that runs hot makes every interface look
// like a finding and buries the one the pack means to show. Problems come from
// the pack's explicit faults (see pack_findings.go); this file only decides
// what "quiet and normal" looks like for each kind of interface.

// Device roles the baseline tells apart. The rest are server-class (see
// isServerRole) or the endpoint kinds, which never reach managedDevice.
const (
	roleAccessPoint     = "ap"
	roleAccessSwitch    = "access"
	roleServerSwitch    = "server-switch"
	roleCoreSwitch      = "core"
	roleDistribution    = "distribution"
	roleWAN             = "wan"
	roleFirewall        = "firewall"
	roleLabEdge         = "lab"
	managementInterface = "management"
)

// loadClass is a kind of interface with its own normal load.
type loadClass int

const (
	loadEndpoint     loadClass = iota // client NICs, and the switch ports they patch into
	loadAccessPoint                   // radios and the switch port feeding one
	loadServer                        // servers, controllers and the switch ports that serve them
	loadSVI                           // routed gateways, management and loopbacks
	loadAccessUplink                  // closet and server-leaf uplinks, ring and chain links
	loadAggregation                   // core and distribution interconnects
	loadTransit                       // routers, firewalls and the WAN edge
)

// loadBand is an inclusive range of percent utilization.
type loadBand struct{ floor, ceiling uint32 }

// band is what a measured production network carries, not a demo's idea of
// busy. A client port idles in the low single digits, a link aggregating many
// of them runs several times hotter, and nothing in a healthy baseline
// approaches the 80% where Link-Live raises a Warning.
func (class loadClass) band() loadBand {
	switch class {
	case loadEndpoint:
		return loadBand{1, 10}
	case loadAccessPoint:
		return loadBand{5, 15}
	case loadServer:
		return loadBand{5, 25}
	case loadSVI:
		return loadBand{2, 8}
	case loadAccessUplink:
		return loadBand{10, 30}
	case loadAggregation:
		return loadBand{15, 40}
	case loadTransit:
		return loadBand{20, 40}
	}
	panic("unknown load class")
}

const (
	// outShareFloor and outShareSpread make transmit a stable 50-90% of
	// receive, so a port is asymmetric the way real traffic is without a second
	// independent draw escaping the band.
	outShareFloor  = 50
	outShareSpread = 41
	percentScale   = 100
)

// baselineLoad derives one interface's in and out utilization from a seed. It
// is a pure function of the seed, keeping every run reproducible, which the
// Link-Live comparator depends on.
func baselineLoad(seed string, class loadClass) (float64, float64) {
	band := class.band()
	in := band.floor + seedHash(seed)%(band.ceiling-band.floor+1)
	share := outShareFloor + seedHash(seed+"/out")%outShareSpread
	out := max(band.floor, (in*share+percentScale/2)/percentScale)
	return float64(in), float64(out)
}

func seedHash(seed string) uint32 {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(seed))
	return hash.Sum32()
}

// peerOf is what the far end of a linked interface is, as far as load goes.
type peerOf struct {
	role string
	// endpoint is set for an fdbOnly link, whose far end is a host rather than
	// a device the generator names.
	endpoint bool
}

// applyBaselineUtilization sets the load on every interface of a managed
// device that is up. Spare ports are down and stay at zero.
func applyBaselineUtilization(device, role string, interfaces []converter.Interface, links []link) {
	peers := make(map[string]peerOf, len(links))
	for _, peer := range links {
		far := peerOf{endpoint: peer.fdbOnly}
		if !peer.fdbOnly {
			far.role = roleOfDevice(peer.remoteDevice)
		}
		peers[peer.localInterface] = far
		for _, member := range peer.members {
			peers[member] = far
		}
	}
	for index := range interfaces {
		iface := &interfaces[index]
		if iface.OperStatus != "up" {
			continue
		}
		far, linked := peers[iface.Name]
		class := interfaceClass(role, iface.Name, far, linked)
		iface.InUtilization, iface.OutUtilization = baselineLoad(device+"/"+iface.Name, class)
	}
}

func interfaceClass(role, name string, far peerOf, linked bool) loadClass {
	switch {
	case role == roleAccessPoint:
		return loadAccessPoint
	case isServerRole(role):
		return loadServer
	case strings.HasPrefix(name, "Vlan"), strings.HasPrefix(name, "Loopback"), name == managementInterface:
		return loadSVI
	case linked && far.endpoint && role == roleServerSwitch:
		return loadServer
	case linked && far.endpoint:
		return loadEndpoint
	case linked:
		return linkClass(role, far.role)
	}
	return roleClass(role)
}

// linkClass classifies a link between two devices the same from either end, so
// both ports of one cable agree.
func linkClass(local, remote string) loadClass {
	for _, role := range []string{local, remote} {
		switch role {
		case roleAccessPoint:
			return loadAccessPoint
		case roleWAN, roleFirewall, roleLabEdge:
			return loadTransit
		}
	}
	for _, role := range []string{local, remote} {
		if role == roleAccessSwitch || role == roleServerSwitch {
			return loadAccessUplink
		}
	}
	return loadAggregation
}

// roleClass is the load of an interface no link describes, such as a router's
// transit port or a core's firewall-facing uplink.
func roleClass(role string) loadClass {
	switch role {
	case roleWAN, roleFirewall, roleLabEdge:
		return loadTransit
	case roleAccessSwitch, roleServerSwitch:
		return loadAccessUplink
	}
	return loadAggregation
}

func isServerRole(role string) bool {
	switch role {
	case "server", "hypervisor", "storage-array", "backup-server", "controller":
		return true
	}
	return false
}
