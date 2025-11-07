// Package capture provides network packet capture and injection functionality
package capture

import (
	"fmt"
	"log"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// Engine handles packet capture and injection
type Engine struct {
	interfaceName string
	handle        *pcap.Handle
	debugLevel    int
}

// New creates a new capture engine
func New(interfaceName string, debugLevel int) (*Engine, error) {
	// Open interface in promiscuous mode
	handle, err := pcap.OpenLive(
		interfaceName,
		1600, // snapshot length
		true, // promiscuous mode
		pcap.BlockForever,
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

// Close closes the capture engine
func (e *Engine) Close() {
	if e.handle != nil {
		e.handle.Close()
	}
}

// SendPacket sends a raw packet on the interface
func (e *Engine) SendPacket(packet []byte) error {
	if err := e.handle.WritePacketData(packet); err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	if e.debugLevel >= 3 {
		log.Printf("Sent packet: %d bytes", len(packet))
	}

	return nil
}

// SendEthernet sends an Ethernet frame
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

	if err := gopacket.SerializeLayers(buf, opts, eth, gopacket.Payload(payload)); err != nil {
		return fmt.Errorf("failed to serialize packet: %w", err)
	}

	return e.SendPacket(buf.Bytes())
}

// ReadPacket reads a single packet from the interface
// Returns the packet data or nil on timeout/error
func (e *Engine) ReadPacket(buffer []byte) ([]byte, error) {
	data, _, err := e.handle.ReadPacketData()
	if err != nil {
		return nil, err
	}

	// Copy to provided buffer if it fits, otherwise return the data directly
	if len(data) <= len(buffer) {
		copy(buffer, data)
		return buffer[:len(data)], nil
	}

	return data, nil
}

// StartCapture starts capturing packets and calls handler for each packet
func (e *Engine) StartCapture(handler func(gopacket.Packet)) error {
	packetSource := gopacket.NewPacketSource(e.handle, e.handle.LinkType())

	if e.debugLevel >= 1 {
		log.Printf("Started packet capture on %s", e.interfaceName)
	}

	for packet := range packetSource.Packets() {
		handler(packet)
	}

	return nil
}

// SetFilter sets a BPF filter on the capture
func (e *Engine) SetFilter(filter string) error {
	return e.handle.SetBPFFilter(filter)
}

// Stats returns capture statistics
func (e *Engine) Stats() (*pcap.Stats, error) {
	stats, err := e.handle.Stats()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	return stats, nil
}

// SendARP sends an ARP packet
func (e *Engine) SendARP(srcMAC, dstMAC []byte, srcIP, dstIP string, isRequest bool) error {
	operation := uint16(layers.ARPReply)
	if isRequest {
		operation = uint16(layers.ARPRequest)
	}

	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
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

	if err := gopacket.SerializeLayers(buf, opts, eth, arp); err != nil {
		return err
	}

	return e.SendPacket(buf.Bytes())
}

// GetInterfaceMAC returns the MAC address of the interface
func (e *Engine) GetInterfaceMAC() ([]byte, error) {
	iface, err := GetInterface(e.interfaceName)
	if err != nil {
		return nil, err
	}

	// Get first MAC address from interface
	for _, addr := range iface.Addresses {
		if len(addr.Broadaddr) == 6 {
			return addr.Broadaddr, nil
		}
	}

	return nil, fmt.Errorf("no MAC address found for interface %s", e.interfaceName)
}

// RateLimiter controls packet sending rate
type RateLimiter struct {
	packetsPerSecond int
	ticker           *time.Ticker
	tokens           chan struct{}
	done             chan struct{} // Signals goroutine to stop
}

// NewRateLimiter creates a rate limiter
func NewRateLimiter(packetsPerSecond int) *RateLimiter {
	rl := &RateLimiter{
		packetsPerSecond: packetsPerSecond,
		tokens:           make(chan struct{}, packetsPerSecond),
		done:             make(chan struct{}),
	}

	// Fill token bucket initially
	for i := 0; i < packetsPerSecond; i++ {
		rl.tokens <- struct{}{}
	}

	// Refill tokens periodically with proper cleanup
	rl.ticker = time.NewTicker(time.Second / time.Duration(packetsPerSecond))
	go func() {
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
	}()

	return rl
}

// Wait blocks until a token is available
func (rl *RateLimiter) Wait() {
	<-rl.tokens
}

// Stop stops the rate limiter and cleans up goroutine
func (rl *RateLimiter) Stop() {
	rl.ticker.Stop()
	close(rl.done) // Signal goroutine to exit
}
