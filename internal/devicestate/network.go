package devicestate

import "net/netip"

// Interface is mutable runtime state for one simulated device interface.
type Interface struct {
	Name        string
	Network     string
	Address     netip.Prefix
	Description string
	VLANs       []int
	AdminUp     bool
	OperUp      bool
}

// Route is one route installed in a simulated device's forwarding table.
type Route struct {
	Destination netip.Prefix
	Via         string
	NextHop     netip.Addr
	Connected   bool
}

// Network contains interface and routing state for one simulated device.
type Network struct {
	Interfaces []Interface
	Routes     []Route
	VLANs      []VLAN
	Routers    []Router
}

// VLAN is one configured virtual LAN.
type VLAN struct {
	ID     int
	Name   string
	Active bool
}

// RouterNetwork is one routing-process network statement.
type RouterNetwork struct {
	Address  string
	Wildcard string
	Area     string
}

// Router is one configured routing process.
type Router struct {
	Protocol  string
	ProcessID string
	Networks  []RouterNetwork
}

func cloneNetwork(network Network) Network {
	result := Network{
		Interfaces: append([]Interface(nil), network.Interfaces...),
		Routes:     append([]Route(nil), network.Routes...),
		VLANs:      append([]VLAN(nil), network.VLANs...),
		Routers:    append([]Router(nil), network.Routers...),
	}
	for index := range result.Routers {
		result.Routers[index].Networks = append([]RouterNetwork(nil), result.Routers[index].Networks...)
	}
	for index := range result.Interfaces {
		result.Interfaces[index] = cloneInterface(result.Interfaces[index])
	}

	return result
}
