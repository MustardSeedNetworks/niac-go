package protocols

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

func (s *Stack) receiveThread() {
	defer s.wg.Done()

	buffer := make([]byte, stackReceiveBufferSize)

	for s.running.Load() {
		select {
		case <-s.stopChan:
			return
		default:
			s.receiveAndQueuePacket(buffer)
		}
	}
}

// receiveAndQueuePacket reads a single packet and queues it for processing.
func (s *Stack) receiveAndQueuePacket(buffer []byte) {
	data, err := s.capture.ReadPacket(buffer)
	if err != nil {
		if s.debugConfig.GetGlobal() >= DebugLevelVerbose {
			_, _ = fmt.Fprintf(os.Stdout, "Error reading packet: %v\n", err)
		}

		return
	}

	if len(data) == 0 {
		return
	}

	pkt := s.parseReceivedPacket(data)
	if pkt == nil {
		return
	}

	s.queuePacket(pkt)
}

// parseReceivedPacket parses received data into a Packet, updating stats.
//
// The incoming slice aliases a capture buffer that ReadPacket reuses on the
// next read. The returned Packet is queued onto a channel and decoded by a
// separate goroutine, so it must own its bytes; otherwise the next read
// corrupts still-in-flight packets — a data race that torn-reads every
// handler's header parsing. Clone at this ownership boundary.
func (s *Stack) parseReceivedPacket(data []byte) *Packet {
	data = bytes.Clone(data)

	s.mu.Lock()
	s.serialNumber++
	serialNum := s.serialNumber
	s.mu.Unlock()

	pkt, err := ParsePacket(data, serialNum)
	if err != nil {
		s.stats.mu.Lock()
		s.stats.Errors++
		s.stats.mu.Unlock()

		return nil
	}

	s.stats.mu.Lock()
	s.stats.PacketsReceived++
	s.stats.mu.Unlock()

	return pkt
}

// queuePacket queues a packet for decoding.
func (s *Stack) queuePacket(pkt *Packet) {
	select {
	case s.recvQueue <- pkt:
	default:
		if s.fabric != nil {
			pkt.fabricTrace.IngressNetwork = s.fabric.attachmentNetwork
			pkt.fabricTrace.PhysicalVLAN = s.fabric.binding.AccessVLAN
			pkt.fabricTrace.RouteDecision = fabricRouteDecisionDropped
			pkt.fabricTrace.RejectionReason = "receive_queue_full"
			s.stats.mu.Lock()
			s.stats.FabricDrops++
			s.stats.mu.Unlock()
		}
		s.notifyObservers("rx", pkt)
		if s.debugConfig.GetGlobal() >= DebugLevelInfo {
			_, _ = fmt.Fprintln(os.Stdout, "Receive queue full, dropping packet")
		}
	}
}

// decodeThread decodes and routes packets to protocol handlers.
func (s *Stack) decodeThread() {
	defer s.wg.Done()

	for s.running.Load() {
		select {
		case <-s.stopChan:
			return
		case pkt := <-s.recvQueue:
			s.decodePacket(pkt)
		case <-time.After(stackSelectTimeoutMs * time.Millisecond):
			// Periodic check
		}
	}
}

// decodePacket decodes a packet and routes to appropriate handler.
func (s *Stack) decodePacket(pkt *Packet) {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	defer s.notifyObservers("rx", pkt)
	if s.fabric != nil {
		pkt.fabricTrace.IngressNetwork = s.fabric.attachmentNetwork
		pkt.fabricTrace.PhysicalVLAN = s.fabric.binding.AccessVLAN
	}

	// Defense in depth: a simulator is exposed to arbitrary, adversarial, and
	// malformed traffic (discovery tools, scanners, fuzzers). A panic while
	// handling one packet must never take down the whole sim — recover, log,
	// and drop the offending packet so every other device keeps responding.
	defer func() {
		if r := recover(); r != nil {
			s.stats.mu.Lock()
			s.stats.Errors++
			s.stats.mu.Unlock()

			if s.debugConfig.GetGlobal() >= DebugLevelBasic {
				_, _ = fmt.Fprintf(os.Stderr,
					"recovered from panic decoding packet sn=%d: %v\n%s\n",
					pkt.SerialNumber, r, debug.Stack())
			}
		}
	}()

	if s.fabric != nil && !s.fabric.acceptsFrame(pkt.VLAN, pkt.VLANTagged) {
		pkt.fabricTrace.RouteDecision = fabricRouteDecisionDropped
		pkt.fabricTrace.RejectionReason = "physical_vlan_rejected"
		s.stats.mu.Lock()
		s.stats.FabricDrops++
		s.stats.mu.Unlock()
		return
	}

	// VLAN confinement: when the sim serves tagged VLANs, ignore untagged frames
	// entirely. This guarantees it only ever replays on its tagged VLAN(s) and can
	// never respond on the native/default VLAN — so a misconfigured trunk can't
	// turn it into a rogue DHCP/ARP responder on a management network. Tagged
	// traffic (incl. a tester's cross-subnet queries, which arrive on its own
	// VLAN) is unaffected.
	if s.fabric == nil && s.vlanMode && pkt.VLAN <= 0 {
		return
	}

	// Segment confinement (ADR 0008 multi-VLAN): in segment mode every VLAN
	// tag must be bound to a segment/"demo" to get a reply. This is the
	// segment-mode analogue of the vlanMode check above — keyed by
	// devicesFor's per-tag lookup instead of a single global flag — so a tag
	// with no bound segment is dropped here rather than falling through to
	// per-handler device lookups that would just come up empty anyway.
	if s.segmentTables != nil && s.devicesFor(pkt.VLAN) == nil {
		return
	}

	// Try MAC-based routing first (multicast protocols)
	if s.routeByMAC(pkt) {
		return
	}

	// Route by EtherType
	s.routeByEtherType(pkt)
}

// routeByMAC routes packets based on multicast MAC addresses.
// Returns true if packet was handled.
func (s *Stack) routeByMAC(pkt *Packet) bool {
	dstMAC := pkt.GetDestMAC()
	if len(dstMAC) != macAddrLen {
		return false
	}

	// Check for STP (multicast MAC 01:80:C2:00:00:00)
	if matchMAC(
		dstMAC,
		macMulticastIEEE,
		macSecondByteSTP,
		macThirdByteSTP,
		macUnicastZero,
		macUnicastZero,
		macUnicastZero,
	) {
		s.stpHandler.HandlePacket(pkt)
		return true
	}

	// Check for LLDP (multicast MAC 01:80:C2:00:00:0E)
	if matchMAC(
		dstMAC,
		macMulticastIEEE,
		macSecondByteSTP,
		macThirdByteSTP,
		macUnicastZero,
		macUnicastZero,
		macByteLLDP,
	) {
		s.lldpHandler.HandlePacket(pkt)
		return true
	}

	// Check for CDP (multicast MAC 01:00:0C:CC:CC:CC)
	if matchMAC(
		dstMAC,
		macMulticastIEEE,
		macUnicastZero,
		macThirdByteCDP,
		macByteCC,
		macByteCC,
		macByteCC,
	) {
		s.cdpHandler.HandlePacket(pkt)
		return true
	}

	// Check for EDP (multicast MAC 00:E0:2B:00:00:00)
	if matchMAC(
		dstMAC,
		macUnicastZero,
		macSecondByteEDP,
		macThirdByteEDP,
		macUnicastZero,
		macUnicastZero,
		macUnicastZero,
	) {
		s.edpHandler.HandlePacket(pkt)
		return true
	}

	// Check for FDP (multicast MAC 01:E0:52:CC:CC:CC)
	if matchMAC(
		dstMAC,
		macMulticastIEEE,
		macSecondByteEDP,
		macThirdByteFDP,
		macByteCC,
		macByteCC,
		macByteCC,
	) {
		s.fdpHandler.HandlePacket(pkt)
		return true
	}

	return false
}

// matchMAC checks if a MAC address matches the given bytes.
func matchMAC(mac []byte, b0, b1, b2, b3, b4, b5 byte) bool {
	return mac[0] == b0 && mac[1] == b1 && mac[2] == b2 &&
		mac[3] == b3 && mac[4] == b4 && mac[5] == b5
}

// routeByEtherType routes packets based on EtherType.
func (s *Stack) routeByEtherType(pkt *Packet) {
	etherType := pkt.GetEtherType()

	// Check for VLAN
	offset := SizeOfMac * ethMACCount
	if etherType == EtherTypeVLAN {
		// VLAN present, get actual EtherType
		offset += 4
		etherType = pkt.Get16(offset)
	}

	if s.debugConfig.GetGlobal() >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"Decoding packet sn=%d etherType=0x%04x\n",
			pkt.SerialNumber,
			etherType,
		)
	}

	// Route to protocol handler
	switch etherType {
	case EtherTypeARP:
		s.arpHandler.HandlePacket(pkt)
	case EtherTypeIP:
		s.ipHandler.HandlePacket(pkt)
	case EtherTypeIPv6:
		s.ipv6Handler.HandlePacket(pkt)
	case EtherTypeLLDP:
		s.lldpHandler.HandlePacket(pkt)
	case EtherTypeEDP:
		s.edpHandler.HandlePacket(pkt)
	case EtherTypeFDP:
		s.fdpHandler.HandlePacket(pkt)
	default:
		if s.debugConfig.GetGlobal() >= DebugLevelInfo {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"Unknown EtherType 0x%04x sn=%d\n",
				etherType,
				pkt.SerialNumber,
			)
		}
	}
}

// sendThread sends packets to the network.
func (s *Stack) sendThread() {
	defer s.wg.Done()

	for s.running.Load() {
		select {
		case <-s.stopChan:
			return
		case pkt := <-s.sendQueue:
			s.sendPacket(pkt)
		case <-time.After(stackSelectTimeoutMs * time.Millisecond):
			// Periodic check
		}
	}
}

// sendPacket sends a packet to the network.
func (s *Stack) sendPacket(pkt *Packet) {
	frame, wireVLAN, err := s.finalizeEgressFrame(pkt)
	if err != nil {
		if errors.Is(err, errDiscoveryOffAttachment) {
			s.recordDiscoveryEgressDrop(pkt)
			return
		}
		s.recordSendError(pkt, err)
		return
	}

	err = s.capture.SendPacket(frame)
	if err != nil {
		s.recordSendError(pkt, err)
		return
	}

	s.stats.mu.Lock()
	s.stats.PacketsSent++
	s.stats.mu.Unlock()

	transmitted := pkt.Clone()
	transmitted.Buffer = bytes.Clone(frame)
	transmitted.Length = len(frame)
	transmitted.VLAN = wireVLAN
	s.recordOutboundProtocol(transmitted)
	s.notifyObservers("tx", transmitted)

	if s.debugConfig.GetGlobal() >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "Sent packet sn=%d length=%d\n", pkt.SerialNumber, pkt.Length)
	}

	// Reschedule if looping
	if pkt.LoopTime > 0 {
		go func() {
			time.Sleep(pkt.LoopTime)

			if s.running.Load() {
				s.Send(pkt)
			}
		}()
	}
}

func (s *Stack) finalizeEgressFrame(pkt *Packet) ([]byte, int, error) {
	if pkt == nil {
		return nil, -1, errors.New("cannot transmit a nil packet")
	}
	if err := s.validateDiscoveryEgress(pkt); err != nil {
		return nil, -1, err
	}
	length := pkt.Length
	if length == 0 {
		length = len(pkt.Buffer)
	}
	if length < ethHeaderLen || length > len(pkt.Buffer) {
		return nil, -1, fmt.Errorf("invalid Ethernet frame length %d", length)
	}
	frame := pkt.Buffer[:length]
	if s.fabric != nil {
		untagged, err := stripDot1Q(frame)
		if err != nil {
			return nil, -1, err
		}
		if !s.fabric.binding.WireTagged {
			return untagged, -1, nil
		}
		physicalVLAN := int(s.fabric.binding.AccessVLAN)
		return insertDot1Q(untagged, physicalVLAN), physicalVLAN, nil
	}
	if pkt.VLAN > 0 {
		frame = insertDot1Q(frame, pkt.VLAN)
	}
	return frame, pkt.VLAN, nil
}

func (s *Stack) recordSendError(pkt *Packet, err error) {
	if s.debugConfig.GetGlobal() >= DebugLevelInfo {
		serial := 0
		if pkt != nil {
			serial = pkt.SerialNumber
		}
		_, _ = fmt.Fprintf(os.Stdout, "Error sending packet sn=%d: %v\n", serial, err)
	}
	s.stats.mu.Lock()
	s.stats.Errors++
	if s.fabric != nil {
		s.stats.FabricDrops++
		if pkt != nil {
			pkt.fabricTrace.IngressNetwork = s.fabric.attachmentNetwork
			pkt.fabricTrace.PhysicalVLAN = s.fabric.binding.AccessVLAN
			pkt.fabricTrace.RouteDecision = fabricRouteDecisionDropped
			pkt.fabricTrace.RejectionReason = "egress_rejected"
		}
	}
	s.stats.mu.Unlock()
	if s.fabric != nil && pkt != nil {
		s.notifyObservers("tx", pkt)
	}
}

// babbleThread generates periodic network traffic.
func (s *Stack) babbleThread() {
	defer s.wg.Done()

	ticker := time.NewTicker(stackBabbleIntervalSec * time.Second)
	defer ticker.Stop()

	for s.running.Load() {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			for _, device := range s.devices.GetAll() {
				if device == nil || !device.Babble {
					continue
				}

				s.sendBabble(device)
				time.Sleep(stackBabbleDelayMs * time.Millisecond)
			}
		}
	}
}

func (s *Stack) sendBabble(device *config.Device) {
	if device == nil || len(device.MACAddress) == 0 {
		return
	}

	srcIP := s.firstStateIPv4Address(device)
	if srcIP == nil {
		return
	}

	broadcastMAC := net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	targetIP := net.IPv4(stackBabbleTargetIPOctet3, 1, 1, 1)

	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     macAddrLen,
		ProtAddressSize:   ipv4AddrLen,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   []byte(device.MACAddress),
		SourceProtAddress: []byte(srcIP.To4()),
		DstHwAddress:      []byte{0, 0, 0, 0, 0, 0},
		DstProtAddress:    []byte(targetIP.To4()),
	}

	eth := &layers.Ethernet{
		SrcMAC:       device.MACAddress,
		DstMAC:       broadcastMAC,
		EthernetType: layers.EthernetTypeARP,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}

	vlan := device.VLAN
	if vlan == 0 {
		if v, ok := device.Properties["vlan"]; ok {
			if parsed, err := strconv.Atoi(v); err == nil {
				vlan = parsed
			}
		}
	}

	if vlan > 0 {
		eth.EthernetType = layers.EthernetTypeDot1Q
		dot1q := &layers.Dot1Q{
			Priority:       0,
			DropEligible:   false,
			VLANIdentifier: safeconv.Uint16(vlan),
			Type:           layers.EthernetTypeARP,
		}
		_ = gopacket.SerializeLayers(buf, opts, eth, dot1q, arp)
	} else {
		_ = gopacket.SerializeLayers(buf, opts, eth, arp)
	}

	_ = s.SendRawPacket(buf.Bytes())
}
