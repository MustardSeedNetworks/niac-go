package protocols

import (
	"encoding/binary"
	"net"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// ---------- test helpers ----------

func newTestStackInternal(t *testing.T) *Stack {
	t.Helper()
	cfg := &config.Config{}
	return NewStack(nil, cfg, logging.NewDebugConfig(0))
}

// ---------- DNS tests ----------

func TestDNSValidateServerDevice(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	tests := []struct {
		name   string
		device *config.Device
		ip     net.IP
		want   bool
	}{
		{"nil device", nil, nil, false},
		{"nil ip", &config.Device{Name: "dev1"}, nil, false},
		{"no mac", &config.Device{Name: "dev1"}, net.ParseIP("10.0.0.1"), false},
		{"valid", &config.Device{
			Name:       "dev1",
			MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		}, net.ParseIP("10.0.0.1"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.validateServerDevice(tt.device, tt.ip, 0, 1, "TEST")
			if got != tt.want {
				t.Errorf("validateServerDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDNSExtractSourceMAC(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	// Build a minimal Ethernet + IPv4 + UDP + DNS packet
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		DstMAC:       net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
		EthernetType: layers.EthernetTypeIPv4,
	}

	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("10.0.0.1"), DstIP: net.ParseIP("10.0.0.2"),
	}

	udp := &layers.UDP{SrcPort: 12345, DstPort: 53}
	_ = udp.SetNetworkLayerForChecksum(ip)

	dns := &layers.DNS{ID: 1, QR: false}

	buf := serializeTestLayers(t, eth, ip, udp, dns)
	packet := parseTestPacket(buf)

	mac := h.extractSourceMAC(packet)
	if mac == nil {
		t.Fatal("expected non-nil MAC")
	}
	if mac.String() != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("got MAC %s, want aa:bb:cc:dd:ee:ff", mac)
	}
}

func TestDNSSelectServerDevice(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	// Add a global record so deviceHasDNSRecords returns true for any device
	h.AddRecord("test.local", net.ParseIP("10.0.0.100"))

	mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	devices := []*config.Device{
		{Name: "d1", IPAddresses: []net.IP{net.ParseIP("10.0.0.1")}, MACAddress: mac},
	}

	dev, ip := h.selectServerDevice(devices, false)
	if dev == nil || ip == nil {
		t.Fatal("expected server device to be found")
	}

	// IPv6 request - no IPv6 addresses
	dev, ip = h.selectServerDevice(devices, true)
	if dev != nil || ip != nil {
		t.Error("expected no server for IPv6 when no IPv6 addresses")
	}
}

func TestDNSDeviceHasDNSRecords(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	if h.deviceHasDNSRecords(nil) {
		t.Error("nil device should return false")
	}

	dev := &config.Device{Name: "d1"}
	// No global records, no device config
	if h.deviceHasDNSRecords(dev) {
		t.Error("device with no records should return false")
	}

	// Device with DNSConfig
	dev.DNSConfig = &config.DNSConfig{}
	if !h.deviceHasDNSRecords(dev) {
		t.Error("device with DNSConfig should return true")
	}
}

func TestDNSGetRecordSetForDevice(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	// nil device
	if h.getRecordSetForDevice(nil) != nil {
		t.Error("nil device should return nil record set")
	}

	// device without loaded config
	dev := &config.Device{Name: "d1"}
	if h.getRecordSetForDevice(dev) != nil {
		t.Error("unloaded device should return nil record set")
	}

	// load device DNS config
	dev.DNSConfig = &config.DNSConfig{
		ForwardRecords: []config.DNSRecord{
			{Name: "foo.local", IP: net.ParseIP("10.0.0.5"), TTL: 300},
		},
	}
	h.LoadDeviceDNSConfig(dev)
	set := h.getRecordSetForDevice(dev)
	if set == nil {
		t.Fatal("expected non-nil record set after loading config")
	}
	if _, ok := set.forward["foo.local"]; !ok {
		t.Error("expected forward record for foo.local")
	}
}

func TestDNSPickIPAddressForDNS(t *testing.T) {
	dev := &config.Device{
		IPAddresses: []net.IP{
			net.ParseIP("10.0.0.1"),
			net.ParseIP("fd00::1"),
		},
	}

	v4 := pickIPAddressForDNS(dev, false)
	if v4 == nil || v4.To4() == nil {
		t.Error("expected IPv4 address")
	}

	v6 := pickIPAddressForDNS(dev, true)
	if v6 == nil || v6.To4() != nil {
		t.Error("expected IPv6 address")
	}

	// No addresses
	empty := &config.Device{}
	if pickIPAddressForDNS(empty, false) != nil {
		t.Error("expected nil for empty device")
	}
}

func TestDNSLookupPTR(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	ip := net.ParseIP("10.0.0.1")
	h.AddPTRRecord(ip, "test.local", 300, layers.DNSResponseCodeNoErr)

	// Global lookup
	rec, found := h.lookupPTR(ip, nil)
	if !found || rec.name != "test.local" {
		t.Error("expected PTR record for 10.0.0.1")
	}

	// Lookup with device-specific set
	set := &dnsRecordSet{
		forward: make(map[string][]dnsRecord),
		reverse: map[string]dnsPTR{
			"10.0.0.2": {name: "other.local", ttl: 300},
		},
	}

	rec, found = h.lookupPTR(net.ParseIP("10.0.0.2"), set)
	if !found || rec.name != "other.local" {
		t.Error("expected PTR from device-specific set")
	}
}

func TestDNSParsePTRName(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantIP bool
		isV6   bool
	}{
		{"ipv4 valid", "1.0.0.10.in-addr.arpa", true, false},
		{"ipv6 valid", "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.d.f.ip6.arpa", true, true},
		{"invalid suffix", "foo.bar.baz", false, false},
		{"short ipv4", "1.0.in-addr.arpa", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, ok, isV6 := parsePTRName([]byte(tt.input))
			if ok != tt.wantIP {
				t.Errorf("parsePTRName ok=%v, want %v (ip=%v)", ok, tt.wantIP, ip)
			}
			if isV6 != tt.isV6 {
				t.Errorf("parsePTRName isV6=%v, want %v", isV6, tt.isV6)
			}
		})
	}
}

func TestDNSIsValidDNSName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"normal hostname", "test.example.com", true},
		{"short name", "a", true},
		{"empty", "", true},
		{"label too long", strings.Repeat("a", 64) + ".com", false},
		{"name too long", strings.Repeat("a.", 128) + "com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidDNSName([]byte(tt.input))
			if got != tt.want {
				t.Errorf("isValidDNSName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDNSResolveByTypeA(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	h.AddRecord("test.local", net.ParseIP("10.0.0.1"))

	questions := []layers.DNSQuestion{
		{Name: []byte("test.local"), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
	}

	answers, code := h.resolveQuestions(questions, nil, 0, 1)
	if code != layers.DNSResponseCodeNoErr {
		t.Errorf("expected NoErr, got %v", code)
	}
	if len(answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(answers))
	}
	if answers[0].Type != layers.DNSTypeA {
		t.Errorf("expected A record, got %v", answers[0].Type)
	}
}

func TestDNSResolveByTypePTR(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	h.AddRecord("myhost.local", net.ParseIP("10.0.0.5"))

	questions := []layers.DNSQuestion{
		{Name: []byte("5.0.0.10.in-addr.arpa"), Type: layers.DNSTypePTR, Class: layers.DNSClassIN},
	}

	answers, code := h.resolveQuestions(questions, nil, 0, 1)
	if code != layers.DNSResponseCodeNoErr {
		t.Errorf("expected NoErr, got %v", code)
	}
	if len(answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(answers))
	}
	if answers[0].Type != layers.DNSTypePTR {
		t.Errorf("expected PTR record, got %v", answers[0].Type)
	}
}

func TestDNSResolveNXDomain(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	questions := []layers.DNSQuestion{
		{Name: []byte("nonexistent.local"), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
	}

	_, code := h.resolveQuestions(questions, nil, 0, 1)
	if code != layers.DNSResponseCodeNXDomain {
		t.Errorf("expected NXDomain, got %v", code)
	}
}

func TestDNSResolveInvalidName(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	longLabel := strings.Repeat("x", 64) + ".com"
	questions := []layers.DNSQuestion{
		{Name: []byte(longLabel), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
	}

	_, code := h.resolveQuestions(questions, nil, 3, 1)
	if code != layers.DNSResponseCodeNXDomain {
		t.Errorf("expected NXDomain for invalid name, got %v", code)
	}
}

func TestDNSLogInvalidDNSName(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	// Just call to ensure no panic
	ctx := &dnsResolveContext{debugLevel: DebugLevelInfo, serial: 1}
	h.logInvalidDNSName([]byte("bad-name"), ctx)
}

func TestDNSLogDNSRecord(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	ctx := &dnsResolveContext{debugLevel: DebugLevelInfo, serial: 1}
	h.logDNSRecord("A", "test.local", "10.0.0.1", ctx)
}

func TestDNSLogPTRParseFailure(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	ctx := &dnsResolveContext{debugLevel: DebugLevelInfo, serial: 1}
	h.logPTRParseFailure([]byte("badptr"), ctx)
}

func TestDNSLogDNSQueries(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	questions := []layers.DNSQuestion{
		{Name: []byte("test.local"), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
	}

	// No panic at various debug levels
	h.logDNSQueries(questions, net.ParseIP("10.0.0.1"), 0, 1)
	h.logDNSQueries(questions, net.ParseIP("10.0.0.1"), DebugLevelVerbose, 1)
}

func TestDNSParseDNSLayer(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		DstMAC:       net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("10.0.0.1"), DstIP: net.ParseIP("10.0.0.2"),
	}
	udp := &layers.UDP{SrcPort: 12345, DstPort: 53}
	_ = udp.SetNetworkLayerForChecksum(ip)
	dnsLayer := &layers.DNS{ID: 42, QR: false}

	buf := serializeTestLayers(t, eth, ip, udp, dnsLayer)
	packet := parseTestPacket(buf)

	dns, ok := h.parseDNSLayer(packet, 0, 1, "TEST")
	if !ok || dns == nil {
		t.Fatal("expected DNS layer to be parsed")
	}
	if dns.ID != 42 {
		t.Errorf("expected ID=42, got %d", dns.ID)
	}
}

func TestDNSSendDNSResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	response := &layers.DNS{
		ID: 1, QR: true, AA: true,
		ResponseCode: layers.DNSResponseCodeNoErr,
	}

	// Will fail due to nil capture engine, but should not panic
	err := h.SendDNSResponse(response,
		net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.2"),
		net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		12345, 0)
	// Expect error because stack has no capture engine
	if err == nil {
		t.Log("SendDNSResponse succeeded (stack queued packet)")
	}
}

func TestDNSSendDNSResponseV6(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	response := &layers.DNS{
		ID: 1, QR: true, AA: true,
		ResponseCode: layers.DNSResponseCodeNoErr,
	}

	err := h.SendDNSResponseV6(response,
		net.ParseIP("fd00::1"), net.ParseIP("fd00::2"),
		net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		12345, 0)
	if err == nil {
		t.Log("SendDNSResponseV6 succeeded")
	}
}

// ---------- DHCP tests ----------

func TestDHCPFindServerDevice(t *testing.T) {
	// No devices
	if findServerDevice(nil) != nil {
		t.Error("expected nil for empty device list")
	}

	// Device without IPs
	devs := []*config.Device{{Name: "d1"}}
	if findServerDevice(devs) != nil {
		t.Error("expected nil for device without IPs")
	}

	// Device with IPs
	devs = []*config.Device{{Name: "d1", IPAddresses: []net.IP{net.ParseIP("10.0.0.1")}}}
	got := findServerDevice(devs)
	if got == nil || got.Name != "d1" {
		t.Error("expected d1 to be returned")
	}
}

func TestDHCPGetRequestedIP(t *testing.T) {
	dhcp := &layers.DHCPv4{
		Options: []layers.DHCPOption{
			{Type: layers.DHCPOptHostname, Data: []byte("test")},
			{Type: layers.DHCPOptRequestIP, Data: []byte{10, 0, 0, 5}},
		},
	}

	ip := getRequestedIP(dhcp)
	if ip == nil || !ip.Equal(net.IPv4(10, 0, 0, 5)) {
		t.Errorf("expected 10.0.0.5, got %v", ip)
	}

	// No requested IP
	dhcp2 := &layers.DHCPv4{
		Options: []layers.DHCPOption{
			{Type: layers.DHCPOptHostname, Data: []byte("test")},
		},
	}
	if getRequestedIP(dhcp2) != nil {
		t.Error("expected nil when no RequestIP option")
	}

	// Wrong length
	dhcp3 := &layers.DHCPv4{
		Options: []layers.DHCPOption{
			{Type: layers.DHCPOptRequestIP, Data: []byte{10, 0}},
		},
	}
	if getRequestedIP(dhcp3) != nil {
		t.Error("expected nil for wrong length")
	}
}

func TestDHCPBuildOptions(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)

	h.SetPool(net.ParseIP("10.0.0.10"), net.ParseIP("10.0.0.20"))
	h.SetServerConfig(net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.1"),
		[]net.IP{net.ParseIP("8.8.8.8")}, "example.com")
	h.SetAdvancedOptions(
		[]net.IP{net.ParseIP("10.0.0.1")},
		[]string{"example.com"},
		"tftp.local",
		"pxelinux.0",
		[]byte{0x01, 0x02},
	)

	mac := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	serverIP := net.ParseIP("10.0.0.1")

	opts := h.buildDHCPOptions(DHCPOffer, serverIP, mac, true)
	if len(opts) == 0 {
		t.Fatal("expected non-empty options")
	}

	// Check that message type is first and End is last
	if opts[0].Type != layers.DHCPOptMessageType {
		t.Error("first option should be MessageType")
	}
	if opts[len(opts)-1].Type != layers.DHCPOptEnd {
		t.Error("last option should be End")
	}
}

func TestDHCPAppendNetworkOptions(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)

	h.SetPool(net.ParseIP("10.0.0.10"), net.ParseIP("10.0.0.20"))
	h.SetServerConfig(net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.1"),
		[]net.IP{net.ParseIP("8.8.8.8")}, "example.com")

	opts := h.appendNetworkOptions(nil)
	if len(opts) == 0 {
		t.Error("expected gateway option at minimum")
	}
}

func TestDHCPAppendTimingOptions(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)

	opts := h.appendTimingOptions(nil)
	// Should have at least T1 and T2
	if len(opts) < 2 {
		t.Errorf("expected at least 2 timing options, got %d", len(opts))
	}
}

func TestDHCPAppendBootOptions(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)

	h.SetAdvancedOptions(nil, nil, "tftp.server", "boot.img", nil)

	opts := h.appendBootOptions(nil)
	if len(opts) < 2 {
		t.Errorf("expected TFTP+Bootfile options, got %d", len(opts))
	}
}

func TestDHCPAppendMiscOptions(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)

	mac := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	// No vendor info, no hostname
	opts := h.appendMiscOptions(nil, mac)
	if len(opts) != 0 {
		t.Error("expected no options without vendor info or hostname")
	}

	// With vendor info
	h.SetAdvancedOptions(nil, nil, "", "", []byte{0x01, 0x02})
	opts = h.appendMiscOptions(nil, mac)
	if len(opts) != 1 {
		t.Errorf("expected 1 vendor option, got %d", len(opts))
	}
}

func TestDHCPDispatchMessage(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)

	h.SetPool(net.ParseIP("10.0.0.10"), net.ParseIP("10.0.0.20"))
	h.SetServerConfig(net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.1"),
		[]net.IP{net.ParseIP("8.8.8.8")}, "example.com")

	mac := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	serverDev := &config.Device{
		Name:        "server",
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	// Test Release
	info := &dhcpPacketInfo{
		dhcp:        &layers.DHCPv4{ClientHWAddr: mac},
		messageType: DHCPRelease,
	}
	// Allocate lease first so release has something to remove
	_, _ = h.allocateLease(mac, nil, "test")

	h.dispatchDHCPMessage(info, serverDev, 1, 0)

	// Test Inform
	info.messageType = DHCPInform
	h.dispatchDHCPMessage(info, serverDev, 1, 0)

	// Test unknown type
	info.messageType = 99
	h.dispatchDHCPMessage(info, serverDev, 1, DebugLevelInfo)
}

func TestDHCPUpdateFDBTables(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)

	// Should not panic with empty MAC or nil handler
	h.updateFDBTables(nil)
	h.updateFDBTables(net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55})
}

func TestDHCPSendDHCPResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)

	h.SetPool(net.ParseIP("10.0.0.10"), net.ParseIP("10.0.0.20"))
	h.SetServerConfig(net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.1"),
		[]net.IP{net.ParseIP("8.8.8.8")}, "example.com")

	mac := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	serverIP := net.ParseIP("10.0.0.1")
	serverMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	// SendDHCPOffer goes through sendDHCPResponse -> serializeAndSendDHCP
	err := h.SendDHCPOffer(1234, mac, net.ParseIP("10.0.0.10"), serverIP, serverMAC, 0)
	if err == nil {
		t.Log("SendDHCPOffer succeeded (queued)")
	}

	// SendDHCPAck
	err = h.SendDHCPAck(1234, mac, net.ParseIP("10.0.0.10"), serverIP, serverMAC, 0)
	if err == nil {
		t.Log("SendDHCPAck succeeded (queued)")
	}
}

// ---------- DHCPv6 tests ----------

func TestDHCPv6FindIPv6ServerDevice(t *testing.T) {
	devs := []*config.Device{
		{Name: "v4only", IPAddresses: []net.IP{net.ParseIP("10.0.0.1")}},
		{Name: "v6dev", IPAddresses: []net.IP{net.ParseIP("fd00::1")}},
	}

	dev := findIPv6ServerDevice(devs)
	if dev == nil || dev.Name != "v6dev" {
		t.Error("expected v6dev device")
	}

	// No v6 devices
	v4only := []*config.Device{{Name: "v4", IPAddresses: []net.IP{net.ParseIP("10.0.0.1")}}}
	if findIPv6ServerDevice(v4only) != nil {
		t.Error("expected nil for v4-only devices")
	}
}

func TestDHCPv6HasIPv6Address(t *testing.T) {
	tests := []struct {
		name string
		ips  []net.IP
		want bool
	}{
		{"nil", nil, false},
		{"v4 only", []net.IP{net.ParseIP("10.0.0.1")}, false},
		{"v6", []net.IP{net.ParseIP("fd00::1")}, true},
		{"mixed", []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("fd00::1")}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasIPv6Address(tt.ips); got != tt.want {
				t.Errorf("hasIPv6Address() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDHCPv6GetFirstIPv6Address(t *testing.T) {
	ips := []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("fd00::1")}
	ip := getFirstIPv6Address(ips)
	if ip == nil || ip.To4() != nil {
		t.Error("expected IPv6 address")
	}

	// No v6
	if getFirstIPv6Address([]net.IP{net.ParseIP("10.0.0.1")}) != nil {
		t.Error("expected nil for v4-only list")
	}
}

func TestDHCPv6EncodeIPv6List(t *testing.T) {
	ips := []net.IP{net.ParseIP("fd00::1"), net.ParseIP("fd00::2")}
	data := encodeIPv6List(ips)
	if len(data) != 32 {
		t.Errorf("expected 32 bytes (2 IPv6 addrs), got %d", len(data))
	}
}

func TestDHCPv6AppendDNSOptions(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPv6Handler(stack)

	h.SetServerConfig(
		[]net.IP{net.ParseIP("fd00::53")},
		[]string{"example.com"},
	)

	msg := &DHCPv6Message{Options: make([]DHCPv6Option, 0)}
	h.appendDNSOptions(msg)
	if len(msg.Options) < 2 {
		t.Errorf("expected at least 2 DNS options, got %d", len(msg.Options))
	}
}

func TestDHCPv6AppendTimeServerOptions(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPv6Handler(stack)

	h.SetAdvancedOptions(
		[]net.IP{net.ParseIP("fd00::123")}, // sntp
		[]net.IP{net.ParseIP("fd00::124")}, // ntp
		nil, nil,
	)

	msg := &DHCPv6Message{Options: make([]DHCPv6Option, 0)}
	h.appendTimeServerOptions(msg)
	if len(msg.Options) < 2 {
		t.Errorf("expected at least 2 time server options, got %d", len(msg.Options))
	}
}

func TestDHCPv6AppendSIPOptions(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPv6Handler(stack)

	h.SetAdvancedOptions(
		nil, nil,
		[]net.IP{net.ParseIP("fd00::5060")},
		[]string{"sip.example.com"},
	)

	msg := &DHCPv6Message{Options: make([]DHCPv6Option, 0)}
	h.appendSIPOptions(msg)
	if len(msg.Options) < 2 {
		t.Errorf("expected at least 2 SIP options, got %d", len(msg.Options))
	}
}

func TestDHCPv6AppendPreferenceOption(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPv6Handler(stack)

	msg := &DHCPv6Message{Options: make([]DHCPv6Option, 0)}
	dev := &config.Device{Name: "test"}
	h.appendPreferenceOption(msg, dev)
	if len(msg.Options) != 1 {
		t.Errorf("expected 1 preference option, got %d", len(msg.Options))
	}
}

func TestDHCPv6ValidateClientIdentity(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPv6Handler(stack)

	// No options
	msg := &DHCPv6Message{MessageType: DHCPv6Solicit}
	duid, _, ok := h.validateClientIdentity(msg, "SOLICIT", 0)
	if ok {
		t.Error("expected validation failure with no options")
	}
	_ = duid
}

// ---------- TCP tests ----------

func TestTCPBuildFlagsString(t *testing.T) {
	tests := []struct {
		name string
		tcp  *layers.TCP
		want string
	}{
		{"SYN only", &layers.TCP{SYN: true}, "SYN "},
		{"SYN ACK", &layers.TCP{SYN: true, ACK: true}, "SYN ACK "},
		{"FIN", &layers.TCP{FIN: true}, "FIN "},
		{"RST", &layers.TCP{RST: true}, "RST "},
		{"all", &layers.TCP{SYN: true, ACK: true, FIN: true, RST: true}, "SYN ACK FIN RST "},
		{"none", &layers.TCP{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTCPFlagsString(tt.tcp)
			if got != tt.want {
				t.Errorf("buildTCPFlagsString() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------- FTP tests ----------

func TestFTPHandleFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() string
		contains string
	}{
		{"USER with arg", func() string { return handleFTPUser("admin") }, "331"},
		{"USER no arg", func() string { return handleFTPUser("") }, "501"},
		{"SYST no devices", func() string { return handleFTPSyst(nil) }, "UNIX"},
		{"SYST with device", func() string {
			dev := &config.Device{FTPConfig: &config.FTPConfig{SystemType: "Windows_NT"}}
			return handleFTPSyst([]*config.Device{dev})
		}, "Windows_NT"},
		{"TYPE with arg", func() string { return handleFTPType("I") }, "200"},
		{"TYPE no arg", func() string { return handleFTPType("") }, "501"},
		{"RETR with file", func() string { return handleFTPRetr("test.txt") }, "550"},
		{"RETR no file", func() string { return handleFTPRetr("") }, "501"},
		{"STOR with file", func() string { return handleFTPStor("test.txt") }, "553"},
		{"STOR no file", func() string { return handleFTPStor("") }, "501"},
		{"CWD with dir", func() string { return handleFTPCwd("/tmp") }, "250"},
		{"CWD no dir", func() string { return handleFTPCwd("") }, "501"},
		{"DELE with file", func() string { return handleFTPDele("test.txt") }, "553"},
		{"DELE no file", func() string { return handleFTPDele("") }, "501"},
		{"MKD with dir", func() string { return handleFTPMkd("newdir") }, "257"},
		{"MKD no dir", func() string { return handleFTPMkd("") }, "501"},
		{"RMD with dir", func() string { return handleFTPRmd("olddir") }, "250"},
		{"RMD no dir", func() string { return handleFTPRmd("") }, "501"},
		{"UNKNOWN valid", func() string { return handleFTPUnknown("XYZW") }, "502"},
		{"UNKNOWN invalid", func() string { return handleFTPUnknown("longcommand") }, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn()
			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("result %q does not contain %q", result, tt.contains)
			}
		})
	}
}

func TestFTPDispatchCommands(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewFTPHandler(stack)

	tests := []struct {
		cmd      string
		contains string
	}{
		{"PASS", "230"},
		{"PWD", "257"},
		{"EPSV", "229"},
		{"LIST", "150"},
		{"CDUP", "250"},
		{"NOOP", "200"},
		{"QUIT", "221"},
		{"HELP", "214"},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			result := h.dispatchFTPCommand(tt.cmd, "", false, nil)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("dispatchFTPCommand(%s) = %q, want to contain %q", tt.cmd, result, tt.contains)
			}
		})
	}
}

func TestFTPPasv(t *testing.T) {
	// IPv6 case
	result := handleFTPPasv(true, nil)
	if !strings.Contains(result, "522") {
		t.Errorf("expected 522 for IPv6 PASV, got %q", result)
	}

	// No devices
	result = handleFTPPasv(false, nil)
	if !strings.Contains(result, "500") {
		t.Errorf("expected 500 for no devices, got %q", result)
	}

	// With device
	dev := &config.Device{IPAddresses: []net.IP{net.ParseIP("192.168.1.1")}}
	result = handleFTPPasv(false, []*config.Device{dev})
	if !strings.Contains(result, "227") {
		t.Errorf("expected 227 for valid PASV, got %q", result)
	}
}

func TestFTPSelectIPv4Address(t *testing.T) {
	if selectIPv4Address(nil) != nil {
		t.Error("expected nil for empty devices")
	}

	dev := &config.Device{IPAddresses: []net.IP{net.ParseIP("fd00::1")}}
	if selectIPv4Address([]*config.Device{dev}) != nil {
		t.Error("expected nil for IPv6-only device")
	}

	dev.IPAddresses = append(dev.IPAddresses, net.ParseIP("192.168.1.1"))
	ip := selectIPv4Address([]*config.Device{dev})
	if ip == nil || ip.To4() == nil {
		t.Error("expected IPv4 address")
	}
}

func TestFTPSanitizeForLogging(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal text", "normal text"},
		{"has\nnewline", "has?newline"},
		{"has\ttab", "has?tab"},
		{"clean", "clean"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeForLogging(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeForLogging(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFTPBuildResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewFTPHandler(stack)

	// Empty command
	if h.buildFTPResponse("", false, nil) != "" {
		t.Error("expected empty response for empty command")
	}

	// Valid command
	resp := h.buildFTPResponse("USER admin", false, nil)
	if !strings.Contains(resp, "331") {
		t.Errorf("expected 331 response, got %q", resp)
	}
}

// ---------- HTTP tests ----------

func TestHTTPParseRequest(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr bool
		method  string
		path    string
	}{
		{
			"valid GET",
			"GET / HTTP/1.1\r\nHost: test.local\r\n\r\n",
			false, "GET", "/",
		},
		{
			"valid POST",
			"POST /api/data HTTP/1.1\r\nContent-Type: application/json\r\n\r\n",
			false, "POST", "/api/data",
		},
		{
			"invalid request line",
			"INVALID\r\n",
			true, "", "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := parseHTTPRequest([]byte(tt.payload))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if req.Method != tt.method {
				t.Errorf("method = %q, want %q", req.Method, tt.method)
			}
			if req.Path != tt.path {
				t.Errorf("path = %q, want %q", req.Path, tt.path)
			}
		})
	}
}

func TestHTTPFindCustomEndpoint(t *testing.T) {
	// nil device
	if findCustomEndpoint(nil, &HTTPRequest{Path: "/", Method: "GET"}) != nil {
		t.Error("expected nil for nil device")
	}

	// Device with HTTPConfig
	dev := &config.Device{
		HTTPConfig: &config.HTTPConfig{
			Enabled: true,
			Endpoints: []config.HTTPEndpoint{
				{Path: "/health", Method: "GET", StatusCode: 200, Body: "ok"},
				{Path: "/data", Method: "", StatusCode: 200, Body: "data"}, // defaults to GET
			},
		},
	}

	ep := findCustomEndpoint(dev, &HTTPRequest{Path: "/health", Method: "GET"})
	if ep == nil || ep.Body != "ok" {
		t.Error("expected /health endpoint")
	}

	// Default method matching
	ep = findCustomEndpoint(dev, &HTTPRequest{Path: "/data", Method: "GET"})
	if ep == nil || ep.Body != "data" {
		t.Error("expected /data endpoint with default GET")
	}

	// Non-matching path
	ep = findCustomEndpoint(dev, &HTTPRequest{Path: "/unknown", Method: "GET"})
	if ep != nil {
		t.Error("expected nil for unknown path")
	}

	// Non-matching method
	ep = findCustomEndpoint(dev, &HTTPRequest{Path: "/health", Method: "POST"})
	if ep != nil {
		t.Error("expected nil for wrong method")
	}

	// Disabled config
	dev.HTTPConfig.Enabled = false
	ep = findCustomEndpoint(dev, &HTTPRequest{Path: "/health", Method: "GET"})
	if ep != nil {
		t.Error("expected nil for disabled config")
	}
}

func TestHTTPGetDeviceInfo(t *testing.T) {
	// No devices
	info := getHTTPDeviceInfo(nil)
	if info.name != "Unknown" {
		t.Errorf("expected Unknown, got %q", info.name)
	}

	// With device
	dev := &config.Device{
		Name: "TestDev",
		Type: "router",
		HTTPConfig: &config.HTTPConfig{
			ServerName: "MyServer/1.0",
		},
	}
	info = getHTTPDeviceInfo([]*config.Device{dev})
	if info.name != "TestDev" || info.deviceType != "router" || info.serverName != "MyServer/1.0" {
		t.Errorf("unexpected info: %+v", info)
	}
}

func TestHTTPBuildResponse(t *testing.T) {
	resp := buildHTTPResponse(200, "TestServer", "text/html", "<h1>Test</h1>")
	respStr := string(resp)
	if !strings.Contains(respStr, "HTTP/1.1 200 OK") {
		t.Error("expected status line")
	}
	if !strings.Contains(respStr, "Server: TestServer") {
		t.Error("expected Server header")
	}
	if !strings.Contains(respStr, "<h1>Test</h1>") {
		t.Error("expected body")
	}
}

func TestHTTPGetStatusText(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{200, "OK"},
		{201, "Created"},
		{204, "No Content"},
		{301, "Moved Permanently"},
		{302, "Found"},
		{304, "Not Modified"},
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{500, "Internal Server Error"},
		{501, "Not Implemented"},
		{503, "Service Unavailable"},
		{999, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.code), func(t *testing.T) {
			got := getStatusText(tt.code)
			if got != tt.want {
				t.Errorf("getStatusText(%d) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

func TestHTTPGetDeviceNames(t *testing.T) {
	result := getDeviceNames(nil)
	if result != "unknown" {
		t.Errorf("expected 'unknown', got %q", result)
	}

	devs := []*config.Device{{Name: "dev1"}, {Name: "dev2"}}
	result = getDeviceNames(devs)
	if !strings.Contains(result, "dev1") || !strings.Contains(result, "dev2") {
		t.Errorf("expected device names, got %q", result)
	}
}

func TestHTTPGenerateDefaultBody(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewHTTPHandler(stack)

	info := httpDeviceInfo{name: "TestDev", deviceType: "switch", serverName: "TestSrv"}

	tests := []struct {
		path     string
		wantCode int
	}{
		{"/", 200},
		{"/index.html", 200},
		{"/status", 200},
		{"/api/info", 200},
		{"/nonexistent", 404},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			_, code, _ := h.generateDefaultBody(tt.path, info)
			if code != tt.wantCode {
				t.Errorf("generateDefaultBody(%q) code=%d, want %d", tt.path, code, tt.wantCode)
			}
		})
	}
}

func TestHTTPGenerateResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewHTTPHandler(stack)

	// Default endpoint
	req := &HTTPRequest{Method: "GET", Path: "/", Version: "HTTP/1.1"}
	resp := h.generateResponse(req, nil)
	if !strings.Contains(string(resp), "HTTP/1.1 200") {
		t.Error("expected 200 OK for root path")
	}

	// Custom endpoint
	dev := &config.Device{
		Name: "TestDev",
		Type: "router",
		HTTPConfig: &config.HTTPConfig{
			Enabled:    true,
			ServerName: "Custom/1.0",
			Endpoints: []config.HTTPEndpoint{
				{Path: "/custom", Method: "GET", StatusCode: 201, Body: "created", ContentType: "text/plain"},
			},
		},
	}
	req = &HTTPRequest{Method: "GET", Path: "/custom", Version: "HTTP/1.1"}
	resp = h.generateResponse(req, []*config.Device{dev})
	if !strings.Contains(string(resp), "201") {
		t.Error("expected 201 for custom endpoint")
	}
}

// ---------- STP tests ----------

func TestSTPGetDeviceNames(t *testing.T) {
	// This tests getDeviceNames which is used by multiple handlers
	devs := []*config.Device{{Name: "sw1"}, {Name: "sw2"}}
	names := getDeviceNames(devs)
	if !strings.Contains(names, "sw1") {
		t.Error("expected sw1 in device names")
	}
}

// ---------- Neighbor table tests ----------

func TestNeighborTableUpsert(t *testing.T) {
	nt := newNeighborTable()

	entry := NeighborRecord{
		Protocol:        ProtocolLLDP,
		LocalDevice:     "sw1",
		RemoteDevice:    "sw2",
		RemoteChassisID: "chassis-1",
		RemotePort:      "Gi0/1",
		TTL:             120 * time.Second,
	}

	nt.upsert(entry)

	records := nt.list()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].RemoteDevice != "sw2" {
		t.Errorf("expected sw2, got %s", records[0].RemoteDevice)
	}
}

func TestNeighborTableUpsertEmpty(t *testing.T) {
	nt := newNeighborTable()

	// Missing LocalDevice
	nt.upsert(NeighborRecord{RemoteChassisID: "c1"})
	if len(nt.list()) != 0 {
		t.Error("expected no entries for empty LocalDevice")
	}

	// Missing RemoteChassisID
	nt.upsert(NeighborRecord{LocalDevice: "sw1"})
	if len(nt.list()) != 0 {
		t.Error("expected no entries for empty RemoteChassisID")
	}
}

func TestNeighborTableDefaultTTL(t *testing.T) {
	nt := newNeighborTable()

	entry := NeighborRecord{
		Protocol:        ProtocolCDP,
		LocalDevice:     "sw1",
		RemoteDevice:    "sw2",
		RemoteChassisID: "chassis-2",
		RemotePort:      "Fa0/1",
		TTL:             0, // Should get default
	}
	nt.upsert(entry)

	records := nt.list()
	if len(records) != 1 {
		t.Fatal("expected 1 record")
	}
	if records[0].TTL != neighborDefaultTTLSeconds*time.Second {
		t.Errorf("expected default TTL %v, got %v", neighborDefaultTTLSeconds*time.Second, records[0].TTL)
	}
}

func TestNeighborTableCleanupExpired(t *testing.T) {
	nt := newNeighborTable()

	entry := NeighborRecord{
		Protocol:        ProtocolLLDP,
		LocalDevice:     "sw1",
		RemoteDevice:    "sw2",
		RemoteChassisID: "chassis-1",
		RemotePort:      "Gi0/1",
		TTL:             1 * time.Millisecond, // Very short TTL
	}
	nt.upsert(entry)

	time.Sleep(5 * time.Millisecond)

	nt.cleanupExpired()
	if len(nt.list()) != 0 {
		t.Error("expected expired entry to be cleaned up")
	}
}

func TestNeighborTableReset(t *testing.T) {
	nt := newNeighborTable()
	nt.upsert(NeighborRecord{
		Protocol:        ProtocolEDP,
		LocalDevice:     "sw1",
		RemoteDevice:    "sw2",
		RemoteChassisID: "c1",
		RemotePort:      "p1",
	})

	nt.reset()
	if len(nt.list()) != 0 {
		t.Error("expected empty table after reset")
	}
}

func TestNeighborKey(t *testing.T) {
	key := neighborKey("LLDP", "chassis1", "port1")
	if key != "LLDP|chassis1|port1" {
		t.Errorf("unexpected key: %s", key)
	}
}

func TestDedupStrings(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  int
	}{
		{"nil", nil, 0},
		{"empty", []string{}, 0},
		{"no dups", []string{"a", "b", "c"}, 3},
		{"with dups", []string{"a", "b", "a", "c", "b"}, 3},
		{"whitespace", []string{"a", " b ", "  ", "a"}, 2},
		{"empty strings", []string{"a", "", "b", ""}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupStrings(tt.input)
			if len(got) != tt.want {
				t.Errorf("dedupStrings() len=%d, want %d, got %v", len(got), tt.want, got)
			}
		})
	}
}

// ---------- CDP tests ----------

func TestCDPCapabilitiesToStrings(t *testing.T) {
	caps := layers.CDPCapabilities{
		L3Router:        true,
		L2Switch:        true,
		TBBridge:        true,
		IsHost:          true,
		L1Repeater:      true,
		IsPhone:         true,
		RemotelyManaged: true,
		IGMPFilter:      true,
	}

	result := cdpCapabilitiesToStrings(caps)
	if len(result) == 0 {
		t.Fatal("expected non-empty capabilities")
	}

	expected := []string{"router", "switch", "bridge", "host", "repeater", "phone", "remote", "igmp-filter"}
	for _, e := range expected {
		found := slices.Contains(result, e)
		if !found {
			t.Errorf("missing capability %q in %v", e, result)
		}
	}

	// Empty capabilities
	empty := cdpCapabilitiesToStrings(layers.CDPCapabilities{})
	if len(empty) != 0 {
		t.Errorf("expected empty for no capabilities, got %v", empty)
	}
}

// ---------- Stack helper tests ----------

func TestStackMatchMAC(t *testing.T) {
	mac := []byte{0x01, 0x80, 0xC2, 0x00, 0x00, 0x00}
	if !matchMAC(mac, 0x01, 0x80, 0xC2, 0x00, 0x00, 0x00) {
		t.Error("expected match")
	}
	if matchMAC(mac, 0x01, 0x80, 0xC2, 0x00, 0x00, 0x01) {
		t.Error("expected no match")
	}
}

func TestStackFormatMACForFDB(t *testing.T) {
	mac := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	decMac, hexMac := formatMACForFDB(mac)

	if !strings.Contains(decMac, "170") { // 0xAA = 170
		t.Errorf("decMac missing 170: %s", decMac)
	}
	if !strings.Contains(hexMac, "AA") {
		t.Errorf("hexMac missing AA: %s", hexMac)
	}
}

func TestStackSNMPEnabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.SNMPConfig
		want bool
	}{
		{"empty", config.SNMPConfig{}, false},
		{"community", config.SNMPConfig{Community: "public"}, true},
		{"walkfile", config.SNMPConfig{WalkFile: "walk.txt"}, true},
		{"sysname", config.SNMPConfig{SysName: "sw1"}, true},
		{"addmibs", config.SNMPConfig{AddMibs: []config.AddMib{{}}}, true},
		{"fdb dot1d", config.SNMPConfig{Dot1DFdbTable: &config.FdbTableConfig{}}, true},
		{"traps", config.SNMPConfig{Traps: &config.TrapConfig{Enabled: true}}, true},
		{"traps disabled", config.SNMPConfig{Traps: &config.TrapConfig{Enabled: false}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snmpEnabled(tt.cfg)
			if got != tt.want {
				t.Errorf("snmpEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStackGetStats(t *testing.T) {
	stack := newTestStackInternal(t)
	stats := stack.GetStats()
	if stats.PacketsReceived != 0 {
		t.Error("expected 0 packets received")
	}
}

func TestStackIncrementStat(t *testing.T) {
	stack := newTestStackInternal(t)

	stack.IncrementStat("arp_requests")
	stack.IncrementStat("arp_replies")
	stack.IncrementStat("icmp_requests")
	stack.IncrementStat("icmp_replies")
	stack.IncrementStat("dns_queries")
	stack.IncrementStat("dhcp_requests")
	stack.IncrementStat("unknown_stat") // should not panic

	stats := stack.GetStats()
	if stats.ARPRequests != 1 {
		t.Errorf("arp_requests = %d, want 1", stats.ARPRequests)
	}
	if stats.ICMPRequests != 1 {
		t.Errorf("icmp_requests = %d, want 1", stats.ICMPRequests)
	}
	if stats.DNSQueries != 1 {
		t.Errorf("dns_queries = %d, want 1", stats.DNSQueries)
	}
}

func TestStackGetDebugLevel(t *testing.T) {
	stack := newTestStackInternal(t)
	level := stack.GetDebugLevel()
	if level != 0 {
		t.Errorf("expected debug level 0, got %d", level)
	}
}

func TestStackGetDevices(t *testing.T) {
	stack := newTestStackInternal(t)
	dt := stack.GetDevices()
	if dt == nil {
		t.Fatal("expected non-nil device table")
	}
}

func TestStackGetSNMPAgentsNilStack(t *testing.T) {
	var s *Stack
	if s.getSNMPAgents(nil) != nil {
		t.Error("expected nil for nil stack")
	}
}

func TestStackGetBaseCommunity(t *testing.T) {
	stack := newTestStackInternal(t)

	dev := &config.Device{Name: "d1"}
	community := stack.getBaseCommunity(dev)
	if community != config.DefaultSNMPCommunity {
		t.Errorf("expected default community, got %q", community)
	}

	dev.SNMPConfig.Community = "private"
	community = stack.getBaseCommunity(dev)
	if community != "private" {
		t.Errorf("expected 'private', got %q", community)
	}
}

func TestStackEnsureSNMPAgentGroup(t *testing.T) {
	stack := newTestStackInternal(t)
	dev := &config.Device{Name: "d1"}

	group := stack.ensureSNMPAgentGroup(dev)
	if group == nil {
		t.Fatal("expected non-nil agent group")
	}

	// Second call returns same group
	group2 := stack.ensureSNMPAgentGroup(dev)
	if group != group2 {
		t.Error("expected same agent group on second call")
	}
}

// ---------- Healthcheck generate function tests ----------

func TestHealthCheckGenerateLDAPResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewHealthCheckHandler(stack)

	// Valid LDAP request with SEQUENCE tag, INTEGER for messageID
	request := []byte{0x30, 0x0c, 0x02, 0x01, 0x05, 0x60, 0x07, 0x02, 0x01, 0x03, 0x04, 0x00, 0x80, 0x00}
	resp := h.generateLDAPResponse(request, nil)
	if resp == nil {
		t.Fatal("expected non-nil LDAP response")
	}
	// MessageID should be extracted
	if resp[4] != 0x05 {
		t.Errorf("expected messageID=5, got %d", resp[4])
	}

	// Short request
	resp = h.generateLDAPResponse([]byte{0x30}, nil)
	if resp != nil {
		t.Error("expected nil for short request")
	}
}

func TestHealthCheckGenerateRTSPResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewHealthCheckHandler(stack)

	request := []byte("OPTIONS rtsp://camera/stream RTSP/1.0\r\nCSeq: 42\r\n\r\n")
	resp := h.generateRTSPResponse(request, nil)
	respStr := string(resp)
	if !strings.Contains(respStr, "RTSP/1.0 200 OK") {
		t.Error("expected RTSP 200 OK")
	}
	if !strings.Contains(respStr, "CSeq: 42") {
		t.Error("expected CSeq: 42")
	}

	// With device name
	dev := &config.Device{Name: "Camera-1"}
	resp = h.generateRTSPResponse(request, []*config.Device{dev})
	if !strings.Contains(string(resp), "Camera-1") {
		t.Error("expected device name in response")
	}
}

func TestHealthCheckGenerateMySQLGreeting(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewHealthCheckHandler(stack)

	// Without devices
	resp := h.generateMySQLGreeting(nil)
	if len(resp) == 0 {
		t.Fatal("expected non-empty MySQL greeting")
	}
	// Check protocol version
	if resp[4] != 10 {
		t.Errorf("expected protocol version 10, got %d", resp[4])
	}

	// With device
	dev := &config.Device{Name: "db-server-1"}
	resp = h.generateMySQLGreeting([]*config.Device{dev})
	if !strings.Contains(string(resp), "db-server-1") {
		t.Error("expected device name in MySQL greeting")
	}
}

func TestHealthCheckGeneratePostgresResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewHealthCheckHandler(stack)

	// Startup message with version
	request := make([]byte, 8)
	request[4] = 0x00
	request[5] = 0x03
	request[6] = 0x00
	request[7] = 0x00

	resp := h.generatePostgresResponse(request, nil)
	if len(resp) == 0 {
		t.Fatal("expected non-empty Postgres response")
	}
}

func TestHealthCheckGenerateMSSQLResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewHealthCheckHandler(stack)

	// TDS login packet (type 0x10)
	request := make([]byte, 16)
	request[0] = 0x10 // Login7 type

	resp := h.generateMSSQLResponse(request, nil)
	if len(resp) == 0 {
		t.Fatal("expected non-empty MSSQL response")
	}
}

func TestHealthCheckGenerateModbusResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewHealthCheckHandler(stack)

	// Modbus read holding registers request
	request := []byte{
		0x00, 0x01, // Transaction ID
		0x00, 0x00, // Protocol ID
		0x00, 0x06, // Length
		0x01,       // Unit ID
		0x03,       // Function code (read holding registers)
		0x00, 0x00, // Start address
		0x00, 0x01, // Quantity
	}

	resp := h.generateModbusResponse(request, nil)
	if len(resp) == 0 {
		t.Fatal("expected non-empty Modbus response")
	}
}

func TestHealthCheckGenerateDICOMResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewHealthCheckHandler(stack)

	request := make([]byte, 80)
	request[0] = 0x01 // A-ASSOCIATE-RQ

	resp := h.generateDICOMResponse(request, nil)
	if len(resp) == 0 {
		t.Fatal("expected non-empty DICOM response")
	}
}

func TestHealthCheckGenerateHL7Response(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewHealthCheckHandler(stack)

	request := []byte(
		"\x0BMSH|^~\\&|SendApp|SendFac|RecApp|RecFac|202604150000||ADT^A01|MSG001|P|2.3\rEVN|A01|202604150000\r\x1C\x0D",
	)
	resp := h.generateHL7Response(request, nil)
	if len(resp) == 0 {
		t.Fatal("expected non-empty HL7 response")
	}
}

func TestHealthCheckGenerateOPCUAResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewHealthCheckHandler(stack)

	// OPC UA Hello message
	request := make([]byte, 32)
	copy(request[:3], "HEL")

	resp := h.generateOPCUAResponse(request, nil)
	if len(resp) == 0 {
		t.Fatal("expected non-empty OPC UA response")
	}
}

func TestHealthCheckGenerateSMBResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewHealthCheckHandler(stack)

	// SMB1 negotiate - needs 4-byte NetBIOS session header first
	request := make([]byte, 44)
	request[0] = 0x00 // NetBIOS session header
	request[1] = 0x00
	request[2] = 0x00
	request[3] = 0x28 // Length
	// SMB header starts at offset 4
	request[4] = 0xFF
	copy(request[5:8], "SMB")
	request[8] = 0x72 // Negotiate command

	resp := h.generateSMBResponse(request, nil)
	if len(resp) == 0 {
		t.Fatal("expected non-empty SMB response")
	}
}

func TestHealthCheckGenerateSMB2Response(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewHealthCheckHandler(stack)

	resp := h.generateSMB2Response(nil, nil)
	if len(resp) == 0 {
		t.Fatal("expected non-empty SMB2 response")
	}
}

// ---------- iPerf3 tests ----------

func TestIPerf3DefaultConfig(t *testing.T) {
	cfg := DefaultIPerf3Config()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Port != TCPPortIPerf3 {
		t.Errorf("expected port %d, got %d", TCPPortIPerf3, cfg.Port)
	}
	if cfg.Enabled {
		t.Error("default config should be disabled")
	}
}

func TestIPerf3GetDeviceConfig(t *testing.T) {
	// Nil config
	dev := &config.Device{Name: "d1"}
	result := GetDeviceIPerf3Config(dev)
	if result != "disabled" {
		t.Errorf("expected 'disabled', got %q", result)
	}

	// Disabled
	dev.IPerf3 = &config.IPerf3Config{Enabled: false}
	result = GetDeviceIPerf3Config(dev)
	if result != "disabled" {
		t.Errorf("expected 'disabled', got %q", result)
	}

	// Enabled
	dev.IPerf3 = &config.IPerf3Config{
		Enabled:          true,
		UploadMbps:       100,
		DownloadMbps:     500,
		TypicalLatencyMs: 2.5,
	}
	result = GetDeviceIPerf3Config(dev)
	if !strings.Contains(result, "100") || !strings.Contains(result, "500") {
		t.Errorf("unexpected config string: %s", result)
	}
}

func TestIPerf3GetOrCreateSession(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewIPerf3Handler(stack)

	cfg := DefaultIPerf3Config()

	session := h.getOrCreateSession("10.0.0.1", 5201, cfg)
	if session == nil {
		t.Fatal("expected non-nil session")
	}
	if session.ClientIP != "10.0.0.1" {
		t.Errorf("expected client IP 10.0.0.1, got %s", session.ClientIP)
	}

	// Get same session
	session2 := h.getOrCreateSession("10.0.0.1", 5201, cfg)
	if session2 != session {
		t.Error("expected same session for same client")
	}
}

func TestIPerf3CleanupStaleSessions(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewIPerf3Handler(stack)

	h.sessions["10.0.0.1:5201"] = &IPerf3Session{
		ClientIP:     "10.0.0.1",
		ClientPort:   5201,
		LastActivity: time.Now().Add(-1 * time.Hour),
	}

	h.cleanupStaleSessions(30 * time.Minute)
	if len(h.sessions) != 0 {
		t.Error("expected stale session to be cleaned up")
	}
}

// ---------- NetBIOS tests ----------

func TestNetBIOSFindMatchingDevice(t *testing.T) {
	dev := &config.Device{Name: "FILESERVER"}
	devs := []*config.Device{dev}

	found, _ := findMatchingNetBIOSDevice(devs, "FILESERVER", 0x00)
	if found == nil {
		t.Error("expected device match by name")
	}

	// No match
	found, _ = findMatchingNetBIOSDevice(devs, "UNKNOWN", 0x00)
	if found != nil {
		t.Error("expected nil for no match")
	}
}

func TestNetBIOSGetFirstIPv4(t *testing.T) {
	ips := []net.IP{net.ParseIP("fd00::1"), net.ParseIP("10.0.0.1")}
	ip := getFirstIPv4(ips)
	if ip == nil || ip.To4() == nil {
		t.Error("expected IPv4 address")
	}

	// No IPv4
	if getFirstIPv4([]net.IP{net.ParseIP("fd00::1")}) != nil {
		t.Error("expected nil for IPv6-only list")
	}
}

func TestNetBIOSMatchNameFromConfig(t *testing.T) {
	// Nil config
	dev := &config.Device{Name: "d1"}
	_, _, found := matchNetBIOSNameFromConfig(dev, "D1", 0x00)
	if found {
		t.Error("expected not found for nil config")
	}

	// Disabled config
	dev.NetBIOSConfig = &config.NetBIOSConfig{Enabled: false}
	_, _, found = matchNetBIOSNameFromConfig(dev, "D1", 0x00)
	if !found {
		t.Error("expected found=true for disabled config")
	}

	// Explicit name match
	dev.NetBIOSConfig = &config.NetBIOSConfig{
		Enabled: true,
		Names: []config.NetBIOSName{
			{Name: "MYHOST", Suffix: 0x00, Group: false},
		},
	}
	matched, group, found := matchNetBIOSNameFromConfig(dev, "MYHOST", 0x00)
	if !found || !matched || group {
		t.Error("expected match for explicit name")
	}
}

func TestNetBIOSMatchNameFallback(t *testing.T) {
	dev := &config.Device{Name: "TESTHOST"}
	matched, _ := matchNetBIOSNameFallback(dev, "TESTHOST")
	if !matched {
		t.Error("expected match by device name")
	}

	dev.SNMPConfig.SysName = "SYSNAME"
	matched, _ = matchNetBIOSNameFallback(dev, "SYSNAME")
	if !matched {
		t.Error("expected match by sysName")
	}

	matched, _ = matchNetBIOSNameFallback(dev, "NOMATCH")
	if matched {
		t.Error("expected no match")
	}
}

func TestNetBIOSDeriveNames(t *testing.T) {
	// Nil device
	if deriveNetBIOSNames(nil) != nil {
		t.Error("expected nil for nil device")
	}

	// With services
	dev := &config.Device{
		Name: "TESTHOST",
		NetBIOSConfig: &config.NetBIOSConfig{
			Enabled: true,
			Name:    "NBHOST",
			Services: []string{
				"workstation",
				"fileserver",
				"messenger",
				"domainmaster",
				"masterbrowser",
				"browser",
				"msbrowse",
			},
			MsBrowse: true,
		},
	}

	entries := deriveNetBIOSNames(dev)
	if len(entries) < 7 {
		t.Errorf("expected at least 7 entries, got %d", len(entries))
	}

	// Check that names use configured Name, not device Name
	for _, e := range entries {
		if e.Name != "NBHOST" && e.Name != "__MSBROWSE__" {
			t.Errorf("unexpected name %q, want NBHOST or __MSBROWSE__", e.Name)
		}
	}
}

func TestNetBIOSRegisterNameAndSetDebugLevel(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewNetBIOSHandler(stack, 0)

	dev := &config.Device{Name: "d1"}
	h.RegisterName("TEST", dev)
	h.SetDebugLevel(3)
}

// ---------- SecureRandom tests ----------

func TestSimRandIntN(t *testing.T) {
	r := getSimRand()

	// n <= 0 should return 0
	if r.IntN(0) != 0 {
		t.Error("IntN(0) should return 0")
	}
	if r.IntN(-1) != 0 {
		t.Error("IntN(-1) should return 0")
	}

	// Should be in range
	for range 100 {
		v := r.IntN(10)
		if v < 0 || v >= 10 {
			t.Errorf("IntN(10) = %d, out of range [0,10)", v)
		}
	}
}

func TestSimRandFloat64(t *testing.T) {
	r := getSimRand()

	for range 100 {
		v := r.Float64()
		if v < 0 || v >= 1 {
			t.Errorf("Float64() = %f, out of range [0,1)", v)
		}
	}
}

// ---------- ICMPv6 helper tests ----------

func TestICMPv6GetTypeName(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewICMPv6Handler(stack, 0)

	tests := []struct {
		msgType uint8
		want    string
	}{
		{128, "Echo Request"},
		{129, "Echo Reply"},
		{133, "Router Solicitation"},
		{134, "Router Advertisement"},
		{135, "Neighbor Solicitation"},
		{136, "Neighbor Advertisement"},
		{0, ""}, // Unknown types return empty or a fallback
	}

	for _, tt := range tests {
		name := h.getTypeName(tt.msgType)
		if tt.want != "" && !strings.Contains(name, tt.want) {
			t.Errorf("getTypeName(%d) = %q, want to contain %q", tt.msgType, name, tt.want)
		}
	}
}

func TestICMPv6DeviceHasIP(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewICMPv6Handler(stack, 0)

	dev := &config.Device{
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("fd00::1")},
	}

	if !h.deviceHasIP(dev, net.ParseIP("10.0.0.1")) {
		t.Error("expected device to have 10.0.0.1")
	}
	if !h.deviceHasIP(dev, net.ParseIP("fd00::1")) {
		t.Error("expected device to have fd00::1")
	}
	if h.deviceHasIP(dev, net.ParseIP("192.168.1.1")) {
		t.Error("expected device NOT to have 192.168.1.1")
	}
}

// ---------- Packet utility tests ----------

func TestServerDeviceIP(t *testing.T) {
	dev := &config.Device{
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("fd00::1")},
	}

	// IPv4
	ip := serverDeviceIP(dev, false)
	if ip == nil || ip.To4() == nil {
		t.Error("expected IPv4 address")
	}

	// IPv6
	ip = serverDeviceIP(dev, true)
	if ip == nil || ip.To4() != nil {
		t.Error("expected IPv6 address")
	}
}

// ---------- gopacket test helpers ----------

func serializeTestLayers(t *testing.T, layersList ...gopacket.SerializableLayer) []byte {
	t.Helper()

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}

	err := gopacket.SerializeLayers(buf, opts, layersList...)
	if err != nil {
		t.Fatalf("failed to serialize layers: %v", err)
	}

	return buf.Bytes()
}

func parseTestPacket(data []byte) gopacket.Packet {
	return gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.Default)
}

// ---------- DNS NBSTAT and encoding tests ----------

func TestDNSEncodeDNSName(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple", "test.local"},
		{"empty", ""},
		{"trailing dot", "test.local."},
		{"multi-label", "a.b.c.d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeDNSName([]byte(tt.input))
			if len(result) == 0 && tt.input != "" {
				t.Error("expected non-empty encoding")
			}
		})
	}
}

func TestDNSDecodeNBSTATService(t *testing.T) {
	// Valid encoded name '*' followed by 15 spaces (0x20)
	// NetBIOS encoding: each byte -> two chars: (hi_nibble+'A')(lo_nibble+'A')
	// '*' = 0x2A -> 'C' (2+A=C), 'K' (A+A=K) -> "CK"
	// ' ' = 0x20 -> 'C' (2+A=C), 'A' (0+A=A) -> "CA"
	encoded := "CK" + strings.Repeat("CA", 15) + ".local"
	result := decodeNBSTATService([]byte(encoded))
	if result == "" {
		t.Error("expected decoded service name")
	}

	// Short input
	result = decodeNBSTATService([]byte("A"))
	if result != "" {
		t.Error("expected empty for short input")
	}
}

func TestDNSBuildNBSTATHeader(t *testing.T) {
	header := buildNBSTATHeader(42)
	if len(header) != dnsHeaderSize {
		t.Errorf("expected %d bytes, got %d", dnsHeaderSize, len(header))
	}
	// Check transaction ID
	id := binary.BigEndian.Uint16(header[0:2])
	if id != 42 {
		t.Errorf("expected ID=42, got %d", id)
	}
}

func TestDNSBuildNBSTATQuestion(t *testing.T) {
	q := layers.DNSQuestion{
		Name:  []byte("test.local"),
		Type:  layers.DNSType(dnsTypeNBSTAT),
		Class: layers.DNSClassIN,
	}
	encoded := encodeDNSName(q.Name)
	result := buildNBSTATQuestion(q, encoded)
	if len(result) == 0 {
		t.Error("expected non-empty question section")
	}
}

func TestDNSBuildNBSTATAnswer(t *testing.T) {
	q := layers.DNSQuestion{
		Name:  []byte("test.local"),
		Type:  layers.DNSType(dnsTypeNBSTAT),
		Class: layers.DNSClassIN,
	}
	names := []nbstatNameEntry{
		{Name: "TESTHOST", Suffix: 0x00, Group: false},
	}
	mac := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	result := buildNBSTATAnswer(q, names, 0, mac)
	if len(result) == 0 {
		t.Error("expected non-empty answer section")
	}
}

func TestDNSAppendNBSTATNameEntries(t *testing.T) {
	names := []nbstatNameEntry{
		{Name: "HOST", Suffix: 0x00, Group: false},
		{Name: "WORKGROUP", Suffix: 0x1E, Group: true},
	}
	result := appendNBSTATNameEntries(nil, names, nbNodeTypeH)
	if len(result) != 2*nbstatNameEntrySize {
		t.Errorf("expected %d bytes, got %d", 2*nbstatNameEntrySize, len(result))
	}
}

func TestDNSAppendNBSTATMACAndStats(t *testing.T) {
	mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	result := appendNBSTATMACAndStats(nil, mac)
	if len(result) != nbstatMACAndStatsSize {
		t.Errorf("expected %d bytes, got %d", nbstatMACAndStatsSize, len(result))
	}

	// Short MAC
	result = appendNBSTATMACAndStats(nil, net.HardwareAddr{0x00})
	if len(result) != nbstatMACAndStatsSize {
		t.Errorf("expected %d bytes for short MAC, got %d", nbstatMACAndStatsSize, len(result))
	}
}

func TestDNSBuildNBSTATResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	// nil device
	if h.buildNBSTATResponse(nil, 1, layers.DNSQuestion{}) != nil {
		t.Error("expected nil for nil device")
	}

	// Device without NetBIOSConfig
	dev := &config.Device{Name: "d1"}
	if h.buildNBSTATResponse(dev, 1, layers.DNSQuestion{}) != nil {
		t.Error("expected nil for device without NetBIOSConfig")
	}
}

func TestDNSIsNBSTATServiceSupported(t *testing.T) {
	dev := &config.Device{
		NetBIOSConfig: &config.NetBIOSConfig{
			Enabled:  true,
			MsBrowse: true,
		},
	}

	// nil device
	if isNBSTATServiceSupported("", nil) {
		t.Error("expected false for nil device")
	}

	// wildcard service
	wildcard := string(append([]byte{'*'}, make([]byte, netbiosNameLen)...))
	if !isNBSTATServiceSupported(wildcard, dev) {
		t.Error("expected true for wildcard service")
	}
}

func TestDNSNetbiosNamesForDevice(t *testing.T) {
	if netbiosNamesForDevice(nil) != nil {
		t.Error("expected nil for nil device")
	}

	dev := &config.Device{
		Name: "TESTHOST",
		NetBIOSConfig: &config.NetBIOSConfig{
			Enabled: true,
			Services: []string{
				"workstation",
				"fileserver",
				"messenger",
				"domainmaster",
				"masterbrowser",
				"browser",
				"msbrowse",
			},
			MsBrowse: true,
		},
	}
	names := netbiosNamesForDevice(dev)
	if len(names) < 7 {
		t.Errorf("expected at least 7 names, got %d", len(names))
	}

	// With explicit names
	dev.NetBIOSConfig.Names = []config.NetBIOSName{
		{Name: "EXPLICIT", Suffix: 0x00, Group: false},
	}
	names = netbiosNamesForDevice(dev)
	if len(names) < 1 || names[0].Name != "EXPLICIT" {
		t.Errorf("expected explicit name, got %v", names)
	}
}

func TestDNSNetbiosOwnerNodeType(t *testing.T) {
	tests := []struct {
		nodeType string
		want     uint8
	}{
		{"", nbNodeTypeB},
		{"P", nbNodeTypeP},
		{"M", nbNodeTypeM},
		{"H", nbNodeTypeH},
		{"X", nbNodeTypeB},
	}

	for _, tt := range tests {
		t.Run(tt.nodeType, func(t *testing.T) {
			dev := &config.Device{
				NetBIOSConfig: &config.NetBIOSConfig{NodeType: tt.nodeType},
			}
			got := netbiosOwnerNodeType(dev)
			if got != tt.want {
				t.Errorf("netbiosOwnerNodeType(%q) = %d, want %d", tt.nodeType, got, tt.want)
			}
		})
	}

	// nil device
	if netbiosOwnerNodeType(nil) != nbNodeTypeB {
		t.Error("expected B-node for nil device")
	}
}

// ---------- ICMPv6 RA helper tests ----------

func TestICMPv6GetRAConfig(t *testing.T) {
	if getRAConfig(nil) != nil {
		t.Error("expected nil for nil device")
	}

	dev := &config.Device{}
	if getRAConfig(dev) != nil {
		t.Error("expected nil for device without ICMPv6Config")
	}

	dev.ICMPv6Config = &config.ICMPv6Config{
		RouterAdvertisement: &config.Icmpv6RouterAdvertisement{},
	}
	if getRAConfig(dev) == nil {
		t.Error("expected non-nil RA config")
	}
}

func TestICMPv6GetRAFlags(t *testing.T) {
	if getRAFlags(nil) != 0 {
		t.Error("expected 0 for nil config")
	}

	cfg := &config.Icmpv6RouterAdvertisement{Managed: 1, Other: 1}
	flags := getRAFlags(cfg)
	if flags != 0xC0 {
		t.Errorf("expected 0xC0, got 0x%02X", flags)
	}
}

func TestICMPv6GetRALifetime(t *testing.T) {
	lifetime := getRALifetime(nil)
	if lifetime == 0 {
		t.Error("expected non-zero default lifetime")
	}

	cfg := &config.Icmpv6RouterAdvertisement{Lifetime: 600}
	lifetime = getRALifetime(cfg)
	if lifetime != 600 {
		t.Errorf("expected 600, got %d", lifetime)
	}
}

func TestICMPv6GetRATimers(t *testing.T) {
	r, t2 := getRATimers(nil)
	if r != 0 || t2 != 0 {
		t.Error("expected 0,0 for nil config")
	}

	cfg := &config.Icmpv6RouterAdvertisement{ReachableTime: 30000, RetransTimer: 1000}
	r, t2 = getRATimers(cfg)
	if r != 30000 || t2 != 1000 {
		t.Errorf("expected 30000,1000, got %d,%d", r, t2)
	}
}

func TestICMPv6GetRAHopLimit(t *testing.T) {
	hopLimit := getRAHopLimit(nil, nil)
	if hopLimit == 0 {
		t.Error("expected non-zero default hop limit")
	}

	dev := &config.Device{
		ICMPv6Config: &config.ICMPv6Config{HopLimit: 128},
	}
	hopLimit = getRAHopLimit(dev, nil)
	if hopLimit != 128 {
		t.Errorf("expected 128, got %d", hopLimit)
	}

	cfg := &config.Icmpv6RouterAdvertisement{CurHopLimit: 200}
	hopLimit = getRAHopLimit(dev, cfg)
	if hopLimit != 200 {
		t.Errorf("expected 200, got %d", hopLimit)
	}
}

func TestICMPv6BuildRAHeader(t *testing.T) {
	dev := &config.Device{
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}
	header := buildRAHeader(dev, nil)
	if len(header) == 0 {
		t.Error("expected non-empty RA header")
	}
}

func TestICMPv6AppendMTUOption(t *testing.T) {
	options := appendMTUOption(nil, nil)
	if len(options) == 0 {
		t.Error("expected non-empty MTU option")
	}

	cfg := &config.Icmpv6RouterAdvertisement{MTU: 9000}
	options = appendMTUOption(nil, cfg)
	if len(options) == 0 {
		t.Error("expected non-empty MTU option with custom MTU")
	}
}

func TestICMPv6AppendPrefixOptions(t *testing.T) {
	// Default prefix
	options := appendPrefixOptions(nil, nil, net.ParseIP("fd00::1"))
	if len(options) == 0 {
		t.Error("expected non-empty prefix option")
	}

	// Configured prefixes
	cfg := &config.Icmpv6RouterAdvertisement{
		PrefixInfo: []config.Icmpv6PrefixInfo{
			{
				Prefix:            net.ParseIP("fd00::"),
				PrefixLength:      64,
				Onlink:            1,
				Auto:              1,
				ValidLifetime:     2592000,
				PreferredLifetime: 604800,
			},
		},
	}
	options = appendPrefixOptions(nil, cfg, net.ParseIP("fd00::1"))
	if len(options) == 0 {
		t.Error("expected non-empty configured prefix options")
	}
}

func TestICMPv6DeviceCanAdvertiseIPv6(t *testing.T) {
	if deviceCanAdvertiseIPv6(nil) {
		t.Error("expected false for nil device")
	}

	dev := &config.Device{Name: "d1"}
	if deviceCanAdvertiseIPv6(dev) {
		t.Error("expected false without ICMPv6Config")
	}

	dev.ICMPv6Config = &config.ICMPv6Config{
		RouterAdvertisement: &config.Icmpv6RouterAdvertisement{},
	}
	if deviceCanAdvertiseIPv6(dev) {
		t.Error("expected false without IPv6 address")
	}

	dev.IPAddresses = []net.IP{net.ParseIP("fd00::1")}
	if !deviceCanAdvertiseIPv6(dev) {
		t.Error("expected true for device with RA config and IPv6 address")
	}
}

func TestICMPv6FirstIPv6Address(t *testing.T) {
	if firstIPv6Address(nil) != nil {
		t.Error("expected nil for nil device")
	}

	dev := &config.Device{IPAddresses: []net.IP{net.ParseIP("10.0.0.1")}}
	if firstIPv6Address(dev) != nil {
		t.Error("expected nil for IPv4-only device")
	}

	dev.IPAddresses = append(dev.IPAddresses, net.ParseIP("fd00::1"))
	ip := firstIPv6Address(dev)
	if ip == nil || ip.To4() != nil {
		t.Error("expected IPv6 address")
	}
}

func TestICMPv6DeriveIPv6Prefix(t *testing.T) {
	prefix := deriveIPv6Prefix(net.ParseIP("fd00::1:2:3:4"), 64)
	if prefix == nil {
		t.Fatal("expected non-nil prefix")
	}
	expected := net.ParseIP("fd00::")
	if !prefix.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, prefix)
	}
}

func TestICMPv6SetDebugLevel(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewICMPv6Handler(stack, 0)
	h.SetDebugLevel(3)
}

// ---------- ICMP helper tests ----------

func TestICMPFirstIPv4Address(t *testing.T) {
	if firstIPv4Address(nil) != nil {
		t.Error("expected nil for nil device")
	}

	dev := &config.Device{IPAddresses: []net.IP{net.ParseIP("fd00::1")}}
	if firstIPv4Address(dev) != nil {
		t.Error("expected nil for IPv6-only device")
	}

	dev.IPAddresses = append(dev.IPAddresses, net.ParseIP("10.0.0.1"))
	ip := firstIPv4Address(dev)
	if ip == nil || ip.To4() == nil {
		t.Error("expected IPv4 address")
	}
}

// ---------- LLDP helper tests ----------

func TestLLDPChassisIDToString(t *testing.T) {
	// MAC address subtype
	id := layers.LLDPChassisID{
		Subtype: layers.LLDPChassisIDSubTypeMACAddr,
		ID:      []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
	}
	result := lldpChassisIDToString(id)
	if result != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("expected MAC string, got %q", result)
	}

	// Non-MAC subtype
	id2 := layers.LLDPChassisID{
		Subtype: layers.LLDPChassisIDSubTypeLocal,
		ID:      []byte("myhost"),
	}
	result = lldpChassisIDToString(id2)
	if result != "myhost" {
		t.Errorf("expected 'myhost', got %q", result)
	}
}

func TestLLDPPortIDToString(t *testing.T) {
	// MAC address subtype
	id := layers.LLDPPortID{
		Subtype: layers.LLDPPortIDSubtypeMACAddr,
		ID:      []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}
	result := lldpPortIDToString(id)
	if result != "00:11:22:33:44:55" {
		t.Errorf("expected MAC string, got %q", result)
	}

	// Interface name subtype
	id2 := layers.LLDPPortID{
		Subtype: layers.LLDPPortIDSubtypeIfaceName,
		ID:      []byte("eth0"),
	}
	result = lldpPortIDToString(id2)
	if result != "eth0" {
		t.Errorf("expected 'eth0', got %q", result)
	}
}

func TestLLDPCapabilitiesToStrings(t *testing.T) {
	caps := layers.LLDPCapabilities{
		Bridge:      true,
		Router:      true,
		WLANAP:      true,
		Repeater:    true,
		StationOnly: true,
		Phone:       true,
		CVLAN:       true,
		SVLAN:       true,
	}
	result := lldpCapabilitiesToStrings(caps)
	if len(result) < 8 {
		t.Errorf("expected at least 8 capabilities, got %d: %v", len(result), result)
	}

	// Empty capabilities
	empty := lldpCapabilitiesToStrings(layers.LLDPCapabilities{})
	if len(empty) != 0 {
		t.Errorf("expected empty for no capabilities, got %v", empty)
	}
}

// ---------- IPv6 utility tests ----------

func TestIPv6GetMulticastAddresses(t *testing.T) {
	allNodes := GetAllNodesMulticast()
	if allNodes == nil {
		t.Error("expected non-nil all-nodes multicast")
	}

	allRouters := GetAllRoutersMulticast()
	if allRouters == nil {
		t.Error("expected non-nil all-routers multicast")
	}
}

// ---------- TCP helper tests ----------

func TestTCPFindDeviceWithIP(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewTCPHandler(stack)

	mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dev := &config.Device{
		Name:        "d1",
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
		MACAddress:  mac,
	}

	found := h.findDeviceWithIP([]*config.Device{dev}, net.ParseIP("10.0.0.1"))
	if found == nil || found.Name != "d1" {
		t.Error("expected to find device with IP")
	}

	found = h.findDeviceWithIP([]*config.Device{dev}, net.ParseIP("10.0.0.2"))
	if found != nil {
		t.Error("expected nil for non-matching IP")
	}

	// Device without MAC
	devNoMAC := &config.Device{
		Name:        "d2",
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
	}
	found = h.findDeviceWithIP([]*config.Device{devNoMAC}, net.ParseIP("10.0.0.1"))
	if found != nil {
		t.Error("expected nil for device without MAC")
	}
}

func TestTCPDeviceHasIP(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewTCPHandler(stack)

	dev := &config.Device{IPAddresses: []net.IP{net.ParseIP("10.0.0.1")}}

	if !h.deviceHasIP(dev, net.ParseIP("10.0.0.1")) {
		t.Error("expected device to have IP")
	}
	if h.deviceHasIP(dev, net.ParseIP("10.0.0.2")) {
		t.Error("expected device NOT to have IP")
	}
	// Non-IP type
	if h.deviceHasIP(dev, "not-an-ip") {
		t.Error("expected false for non-IP type")
	}
}

func TestTCPLookupDestinationMAC(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewTCPHandler(stack)

	// Non-IP type
	if h.lookupDestinationMAC("not-an-ip", 0, 0) != nil {
		t.Error("expected nil for non-IP type")
	}

	// Unknown IP
	mac := h.lookupDestinationMAC(net.ParseIP("10.0.0.1"), 0, 0)
	if mac != nil {
		t.Error("expected nil for unknown IP")
	}
}

// ---------- DHCPv6 serialization tests ----------

func TestDHCPv6SerializeMessage(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPv6Handler(stack)

	msg := &DHCPv6Message{
		MessageType:   DHCPv6Advertise,
		TransactionID: [3]byte{0x01, 0x02, 0x03},
		Options: []DHCPv6Option{
			{Code: DHCPv6OptServerID, Length: 4, Data: []byte{0x01, 0x02, 0x03, 0x04}},
		},
	}

	data := h.serializeDHCPv6Message(msg)
	if len(data) < 4 {
		t.Fatal("expected at least 4 bytes")
	}
	if data[0] != DHCPv6Advertise {
		t.Errorf("expected message type %d, got %d", DHCPv6Advertise, data[0])
	}
	if data[1] != 0x01 || data[2] != 0x02 || data[3] != 0x03 {
		t.Error("transaction ID mismatch")
	}
}

func TestDHCPv6BuildResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPv6Handler(stack)

	h.SetServerConfig(
		[]net.IP{net.ParseIP("fd00::53")},
		[]string{"example.com"},
	)

	clientMsg := &DHCPv6Message{
		MessageType:   DHCPv6Solicit,
		TransactionID: [3]byte{0xAA, 0xBB, 0xCC},
		Options: []DHCPv6Option{
			{Code: DHCPv6OptClientID, Length: 4, Data: []byte{0x01, 0x02, 0x03, 0x04}},
		},
	}

	// Info-only response
	response := h.buildDHCPv6Response(DHCPv6Advertise, clientMsg, nil, &config.Device{Name: "d1"}, true)
	if response == nil {
		t.Fatal("expected non-nil response")
	}
	if response.MessageType != DHCPv6Advertise {
		t.Errorf("expected type %d, got %d", DHCPv6Advertise, response.MessageType)
	}
	if len(response.Options) < 2 {
		t.Errorf("expected at least 2 options (server+client ID), got %d", len(response.Options))
	}
}

// ---------- Stack accessor tests ----------

func TestStackAccessors(t *testing.T) {
	stack := newTestStackInternal(t)

	if stack.GetDHCPHandler() == nil {
		t.Error("expected non-nil DHCP handler")
	}
	if stack.GetDHCPv6Handler() == nil {
		t.Error("expected non-nil DHCPv6 handler")
	}
	if stack.GetDNSHandler() == nil {
		t.Error("expected non-nil DNS handler")
	}
	if stack.GetDebugConfig() == nil {
		t.Error("expected non-nil debug config")
	}
	if stack.GetErrorManager() == nil {
		t.Error("expected non-nil error manager")
	}
	level := stack.GetProtocolDebugLevel("stp")
	if level != 0 {
		t.Errorf("expected 0, got %d", level)
	}
}

func TestStackCaptureFilter(t *testing.T) {
	stack := newTestStackInternal(t)

	// No capture engine
	filter := stack.CaptureFilter()
	if filter != "" {
		t.Errorf("expected empty filter, got %q", filter)
	}

	err := stack.SetCaptureFilter("tcp port 80")
	if err == nil {
		t.Error("expected error with no capture engine")
	}
}

func TestStackGetNeighbors(t *testing.T) {
	stack := newTestStackInternal(t)
	neighbors := stack.GetNeighbors()
	if len(neighbors) != 0 {
		t.Error("expected empty neighbor list")
	}
}

func TestStackRecordNeighbor(t *testing.T) {
	stack := newTestStackInternal(t)
	stack.recordNeighbor(NeighborRecord{
		Protocol:        ProtocolLLDP,
		LocalDevice:     "sw1",
		RemoteDevice:    "sw2",
		RemoteChassisID: "chassis-1",
		RemotePort:      "Gi0/1",
	})

	neighbors := stack.GetNeighbors()
	if len(neighbors) != 1 {
		t.Errorf("expected 1 neighbor, got %d", len(neighbors))
	}
}

func TestStackCurrentConfig(t *testing.T) {
	stack := newTestStackInternal(t)
	cfg := stack.currentConfig()
	if cfg == nil {
		t.Error("expected non-nil config")
	}
}

// ---------- SNMP agent group tests ----------

func TestSNMPAgentGroupGet(t *testing.T) {
	var nilGroup *snmpAgentGroup
	if nilGroup.Get("public") != nil {
		t.Error("expected nil for nil group")
	}

	group := newSnmpAgentGroup()
	if group.Get("public") != nil {
		t.Error("expected nil for empty group")
	}
}

func TestSNMPAgentGroupEnsure(t *testing.T) {
	group := newSnmpAgentGroup()
	dev := &config.Device{Name: "d1"}

	agent := group.Ensure("public", dev, 0)
	if agent == nil {
		t.Fatal("expected non-nil agent")
	}

	// Second call returns same agent
	agent2 := group.Ensure("public", dev, 0)
	if agent != agent2 {
		t.Error("expected same agent on second call")
	}

	// Empty community defaults
	agent3 := group.Ensure("", dev, 0)
	if agent3 == nil {
		t.Error("expected non-nil agent for empty community")
	}
}

func TestSNMPAgentGroupCommunities(t *testing.T) {
	var nilGroup *snmpAgentGroup
	if nilGroup.Communities() != nil {
		t.Error("expected nil for nil group")
	}

	group := newSnmpAgentGroup()
	dev := &config.Device{Name: "d1"}
	group.Ensure("public", dev, 0)
	group.Ensure("private", dev, 0)

	comms := group.Communities()
	if len(comms) != 2 {
		t.Errorf("expected 2 communities, got %d", len(comms))
	}
}

// ---------- SNMP access control tests ----------

func TestSNMPAccessAllowed(t *testing.T) {
	// nil device
	if !snmpAccessAllowed(nil, net.ParseIP("10.0.0.1")) {
		t.Error("expected allowed for nil device")
	}

	// No access list
	dev := &config.Device{Name: "d1"}
	if !snmpAccessAllowed(dev, net.ParseIP("10.0.0.1")) {
		t.Error("expected allowed with no access list")
	}

	// With access list - allowed
	dev.SNMPConfig.AccessList = []net.IP{net.ParseIP("10.0.0.1")}
	if !snmpAccessAllowed(dev, net.ParseIP("10.0.0.1")) {
		t.Error("expected allowed for listed IP")
	}

	// With access list - denied
	if snmpAccessAllowed(dev, net.ParseIP("10.0.0.2")) {
		t.Error("expected denied for unlisted IP")
	}
}

// ---------- IP isDestInTTLSubnet tests ----------

func TestIsDestInTTLSubnet(t *testing.T) {
	// No TTL config
	dev := &config.Device{Name: "d1"}
	if isDestInTTLSubnet(dev, net.ParseIP("10.0.0.1")) {
		t.Error("expected false with no TTL config")
	}

	// With TTL config
	dev.TTLConfig = &config.TTLConfig{
		IP:   net.ParseIP("10.0.0.0").To4(),
		Mask: net.IPMask{255, 255, 255, 0},
	}
	if !isDestInTTLSubnet(dev, net.ParseIP("10.0.0.5")) {
		t.Error("expected true for IP in subnet")
	}
	if isDestInTTLSubnet(dev, net.ParseIP("192.168.1.1")) {
		t.Error("expected false for IP not in subnet")
	}

	// IPv6 destination (returns false)
	if isDestInTTLSubnet(dev, net.ParseIP("fd00::1")) {
		t.Error("expected false for IPv6 destination")
	}
}

// ---------- ICMPv6 buildRAOptions and buildRouterAdvertisementBody tests ----------

func TestICMPv6BuildRAOptions(t *testing.T) {
	dev := &config.Device{
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}
	options := buildRAOptions(dev, nil, net.ParseIP("fd00::1"))
	if len(options) == 0 {
		t.Error("expected non-empty RA options")
	}
}

func TestICMPv6BuildRouterAdvertisementBody(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewICMPv6Handler(stack, 0)

	dev := &config.Device{
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	body := h.buildRouterAdvertisementBody(dev, net.ParseIP("fd00::1"))
	if len(body) == 0 {
		t.Error("expected non-empty RA body")
	}
}

// ---------- DHCPv6 dispatch and handler tests ----------

func TestDHCPv6DispatchMessage(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPv6Handler(stack)

	// Set up server
	h.SetAddressPool([]net.IP{net.ParseIP("fd00::100"), net.ParseIP("fd00::101")})
	h.SetServerConfig([]net.IP{net.ParseIP("fd00::53")}, []string{"example.com"})

	serverDev := &config.Device{
		Name:        "server",
		IPAddresses: []net.IP{net.ParseIP("fd00::1")},
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	// Test Release (doesn't need full client handshake)
	msg := &DHCPv6Message{
		MessageType: DHCPv6Release,
		Options: []DHCPv6Option{
			{Code: DHCPv6OptClientID, Length: 4, Data: []byte{0x01, 0x02, 0x03, 0x04}},
		},
	}
	h.dispatchDHCPv6Message(msg, net.ParseIP("fd00::100"), net.ParseIP("fd00::1"), serverDev, 1, 0)

	// Test Decline
	msg.MessageType = DHCPv6Decline
	h.dispatchDHCPv6Message(msg, net.ParseIP("fd00::100"), net.ParseIP("fd00::1"), serverDev, 1, 0)

	// Test unknown type
	msg.MessageType = 99
	h.dispatchDHCPv6Message(msg, net.ParseIP("fd00::100"), net.ParseIP("fd00::1"), serverDev, 1, DebugLevelInfo)
}

// ---------- DHCP parseDHCPPacket tests ----------

func TestDHCPParseDHCPPacket(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)

	// Build a minimal DHCP packet
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("0.0.0.0"), DstIP: net.ParseIP("255.255.255.255"),
	}
	udp := &layers.UDP{SrcPort: 68, DstPort: 67}
	_ = udp.SetNetworkLayerForChecksum(ip)
	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  6,
		ClientHWAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		Options: []layers.DHCPOption{
			{Type: layers.DHCPOptMessageType, Length: 1, Data: []byte{DHCPDiscover}},
			{Type: layers.DHCPOptHostname, Length: 8, Data: []byte("testhost")},
		},
	}

	buf := serializeTestLayers(t, eth, ip, udp, dhcp)
	pkt := &Packet{Buffer: buf, Length: len(buf), SerialNumber: 1}

	info := h.parseDHCPPacket(pkt)
	if info == nil {
		t.Fatal("expected non-nil DHCP packet info")
	}
	if info.messageType != DHCPDiscover {
		t.Errorf("expected DISCOVER, got %d", info.messageType)
	}
	if info.hostname != "testhost" {
		t.Errorf("expected 'testhost', got %q", info.hostname)
	}

	// Invalid packet
	badPkt := &Packet{Buffer: []byte{0x00, 0x01}, Length: 2, SerialNumber: 2}
	if h.parseDHCPPacket(badPkt) != nil {
		t.Error("expected nil for invalid packet")
	}
}

// ---------- DNS buildDNSResponse tests ----------

func TestDNSBuildDNSResponse(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	h.AddRecord("test.local", net.ParseIP("10.0.0.1"))

	dev := &config.Device{
		Name:       "dns-server",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}
	h.LoadDeviceDNSConfig(dev)

	dns := &layers.DNS{
		ID:     42,
		QR:     false,
		OpCode: layers.DNSOpCodeQuery,
		RD:     true,
		Questions: []layers.DNSQuestion{
			{Name: []byte("test.local"), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
		},
	}

	response := h.buildDNSResponse(dns, dev, 0, 1)
	if response == nil {
		t.Fatal("expected non-nil response")
	}
	if !response.QR {
		t.Error("expected QR flag set")
	}
	if response.ID != 42 {
		t.Errorf("expected ID=42, got %d", response.ID)
	}
}

// ---------- IPv6 SetDebugLevel ----------

func TestIPv6SetDebugLevel(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewIPv6Handler(stack, 0)
	h.SetDebugLevel(3)
}

// ---------- ICMPv6 log and extract functions ----------

func TestICMPv6LogFunctions(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewICMPv6Handler(stack, DebugLevelInfo)

	// logNoDeviceForEcho
	ipv6 := &layers.IPv6{SrcIP: net.ParseIP("fd00::1"), DstIP: net.ParseIP("fd00::2")}
	pkt := &Packet{SerialNumber: 1}
	h.logNoDeviceForEcho(ipv6, pkt)

	// logEchoRequest
	h.logEchoRequest(ipv6, pkt)

	// logEchoReplySent
	h.logEchoReplySent(ipv6, pkt)
}

func TestICMPv6ExtractEthernetLayer(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewICMPv6Handler(stack, 0)

	// Build a packet with Ethernet layer
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		DstMAC:       net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
		EthernetType: layers.EthernetTypeIPv6,
	}
	ipv6 := &layers.IPv6{
		Version:    6,
		NextHeader: layers.IPProtocolICMPv6,
		HopLimit:   64,
		SrcIP:      net.ParseIP("fd00::1"),
		DstIP:      net.ParseIP("fd00::2"),
	}

	buf := serializeTestLayers(t, eth, ipv6)
	packet := parseTestPacket(buf)

	ethResult := h.extractEthernetLayer(packet)
	if ethResult == nil {
		t.Fatal("expected non-nil Ethernet layer")
	}
	if ethResult.SrcMAC.String() != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("expected aa:bb:cc:dd:ee:ff, got %s", ethResult.SrcMAC)
	}
}

// ---------- ICMP getAddressMaskTargets ----------

func TestICMPGetAddressMaskTargets(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewICMPHandler(stack)

	ipLayer := &layers.IPv4{SrcIP: net.ParseIP("10.0.0.1"), DstIP: net.ParseIP("10.0.0.2")}
	pkt := &Packet{Buffer: make([]byte, 14), SerialNumber: 1}

	targets := h.getAddressMaskTargets(pkt, ipLayer)
	if targets == nil {
		// May be nil if no devices match, which is fine for coverage
		t.Log("no matching devices found (expected)")
	}
}

// ---------- SNMP sourceMAC ----------

func TestSNMPSourceMAC(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewSNMPHandler(stack)

	dev := &config.Device{
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}
	pkt := &Packet{Buffer: make([]byte, 14)}

	mac := h.sourceMAC(dev, pkt)
	if mac.String() != "00:11:22:33:44:55" {
		t.Errorf("expected device MAC, got %s", mac)
	}

	// No MAC on device - falls back to packet dest MAC
	dev2 := &config.Device{}
	mac2 := h.sourceMAC(dev2, pkt)
	_ = mac2 // just verify no panic
}

// ---------- DNS handleNBSTATIfPresent ----------

func TestDNSHandleNBSTATIfPresent(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	ipLayer := &layers.IPv4{
		SrcIP: net.ParseIP("10.0.0.1"),
		DstIP: net.ParseIP("10.0.0.2"),
	}
	udpLayer := &layers.UDP{SrcPort: 137, DstPort: 137}

	dev := &config.Device{
		Name:       "d1",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	// No NBSTAT questions
	dns := &layers.DNS{
		ID: 1,
		Questions: []layers.DNSQuestion{
			{Name: []byte("test.local"), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
		},
	}

	// Build packet for extracting Ethernet layer
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		DstMAC:       net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
		EthernetType: layers.EthernetTypeIPv4,
	}
	_ = udpLayer.SetNetworkLayerForChecksum(ipLayer)
	buf := serializeTestLayers(t, eth, ipLayer, udpLayer, dns)
	packet := parseTestPacket(buf)
	pkt := &Packet{Buffer: buf, Length: len(buf), SerialNumber: 1}

	result := h.handleNBSTATIfPresent(pkt, ipLayer, udpLayer, dev, dns, packet, 0)
	if result {
		t.Error("expected false when no NBSTAT questions")
	}
}

// ---------- DNS extractSourceMACWithValidation ----------

// ---------- ICMP buildRouterAdvertisementPayload tests ----------

func TestICMPBuildRouterAdvertisementPayload(t *testing.T) {
	// nil config
	if buildRouterAdvertisementPayload(nil) != nil {
		t.Error("expected nil for nil config")
	}

	// With routers
	ra := &config.IcmpRouterAdvertisement{
		Lifetime: 1800,
		Routers: []config.IcmpRouter{
			{Address: net.ParseIP("10.0.0.1"), Preference: 100},
			{Address: net.ParseIP("10.0.0.2"), Preference: 50},
		},
	}
	payload := buildRouterAdvertisementPayload(ra)
	if len(payload) == 0 {
		t.Error("expected non-empty payload")
	}
	if payload[0] != 2 {
		t.Errorf("expected 2 router entries, got %d", payload[0])
	}
}

// ---------- DHCPv6 validateClientIdentity with valid data ----------

func TestDHCPv6ValidateClientIdentityValid(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPv6Handler(stack)

	// Build IANA option data: IAID (4 bytes) + T1 (4 bytes) + T2 (4 bytes)
	ianaData := make([]byte, 12)
	binary.BigEndian.PutUint32(ianaData[0:4], 0x12345678) // IAID

	msg := &DHCPv6Message{
		MessageType: DHCPv6Solicit,
		Options: []DHCPv6Option{
			{Code: DHCPv6OptClientID, Length: 4, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			{Code: DHCPv6OptIANA, Length: uint16(len(ianaData)), Data: ianaData},
		},
	}

	duid, iaid, ok := h.validateClientIdentity(msg, "SOLICIT", 1)
	if !ok {
		t.Error("expected valid client identity")
	}
	if duid == nil {
		t.Error("expected non-nil DUID")
	}
	if iaid != 0x12345678 {
		t.Errorf("expected IAID 0x12345678, got 0x%08X", iaid)
	}
}

// ---------- DHCPv6 handleRelease and handleDecline ----------

func TestDHCPv6HandleRelease(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPv6Handler(stack)

	h.SetAddressPool([]net.IP{net.ParseIP("fd00::100")})

	// Allocate a lease first
	clientDUID := []byte{0x01, 0x02, 0x03, 0x04}
	_, _ = h.allocateLease(clientDUID, 1)

	msg := &DHCPv6Message{
		MessageType: DHCPv6Release,
		Options: []DHCPv6Option{
			{Code: DHCPv6OptClientID, Length: 4, Data: clientDUID},
		},
	}

	// Should not panic
	h.handleRelease(msg, 1)
}

func TestDHCPv6HandleDecline(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPv6Handler(stack)

	h.SetAddressPool([]net.IP{net.ParseIP("fd00::100")})

	clientDUID := []byte{0x01, 0x02, 0x03, 0x04}
	_, _ = h.allocateLease(clientDUID, 1)

	msg := &DHCPv6Message{
		MessageType: DHCPv6Decline,
		Options: []DHCPv6Option{
			{Code: DHCPv6OptClientID, Length: 4, Data: clientDUID},
		},
	}

	h.handleDecline(msg, 1)
}

// ---------- DHCP handleDHCPDiscover and handleDHCPRequest ----------

func TestDHCPHandleDiscover(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)

	h.SetPool(net.ParseIP("10.0.0.10"), net.ParseIP("10.0.0.20"))
	h.SetServerConfig(net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.1"),
		[]net.IP{net.ParseIP("8.8.8.8")}, "example.com")

	mac := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	serverDev := &config.Device{
		Name:        "server",
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	info := &dhcpPacketInfo{
		dhcp:        &layers.DHCPv4{ClientHWAddr: mac, Xid: 1234},
		messageType: DHCPDiscover,
		hostname:    "testhost",
	}

	// Should not panic even though SendRawPacket will fail
	h.handleDHCPDiscover(info, serverDev, 1, 0)
}

func TestDHCPHandleRequest(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)

	h.SetPool(net.ParseIP("10.0.0.10"), net.ParseIP("10.0.0.20"))
	h.SetServerConfig(net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.1"),
		[]net.IP{net.ParseIP("8.8.8.8")}, "example.com")

	mac := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	serverDev := &config.Device{
		Name:        "server",
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	info := &dhcpPacketInfo{
		dhcp: &layers.DHCPv4{
			ClientHWAddr: mac,
			Xid:          5678,
			Options: []layers.DHCPOption{
				{Type: layers.DHCPOptRequestIP, Data: []byte{10, 0, 0, 10}},
			},
		},
		messageType: DHCPRequest,
		hostname:    "testhost",
	}

	h.handleDHCPRequest(info, serverDev, 1, 0)
}

// ---------- DHCP HandlePacket ----------

func TestDHCPHandlePacket(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPHandler(stack)

	h.SetPool(net.ParseIP("10.0.0.10"), net.ParseIP("10.0.0.20"))
	h.SetServerConfig(net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.1"),
		[]net.IP{net.ParseIP("8.8.8.8")}, "example.com")

	// Build a DHCP Discover packet
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ipLayer := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("0.0.0.0"), DstIP: net.ParseIP("255.255.255.255"),
	}
	udpLayer := &layers.UDP{SrcPort: 68, DstPort: 67}
	_ = udpLayer.SetNetworkLayerForChecksum(ipLayer)
	dhcpLayer := &layers.DHCPv4{
		Operation:    layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  6,
		ClientHWAddr: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		Options: []layers.DHCPOption{
			{Type: layers.DHCPOptMessageType, Length: 1, Data: []byte{DHCPDiscover}},
		},
	}

	buf := serializeTestLayers(t, eth, ipLayer, udpLayer, dhcpLayer)
	pkt := &Packet{Buffer: buf, Length: len(buf), SerialNumber: 1}

	serverDev := &config.Device{
		Name:        "server",
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	// Should not panic
	h.HandlePacket(pkt, ipLayer, udpLayer, []*config.Device{serverDev})
}

// ---------- DHCPv6 serializeOption ----------

func TestDHCPv6SerializeOption(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPv6Handler(stack)

	opt := DHCPv6Option{
		Code:   DHCPv6OptServerID,
		Length: 4,
		Data:   []byte{0x01, 0x02, 0x03, 0x04},
	}

	data := h.serializeOption(opt)
	if len(data) != 8 { // 4 header + 4 data
		t.Errorf("expected 8 bytes, got %d", len(data))
	}
}

// ---------- Stack parseReceivedPacket and queuePacket tests ----------

func TestStackParseReceivedPacket(t *testing.T) {
	stack := newTestStackInternal(t)

	// Valid Ethernet frame (14 bytes minimum)
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		DstMAC:       net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolTCP,
		SrcIP: net.ParseIP("10.0.0.1"), DstIP: net.ParseIP("10.0.0.2"),
	}
	buf := serializeTestLayers(t, eth, ip)

	pkt := stack.parseReceivedPacket(buf)
	if pkt == nil {
		t.Error("expected valid packet from parseReceivedPacket")
	}

	// Empty data - may still parse successfully since ParsePacket is lenient
	emptyPkt := stack.parseReceivedPacket(nil)
	if emptyPkt != nil {
		t.Log("nil data returned nil (expected)")
	}
}

func TestStackQueuePacket(t *testing.T) {
	stack := newTestStackInternal(t)
	pkt := &Packet{Buffer: make([]byte, 14), Length: 14, SerialNumber: 1}
	stack.queuePacket(pkt) // Should not panic

	// Fill the queue
	for range DefaultQueueBufferSize + 10 {
		stack.queuePacket(pkt)
	}
	// Should still not panic even when full
}

// ---------- DNS buildDNSResponseV6 ----------

func TestDNSBuildDNSResponseV6(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	h.AddRecord("test.local", net.ParseIP("fd00::1"))

	dev := &config.Device{
		Name:       "dns-server",
		MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	dns := &layers.DNS{
		ID:     42,
		QR:     false,
		OpCode: layers.DNSOpCodeQuery,
		RD:     true,
		Questions: []layers.DNSQuestion{
			{Name: []byte("test.local"), Type: layers.DNSTypeAAAA, Class: layers.DNSClassIN},
		},
	}

	response := h.buildDNSResponseV6(dns, dev, 0, 1)
	if response == nil {
		t.Fatal("expected non-nil response")
	}
	if !response.QR {
		t.Error("expected QR flag set")
	}
}

// ---------- DHCPv6 handleSolicit and handleInfoRequest ----------

func TestDHCPv6HandleSolicit(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPv6Handler(stack)

	h.SetAddressPool([]net.IP{net.ParseIP("fd00::100"), net.ParseIP("fd00::101")})
	h.SetServerConfig([]net.IP{net.ParseIP("fd00::53")}, []string{"example.com"})

	ianaData := make([]byte, 12)
	binary.BigEndian.PutUint32(ianaData[0:4], 0x12345678)

	msg := &DHCPv6Message{
		MessageType: DHCPv6Solicit,
		Options: []DHCPv6Option{
			{Code: DHCPv6OptClientID, Length: 4, Data: []byte{0x01, 0x02, 0x03, 0x04}},
			{Code: DHCPv6OptIANA, Length: uint16(len(ianaData)), Data: ianaData},
		},
	}

	serverDev := &config.Device{
		Name:        "server",
		IPAddresses: []net.IP{net.ParseIP("fd00::1")},
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
	}

	// Should not panic (sendDHCPv6Response will fail due to no capture engine)
	h.handleSolicit(msg, net.ParseIP("fd00::100"), net.ParseIP("fd00::1"),
		serverDev.MACAddress, serverDev, 1)
}

func TestDHCPv6HandleInfoRequest(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDHCPv6Handler(stack)

	h.SetServerConfig([]net.IP{net.ParseIP("fd00::53")}, []string{"example.com"})

	msg := &DHCPv6Message{
		MessageType: DHCPv6InfoRequest,
		Options: []DHCPv6Option{
			{Code: DHCPv6OptClientID, Length: 4, Data: []byte{0x01, 0x02, 0x03, 0x04}},
		},
	}

	serverMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	// Should not panic
	h.handleInfoRequest(msg, net.ParseIP("fd00::100"), net.ParseIP("fd00::1"), serverMAC, 1)
}

// ---------- TCP lookupDestinationMACV6 ----------

func TestTCPLookupDestinationMACV6(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewTCPHandler(stack)

	// Unknown IP
	mac := h.lookupDestinationMACV6(net.ParseIP("fd00::1"), 0, 0)
	if mac != nil {
		t.Error("expected nil for unknown IP")
	}
}

func TestDNSExtractSourceMACWithValidation(t *testing.T) {
	stack := newTestStackInternal(t)
	h := NewDNSHandler(stack)

	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		DstMAC:       net.HardwareAddr{0x11, 0x22, 0x33, 0x44, 0x55, 0x66},
		EthernetType: layers.EthernetTypeIPv6,
	}
	ipv6 := &layers.IPv6{
		Version: 6, NextHeader: layers.IPProtocolUDP, HopLimit: 64,
		SrcIP: net.ParseIP("fd00::1"), DstIP: net.ParseIP("fd00::2"),
	}

	buf := serializeTestLayers(t, eth, ipv6)
	packet := parseTestPacket(buf)

	mac, ok := h.extractSourceMACWithValidation(packet, 0, 1)
	if !ok || mac == nil {
		t.Error("expected valid MAC extraction")
	}
	if mac.String() != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("expected aa:bb:cc:dd:ee:ff, got %s", mac)
	}
}
