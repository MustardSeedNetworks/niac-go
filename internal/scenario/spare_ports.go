package scenario

import (
	"fmt"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// AP-3: a real access switch is never fully patched. Every access and server
// switch therefore declares a block of unoccupied access ports -- a pool, not
// one port -- and that pool is where a tester arrives.
//
// The ports have to be authored rather than implied: a device declares only the
// interfaces its links create plus its own SVI, so an undeclared port cannot be
// named by an attachment at all (fabric reports unknown_attachment_port).
//
// Where the block sits is fixed by the rest of the port plan, and the ordering
// is deliberate: access points take TenGigabitEthernet1/0/1..9, wired clients
// take GigabitEthernet1/0/10 upwards, the ring and chain access layers own
// TenGigabitEthernet1/0/47 and /48, and uplinks start at 49. The spare block is
// the last space between the client range and the ring ports, which is why
// maxWorkstationsPerAccess is bounded by sparePortFirst rather than by the
// platform's 48 ports.
const (
	sparePortCount = 4
	sparePortFirst = 43
)

// sparePortNames are the pool ports on one switch, in ascending port order so
// two generations of the same request are byte-identical.
func sparePortNames(prefix string) []string {
	names := make([]string, 0, sparePortCount)
	for offset := range sparePortCount {
		names = append(names, fmt.Sprintf("%s%d", prefix, sparePortFirst+offset))
	}
	return names
}

// sparePorts authors the pool. A spare port is administratively enabled and
// operationally down with no traffic on it -- "notconnect" on a real switch --
// because nothing is plugged into it until a tester is. AP-2 is what brings one
// up for the client it places there.
//
// It builds on newInterface so the MTU and interface type stay decided in one
// place, then overrides what a patched port has and a spare one does not.
//
// It carries no network and no address, which is not an omission: a layer-2
// access port on a switch that owns only its management SVI has no address of
// its own, and naming a network on it would both duplicate and contradict the
// compiler, which derives the network from the port's VLAN
// (fabric.networkServingVLAN). An interface that names a network must carry an
// IPv4 prefix, so authoring one here made every pack fail to compile.
func sparePorts(prefix string, vlan, speed int) []converter.Interface {
	ports := make([]converter.Interface, 0, sparePortCount)
	for _, name := range sparePortNames(prefix) {
		port := newInterface(name, "", "", speed, "Spare access port")
		port.OperStatus = "down"
		port.InUtilization = 0
		port.OutUtilization = 0
		port.VLANs = []int{vlan}
		ports = append(ports, port)
	}
	return ports
}
