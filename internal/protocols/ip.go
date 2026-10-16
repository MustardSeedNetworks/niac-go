package protocols

import (
	"net"
	"net/netip"
	"slices"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/ip4defrag"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// IP protocol numbers.
const (
	IPProtocolICMP = 1
	IPProtocolTCP  = 6
	IPProtocolUDP  = 17
)

// IPv4 header field constants.
const (
	ipIPv4Version = 4 // IPv4 version field
	ipIPv4IHL     = 5 // IPv4 header length (5 = 20 bytes)
	ipv4IHLBytes  = 4
)

// IPHandler handles IP packets.
type IPHandler struct {
	stack      *Stack
	defragMu   sync.Mutex
	defraggers map[fragmentDomain]*ip4defrag.IPv4Defragmenter
	now        func() time.Time
}

// NewIPHandler creates a new IP handler.
func NewIPHandler(stack *Stack) *IPHandler {
	return &IPHandler{
		stack: stack, defraggers: make(map[fragmentDomain]*ip4defrag.IPv4Defragmenter), now: time.Now,
	}
}

// HandlePacket processes an IP packet.
func (h *IPHandler) HandlePacket(pkt *Packet) {
	debugLevel := h.stack.GetDebugLevel()

	ip := h.parseIPv4Layer(pkt, debugLevel)
	if ip == nil {
		return
	}
	if h.stack.fabric != nil && !h.stack.fabric.acceptsIPv4Source(ip.SrcIP, ip.DstIP, uint8(ip.Protocol)) {
		h.stack.rejectFabricIngress(pkt, fabricRejectionAttachmentSource)
		return
	}

	if debugLevel >= DebugLevelVerbose {
		logging.Debugf("IP packet: %s -> %s protocol=%d sn=%d",
			ip.SrcIP, ip.DstIP, ip.Protocol, pkt.SerialNumber)
	}

	devices := h.getTargetDevices(ip, pkt, debugLevel, pkt.VLAN)
	if devices == nil {
		return
	}
	h.stack.recordInboundProtocol(pkt, ip, devices)
	if ip.Flags&layers.IPv4MoreFragments != 0 || ip.FragOffset > 0 {
		var complete bool
		ip, complete = h.reassembleIPv4(pkt, ip, devices)
		if !complete {
			return
		}
	}
	if h.handleFabricTTLTimeout(pkt, ip) {
		return
	}
	h.stack.recordFabricForwarded(pkt)

	if len(pkt.fabricFirstHopIP) == 0 && h.shouldProcessTTL(ip, devices) && h.handleTTLTimeout(pkt, ip) {
		return
	}

	h.routeToProtocolHandler(pkt, ip, devices, debugLevel)
}

// parseIPv4Layer extracts the IPv4 layer from a packet.
func (h *IPHandler) parseIPv4Layer(pkt *Packet, debugLevel int) *layers.IPv4 {
	packet := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		if debugLevel >= DebugLevelInfo {
			logging.Debugf("IP packet missing IPv4 layer sn=%d", pkt.SerialNumber)
		}

		return nil
	}

	ip, ok := ipLayer.(*layers.IPv4)
	if !ok {
		return nil
	}
	headerLength := int(ip.IHL) * ipv4IHLBytes
	if ip.IHL < ipIPv4IHL || len(ip.Contents) < headerLength ||
		CalculateIPChecksum(ip.Contents[:headerLength]) != 0 {
		h.stack.rejectFabricIngress(pkt, fabricRejectionInvalidIPv4Checksum)
		return nil
	}

	return ip
}

func (s *Stack) rejectFabricIngress(pkt *Packet, reason string) {
	if s.fabric == nil {
		return
	}
	pkt.fabricTrace.RouteDecision = fabricRouteDecisionDropped
	pkt.fabricTrace.RejectionReason = reason
	s.stats.mu.Lock()
	s.stats.FabricDrops++
	s.stats.mu.Unlock()
}

// getTargetDevices returns devices that should receive this packet, scoped to
// the VLAN segment (if any) the packet arrived on.
func (h *IPHandler) getTargetDevices(
	ip *layers.IPv4,
	pkt *Packet,
	debugLevel int,
	vlan int,
) []*config.Device {
	if h.stack.fabric != nil {
		return h.getFabricTarget(ip, pkt)
	}
	serialNum := pkt.SerialNumber
	isBroadcast := ip.DstIP.Equal([]byte{255, 255, 255, 255})
	devices := h.stack.devicesForStateIPv4(vlan, ip.DstIP)

	if len(devices) == 0 && !isBroadcast {
		if debugLevel >= DebugLevelVerbose {
			logging.Debugf("IP packet not for our devices: %s sn=%d", ip.DstIP, serialNum)
		}

		return nil
	}

	if isBroadcast && len(devices) == 0 {
		devices = h.stack.devicesFor(vlan).GetAll()
	}

	return devices
}

func (h *IPHandler) getFabricTarget(ip *layers.IPv4, pkt *Packet) []*config.Device {
	if ip.DstIP.Equal(net.IPv4bcast) && len(h.stack.fabric.attachmentDHCP) > 0 {
		return h.stack.fabric.attachmentDHCP
	}
	dst, ok := netip.AddrFromSlice(ip.DstIP)
	if !ok {
		return nil
	}
	resolution, resolved := h.stack.fabric.resolveIPv4(dst.Unmap(), pkt.GetDestMAC())
	return h.stack.fabricTarget(pkt, resolution, resolved)
}

// fabricTarget records how the fabric resolved pkt's destination and returns
// the one device that answers it, or nil when nothing on the fabric does.
func (s *Stack) fabricTarget(pkt *Packet, resolution fabricResolution, resolved bool) []*config.Device {
	if !resolved {
		s.rejectFabricIngress(pkt, fabricRejectionNoRoute)
		return nil
	}
	pkt.fabricReplySourceMAC = cloneMAC(resolution.replySourceMAC)
	pkt.fabricTrace.EgressNetwork = resolution.egressNetwork
	if !resolution.routed {
		pkt.fabricTrace.RouteDecision = "local"
		return []*config.Device{resolution.device}
	}
	pkt.fabricFirstHopIP = net.IP(resolution.firstHopIP.AsSlice())
	pkt.fabricFirstHopMAC = cloneMAC(resolution.firstHopMAC)
	pkt.fabricFirstHopDevice = resolution.firstHopDevice
	pkt.fabricTrace.Hop = resolution.firstHopDevice.Name + ":" + resolution.routeVia
	return []*config.Device{resolution.device}
}

func (h *IPHandler) handleFabricTTLTimeout(pkt *Packet, ipLayer *layers.IPv4) bool {
	if h.stack.fabric == nil || len(pkt.fabricFirstHopIP) == 0 || ipLayer.TTL > 1 {
		return false
	}
	pkt.fabricTrace.RouteDecision = fabricRouteDecisionDropped
	pkt.fabricTrace.RejectionReason = fabricRejectionTTLExpired
	h.stack.stats.mu.Lock()
	h.stack.stats.FabricDrops++
	h.stack.stats.mu.Unlock()
	dstMAC := pkt.GetSourceMAC()
	if len(dstMAC) == 0 || len(pkt.fabricFirstHopMAC) == 0 {
		return true
	}
	_ = h.stack.icmpHandler.SendICMPTimeExceeded(
		pkt.fabricFirstHopIP,
		ipLayer.SrcIP,
		pkt.fabricFirstHopMAC,
		dstMAC,
		ipLayer,
		pkt.fabricFirstHopDevice,
		pkt.VLAN,
	)
	return true
}

func (s *Stack) recordFabricForwarded(pkt *Packet) {
	if s.fabric == nil || len(pkt.fabricFirstHopIP) == 0 {
		return
	}
	pkt.fabricTrace.RouteDecision = fabricRouteDecisionForwarded
	s.stats.mu.Lock()
	s.stats.FabricForwarded++
	s.stats.mu.Unlock()
}

// shouldProcessTTL determines if TTL handling should be applied.
func (h *IPHandler) shouldProcessTTL(ip *layers.IPv4, devices []*config.Device) bool {
	if ip.Protocol != IPProtocolUDP {
		return true
	}

	return !slices.ContainsFunc(devices, func(device *config.Device) bool {
		return device.MapToIP != nil
	})
}

// routeToProtocolHandler dispatches the packet to the appropriate L4 handler.
func (h *IPHandler) routeToProtocolHandler(pkt *Packet, ip *layers.IPv4, devices []*config.Device, debugLevel int) {
	//exhaustive:ignore
	switch ip.Protocol {
	case IPProtocolICMP:
		h.stack.icmpHandler.HandlePacket(pkt, ip, devices)
	case IPProtocolUDP:
		h.stack.udpHandler.HandlePacket(pkt, ip, devices)
	case IPProtocolTCP:
		h.stack.tcpHandler.HandlePacket(pkt, ip, devices)
	default:
		if debugLevel >= DebugLevelInfo {
			logging.Debugf("Unhandled IP protocol %d sn=%d", ip.Protocol, pkt.SerialNumber)
		}
	}
}

// isDestInTTLSubnet checks if the destination IP is in the device's TTL subnet.
func isDestInTTLSubnet(device *config.Device, dstIP net.IP) bool {
	if device.TTLConfig == nil || device.TTLConfig.IP == nil || device.TTLConfig.Mask == nil {
		return false
	}

	dst := dstIP.To4()
	ttlIP := device.TTLConfig.IP.To4()

	if dst == nil || ttlIP == nil {
		return false
	}

	for i := range 4 {
		if (dst[i] & device.TTLConfig.Mask[i]) != ttlIP[i] {
			return false
		}
	}

	return true
}

func (h *IPHandler) handleTTLTimeout(pkt *Packet, ipLayer *layers.IPv4) bool {
	if h.stack == nil || h.stack.icmpHandler == nil {
		return false
	}

	if ipLayer == nil {
		return false
	}

	ttl := int(ipLayer.TTL)

	device := h.stack.devicesFor(pkt.VLAN).GetDeviceByTTL(ttl)
	if device == nil {
		return false
	}

	srcIP := h.stack.firstStateIPv4Address(device)
	if srcIP == nil || len(device.MACAddress) == 0 {
		return false
	}

	// If the TTL device is the destination, ignore.
	if h.stack.deviceOwnsIPv4(device, ipLayer.DstIP) {
		return false
	}

	// Skip if destination is in TTL subnet
	if isDestInTTLSubnet(device, ipLayer.DstIP) {
		return false
	}

	dstMAC := pkt.GetSourceMAC()
	if dstMAC == nil {
		return false
	}

	err := h.stack.icmpHandler.SendICMPTimeExceeded(
		srcIP,
		ipLayer.SrcIP,
		device.MACAddress,
		dstMAC,
		ipLayer,
		device,
		pkt.VLAN,
	)
	if err != nil {
		return false
	}

	h.stack.devicesFor(pkt.VLAN).IncrementTTLCount(device)

	return true
}
