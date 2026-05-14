package api

// SSE packet observer: bridges the protocol stack's PacketObserver
// hook into the SSE hub so /api/v1/stream/packets subscribers see live
// frames. Previously the hub had BroadcastPacket defined but it was
// never called, leaving Packet Capture perpetually empty even while a
// simulation was clearly handling traffic.

import (
	"encoding/hex"
	"strconv"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/protocols"
)

// sseHubPacketObserver implements protocols.PacketObserver and forwards
// each packet event onto the SSE hub's packets stream.
//
// The hub's Broadcast is non-blocking (drops on full channel) so a slow
// SSE consumer can never back-pressure the protocol stack. Observers
// must be cheap; this one does a thin gopacket pass and a hex encode
// of the truncated raw bytes.
type sseHubPacketObserver struct {
	hub *SSEHub
}

// sseMaxRawPacketBytes caps how many bytes of the raw frame we ship in
// each event. The UI hex-viewer can render the full Ethernet MTU but
// at high packet rates we'd flood the SSE buffer. 256 bytes covers
// most useful headers (Ethernet + IPv4 + TCP options + a small
// payload) and keeps each event well under the SSE per-message budget.
const sseMaxRawPacketBytes = 256

// ipv4Len is the byte length of an IPv4 protocol address.
const ipv4Len = 4

// OnPacket is called by the stack for every rx/tx packet.
func (o *sseHubPacketObserver) OnPacket(direction string, pkt *protocols.Packet) {
	if o == nil || o.hub == nil || pkt == nil {
		return
	}
	o.hub.BroadcastPacket(packetToWire(direction, pkt))
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
	if len(raw) > sseMaxRawPacketBytes {
		raw = raw[:sseMaxRawPacketBytes]
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

	if ethLayer := packet.Layer(layers.LayerTypeEthernet); ethLayer != nil {
		if eth, ok := ethLayer.(*layers.Ethernet); ok {
			out["src_mac"] = eth.SrcMAC.String()
			out["dst_mac"] = eth.DstMAC.String()
		}
	}

	enrichNetworkLayer(out, packet)
	enrichTransportLayer(out, packet)
}

// enrichNetworkLayer sets source/dest IP and the L3 protocol label.
func enrichNetworkLayer(out map[string]any, packet gopacket.Packet) {
	if l := packet.Layer(layers.LayerTypeIPv4); l != nil {
		if ip, ok := l.(*layers.IPv4); ok {
			out["source_ip"] = ip.SrcIP.String()
			out["dest_ip"] = ip.DstIP.String()
			out["protocol"] = "IPv4"
		}
		return
	}
	if l := packet.Layer(layers.LayerTypeIPv6); l != nil {
		if ip, ok := l.(*layers.IPv6); ok {
			out["source_ip"] = ip.SrcIP.String()
			out["dest_ip"] = ip.DstIP.String()
			out["protocol"] = "IPv6"
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

// ipv4String formats a 4-byte IPv4 address as dotted-quad. Avoids a
// net.IP allocation on the hot path.
func ipv4String(b []byte) string {
	return strconv.Itoa(int(b[0])) + "." +
		strconv.Itoa(int(b[1])) + "." +
		strconv.Itoa(int(b[2])) + "." +
		strconv.Itoa(int(b[3]))
}
