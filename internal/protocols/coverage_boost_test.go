package protocols_test

import (
	"encoding/binary"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gopacket/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/logging"
	"github.com/krisarmstrong/niac-go/internal/protocols"
)

// ---------- helpers ----------

func newTestStack(t *testing.T) *protocols.Stack {
	t.Helper()
	cfg := &config.Config{}
	return protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
}

// ---------- DHCP pool & lease tests ----------

func TestDHCPGenerateIPPool(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPHandler(stack)

	tests := []struct {
		name    string
		start   net.IP
		end     net.IP
		wantLen int
		wantErr bool
	}{
		{
			name:    "small pool",
			start:   net.ParseIP("10.0.0.1"),
			end:     net.ParseIP("10.0.0.5"),
			wantLen: 5,
		},
		{
			name:    "single IP pool",
			start:   net.ParseIP("10.0.0.1"),
			end:     net.ParseIP("10.0.0.1"),
			wantLen: 1,
		},
		{
			name:    "reversed range",
			start:   net.ParseIP("10.0.0.5"),
			end:     net.ParseIP("10.0.0.1"),
			wantErr: true,
		},
		{
			name:    "class C pool",
			start:   net.ParseIP("192.168.1.1"),
			end:     net.ParseIP("192.168.1.254"),
			wantLen: 254,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkIPPool(t, handler, tt.start, tt.end, tt.wantLen, tt.wantErr)
		})
	}
}

func checkIPPool(t *testing.T, handler *protocols.DHCPHandler, start, end net.IP, wantLen int, wantErr bool) {
	t.Helper()
	pool, err := handler.GenerateIPPool(start, end)
	if wantErr {
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pool) != wantLen {
		t.Errorf("pool length = %d, want %d", len(pool), wantLen)
	}
	if len(pool) == 0 {
		return
	}
	if !pool[0].Equal(start.To4()) {
		t.Errorf("first IP = %s, want %s", pool[0], start)
	}
	if !pool[len(pool)-1].Equal(end.To4()) {
		t.Errorf("last IP = %s, want %s", pool[len(pool)-1], end)
	}
}

func TestDHCPSetPoolAndServerConfig(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPHandler(stack)

	handler.SetPool(net.ParseIP("10.0.0.10"), net.ParseIP("10.0.0.20"))

	pool := handler.DHCPHandlerIPPool()
	if len(pool) != 11 {
		t.Errorf("pool size = %d, want 11", len(pool))
	}

	serverIP := net.ParseIP("10.0.0.1")
	gateway := net.ParseIP("10.0.0.1")
	dnsServers := []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("8.8.4.4")}
	handler.SetServerConfig(serverIP, gateway, dnsServers, "example.com")

	if !handler.DHCPHandlerServerIP().Equal(serverIP) {
		t.Error("server IP not set correctly")
	}
	if !handler.DHCPHandlerGateway().Equal(gateway) {
		t.Error("gateway not set correctly")
	}
	if len(handler.DHCPHandlerDNSServers()) != 2 {
		t.Errorf("DNS servers count = %d, want 2", len(handler.DHCPHandlerDNSServers()))
	}
	if handler.DHCPHandlerDomainName() != "example.com" {
		t.Errorf("domain name = %q, want %q", handler.DHCPHandlerDomainName(), "example.com")
	}
}

func TestDHCPSetAdvancedOptions(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPHandler(stack)

	ntpServers := []net.IP{net.ParseIP("10.0.0.100")}
	domainSearch := []string{"example.com", "local"}
	handler.SetAdvancedOptions(ntpServers, domainSearch, "tftp.example.com", "pxelinux.0", nil)

	if len(handler.DHCPHandlerNTPServers()) != 1 {
		t.Error("NTP servers not set")
	}
	if len(handler.DHCPHandlerDomainSearch()) != 2 {
		t.Error("domain search not set")
	}
	if handler.DHCPHandlerTFTPServerName() != "tftp.example.com" {
		t.Error("TFTP server not set")
	}
	if handler.DHCPHandlerBootfileName() != "pxelinux.0" {
		t.Error("bootfile not set")
	}
}

func TestDHCPSetStaticLeases(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPHandler(stack)

	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	staticIP := net.ParseIP("10.0.0.100")

	handler.SetStaticLeases([]config.DHCPLease{
		{MACAddress: mac, ClientIP: staticIP},
	})

	// Should match static lease
	got := handler.DHCPMatchStaticLease(mac)
	if !got.Equal(staticIP) {
		t.Errorf("static lease IP = %s, want %s", got, staticIP)
	}

	// Non-matching MAC
	otherMAC, _ := net.ParseMAC("AA:BB:CC:DD:EE:FF")
	got = handler.DHCPMatchStaticLease(otherMAC)
	if got != nil {
		t.Errorf("expected nil for non-matching MAC, got %s", got)
	}
}

func TestDHCPFindAvailableIPAndAllocate(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPHandler(stack)

	handler.SetPool(net.ParseIP("10.0.0.10"), net.ParseIP("10.0.0.12"))

	// First allocation
	mac1, _ := net.ParseMAC("00:11:22:33:44:01")
	lease1, err := handler.AllocateLease(mac1, nil, "host1")
	if err != nil {
		t.Fatalf("allocate lease 1: %v", err)
	}
	if lease1.Hostname != "host1" {
		t.Errorf("hostname = %q, want %q", lease1.Hostname, "host1")
	}
	if !lease1.IP.Equal(net.ParseIP("10.0.0.10").To4()) {
		t.Errorf("first lease IP = %s, want 10.0.0.10", lease1.IP)
	}

	// Second allocation
	mac2, _ := net.ParseMAC("00:11:22:33:44:02")
	lease2, err := handler.AllocateLease(mac2, nil, "host2")
	if err != nil {
		t.Fatalf("allocate lease 2: %v", err)
	}
	if lease2.IP.Equal(lease1.IP) {
		t.Error("second lease got same IP as first")
	}

	// Renew existing lease
	lease1Again, err := handler.AllocateLease(mac1, nil, "host1-updated")
	if err != nil {
		t.Fatalf("renew lease: %v", err)
	}
	if lease1Again.Hostname != "host1-updated" {
		t.Errorf("renewed hostname = %q, want %q", lease1Again.Hostname, "host1-updated")
	}
	if !lease1Again.IP.Equal(lease1.IP) {
		t.Error("renewed lease should keep same IP")
	}

	// Third allocation
	mac3, _ := net.ParseMAC("00:11:22:33:44:03")
	_, err = handler.AllocateLease(mac3, nil, "host3")
	if err != nil {
		t.Fatalf("allocate lease 3: %v", err)
	}

	// Pool exhausted
	mac4, _ := net.ParseMAC("00:11:22:33:44:04")
	_, err = handler.AllocateLease(mac4, nil, "host4")
	if err == nil {
		t.Error("expected pool exhaustion error")
	}
}

func TestDHCPAllocateLeaseWithRequestedIP(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPHandler(stack)

	handler.SetPool(net.ParseIP("10.0.0.10"), net.ParseIP("10.0.0.20"))

	mac, _ := net.ParseMAC("00:11:22:33:44:01")
	requestedIP := net.ParseIP("10.0.0.15")

	lease, err := handler.AllocateLease(mac, requestedIP, "host-req")
	if err != nil {
		t.Fatalf("allocate with requested IP: %v", err)
	}
	if !lease.IP.Equal(requestedIP.To4()) {
		t.Errorf("lease IP = %s, want %s", lease.IP, requestedIP)
	}
}

func TestDHCPAllocateLeaseWithStaticLease(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPHandler(stack)

	handler.SetPool(net.ParseIP("10.0.0.10"), net.ParseIP("10.0.0.20"))

	staticMAC, _ := net.ParseMAC("00:11:22:33:44:55")
	staticIP := net.ParseIP("10.0.0.200")
	handler.SetStaticLeases([]config.DHCPLease{
		{MACAddress: staticMAC, ClientIP: staticIP},
	})

	// Allocate with static MAC - should get static IP
	lease, err := handler.AllocateLease(staticMAC, nil, "static-host")
	if err != nil {
		t.Fatalf("allocate static lease: %v", err)
	}
	if !lease.IP.Equal(staticIP) {
		t.Errorf("static lease IP = %s, want %s", lease.IP, staticIP)
	}

	// Renew static lease
	lease2, err := handler.AllocateLease(staticMAC, nil, "static-host-renewed")
	if err != nil {
		t.Fatalf("renew static lease: %v", err)
	}
	if !lease2.IP.Equal(staticIP) {
		t.Errorf("renewed static lease IP = %s, want %s", lease2.IP, staticIP)
	}
	if lease2.Hostname != "static-host-renewed" {
		t.Errorf("renewed hostname = %q, want %q", lease2.Hostname, "static-host-renewed")
	}
}

func TestDHCPIsIPInPool(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPHandler(stack)

	handler.SetPool(net.ParseIP("10.0.0.10"), net.ParseIP("10.0.0.15"))

	tests := []struct {
		ip   string
		want bool
	}{
		{"10.0.0.10", true},
		{"10.0.0.12", true},
		{"10.0.0.15", true},
		{"10.0.0.9", false},
		{"10.0.0.16", false},
		{"192.168.1.1", false},
	}

	for _, tt := range tests {
		got := handler.IsIPInPool(net.ParseIP(tt.ip))
		if got != tt.want {
			t.Errorf("IsIPInPool(%s) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestDHCPIsIPLeased(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPHandler(stack)

	handler.SetPool(net.ParseIP("10.0.0.10"), net.ParseIP("10.0.0.15"))

	mac, _ := net.ParseMAC("00:11:22:33:44:01")
	_, err := handler.AllocateLease(mac, net.ParseIP("10.0.0.10"), "test")
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}

	if !handler.DHCPIsIPLeased(net.ParseIP("10.0.0.10")) {
		t.Error("10.0.0.10 should be leased")
	}
	if handler.DHCPIsIPLeased(net.ParseIP("10.0.0.11")) {
		t.Error("10.0.0.11 should not be leased")
	}
}

func TestDHCPMessageTypeString(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPHandler(stack)

	tests := []struct {
		msgType  uint8
		expected string
	}{
		{protocols.DHCPDiscover, "DISCOVER"},
		{protocols.DHCPOffer, "OFFER"},
		{protocols.DHCPRequest, "REQUEST"},
		{protocols.DHCPDecline, "DECLINE"},
		{protocols.DHCPAck, "ACK"},
		{protocols.DHCPNak, "NAK"},
		{protocols.DHCPRelease, "RELEASE"},
		{protocols.DHCPInform, "INFORM"},
		{99, "UNKNOWN(99)"},
	}

	for _, tt := range tests {
		got := handler.DHCPMessageTypeString(tt.msgType)
		if got != tt.expected {
			t.Errorf("DHCPMessageTypeString(%d) = %q, want %q", tt.msgType, got, tt.expected)
		}
	}
}

func TestDHCPEncodeUint32(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPHandler(stack)

	tests := []struct {
		val      uint32
		expected [4]byte
	}{
		{0, [4]byte{0, 0, 0, 0}},
		{1, [4]byte{0, 0, 0, 1}},
		{256, [4]byte{0, 0, 1, 0}},
		{0x12345678, [4]byte{0x12, 0x34, 0x56, 0x78}},
	}

	for _, tt := range tests {
		got := handler.DHCPEncodeUint32(tt.val)
		if len(got) != 4 {
			t.Fatalf("encodeUint32(%d) length = %d, want 4", tt.val, len(got))
		}
		val := binary.BigEndian.Uint32(got)
		if val != tt.val {
			t.Errorf("encodeUint32(%d) decoded = %d", tt.val, val)
		}
	}
}

func TestDHCPEncodeDomainSearchList(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPHandler(stack)

	tests := []struct {
		name    string
		domains []string
		wantErr bool
	}{
		{
			name:    "single domain",
			domains: []string{"example.com"},
		},
		{
			name:    "multiple domains",
			domains: []string{"example.com", "test.local", "corp.net"},
		},
		{
			name:    "trailing dot",
			domains: []string{"example.com."},
		},
		{
			name:    "too many domains",
			domains: make([]string, 11),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkDomainSearchList(t, handler, tt.domains, tt.wantErr)
		})
	}
}

func checkDomainSearchList(t *testing.T, handler *protocols.DHCPHandler, domains []string, wantErr bool) {
	t.Helper()
	// Fill too-many-domains entries
	if len(domains) == 11 {
		for i := range domains {
			domains[i] = "example.com"
		}
	}

	result, err := handler.DHCPEncodeDomainSearchList(domains)
	if wantErr {
		if err == nil {
			t.Error("expected error, got nil")
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
	if result[len(result)-1] != 0 {
		t.Error("encoded domain should end with null byte")
	}
}

func TestSplitDomain(t *testing.T) {
	tests := []struct {
		domain string
		want   []string
	}{
		{"example.com", []string{"example", "com"}},
		{"sub.example.com", []string{"sub", "example", "com"}},
		{"example.com.", []string{"example", "com"}},
		{"", nil},
		{"single", []string{"single"}},
	}

	for _, tt := range tests {
		got := protocols.SplitDomain(tt.domain)
		if len(got) != len(tt.want) {
			t.Errorf("SplitDomain(%q) = %v, want %v", tt.domain, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("SplitDomain(%q)[%d] = %q, want %q", tt.domain, i, got[i], tt.want[i])
			}
		}
	}
}

func TestMACMatchesMask(t *testing.T) {
	tests := []struct {
		name  string
		mac   net.HardwareAddr
		match net.HardwareAddr
		mask  net.HardwareAddr
		want  bool
	}{
		{
			name:  "exact match no mask",
			mac:   net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			match: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			mask:  nil,
			want:  true,
		},
		{
			name:  "no match no mask",
			mac:   net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			match: net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
			mask:  nil,
			want:  false,
		},
		{
			name:  "match with OUI mask",
			mac:   net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			match: net.HardwareAddr{0x00, 0x11, 0x22, 0x00, 0x00, 0x00},
			mask:  net.HardwareAddr{0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00},
			want:  true,
		},
		{
			name:  "no match with OUI mask",
			mac:   net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			match: net.HardwareAddr{0xAA, 0xBB, 0xCC, 0x00, 0x00, 0x00},
			mask:  net.HardwareAddr{0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00},
			want:  false,
		},
		{
			name:  "empty mac",
			mac:   nil,
			match: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			mask:  nil,
			want:  false,
		},
		{
			name:  "empty match",
			mac:   net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			match: nil,
			mask:  nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := protocols.MACMatchesMask(tt.mac, tt.match, tt.mask)
			if got != tt.want {
				t.Errorf("MACMatchesMask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDHCPSetPoolInvalidRange(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPHandler(stack)

	// Set pool with reversed range - should result in empty pool
	handler.SetPool(net.ParseIP("10.0.0.20"), net.ParseIP("10.0.0.10"))

	pool := handler.DHCPHandlerIPPool()
	if len(pool) != 0 {
		t.Errorf("reversed range pool should be empty, got %d", len(pool))
	}
}

// ---------- DHCPv6 tests ----------

func TestDHCPv6ParseMessage(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPv6Handler(stack)

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		msgType uint8
	}{
		{
			name:    "too short",
			data:    []byte{0x01, 0x02},
			wantErr: true,
		},
		{
			name:    "minimal solicit",
			data:    []byte{0x01, 0xAA, 0xBB, 0xCC},
			msgType: protocols.DHCPv6Solicit,
		},
		{
			name: "solicit with client ID option",
			data: func() []byte {
				msg := []byte{0x01, 0x00, 0x01, 0x02} // Solicit, txn ID
				// Client ID option (code=1, length=4, data=0x01020304)
				msg = append(msg, 0x00, 0x01, 0x00, 0x04, 0x01, 0x02, 0x03, 0x04)
				return msg
			}(),
			msgType: protocols.DHCPv6Solicit,
		},
		{
			name: "option data exceeds length",
			data: func() []byte {
				msg := []byte{0x01, 0x00, 0x01, 0x02}
				// Option with length=10 but only 2 bytes of data
				msg = append(msg, 0x00, 0x01, 0x00, 0x0A, 0x01, 0x02)
				return msg
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := handler.ParseDHCPv6Message(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if msg.MessageType != tt.msgType {
				t.Errorf("message type = %d, want %d", msg.MessageType, tt.msgType)
			}
		})
	}
}

func TestDHCPv6FindOption(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPv6Handler(stack)

	msg := &protocols.DHCPv6Message{
		MessageType: protocols.DHCPv6Solicit,
		Options: []protocols.DHCPv6Option{
			{Code: protocols.DHCPv6OptClientID, Length: 4, Data: []byte{1, 2, 3, 4}},
			{Code: protocols.DHCPv6OptIANA, Length: 12, Data: make([]byte, 12)},
		},
	}

	// Found
	opt := handler.FindOption(msg, protocols.DHCPv6OptClientID)
	if opt == nil {
		t.Fatal("expected to find ClientID option")
	}
	if opt.Code != protocols.DHCPv6OptClientID {
		t.Errorf("option code = %d, want %d", opt.Code, protocols.DHCPv6OptClientID)
	}

	// Not found
	opt = handler.FindOption(msg, protocols.DHCPv6OptDNSServers)
	if opt != nil {
		t.Error("expected nil for missing option")
	}
}

func TestDHCPv6ExtractClientDUID(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPv6Handler(stack)

	duid := []byte{0x00, 0x03, 0x00, 0x01, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	msg := &protocols.DHCPv6Message{
		Options: []protocols.DHCPv6Option{
			{Code: protocols.DHCPv6OptClientID, Length: uint16(len(duid)), Data: duid},
		},
	}

	got := handler.ExtractClientDUID(msg)
	if len(got) != len(duid) {
		t.Fatalf("DUID length = %d, want %d", len(got), len(duid))
	}

	// No client ID
	emptyMsg := &protocols.DHCPv6Message{Options: []protocols.DHCPv6Option{}}
	got = handler.ExtractClientDUID(emptyMsg)
	if got != nil {
		t.Error("expected nil DUID for message without client ID")
	}
}

func TestDHCPv6ExtractIANA(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPv6Handler(stack)

	ianaData := make([]byte, 12)
	binary.BigEndian.PutUint32(ianaData[0:4], 42) // IAID=42
	msg := &protocols.DHCPv6Message{
		Options: []protocols.DHCPv6Option{
			{Code: protocols.DHCPv6OptIANA, Length: 12, Data: ianaData},
		},
	}

	iaid, ok := handler.ExtractIANA(msg)
	if !ok {
		t.Fatal("expected IANA extraction to succeed")
	}
	if iaid != 42 {
		t.Errorf("IAID = %d, want 42", iaid)
	}

	// Missing IANA
	emptyMsg := &protocols.DHCPv6Message{Options: []protocols.DHCPv6Option{}}
	_, ok = handler.ExtractIANA(emptyMsg)
	if ok {
		t.Error("expected IANA extraction to fail for empty message")
	}

	// IANA too short
	shortMsg := &protocols.DHCPv6Message{
		Options: []protocols.DHCPv6Option{
			{Code: protocols.DHCPv6OptIANA, Length: 2, Data: []byte{0, 1}},
		},
	}
	_, ok = handler.ExtractIANA(shortMsg)
	if ok {
		t.Error("expected IANA extraction to fail for short data")
	}
}

func TestDHCPv6AllocateAndConfirmLease(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPv6Handler(stack)

	pool := []net.IP{
		net.ParseIP("2001:db8::10"),
		net.ParseIP("2001:db8::11"),
	}
	handler.DHCPv6SetAddressPool(pool)

	clientDUID := []byte{0x00, 0x03, 0x00, 0x01, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0x01}

	// Allocate
	lease, err := handler.DHCPv6AllocateLease(clientDUID, 1)
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if lease.Address == nil {
		t.Fatal("lease address is nil")
	}

	// Re-allocate same client should renew
	lease2, err := handler.DHCPv6AllocateLease(clientDUID, 1)
	if err != nil {
		t.Fatalf("re-allocate: %v", err)
	}
	if !lease2.Address.Equal(lease.Address) {
		t.Error("re-allocation should return same address")
	}

	// Confirm lease
	lease3, err := handler.ConfirmLease(clientDUID, 1)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if !lease3.Address.Equal(lease.Address) {
		t.Error("confirm should return same address")
	}

	// Confirm for new client should allocate
	newDUID := []byte{0x00, 0x03, 0x00, 0x01, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0x02}
	lease4, err := handler.ConfirmLease(newDUID, 2)
	if err != nil {
		t.Fatalf("confirm new client: %v", err)
	}
	if lease4.Address.Equal(lease.Address) {
		t.Error("new client should get different address")
	}

	// Pool exhausted
	thirdDUID := []byte{0x00, 0x03, 0x00, 0x01, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0x03}
	_, err = handler.DHCPv6AllocateLease(thirdDUID, 3)
	if err == nil {
		t.Error("expected pool exhaustion error")
	}
}

func TestDHCPv6FindLeaseAndRenew(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPv6Handler(stack)

	handler.DHCPv6SetAddressPool([]net.IP{net.ParseIP("2001:db8::10")})

	clientDUID := []byte{0x00, 0x03, 0x00, 0x01, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0x01}

	// No lease yet
	found := handler.FindLease(clientDUID)
	if found != nil {
		t.Error("expected nil lease before allocation")
	}

	// Allocate
	lease, err := handler.DHCPv6AllocateLease(clientDUID, 1)
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}

	// Find it
	found = handler.FindLease(clientDUID)
	if found == nil {
		t.Fatal("expected to find lease after allocation")
	}
	if !found.Address.Equal(lease.Address) {
		t.Error("found lease has wrong address")
	}

	// Renew
	oldRenewal := found.LastRenewal
	time.Sleep(time.Millisecond) // ensure time difference
	handler.RenewLease(found)
	if !found.LastRenewal.After(oldRenewal) {
		t.Error("renewal should update LastRenewal")
	}
}

func TestDHCPv6MessageTypeString(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPv6Handler(stack)

	tests := []struct {
		msgType  uint8
		contains string
	}{
		{protocols.DHCPv6Solicit, "SOLICIT"},
		{protocols.DHCPv6Advertise, "ADVERTISE"},
		{protocols.DHCPv6Request, "REQUEST"},
		{protocols.DHCPv6Confirm, "CONFIRM"},
		{protocols.DHCPv6Renew, "RENEW"},
		{protocols.DHCPv6Rebind, "REBIND"},
		{protocols.DHCPv6Reply, "REPLY"},
		{protocols.DHCPv6Release, "RELEASE"},
		{protocols.DHCPv6Decline, "DECLINE"},
		{protocols.DHCPv6InfoRequest, "INFORMATION-REQUEST"},
		{protocols.DHCPv6RelayForw, "RELAY-FORW"},
		{protocols.DHCPv6RelayRepl, "RELAY-REPL"},
		{99, "UNKNOWN"},
	}

	for _, tt := range tests {
		got := handler.MessageTypeString(tt.msgType)
		if !strings.Contains(got, tt.contains) {
			t.Errorf("MessageTypeString(%d) = %q, should contain %q", tt.msgType, got, tt.contains)
		}
	}
}

func TestDHCPv6EncodeDomainList(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPv6Handler(stack)

	encoded := handler.EncodeDomainList([]string{"example.com", "test.local"})
	if len(encoded) == 0 {
		t.Error("expected non-empty encoded domain list")
	}
}

func TestDHCPv6BuildIANAAndIAAddrOption(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPv6Handler(stack)

	lease := &protocols.DHCPv6Lease{
		Address:           net.ParseIP("2001:db8::10"),
		IAID:              42,
		PreferredLifetime: time.Now().Add(7 * 24 * time.Hour),
		ValidLifetime:     time.Now().Add(30 * 24 * time.Hour),
	}

	ianaOpt := handler.BuildIANAOption(lease)
	if ianaOpt.Code != protocols.DHCPv6OptIANA {
		t.Errorf("IANA option code = %d, want %d", ianaOpt.Code, protocols.DHCPv6OptIANA)
	}
	if len(ianaOpt.Data) == 0 {
		t.Error("IANA option data should not be empty")
	}

	iaAddrOpt := handler.BuildIAAddrOption(lease)
	if iaAddrOpt.Code != protocols.DHCPv6OptIAAddr {
		t.Errorf("IAAddr option code = %d, want %d", iaAddrOpt.Code, protocols.DHCPv6OptIAAddr)
	}
	if len(iaAddrOpt.Data) == 0 {
		t.Error("IAAddr option data should not be empty")
	}
}

func TestDHCPv6SetServerConfigAndAdvanced(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPv6Handler(stack)

	dnsServers := []net.IP{net.ParseIP("2001:4860:4860::8888")}
	handler.DHCPv6SetServerConfig(dnsServers, []string{"example.com"})

	if len(handler.DHCPv6HandlerDNSServers()) != 1 {
		t.Error("DNS servers not set")
	}
	if len(handler.DHCPv6HandlerDomainList()) != 1 {
		t.Error("domain list not set")
	}

	sntpServers := []net.IP{net.ParseIP("2001:db8::1")}
	ntpServers := []net.IP{net.ParseIP("2001:db8::2")}
	sipServers := []net.IP{net.ParseIP("2001:db8::3")}
	handler.DHCPv6SetAdvancedOptions(sntpServers, ntpServers, sipServers, []string{"sip.example.com"})

	if len(handler.DHCPv6HandlerSNTPServers()) != 1 {
		t.Error("SNTP servers not set")
	}
	if len(handler.DHCPv6HandlerNTPServers()) != 1 {
		t.Error("NTP servers not set")
	}
	if len(handler.DHCPv6HandlerSIPServers()) != 1 {
		t.Error("SIP servers not set")
	}
	if len(handler.DHCPv6HandlerSIPDomains()) != 1 {
		t.Error("SIP domains not set")
	}
}

func TestDHCPv6MulticastAddresses(t *testing.T) {
	relay := protocols.GetAllDHCPRelayAgentsAndServersExported()
	if relay == nil {
		t.Fatal("GetAllDHCPRelayAgentsAndServers returned nil")
	}
	if relay.String() != "ff02::1:2" {
		t.Errorf("relay address = %s, want ff02::1:2", relay)
	}

	servers := protocols.GetAllDHCPServersExported()
	if servers == nil {
		t.Fatal("GetAllDHCPServers returned nil")
	}
	if servers.String() != "ff05::1:3" {
		t.Errorf("servers address = %s, want ff05::1:3", servers)
	}
}

func TestDHCPv6Reset(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPv6Handler(stack)

	handler.DHCPv6SetAddressPool([]net.IP{net.ParseIP("2001:db8::10")})
	handler.DHCPv6SetServerConfig([]net.IP{net.ParseIP("2001:db8::1")}, []string{"test.com"})

	clientDUID := []byte{0x01, 0x02, 0x03, 0x04}
	_, err := handler.DHCPv6AllocateLease(clientDUID, 1)
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}

	handler.Reset()

	if len(handler.DHCPv6HandlerLeases()) != 0 {
		t.Error("leases should be cleared after reset")
	}
	if len(handler.DHCPv6HandlerDNSServers()) != 0 {
		t.Error("DNS servers should be cleared after reset")
	}
	// DUID should be regenerated
	if len(handler.DHCPv6HandlerServerDUID()) == 0 {
		t.Error("server DUID should be regenerated after reset")
	}
}

func TestDHCPv6SetPrefixPool(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPv6Handler(stack)

	_, prefix, _ := net.ParseCIDR("2001:db8::/48")
	handler.SetPrefixPool([]net.IPNet{*prefix})

	if len(handler.DHCPv6HandlerPrefixPool()) != 1 {
		t.Error("prefix pool not set")
	}
}

// ---------- STP tests ----------

func TestSTPHandlePacket(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewSTPHandler(stack, 0)

	tests := []struct {
		name string
		pkt  func() *protocols.Packet
	}{
		{
			name: "packet too short",
			pkt: func() *protocols.Packet {
				return &protocols.Packet{Buffer: make([]byte, 20), Length: 20, SerialNumber: 1}
			},
		},
		{
			name: "invalid LLC header",
			pkt: func() *protocols.Packet {
				buf := make([]byte, 64)
				buf[14] = 0x00 // Invalid DSAP
				buf[15] = 0x00 // Invalid SSAP
				return &protocols.Packet{Buffer: buf, Length: 64, SerialNumber: 2}
			},
		},
		{
			name: "invalid protocol ID",
			pkt: func() *protocols.Packet {
				buf := make([]byte, 64)
				buf[14] = 0x42 // DSAP
				buf[15] = 0x42 // SSAP
				buf[16] = 0x03 // Control
				buf[17] = 0xFF // Bad protocol ID (high byte)
				buf[18] = 0xFF // Bad protocol ID (low byte)
				return &protocols.Packet{Buffer: buf, Length: 64, SerialNumber: 3}
			},
		},
		{
			name: "valid config BPDU",
			pkt: func() *protocols.Packet {
				buf := make([]byte, 64)
				// Ethernet header (14 bytes) - empty MACs
				// LLC header
				buf[14] = 0x42 // DSAP
				buf[15] = 0x42 // SSAP
				buf[16] = 0x03 // Control
				// BPDU header
				buf[17] = 0x00 // Protocol ID (high)
				buf[18] = 0x00 // Protocol ID (low)
				buf[19] = 0x00 // Version
				buf[20] = 0x00 // Config BPDU type
				// Config BPDU needs at least 35 bytes from offset 17
				// offset 17 + 35 = 52, so 64 byte buffer is enough
				return &protocols.Packet{Buffer: buf, Length: 64, SerialNumber: 4}
			},
		},
		{
			name: "TCN BPDU",
			pkt: func() *protocols.Packet {
				buf := make([]byte, 64)
				buf[14] = 0x42
				buf[15] = 0x42
				buf[16] = 0x03
				buf[17] = 0x00
				buf[18] = 0x00
				buf[19] = 0x00
				buf[20] = 0x80 // TCN type
				return &protocols.Packet{Buffer: buf, Length: 64, SerialNumber: 5}
			},
		},
		{
			name: "unknown BPDU type",
			pkt: func() *protocols.Packet {
				buf := make([]byte, 64)
				buf[14] = 0x42
				buf[15] = 0x42
				buf[16] = 0x03
				buf[17] = 0x00
				buf[18] = 0x00
				buf[19] = 0x00
				buf[20] = 0xFE // Unknown type
				return &protocols.Packet{Buffer: buf, Length: 64, SerialNumber: 6}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// Should not panic
			handler.HandlePacket(tt.pkt())
		})
	}
}

func TestSTPHandleConfigBPDUShort(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewSTPHandler(stack, 2) // debug level 2 for info logging

	// Config BPDU too short (less than 35 bytes from BPDU start)
	buf := make([]byte, 45) // 14 eth + 3 LLC + 28 = too short for config BPDU
	buf[14] = 0x42
	buf[15] = 0x42
	buf[16] = 0x03
	buf[17] = 0x00
	buf[18] = 0x00
	buf[19] = 0x00
	buf[20] = 0x00 // Config type
	pkt := &protocols.Packet{Buffer: buf, Length: len(buf), SerialNumber: 1}

	// Should not panic
	handler.HandlePacket(pkt)
}

func TestSTPGetSTPParams(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewSTPHandler(stack, 0)

	// Default params (no STP config)
	device := &config.Device{Name: "test"}
	bp, ht, ma, fd := handler.STPGetSTPParams(device)
	if bp != 32768 || ht != 2 || ma != 20 || fd != 15 {
		t.Errorf("default params wrong: bp=%d ht=%d ma=%d fd=%d", bp, ht, ma, fd)
	}

	// With custom STP config
	device2 := &config.Device{
		Name: "custom",
		STPConfig: &config.STPConfig{
			BridgePriority: 4096,
			HelloTime:      1,
			MaxAge:         10,
			ForwardDelay:   5,
		},
	}
	bp, ht, ma, fd = handler.STPGetSTPParams(device2)
	if bp != 4096 || ht != 1 || ma != 10 || fd != 5 {
		t.Errorf("custom params wrong: bp=%d ht=%d ma=%d fd=%d", bp, ht, ma, fd)
	}
}

func TestSTPSendConfigBPDU(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewSTPHandler(stack, 0)

	// No MAC address - should error
	device := &config.Device{Name: "test"}
	err := handler.SendConfigBPDU(device)
	if err == nil {
		t.Error("expected error for device without MAC")
	}

	// STP disabled
	device2 := &config.Device{
		Name:       "test2",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		STPConfig:  &config.STPConfig{Enabled: false},
	}
	err = handler.SendConfigBPDU(device2)
	if err != nil {
		t.Errorf("disabled STP should return nil, got: %v", err)
	}

	// Valid device - should succeed (packet goes to nil handle, but no error)
	device3 := &config.Device{
		Name:       "test3",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}
	err = handler.SendConfigBPDU(device3)
	// Send might fail because stack has no handle, but SendConfigBPDU itself shouldn't error
	// (it just calls stack.Send which does nothing with nil handle)
	if err != nil {
		t.Errorf("valid device should not error: %v", err)
	}
}

func TestSTPSetDebugLevel(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewSTPHandler(stack, 0)
	handler.STPSetDebugLevel(3)
	// No panic means success (debug level is internal)
}

// ---------- Device Table extended tests ----------

func TestDeviceTableGetByIPv6(t *testing.T) {
	dt := protocols.NewDeviceTable()

	ipv6 := net.ParseIP("2001:db8::1")
	device := &config.Device{
		Name:        "IPv6Device",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{ipv6},
	}
	dt.AddByMAC(device.MACAddress, device)
	dt.AddByIP(ipv6, device)

	found := dt.GetByIPv6(ipv6)
	if len(found) != 1 {
		t.Fatalf("expected 1 device, got %d", len(found))
	}
	if found[0].Name != "IPv6Device" {
		t.Errorf("wrong device: %s", found[0].Name)
	}
}

func TestDeviceTableTTLOperations(t *testing.T) {
	dt := protocols.NewDeviceTable()

	device1 := &config.Device{
		Name:       "TTL128",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x01},
		TTLConfig:  &config.TTLConfig{TTL: 128},
	}
	device2 := &config.Device{
		Name:       "TTL128-2",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x02},
		TTLConfig:  &config.TTLConfig{TTL: 128},
	}
	device3 := &config.Device{
		Name:       "TTL64",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x03},
		TTLConfig:  &config.TTLConfig{TTL: 64},
	}

	// Register
	dt.DTRegisterTTL(device1)
	dt.DTRegisterTTL(device2)
	dt.DTRegisterTTL(device3)

	// Nil device should be safe
	dt.DTRegisterTTL(nil)

	// Invalid TTL
	dt.DTRegisterTTL(&config.Device{TTLConfig: &config.TTLConfig{TTL: 0}})
	dt.DTRegisterTTL(&config.Device{TTLConfig: &config.TTLConfig{TTL: 256}})

	// No TTL config
	dt.DTRegisterTTL(&config.Device{Name: "noconfig"})

	// Get by TTL
	got := dt.DTGetDeviceByTTL(128)
	if got == nil {
		t.Fatal("expected device for TTL 128")
	}

	// Non-existent TTL
	got = dt.DTGetDeviceByTTL(200)
	if got != nil {
		t.Error("expected nil for TTL 200")
	}

	// Increment count and verify round-robin behavior
	dt.DTIncrementTTLCount(device1)
	got = dt.DTGetDeviceByTTL(128)
	if got == nil {
		t.Fatal("expected device after increment")
	}
	// Device2 should be selected now (lower count)
	if got.Name != "TTL128-2" {
		t.Errorf("expected TTL128-2, got %s", got.Name)
	}

	// Nil increment should be safe
	dt.DTIncrementTTLCount(nil)
}

func TestDeviceTableForwardingDevices(t *testing.T) {
	dt := protocols.NewDeviceTable()

	device1 := &config.Device{Name: "fwd1", MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x01}}
	device2 := &config.Device{Name: "fwd2", MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x02}}

	dt.DTRegisterForwardingDevice(device1)
	dt.DTRegisterForwardingDevice(device2)
	dt.DTRegisterForwardingDevice(nil) // nil should be safe

	fwd := dt.DTGetForwardingDevices()
	if len(fwd) != 2 {
		t.Errorf("expected 2 forwarding devices, got %d", len(fwd))
	}
}

func TestDeviceTableReset(t *testing.T) {
	dt := protocols.NewDeviceTable()

	device := &config.Device{
		Name:        "test",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
		TTLConfig:   &config.TTLConfig{TTL: 128},
	}
	dt.AddByMAC(device.MACAddress, device)
	dt.AddByIP(device.IPAddresses[0], device)
	dt.DTRegisterTTL(device)
	dt.DTRegisterForwardingDevice(device)

	dt.Reset()

	if dt.Count() != 0 {
		t.Error("count should be 0 after reset")
	}
	if len(dt.DTGetForwardingDevices()) != 0 {
		t.Error("forwarding devices should be empty after reset")
	}
}

// ---------- DNS tests ----------

func TestDNSLookupHost(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDNSHandler(stack)

	handler.DNSSetDomain("test.local")
	handler.DNSAddRecord("myhost.test.local", net.ParseIP("10.0.0.1"))
	handler.DNSAddRecord("myhost.test.local", net.ParseIP("10.0.0.2"))

	tests := []struct {
		name     string
		hostname string
		wantLen  int
	}{
		{"exact match", "myhost.test.local", 2},
		{"short name resolved with domain", "myhost", 2},
		{"not found", "nonexistent.test.local", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs := handler.LookupHost(tt.hostname, nil)
			if len(recs) != tt.wantLen {
				t.Errorf("lookupHost(%q) returned %d records, want %d", tt.hostname, len(recs), tt.wantLen)
			}
		})
	}
}

func TestDNSResolveQuestionsA(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDNSHandler(stack)

	handler.DNSSetDomain("corp.local")
	handler.DNSAddRecord("web.corp.local", net.ParseIP("10.10.0.100"))

	questions := []layers.DNSQuestion{
		{Name: []byte("web.corp.local"), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
	}
	answers, rcode := handler.ResolveQuestions(questions, nil, 0, 1)
	if rcode != layers.DNSResponseCodeNoErr {
		t.Errorf("response code = %d, want NoErr", rcode)
	}
	if len(answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(answers))
	}
	if !answers[0].IP.Equal(net.ParseIP("10.10.0.100")) {
		t.Errorf("answer IP = %s, want 10.10.0.100", answers[0].IP)
	}
}

func TestDNSResolveQuestionsMultipleTypes(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDNSHandler(stack)

	handler.DNSAddRecord("host.local", net.ParseIP("10.0.0.1"))
	handler.DNSAddRecord("host.local", net.ParseIP("2001:db8::1"))
	handler.AddPTRRecord(net.ParseIP("10.0.0.1"), "host.local", 300, layers.DNSResponseCodeNoErr)

	tests := []struct {
		name    string
		qType   layers.DNSType
		qName   string
		wantLen int
	}{
		{"A record", layers.DNSTypeA, "host.local", 1},
		{"AAAA record", layers.DNSTypeAAAA, "host.local", 1},
		{"NXDOMAIN", layers.DNSTypeA, "nonexistent.local", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			questions := []layers.DNSQuestion{
				{Name: []byte(tt.qName), Type: tt.qType, Class: layers.DNSClassIN},
			}
			answers, rcode := handler.ResolveQuestions(questions, nil, 0, 1)
			if len(answers) != tt.wantLen {
				t.Errorf("got %d answers, want %d", len(answers), tt.wantLen)
			}
			if tt.wantLen == 0 && rcode != layers.DNSResponseCodeNXDomain {
				t.Errorf("expected NXDOMAIN, got %d", rcode)
			}
		})
	}
}

func TestDNSLoadDeviceRecords(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDNSHandler(stack)

	devices := []*config.Device{
		{
			Name:        "router1",
			IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
			SNMPConfig:  config.SNMPConfig{SysName: "core-router"},
		},
		{
			Name:        "switch1",
			IPAddresses: []net.IP{net.ParseIP("10.0.0.2"), net.ParseIP("2001:db8::2")},
			SNMPConfig:  config.SNMPConfig{},
		},
	}

	handler.DNSLoadDeviceRecords(devices)

	// Should be able to look up by SNMP sysName
	recs := handler.LookupHost("core-router", nil)
	if len(recs) != 1 {
		t.Errorf("core-router lookup: got %d records, want 1", len(recs))
	}

	// Fallback to device name
	recs = handler.LookupHost("switch1", nil)
	if len(recs) != 2 {
		t.Errorf("switch1 lookup: got %d records, want 2", len(recs))
	}
}

func TestDNSSetDomainAndReset(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDNSHandler(stack)

	handler.DNSSetDomain("example.com")
	if handler.DNSHandlerDomain() != "example.com" {
		t.Errorf("domain = %q, want %q", handler.DNSHandlerDomain(), "example.com")
	}

	handler.DNSAddRecord("test.example.com", net.ParseIP("10.0.0.1"))
	handler.Reset()

	if handler.DNSHandlerDomain() != "local" {
		t.Errorf("domain after reset = %q, want %q", handler.DNSHandlerDomain(), "local")
	}
	if len(handler.DNSHandlerRecords()) != 0 {
		t.Error("records should be empty after reset")
	}
}

func TestDNSAddPTRRecord(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDNSHandler(stack)

	ip := net.ParseIP("10.0.0.1")
	handler.AddPTRRecord(ip, "host.example.com.", 600, layers.DNSResponseCodeNoErr)

	ptrs := handler.DNSHandlerPTRRecords()
	_, ok := ptrs[ip.String()]
	if !ok {
		t.Fatal("PTR record not found")
	}
}

// ---------- Packet tests ----------

func TestParsePacket(t *testing.T) {
	// Standard packet (no VLAN)
	data := make([]byte, 64)
	binary.BigEndian.PutUint16(data[12:14], 0x0800) // IPv4
	pkt, err := protocols.ParsePacket(data, 1)
	if err != nil {
		t.Fatalf("ParsePacket: %v", err)
	}
	if pkt.VLAN != -1 {
		t.Errorf("VLAN = %d, want -1", pkt.VLAN)
	}

	// VLAN tagged packet
	vlanData := make([]byte, 68)
	binary.BigEndian.PutUint16(vlanData[12:14], 0x8100) // VLAN tag
	binary.BigEndian.PutUint16(vlanData[14:16], 0x0064) // VLAN ID 100
	pkt, err = protocols.ParsePacket(vlanData, 2)
	if err != nil {
		t.Fatalf("ParsePacket VLAN: %v", err)
	}
	if pkt.VLAN != 100 {
		t.Errorf("VLAN = %d, want 100", pkt.VLAN)
	}
}

func TestCalculateIPChecksumBoost(t *testing.T) {
	// Simple test: all zeros should give 0xFFFF
	header := make([]byte, 20)
	checksum := protocols.CalculateIPChecksum(header)
	if checksum != 0xFFFF {
		t.Errorf("checksum for zero header = 0x%04x, want 0xFFFF", checksum)
	}

	// Known good IPv4 header checksum
	header2 := []byte{
		0x45, 0x00, 0x00, 0x3c, // version, IHL, DSCP, total length
		0x1c, 0x46, 0x40, 0x00, // identification, flags, fragment offset
		0x40, 0x06, 0x00, 0x00, // TTL, protocol, checksum (zeroed)
		0xac, 0x10, 0x0a, 0x63, // source IP
		0xac, 0x10, 0x0a, 0x0c, // dest IP
	}
	checksum2 := protocols.CalculateIPChecksum(header2)
	if checksum2 == 0 {
		t.Error("checksum should not be zero for a real header")
	}
}

func TestDecodePacket(t *testing.T) {
	// Build a valid Ethernet + IPv4 frame
	data := make([]byte, 64)
	// Ethernet header (14 bytes): dst MAC, src MAC, EtherType
	data[12] = 0x08
	data[13] = 0x00 // IPv4
	// IPv4 header starts at offset 14
	data[14] = 0x45 // Version 4, IHL 5 (20 bytes)
	data[15] = 0x00 // DSCP/ECN
	data[16] = 0x00 // Total length (high byte)
	data[17] = 0x1c // Total length = 28 (20 IP + 8 UDP)
	data[23] = 0x11 // Protocol: UDP
	data[26] = 10   // Src IP: 10.0.0.1
	data[29] = 1
	data[30] = 10 // Dst IP: 10.0.0.2
	data[33] = 2

	pkt, err := protocols.DecodePacket(data)
	if err != nil {
		t.Fatalf("DecodePacket: %v", err)
	}
	if pkt == nil {
		t.Fatal("DecodePacket returned nil packet")
	}
}

func TestNewPacketBoundsValidation(t *testing.T) {
	// Negative size should be capped
	pkt := protocols.NewPacket(-1)
	if pkt == nil {
		t.Fatal("NewPacket(-1) returned nil")
	}
	if len(pkt.Buffer) != 1514 {
		t.Errorf("negative size should default to 1514, got %d", len(pkt.Buffer))
	}

	// Oversized should be capped
	pkt = protocols.NewPacket(100000)
	if len(pkt.Buffer) != 1514 {
		t.Errorf("oversized should default to 1514, got %d", len(pkt.Buffer))
	}
}

func TestPacketBoundaryEdgeCases(t *testing.T) {
	pkt := protocols.NewPacket(10)

	// Get16 at edge
	val := pkt.Get16(9)
	if val != 0 {
		t.Errorf("Get16 at boundary should return 0, got %d", val)
	}

	// Get32 at edge
	val32 := pkt.Get32(8)
	if val32 != 0 {
		t.Errorf("Get32 at boundary should return 0, got %d", val32)
	}

	// GetMAC at edge
	mac := pkt.GetMAC(5)
	if mac != nil {
		t.Error("GetMAC at boundary should return nil")
	}

	// GetIP at edge
	ip := pkt.GetIP(7)
	if ip != nil {
		t.Error("GetIP at boundary should return nil")
	}

	// Put operations at boundary (should not panic)
	pkt.Put16(0x1234, 9)
	pkt.Put32(0x12345678, 8)
	pkt.PutMAC(net.HardwareAddr{0, 1, 2, 3, 4, 5}, 5)
	pkt.PutIP(net.ParseIP("1.2.3.4"), 7)
}

func TestGetEtherType(t *testing.T) {
	pkt := protocols.NewPacket(64)
	binary.BigEndian.PutUint16(pkt.Buffer[12:14], protocols.EtherTypeIP)

	et := pkt.GetEtherType()
	if et != protocols.EtherTypeIP {
		t.Errorf("EtherType = 0x%04x, want 0x%04x", et, protocols.EtherTypeIP)
	}
}

// ---------- DHCP Reset test ----------

func TestDHCPReset(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDHCPHandler(stack)

	handler.SetPool(net.ParseIP("10.0.0.10"), net.ParseIP("10.0.0.20"))
	handler.SetServerConfig(net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.1"),
		[]net.IP{net.ParseIP("8.8.8.8")}, "example.com")
	handler.SetAdvancedOptions(
		[]net.IP{net.ParseIP("10.0.0.100")},
		[]string{"example.com"},
		"tftp.example.com",
		"pxelinux.0",
		nil,
	)

	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	_, err := handler.AllocateLease(mac, nil, "test")
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}

	handler.Reset()

	if len(handler.DHCPHandlerLeases()) != 0 {
		t.Error("leases not cleared")
	}
	if handler.DHCPHandlerIPPool() != nil {
		t.Error("pool not cleared")
	}
	if handler.DHCPHandlerServerIP() != nil {
		t.Error("server IP not cleared")
	}
	if handler.DHCPHandlerGateway() != nil {
		t.Error("gateway not cleared")
	}
	if handler.DHCPHandlerDomainName() != "" {
		t.Error("domain name not cleared")
	}
	if handler.DHCPHandlerTFTPServerName() != "" {
		t.Error("TFTP server not cleared")
	}
	if handler.DHCPHandlerBootfileName() != "" {
		t.Error("bootfile not cleared")
	}
}

// ---------- DNS LoadDeviceDNSConfig test ----------

func TestDNSLoadDeviceDNSConfig(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDNSHandler(stack)

	// Nil device should be safe
	handler.LoadDeviceDNSConfig(nil)

	// Device without DNS config should be safe
	handler.LoadDeviceDNSConfig(&config.Device{Name: "noconfig"})

	// Device with DNS config
	device := &config.Device{
		Name: "dns-device",
		DNSConfig: &config.DNSConfig{
			ForwardRecords: []config.DNSRecord{
				{Name: "host1.example.com", IP: net.ParseIP("10.0.0.1"), TTL: 300, RCode: 0},
			},
			ReverseRecords: []config.DNSRecord{
				{Name: "host1.example.com", IP: net.ParseIP("10.0.0.1"), TTL: 300, RCode: 0},
			},
		},
	}
	handler.LoadDeviceDNSConfig(device)
}

// ---------- DNS AddRecordWithTTL test ----------

func TestDNSAddRecordWithTTL(t *testing.T) {
	stack := newTestStack(t)
	handler := protocols.NewDNSHandler(stack)

	handler.DNSAddRecordWithTTL("test.local.", net.ParseIP("10.0.0.1"), 600, layers.DNSResponseCodeNoErr)

	// Verify the record is resolvable
	recs := handler.LookupHost("test.local", nil)
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}

	// Verify TTL via resolveQuestions which returns DNSResourceRecord with exported fields
	questions := []layers.DNSQuestion{
		{Name: []byte("test.local"), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
	}
	answers, _ := handler.ResolveQuestions(questions, nil, 0, 1)
	if len(answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(answers))
	}
	if answers[0].TTL != 600 {
		t.Errorf("TTL = %d, want 600", answers[0].TTL)
	}
}
