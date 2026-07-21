package protocols

import (
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/ip4defrag"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

type fragmentDomain struct {
	vlan   int
	device *config.Device
}

func (h *IPHandler) reassembleIPv4(pkt *Packet, fragment *layers.IPv4, devices []*config.Device) (*layers.IPv4, bool) {
	if len(devices) != 1 {
		return fragment, false
	}
	domain := fragmentDomain{vlan: pkt.VLAN, device: devices[0]}
	now := h.now()
	h.defragMu.Lock()
	defragger := h.defraggers[domain]
	if defragger == nil {
		defragger = ip4defrag.NewIPv4Defragmenter()
		h.defraggers[domain] = defragger
	}
	expired := defragger.DiscardOlderThan(now.Add(-time.Minute))
	reassembled, err := defragger.DefragIPv4WithTimestamp(fragment, now)
	h.defragMu.Unlock()
	group := h.stack.getSNMPAgents(devices[0])
	if group != nil && expired > 0 {
		group.telemetry.RecordReassemblyFailures(safeconv.Uint32(expired))
	}
	if err != nil {
		if group != nil {
			group.telemetry.RecordReassemblyFailures(1)
		}
		return fragment, false
	}
	if reassembled == nil {
		return fragment, false
	}
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.NoCopy)
	eth, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if !ok {
		return fragment, false
	}
	buffer := gopacket.NewSerializeBuffer()
	if serializeErr := gopacket.SerializeLayers(
		buffer, gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		eth, reassembled, gopacket.Payload(reassembled.Payload),
	); serializeErr != nil {
		if group != nil {
			group.telemetry.RecordReassemblyFailures(1)
		}
		return fragment, false
	}
	pkt.Buffer, pkt.Length = buffer.Bytes(), len(buffer.Bytes())
	if group != nil {
		fullPacket := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.NoCopy)
		group.telemetry.RecordReassemblySuccess(ipv4ProtocolEvent(fullPacket, reassembled))
	}
	return reassembled, true
}
