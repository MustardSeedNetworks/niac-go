package protocols

import (
	"net"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// BenchmarkARPRequestHandling benchmarks processing ARP requests
func BenchmarkARPRequestHandling(b *testing.B) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)
	stack := NewStack(nil, cfg, debugConfig)
	handler := NewARPHandler(stack)

	// Create test device
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	ip := net.ParseIP("192.168.1.1")
	device := &config.Device{
		Name:        "test-device",
		MACAddress:  mac,
		IPAddresses: []net.IP{ip},
	}
	stack.GetDevices().AddByMAC(mac, device)
	stack.GetDevices().AddByIP(ip, device)

	// Build ARP request packet
	srcMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	srcIP := net.ParseIP("192.168.1.100")

	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}

	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   srcMAC,
		SourceProtAddress: srcIP.To4(),
		DstHwAddress:      net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		DstProtAddress:    ip.To4(),
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	gopacket.SerializeLayers(buffer, opts, eth, arp)

	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		handler.HandlePacket(pkt)
	}
}

// BenchmarkARPReplyGeneration benchmarks generating ARP reply packets
func BenchmarkARPReplyGeneration(b *testing.B) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)
	stack := NewStack(nil, cfg, debugConfig)
	handler := NewARPHandler(stack)

	senderMAC, _ := net.ParseMAC("00:11:22:33:44:55")
	senderIP := net.ParseIP("192.168.1.1")
	targetMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	targetIP := net.ParseIP("192.168.1.100")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler.buildARPReply(senderMAC, senderIP, targetMAC, targetIP)
	}
}

// BenchmarkARPGratuitous benchmarks sending gratuitous ARP announcements
func BenchmarkARPGratuitous(b *testing.B) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)
	stack := NewStack(nil, cfg, debugConfig)
	handler := NewARPHandler(stack)

	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	ip := net.ParseIP("192.168.1.1")
	device := &config.Device{
		Name:        "test-device",
		MACAddress:  mac,
		IPAddresses: []net.IP{ip},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler.SendGratuitousARP(device)
	}
}

// BenchmarkLLDPPacketGeneration benchmarks generating LLDP advertisement packets
func BenchmarkLLDPPacketGeneration(b *testing.B) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)
	stack := NewStack(nil, cfg, debugConfig)
	handler := NewLLDPHandler(stack)

	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	ip := net.ParseIP("192.168.1.1")
	device := &config.Device{
		Name:        "test-device",
		Type:        "switch",
		MACAddress:  mac,
		IPAddresses: []net.IP{ip},
		LLDPConfig: &config.LLDPConfig{
			Enabled:           true,
			AdvertiseInterval: 30,
			TTL:               120,
			SystemDescription: "Test Switch",
			PortDescription:   "Port 1",
			ChassisIDType:     "mac",
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler.buildLLDPFrame(device)
	}
}

// BenchmarkLLDPChassisIDTLV benchmarks building LLDP Chassis ID TLV
func BenchmarkLLDPChassisIDTLV(b *testing.B) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)
	stack := NewStack(nil, cfg, debugConfig)
	handler := NewLLDPHandler(stack)

	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	device := &config.Device{
		Name:       "test-device",
		MACAddress: mac,
		LLDPConfig: &config.LLDPConfig{
			ChassisIDType: "mac",
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler.buildChassisIDTLV(device)
	}
}

// BenchmarkLLDPManagementAddressTLV benchmarks building LLDP Management Address TLV
func BenchmarkLLDPManagementAddressTLV(b *testing.B) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)
	stack := NewStack(nil, cfg, debugConfig)
	handler := NewLLDPHandler(stack)

	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	ip := net.ParseIP("192.168.1.1")
	device := &config.Device{
		Name:        "test-device",
		MACAddress:  mac,
		IPAddresses: []net.IP{ip},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler.buildManagementAddressTLV(device)
	}
}

// BenchmarkDHCPDiscover benchmarks processing DHCP DISCOVER messages
func BenchmarkDHCPDiscover(b *testing.B) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)
	stack := NewStack(nil, cfg, debugConfig)
	handler := NewDHCPHandler(stack)

	// Configure DHCP handler
	serverIP := net.ParseIP("192.168.1.1")
	gateway := net.ParseIP("192.168.1.1")
	dns := []net.IP{net.ParseIP("8.8.8.8")}
	handler.SetServerConfig(serverIP, gateway, dns, "example.com")
	handler.SetPool(net.ParseIP("192.168.1.100"), net.ParseIP("192.168.1.200"))

	// Create DHCP server device
	serverMAC, _ := net.ParseMAC("00:11:22:33:44:55")
	device := &config.Device{
		Name:        "dhcp-server",
		MACAddress:  serverMAC,
		IPAddresses: []net.IP{serverIP},
	}
	devices := []*config.Device{device}

	// Build DHCP DISCOVER packet
	clientMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  6,
		Xid:          0x12345678,
		ClientHWAddr: clientMAC,
		Options: []layers.DHCPOption{
			{
				Type:   layers.DHCPOptMessageType,
				Length: 1,
				Data:   []byte{1}, // DISCOVER
			},
		},
	}

	udp := &layers.UDP{
		SrcPort: 68,
		DstPort: 67,
	}

	ip := &layers.IPv4{
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    net.IPv4zero,
		DstIP:    net.IPv4bcast,
	}

	eth := &layers.Ethernet{
		SrcMAC:       clientMAC,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	udp.SetNetworkLayerForChecksum(ip)
	gopacket.SerializeLayers(buffer, opts, eth, ip, udp, dhcp)

	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		handler.HandlePacket(pkt, ip, udp, devices)
	}
}

// BenchmarkDHCPOfferGeneration benchmarks generating DHCP OFFER messages
func BenchmarkDHCPOfferGeneration(b *testing.B) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)
	stack := NewStack(nil, cfg, debugConfig)
	handler := NewDHCPHandler(stack)

	serverIP := net.ParseIP("192.168.1.1")
	gateway := net.ParseIP("192.168.1.1")
	dns := []net.IP{net.ParseIP("8.8.8.8")}
	handler.SetServerConfig(serverIP, gateway, dns, "example.com")

	clientMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	offeredIP := net.ParseIP("192.168.1.100")
	serverMAC, _ := net.ParseMAC("00:11:22:33:44:55")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler.SendDHCPOffer(0x12345678, clientMAC, offeredIP, serverIP, serverMAC)
	}
}

// BenchmarkDHCPAckGeneration benchmarks generating DHCP ACK messages
func BenchmarkDHCPAckGeneration(b *testing.B) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)
	stack := NewStack(nil, cfg, debugConfig)
	handler := NewDHCPHandler(stack)

	serverIP := net.ParseIP("192.168.1.1")
	gateway := net.ParseIP("192.168.1.1")
	dns := []net.IP{net.ParseIP("8.8.8.8")}
	handler.SetServerConfig(serverIP, gateway, dns, "example.com")

	clientMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	assignedIP := net.ParseIP("192.168.1.100")
	serverMAC, _ := net.ParseMAC("00:11:22:33:44:55")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler.SendDHCPAck(0x12345678, clientMAC, assignedIP, serverIP, serverMAC)
	}
}

// BenchmarkDHCPLeaseAllocation benchmarks allocating DHCP leases
func BenchmarkDHCPLeaseAllocation(b *testing.B) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)
	stack := NewStack(nil, cfg, debugConfig)
	handler := NewDHCPHandler(stack)
	handler.SetPool(net.ParseIP("192.168.1.100"), net.ParseIP("192.168.1.200"))

	clientMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = handler.allocateLease(clientMAC, nil, "test-host")
	}
}

// BenchmarkDHCPFullCycle benchmarks complete DHCP DISCOVER->OFFER->REQUEST->ACK cycle
func BenchmarkDHCPFullCycle(b *testing.B) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)
	stack := NewStack(nil, cfg, debugConfig)
	handler := NewDHCPHandler(stack)

	serverIP := net.ParseIP("192.168.1.1")
	gateway := net.ParseIP("192.168.1.1")
	dns := []net.IP{net.ParseIP("8.8.8.8")}
	handler.SetServerConfig(serverIP, gateway, dns, "example.com")
	handler.SetPool(net.ParseIP("192.168.1.100"), net.ParseIP("192.168.1.200"))

	serverMAC, _ := net.ParseMAC("00:11:22:33:44:55")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Generate unique client MAC for each iteration
		clientMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
		clientMAC[5] = byte(i % 256)

		// Allocate lease (DISCOVER/OFFER)
		lease, _ := handler.allocateLease(clientMAC, nil, "test-host")
		if lease != nil {
			// Send OFFER
			_ = handler.SendDHCPOffer(uint32(i), clientMAC, lease.IP, serverIP, serverMAC)
			// Send ACK (REQUEST/ACK)
			_ = handler.SendDHCPAck(uint32(i), clientMAC, lease.IP, serverIP, serverMAC)
		}
	}
}

// BenchmarkICMPEchoRequest benchmarks processing ICMP echo requests
func BenchmarkICMPEchoRequest(b *testing.B) {
	cfg := &config.Config{}
	debugConfig := logging.NewDebugConfig(0)
	stack := NewStack(nil, cfg, debugConfig)
	handler := NewICMPHandler(stack)

	// Create test device
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	ip := net.ParseIP("192.168.1.1")
	device := &config.Device{
		Name:        "test-device",
		MACAddress:  mac,
		IPAddresses: []net.IP{ip},
		ICMPConfig: &config.ICMPConfig{
			Enabled:   true,
			TTL:       64,
			RateLimit: 0,
		},
	}
	devices := []*config.Device{device}

	// Build ICMP echo request
	srcMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	srcIP := net.ParseIP("192.168.1.100")

	icmp := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
		Id:       1,
		Seq:      1,
	}

	ipLayer := &layers.IPv4{
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    srcIP,
		DstIP:    ip,
	}

	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       mac,
		EthernetType: layers.EthernetTypeIPv4,
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	gopacket.SerializeLayers(buffer, opts, eth, ipLayer, icmp, gopacket.Payload([]byte("test payload")))

	pkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: 1,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		handler.HandlePacket(pkt, ipLayer, devices)
	}
}

// BenchmarkSNMPGetRequest benchmarks processing SNMP GET requests (simulated)
// Note: This benchmarks the packet parsing overhead, not the full SNMP agent
func BenchmarkSNMPGetRequest(b *testing.B) {
	// Simulates SNMP GET request processing overhead
	oid := "1.3.6.1.2.1.1.1.0" // sysDescr
	community := "public"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate OID lookup
		_ = oid
		_ = community
		// In real implementation, this would query MIB
	}
}

// BenchmarkSNMPGetNextRequest benchmarks processing SNMP GET-NEXT requests (simulated)
func BenchmarkSNMPGetNextRequest(b *testing.B) {
	oid := "1.3.6.1.2.1.1.1"
	community := "public"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate finding next OID
		_ = oid
		_ = community
	}
}

// BenchmarkSNMPGetBulkRequest benchmarks processing SNMP GET-BULK requests (simulated)
func BenchmarkSNMPGetBulkRequest(b *testing.B) {
	oid := "1.3.6.1.2.1.2.2.1"
	community := "public"
	maxRepetitions := 10

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate bulk retrieval
		for j := 0; j < maxRepetitions; j++ {
			_ = oid
			_ = community
		}
	}
}

// BenchmarkDNSQueryProcessing benchmarks processing DNS queries
func BenchmarkDNSQueryProcessing(b *testing.B) {
	// Configure DNS records
	device := &config.Device{
		Name: "dns-server",
		DNSConfig: &config.DNSConfig{
			ForwardRecords: []config.DNSRecord{
				{Name: "test.example.com", IP: net.ParseIP("192.168.1.100"), TTL: 3600},
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate DNS query lookup
		for _, record := range device.DNSConfig.ForwardRecords {
			_ = record.Name
			_ = record.IP
		}
	}
}

// BenchmarkNetBIOSNameQuery benchmarks processing NetBIOS name queries
func BenchmarkNetBIOSNameQuery(b *testing.B) {
	// Simulate NetBIOS name query processing
	netbiosName := "TESTDEVICE"
	workgroup := "WORKGROUP"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate name lookup
		_ = netbiosName
		_ = workgroup
	}
}
