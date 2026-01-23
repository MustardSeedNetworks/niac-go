// Package capture provides network packet capture and injection functionality
package capture

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// Capture engine constants.
const (
	snapshotLength    = 1600 // pcap snapshot length in bytes
	captureTimeoutMs  = 100  // capture timeout for responsive shutdown
	debugLevelVerbose = 3    // debug level for verbose packet logging
	macAddressSize    = 6    // MAC/hardware address size in bytes
	ipv4AddressSize   = 4    // IPv4 protocol address size in bytes
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
	// Open interface in promiscuous mode with timeout
	// Use 100ms timeout to allow responsive shutdown on Ctrl+C
	handle, err := pcap.OpenLive(
		interfaceName,
		snapshotLength,                    // snapshot length
		true,                              // promiscuous mode
		captureTimeoutMs*time.Millisecond, // timeout for responsive shutdown
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open interface %s: %w", interfaceName, err)
	}

	return &Engine{
		interfaceName: interfaceName,
		handle:        handle,
		debugLevel:    debugLevel,
	}, nil
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
func (e *Engine) StartCapture(handler func(gopacket.Packet)) error {
	packetSource := gopacket.NewPacketSource(e.handle, e.handle.LinkType())

	if e.debugLevel >= 1 {
		logger := slog.Default()
		logger.Info("Started packet capture", "interface", e.interfaceName)
	}

	for packet := range packetSource.Packets() {
		handler(packet)
	}

	return nil
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

	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     macAddressSize,
		ProtAddressSize:   ipv4AddressSize,
		Operation:         operation,
		SourceHwAddress:   srcMAC,
		SourceProtAddress: []byte(srcIP),
		DstHwAddress:      dstMAC,
		DstProtAddress:    []byte(dstIP),
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
	iface, err := GetInterface(e.interfaceName)
	if err != nil {
		return nil, err
	}

	// Get first MAC address from interface
	for _, addr := range iface.Addresses {
		if len(addr.Broadaddr) == macAddressSize {
			return addr.Broadaddr, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrNoMACAddressFound, e.interfaceName)
}

// RateLimiter controls packet sending rate.
type RateLimiter struct {
	packetsPerSecond int
	ticker           *time.Ticker
	tokens           chan struct{}
	done             chan struct{} // Signals goroutine to stop
	wg               sync.WaitGroup
	stopped          uint32 // Atomic flag for idempotent Stop()
}

// NewRateLimiter creates a rate limiter.
func NewRateLimiter(packetsPerSecond int) *RateLimiter {
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
	if atomic.CompareAndSwapUint32(&rl.stopped, 0, 1) {
		rl.ticker.Stop()
		close(rl.done)
		rl.wg.Wait() // Wait for goroutine to fully exit
	}
}
