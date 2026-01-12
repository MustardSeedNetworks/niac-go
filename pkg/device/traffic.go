package device

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2" // Note: math/rand used for network traffic simulation (not security-critical)
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
)

// Sentinel errors for traffic generator.
var ErrTrafficGeneratorAlreadyRunning = errors.New("traffic generator already running")

// TrafficGenerator generates background network traffic.
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

// TrafficPattern represents a traffic generation pattern.
type TrafficPattern struct {
	Name     string
	Interval time.Duration
	Enabled  bool
	LastRun  time.Time
}

// NewTrafficGenerator creates a new traffic generator.
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

// Start starts the traffic generator.
func (tg *TrafficGenerator) Start() error {
	if tg.running {
		return ErrTrafficGeneratorAlreadyRunning
	}

	tg.running = true

	// Start unified traffic generation loop (v1.6.0)
	// Uses 10-second ticker to check all devices and their configured intervals
	go tg.trafficGenerationLoop()

	if tg.debugLevel >= 1 {
		slog.Info("Traffic generator started (v1.6.0 configurable traffic)")
	}

	return nil
}

// Stop stops the traffic generator.
func (tg *TrafficGenerator) Stop() {
	if !tg.running {
		return
	}

	tg.running = false
	close(tg.stopChan)

	if tg.debugLevel >= 1 {
		slog.Info("Traffic generator stopped")
	}
}

// trafficGenerationLoop unified traffic generation with per-device config support (v1.6.0).
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

// checkAndGenerateTraffic checks each device's config and generates traffic if intervals have elapsed.
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

// sendARPAnnouncement sends a gratuitous ARP for a single device (v1.6.0).
func (tg *TrafficGenerator) sendARPAnnouncement(device *SimulatedDevice) {
	_ = tg.sendGratuitousARP(device) // error is non-critical for traffic simulation
}

// sendPeriodicPing sends an ICMP ping from one device to another with configurable payload (v1.6.0).
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
	dst := deviceList[rand.IntN(len(deviceList))] // #nosec G404 -- network traffic simulation, not cryptographic

	if len(dst.Config.MACAddress) == 0 || len(dst.Config.IPAddresses) == 0 {
		return
	}

	// Use existing sendPing (currently doesn't use payloadSize, but we'll pass it for future use)
	_ = tg.sendPing(device, dst) // error is non-critical for traffic simulation
}

// generateRandomTrafficForDevice generates random traffic for a single device (v1.6.0).
func (tg *TrafficGenerator) generateRandomTrafficForDevice(
	device *SimulatedDevice,
	packetCount int,
	patterns []string,
) {
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
	for range packetCount {
		// Pick random pattern from configured patterns
		if len(patterns) == 0 {
			patterns = []string{"broadcast_arp", "multicast", "udp"}
		}

		pattern := patterns[rand.IntN(len(patterns))] // #nosec G404 -- network traffic simulation, not cryptographic

		switch pattern {
		case "broadcast_arp":
			_ = tg.sendBroadcastARP(device) // error is non-critical for traffic simulation
		case "multicast":
			_ = tg.sendMulticast(device) // error is non-critical for traffic simulation
		case "udp":
			if len(deviceList) > 1 {
				dst := deviceList[rand.IntN(len(deviceList))] // #nosec G404 -- network traffic simulation, not cryptographic
				if dst != device && len(dst.Config.MACAddress) > 0 {
					_ = tg.sendRandomUDP(device, dst) // error is non-critical for traffic simulation
				}
			}
		}

		// Small delay between packets
		time.Sleep(
			time.Duration(rand.IntN(100)) * time.Millisecond,
		) // #nosec G404 -- network traffic simulation, not cryptographic
	}

	if tg.debugLevel >= 3 {
		slog.Debug("Generated random packets", "device", device.Config.Name, "count", packetCount)
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
		return fmt.Errorf("failed to serialize ARP: %w", err)
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

// sendPing sends an ICMP Echo Request.
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
	// #nosec G404 -- network traffic simulation, not cryptographic
	icmpLayer := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
		Id:       uint16(rand.IntN(65536)), //nolint:gosec // G115: bounded by 65536
		Seq:      uint16(rand.IntN(65536)), //nolint:gosec // G115: bounded by 65536
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
		return fmt.Errorf("failed to serialize ping: %w", err)
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

// sendBroadcastARP sends a broadcast ARP request.
func (tg *TrafficGenerator) sendBroadcastARP(src *SimulatedDevice) error {
	// Pick a random IP to query
	randomIP := []byte{
		192,
		168,
		1,
		byte(rand.IntN(254) + 1),
	} // #nosec G404 -- network traffic simulation, not cryptographic

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

	err := gopacket.SerializeLayers(buffer, opts, eth, arp)
	if err != nil {
		return fmt.Errorf("failed to serialize ARP layers: %w", err)
	}

	pkt := &protocols.Packet{
		Buffer: buffer.Bytes(),
		Length: len(buffer.Bytes()),
		Device: src.Config,
	}

	tg.stack.Send(pkt)
	tg.simulator.IncrementCounter(src.Config.Name, "packets_sent")

	return nil
}

// sendMulticast sends a multicast packet.
func (tg *TrafficGenerator) sendMulticast(src *SimulatedDevice) error {
	// Send to multicast MAC
	multicastMAC := []byte{
		0x01,
		0x00,
		0x5e,
		byte(rand.IntN(128)),
		byte(rand.IntN(256)),
		byte(rand.IntN(256)),
	} // #nosec G404 -- network traffic simulation, not cryptographic

	eth := &layers.Ethernet{
		SrcMAC:       src.Config.MACAddress,
		DstMAC:       multicastMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true}

	err := gopacket.SerializeLayers(buffer, opts, eth, gopacket.Payload([]byte("multicast data")))
	if err != nil {
		return fmt.Errorf("failed to serialize multicast layers: %w", err)
	}

	pkt := &protocols.Packet{
		Buffer: buffer.Bytes(),
		Length: len(buffer.Bytes()),
		Device: src.Config,
	}

	tg.stack.Send(pkt)
	tg.simulator.IncrementCounter(src.Config.Name, "packets_sent")

	return nil
}

// sendRandomUDP sends a random UDP packet.
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

	// #nosec G404 -- network traffic simulation, not cryptographic
	udpLayer := &layers.UDP{
		SrcPort: layers.UDPPort(rand.IntN(60000) + 1024), //nolint:gosec // G115: bounded port range
		DstPort: layers.UDPPort(rand.IntN(60000) + 1024), //nolint:gosec // G115: bounded port range
	}
	_ = udpLayer.SetNetworkLayerForChecksum(ipLayer) // error is non-critical for checksum setup

	payload := []byte("random UDP data")

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipLayer, udpLayer, gopacket.Payload(payload))
	if err != nil {
		return fmt.Errorf("failed to serialize UDP layers: %w", err)
	}

	pkt := &protocols.Packet{
		Buffer: buffer.Bytes(),
		Length: len(buffer.Bytes()),
		Device: src.Config,
	}

	tg.stack.Send(pkt)
	tg.simulator.IncrementCounter(src.Config.Name, "packets_sent")

	return nil
}
