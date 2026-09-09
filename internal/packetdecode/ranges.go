package packetdecode

import (
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// ByteRange identifies a decoded field in the original frame, using an exclusive end.
// An empty Field identifies the complete layer header, including its options.
type ByteRange struct {
	Layer string `json:"layer"`
	Field string `json:"field"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

func decodedRanges(packet gopacket.Packet) []ByteRange {
	result := []ByteRange{}
	seen := make(map[gopacket.LayerType]bool)
	offset := 0
	for _, layer := range packet.Layers() {
		size := len(layer.LayerContents())
		// Enrich reads packet.Layer, which selects the first occurrence.
		if !seen[layer.LayerType()] {
			result = append(result, layerRanges(layer, offset)...)
			seen[layer.LayerType()] = true
		}
		offset += size
	}
	return result
}

func layerRanges(layer gopacket.Layer, offset int) []ByteRange {
	size := len(layer.LayerContents())
	name, fields := rangeFields(layer, size)
	if name == "" {
		return nil
	}
	result := []ByteRange{{Layer: name, Start: offset, End: offset + size}}
	for _, field := range fields {
		if field.End > size || field.End <= field.Start {
			continue
		}
		result = append(
			result,
			ByteRange{Layer: name, Field: field.Field, Start: offset + field.Start, End: offset + field.End},
		)
	}
	return result
}

func rangeFields(layer gopacket.Layer, size int) (string, []ByteRange) {
	const (
		macEnd          = 6
		macPairEnd      = 12
		ethernetEnd     = 14
		vlanControlEnd  = 2
		vlanEnd         = 4
		ipv4Source      = 12
		ipv4Destination = 16
		ipv4End         = 20
		ipv4TTL         = 8
		ipv4Protocol    = 9
		ipv4Checksum    = 10
		ipv6Source      = 8
		ipv6Destination = 24
		ipv6End         = 40
		ipv6HopLimit    = 7
		ipv6NextHeader  = 6
	)
	switch layer.LayerType() {
	case layers.LayerTypeEthernet:
		return "ethernet", []ByteRange{
			{Field: "Destination MAC", End: macEnd},
			{Field: "Source MAC", Start: macEnd, End: macPairEnd},
			{Field: "EtherType", Start: macPairEnd, End: ethernetEnd},
		}
	case layers.LayerTypeDot1Q:
		return "dot1q", []ByteRange{
			{Field: "VLAN", End: vlanControlEnd},
			{Field: "Priority", End: vlanControlEnd},
			{Field: "EtherType", Start: vlanControlEnd, End: vlanEnd},
		}
	case layers.LayerTypeIPv4:
		return "ipv4", []ByteRange{
			{Field: "Source", Start: ipv4Source, End: ipv4Destination},
			{Field: "Destination", Start: ipv4Destination, End: ipv4End},
			{Field: "TTL", Start: ipv4TTL, End: ipv4Protocol},
			{Field: "Protocol", Start: ipv4Protocol, End: ipv4Checksum},
			{Field: "Options", Start: ipv4End, End: size},
		}
	case layers.LayerTypeIPv6:
		return "ipv6", []ByteRange{
			{Field: "Source", Start: ipv6Source, End: ipv6Destination},
			{Field: "Destination", Start: ipv6Destination, End: ipv6End},
			{Field: "Hop Limit", Start: ipv6HopLimit, End: ipv6Source},
			{Field: "Next Header", Start: ipv6NextHeader, End: ipv6HopLimit},
		}
	default:
		return transportRangeFields(layer, size)
	}
}

func transportRangeFields(layer gopacket.Layer, size int) (string, []ByteRange) {
	const icmpTypeCodeLength = 2
	const (
		sourcePortEnd     = 2
		portsEnd          = 4
		sequenceEnd       = 8
		acknowledgmentEnd = 12
		flagsStart        = 13
		flagsEnd          = 14
		windowEnd         = 16
		tcpEnd            = 20
		udpLengthEnd      = 6
	)
	switch layer.LayerType() {
	case layers.LayerTypeTCP:
		return "tcp", []ByteRange{
			{Field: "Source Port", End: sourcePortEnd},
			{Field: "Destination Port", Start: sourcePortEnd, End: portsEnd},
			{Field: "Sequence Number", Start: portsEnd, End: sequenceEnd},
			{Field: "Acknowledgment Number", Start: sequenceEnd, End: acknowledgmentEnd},
			{Field: "Flags", Start: flagsStart, End: flagsEnd},
			{Field: "Window Size", Start: flagsEnd, End: windowEnd},
			{Field: "Options", Start: tcpEnd, End: size},
		}
	case layers.LayerTypeUDP:
		return "udp", []ByteRange{
			{Field: "Source Port", End: sourcePortEnd},
			{Field: "Destination Port", Start: sourcePortEnd, End: portsEnd},
			{Field: "Length", Start: portsEnd, End: udpLengthEnd},
		}
	case layers.LayerTypeICMPv4, layers.LayerTypeICMPv6:
		return "icmp", []ByteRange{{Field: "Type", End: 1}, {Field: "Code", Start: 1, End: icmpTypeCodeLength}}
	case layers.LayerTypeIPv6HopByHop,
		layers.LayerTypeIPv6Destination,
		layers.LayerTypeIPv6Routing,
		layers.LayerTypeIPv6Fragment:
		return layer.LayerType().String(), nil
	default:
		return "", nil
	}
}
