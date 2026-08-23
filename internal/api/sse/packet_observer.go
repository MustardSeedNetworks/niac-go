package sse

// SSE packet observer: bridges the protocol stack's PacketObserver
// hook into the SSE hub so /api/v1/stream/packets subscribers see live
// frames. Previously the hub had BroadcastPacket defined but it was
// never called, leaving Packet Capture perpetually empty even while a
// simulation was clearly handling traffic.

import (
	"encoding/hex"
	"strconv"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// hubPacketObserver implements protocols.PacketObserver and forwards
// each packet event onto the SSE hub's packets stream.
//
// The hub's Broadcast is non-blocking (drops on full channel) so a slow
// SSE consumer can never back-pressure the protocol stack. Observers
// must be cheap; this one does a thin gopacket pass and a hex encode
// of the truncated raw bytes.
type hubPacketObserver struct {
	hub       *Hub
	sessionID string
}

// maxRawPacketBytes caps how many bytes of the raw frame we ship in
// each event. The UI hex-viewer can render the full Ethernet MTU but
// at high packet rates we'd flood the SSE buffer. 256 bytes covers
// most useful headers (Ethernet + IPv4 + TCP options + a small
// payload) and keeps each event well under the SSE per-message budget.
const maxRawPacketBytes = 256

// ipv4Len is the byte length of an IPv4 protocol address.
const ipv4Len = 4

// dot1qLLCMinPayload is a length field plus a minimal LLC header.
const dot1qLLCMinPayload = 5

// minEtherType is the first value that is an EtherType rather than a length.
const minEtherType = 0x0600

// headerLayerHint pre-sizes the per-packet headers map: ethernet, dot1q and one
// network layer covers almost every frame NIAC sees.
const headerLayerHint = 4

// NewPacketObserver returns a protocols.PacketObserver that forwards each
// observed packet onto the hub's packets stream. Used to wire the SSE bridge
// into a protocol stack from the api layer without exposing hub internals.
func NewPacketObserver(hub *Hub, sessionID ...string) protocols.PacketObserver {
	scope := ""
	if len(sessionID) > 0 {
		scope = sessionID[0]
	}
	return &hubPacketObserver{hub: hub, sessionID: scope}
}

// OnPacket is called by the stack for every rx/tx packet.
func (o *hubPacketObserver) OnPacket(direction string, pkt *protocols.Packet) {
	if o == nil || o.hub == nil || pkt == nil {
		return
	}
	if o.sessionID == "" {
		o.hub.BroadcastPacket(packetToWire(direction, pkt))
		return
	}
	o.hub.BroadcastPacketForSession(o.sessionID, packetToWire(direction, pkt))
}

// GopacketToWire is the standalone-capture cousin of packetToWire. It
// takes a gopacket.Packet directly rather than the protocols.Packet
// wrapper. Same JSON shape; the only loss is the serial / VLAN
// metadata that the protocols stack injects, which a sniff-only
// session doesn't have anyway.
func GopacketToWire(pkt gopacket.Packet) map[string]any {
	md := pkt.Metadata()
	raw := pkt.Data()
	size := len(raw)
	truncated := false
	if size > maxRawPacketBytes {
		raw = raw[:maxRawPacketBytes]
		truncated = true
	}

	ts := md.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	out := map[string]any{
		"timestamp": ts.UTC().Format("2006-01-02T15:04:05.000Z"),
		"direction": "rx",
		"size":      size,
		"raw_data":  hex.EncodeToString(raw),
		"truncated": truncated,
		"protocol":  "Unknown",
		"summary":   "",
	}
	enrichWithLayers(out, pkt.Data())
	return out
}

// packetToWire flattens a Packet into a map suitable for SSE JSON.
// Fields use snake_case to match the UI's existing fallback path —
// SSE payloads don't go through the camelCase converter.
func packetToWire(direction string, pkt *protocols.Packet) map[string]any {
	raw := pkt.Buffer
	if pkt.Length > 0 && pkt.Length <= len(raw) {
		raw = raw[:pkt.Length]
	}
	truncated := false
	if len(raw) > maxRawPacketBytes {
		raw = raw[:maxRawPacketBytes]
		truncated = true
	}

	out := map[string]any{
		"timestamp": pkt.Timestamp.UTC().Format("2006-01-02T15:04:05.000Z"),
		"direction": direction,
		"size":      pkt.Length,
		"raw_data":  hex.EncodeToString(raw),
		"truncated": truncated,
		"serial":    pkt.SerialNumber,
		"protocol":  "Unknown",
		"summary":   "",
	}
	if pkt.VLAN >= 0 {
		out["vlan"] = pkt.VLAN
	}
	trace := pkt.FabricTrace()
	if trace.IngressNetwork != "" {
		out["ingress_network"] = trace.IngressNetwork
		if trace.PhysicalVLAN > 0 {
			out["physical_vlan"] = trace.PhysicalVLAN
		}
		out["route_decision"] = trace.RouteDecision
		out["hop"] = trace.Hop
		out["egress_network"] = trace.EgressNetwork
		out["egress_rejection_reason"] = trace.RejectionReason
	}
	enrichWithLayers(out, pkt.Buffer)
	return out
}

// enrichWithLayers does a best-effort gopacket decode to fill in
// source/dest IPs, ports, and protocol. Decode errors are swallowed —
// the event still carries the raw bytes.
func enrichWithLayers(out map[string]any, buf []byte) {
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
	enrichTransportLayer(out, packet)
	enrichLinkLayer(out, packet)

	if len(headers) > 0 {
		out["headers"] = headers
	}
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
				"src": ip.SrcIP.String(), "dst": ip.DstIP.String(), "ttl": ip.TTL,
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
	if len(arp.SourceProtAddress) == ipv4Len {
		out["source_ip"] = ipv4String(arp.SourceProtAddress)
	}
	if len(arp.DstProtAddress) == ipv4Len {
		out["dest_ip"] = ipv4String(arp.DstProtAddress)
	}
}

// enrichTransportLayer sets ports + L4 protocol label.
func enrichTransportLayer(out map[string]any, packet gopacket.Packet) {
	if l := packet.Layer(layers.LayerTypeTCP); l != nil {
		if tcp, ok := l.(*layers.TCP); ok {
			out["source_port"] = uint16(tcp.SrcPort)
			out["dest_port"] = uint16(tcp.DstPort)
			out["protocol"] = "TCP"
			out["summary"] = "TCP " + strconv.Itoa(int(tcp.SrcPort)) + "→" + strconv.Itoa(int(tcp.DstPort))
		}
		return
	}
	if l := packet.Layer(layers.LayerTypeUDP); l != nil {
		if udp, ok := l.(*layers.UDP); ok {
			out["source_port"] = uint16(udp.SrcPort)
			out["dest_port"] = uint16(udp.DstPort)
			out["protocol"] = "UDP"
			out["summary"] = "UDP " + strconv.Itoa(int(udp.SrcPort)) + "→" + strconv.Itoa(int(udp.DstPort))
		}
		return
	}
	if packet.Layer(layers.LayerTypeICMPv4) != nil {
		out["protocol"] = "ICMP"
		return
	}
	if packet.Layer(layers.LayerTypeICMPv6) != nil {
		out["protocol"] = "ICMPv6"
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
