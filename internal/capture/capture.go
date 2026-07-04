// Package capture provides network packet capture and injection functionality
package capture

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

// Capture engine constants.
const (
	snapshotLength    = 1600     // pcap snapshot length in bytes
	captureBufferSize = 16 << 20 // 16 MiB kernel ring buffer
	captureTimeoutMs  = 100      // capture timeout for responsive shutdown
	debugLevelVerbose = 3        // debug level for verbose packet logging
	macAddressSize    = 6        // MAC/hardware address size in bytes
	ipv4AddressSize   = 4        // IPv4 protocol address size in bytes
)

// ErrNoMACAddressFound is returned when an interface has no MAC address.
var ErrNoMACAddressFound = errors.New("no MAC address found for interface")

// Engine handles packet capture and injection.
type Engine struct {
	interfaceName string
	handle        *pcap.Handle
	debugLevel    int
	activeFilter  string
	filterMu      sync.RWMutex
}

// New creates a new capture engine.
func New(interfaceName string, debugLevel int) (*Engine, error) {
	handle, err := openImmediate(interfaceName)
	if err != nil {
		return nil, err
	}

	return &Engine{
		interfaceName: interfaceName,
		handle:        handle,
		debugLevel:    debugLevel,
	}, nil
}

// openImmediate opens the interface in promiscuous, immediate mode.
//
// Immediate mode is essential for a simulator: without it, libpcap holds
// captured frames in the kernel buffer until it fills or the read timeout
// expires, adding up to captureTimeoutMs of latency to every request on a
// quiet link. That turns a 30k-OID SNMP walk into a slow, timing-out crawl.
// Immediate mode delivers each frame as it arrives while the read timeout
// still bounds shutdown responsiveness.
//
// Falls back to OpenLive if the platform's libpcap cannot honor the inactive
// handle builder, so behavior degrades to correct-but-slower rather than
// failing to start.
func openImmediate(interfaceName string) (*pcap.Handle, error) {
	inactive, err := pcap.NewInactiveHandle(interfaceName)
	if err != nil {
		return openLiveFallback(interfaceName)
	}
	defer inactive.CleanUp()

	// Any failure to configure the inactive handle degrades to the legacy
	// buffered open so the simulator still starts (correct but slower).
	configure := []func() error{
		func() error { return inactive.SetSnapLen(snapshotLength) },
		func() error { return inactive.SetPromisc(true) },
		func() error { return inactive.SetTimeout(captureTimeoutMs * time.Millisecond) },
		func() error { return inactive.SetImmediateMode(true) },
		// A CyberScope discovery walks every device concurrently — hundreds of
		// GETNEXT round-trips per big switch/router arriving in bursts. The
		// default kernel ring is small enough that a burst overflows it and
		// drops frames the reader never sees, which stalls a big-device walk
		// mid-stream (it times out with "no SNMP details") while a tiny AP's
		// short walk survives. A generous ring absorbs the burst.
		func() error { return inactive.SetBufferSize(captureBufferSize) },
	}
	for _, apply := range configure {
		if applyErr := apply(); applyErr != nil {
			return openLiveFallback(interfaceName)
		}
	}

	handle, err := inactive.Activate()
	if err != nil {
		return openLiveFallback(interfaceName)
	}

	return handle, nil
}

// openLiveFallback opens the interface with the legacy buffered API.
func openLiveFallback(interfaceName string) (*pcap.Handle, error) {
	handle, err := pcap.OpenLive(
		interfaceName,
		snapshotLength,
		true,
		captureTimeoutMs*time.Millisecond,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open interface %s: %w", interfaceName, err)
	}

	return handle, nil
}

// Close closes the capture engine.
func (e *Engine) Close() {
	if e.handle != nil {
		e.handle.Close()
	}
}

// SendPacket sends a raw packet on the interface.
func (e *Engine) SendPacket(packet []byte) error {
	err := e.handle.WritePacketData(packet)
	if err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	if e.debugLevel >= debugLevelVerbose {
		logger := slog.Default()
		logger.Debug("Sent packet", "bytes", len(packet))
	}

	return nil
}

// SendEthernet sends an Ethernet frame.
func (e *Engine) SendEthernet(dstMAC, srcMAC []byte, etherType uint16, payload []byte) error {
	// Build Ethernet layer
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetType(etherType),
	}

	// Serialize packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buf, opts, eth, gopacket.Payload(payload))
	if err != nil {
		return fmt.Errorf("failed to serialize packet: %w", err)
	}

	return e.SendPacket(buf.Bytes())
}

// ReadPacket reads a single packet from the interface
// Returns the packet data or nil on timeout/error
// Timeouts are not treated as errors to allow responsive shutdown.
func (e *Engine) ReadPacket(buffer []byte) ([]byte, error) {
	data, _, err := e.handle.ReadPacketData()
	if err != nil {
		// Timeout is expected and allows responsive shutdown
		if errors.Is(err, pcap.NextErrorTimeoutExpired) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to read packet data: %w", err)
	}

	// Copy to provided buffer if it fits, otherwise return the data directly
	if len(data) <= len(buffer) {
		copy(buffer, data)

		return buffer[:len(data)], nil
	}

	return data, nil
}

// StartCapture starts capturing packets and calls handler for each packet.
// Blocks until the underlying handle is closed (e.g. via Engine.Close).
func (e *Engine) StartCapture(handler func(gopacket.Packet)) error {
	return e.StartCaptureContext(context.Background(), handler)
}

// StartCaptureContext is the cancellable form of StartCapture. When ctx is
// cancelled, the loop exits at the next packet boundary or after captureTimeoutMs
// of idle time (because pcap.Handle was opened with that timeout).
func (e *Engine) StartCaptureContext(ctx context.Context, handler func(gopacket.Packet)) error {
	packetSource := gopacket.NewPacketSource(e.handle, e.handle.LinkType())

	if e.debugLevel >= 1 {
		slog.Default().InfoContext(ctx, "Started packet capture", "interface", e.interfaceName)
	}

	packets := packetSource.Packets()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case packet, ok := <-packets:
			if !ok {
				return nil
			}
			handler(packet)
		}
	}
}

// SetFilter sets a BPF filter on the capture.
func (e *Engine) SetFilter(filter string) error {
	if err := e.handle.SetBPFFilter(filter); err != nil {
		return fmt.Errorf("failed to set BPF filter: %w", err)
	}

	e.filterMu.Lock()
	e.activeFilter = filter
	e.filterMu.Unlock()

	return nil
}

// Filter returns the currently active BPF filter expression.
func (e *Engine) Filter() string {
	e.filterMu.RLock()
	defer e.filterMu.RUnlock()

	return e.activeFilter
}

// Stats returns capture statistics.
func (e *Engine) Stats() (*pcap.Stats, error) {
	stats, err := e.handle.Stats()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return stats, nil
}

// SendARP sends an ARP packet.
func (e *Engine) SendARP(srcMAC, dstMAC []byte, srcIP, dstIP string, isRequest bool) error {
	operation := uint16(layers.ARPReply)
	if isRequest {
		operation = uint16(layers.ARPRequest)
	}

	srcIPBytes := net.ParseIP(srcIP).To4()
	if srcIPBytes == nil {
		return fmt.Errorf("invalid source IP address: %s", srcIP)
	}

	dstIPBytes := net.ParseIP(dstIP).To4()
	if dstIPBytes == nil {
		return fmt.Errorf("invalid destination IP address: %s", dstIP)
	}

	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     macAddressSize,
		ProtAddressSize:   ipv4AddressSize,
		Operation:         operation,
		SourceHwAddress:   srcMAC,
		SourceProtAddress: srcIPBytes,
		DstHwAddress:      dstMAC,
		DstProtAddress:    dstIPBytes,
	}

	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeARP,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buf, opts, eth, arp)
	if err != nil {
		return fmt.Errorf("failed to serialize ARP packet: %w", err)
	}

	return e.SendPacket(buf.Bytes())
}

// GetInterfaceMAC returns the MAC address of the interface.
func (e *Engine) GetInterfaceMAC() ([]byte, error) {
	iface, err := net.InterfaceByName(e.interfaceName)
	if err != nil {
		return nil, fmt.Errorf("lookup interface %s: %w", e.interfaceName, err)
	}

	if len(iface.HardwareAddr) != macAddressSize {
		return nil, fmt.Errorf("%w: %s", ErrNoMACAddressFound, e.interfaceName)
	}

	return iface.HardwareAddr, nil
}

// RateLimiter controls packet sending rate.
type RateLimiter struct {
	packetsPerSecond int
	ticker           *time.Ticker
	tokens           chan struct{}
	done             chan struct{} // Signals goroutine to stop
	wg               sync.WaitGroup
	stopped          atomic.Uint32 // Atomic flag for idempotent Stop()
}

// NewRateLimiter creates a rate limiter.
// If packetsPerSecond is <= 0, it defaults to 1 to prevent panics.
func NewRateLimiter(packetsPerSecond int) *RateLimiter {
	if packetsPerSecond <= 0 {
		packetsPerSecond = 1
	}
	rl := &RateLimiter{
		packetsPerSecond: packetsPerSecond,
		tokens:           make(chan struct{}, packetsPerSecond),
		done:             make(chan struct{}),
	}

	// Fill token bucket initially
	for range packetsPerSecond {
		rl.tokens <- struct{}{}
	}

	// Refill tokens periodically with proper cleanup
	rl.ticker = time.NewTicker(time.Second / time.Duration(packetsPerSecond))
	rl.wg.Go(func() {
		for {
			select {
			case <-rl.ticker.C:
				select {
				case rl.tokens <- struct{}{}:
				default:
					// Bucket full
				}
			case <-rl.done:
				return // Clean exit
			}
		}
	})

	return rl
}

// Wait blocks until a token is available.
func (rl *RateLimiter) Wait() {
	<-rl.tokens
}

// Stop stops the rate limiter and cleans up goroutine
// This method is idempotent and safe to call multiple times.
func (rl *RateLimiter) Stop() {
	if rl.stopped.CompareAndSwap(0, 1) {
		rl.ticker.Stop()
		close(rl.done)
		rl.wg.Wait() // Wait for goroutine to fully exit
	}
}
