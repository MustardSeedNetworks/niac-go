package protocols

import (
	"cmp"
	"encoding/binary"
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// stpPosition is where one bridge sits in its spanning tree: the root it
// agrees on, what it costs to reach it, and the port it reaches it through and
// the bridge at the other end of that port (both zero on the root itself). The BPDUs a bridge originates and its dot1dStp
// scalars both read this, so a consumer comparing the two sees one tree.
type stpPosition struct {
	root          uint64
	cost          uint32
	rootInterface string
	upstream      uint64
}

func (p stpPosition) snmp() snmp.SpanningTreePosition {
	tree := snmp.SpanningTreePosition{
		Root:          binary.BigEndian.AppendUint64(nil, p.root),
		Cost:          int(p.cost),
		RootInterface: p.rootInterface,
	}
	if p.rootInterface != "" {
		tree.Upstream = binary.BigEndian.AppendUint64(nil, p.upstream)
	}
	return tree
}

type stpLink struct {
	local  string
	remote *config.Device
}

// electSpanningTree runs the 802.1D election over the STP-enabled devices of
// one device table, joined by the trunks they author towards each other. Each
// connected set of bridges elects the lowest bridge ID as its root; every
// other bridge takes the port on its cheapest path there, ties going to the
// lower upstream bridge ID and then the lower local interface name. Every link
// costs the same, the path cost dot1dStpPortPathCost reports.
func electSpanningTree(devices []config.Device) map[*config.Device]stpPosition {
	byName := make(map[string]*config.Device)
	for i := range devices {
		if stpEnabled(&devices[i]) {
			byName[devices[i].Name] = &devices[i]
		}
	}

	links := make(map[*config.Device][]stpLink, len(byName))
	for _, device := range byName {
		for _, trunk := range device.TrunkPorts {
			remote, ok := byName[trunk.RemoteDevice]
			if !ok || trunk.FDBOnly || remote == device {
				continue
			}
			links[device] = append(links[device], stpLink{local: trunk.Interface, remote: remote})
			links[remote] = append(links[remote], stpLink{local: trunk.RemoteInterface, remote: device})
		}
	}

	positions := make(map[*config.Device]stpPosition, len(byName))
	for _, bridge := range sortedByBridgeID(byName) {
		if _, placed := positions[bridge]; placed {
			continue
		}
		placeTree(bridge, links, positions)
	}
	return positions
}

// placeTree places every bridge reachable from root, which is the lowest
// bridge ID of its component because bridges are visited in bridge ID order.
func placeTree(root *config.Device, links map[*config.Device][]stpLink, positions map[*config.Device]stpPosition) {
	rootID := stpBridgeID(root)
	hops := map[*config.Device]uint32{root: 0}
	order := []*config.Device{root}
	for i := 0; i < len(order); i++ {
		for _, link := range links[order[i]] {
			if _, seen := hops[link.remote]; !seen {
				hops[link.remote] = hops[order[i]] + 1
				order = append(order, link.remote)
			}
		}
	}

	positions[root] = stpPosition{root: rootID}
	for _, bridge := range order[1:] {
		link := rootLink(links[bridge], hops, hops[bridge]-1)
		positions[bridge] = stpPosition{
			root:          rootID,
			cost:          hops[bridge] * snmp.STPPortPathCostDefault,
			rootInterface: link.local,
			upstream:      stpBridgeID(link.remote),
		}
	}
}

// rootLink picks the link towards the upstream bridge with the lowest bridge
// ID one hop closer to the root.
func rootLink(links []stpLink, hops map[*config.Device]uint32, upstreamHops uint32) *stpLink {
	var best *stpLink
	for i := range links {
		link := &links[i]
		if hops[link.remote] != upstreamHops {
			continue
		}
		if best == nil || stpBridgeID(link.remote) < stpBridgeID(best.remote) ||
			(link.remote == best.remote && link.local < best.local) {
			best = link
		}
	}
	return best
}

func sortedByBridgeID(byName map[string]*config.Device) []*config.Device {
	bridges := make([]*config.Device, 0, len(byName))
	for _, device := range byName {
		bridges = append(bridges, device)
	}
	slices.SortFunc(bridges, func(a, b *config.Device) int {
		return cmp.Compare(stpBridgeID(a), stpBridgeID(b))
	})
	return bridges
}

func stpEnabled(device *config.Device) bool {
	return device.STPConfig != nil && device.STPConfig.Enabled && len(device.MACAddress) > 0
}

// stpBridgeID is the bridge priority in the top two octets over the bridge MAC.
func stpBridgeID(device *config.Device) uint64 {
	priority := uint16(stpDefaultBridgePri)
	if device.STPConfig != nil && device.STPConfig.BridgePriority > 0 {
		priority = device.STPConfig.BridgePriority
	}
	return makeBridgeID(priority, device.MACAddress)
}
