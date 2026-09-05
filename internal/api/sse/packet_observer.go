package sse

// SSE packet observer: bridges the protocol stack's PacketObserver
// hook into the SSE hub so /api/v1/stream/packets subscribers see live
// frames. Previously the hub had BroadcastPacket defined but it was
// never called, leaving Packet Capture perpetually empty even while a
// simulation was clearly handling traffic.

import (
	"encoding/hex"
	"time"

	"github.com/gopacket/gopacket"

	"github.com/MustardSeedNetworks/niac-go/internal/packetdecode"
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
	packetdecode.Enrich(out, pkt.Data())
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

	// Only NewPacket and ParsePacket stamp a Timestamp; the protocol emitters
	// build Packet literals directly and leave it zero. Stamp observation time
	// here, as GopacketToWire does, so the wire never carries the zero value.
	ts := pkt.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	out := map[string]any{
		"timestamp": ts.UTC().Format("2006-01-02T15:04:05.000Z"),
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
	packetdecode.Enrich(out, pkt.Buffer)
	return out
}
