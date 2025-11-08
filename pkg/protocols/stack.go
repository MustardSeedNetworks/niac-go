package protocols

import (
	"fmt"
	"sync"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/capture"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// Stack manages the network protocol stack
type Stack struct {
	capture      *capture.Engine
	config       *config.Config
	devices      *DeviceTable
	serialNumber int
	mu           sync.Mutex

	// Packet queues
	sendQueue chan *Packet
	recvQueue chan *Packet

	// Protocol handlers
	arpHandler     *ARPHandler
	ipHandler      *IPHandler
	icmpHandler    *ICMPHandler
	ipv6Handler    *IPv6Handler
	icmpv6Handler  *ICMPv6Handler
	udpHandler     *UDPHandler
	tcpHandler     *TCPHandler
	dnsHandler     *DNSHandler
	dhcpHandler    *DHCPHandler
	dhcpv6Handler  *DHCPv6Handler
	httpHandler    *HTTPHandler
	ftpHandler     *FTPHandler
	netbiosHandler *NetBIOSHandler
	stpHandler     *STPHandler
	lldpHandler    *LLDPHandler
	cdpHandler     *CDPHandler
	edpHandler     *EDPHandler
	fdpHandler     *FDPHandler

	// Statistics
	stats *Statistics

	// Control
	running  bool
	stopChan chan struct{}
	wg       sync.WaitGroup

	debugConfig *logging.DebugConfig
}

// Statistics holds protocol statistics
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
	Errors          uint64
}

// NewStack creates a new protocol stack
func NewStack(captureEngine *capture.Engine, cfg *config.Config, debugConfig *logging.DebugConfig) *Stack {
	s := &Stack{
		capture:     captureEngine,
		config:      cfg,
		devices:     NewDeviceTable(),
		sendQueue:   make(chan *Packet, 1000),
		recvQueue:   make(chan *Packet, 1000),
		stats:       &Statistics{},
		stopChan:    make(chan struct{}),
		debugConfig: debugConfig,
	}

	// Initialize device table from config
	s.initializeDevices()

	// Create protocol handlers
	s.arpHandler = NewARPHandler(s)
	s.ipHandler = NewIPHandler(s)
	s.icmpHandler = NewICMPHandler(s)
	s.ipv6Handler = NewIPv6Handler(s, debugConfig.GetProtocolLevel(logging.ProtocolIPv6))
	s.icmpv6Handler = NewICMPv6Handler(s, debugConfig.GetProtocolLevel(logging.ProtocolICMPv6))
	s.udpHandler = NewUDPHandler(s)
	s.tcpHandler = NewTCPHandler(s)
	s.dnsHandler = NewDNSHandler(s)
	s.dhcpHandler = NewDHCPHandler(s)
	s.dhcpv6Handler = NewDHCPv6Handler(s)
	s.httpHandler = NewHTTPHandler(s)
	s.ftpHandler = NewFTPHandler(s)
	s.netbiosHandler = NewNetBIOSHandler(s, debugConfig.GetProtocolLevel(logging.ProtocolNetBIOS))
	s.stpHandler = NewSTPHandler(s, debugConfig.GetProtocolLevel(logging.ProtocolSTP))
	s.lldpHandler = NewLLDPHandler(s)
	s.cdpHandler = NewCDPHandler(s)
	s.edpHandler = NewEDPHandler(s)
	s.fdpHandler = NewFDPHandler(s)

	return s
}

// initializeDevices populates the device table from config
func (s *Stack) initializeDevices() {
	for i := range s.config.Devices {
		device := &s.config.Devices[i]

		// Add by MAC
		if len(device.MACAddress) > 0 {
			s.devices.AddByMAC(device.MACAddress, device)
		}

		// Add by IP addresses
		for _, ip := range device.IPAddresses {
			s.devices.AddByIP(ip, device)
		}

		// Configure DHCP server if device has DHCP config
		if device.DHCPConfig != nil {
			// Set pool if configured
			if device.DHCPConfig.PoolStart != nil && device.DHCPConfig.PoolEnd != nil {
				s.dhcpHandler.SetPool(device.DHCPConfig.PoolStart, device.DHCPConfig.PoolEnd)
			}

			// Set server configuration
			serverIP := device.DHCPConfig.ServerIdentifier
			if serverIP == nil && len(device.IPAddresses) > 0 {
				serverIP = device.IPAddresses[0] // Use first IP as server ID if not specified
			}
			gateway := device.DHCPConfig.Router
			dnsServers := device.DHCPConfig.DomainNameServer
			domain := device.DHCPConfig.DomainName

			s.dhcpHandler.SetServerConfig(serverIP, gateway, dnsServers, domain)

			// Set advanced options
			s.dhcpHandler.SetAdvancedOptions(
				device.DHCPConfig.NTPServers,
				device.DHCPConfig.DomainSearch,
				device.DHCPConfig.TFTPServerName,
				device.DHCPConfig.BootfileName,
				device.DHCPConfig.VendorSpecific,
			)

			if s.debugConfig.GetGlobal() >= 1 {
				fmt.Printf("Configured DHCP server for device %s\n", device.Name)
			}
		}
	}

	if s.debugConfig.GetGlobal() >= 1 {
		fmt.Printf("Initialized %d devices from configuration\n", len(s.config.Devices))
	}
}

// Start starts the protocol stack processing
func (s *Stack) Start() error {
	if s.running {
		return fmt.Errorf("stack already running")
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

	if s.debugConfig.GetGlobal() >= 1 {
		fmt.Println("Protocol stack started")
	}

	return nil
}

// Stop stops the protocol stack
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

	if s.debugConfig.GetGlobal() >= 1 {
		fmt.Println("Protocol stack stopped")
	}
}

// receiveThread receives packets from the network
func (s *Stack) receiveThread() {
	defer s.wg.Done()

	buffer := make([]byte, 65536)

	for s.running {
		select {
		case <-s.stopChan:
			return
		default:
			// Read packet (non-blocking with timeout handled by pcap)
			data, err := s.capture.ReadPacket(buffer)
			if err != nil {
				if s.debugConfig.GetGlobal() >= 3 {
					fmt.Printf("Error reading packet: %v\n", err)
				}
				continue
			}

			if len(data) == 0 {
				continue
			}

			// Parse packet
			s.mu.Lock()
			s.serialNumber++
			serialNum := s.serialNumber
			s.mu.Unlock()

			pkt, err := ParsePacket(data, serialNum)
			if err != nil {
				s.stats.mu.Lock()
				s.stats.Errors++
				s.stats.mu.Unlock()
				continue
			}

			s.stats.mu.Lock()
			s.stats.PacketsReceived++
			s.stats.mu.Unlock()

			// Queue for decoding
			select {
			case s.recvQueue <- pkt:
			default:
				// Queue full, drop packet
				if s.debugConfig.GetGlobal() >= 2 {
					fmt.Println("Receive queue full, dropping packet")
				}
			}
		}
	}
}

// decodeThread decodes and routes packets to protocol handlers
func (s *Stack) decodeThread() {
	defer s.wg.Done()

	for s.running {
		select {
		case <-s.stopChan:
			return
		case pkt := <-s.recvQueue:
			s.decodePacket(pkt)
		case <-time.After(100 * time.Millisecond):
			// Periodic check
		}
	}
}

// decodePacket decodes a packet and routes to appropriate handler
func (s *Stack) decodePacket(pkt *Packet) {
	// Check for STP (multicast MAC 01:80:C2:00:00:00)
	dstMAC := pkt.GetDestMAC()
	if len(dstMAC) == 6 && dstMAC[0] == 0x01 && dstMAC[1] == 0x80 &&
		dstMAC[2] == 0xC2 && dstMAC[3] == 0x00 && dstMAC[4] == 0x00 && dstMAC[5] == 0x00 {
		s.stpHandler.HandlePacket(pkt)
		return
	}

	// Check for LLDP (multicast MAC 01:80:C2:00:00:0E)
	if len(dstMAC) == 6 && dstMAC[0] == 0x01 && dstMAC[1] == 0x80 &&
		dstMAC[2] == 0xC2 && dstMAC[3] == 0x00 && dstMAC[4] == 0x00 && dstMAC[5] == 0x0E {
		s.lldpHandler.HandlePacket(pkt)
		return
	}

	// Check for CDP (multicast MAC 01:00:0C:CC:CC:CC)
	if len(dstMAC) == 6 && dstMAC[0] == 0x01 && dstMAC[1] == 0x00 &&
		dstMAC[2] == 0x0C && dstMAC[3] == 0xCC && dstMAC[4] == 0xCC && dstMAC[5] == 0xCC {
		s.cdpHandler.HandlePacket(pkt)
		return
	}

	// Check for EDP (multicast MAC 00:E0:2B:00:00:00)
	if len(dstMAC) == 6 && dstMAC[0] == 0x00 && dstMAC[1] == 0xE0 &&
		dstMAC[2] == 0x2B && dstMAC[3] == 0x00 && dstMAC[4] == 0x00 && dstMAC[5] == 0x00 {
		s.edpHandler.HandlePacket(pkt)
		return
	}

	// Check for FDP (multicast MAC 01:E0:52:CC:CC:CC)
	if len(dstMAC) == 6 && dstMAC[0] == 0x01 && dstMAC[1] == 0xE0 &&
		dstMAC[2] == 0x52 && dstMAC[3] == 0xCC && dstMAC[4] == 0xCC && dstMAC[5] == 0xCC {
		s.fdpHandler.HandlePacket(pkt)
		return
	}

	// Get EtherType
	etherType := pkt.GetEtherType()

	// Check for VLAN
	offset := SizeOfMac * 2
	if etherType == EtherTypeVLAN {
		// VLAN present, get actual EtherType
		offset += 4
		etherType = pkt.Get16(offset)
	}

	if s.debugConfig.GetGlobal() >= 3 {
		fmt.Printf("Decoding packet sn=%d etherType=0x%04x\n", pkt.SerialNumber, etherType)
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
		if s.debugConfig.GetGlobal() >= 2 {
			fmt.Printf("Unknown EtherType 0x%04x sn=%d\n", etherType, pkt.SerialNumber)
		}
	}
}

// sendThread sends packets to the network
func (s *Stack) sendThread() {
	defer s.wg.Done()

	for s.running {
		select {
		case <-s.stopChan:
			return
		case pkt := <-s.sendQueue:
			s.sendPacket(pkt)
		case <-time.After(100 * time.Millisecond):
			// Periodic check
		}
	}
}

// sendPacket sends a packet to the network
func (s *Stack) sendPacket(pkt *Packet) {
	if pkt.Length == 0 {
		pkt.Length = len(pkt.Buffer)
	}

	err := s.capture.SendPacket(pkt.Buffer[:pkt.Length])
	if err != nil {
		if s.debugConfig.GetGlobal() >= 2 {
			fmt.Printf("Error sending packet sn=%d: %v\n", pkt.SerialNumber, err)
		}
		s.stats.mu.Lock()
		s.stats.Errors++
		s.stats.mu.Unlock()
		return
	}

	s.stats.mu.Lock()
	s.stats.PacketsSent++
	s.stats.mu.Unlock()

	if s.debugConfig.GetGlobal() >= 3 {
		fmt.Printf("Sent packet sn=%d length=%d\n", pkt.SerialNumber, pkt.Length)
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

// babbleThread generates periodic network traffic
func (s *Stack) babbleThread() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for s.running {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			// Generate periodic traffic (ARP announcements, etc.)
			// TODO: Implement periodic traffic generation
		}
	}
}

// Send queues a packet for sending
func (s *Stack) Send(pkt *Packet) {
	select {
	case s.sendQueue <- pkt:
	default:
		if s.debugConfig.GetGlobal() >= 2 {
			fmt.Println("Send queue full, dropping packet")
		}
	}
}

// SendRawPacket queues raw bytes as a packet for sending
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

// GetDevices returns the device table
func (s *Stack) GetDevices() *DeviceTable {
	return s.devices
}

// GetStats returns current statistics (copy without mutex)
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
		Errors:          s.stats.Errors,
	}
}

// IncrementStat increments a specific statistic
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

// GetDebugLevel returns the current global debug level
func (s *Stack) GetDebugLevel() int {
	return s.debugConfig.GetGlobal()
}

// GetProtocolDebugLevel returns the debug level for a specific protocol
func (s *Stack) GetProtocolDebugLevel(protocol string) int {
	return s.debugConfig.GetProtocolLevel(protocol)
}

// GetDebugConfig returns the debug configuration
func (s *Stack) GetDebugConfig() *logging.DebugConfig {
	return s.debugConfig
}

// GetDHCPHandler returns the DHCP handler for configuration
func (s *Stack) GetDHCPHandler() *DHCPHandler {
	return s.dhcpHandler
}

// GetDHCPv6Handler returns the DHCPv6 handler for configuration
func (s *Stack) GetDHCPv6Handler() *DHCPv6Handler {
	return s.dhcpv6Handler
}

// GetDNSHandler returns the DNS handler for configuration
func (s *Stack) GetDNSHandler() *DNSHandler {
	return s.dnsHandler
}
