package config

import "github.com/MustardSeedNetworks/niac-go/internal/converter"

func convertNetworks(in []converter.Network) []Network {
	out := make([]Network, len(in))
	for i, network := range in {
		out[i] = Network{
			Name: network.Name, Subnet: network.Subnet, VirtualVLAN: network.VirtualVLAN,
		}
	}
	return out
}

func convertLogicalAttachments(in []converter.LogicalAttachment) []LogicalAttachment {
	out := make([]LogicalAttachment, len(in))
	for i, attachment := range in {
		out[i] = LogicalAttachment{Name: attachment.Name, Network: attachment.Connect}
	}
	return out
}

func convertRoutes(in []converter.Route) []Route {
	out := make([]Route, len(in))
	for i, route := range in {
		out[i] = Route{Destination: route.Destination, Via: route.Via, NextHop: route.NextHop}
	}
	return out
}
