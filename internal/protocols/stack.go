package protocols

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/MustardSeedNetworks/niac-go/internal/apperr"
	"github.com/MustardSeedNetworks/niac-go/internal/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

const (
	// DefaultQueueBufferSize is the default buffer size for send/receive queues.
	// Increase this for high-traffic scenarios to prevent packet drops.
	// Decrease for memory-constrained environments.
	DefaultQueueBufferSize = 1000

	// Recommended sizes for different scenarios:
	// - Low traffic (< 100 pps): 500
	// - Normal traffic (100-1000 pps): 1000 (default)
	// - High traffic (1000-10000 pps): 5000
	// - Very high traffic (> 10000 pps): 10000.
)

// Stack internal constants.
const (
	stackReceiveBufferSize    = 65536 // Receive buffer size
	stackSelectTimeoutMs      = 100   // Select timeout in milliseconds
	stackBabbleIntervalSec    = 15    // Babble traffic interval in seconds
	stackBabbleDelayMs        = 10    // Delay between babble packets
	stackNeighborCleanupSec   = 30    // Neighbor cleanup interval in seconds
	stackBabbleTargetIPOctet3 = 10    // Babble target IP octet (10.1.1.1)
)

// Multicast MAC address bytes for protocol detection.
const (
	macAddrLen       = 6    // MAC address length in bytes
	ipv4AddrLen      = 4    // IPv4 address length in bytes
	ethMACCount      = 2    // Number of MAC addresses in Ethernet header (src + dst)
	macMulticastIEEE = 0x01 // IEEE multicast first byte (STP, LLDP, CDP, FDP)
	macUnicastZero   = 0x00 // Unicast/EDP first byte
	macSecondByteSTP = 0x80 // Second byte for STP/LLDP multicast
	macSecondByteEDP = 0xE0 // Second byte for EDP/FDP multicast
	macThirdByteSTP  = 0xC2 // Third byte for STP/LLDP (01:80:C2:...)
	macThirdByteCDP  = 0x0C // Third byte for CDP (01:00:0C:...)
	macThirdByteEDP  = 0x2B // Third byte for EDP (00:E0:2B:...)
	macThirdByteFDP  = 0x52 // Third byte for FDP (01:E0:52:...)
	macByteLLDP      = 0x0E // Last byte for LLDP multicast
	macByteCC        = 0xCC // CC byte used in CDP/FDP patterns
)

// Debug level thresholds for consistent logging.
const (
	DebugLevelBasic   = 1 // Basic debug output
	DebugLevelInfo    = 2 // Info-level debug output
	DebugLevelVerbose = 3 // Verbose debug output
	DebugLevelTrace   = 4 // Trace-level debug output
)

// Stack manages the network protocol stack.
type Stack struct {
	capture      *capture.Engine
	config       *config.Config
	configMu     sync.RWMutex
	reloadMu     sync.Mutex
	devices      *DeviceTable
	vlanMode     bool // any device is VLAN-tagged: ignore untagged frames (no native/default replay)
	serialNumber int
	mu           sync.Mutex

	// Packet queues
	sendQueue chan *Packet
	recvQueue chan *Packet

	// Protocol handlers
	arpHandler         *ARPHandler
	ipHandler          *IPHandler
	icmpHandler        *ICMPHandler
	ipv6Handler        *IPv6Handler
	icmpv6Handler      *ICMPv6Handler
	udpHandler         *UDPHandler
	tcpHandler         *TCPHandler
	dnsHandler         *DNSHandler
	dhcpHandler        *DHCPHandler
	dhcpv6Handler      *DHCPv6Handler
	httpHandler        *HTTPHandler
	ftpHandler         *FTPHandler
	netbiosHandler     *NetBIOSHandler
	stpHandler         *STPHandler
	lldpHandler        *LLDPHandler
	cdpHandler         *CDPHandler
	edpHandler         *EDPHandler
	fdpHandler         *FDPHandler
	snmpHandler        *SNMPHandler
	healthCheckHandler *HealthCheckHandler
	iperf3Handler      *IPerf3Handler
	neighbors          *neighborTable

	// Statistics
	stats *Statistics

	// Control
	running  atomic.Bool
	stopChan chan struct{}
	wg       sync.WaitGroup

	debugConfig  *logging.DebugConfig
	snmpAgents   map[*config.Device]*snmpAgentGroup
	errorManager *apperr.StateManager

	// observers receive every packet the stack sees (rx) or sends (tx).
	// The API server registers one to feed its SSE hub. Observers are
	// expected to be cheap; the stack drops to the floor on observer
	// panic via the recover in notifyObservers.
	observerMu sync.RWMutex
	observers  []PacketObserver
}

// PacketObserver receives every packet the stack handles. Direction is
// "rx" for inbound (just decoded) or "tx" for outbound (about to send).
//
// Implementations MUST NOT block — the stack holds an RLock while
// iterating observers, and a slow observer would back-pressure the
// receive thread. Use a buffered channel or goroutine on the consumer
// side if any heavier processing is needed.
type PacketObserver interface {
	OnPacket(direction string, pkt *Packet)
}

// Statistics holds protocol statistics.
type Statistics struct {
	mu              sync.RWMutex
	PacketsReceived uint64
	PacketsSent     uint64
	ARPRequests     uint64
	ARPReplies      uint64
	ICMPRequests    uint64
	ICMPReplies     uint64
	DNSQueries      uint64
	DHCPRequests    uint64
	SNMPQueries     uint64
	Errors          uint64
}

// NewStack creates a new protocol stack.
// configUsesVLANs reports whether any device is assigned a VLAN. When true the
// stack runs "VLAN mode" and ignores untagged frames, so it can never respond on
// the native/default VLAN — preventing a leak (e.g. a rogue DHCP offer) onto an
// untagged management network no matter how the upstream trunk is configured.
func configUsesVLANs(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	for i := range cfg.Devices {
		if cfg.Devices[i].VLAN > 0 {
			return true
		}
	}

	return false
}

func NewStack(
	captureEngine *capture.Engine,
	cfg *config.Config,
	debugConfig *logging.DebugConfig,
) *Stack {
	// FEATURE #124: Use configurable buffer size
	bufferSize := DefaultQueueBufferSize

	stack := &Stack{
		capture:      captureEngine,
		config:       cfg,
		devices:      NewDeviceTable(),
		sendQueue:    make(chan *Packet, bufferSize),
		recvQueue:    make(chan *Packet, bufferSize),
		stats:        &Statistics{},
		stopChan:     make(chan struct{}),
		debugConfig:  debugConfig,
		snmpAgents:   make(map[*config.Device]*snmpAgentGroup),
		neighbors:    newNeighborTable(),
		errorManager: apperr.NewStateManager(),
		vlanMode:     configUsesVLANs(cfg),
	}

	// Create protocol handlers
	stack.arpHandler = NewARPHandler(stack)
	stack.ipHandler = NewIPHandler(stack)
	stack.icmpHandler = NewICMPHandler(stack)
	stack.ipv6Handler = NewIPv6Handler(stack, debugConfig.GetProtocolLevel(logging.ProtocolIPv6))
	stack.icmpv6Handler = NewICMPv6Handler(
		stack,
		debugConfig.GetProtocolLevel(logging.ProtocolICMPv6),
	)
	stack.udpHandler = NewUDPHandler(stack)
	stack.tcpHandler = NewTCPHandler(stack)
	stack.dnsHandler = NewDNSHandler(stack)
	stack.dhcpHandler = NewDHCPHandler(stack)
	stack.dhcpv6Handler = NewDHCPv6Handler(stack)
	stack.httpHandler = NewHTTPHandler(stack)
	stack.ftpHandler = NewFTPHandler(stack)
	stack.netbiosHandler = NewNetBIOSHandler(
		stack,
		debugConfig.GetProtocolLevel(logging.ProtocolNetBIOS),
	)
	stack.stpHandler = NewSTPHandler(stack, debugConfig.GetProtocolLevel(logging.ProtocolSTP))
	stack.lldpHandler = NewLLDPHandler(stack)
	stack.cdpHandler = NewCDPHandler(stack)
	stack.edpHandler = NewEDPHandler(stack)
	stack.fdpHandler = NewFDPHandler(stack)
	stack.snmpHandler = NewSNMPHandler(stack)
	stack.healthCheckHandler = NewHealthCheckHandler(stack)
	stack.iperf3Handler = NewIPerf3Handler(stack)

	// Initialize device table from config (requires handlers for DHCP/SNMP setup)
	stack.initializeDevices(cfg)

	return stack
}

// AddPacketObserver registers an observer for stack packet events.

func (s *Stack) Start() error {
	if !s.running.CompareAndSwap(false, true) {
		return ErrStackAlreadyRunning
	}

	// Start receive thread
	s.wg.Add(1)

	go s.receiveThread()

	// Start decode thread
	s.wg.Add(1)

	go s.decodeThread()

	// Start send thread
	s.wg.Add(1)

	go s.sendThread()

	// Start babble thread (periodic packet generation)
	s.wg.Add(1)

	go s.babbleThread()

	// Start discovery protocol periodic advertisements
	s.lldpHandler.Start()
	s.cdpHandler.Start()
	s.edpHandler.Start()
	s.fdpHandler.Start()
	s.startNeighborCleanupLoop()

	if s.debugConfig.GetGlobal() >= DebugLevelBasic {
		_, _ = fmt.Fprintln(os.Stdout, "Protocol stack started")
	}

	return nil
}

// Stop stops the protocol stack.
func (s *Stack) Stop() {
	if !s.running.CompareAndSwap(true, false) {
		return
	}

	// Stop discovery protocol handlers
	s.lldpHandler.Stop()
	s.cdpHandler.Stop()
	s.edpHandler.Stop()
	s.fdpHandler.Stop()

	close(s.stopChan)
	s.wg.Wait()

	if s.debugConfig.GetGlobal() >= DebugLevelBasic {
		_, _ = fmt.Fprintln(os.Stdout, "Protocol stack stopped")
	}
}

func (s *Stack) Send(pkt *Packet) {
	select {
	case s.sendQueue <- pkt:
	default:
		if s.debugConfig.GetGlobal() >= DebugLevelInfo {
			_, _ = fmt.Fprintln(os.Stdout, "Send queue full, dropping packet")
		}
	}
}

// SendRawPacket queues raw bytes as a packet for sending (untagged).
func (s *Stack) SendRawPacket(data []byte) error {
	return s.SendRawPacketVLAN(data, 0)
}

// SendRawPacketVLAN queues raw bytes as a packet, tagging it onto vlan (1..4094)
// so the reply lands on the same VLAN the request arrived on. vlan <= 0 sends
// untagged.
func (s *Stack) SendRawPacketVLAN(data []byte, vlan int) error {
	s.mu.Lock()
	s.serialNumber++
	serialNum := s.serialNumber
	s.mu.Unlock()

	pkt := &Packet{
		Buffer:       data,
		Length:       len(data),
		SerialNumber: serialNum,
		VLAN:         vlan,
	}

	s.Send(pkt)

	return nil
}

// GetDevices returns the device table.
func (s *Stack) GetDevices() *DeviceTable {
	return s.devices
}

// ReloadConfig applies a new configuration to the running stack.
func (s *Stack) ReloadConfig(cfg *config.Config) error {
	if cfg == nil {
		return ErrNilConfig
	}

	s.reloadMu.Lock()
	defer s.reloadMu.Unlock()

	s.initializeDevices(cfg)

	if s.neighbors != nil {
		s.neighbors.reset()
	}

	if s.debugConfig.GetGlobal() >= DebugLevelBasic {
		_, _ = fmt.Fprintf(os.Stdout, "Protocol stack reloaded (%d devices)\n", len(cfg.Devices))
	}

	return nil
}

// GetStats returns current statistics (copy without mutex).
func (s *Stack) GetStats() Statistics {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	// Return copy of data without mutex
	return Statistics{
		PacketsReceived: s.stats.PacketsReceived,
		PacketsSent:     s.stats.PacketsSent,
		ARPRequests:     s.stats.ARPRequests,
		ARPReplies:      s.stats.ARPReplies,
		ICMPRequests:    s.stats.ICMPRequests,
		ICMPReplies:     s.stats.ICMPReplies,
		DNSQueries:      s.stats.DNSQueries,
		DHCPRequests:    s.stats.DHCPRequests,
		SNMPQueries:     s.stats.SNMPQueries,
		Errors:          s.stats.Errors,
	}
}

func (s *Stack) IncrementStat(stat string) {
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()

	switch stat {
	case "arp_requests":
		s.stats.ARPRequests++
	case "arp_replies":
		s.stats.ARPReplies++
	case "icmp_requests":
		s.stats.ICMPRequests++
	case "icmp_replies":
		s.stats.ICMPReplies++
	case "dns_queries":
		s.stats.DNSQueries++
	case "dhcp_requests":
		s.stats.DHCPRequests++
	}
}

// GetDebugLevel returns the current global debug level.
func (s *Stack) GetDebugLevel() int {
	return s.debugConfig.GetGlobal()
}

// GetProtocolDebugLevel returns the debug level for a specific protocol.
func (s *Stack) GetProtocolDebugLevel(protocol string) int {
	return s.debugConfig.GetProtocolLevel(protocol)
}

// GetDebugConfig returns the debug configuration.
func (s *Stack) GetDebugConfig() *logging.DebugConfig {
	return s.debugConfig
}

// GetDHCPHandler returns the DHCP handler for configuration.
func (s *Stack) GetDHCPHandler() *DHCPHandler {
	return s.dhcpHandler
}

// GetDHCPv6Handler returns the DHCPv6 handler for configuration.
func (s *Stack) GetDHCPv6Handler() *DHCPv6Handler {
	return s.dhcpv6Handler
}

// GetDNSHandler returns the DNS handler for configuration.
func (s *Stack) GetDNSHandler() *DNSHandler {
	return s.dnsHandler
}

func (s *Stack) currentConfig() *config.Config {
	s.configMu.RLock()
	defer s.configMu.RUnlock()

	return s.config
}

// GetNeighbors returns a snapshot of the current neighbor table.
func (s *Stack) GetNeighbors() []NeighborRecord {
	if s.neighbors == nil {
		return nil
	}

	return s.neighbors.list()
}

// GetErrorManager returns the error state manager.
func (s *Stack) GetErrorManager() *apperr.StateManager {
	return s.errorManager
}

// SetCaptureFilter sets a BPF filter on the underlying capture engine.
func (s *Stack) SetCaptureFilter(filter string) error {
	if s.capture == nil {
		return errors.New("capture engine not initialized")
	}

	return s.capture.SetFilter(filter)
}

// CaptureFilter returns the currently active BPF filter expression.
func (s *Stack) CaptureFilter() string {
	if s.capture == nil {
		return ""
	}

	return s.capture.Filter()
}
