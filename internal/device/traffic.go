package device

import (
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/protocols"
	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

// Traffic generation constants.
const (
	// trafficTickerInterval is the interval between traffic generation checks.
	trafficTickerInterval = 10 * time.Second

	// maxRandomDelayMs is the maximum random delay between packets in milliseconds.
	maxRandomDelayMs = 100

	// Network layer constants (debugLevelVerbose is in simulator.go).
	macAddressLen    = 6     // Hardware address size for Ethernet
	ipv4AddressLen   = 4     // Protocol address size for IPv4
	ipv4Version      = 4     // IP version 4
	ipv4IHL          = 5     // IP header length in 32-bit words (5 * 4 = 20 bytes)
	defaultTTL       = 64    // Default time-to-live
	maxUint16PlusOne = 65536 // 2^16 for random uint16 generation
	maxHostNumber    = 254   // Max host number (excluding .0 and .255)
	multicastMask    = 128   // First byte mask for IPv4 multicast
	ephemeralPortMin = 1024  // Minimum ephemeral port number
	ephemeralPortMax = 60000 // Maximum ephemeral port range
	byteRange        = 256   // Range of values for a single byte (0-255)
)

// ErrTrafficGeneratorAlreadyRunning is returned when starting an active generator.
var ErrTrafficGeneratorAlreadyRunning = errors.New("traffic generator already running")

// TrafficPattern represents a traffic generation pattern with timing state.
type TrafficPattern struct {
	Name     string
	Interval time.Duration
	Enabled  bool
	LastRun  time.Time
}

// Pattern name constants.
const (
	patternARP    = "arp_announcements"
	patternPing   = "periodic_pings"
	patternRandom = "random_traffic"
)

// TrafficGenerator generates background network traffic.
type TrafficGenerator struct {
	simulator  *Simulator
	stack      *protocols.Stack
	running    atomic.Bool
	stopChan   chan struct{}
	debugLevel int
	patterns   map[string]map[string]*TrafficPattern // device name → pattern name → state
}

// NewTrafficGenerator creates a new traffic generator.
func NewTrafficGenerator(sim *Simulator, stack *protocols.Stack, debugLevel int) *TrafficGenerator {
	return &TrafficGenerator{
		simulator:  sim,
		stack:      stack,
		stopChan:   make(chan struct{}),
		debugLevel: debugLevel,
		patterns:   make(map[string]map[string]*TrafficPattern),
	}
}

// getPattern returns the TrafficPattern for a device/pattern pair, creating it on first access.
func (tg *TrafficGenerator) getPattern(deviceName, patternName string) *TrafficPattern {
	devicePatterns, ok := tg.patterns[deviceName]
	if !ok {
		devicePatterns = make(map[string]*TrafficPattern)
		tg.patterns[deviceName] = devicePatterns
	}
	p, ok := devicePatterns[patternName]
	if !ok {
		p = &TrafficPattern{Name: patternName}
		devicePatterns[patternName] = p
	}
	return p
}

// Start starts the traffic generator.
func (tg *TrafficGenerator) Start() error {
	logger := slog.Default()
	if !tg.running.CompareAndSwap(false, true) {
		return ErrTrafficGeneratorAlreadyRunning
	}

	// Start unified traffic generation loop (v1.6.0)
	// Uses 10-second ticker to check all devices and their configured intervals
	go tg.trafficGenerationLoop()

	if tg.debugLevel >= 1 {
		logger.Info("Traffic generator started (v1.6.0 configurable traffic)")
	}

	return nil
}

// Stop stops the traffic generator.
func (tg *TrafficGenerator) Stop() {
	logger := slog.Default()
	if !tg.running.CompareAndSwap(true, false) {
		return
	}

	close(tg.stopChan)

	if tg.debugLevel >= 1 {
		logger.Info("Traffic generator stopped")
	}
}

// trafficGenerationLoop unified traffic generation with per-device config support (v1.6.0).
func (tg *TrafficGenerator) trafficGenerationLoop() {
	// Use 10-second ticker to check all devices
	ticker := time.NewTicker(trafficTickerInterval)
	defer ticker.Stop()

	for tg.running.Load() {
		select {
		case <-tg.stopChan:
			return
		case <-ticker.C:
			tg.checkAndGenerateTraffic()
		}
	}
}

// sleepInterruptible sleeps for d, returning false if Stop closes stopChan
// before the timer fires. Callers should treat false as "exit your loop now".
func (tg *TrafficGenerator) sleepInterruptible(d time.Duration) bool {
	if d <= 0 {
		return tg.running.Load()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-tg.stopChan:
		return false
	case <-timer.C:
		return true
	}
}

// checkAndGenerateTraffic checks each device's config and generates traffic if intervals have elapsed.
func (tg *TrafficGenerator) checkAndGenerateTraffic() {
	devices := tg.simulator.GetAllDevices()
	now := time.Now()

	for name, device := range devices {
		if !tg.shouldGenerateTraffic(device) {
			continue
		}
		tg.generateDeviceTraffic(name, device, now)
	}
}

// shouldGenerateTraffic returns true if traffic generation should proceed for the device.
func (tg *TrafficGenerator) shouldGenerateTraffic(device *SimulatedDevice) bool {
	if device.State != StateUp {
		return false
	}
	return device.Config.TrafficConfig != nil && device.Config.TrafficConfig.Enabled
}

// generateDeviceTraffic generates all configured traffic types for a device.
func (tg *TrafficGenerator) generateDeviceTraffic(
	name string,
	device *SimulatedDevice,
	now time.Time,
) {
	cfg := device.Config.TrafficConfig

	tg.checkARPAnnouncements(name, device, cfg, now)
	tg.checkPeriodicPings(name, device, cfg, now)
	tg.checkRandomTraffic(name, device, cfg, now)
}

// checkARPAnnouncements sends ARP announcements if the interval has elapsed.
func (tg *TrafficGenerator) checkARPAnnouncements(
	name string,
	device *SimulatedDevice,
	cfg *config.TrafficConfig,
	now time.Time,
) {
	if cfg.ARPAnnouncements == nil || !cfg.ARPAnnouncements.Enabled {
		return
	}

	p := tg.getPattern(name, patternARP)
	p.Interval = time.Duration(cfg.ARPAnnouncements.Interval) * time.Second
	p.Enabled = true

	if now.Sub(p.LastRun) < p.Interval {
		return
	}

	tg.sendARPAnnouncement(device)
	p.LastRun = now
}

// checkPeriodicPings sends periodic pings if the interval has elapsed.
func (tg *TrafficGenerator) checkPeriodicPings(
	name string,
	device *SimulatedDevice,
	cfg *config.TrafficConfig,
	now time.Time,
) {
	if cfg.PeriodicPings == nil || !cfg.PeriodicPings.Enabled {
		return
	}

	p := tg.getPattern(name, patternPing)
	p.Interval = time.Duration(cfg.PeriodicPings.Interval) * time.Second
	p.Enabled = true

	if now.Sub(p.LastRun) < p.Interval {
		return
	}

	tg.sendPeriodicPing(device, cfg.PeriodicPings.PayloadSize)
	p.LastRun = now
}

// checkRandomTraffic generates random traffic if the interval has elapsed.
func (tg *TrafficGenerator) checkRandomTraffic(
	name string,
	device *SimulatedDevice,
	cfg *config.TrafficConfig,
	now time.Time,
) {
	if cfg.RandomTraffic == nil || !cfg.RandomTraffic.Enabled {
		return
	}

	p := tg.getPattern(name, patternRandom)
	p.Interval = time.Duration(cfg.RandomTraffic.Interval) * time.Second
	p.Enabled = true

	if now.Sub(p.LastRun) < p.Interval {
		return
	}

	tg.generateRandomTrafficForDevice(
		device,
		cfg.RandomTraffic.PacketCount,
		cfg.RandomTraffic.Patterns,
	)
	p.LastRun = now
}

// sendARPAnnouncement sends a gratuitous ARP for a single device (v1.6.0).
func (tg *TrafficGenerator) sendARPAnnouncement(device *SimulatedDevice) {
	_ = tg.sendGratuitousARP(device) // error is non-critical for traffic simulation
}

// sendPeriodicPing sends an ICMP ping from one device to another with configurable payload (v1.6.0).
func (tg *TrafficGenerator) sendPeriodicPing(device *SimulatedDevice, _ int) {
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
	dst := deviceList[simRand.IntN(len(deviceList))]
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
	logger := slog.Default()
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

		pattern := patterns[simRand.IntN(len(patterns))]

		switch pattern {
		case "broadcast_arp":
			_ = tg.sendBroadcastARP(device) // error is non-critical for traffic simulation
		case "multicast":
			_ = tg.sendMulticast(device) // error is non-critical for traffic simulation
		case "udp":
			if len(deviceList) > 1 {
				dst := deviceList[simRand.IntN(len(deviceList))]
				if dst != device && len(dst.Config.MACAddress) > 0 {
					_ = tg.sendRandomUDP(
						device,
						dst,
					) // error is non-critical for traffic simulation
				}
			}
		}

		// Inter-packet jitter is interruptible so Stop() doesn't block waiting for
		// up to maxRandomDelayMs before this loop yields.
		delay := time.Duration(simRand.IntN(maxRandomDelayMs)) * time.Millisecond
		if !tg.sleepInterruptible(delay) {
			return
		}
	}

	if tg.debugLevel >= debugLevelVerbose {
		logger.Debug("Generated random packets", "device", device.Config.Name, "count", packetCount)
	}
}

func (tg *TrafficGenerator) sendGratuitousARP(device *SimulatedDevice) error {
	mac := device.Config.MACAddress
	if len(mac) == 0 {
		return errors.New("device has no MAC address")
	}
	if len(device.Config.IPAddresses) == 0 {
		return errors.New("device has no IP address")
	}

	ip := device.Config.IPAddresses[0].To4()
	if ip == nil {
		return errors.New("device has no IPv4 address")
	}

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
		HwAddressSize:     macAddressLen,
		ProtAddressSize:   ipv4AddressLen,
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
	eth, ipLayer, icmpLayer := buildPingLayers(src, dst)
	payload := []byte("NIAC-Go ping test data")

	buffer, err := serializeLayers(eth, ipLayer, icmpLayer, gopacket.Payload(payload))
	if err != nil {
		return fmt.Errorf("failed to serialize ping: %w", err)
	}

	tg.stack.Send(&protocols.Packet{Buffer: buffer, Length: len(buffer), Device: src.Config})
	tg.simulator.IncrementCounter(src.Config.Name, "packets_sent")

	return nil
}

// buildPingLayers creates the Ethernet, IP, and ICMP layers for a ping packet.
func buildPingLayers(src, dst *SimulatedDevice) (*layers.Ethernet, *layers.IPv4, *layers.ICMPv4) {
	eth := &layers.Ethernet{
		SrcMAC:       src.Config.MACAddress,
		DstMAC:       dst.Config.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ipLayer := &layers.IPv4{
		Version:  ipv4Version,
		IHL:      ipv4IHL,
		TTL:      defaultTTL,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    src.Config.IPAddresses[0].To4(),
		DstIP:    dst.Config.IPAddresses[0].To4(),
	}
	icmpLayer := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
		Id:       safeconv.Uint16(simRand.IntN(maxUint16PlusOne)),
		Seq:      safeconv.Uint16(simRand.IntN(maxUint16PlusOne)),
	}
	return eth, ipLayer, icmpLayer
}

// serializeLayers serializes packet layers into a byte buffer.
func serializeLayers(layersList ...gopacket.SerializableLayer) ([]byte, error) {
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buffer, opts, layersList...); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// sendBroadcastARP sends a broadcast ARP request.
func (tg *TrafficGenerator) sendBroadcastARP(src *SimulatedDevice) error {
	// Pick a random IP to query
	randomIP := []byte{192, 168, 1, safeconv.Byte(simRand.IntN(maxHostNumber) + 1)}

	eth := &layers.Ethernet{
		SrcMAC:       src.Config.MACAddress,
		DstMAC:       []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}

	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     macAddressLen,
		ProtAddressSize:   ipv4AddressLen,
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
		safeconv.Byte(simRand.IntN(multicastMask)),
		safeconv.Byte(simRand.IntN(byteRange)),
		safeconv.Byte(simRand.IntN(byteRange)),
	}

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
		Version:  ipv4Version,
		IHL:      ipv4IHL,
		TTL:      defaultTTL,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    src.Config.IPAddresses[0].To4(),
		DstIP:    dst.Config.IPAddresses[0].To4(),
	}

	udpLayer := &layers.UDP{
		SrcPort: layers.UDPPort(safeconv.Uint16(simRand.IntN(ephemeralPortMax) + ephemeralPortMin)),
		DstPort: layers.UDPPort(safeconv.Uint16(simRand.IntN(ephemeralPortMax) + ephemeralPortMin)),
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
