package snmp

import (
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// ifStackTable (RFC 2863 §6) says which interface runs over which. Its one
// readable column is indexed by ifStackHigherLayer.ifStackLowerLayer, and
// ifIndex 0 stands for "nothing": 0.N means nothing runs over N, N.0 means N
// runs over nothing.
const (
	ifStackStatus     = ifMIBObjects + ".2.1.3"
	ifStackLastChange = ifMIBObjects + ".6.0"
)

// portBundle is one authored port-channel resolved to the ifTable names of its
// aggregate and member ports.
type portBundle struct {
	aggregate string
	members   []string
}

// portChannelBundles resolves the device's port-channels against the names it
// already lists. The schema names a bundle port-channel<id>, but a trunk or an
// authored interface may spell it Port-channel1, and one bundle must not become
// two ifTable rows.
func portChannelBundles(device *config.Device, listed []string) []portBundle {
	bundles := make([]portBundle, 0, len(device.PortChannels))
	for _, channel := range device.PortChannels {
		aggregate := channel.Name()
		for _, name := range listed {
			if strings.EqualFold(name, aggregate) {
				aggregate = name
				break
			}
		}
		bundles = append(bundles, portBundle{aggregate: aggregate, members: channel.Members})
	}
	return bundles
}

// withPortBundles appends the bundles' members and aggregates the listed names
// lack. Appending keeps every earlier ifIndex where SynthesizedInterfaceOrder
// put it.
func withPortBundles(names []string, bundles []portBundle) []string {
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		seen[name] = struct{}{}
	}
	for _, bundle := range bundles {
		for _, member := range bundle.members {
			appendUniqueInterface(&names, seen, member)
		}
		appendUniqueInterface(&names, seen, bundle.aggregate)
	}
	return names
}

// bundleSpeedBps is the aggregate's bandwidth: the sum of its members, which is
// what a switch reports for a port-channel whose members are all up.
func bundleSpeedBps(device *config.Device, bundle portBundle) uint64 {
	var total uint64
	for _, member := range bundle.members {
		total += authoredSpeedBps(device, member)
	}
	return total
}

func authoredSpeedBps(device *config.Device, name string) uint64 {
	for _, iface := range device.Interfaces {
		if iface.Name == name && iface.Speed > 0 {
			return uint64(iface.Speed) * MicrosPerSec
		}
	}
	return getInterfaceSpeed(name)
}

// initializeIfStackTable publishes a complete stack for a device that bundles
// ports: each aggregate over its members, and a 0 row on every open end. A
// device with no bundle has no layering worth publishing, and leaves the table
// out as most agents do.
func (a *Agent) initializeIfStackTable(names []string, bundles []portBundle) {
	if len(bundles) == 0 {
		return
	}
	index := make(map[string]int, len(names))
	for position, name := range names {
		index[name] = position + 1
	}
	hasHigher := make(map[int]bool)
	hasLower := make(map[int]bool)
	for _, bundle := range bundles {
		aggregate := index[bundle.aggregate]
		for _, member := range bundle.members {
			lower := index[member]
			a.setIfStackRow(aggregate, lower)
			hasHigher[lower] = true
			hasLower[aggregate] = true
		}
	}
	for _, ifIdx := range index {
		if !hasHigher[ifIdx] {
			a.setIfStackRow(0, ifIdx)
		}
		if !hasLower[ifIdx] {
			a.setIfStackRow(ifIdx, 0)
		}
	}
	a.mib.Set(ifStackLastChange, &OIDValue{Type: gosnmp.TimeTicks, Value: uint32(0)})
}

func (a *Agent) setIfStackRow(higher, lower int) {
	a.mib.Set(
		ifStackStatus+"."+strconv.Itoa(higher)+"."+strconv.Itoa(lower),
		&OIDValue{Type: gosnmp.Integer, Value: rowStatusActive},
	)
}
