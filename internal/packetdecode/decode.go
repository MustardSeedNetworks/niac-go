// Package packetdecode turns raw Ethernet frames into the field set both packet
// surfaces render: the live inspector's SSE stream and the uploaded-capture
// analyzer.
//
// It exists because those two had separate decoders that disagreed. The live
// path handled 802.1Q, STP, CDP, LLDP, LLC and SNAP -- including the tagged-LLC
// case gopacket cannot chain on its own -- while the analyzer handled DNS and
// named a protocol by its well-known port. A frame that the inspector called
// "STP" the analyzer called "Unknown", and the pcapng an operator exported from
// the inspector did not read back the way it had just been shown (niac#1798,
// P1c-11). One decoder, so both surfaces agree by construction.
//
// Output keys are the SSE wire spelling (snake_case); the analyzer projects
// them onto its own struct. That direction was chosen deliberately: the live
// stream's shape is the one a browser is already parsing, so making the
// analyzer follow it changes nothing a user is looking at today.
package packetdecode

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// ipv4Len is the byte length of an IPv4 protocol address.
const ipv4Len = 4

// dot1qLLCMinPayload is a length field plus a minimal LLC header.
const dot1qLLCMinPayload = 5

// minEtherType is the first value that is an EtherType rather than a length.
const minEtherType = 0x0600

// arpOperationRequest / arpOperationReply are the two ARP opcodes.
const (
	arpOperationRequest = 1
	arpOperationReply   = 2
)

// icmpTypeEchoReply / icmpTypeEchoRequest are the two ICMP types a tester
// looks for by name.
const (
	icmpTypeEchoReply   = 0
	icmpTypeEchoRequest = 8
)

// headerLayerHint pre-sizes the per-packet headers map: ethernet, dot1q and one
// network layer covers almost every frame NIAC sees.
const headerLayerHint = 4

// Enrich does a best-effort gopacket decode of buf and writes the decoded
// fields into out. Decode errors are swallowed: the caller still has the raw
// bytes, and a frame NIAC cannot name is better shown unnamed than dropped.
func Enrich(out map[string]any, buf []byte) {
	if len(buf) == 0 {
		return
	}
	packet := gopacket.NewPacket(buf, layers.LayerTypeEthernet, gopacket.NoCopy)

	// The Packet Inspector builds its layer tree from a nested `headers` map.
	// Emitting only flat keys meant it never found an ethernet layer and showed
	// "(not parsed)" for the MACs of every packet, not just the odd ones (D16).
	headers := make(map[string]any, headerLayerHint)

	if ethLayer := packet.Layer(layers.LayerTypeEthernet); ethLayer != nil {
		if eth, ok := ethLayer.(*layers.Ethernet); ok {
			out["src_mac"] = eth.SrcMAC.String()
			out["dst_mac"] = eth.DstMAC.String()
			headers["ethernet"] = map[string]any{
				"srcMac":    eth.SrcMAC.String(),
				"dstMac":    eth.DstMAC.String(),
				"etherType": eth.EthernetType.String(),
			}
		}
	}

	enrichVLAN(out, headers, packet)
	enrichNetworkLayer(out, headers, packet)
	enrichTransportLayer(out, headers, packet)
	enrichApplicationLayer(out, packet)
	enrichLinkLayer(out, packet)

	if len(headers) > 0 {
		out["headers"] = headers
	}
}

// enrichApplicationLayer names the application protocol above TCP/UDP.
//
// The live stream reported "UDP 161->1234" for an SNMP GET and "UDP 68->67" for
// a DHCP offer: correct, and useless to anyone reading a NIAC capture, since
// those are the two protocols NIAC exists to answer. The analyzer already named
// them by port and decoded DNS; folding that in here is what makes the two
// surfaces agree, and it is a straight improvement for the live one.
//
// Runs after the transport layer, whose ports it reads, and only ever refines a
// TCP/UDP label into something more specific.
func enrichApplicationLayer(out map[string]any, packet gopacket.Packet) {
	proto, _ := out["protocol"].(string)
	if proto != "TCP" && proto != "UDP" {
		return
	}

	// A real DNS decode beats the port guess: it names the query, and it is
	// right about DNS on a non-standard port.
	if l := packet.Layer(layers.LayerTypeDNS); l != nil {
		if dns, ok := l.(*layers.DNS); ok {
			out["protocol"] = "DNS"
			out["summary"] = dnsSummary(dns)

			return
		}
	}

	// A port number alone does not make a TCP segment that protocol. The
	// three-way handshake and bare ACKs on port 80 carry no application data,
	// and both tcpdump and Wireshark show those as TCP, reserving the
	// application name for segments that actually carry payload. Labelling them
	// HTTP made a NIAC capture disagree with tcpdump on the first three rows of
	// every conversation. UDP has no such handshake, so its ports always name it.
	if proto == "TCP" && !tcpCarriesPayload(packet) {
		return
	}

	src, _ := out["source_port"].(uint16)
	dst, _ := out["dest_port"].(uint16)
	named := protocolByPort(src, dst)
	if named == "" {
		return
	}
	out["protocol"] = named
	out["summary"] = named + " " + strconv.Itoa(int(src)) + "\u2192" + strconv.Itoa(int(dst))
}

// tcpCarriesPayload reports whether the segment has application bytes.
func tcpCarriesPayload(packet gopacket.Packet) bool {
	l := packet.Layer(layers.LayerTypeTCP)
	if l == nil {
		return false
	}
	tcp, ok := l.(*layers.TCP)

	return ok && len(tcp.Payload) > 0
}

// dnsSummary describes a DNS message the way a packet list column wants it.
func dnsSummary(dns *layers.DNS) string {
	if dns.QR {
		return "Response: " + strconv.Itoa(len(dns.Answers)) + " answers"
	}
	if len(dns.Questions) > 0 {
		return "Query: " + string(dns.Questions[0].Name)
	}

	return "Query"
}

// wellKnownPorts names the protocols NIAC simulates or a tester is likely to
// look for. Destination is checked first: for a request that is the service,
// which is the name an operator scans the list for.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var wellKnownPorts = map[uint16]string{
	20: "FTP-DATA", 21: "FTP", 22: "SSH", 23: "Telnet", 25: "SMTP",
	53: "DNS", 67: "DHCP", 68: "DHCP", 80: "HTTP", 110: "POP3",
	123: "NTP", 137: "NetBIOS", 138: "NetBIOS", 139: "NetBIOS", 143: "IMAP",
	161: "SNMP", 162: "SNMP-Trap", 443: "HTTPS", 445: "SMB", 514: "Syslog",
	546: "DHCPv6", 547: "DHCPv6", 993: "IMAPS", 995: "POP3S", 3306: "MySQL",
	3389: "RDP", 5353: "mDNS", 5432: "PostgreSQL", 6379: "Redis",
	8080: "HTTP-Alt", 8443: "HTTPS-Alt",
}

// protocolByPort names a well-known service, or "" when neither port is one.
func protocolByPort(src, dst uint16) string {
	if name, ok := wellKnownPorts[dst]; ok {
		return name
	}
	if name, ok := wellKnownPorts[src]; ok {
		return name
	}

	return ""
}

// enrichVLAN records an 802.1Q tag when the frame carries one.
func enrichVLAN(out map[string]any, headers map[string]any, packet gopacket.Packet) {
	l := packet.Layer(layers.LayerTypeDot1Q)
	if l == nil {
		return
	}
	dot1q, ok := l.(*layers.Dot1Q)
	if !ok {
		return
	}
	out["vlan_tag"] = dot1q.VLANIdentifier
	headers["dot1q"] = map[string]any{
		"vlanId":   dot1q.VLANIdentifier,
		"priority": dot1q.Priority,
	}
}

// enrichLinkLayer names the L2 control protocols NIAC itself emits.
//
// Nothing here was decoded before: enrichNetworkLayer and
// enrichTransportLayer only ever tested IPv4/IPv6/ARP and TCP/UDP/ICMP, so a
// tagged STP BPDU — exactly what NIAC puts on a trunk — fell through every
// branch and kept the "Unknown" default. gopacket's chain for one of these is
// Ethernet → Dot1Q → LLC → STP, and none of those four layer types was ever
// queried (D16).
//
// Runs last and only fills a protocol still marked Unknown, so it can never
// override a real L3/L4 classification.
func enrichLinkLayer(out map[string]any, packet gopacket.Packet) {
	if out["protocol"] != "Unknown" {
		return
	}
	// gopacket cannot chain past Dot1Q into LLC: Dot1Q.NextLayerType() hands its
	// inner value to EthernetType.LayerType(), and unlike the Ethernet decoder
	// that mapping has no "< 0x0600 means a length, so this is LLC" branch. A
	// tagged LLC frame therefore decodes as Dot1Q → Payload and every layer
	// lookup below misses. Re-decode the tag's payload as LLC by hand.
	if inner := taggedLLCProtocol(packet); inner != "" {
		out["protocol"] = inner
		if inner == "STP" {
			out["summary"] = "Spanning Tree BPDU"
		}

		return
	}

	switch {
	case packet.Layer(layers.LayerTypeSTP) != nil:
		out["protocol"] = "STP"
		out["summary"] = "Spanning Tree BPDU"
	case packet.Layer(layers.LayerTypeCiscoDiscovery) != nil:
		out["protocol"] = "CDP"
	case packet.Layer(layers.LayerTypeLinkLayerDiscovery) != nil:
		out["protocol"] = "LLDP"
	case packet.Layer(layers.LayerTypeSNAP) != nil:
		out["protocol"] = "SNAP"
	case packet.Layer(layers.LayerTypeLLC) != nil:
		out["protocol"] = "LLC"
	}
}

// enrichNetworkLayer sets source/dest IP and the L3 protocol label.
func enrichNetworkLayer(out map[string]any, headers map[string]any, packet gopacket.Packet) {
	if l := packet.Layer(layers.LayerTypeIPv4); l != nil {
		if ip, ok := l.(*layers.IPv4); ok {
			out["source_ip"] = ip.SrcIP.String()
			out["dest_ip"] = ip.DstIP.String()
			out["protocol"] = "IPv4"
			headers["ipv4"] = map[string]any{
				"src": ip.SrcIP.String(), "dst": ip.DstIP.String(),
				"ttl": ip.TTL, "version": ip.Version, "id": ip.Id,
			}
		}
		return
	}
	if l := packet.Layer(layers.LayerTypeIPv6); l != nil {
		if ip, ok := l.(*layers.IPv6); ok {
			out["source_ip"] = ip.SrcIP.String()
			out["dest_ip"] = ip.DstIP.String()
			out["protocol"] = "IPv6"
			headers["ipv6"] = map[string]any{
				"src": ip.SrcIP.String(), "dst": ip.DstIP.String(),
				"version": ip.Version, "hopLimit": ip.HopLimit,
				"nextHeader": ip.NextHeader.String(),
			}
		}
		return
	}
	if l := packet.Layer(layers.LayerTypeARP); l != nil {
		out["protocol"] = "ARP"
		enrichARP(out, l)
	}
}

// enrichARP adds source/dest IPv4 keys for an ARP layer if the
// addresses are well-formed.
func enrichARP(out map[string]any, l gopacket.Layer) {
	arp, ok := l.(*layers.ARP)
	if !ok {
		return
	}
	source, dest := "", ""
	if len(arp.SourceProtAddress) == ipv4Len {
		source = ipv4String(arp.SourceProtAddress)
		out["source_ip"] = source
	}
	if len(arp.DstProtAddress) == ipv4Len {
		dest = ipv4String(arp.DstProtAddress)
		out["dest_ip"] = dest
	}
	out["summary"] = arpSummary(arp.Operation, source, dest)
}

// arpSummary reads the way tcpdump's ARP line does, which is what a tester
// comparing the two expects to see.
func arpSummary(operation uint16, source, dest string) string {
	switch operation {
	case arpOperationRequest:
		return "ARP Request: Who has " + dest + "? Tell " + source
	case arpOperationReply:
		return "ARP Reply: " + source + " is at the sender hardware address"
	default:
		return "ARP operation " + strconv.Itoa(int(operation))
	}
}

// enrichTransportLayer sets ports, the L4 protocol label, and the per-layer
// header detail the packet-details pane renders.
//
// The flags/seq/ack/window detail came from the analyzer, which was the only
// surface that had it; the live inspector showed a bare "TCP 44100->80". Both
// read this now, so both show the same row.
func enrichTransportLayer(out map[string]any, headers map[string]any, packet gopacket.Packet) {
	if l := packet.Layer(layers.LayerTypeTCP); l != nil {
		if tcp, ok := l.(*layers.TCP); ok {
			enrichTCP(out, headers, tcp)
		}

		return
	}
	if l := packet.Layer(layers.LayerTypeUDP); l != nil {
		if udp, ok := l.(*layers.UDP); ok {
			enrichUDP(out, headers, udp)
		}

		return
	}
	if l := packet.Layer(layers.LayerTypeICMPv4); l != nil {
		out["protocol"] = "ICMP"
		if icmp, ok := l.(*layers.ICMPv4); ok {
			out["summary"] = icmpSummary(icmp)
		}

		return
	}
	if packet.Layer(layers.LayerTypeICMPv6) != nil {
		out["protocol"] = "ICMPv6"
	}
}

func enrichTCP(out map[string]any, headers map[string]any, tcp *layers.TCP) {
	src, dst := uint16(tcp.SrcPort), uint16(tcp.DstPort)
	flags := tcpFlags(tcp)
	out["source_port"] = src
	out["dest_port"] = dst
	out["protocol"] = "TCP"
	out["summary"] = fmt.Sprintf("%d -> %d [%s] Seq=%d Ack=%d Win=%d",
		src, dst, flags, tcp.Seq, tcp.Ack, tcp.Window)
	headers["tcp"] = map[string]any{
		"srcPort": src, "dstPort": dst,
		"seq": tcp.Seq, "ack": tcp.Ack, "flags": flags, "window": tcp.Window,
	}
}

func enrichUDP(out map[string]any, headers map[string]any, udp *layers.UDP) {
	src, dst := uint16(udp.SrcPort), uint16(udp.DstPort)
	out["source_port"] = src
	out["dest_port"] = dst
	out["protocol"] = "UDP"
	out["summary"] = fmt.Sprintf("%d -> %d Len=%d", src, dst, udp.Length)
	headers["udp"] = map[string]any{"srcPort": src, "dstPort": dst, "length": udp.Length}
}

// tcpFlags renders the set flags the way tcpdump does, so a NIAC row and a
// tcpdump line can be compared without translation.
func tcpFlags(tcp *layers.TCP) string {
	var flags []string
	for _, f := range []struct {
		set  bool
		name string
	}{
		{tcp.SYN, "SYN"},
		{tcp.ACK, "ACK"},
		{tcp.FIN, "FIN"},
		{tcp.RST, "RST"},
		{tcp.PSH, "PSH"},
		{tcp.URG, "URG"},
	} {
		if f.set {
			flags = append(flags, f.name)
		}
	}
	if len(flags) == 0 {
		return "none"
	}

	return strings.Join(flags, ",")
}

// icmpSummary names the two ICMP types anyone actually looks for and falls back
// to the numbers for the rest.
func icmpSummary(icmp *layers.ICMPv4) string {
	switch icmp.TypeCode.Type() {
	case icmpTypeEchoReply:
		return "Echo Reply (pong)"
	case icmpTypeEchoRequest:
		return "Echo Request (ping)"
	default:
		return fmt.Sprintf("Type=%d Code=%d", icmp.TypeCode.Type(), icmp.TypeCode.Code())
	}
}

// taggedLLCProtocol names the control protocol inside an 802.1Q tag, or "" if
// the frame is not tagged LLC.
func taggedLLCProtocol(packet gopacket.Packet) string {
	l := packet.Layer(layers.LayerTypeDot1Q)
	if l == nil {
		return ""
	}
	dot1q, ok := l.(*layers.Dot1Q)
	if !ok {
		return ""
	}

	return decodeTaggedLLC(uint16(dot1q.Type), l.LayerPayload())
}

// decodeTaggedLLC decodes an 802.1Q tag's payload as LLC and names the control
// protocol it carries. Returns "" when the tag does not carry LLC.
//
// innerType is the Dot1Q header's own type field, which holds a *length* rather
// than an EtherType when it is below 0x0600 — that is what marks the payload as
// LLC. gopacket already consumed those two bytes, so payload starts at the LLC
// header itself.
func decodeTaggedLLC(innerType uint16, payload []byte) string {
	if innerType >= minEtherType || len(payload) < dot1qLLCMinPayload {
		return ""
	}
	inner := gopacket.NewPacket(payload, layers.LayerTypeLLC, gopacket.NoCopy)
	switch {
	case inner.Layer(layers.LayerTypeSTP) != nil:
		return "STP"
	case inner.Layer(layers.LayerTypeCiscoDiscovery) != nil:
		return "CDP"
	case inner.Layer(layers.LayerTypeSNAP) != nil:
		return "SNAP"
	case inner.Layer(layers.LayerTypeLLC) != nil:
		return "LLC"
	}

	return ""
}

// ipv4String formats a 4-byte IPv4 address as dotted-quad. Avoids a
// net.IP allocation on the hot path.
func ipv4String(b []byte) string {
	return strconv.Itoa(int(b[0])) + "." +
		strconv.Itoa(int(b[1])) + "." +
		strconv.Itoa(int(b[2])) + "." +
		strconv.Itoa(int(b[3]))
}
