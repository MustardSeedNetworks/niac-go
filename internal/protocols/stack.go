package protocols

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/krisarmstrong/niac-go/internal/apperr"
	"github.com/krisarmstrong/niac-go/internal/capture"
	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/logging"
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

// Start starts the protocol stack processing.
func (s *Stack) Start() error {
	if !s.running.CompareAndSwap(false, true) {
		return ErrStackAlreadyRunning
	}

	// Start receive thread
	s.wg.Go(s.receiveThread)

	// Start decode thread
	s.wg.Go(s.decodeThread)

	// Start send thread
	s.wg.Go(s.sendThread)

	// Start babble thread (periodic packet generation)
	s.wg.Go(s.babbleThread)

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
