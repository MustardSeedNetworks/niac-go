package protocols

import (
	"net"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

func protocolEvent(packet gopacket.Packet, protocol layers.IPProtocol) snmp.ProtocolEvent {
	event := snmp.ProtocolEvent{Protocol: uint8(protocol)}
	if layer := packet.Layer(layers.LayerTypeICMPv4); layer != nil {
		if icmp, ok := layer.(*layers.ICMPv4); ok {
			event.ICMPType = icmp.TypeCode.Type()
		}
	}
	if layer := packet.Layer(layers.LayerTypeTCP); layer != nil {
		if tcp, ok := layer.(*layers.TCP); ok {
			event.TCPSYN, event.TCPACK, event.TCPRST, event.TCPFIN = tcp.SYN, tcp.ACK, tcp.RST, tcp.FIN
			event.SourcePort, event.DestinationPort = uint16(tcp.SrcPort), uint16(tcp.DstPort)
		}
	}

	return event
}

func ipv4ProtocolEvent(packet gopacket.Packet, ip *layers.IPv4) snmp.ProtocolEvent {
	event := protocolEvent(packet, ip.Protocol)
	event.SourceIP, event.DestinationIP = ip.SrcIP.String(), ip.DstIP.String()
	event.MoreFragments = ip.Flags&layers.IPv4MoreFragments != 0
	event.FragmentOffset = ip.FragOffset

	return event
}

func (s *Stack) recordInboundProtocol(pkt *Packet, ip *layers.IPv4, devices []*config.Device) {
	event := ipv4ProtocolEvent(gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.NoCopy), ip)
	for _, device := range devices {
		if group := s.getSNMPAgents(device); group != nil {
			group.telemetry.RecordInbound(event)
			group.telemetry.RecordInterfaceInbound(
				interfaceNameForIP(device, ip.DstIP), pkt.Length,
				frameIsNonUnicast(pkt), frameIsBroadcast(pkt),
			)
		}
	}
	if pkt.fabricFirstHopDevice != nil {
		if group := s.getSNMPAgents(pkt.fabricFirstHopDevice); group != nil {
			group.telemetry.RecordForwarded()
			group.telemetry.RecordInterfaceInbound(
				interfaceNameForIP(pkt.fabricFirstHopDevice, pkt.fabricFirstHopIP),
				pkt.Length,
				frameIsNonUnicast(pkt),
				frameIsBroadcast(pkt),
			)
		}
	}
}

func (s *Stack) recordUDPNoPort(devices []*config.Device) {
	for _, device := range devices {
		if group := s.getSNMPAgents(device); group != nil {
			group.telemetry.RecordUDPNoPort()
		}
	}
}

func (s *Stack) recordOutboundProtocol(pkt *Packet) {
	if pkt == nil {
		return
	}
	device, ok := pkt.Device.(*config.Device)
	if !ok {
		return
	}
	group := s.getSNMPAgents(device)
	if group == nil {
		return
	}
	packet := gopacket.NewPacket(pkt.Buffer[:pkt.Length], layers.LayerTypeEthernet, gopacket.NoCopy)
	layer := packet.Layer(layers.LayerTypeIPv4)
	ip, ok := layer.(*layers.IPv4)
	if !ok {
		return
	}
	group.telemetry.RecordOutbound(ipv4ProtocolEvent(packet, ip))
	group.telemetry.RecordInterfaceOutbound(
		interfaceNameForIP(device, ip.SrcIP), pkt.Length,
		frameIsNonUnicast(pkt), frameIsBroadcast(pkt),
	)
	if pkt.fabricFirstHopDevice != nil && pkt.fabricFirstHopDevice != device {
		if firstHopGroup := s.getSNMPAgents(pkt.fabricFirstHopDevice); firstHopGroup != nil {
			firstHopGroup.telemetry.RecordInterfaceOutbound(
				interfaceNameForIP(pkt.fabricFirstHopDevice, pkt.fabricFirstHopIP), pkt.Length,
				frameIsNonUnicast(pkt), frameIsBroadcast(pkt),
			)
		}
	}
}

func interfaceNameForIP(device *config.Device, target net.IP) string {
	if device == nil || target == nil {
		return ""
	}
	for _, iface := range device.Interfaces {
		ip, _, err := net.ParseCIDR(iface.Address)
		if err == nil && ip.Equal(target) {
			return iface.Name
		}
	}
	return ""
}

func frameIsNonUnicast(pkt *Packet) bool {
	mac := pkt.GetDestMAC()
	return len(mac) > 0 && mac[0]&1 != 0
}

func frameIsBroadcast(pkt *Packet) bool {
	const broadcastOctet = 0xff
	mac := pkt.GetDestMAC()
	if len(mac) == 0 {
		return false
	}
	for _, octet := range mac {
		if octet != broadcastOctet {
			return false
		}
	}
	return true
}
