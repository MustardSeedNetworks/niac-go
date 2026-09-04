package capturering

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// snapLength is the interface block's declared snap length. The ring stores
// whole frames, so this is the jumbo ceiling rather than a truncation point.
const snapLength = 65536

// commentHint pre-sizes the per-frame annotation list: direction, vlan and
// the four fabric fields.
const commentHint = 6

// WritePcapng writes frames as a pcapng capture with one interface block
// named for the session's interface, per-frame timestamps, and the fabric
// decision recorded as a packet comment.
//
// The comment is what makes this worth exporting rather than reading the
// wire: "this ARP was dropped because the ingress network has no route to
// the egress" is a fact only NIAC knows, and Wireshark renders a comment in
// the packet detail pane.
func WritePcapng(w io.Writer, ifaceName string, frames []Frame) error {
	if ifaceName == "" {
		ifaceName = "niac"
	}
	writer, err := pcapgo.NewNgWriterInterface(w, pcapgo.NgInterface{
		Name:       ifaceName,
		LinkType:   layers.LinkTypeEthernet,
		SnapLength: snapLength,
	}, pcapgo.NgWriterOptions{
		SectionInfo: pcapgo.NgSectionInfo{
			Application: "niac",
			Comment:     "Replayed by NIAC. Packet comments carry the fabric decision.",
		},
	})
	if err != nil {
		return fmt.Errorf("open pcapng writer: %w", err)
	}

	for i := range frames {
		frame := &frames[i]
		info := gopacket.CaptureInfo{
			Timestamp:      frame.Timestamp,
			CaptureLength:  len(frame.Data),
			Length:         len(frame.Data),
			InterfaceIndex: 0,
		}
		if err = writer.WritePacketWithOptions(info, frame.Data, pcapgo.NgPacketOptions{
			Comments: frameComments(frame),
		}); err != nil {
			return fmt.Errorf("write frame %d: %w", i, err)
		}
	}

	if err = writer.Flush(); err != nil {
		return fmt.Errorf("flush pcapng: %w", err)
	}

	return nil
}

// frameComments renders one frame's annotations as a single comment line of
// key=value pairs. Empty fields are omitted so a frame the fabric never
// routed does not carry six empty keys.
func frameComments(frame *Frame) []string {
	parts := make([]string, 0, commentHint)
	if frame.Direction != "" {
		parts = append(parts, "direction="+frame.Direction)
	}
	if frame.Serial > 0 {
		parts = append(parts, "serial="+strconv.Itoa(frame.Serial))
	}
	if frame.VLAN >= 0 {
		parts = append(parts, "vlan="+strconv.Itoa(frame.VLAN))
	}
	parts = append(parts, traceComments(frame.Trace)...)
	if len(parts) == 0 {
		return nil
	}

	return []string{strings.Join(parts, " ")}
}

// traceComments renders the fabric decision. An unrouted frame — anything the
// stack answered locally — carries no trace at all, which is why the ingress
// network gates the whole group.
func traceComments(trace protocols.FabricTrace) []string {
	if trace.IngressNetwork == "" {
		return nil
	}
	parts := []string{"ingress=" + trace.IngressNetwork}
	if trace.PhysicalVLAN > 0 {
		parts = append(parts, "physical_vlan="+strconv.FormatUint(uint64(trace.PhysicalVLAN), 10))
	}
	for _, pair := range [][2]string{
		{"route", trace.RouteDecision},
		{"hop", trace.Hop},
		{"egress", trace.EgressNetwork},
		{"rejection", trace.RejectionReason},
	} {
		if pair[1] != "" {
			parts = append(parts, pair[0]+"="+pair[1])
		}
	}

	return parts
}
