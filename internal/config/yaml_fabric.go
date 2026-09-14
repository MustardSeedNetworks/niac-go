package config

import (
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

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
		out[i] = LogicalAttachment{
			Name:    attachment.Name,
			Network: attachment.Connect,
			At:      convertAttachmentPort(attachment.At),
			Pins:    convertAttachmentPins(attachment.Pins),
		}
	}
	return out
}

func convertAttachmentPort(in *converter.AttachmentPort) *AttachmentPort {
	if in == nil {
		return nil
	}
	return &AttachmentPort{Device: in.Device, Ports: slices.Clone(in.Ports)}
}

func convertAttachmentPins(in []converter.AttachmentPin) []AttachmentPin {
	if len(in) == 0 {
		return nil
	}
	out := make([]AttachmentPin, len(in))
	for i, pin := range in {
		out[i] = AttachmentPin{MAC: pin.MAC, Device: pin.Device, Interface: pin.Interface}
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
