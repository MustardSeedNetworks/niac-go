package device

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
)

// TrafficGenerator generates background network traffic
type TrafficGenerator struct {
	simulator  *Simulator
	stack      *protocols.Stack
	running    bool
	stopChan   chan struct{}
	debugLevel int
}

// TrafficPattern represents a traffic generation pattern
type TrafficPattern struct {
	Name        string
	Interval    time.Duration
	Enabled     bool
	LastRun     time.Time
}

// NewTrafficGenerator creates a new traffic generator
func NewTrafficGenerator(sim *Simulator, stack *protocols.Stack, debugLevel int) *TrafficGenerator {
	return &TrafficGenerator{
		simulator:  sim,
		stack:      stack,
		stopChan:   make(chan struct{}),
		debugLevel: debugLevel,
	}
}

// Start starts the traffic generator
func (tg *TrafficGenerator) Start() error {
	if tg.running {
		return fmt.Errorf("traffic generator already running")
	}

	tg.running = true

	// Start traffic generation goroutines
	go tg.arpAnnouncementLoop()
	go tg.periodicPingLoop()
	go tg.randomTrafficLoop()

	if tg.debugLevel >= 1 {
		log.Println("Traffic generator started")
	}

	return nil
}

// Stop stops the traffic generator
func (tg *TrafficGenerator) Stop() {
	if !tg.running {
		return
	}

	tg.running = false
	close(tg.stopChan)

	if tg.debugLevel >= 1 {
		log.Println("Traffic generator stopped")
	}
}

// arpAnnouncementLoop sends periodic gratuitous ARP announcements
func (tg *TrafficGenerator) arpAnnouncementLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for tg.running {
		select {
		case <-tg.stopChan:
			return
		case <-ticker.C:
			tg.sendARPAnnouncements()
		}
	}
}

// sendARPAnnouncements sends gratuitous ARP for all devices
func (tg *TrafficGenerator) sendARPAnnouncements() {
	devices := tg.simulator.GetAllDevices()

	for name, device := range devices {
		if device.State != StateUp {
			continue
		}

		if len(device.Config.MACAddress) == 0 || len(device.Config.IPAddresses) == 0 {
			continue
		}

		// Build gratuitous ARP packet
		err := tg.sendGratuitousARP(device)
		if err != nil && tg.debugLevel >= 2 {
			log.Printf("Failed to send ARP announcement for %s: %v", name, err)
		} else if tg.debugLevel >= 3 {
			log.Printf("Sent ARP announcement for %s (%s)", name, device.Config.IPAddresses[0])
		}
	}
}

// sendGratuitousARP sends a gratuitous ARP packet
func (tg *TrafficGenerator) sendGratuitousARP(device *SimulatedDevice) error {
	mac := device.Config.MACAddress
	ip := device.Config.IPAddresses[0].To4()

	// Build Ethernet header (broadcast)
	eth := &layers.Ethernet{
		SrcMAC:       mac,
		DstMAC:       []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}

	// Build ARP packet
	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   mac,
		SourceProtAddress: ip,
		DstHwAddress:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		DstProtAddress:    ip, // Gratuitous: target is self
	}

	// Serialize
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, arp)
	if err != nil {
		return fmt.Errorf("failed to serialize ARP: %v", err)
	}

	// Send packet
	pkt := &protocols.Packet{
		Buffer: buffer.Bytes(),
		Length: len(buffer.Bytes()),
		Device: device.Config,
	}

	tg.stack.Send(pkt)

	// Update counters
	tg.simulator.IncrementCounter(device.Config.Name, "packets_sent")

	return nil
}

// periodicPingLoop sends periodic ICMP Echo requests
func (tg *TrafficGenerator) periodicPingLoop() {
	ticker := time.NewTicker(120 * time.Second)
	defer ticker.Stop()

	for tg.running {
		select {
		case <-tg.stopChan:
			return
		case <-ticker.C:
			tg.sendPeriodicPings()
		}
	}
}

// sendPeriodicPings sends ICMP echo requests from random devices
func (tg *TrafficGenerator) sendPeriodicPings() {
	devices := tg.simulator.GetAllDevices()
	deviceList := make([]*SimulatedDevice, 0, len(devices))

	for _, device := range devices {
		if device.State == StateUp {
			deviceList = append(deviceList, device)
		}
	}

	if len(deviceList) < 2 {
		return
	}

	// Pick two random devices
	src := deviceList[rand.Intn(len(deviceList))]
	dst := deviceList[rand.Intn(len(deviceList))]

	if src == dst {
		return
	}

	if len(src.Config.MACAddress) == 0 || len(src.Config.IPAddresses) == 0 {
		return
	}
	if len(dst.Config.MACAddress) == 0 || len(dst.Config.IPAddresses) == 0 {
		return
	}

	err := tg.sendPing(src, dst)
	if err != nil && tg.debugLevel >= 2 {
		log.Printf("Failed to send ping from %s to %s: %v", src.Config.Name, dst.Config.Name, err)
	} else if tg.debugLevel >= 3 {
		log.Printf("Sent ping from %s to %s", src.Config.Name, dst.Config.Name)
	}
}

// sendPing sends an ICMP Echo Request
func (tg *TrafficGenerator) sendPing(src, dst *SimulatedDevice) error {
	// Build Ethernet header
	eth := &layers.Ethernet{
		SrcMAC:       src.Config.MACAddress,
		DstMAC:       dst.Config.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}

	// Build IP header
	ipLayer := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    src.Config.IPAddresses[0].To4(),
		DstIP:    dst.Config.IPAddresses[0].To4(),
	}

	// Build ICMP Echo Request
	icmpLayer := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
		Id:       uint16(rand.Intn(65536)),
		Seq:      uint16(rand.Intn(65536)),
	}

	// Payload
	payload := []byte("NIAC-Go ping test data")

	// Serialize
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipLayer, icmpLayer, gopacket.Payload(payload))
	if err != nil {
		return fmt.Errorf("failed to serialize ping: %v", err)
	}

	// Send packet
	pkt := &protocols.Packet{
		Buffer: buffer.Bytes(),
		Length: len(buffer.Bytes()),
		Device: src.Config,
	}

	tg.stack.Send(pkt)

	// Update counters
	tg.simulator.IncrementCounter(src.Config.Name, "packets_sent")

	return nil
}

// randomTrafficLoop generates random low-level traffic
func (tg *TrafficGenerator) randomTrafficLoop() {
	ticker := time.NewTicker(180 * time.Second)
	defer ticker.Stop()

	for tg.running {
		select {
		case <-tg.stopChan:
			return
		case <-ticker.C:
			tg.generateRandomTraffic()
		}
	}
}

// generateRandomTraffic generates random packets for realism
func (tg *TrafficGenerator) generateRandomTraffic() {
	devices := tg.simulator.GetAllDevices()
	deviceList := make([]*SimulatedDevice, 0, len(devices))

	for _, device := range devices {
		if device.State == StateUp {
			deviceList = append(deviceList, device)
		}
	}

	if len(deviceList) == 0 {
		return
	}

	// Generate a few random packets
	numPackets := rand.Intn(5) + 1

	for i := 0; i < numPackets; i++ {
		src := deviceList[rand.Intn(len(deviceList))]

		if len(src.Config.MACAddress) == 0 || len(src.Config.IPAddresses) == 0 {
			continue
		}

		// Pick random traffic type
		trafficType := rand.Intn(3)

		switch trafficType {
		case 0:
			// Broadcast ARP request
			tg.sendBroadcastARP(src)
		case 1:
			// Multicast packet
			tg.sendMulticast(src)
		case 2:
			// Random UDP packet
			if len(deviceList) > 1 {
				dst := deviceList[rand.Intn(len(deviceList))]
				if dst != src && len(dst.Config.MACAddress) > 0 {
					tg.sendRandomUDP(src, dst)
				}
			}
		}

		// Small delay between packets
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	}

	if tg.debugLevel >= 3 {
		log.Printf("Generated %d random packets", numPackets)
	}
}

// sendBroadcastARP sends a broadcast ARP request
func (tg *TrafficGenerator) sendBroadcastARP(src *SimulatedDevice) error {
	// Pick a random IP to query
	randomIP := []byte{192, 168, 1, byte(rand.Intn(254) + 1)}

	eth := &layers.Ethernet{
		SrcMAC:       src.Config.MACAddress,
		DstMAC:       []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}

	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   src.Config.MACAddress,
		SourceProtAddress: src.Config.IPAddresses[0].To4(),
		DstHwAddress:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		DstProtAddress:    randomIP,
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}

	gopacket.SerializeLayers(buffer, opts, eth, arp)

	pkt := &protocols.Packet{
		Buffer: buffer.Bytes(),
		Length: len(buffer.Bytes()),
		Device: src.Config,
	}

	tg.stack.Send(pkt)
	tg.simulator.IncrementCounter(src.Config.Name, "packets_sent")

	return nil
}

// sendMulticast sends a multicast packet
func (tg *TrafficGenerator) sendMulticast(src *SimulatedDevice) error {
	// Send to multicast MAC
	multicastMAC := []byte{0x01, 0x00, 0x5e, byte(rand.Intn(128)), byte(rand.Intn(256)), byte(rand.Intn(256))}

	eth := &layers.Ethernet{
		SrcMAC:       src.Config.MACAddress,
		DstMAC:       multicastMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true}

	gopacket.SerializeLayers(buffer, opts, eth, gopacket.Payload([]byte("multicast data")))

	pkt := &protocols.Packet{
		Buffer: buffer.Bytes(),
		Length: len(buffer.Bytes()),
		Device: src.Config,
	}

	tg.stack.Send(pkt)
	tg.simulator.IncrementCounter(src.Config.Name, "packets_sent")

	return nil
}

// sendRandomUDP sends a random UDP packet
func (tg *TrafficGenerator) sendRandomUDP(src, dst *SimulatedDevice) error {
	eth := &layers.Ethernet{
		SrcMAC:       src.Config.MACAddress,
		DstMAC:       dst.Config.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}

	ipLayer := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    src.Config.IPAddresses[0].To4(),
		DstIP:    dst.Config.IPAddresses[0].To4(),
	}

	udpLayer := &layers.UDP{
		SrcPort: layers.UDPPort(rand.Intn(60000) + 1024),
		DstPort: layers.UDPPort(rand.Intn(60000) + 1024),
	}
	udpLayer.SetNetworkLayerForChecksum(ipLayer)

	payload := []byte("random UDP data")

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}

	gopacket.SerializeLayers(buffer, opts, eth, ipLayer, udpLayer, gopacket.Payload(payload))

	pkt := &protocols.Packet{
		Buffer: buffer.Bytes(),
		Length: len(buffer.Bytes()),
		Device: src.Config,
	}

	tg.stack.Send(pkt)
	tg.simulator.IncrementCounter(src.Config.Name, "packets_sent")

	return nil
}
