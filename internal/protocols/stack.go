package protocols

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

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
	running  bool
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

// initializeDevices repopulates device-dependent state from the provided config.
func (s *Stack) initializeDevices(cfg *config.Config) {
	if cfg == nil {
		return
	}

	s.resetDeviceState()

	for i := range cfg.Devices {
		device := &cfg.Devices[i]
		s.registerDevice(device)
	}

	s.applySNMPAddrMappings(cfg)

	s.configMu.Lock()
	s.config = cfg
	s.configMu.Unlock()

	if s.debugConfig.GetGlobal() >= DebugLevelBasic {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"Initialized %d devices from configuration\n",
			len(cfg.Devices),
		)
	}
}

// resetDeviceState resets all device-related state.
func (s *Stack) resetDeviceState() {
	if s.devices == nil {
		s.devices = NewDeviceTable()
	} else {
		s.devices.Reset()
	}

	s.snmpAgents = make(map[*config.Device]*snmpAgentGroup)

	if s.dhcpHandler != nil {
		s.dhcpHandler.Reset()
	}

	if s.dhcpv6Handler != nil {
		s.dhcpv6Handler.Reset()
	}

	if s.dnsHandler != nil {
		s.dnsHandler.Reset()
	}
}

// registerDevice registers a single device with all relevant handlers.
func (s *Stack) registerDevice(device *config.Device) {
	s.registerDeviceAddresses(device)
	s.registerDeviceFeatures(device)
	s.configureDHCPServer(device)
	s.initSNMPAgent(device)

	if device.DNSConfig != nil {
		s.dnsHandler.LoadDeviceDNSConfig(device)
	}
}

// registerDeviceAddresses registers device MAC and IP addresses.
func (s *Stack) registerDeviceAddresses(device *config.Device) {
	if len(device.MACAddress) > 0 {
		s.devices.AddByMAC(device.MACAddress, device)
	}

	for _, ip := range device.IPAddresses {
		s.devices.AddByIP(ip, device)
	}
}

// registerDeviceFeatures registers TTL and forwarding device features.
func (s *Stack) registerDeviceFeatures(device *config.Device) {
	if device.TTLConfig != nil {
		s.devices.RegisterTTL(device)
	}

	if device.SNMPConfig.Dot1DFdbTable != nil || device.SNMPConfig.Dot1QFdbTable != nil {
		s.devices.RegisterForwardingDevice(device)
	}
}

// configureDHCPServer configures DHCP server for a device if it has DHCP config.
func (s *Stack) configureDHCPServer(device *config.Device) {
	if device.DHCPConfig == nil {
		return
	}

	if device.DHCPConfig.PoolStart != nil && device.DHCPConfig.PoolEnd != nil {
		s.dhcpHandler.SetPool(device.DHCPConfig.PoolStart, device.DHCPConfig.PoolEnd)
	}

	serverIP := device.DHCPConfig.ServerIdentifier
	if serverIP == nil && len(device.IPAddresses) > 0 {
		serverIP = device.IPAddresses[0]
	}

	s.dhcpHandler.SetServerConfig(
		serverIP,
		device.DHCPConfig.Router,
		device.DHCPConfig.DomainNameServer,
		device.DHCPConfig.DomainName,
	)

	s.dhcpHandler.SetAdvancedOptions(
		device.DHCPConfig.NTPServers,
		device.DHCPConfig.DomainSearch,
		device.DHCPConfig.TFTPServerName,
		device.DHCPConfig.BootfileName,
		device.DHCPConfig.VendorSpecific,
	)
	s.dhcpHandler.SetStaticLeases(device.DHCPConfig.ClientLeases)

	if s.debugConfig.GetGlobal() >= DebugLevelBasic {
		_, _ = fmt.Fprintf(os.Stdout, "Configured DHCP server for device %s\n", device.Name)
	}
}

// applySNMPAddrMappings applies SNMP agent sharing mappings.
func (s *Stack) applySNMPAddrMappings(cfg *config.Config) {
	for i := range cfg.Devices {
		device := &cfg.Devices[i]
		if device.SNMPConfig.SnmpAddr == nil {
			continue
		}

		targets := s.devices.GetByIP(device.SNMPConfig.SnmpAddr)
		if len(targets) == 0 {
			continue
		}

		if group, ok := s.snmpAgents[targets[0]]; ok {
			s.snmpAgents[device] = group
		}
	}
}

// Start starts the protocol stack processing.
func (s *Stack) Start() error {
	if s.running {
		return ErrStackAlreadyRunning
	}

	s.running = true

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
	if !s.running {
		return
	}

	s.running = false

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

// receiveThread receives packets from the network.
func (s *Stack) receiveThread() {
	defer s.wg.Done()

	buffer := make([]byte, stackReceiveBufferSize)

	for s.running {
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
	defer s.wg.Done()

	for s.running {
		select {
		case <-s.stopChan:
			return
		case pkt := <-s.recvQueue:
			s.decodePacket(pkt)
		case <-time.After(stackSelectTimeoutMs * time.Millisecond):
			// Periodic check
		}
	}
}

// decodePacket decodes a packet and routes to appropriate handler.
func (s *Stack) decodePacket(pkt *Packet) {
	// Try MAC-based routing first (multicast protocols)
	if s.routeByMAC(pkt) {
		return
	}

	// Route by EtherType
	s.routeByEtherType(pkt)
}

// routeByMAC routes packets based on multicast MAC addresses.
// Returns true if packet was handled.
func (s *Stack) routeByMAC(pkt *Packet) bool {
	dstMAC := pkt.GetDestMAC()
	if len(dstMAC) != macAddrLen {
		return false
	}

	// Check for STP (multicast MAC 01:80:C2:00:00:00)
	if matchMAC(
		dstMAC,
		macMulticastIEEE,
		macSecondByteSTP,
		macThirdByteSTP,
		macUnicastZero,
		macUnicastZero,
		macUnicastZero,
	) {
		s.stpHandler.HandlePacket(pkt)
		return true
	}

	// Check for LLDP (multicast MAC 01:80:C2:00:00:0E)
	if matchMAC(
		dstMAC,
		macMulticastIEEE,
		macSecondByteSTP,
		macThirdByteSTP,
		macUnicastZero,
		macUnicastZero,
		macByteLLDP,
	) {
		s.lldpHandler.HandlePacket(pkt)
		return true
	}

	// Check for CDP (multicast MAC 01:00:0C:CC:CC:CC)
	if matchMAC(
		dstMAC,
		macMulticastIEEE,
		macUnicastZero,
		macThirdByteCDP,
		macByteCC,
		macByteCC,
		macByteCC,
	) {
		s.cdpHandler.HandlePacket(pkt)
		return true
	}

	// Check for EDP (multicast MAC 00:E0:2B:00:00:00)
	if matchMAC(
		dstMAC,
		macUnicastZero,
		macSecondByteEDP,
		macThirdByteEDP,
		macUnicastZero,
		macUnicastZero,
		macUnicastZero,
	) {
		s.edpHandler.HandlePacket(pkt)
		return true
	}

	// Check for FDP (multicast MAC 01:E0:52:CC:CC:CC)
	if matchMAC(
		dstMAC,
		macMulticastIEEE,
		macSecondByteEDP,
		macThirdByteFDP,
		macByteCC,
		macByteCC,
		macByteCC,
	) {
		s.fdpHandler.HandlePacket(pkt)
		return true
	}

	return false
}

// matchMAC checks if a MAC address matches the given bytes.
func matchMAC(mac []byte, b0, b1, b2, b3, b4, b5 byte) bool {
	return mac[0] == b0 && mac[1] == b1 && mac[2] == b2 &&
		mac[3] == b3 && mac[4] == b4 && mac[5] == b5
}

// routeByEtherType routes packets based on EtherType.
func (s *Stack) routeByEtherType(pkt *Packet) {
	etherType := pkt.GetEtherType()

	// Check for VLAN
	offset := SizeOfMac * ethMACCount
	if etherType == EtherTypeVLAN {
		// VLAN present, get actual EtherType
		offset += 4
		etherType = pkt.Get16(offset)
	}

	if s.debugConfig.GetGlobal() >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(
			os.Stdout,
			"Decoding packet sn=%d etherType=0x%04x\n",
			pkt.SerialNumber,
			etherType,
		)
	}

	// Route to protocol handler
	switch etherType {
	case EtherTypeARP:
		s.arpHandler.HandlePacket(pkt)
	case EtherTypeIP:
		s.ipHandler.HandlePacket(pkt)
	case EtherTypeIPv6:
		s.ipv6Handler.HandlePacket(pkt)
	case EtherTypeLLDP:
		s.lldpHandler.HandlePacket(pkt)
	case EtherTypeEDP:
		s.edpHandler.HandlePacket(pkt)
	case EtherTypeFDP:
		s.fdpHandler.HandlePacket(pkt)
	default:
		if s.debugConfig.GetGlobal() >= DebugLevelInfo {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"Unknown EtherType 0x%04x sn=%d\n",
				etherType,
				pkt.SerialNumber,
			)
		}
	}
}

// sendThread sends packets to the network.
func (s *Stack) sendThread() {
	defer s.wg.Done()

	for s.running {
		select {
		case <-s.stopChan:
			return
		case pkt := <-s.sendQueue:
			s.sendPacket(pkt)
		case <-time.After(stackSelectTimeoutMs * time.Millisecond):
			// Periodic check
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
		go func() {
			time.Sleep(pkt.LoopTime)

			if s.running {
				s.Send(pkt)
			}
		}()
	}
}

// babbleThread generates periodic network traffic.
func (s *Stack) babbleThread() {
	defer s.wg.Done()

	ticker := time.NewTicker(stackBabbleIntervalSec * time.Second)
	defer ticker.Stop()

	for s.running {
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

	if vlan > 0 {
		eth.EthernetType = layers.EthernetTypeDot1Q
		dot1q := &layers.Dot1Q{
			Priority:       0,
			DropEligible:   false,
			VLANIdentifier: safeUint16(vlan),
			Type:           layers.EthernetTypeARP,
		}
		_ = gopacket.SerializeLayers(buf, opts, eth, dot1q, arp)
	} else {
		_ = gopacket.SerializeLayers(buf, opts, eth, arp)
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

func (s *Stack) initSNMPAgent(device *config.Device) {
	if !snmpEnabled(device.SNMPConfig) {
		return
	}

	debugLevel := s.debugConfig.GetProtocolLevel(logging.ProtocolSNMP)
	group := newSnmpAgentGroup()

	baseCommunity := device.SNMPConfig.Community
	if baseCommunity == "" {
		baseCommunity = config.DefaultSNMPCommunity
	}

	baseAgent := group.Ensure(baseCommunity, device, debugLevel)

	// Load walk files into base community agent
	for _, walkFile := range device.SNMPConfig.WalkFiles {
		err := baseAgent.LoadWalkFile(walkFile)
		if err != nil && debugLevel >= 1 {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"SNMP: failed to load walk file for %s: %v\n",
				device.Name,
				err,
			)
		}
	}

	if device.SNMPConfig.WalkFile != "" {
		err := baseAgent.LoadWalkFile(device.SNMPConfig.WalkFile)
		if err != nil && debugLevel >= 1 {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"SNMP: failed to load walk file for %s: %v\n",
				device.Name,
				err,
			)
		}
	}

	// Load community-specific walk files
	for _, include := range device.SNMPConfig.CommunityIncludes {
		agent := group.Ensure(include.Community, device, debugLevel)
		err := agent.LoadWalkFile(include.WalkFile)
		if err != nil && debugLevel >= 1 {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"SNMP: failed to load walk file for %s (%s): %v\n",
				device.Name,
				include.Community,
				err,
			)
		}
	}

	// Apply AddMib entries to base community (Java uses public)
	for _, mib := range device.SNMPConfig.AddMibs {
		agent := group.Ensure(config.DefaultSNMPCommunity, device, debugLevel)
		err := agent.AddMib(mib.OID, mib.Type, mib.Value)
		if err != nil && debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(
				os.Stdout,
				"SNMP: AddMib failed for %s oid=%s err=%v\n",
				device.Name,
				mib.OID,
				err,
			)
		}
	}

	s.snmpAgents[device] = group
}

func snmpEnabled(cfg config.SNMPConfig) bool {
	if cfg.Community != "" || cfg.WalkFile != "" || len(cfg.WalkFiles) > 0 || cfg.SysName != "" ||
		cfg.SysDescr != "" || cfg.SysContact != "" || cfg.SysLocation != "" {
		return true
	}

	if len(cfg.AddMibs) > 0 || len(cfg.CommunityIncludes) > 0 || len(cfg.AccessList) > 0 ||
		cfg.SnmpAddr != nil {
		return true
	}

	if cfg.Dot1DFdbTable != nil || cfg.Dot1QFdbTable != nil {
		return true
	}

	if cfg.Traps != nil && cfg.Traps.Enabled {
		return true
	}

	return false
}

func (s *Stack) getSNMPAgents(device *config.Device) *snmpAgentGroup {
	if s == nil {
		return nil
	}

	return s.snmpAgents[device]
}

func (s *Stack) updateFDBTables(mac net.HardwareAddr) {
	if s == nil || len(mac) == 0 {
		return
	}

	decMac, hexMac := formatMACForFDB(mac)

	for _, device := range s.devices.GetForwardingDevices() {
		if device == nil {
			continue
		}

		s.updateDeviceFDBTables(device, decMac, hexMac)
	}
}

// formatMACForFDB formats a MAC address for FDB table entries.
func formatMACForFDB(mac net.HardwareAddr) (string, string) {
	var decMacBuilder, hexMacBuilder strings.Builder

	for _, b := range mac {
		decMacBuilder.WriteString(".")
		decMacBuilder.WriteString(strconv.Itoa(int(b)))
		_, _ = fmt.Fprintf(&hexMacBuilder, "%02X ", b)
	}

	return decMacBuilder.String(), hexMacBuilder.String()
}

// updateDeviceFDBTables updates FDB tables for a single device.
func (s *Stack) updateDeviceFDBTables(device *config.Device, decMac, hexMac string) {
	group := s.ensureSNMPAgentGroup(device)
	debugLevel := s.debugConfig.GetProtocolLevel(logging.ProtocolSNMP)
	baseCommunity := s.getBaseCommunity(device)

	s.updateDot1DFdbTable(device, group, baseCommunity, decMac, hexMac, debugLevel)
	s.updateDot1QFdbTable(device, group, baseCommunity, decMac, hexMac, debugLevel)
}

// ensureSNMPAgentGroup ensures an SNMP agent group exists for a device.
func (s *Stack) ensureSNMPAgentGroup(device *config.Device) *snmpAgentGroup {
	group := s.snmpAgents[device]
	if group == nil {
		group = newSnmpAgentGroup()
		s.snmpAgents[device] = group
	}

	return group
}

// getBaseCommunity returns the base SNMP community for a device.
func (s *Stack) getBaseCommunity(device *config.Device) string {
	if device.SNMPConfig.Community != "" {
		return device.SNMPConfig.Community
	}

	return config.DefaultSNMPCommunity
}

// updateDot1DFdbTable updates the dot1d FDB table for a device.
func (s *Stack) updateDot1DFdbTable(
	device *config.Device,
	group *snmpAgentGroup,
	baseCommunity, decMac, hexMac string,
	debugLevel int,
) {
	if device.SNMPConfig.Dot1DFdbTable == nil {
		return
	}

	port := device.SNMPConfig.Dot1DFdbTable.Port
	vlan := device.SNMPConfig.Dot1DFdbTable.VLAN

	community := baseCommunity
	if vlan > 0 {
		community = fmt.Sprintf("%s@%d", baseCommunity, vlan)
	}

	addressMib := ".1.3.6.1.2.1.17.4.3.1.1" + decMac
	portMib := ".1.3.6.1.2.1.17.4.3.1.2" + decMac
	statusMib := ".1.3.6.1.2.1.17.4.3.1.3" + decMac

	agent := group.Ensure(community, device, debugLevel)
	_ = agent.AddMib(addressMib, "STRING", "fixed("+hexMac+")")
	_ = agent.AddMib(portMib, "INTEGER", fmt.Sprintf("fixed(%d)", port))
	_ = agent.AddMib(statusMib, "INTEGER", "fixed(3)")
}

// updateDot1QFdbTable updates the dot1q FDB table for a device.
func (s *Stack) updateDot1QFdbTable(
	device *config.Device,
	group *snmpAgentGroup,
	baseCommunity, decMac, hexMac string,
	debugLevel int,
) {
	if device.SNMPConfig.Dot1QFdbTable == nil {
		return
	}

	port := device.SNMPConfig.Dot1QFdbTable.Port
	vlan := device.SNMPConfig.Dot1QFdbTable.VLAN

	if vlan <= 0 {
		return
	}

	addressMib := fmt.Sprintf(".1.3.6.1.2.1.17.7.1.2.2.1.1.%d%s", vlan, decMac)
	portMib := fmt.Sprintf(".1.3.6.1.2.1.17.7.1.2.2.1.2.%d%s", vlan, decMac)
	statusMib := fmt.Sprintf(".1.3.6.1.2.1.17.7.1.2.2.1.3.%d%s", vlan, decMac)

	agent := group.Ensure(baseCommunity, device, debugLevel)
	_ = agent.AddMib(addressMib, "STRING", "fixed("+hexMac+")")
	_ = agent.AddMib(portMib, "INTEGER", fmt.Sprintf("fixed(%d)", port))
	_ = agent.AddMib(statusMib, "INTEGER", "fixed(3)")
}

// IncrementStat increments a specific statistic.
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

func (s *Stack) recordNeighbor(entry NeighborRecord) {
	if s.neighbors == nil {
		return
	}

	s.neighbors.upsert(entry)
}

func (s *Stack) startNeighborCleanupLoop() {
	if s.neighbors == nil {
		return
	}

	s.wg.Go(func() {
		ticker := time.NewTicker(stackNeighborCleanupSec * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.neighbors.cleanupExpired()
			case <-s.stopChan:
				return
			}
		}
	})
}

func (s *Stack) selectDiscoveryDevice(proto string) *config.Device {
	cfg := s.currentConfig()
	if cfg == nil {
		return nil
	}

	for i := range cfg.Devices {
		dev := &cfg.Devices[i]
		if s.isDeviceEnabledForProtocol(dev, proto) {
			return dev
		}
	}

	if len(cfg.Devices) > 0 {
		return &cfg.Devices[0]
	}

	return nil
}

// isDeviceEnabledForProtocol checks if a device is enabled for a discovery protocol.
func (s *Stack) isDeviceEnabledForProtocol(dev *config.Device, proto string) bool {
	switch proto {
	case ProtocolLLDP:
		return dev.LLDPConfig == nil || dev.LLDPConfig.Enabled
	case ProtocolCDP:
		return dev.CDPConfig == nil || dev.CDPConfig.Enabled
	case ProtocolEDP:
		return dev.EDPConfig == nil || dev.EDPConfig.Enabled
	case ProtocolFDP:
		return dev.FDPConfig == nil || dev.FDPConfig.Enabled
	default:
		return true
	}
}
