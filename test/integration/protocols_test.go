package integration

import (
	"net"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/snmp"
)

// createTestDevices creates test devices for protocol integration testing
func createTestDevices() []*config.Device {
	mac1, _ := net.ParseMAC("00:11:22:33:44:55")
	mac2, _ := net.ParseMAC("00:11:22:33:44:66")

	dev1 := &config.Device{
		Name:        "router-1",
		Type:        "router",
		MACAddress:  mac1,
		IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
		SNMPConfig: config.SNMPConfig{
			Community: "public",
		},
		Properties: map[string]string{
			"sysDescr": "Test Router 1",
		},
	}

	dev2 := &config.Device{
		Name:        "switch-1",
		Type:        "switch",
		MACAddress:  mac2,
		IPAddresses: []net.IP{net.ParseIP("192.168.1.2")},
		SNMPConfig: config.SNMPConfig{
			Community: "public",
		},
	}

	return []*config.Device{dev1, dev2}
}

// TestProtocolIntegration_ARPPacketStructure tests ARP packet structure creation
func TestProtocolIntegration_ARPPacketStructure(t *testing.T) {
	devices := createTestDevices()

	// Build ARP request packet
	srcMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	srcIP := net.ParseIP("192.168.1.100")
	targetIP := devices[0].IPAddresses[0]

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
		DstProtAddress:    targetIP.To4(),
	}

	// Serialize packet
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, arp)
	if err != nil {
		t.Fatalf("Failed to serialize ARP request: %v", err)
	}

	if len(buffer.Bytes()) == 0 {
		t.Error("ARP packet buffer is empty")
	}

	t.Logf("ARP request packet created: %d bytes", len(buffer.Bytes()))
}

// TestProtocolIntegration_ARPResponseStructure tests ARP response packet structure
func TestProtocolIntegration_ARPResponseStructure(t *testing.T) {
	devices := createTestDevices()

	// Build ARP reply packet
	senderMAC := devices[0].MACAddress
	senderIP := devices[0].IPAddresses[0]
	targetMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	targetIP := net.ParseIP("192.168.1.100")

	eth := &layers.Ethernet{
		SrcMAC:       senderMAC,
		DstMAC:       targetMAC,
		EthernetType: layers.EthernetTypeARP,
	}

	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPReply,
		SourceHwAddress:   senderMAC,
		SourceProtAddress: senderIP.To4(),
		DstHwAddress:      targetMAC,
		DstProtAddress:    targetIP.To4(),
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, arp)
	if err != nil {
		t.Fatalf("Failed to serialize ARP reply: %v", err)
	}

	if len(buffer.Bytes()) == 0 {
		t.Error("ARP reply buffer is empty")
	}

	t.Logf("ARP reply packet created: %d bytes", len(buffer.Bytes()))
}

// TestProtocolIntegration_LLDPFrameStructure tests LLDP frame structure
func TestProtocolIntegration_LLDPFrameStructure(t *testing.T) {
	devices := createTestDevices()
	device := devices[0]

	// Build basic LLDP frame components
	lldpMAC, _ := net.ParseMAC("01:80:c2:00:00:0e")

	eth := &layers.Ethernet{
		SrcMAC:       device.MACAddress,
		DstMAC:       lldpMAC,
		EthernetType: layers.EthernetType(0x88cc), // LLDP EtherType
	}

	// Create minimal LLDP payload (Chassis ID TLV + Port ID TLV + TTL TLV + End TLV)
	var lldpPayload []byte

	// Chassis ID TLV (Type=1, MAC address subtype)
	chassisID := append([]byte{0x02, 0x07, 0x04}, device.MACAddress...)
	lldpPayload = append(lldpPayload, chassisID...)

	// Port ID TLV (Type=2, interface name subtype)
	portID := []byte{0x04, 0x0c, 0x05}
	portID = append(portID, []byte("router-1")...)
	lldpPayload = append(lldpPayload, portID...)

	// TTL TLV (Type=3, 120 seconds)
	ttl := []byte{0x06, 0x02, 0x00, 0x78}
	lldpPayload = append(lldpPayload, ttl...)

	// End TLV
	lldpPayload = append(lldpPayload, 0x00, 0x00)

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: false,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, gopacket.Payload(lldpPayload))
	if err != nil {
		t.Fatalf("Failed to serialize LLDP frame: %v", err)
	}

	if len(buffer.Bytes()) == 0 {
		t.Error("LLDP frame buffer is empty")
	}

	t.Logf("LLDP frame created: %d bytes", len(buffer.Bytes()))
}

// TestProtocolIntegration_SNMPGetRequest tests SNMP GET request handling
func TestProtocolIntegration_SNMPGetRequest(t *testing.T) {
	devices := createTestDevices()

	// Create SNMP agent for device
	agent := snmp.NewAgent(devices[0], 0)

	// Test GET request for sysDescr
	result, err := agent.HandleGet("1.3.6.1.2.1.1.1.0")
	if err != nil {
		t.Fatalf("SNMP GET failed: %v", err)
	}

	if result == nil {
		t.Fatal("SNMP GET returned nil result")
	}

	if result.Value == nil {
		t.Error("SNMP GET result value is nil")
	}

	t.Logf("SNMP GET successful: sysDescr=%v", result.Value)
}

// TestProtocolIntegration_SNMPGetNextRequest tests SNMP GET-NEXT request handling
func TestProtocolIntegration_SNMPGetNextRequest(t *testing.T) {
	devices := createTestDevices()

	agent := snmp.NewAgent(devices[0], 0)

	// Test GET-NEXT starting from system MIB
	nextOID, value, err := agent.HandleGetNext("1.3.6.1.2.1.1")
	if err != nil {
		t.Fatalf("SNMP GET-NEXT failed: %v", err)
	}

	if nextOID == "" {
		t.Error("SNMP GET-NEXT returned empty OID")
	}

	if value == nil {
		t.Error("SNMP GET-NEXT returned nil value")
	}

	t.Logf("SNMP GET-NEXT successful: %s = %v", nextOID, value.Value)
}

// TestProtocolIntegration_SNMPWalk tests SNMP walk operation
func TestProtocolIntegration_SNMPWalk(t *testing.T) {
	devices := createTestDevices()

	agent := snmp.NewAgent(devices[0], 0)

	// Walk through system MIB
	currentOID := "1.3.6.1.2.1.1"
	oidCount := 0
	oids := []string{}

	for i := 0; i < 10; i++ {
		nextOID, value, err := agent.HandleGetNext(currentOID)
		if err != nil || nextOID == "" || value == nil {
			break
		}
		oidCount++
		oids = append(oids, nextOID)
		currentOID = nextOID
	}

	if oidCount == 0 {
		t.Error("SNMP walk returned no OIDs")
	}

	t.Logf("SNMP walk completed: %d OIDs traversed", oidCount)
	for _, oid := range oids {
		t.Logf("  - %s", oid)
	}
}

// TestProtocolIntegration_SNMPGetBulkRequest tests SNMP GET-BULK request
func TestProtocolIntegration_SNMPGetBulkRequest(t *testing.T) {
	devices := createTestDevices()

	agent := snmp.NewAgent(devices[0], 0)

	// Test GET-BULK with maxRepetitions = 5
	results, err := agent.HandleGetBulk("1.3.6.1.2.1.1", 5)
	if err != nil {
		t.Fatalf("SNMP GET-BULK failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("SNMP GET-BULK returned no results")
	}

	if len(results) > 5 {
		t.Errorf("SNMP GET-BULK returned too many results: %d > 5", len(results))
	}

	t.Logf("SNMP GET-BULK successful: %d results returned", len(results))
	for _, result := range results {
		t.Logf("  - %s = %v", result.OID, result.Value.Value)
	}
}

// TestProtocolIntegration_DHCPDiscoverPacket tests DHCP DISCOVER packet structure
func TestProtocolIntegration_DHCPDiscoverPacket(t *testing.T) {
	clientMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")

	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  6,
		Xid:          0x12345678,
		ClientHWAddr: clientMAC,
	}

	dhcp.Options = []layers.DHCPOption{
		{
			Type:   layers.DHCPOptMessageType,
			Length: 1,
			Data:   []byte{1}, // DISCOVER
		},
		{Type: layers.DHCPOptEnd},
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
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	udp.SetNetworkLayerForChecksum(ip)

	err := gopacket.SerializeLayers(buffer, opts, eth, ip, udp, dhcp)
	if err != nil {
		t.Fatalf("Failed to serialize DHCP DISCOVER: %v", err)
	}

	if len(buffer.Bytes()) == 0 {
		t.Error("DHCP DISCOVER packet buffer is empty")
	}

	t.Logf("DHCP DISCOVER packet created: %d bytes", len(buffer.Bytes()))
}

// TestProtocolIntegration_DHCPOfferPacket tests DHCP OFFER packet structure
func TestProtocolIntegration_DHCPOfferPacket(t *testing.T) {
	devices := createTestDevices()
	serverIP := devices[0].IPAddresses[0]
	clientMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	offeredIP := net.ParseIP("192.168.1.100")

	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpReply,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  6,
		Xid:          0x12345678,
		YourClientIP: offeredIP,
		ClientHWAddr: clientMAC,
	}

	dhcp.Options = []layers.DHCPOption{
		{
			Type:   layers.DHCPOptMessageType,
			Length: 1,
			Data:   []byte{2}, // OFFER
		},
		{
			Type:   layers.DHCPOptServerID,
			Length: 4,
			Data:   []byte(serverIP.To4()),
		},
		{Type: layers.DHCPOptEnd},
	}

	udp := &layers.UDP{
		SrcPort: 67,
		DstPort: 68,
	}

	ip := &layers.IPv4{
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    serverIP,
		DstIP:    net.IPv4bcast,
	}

	eth := &layers.Ethernet{
		SrcMAC:       devices[0].MACAddress,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	udp.SetNetworkLayerForChecksum(ip)

	err := gopacket.SerializeLayers(buffer, opts, eth, ip, udp, dhcp)
	if err != nil {
		t.Fatalf("Failed to serialize DHCP OFFER: %v", err)
	}

	if len(buffer.Bytes()) == 0 {
		t.Error("DHCP OFFER packet buffer is empty")
	}

	t.Logf("DHCP OFFER packet created: %d bytes", len(buffer.Bytes()))
}

// TestProtocolIntegration_DHCPRequestPacket tests DHCP REQUEST packet structure
func TestProtocolIntegration_DHCPRequestPacket(t *testing.T) {
	clientMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	requestedIP := net.ParseIP("192.168.1.100")

	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  6,
		Xid:          0x12345678,
		ClientHWAddr: clientMAC,
	}

	dhcp.Options = []layers.DHCPOption{
		{
			Type:   layers.DHCPOptMessageType,
			Length: 1,
			Data:   []byte{3}, // REQUEST
		},
		{
			Type:   layers.DHCPOptRequestIP,
			Length: 4,
			Data:   requestedIP.To4(),
		},
		{Type: layers.DHCPOptEnd},
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
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	udp.SetNetworkLayerForChecksum(ip)

	err := gopacket.SerializeLayers(buffer, opts, eth, ip, udp, dhcp)
	if err != nil {
		t.Fatalf("Failed to serialize DHCP REQUEST: %v", err)
	}

	if len(buffer.Bytes()) == 0 {
		t.Error("DHCP REQUEST packet buffer is empty")
	}

	t.Logf("DHCP REQUEST packet created: %d bytes", len(buffer.Bytes()))
}

// TestProtocolIntegration_DHCPAckPacket tests DHCP ACK packet structure
func TestProtocolIntegration_DHCPAckPacket(t *testing.T) {
	devices := createTestDevices()
	serverIP := devices[0].IPAddresses[0]
	clientMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	assignedIP := net.ParseIP("192.168.1.100")

	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpReply,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  6,
		Xid:          0x12345678,
		YourClientIP: assignedIP,
		ClientHWAddr: clientMAC,
	}

	dhcp.Options = []layers.DHCPOption{
		{
			Type:   layers.DHCPOptMessageType,
			Length: 1,
			Data:   []byte{5}, // ACK
		},
		{
			Type:   layers.DHCPOptServerID,
			Length: 4,
			Data:   []byte(serverIP.To4()),
		},
		{Type: layers.DHCPOptEnd},
	}

	udp := &layers.UDP{
		SrcPort: 67,
		DstPort: 68,
	}

	ip := &layers.IPv4{
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    serverIP,
		DstIP:    net.IPv4bcast,
	}

	eth := &layers.Ethernet{
		SrcMAC:       devices[0].MACAddress,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	udp.SetNetworkLayerForChecksum(ip)

	err := gopacket.SerializeLayers(buffer, opts, eth, ip, udp, dhcp)
	if err != nil {
		t.Fatalf("Failed to serialize DHCP ACK: %v", err)
	}

	if len(buffer.Bytes()) == 0 {
		t.Error("DHCP ACK packet buffer is empty")
	}

	t.Logf("DHCP ACK packet created: %d bytes", len(buffer.Bytes()))
}

// TestProtocolIntegration_MultipleDevices tests protocol operations across multiple devices
func TestProtocolIntegration_MultipleDevices(t *testing.T) {
	devices := createTestDevices()

	// Create SNMP agents for all devices
	agents := make([]*snmp.Agent, len(devices))
	for i, device := range devices {
		agents[i] = snmp.NewAgent(device, 0)
	}

	// Test each agent
	for i, agent := range agents {
		result, err := agent.HandleGet("1.3.6.1.2.1.1.5.0") // sysName
		if err != nil {
			t.Errorf("Device %d: SNMP GET failed: %v", i, err)
			continue
		}

		sysName := result.Value.(string)
		t.Logf("Device %d (%s): sysName=%s", i, devices[i].Name, sysName)
	}
}

// TestProtocolIntegration_CommunityValidation tests SNMP community string validation
func TestProtocolIntegration_CommunityValidation(t *testing.T) {
	devices := createTestDevices()

	// Test specific community
	devices[0].SNMPConfig.Community = "private"
	agent1 := snmp.NewAgent(devices[0], 0)

	if agent1.GetCommunity() != "private" {
		t.Errorf("Expected community 'private', got '%s'", agent1.GetCommunity())
	}

	// Test default community
	devices[1].SNMPConfig.Community = ""
	agent2 := snmp.NewAgent(devices[1], 0)

	if agent2.GetCommunity() != "public" {
		t.Errorf("Expected default community 'public', got '%s'", agent2.GetCommunity())
	}

	t.Log("SNMP community validation successful")
}

// TestProtocolIntegration_ConcurrentOperations tests concurrent protocol operations
func TestProtocolIntegration_ConcurrentOperations(t *testing.T) {
	devices := createTestDevices()
	agent := snmp.NewAgent(devices[0], 0)

	done := make(chan bool, 50)

	// Launch concurrent SNMP GET operations
	for i := 0; i < 50; i++ {
		go func() {
			_, _ = agent.HandleGet("1.3.6.1.2.1.1.3.0")
			done <- true
		}()
	}

	// Wait with timeout
	timeout := time.After(5 * time.Second)
	for i := 0; i < 50; i++ {
		select {
		case <-done:
			// Success
		case <-timeout:
			t.Fatal("Concurrent operations timed out")
		}
	}

	t.Log("Concurrent protocol operations completed successfully")
}

// TestProtocolIntegration_ErrorHandling tests protocol error responses
func TestProtocolIntegration_ErrorHandling(t *testing.T) {
	devices := createTestDevices()
	agent := snmp.NewAgent(devices[0], 0)

	// Test non-existent OID
	_, err := agent.HandleGet("1.2.3.4.5.6.7.8.9.10")
	if err == nil {
		t.Error("Expected error for non-existent OID")
	}

	// Test end of MIB
	_, _, err = agent.HandleGetNext("9.9.9.9.9.9.9")
	if err == nil {
		t.Error("Expected error at end of MIB")
	}

	// Test empty walk file
	err = agent.LoadWalkFile("")
	if err == nil {
		t.Error("Expected error for empty walk file path")
	}

	t.Log("Protocol error handling verified")
}

// TestProtocolIntegration_PacketSizeValidation tests protocol packet size validation
func TestProtocolIntegration_PacketSizeValidation(t *testing.T) {
	// Test ARP packet size
	srcMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	srcIP := net.ParseIP("192.168.1.100")
	targetIP := net.ParseIP("192.168.1.1")

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
		DstProtAddress:    targetIP.To4(),
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(buffer, opts, eth, arp); err != nil {
		t.Fatalf("Failed to serialize ARP packet: %v", err)
	}

	// ARP packet should be at least 42 bytes (14 ethernet + 28 ARP)
	// May be larger due to padding to meet minimum Ethernet frame size (60 bytes)
	minSize := 42
	actualSize := len(buffer.Bytes())

	if actualSize < minSize {
		t.Errorf("Expected ARP packet size >= %d, got %d", minSize, actualSize)
	}

	t.Logf("ARP packet size validation: %d bytes (minimum: %d bytes)", actualSize, minSize)
}
