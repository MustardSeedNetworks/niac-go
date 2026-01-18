package protocols_test

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
)

// TestNewDHCPv6Handler tests handler creation.
func TestNewDHCPv6Handler(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	if handler == nil {
		t.Fatal("NewDHCPv6Handler returned nil")
	}

	if handler.DHCPv6HandlerStack() != stack {
		t.Error("Handler stack not set correctly")
	}

	if handler.DHCPv6HandlerLeases() == nil {
		t.Error("Leases map not initialized")
	}

	if handler.DHCPv6HandlerPreferredLifetime() != protocols.DefaultPreferredLifetime {
		t.Errorf(
			"Expected preferred lifetime %v, got %v",
			protocols.DefaultPreferredLifetime,
			handler.DHCPv6HandlerPreferredLifetime(),
		)
	}

	if handler.DHCPv6HandlerValidLifetime() != protocols.DefaultValidLifetime {
		t.Errorf(
			"Expected valid lifetime %v, got %v",
			protocols.DefaultValidLifetime,
			handler.DHCPv6HandlerValidLifetime(),
		)
	}

	if len(handler.DHCPv6HandlerServerDUID()) == 0 {
		t.Error("Server DUID not generated")
	}
}

// NOTE: TestGenerateDUID is not included in the external test package because
// generateDUID is an unexported package-level function. Testing of DUID generation
// is covered implicitly through TestNewDHCPv6Handler which verifies the server DUID.

// TestSetAddressPool tests address pool configuration.
func TestSetAddressPool(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	addresses := []net.IP{
		net.ParseIP("2001:db8::100"),
		net.ParseIP("2001:db8::101"),
		net.ParseIP("2001:db8::102"),
	}

	handler.SetAddressPool(addresses)

	pool := handler.DHCPv6HandlerAddressPool()
	if len(pool) != 3 {
		t.Errorf("Expected 3 addresses in pool, got %d", len(pool))
	}

	for i, addr := range addresses {
		if !pool[i].Equal(addr) {
			t.Errorf("Address %d mismatch: expected %v, got %v", i, addr, pool[i])
		}
	}
}

// TestSetPrefixPool tests prefix pool configuration.
func TestSetPrefixPool(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	_, prefix1, _ := net.ParseCIDR("2001:db8:1::/48")
	_, prefix2, _ := net.ParseCIDR("2001:db8:2::/48")

	prefixes := []net.IPNet{*prefix1, *prefix2}
	handler.SetPrefixPool(prefixes)

	pool := handler.DHCPv6HandlerPrefixPool()
	if len(pool) != 2 {
		t.Errorf("Expected 2 prefixes in pool, got %d", len(pool))
	}

	if pool[0].String() != prefix1.String() {
		t.Errorf("Prefix 0 mismatch: expected %v, got %v", prefix1, pool[0])
	}
}

// TestSetServerConfigDHCPv6 tests server configuration.
func TestSetServerConfigDHCPv6(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	dnsServers := []net.IP{
		net.ParseIP("2001:4860:4860::8888"),
		net.ParseIP("2001:4860:4860::8844"),
	}
	domainList := []string{"example.com", "test.local"}

	handler.SetServerConfig(dnsServers, domainList)

	dns := handler.DHCPv6HandlerDNSServers()
	if len(dns) != 2 {
		t.Errorf("Expected 2 DNS servers, got %d", len(dns))
	}

	if !dns[0].Equal(dnsServers[0]) {
		t.Errorf("DNS server 0 mismatch: expected %v, got %v", dnsServers[0], dns[0])
	}

	domains := handler.DHCPv6HandlerDomainList()
	if len(domains) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(domains))
	}

	if domains[0] != "example.com" {
		t.Errorf("Domain 0 mismatch: expected example.com, got %s", domains[0])
	}
}

// TestSetAdvancedOptionsDHCPv6 tests advanced DHCPv6 options.
func TestSetAdvancedOptionsDHCPv6(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	sntpServers := []net.IP{net.ParseIP("2001:db8::1")}
	ntpServers := []net.IP{net.ParseIP("2001:db8::2")}
	sipServers := []net.IP{net.ParseIP("2001:db8::3")}
	sipDomains := []string{"sip.example.com"}

	handler.SetAdvancedOptions(sntpServers, ntpServers, sipServers, sipDomains)

	if len(handler.DHCPv6HandlerSNTPServers()) != 1 {
		t.Errorf("Expected 1 SNTP server, got %d", len(handler.DHCPv6HandlerSNTPServers()))
	}

	if len(handler.DHCPv6HandlerNTPServers()) != 1 {
		t.Errorf("Expected 1 NTP server, got %d", len(handler.DHCPv6HandlerNTPServers()))
	}

	if len(handler.DHCPv6HandlerSIPServers()) != 1 {
		t.Errorf("Expected 1 SIP server, got %d", len(handler.DHCPv6HandlerSIPServers()))
	}

	sipDomainsResult := handler.DHCPv6HandlerSIPDomains()
	if len(sipDomainsResult) != 1 || sipDomainsResult[0] != "sip.example.com" {
		t.Errorf("Expected SIP domain 'sip.example.com', got %v", sipDomainsResult)
	}
}

// TestAllocateLeaseDHCPv6 tests lease allocation.
func TestAllocateLeaseDHCPv6(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	addresses := []net.IP{
		net.ParseIP("2001:db8::100"),
		net.ParseIP("2001:db8::101"),
	}
	handler.SetAddressPool(addresses)

	clientDUID := []byte{0x00, 0x03, 0x00, 0x01, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	iaid := uint32(12345)

	lease, err := handler.DHCPv6AllocateLease(clientDUID, iaid)
	if err != nil {
		t.Fatalf("Failed to allocate lease: %v", err)
	}

	if lease == nil {
		t.Fatal("Lease is nil")
	}

	if !lease.Address.Equal(addresses[0]) {
		t.Errorf("Expected address %v, got %v", addresses[0], lease.Address)
	}

	if string(lease.DUID) != string(clientDUID) {
		t.Errorf("Expected DUID %v, got %v", clientDUID, lease.DUID)
	}

	if lease.IAID != iaid {
		t.Errorf("Expected IAID %d, got %d", iaid, lease.IAID)
	}

	// Check lifetimes
	now := time.Now()
	if lease.PreferredLifetime.Before(now) {
		t.Error("Preferred lifetime should be in the future")
	}

	if lease.ValidLifetime.Before(now) {
		t.Error("Valid lifetime should be in the future")
	}

	if lease.PreferredLifetime.After(lease.ValidLifetime) {
		t.Error("Preferred lifetime should be before valid lifetime")
	}
}

// TestAllocateLeaseDHCPv6_Renewal tests existing lease renewal.
func TestAllocateLeaseDHCPv6_Renewal(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	addresses := []net.IP{net.ParseIP("2001:db8::100")}
	handler.SetAddressPool(addresses)

	clientDUID := []byte{0x00, 0x03, 0x00, 0x01, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	iaid := uint32(12345)

	// First allocation
	lease1, _ := handler.DHCPv6AllocateLease(clientDUID, iaid)
	firstRenewal := lease1.LastRenewal

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Second allocation (should renew)
	lease2, err := handler.DHCPv6AllocateLease(clientDUID, iaid)
	if err != nil {
		t.Fatalf("Failed to renew lease: %v", err)
	}

	if !lease2.Address.Equal(lease1.Address) {
		t.Error("Renewed lease should have same address")
	}

	if !lease2.LastRenewal.After(firstRenewal) {
		t.Error("LastRenewal should be updated on renewal")
	}
}

// TestAllocateLeaseDHCPv6_PoolExhaustion tests no available addresses.
func TestAllocateLeaseDHCPv6_PoolExhaustion(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	// Single address pool
	addresses := []net.IP{net.ParseIP("2001:db8::100")}
	handler.SetAddressPool(addresses)

	// Allocate the only address
	clientDUID1 := []byte{0x00, 0x03, 0x00, 0x01, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	_, err := handler.DHCPv6AllocateLease(clientDUID1, 1)
	if err != nil {
		t.Fatalf("First allocation failed: %v", err)
	}

	// Try to allocate to another client
	clientDUID2 := []byte{0x00, 0x03, 0x00, 0x01, 0x00, 0x11, 0x22, 0x33, 0x44, 0x56}

	_, err = handler.DHCPv6AllocateLease(clientDUID2, 2)
	if err == nil {
		t.Error("Expected error when pool exhausted, got nil")
	}
}

// TestConfirmLease tests lease confirmation.
func TestConfirmLease(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	addresses := []net.IP{net.ParseIP("2001:db8::100")}
	handler.SetAddressPool(addresses)

	clientDUID := []byte{0x00, 0x03, 0x00, 0x01, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	iaid := uint32(12345)

	// Allocate first
	lease1, _ := handler.DHCPv6AllocateLease(clientDUID, iaid)

	// Confirm existing lease
	lease2, err := handler.ConfirmLease(clientDUID, iaid)
	if err != nil {
		t.Fatalf("Failed to confirm lease: %v", err)
	}

	if !lease2.Address.Equal(lease1.Address) {
		t.Error("Confirmed lease should have same address")
	}
}

// TestFindLease tests lease lookup.
func TestFindLease(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	addresses := []net.IP{net.ParseIP("2001:db8::100")}
	handler.SetAddressPool(addresses)

	clientDUID := []byte{0x00, 0x03, 0x00, 0x01, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	iaid := uint32(12345)

	// Allocate lease
	allocatedLease, _ := handler.DHCPv6AllocateLease(clientDUID, iaid)

	// Find lease
	foundLease := handler.FindLease(clientDUID)
	if foundLease == nil {
		t.Fatal("Failed to find allocated lease")
	}

	if !foundLease.Address.Equal(allocatedLease.Address) {
		t.Error("Found lease has different address")
	}

	// Try to find non-existent lease
	nonExistentDUID := []byte{0x00, 0x03, 0x00, 0x01, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	notFound := handler.FindLease(nonExistentDUID)
	if notFound != nil {
		t.Error("Should not find lease for non-existent DUID")
	}
}

// TestRenewLease tests lease renewal.
func TestRenewLease(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	addresses := []net.IP{net.ParseIP("2001:db8::100")}
	handler.SetAddressPool(addresses)

	clientDUID := []byte{0x00, 0x03, 0x00, 0x01, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	iaid := uint32(12345)

	lease, _ := handler.DHCPv6AllocateLease(clientDUID, iaid)
	originalRenewal := lease.LastRenewal

	time.Sleep(10 * time.Millisecond)

	handler.RenewLease(lease)

	if !lease.LastRenewal.After(originalRenewal) {
		t.Error("LastRenewal should be updated after renewal")
	}
}

// TestParseDHCPv6Message tests message parsing.
func TestParseDHCPv6Message(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	tests := []struct {
		name        string
		data        []byte
		expectError bool
		msgType     uint8
	}{
		{
			name:        "Too short",
			data:        []byte{0x01},
			expectError: true,
		},
		{
			name: "Valid Solicit",
			data: []byte{
				0x01,             // Message type: Solicit
				0x12, 0x34, 0x56, // Transaction ID
			},
			expectError: false,
			msgType:     protocols.DHCPv6Solicit,
		},
		{
			name: "With options",
			data: []byte{
				0x03,             // Message type: Request
				0xaa, 0xbb, 0xcc, // Transaction ID
				0x00, 0x01, // Option code: Client ID
				0x00, 0x04, // Option length: 4
				0x11, 0x22, 0x33, 0x44, // Option data
			},
			expectError: false,
			msgType:     protocols.DHCPv6Request,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := handler.ParseDHCPv6Message(tt.data)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if msg.MessageType != tt.msgType {
				t.Errorf("Expected message type %d, got %d", tt.msgType, msg.MessageType)
			}
		})
	}
}

// TestFindOption tests option finding.
func TestFindOption(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	msg := &protocols.DHCPv6Message{
		MessageType: protocols.DHCPv6Solicit,
		Options: []protocols.DHCPv6Option{
			{Code: protocols.DHCPv6OptClientID, Length: 4, Data: []byte{0x11, 0x22, 0x33, 0x44}},
			{Code: protocols.DHCPv6OptIANA, Length: 12, Data: make([]byte, 12)},
		},
	}

	// Find existing option
	clientID := handler.FindOption(msg, protocols.DHCPv6OptClientID)
	if clientID == nil {
		t.Fatal("Failed to find Client ID option")
	}

	if clientID.Code != protocols.DHCPv6OptClientID {
		t.Errorf("Expected code %d, got %d", protocols.DHCPv6OptClientID, clientID.Code)
	}

	// Find non-existent option
	serverID := handler.FindOption(msg, protocols.DHCPv6OptServerID)
	if serverID != nil {
		t.Error("Should not find non-existent option")
	}
}

// TestExtractClientDUID tests client DUID extraction.
func TestExtractClientDUID(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	expectedDUID := []byte{0x00, 0x03, 0x00, 0x01, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	msg := &protocols.DHCPv6Message{
		MessageType: protocols.DHCPv6Solicit,
		Options: []protocols.DHCPv6Option{
			{Code: protocols.DHCPv6OptClientID, Length: uint16(len(expectedDUID)), Data: expectedDUID},
		},
	}

	duid := handler.ExtractClientDUID(msg)
	if duid == nil {
		t.Fatal("Failed to extract client DUID")
	}

	if string(duid) != string(expectedDUID) {
		t.Errorf("Expected DUID %v, got %v", expectedDUID, duid)
	}

	// Test message without Client ID
	msgNoClientID := &protocols.DHCPv6Message{
		MessageType: protocols.DHCPv6Solicit,
		Options:     []protocols.DHCPv6Option{},
	}

	noDUID := handler.ExtractClientDUID(msgNoClientID)
	if noDUID != nil {
		t.Error("Should return nil for message without Client ID")
	}
}

// TestExtractIANA tests IANA extraction.
func TestExtractIANA(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	expectedIAID := uint32(0x12345678)
	ianaData := make([]byte, 12)
	binary.BigEndian.PutUint32(ianaData[0:4], expectedIAID)

	msg := &protocols.DHCPv6Message{
		MessageType: protocols.DHCPv6Solicit,
		Options: []protocols.DHCPv6Option{
			{Code: protocols.DHCPv6OptIANA, Length: 12, Data: ianaData},
		},
	}

	iaid, found := handler.ExtractIANA(msg)
	if !found {
		t.Fatal("Failed to extract IANA")
	}

	if iaid != expectedIAID {
		t.Errorf("Expected IAID 0x%08x, got 0x%08x", expectedIAID, iaid)
	}

	// Test message without IANA
	msgNoIANA := &protocols.DHCPv6Message{
		MessageType: protocols.DHCPv6Solicit,
		Options:     []protocols.DHCPv6Option{},
	}

	_, found = handler.ExtractIANA(msgNoIANA)
	if found {
		t.Error("Should not find IANA in message without it")
	}
}

// NOTE: TestSplitDomainLabels is not included because splitDomainLabels is
// an unexported package-level function. Domain label encoding is implicitly
// tested through TestEncodeDomainList.

// TestEncodeDomainList tests domain list encoding.
func TestEncodeDomainList(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	domains := []string{"example.com", "test.local"}
	encoded := handler.EncodeDomainList(domains)

	// Verify encoding (example.com = 7 e x a m p l e 3 c o m 0)
	// example: 7 bytes + com: 3 bytes + null: 1 = 18 bytes
	// test: 4 bytes + local: 5 bytes + null: 1 = 16 bytes
	// Total: 34 bytes (with length prefixes)

	if len(encoded) == 0 {
		t.Fatal("Encoded domain list is empty")
	}

	// Check first label length
	if encoded[0] != 7 {
		t.Errorf("Expected first label length 7, got %d", encoded[0])
	}

	// Check first label content
	expectedFirst := "example"
	if string(encoded[1:8]) != expectedFirst {
		t.Errorf("Expected first label %s, got %s", expectedFirst, string(encoded[1:8]))
	}
}

// TestMessageTypeString tests message type string conversion.
func TestMessageTypeString(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	tests := []struct {
		msgType  uint8
		expected string
	}{
		{protocols.DHCPv6Solicit, "SOLICIT"},
		{protocols.DHCPv6Advertise, "ADVERTISE"},
		{protocols.DHCPv6Request, "REQUEST"},
		{protocols.DHCPv6Renew, "RENEW"},
		{protocols.DHCPv6Rebind, "REBIND"},
		{protocols.DHCPv6Reply, "REPLY"},
		{protocols.DHCPv6Release, "RELEASE"},
		{protocols.DHCPv6Decline, "DECLINE"},
		{protocols.DHCPv6InfoRequest, "INFORMATION-REQUEST"},
		{255, "UNKNOWN(255)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := handler.MessageTypeString(tt.msgType)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestDHCPv6Constants tests DHCPv6 constants.
func TestDHCPv6Constants(t *testing.T) {
	// Test message types
	if protocols.DHCPv6Solicit != 1 {
		t.Errorf("DHCPv6Solicit should be 1, got %d", protocols.DHCPv6Solicit)
	}

	if protocols.DHCPv6Reply != 7 {
		t.Errorf("DHCPv6Reply should be 7, got %d", protocols.DHCPv6Reply)
	}

	// Test option codes
	if protocols.DHCPv6OptClientID != 1 {
		t.Errorf("DHCPv6OptClientID should be 1, got %d", protocols.DHCPv6OptClientID)
	}

	if protocols.DHCPv6OptServerID != 2 {
		t.Errorf("DHCPv6OptServerID should be 2, got %d", protocols.DHCPv6OptServerID)
	}

	if protocols.DHCPv6OptDNSServers != 23 {
		t.Errorf("DHCPv6OptDNSServers should be 23, got %d", protocols.DHCPv6OptDNSServers)
	}

	// Test DUID types
	if protocols.DUIDTypeLL != 3 {
		t.Errorf("DUIDTypeLL should be 3, got %d", protocols.DUIDTypeLL)
	}

	// Test ports
	if protocols.DHCPv6ServerPort != 547 {
		t.Errorf("DHCPv6ServerPort should be 547, got %d", protocols.DHCPv6ServerPort)
	}

	if protocols.DHCPv6ClientPort != 546 {
		t.Errorf("DHCPv6ClientPort should be 546, got %d", protocols.DHCPv6ClientPort)
	}
}

// TestDefaultLifetimes tests default lifetime constants.
func TestDefaultLifetimes(t *testing.T) {
	expectedPreferred := 7 * 24 * time.Hour
	expectedValid := 30 * 24 * time.Hour

	if protocols.DefaultPreferredLifetime != expectedPreferred {
		t.Errorf("DefaultPreferredLifetime should be %v, got %v", expectedPreferred, protocols.DefaultPreferredLifetime)
	}

	if protocols.DefaultValidLifetime != expectedValid {
		t.Errorf("DefaultValidLifetime should be %v, got %v", expectedValid, protocols.DefaultValidLifetime)
	}

	// Valid lifetime should be longer than preferred
	if protocols.DefaultValidLifetime <= protocols.DefaultPreferredLifetime {
		t.Error("Valid lifetime should be longer than preferred lifetime")
	}
}

// TestBuildIANAOption tests IANA option building.
func TestBuildIANAOption(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	lease := &protocols.DHCPv6Lease{
		Address:           net.ParseIP("2001:db8::100"),
		IAID:              0x12345678,
		PreferredLifetime: time.Now().Add(protocols.DefaultPreferredLifetime),
		ValidLifetime:     time.Now().Add(protocols.DefaultValidLifetime),
	}

	opt := handler.BuildIANAOption(lease)

	if opt.Code != protocols.DHCPv6OptIANA {
		t.Errorf("Expected option code %d, got %d", protocols.DHCPv6OptIANA, opt.Code)
	}

	if len(opt.Data) < 12 {
		t.Errorf("IANA option data should be at least 12 bytes, got %d", len(opt.Data))
	}

	// Check IAID
	iaid := binary.BigEndian.Uint32(opt.Data[0:4])
	if iaid != lease.IAID {
		t.Errorf("Expected IAID 0x%08x, got 0x%08x", lease.IAID, iaid)
	}
}

// TestBuildIAAddrOption tests IA Address option building.
func TestBuildIAAddrOption(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	testAddr := net.ParseIP("2001:db8::100")
	lease := &protocols.DHCPv6Lease{
		Address:           testAddr,
		PreferredLifetime: time.Now().Add(1 * time.Hour),
		ValidLifetime:     time.Now().Add(2 * time.Hour),
	}

	opt := handler.BuildIAAddrOption(lease)

	if opt.Code != protocols.DHCPv6OptIAAddr {
		t.Errorf("Expected option code %d, got %d", protocols.DHCPv6OptIAAddr, opt.Code)
	}

	if opt.Length != 24 {
		t.Errorf("Expected option length 24, got %d", opt.Length)
	}

	// Check address
	addr := net.IP(opt.Data[0:16])
	if !addr.Equal(testAddr) {
		t.Errorf("Expected address %v, got %v", testAddr, addr)
	}

	// Check preferred lifetime is non-zero
	preferred := binary.BigEndian.Uint32(opt.Data[16:20])
	if preferred == 0 {
		t.Error("Preferred lifetime should be non-zero for future lease")
	}
}

// Benchmarks

// BenchmarkAllocateLeaseDHCPv6 benchmarks lease allocation.
func BenchmarkAllocateLeaseDHCPv6(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	// Create a pool of 1000 addresses
	addresses := make([]net.IP, 1000)

	for i := range 1000 {
		addr := net.ParseIP("2001:db8::1")
		addr[15] = byte(i)
		addresses[i] = addr
	}

	handler.SetAddressPool(addresses)

	b.ResetTimer()

	for i := range b.N {
		clientDUID := make([]byte, 10)
		binary.BigEndian.PutUint32(clientDUID[6:], uint32(i))
		_, _ = handler.DHCPv6AllocateLease(clientDUID, uint32(i))
	}
}

// NOTE: BenchmarkGenerateDUID is not included because generateDUID is
// an unexported package-level function.

// BenchmarkParseDHCPv6Message benchmarks message parsing.
func BenchmarkParseDHCPv6Message(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPv6Handler(stack)

	// Create a realistic DHCPv6 message with options
	msg := []byte{
		0x01,             // Solicit
		0x12, 0x34, 0x56, // Transaction ID
		0x00, 0x01, 0x00, 0x0a, // Client ID option
		0x00, 0x03, 0x00, 0x01, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55,
		0x00, 0x03, 0x00, 0x0c, // IANA option
		0x12, 0x34, 0x56, 0x78, // IAID
		0x00, 0x00, 0x00, 0x00, // T1
		0x00, 0x00, 0x00, 0x00, // T2
	}

	for b.Loop() {
		_, _ = handler.ParseDHCPv6Message(msg)
	}
}
