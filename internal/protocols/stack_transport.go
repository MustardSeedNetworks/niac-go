package protocols

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// receiveThread receives packets from the network.
func (s *Stack) receiveThread() {
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
func (s *Stack) parseReceivedPacket(data []byte) *Packet {
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
		if s.debugConfig.GetGlobal() >= DebugLevelInfo {
			_, _ = fmt.Fprintln(os.Stdout, "Receive queue full, dropping packet")
		}
	}
}

// decodeThread decodes and routes packets to protocol handlers.
func (s *Stack) decodeThread() {
	timer := time.NewTimer(stackSelectTimeoutMs * time.Millisecond)
	defer timer.Stop()

	for s.running.Load() {
		select {
		case <-s.stopChan:
			return
		case pkt := <-s.recvQueue:
			s.decodePacket(pkt)
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(stackSelectTimeoutMs * time.Millisecond)
		case <-timer.C:
			timer.Reset(stackSelectTimeoutMs * time.Millisecond)
		}
	}
}

// sendThread sends packets to the network.
func (s *Stack) sendThread() {
	timer := time.NewTimer(stackSelectTimeoutMs * time.Millisecond)
	defer timer.Stop()

	for s.running.Load() {
		select {
		case <-s.stopChan:
			return
		case pkt := <-s.sendQueue:
			s.sendPacket(pkt)
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(stackSelectTimeoutMs * time.Millisecond)
		case <-timer.C:
			timer.Reset(stackSelectTimeoutMs * time.Millisecond)
		}
	}
}

// sendPacket sends a packet to the network.
func (s *Stack) sendPacket(pkt *Packet) {
	if pkt.Length == 0 {
		pkt.Length = len(pkt.Buffer)
	}

	err := s.capture.SendPacket(pkt.Buffer[:pkt.Length])
	if err != nil {
		if s.debugConfig.GetGlobal() >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "Error sending packet sn=%d: %v\n", pkt.SerialNumber, err)
		}

		s.stats.mu.Lock()
		s.stats.Errors++
		s.stats.mu.Unlock()

		return
	}

	s.stats.mu.Lock()
	s.stats.PacketsSent++
	s.stats.mu.Unlock()

	if s.debugConfig.GetGlobal() >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "Sent packet sn=%d length=%d\n", pkt.SerialNumber, pkt.Length)
	}

	// Reschedule if looping
	if pkt.LoopTime > 0 {
		s.wg.Go(func() {
			select {
			case <-time.After(pkt.LoopTime):
				if s.running.Load() {
					s.Send(pkt)
				}
			case <-s.stopChan:
			}
		})
	}
}

// babbleThread generates periodic network traffic.
func (s *Stack) babbleThread() {
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

	srcIP := firstIPv4Address(device)
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

	var serErr error
	if vlan > 0 {
		eth.EthernetType = layers.EthernetTypeDot1Q
		dot1q := &layers.Dot1Q{
			Priority:       0,
			DropEligible:   false,
			VLANIdentifier: safeUint16(vlan),
			Type:           layers.EthernetTypeARP,
		}
		serErr = gopacket.SerializeLayers(buf, opts, eth, dot1q, arp)
	} else {
		serErr = gopacket.SerializeLayers(buf, opts, eth, arp)
	}
	if serErr != nil {
		return
	}

	_ = s.SendRawPacket(buf.Bytes())
}

// Send queues a packet for sending.
func (s *Stack) Send(pkt *Packet) {
	select {
	case s.sendQueue <- pkt:
	default:
		if s.debugConfig.GetGlobal() >= DebugLevelInfo {
			_, _ = fmt.Fprintln(os.Stdout, "Send queue full, dropping packet")
		}
	}
}

// SendRawPacket queues raw bytes as a packet for sending.
func (s *Stack) SendRawPacket(data []byte) error {
	s.mu.Lock()
	s.serialNumber++
	serialNum := s.serialNumber
	s.mu.Unlock()

	pkt := &Packet{
		Buffer:       data,
		Length:       len(data),
		SerialNumber: serialNum,
	}

	s.Send(pkt)

	return nil
}
