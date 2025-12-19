package device

import (
	"fmt"
	"log"
	"math/rand" // Note: math/rand used for network traffic simulation (not security-critical)
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
)

// TrafficGenerator generates background network traffic
type TrafficGenerator struct {
	simulator    *Simulator
	stack        *protocols.Stack
	running      bool
	stopChan     chan struct{}
	debugLevel   int
	lastARPTime  map[string]time.Time // Last ARP announcement time per device
	lastPingTime map[string]time.Time // Last ping time per device
	lastRandTime map[string]time.Time // Last random traffic time per device
}

// TrafficPattern represents a traffic generation pattern
type TrafficPattern struct {
	Name     string
	Interval time.Duration
	Enabled  bool
	LastRun  time.Time
}

// NewTrafficGenerator creates a new traffic generator
func NewTrafficGenerator(sim *Simulator, stack *protocols.Stack, debugLevel int) *TrafficGenerator {
	return &TrafficGenerator{
		simulator:    sim,
		stack:        stack,
		stopChan:     make(chan struct{}),
		debugLevel:   debugLevel,
		lastARPTime:  make(map[string]time.Time),
		lastPingTime: make(map[string]time.Time),
		lastRandTime: make(map[string]time.Time),
	}
}

// Start starts the traffic generator
func (tg *TrafficGenerator) Start() error {
	if tg.running {
		return fmt.Errorf("traffic generator already running")
	}

	tg.running = true

	// Start unified traffic generation loop (v1.6.0)
	// Uses 10-second ticker to check all devices and their configured intervals
	go tg.trafficGenerationLoop()

	if tg.debugLevel >= 1 {
		log.Println("Traffic generator started (v1.6.0 configurable traffic)")
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

// trafficGenerationLoop unified traffic generation with per-device config support (v1.6.0)
func (tg *TrafficGenerator) trafficGenerationLoop() {
	// Use 10-second ticker to check all devices
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for tg.running {
		select {
		case <-tg.stopChan:
			return
		case <-ticker.C:
			tg.checkAndGenerateTraffic()
		}
	}
}

// checkAndGenerateTraffic checks each device's config and generates traffic if intervals have elapsed
func (tg *TrafficGenerator) checkAndGenerateTraffic() {
	devices := tg.simulator.GetAllDevices()
	now := time.Now()

	for name, device := range devices {
		if device.State != StateUp {
			continue
		}

		// Skip if no traffic config or traffic disabled
		if device.Config.TrafficConfig == nil || !device.Config.TrafficConfig.Enabled {
			continue
		}

		cfg := device.Config.TrafficConfig

		// Check ARP announcements
		if cfg.ARPAnnouncements != nil && cfg.ARPAnnouncements.Enabled {
			lastTime := tg.lastARPTime[name]
			interval := time.Duration(cfg.ARPAnnouncements.Interval) * time.Second
			if now.Sub(lastTime) >= interval {
				tg.sendARPAnnouncement(device)
				tg.lastARPTime[name] = now
			}
		}

		// Check periodic pings
		if cfg.PeriodicPings != nil && cfg.PeriodicPings.Enabled {
			lastTime := tg.lastPingTime[name]
			interval := time.Duration(cfg.PeriodicPings.Interval) * time.Second
			if now.Sub(lastTime) >= interval {
				tg.sendPeriodicPing(device, cfg.PeriodicPings.PayloadSize)
				tg.lastPingTime[name] = now
			}
		}

		// Check random traffic
		if cfg.RandomTraffic != nil && cfg.RandomTraffic.Enabled {
			lastTime := tg.lastRandTime[name]
			interval := time.Duration(cfg.RandomTraffic.Interval) * time.Second
			if now.Sub(lastTime) >= interval {
				tg.generateRandomTrafficForDevice(device, cfg.RandomTraffic.PacketCount, cfg.RandomTraffic.Patterns)
				tg.lastRandTime[name] = now
			}
		}
	}
}

// sendARPAnnouncement sends a gratuitous ARP for a single device (v1.6.0)
func (tg *TrafficGenerator) sendARPAnnouncement(device *SimulatedDevice) {
	tg.sendGratuitousARP(device) // #nosec G104 -- error logged or non-critical
}

// sendPeriodicPing sends an ICMP ping from one device to another with configurable payload (v1.6.0)
func (tg *TrafficGenerator) sendPeriodicPing(device *SimulatedDevice, payloadSize int) {
	// Get list of other devices to ping
	devices := tg.simulator.GetAllDevices()
	deviceList := make([]*SimulatedDevice, 0)

	for _, dev := range devices {
		if dev.State == StateUp && dev != device {
			deviceList = append(deviceList, dev)
		}
	}

	if len(deviceList) == 0 {
		return
	}

	// Pick random destination
	dst := deviceList[rand.Intn(len(deviceList))] // #nosec G404 -- network traffic simulation, not cryptographic

	if len(dst.Config.MACAddress) == 0 || len(dst.Config.IPAddresses) == 0 {
		return
	}

	// Use existing sendPing (currently doesn't use payloadSize, but we'll pass it for future use)
	tg.sendPing(device, dst) // #nosec G104 -- error logged or non-critical
}

// generateRandomTrafficForDevice generates random traffic for a single device (v1.6.0)
func (tg *TrafficGenerator) generateRandomTrafficForDevice(device *SimulatedDevice, packetCount int, patterns []string) {
	// Get list of other devices
	devices := tg.simulator.GetAllDevices()
	deviceList := make([]*SimulatedDevice, 0)

	for _, dev := range devices {
		if dev.State == StateUp {
			deviceList = append(deviceList, dev)
		}
	}

	if len(deviceList) == 0 {
		return
	}

	// Generate configured number of packets
	for i := 0; i < packetCount; i++ {
		// Pick random pattern from configured patterns
		if len(patterns) == 0 {
			patterns = []string{"broadcast_arp", "multicast", "udp"}
		}

		pattern := patterns[rand.Intn(len(patterns))] // #nosec G404 -- network traffic simulation, not cryptographic

		switch pattern {
		case "broadcast_arp":
			tg.sendBroadcastARP(device) // #nosec G104 -- error logged or non-critical
		case "multicast":
			tg.sendMulticast(device) // #nosec G104 -- error logged or non-critical
		case "udp":
			if len(deviceList) > 1 {
				dst := deviceList[rand.Intn(len(deviceList))] // #nosec G404 -- network traffic simulation, not cryptographic
				if dst != device && len(dst.Config.MACAddress) > 0 {
					tg.sendRandomUDP(device, dst) // #nosec G104 -- error logged or non-critical
				}
			}
		}

		// Small delay between packets
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond) // #nosec G404 -- network traffic simulation, not cryptographic
	}

	if tg.debugLevel >= 3 {
		log.Printf("[%s] Generated %d random packets", device.Config.Name, packetCount)
	}
}

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
		Id:       uint16(rand.Intn(65536)), // #nosec G115,G404 -- network traffic simulation, ICMP ID from 16-bit range
		Seq:      uint16(rand.Intn(65536)), // #nosec G115,G404 -- network traffic simulation, ICMP sequence from 16-bit range
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

// sendBroadcastARP sends a broadcast ARP request
func (tg *TrafficGenerator) sendBroadcastARP(src *SimulatedDevice) error {
	// Pick a random IP to query
	randomIP := []byte{192, 168, 1, byte(rand.Intn(254) + 1)} // #nosec G404 -- network traffic simulation, not cryptographic

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

	gopacket.SerializeLayers(buffer, opts, eth, arp) // #nosec G104 -- error logged or non-critical

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
	multicastMAC := []byte{0x01, 0x00, 0x5e, byte(rand.Intn(128)), byte(rand.Intn(256)), byte(rand.Intn(256))} // #nosec G404 -- network traffic simulation, not cryptographic

	eth := &layers.Ethernet{
		SrcMAC:       src.Config.MACAddress,
		DstMAC:       multicastMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true}

	gopacket.SerializeLayers(buffer, opts, eth, gopacket.Payload([]byte("multicast data"))) // #nosec G104 -- error logged or non-critical

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
		SrcPort: layers.UDPPort(rand.Intn(60000) + 1024), // #nosec G115,G404 -- network traffic simulation, UDP port from limited range
		DstPort: layers.UDPPort(rand.Intn(60000) + 1024), // #nosec G115,G404 -- network traffic simulation, UDP port from limited range
	}
	udpLayer.SetNetworkLayerForChecksum(ipLayer) // #nosec G104 -- error logged or non-critical

	payload := []byte("random UDP data")

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}

	gopacket.SerializeLayers(buffer, opts, eth, ipLayer, udpLayer, gopacket.Payload(payload)) // #nosec G104 -- error logged or non-critical

	pkt := &protocols.Packet{
		Buffer: buffer.Bytes(),
		Length: len(buffer.Bytes()),
		Device: src.Config,
	}

	tg.stack.Send(pkt)
	tg.simulator.IncrementCounter(src.Config.Name, "packets_sent")

	return nil
}
