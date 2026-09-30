package protocols

import (
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func raRouter(name, mac string, period int, ips ...string) config.Device {
	device := config.Device{
		Name: name, Type: "router", MACAddress: mustMAC(mac),
		ICMPv6Config: &config.ICMPv6Config{
			Enabled:             true,
			RouterAdvertisement: &config.Icmpv6RouterAdvertisement{Period: period},
		},
	}
	for _, ip := range ips {
		device.IPAddresses = append(device.IPAddresses, net.ParseIP(ip))
	}
	return device
}

// A passive listener that never solicits learns about a router only from its
// unsolicited advertisements, so each authored period must reach the wire.
func TestRouterAdvertisementOriginationFollowsPeriod(t *testing.T) {
	devices := []config.Device{
		raRouter("fast", "02:00:00:00:00:01", 4, "fd00:1::1"),
		raRouter("default", "02:00:00:00:00:02", 0, "fd00:2::1"),
		raRouter("v4only", "02:00:00:00:00:03", 4, "10.0.0.3"),
		{
			Name: "unauthored", Type: "router", MACAddress: mustMAC("02:00:00:00:00:04"),
			IPAddresses: []net.IP{net.ParseIP("fd00:4::1")},
		},
	}
	stack := NewStack(nil, &config.Config{Devices: devices}, logging.NewDebugConfig(0))
	names := make(map[string]string, len(devices))
	for _, device := range devices {
		names[device.MACAddress.String()] = device.Name
	}

	start := time.Now()
	var due map[*config.Device]time.Time
	for _, step := range []struct {
		after time.Duration
		want  []string
	}{
		{0, []string{"fast", "default"}},
		{3 * time.Second, nil},
		// Late by a few milliseconds, as a ticker is: the next is still due at +8 s.
		{4*time.Second + 10*time.Millisecond, []string{"fast"}},
		{7 * time.Second, nil},
		{8 * time.Second, []string{"fast"}},
		{defaultRAPeriod - time.Second, []string{"fast"}},
		{defaultRAPeriod, []string{"default"}},
	} {
		due = stack.icmpv6Handler.sendDueRouterAdvertisements(start.Add(step.after), due)
		var got []string
		for len(stack.sendQueue) > 0 {
			pkt := <-stack.sendQueue
			got = append(got, names[net.HardwareAddr(pkt.Buffer[6:12]).String()])
		}
		if !sameNames(got, step.want) {
			t.Errorf("at +%v sent %v, want %v", step.after, got, step.want)
		}
	}
}

// A host discards a router advertisement unless it comes from a link-local
// source with hop limit 255 (RFC 4861 section 6.1.2); an unsolicited one goes to
// all-nodes. The prefix still comes from the router's global address.
func TestUnsolicitedRouterAdvertisementIsAcceptedByHosts(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{raRouter("gw", "02:00:00:00:6a:01", 4, "fd00:6a::1")}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))

	stack.icmpv6Handler.sendDueRouterAdvertisements(time.Now(), nil)
	if len(stack.sendQueue) != 1 {
		t.Fatalf("queued %d advertisements, want 1", len(stack.sendQueue))
	}
	pkt := <-stack.sendQueue
	frame := gopacket.NewPacket(pkt.Buffer[:pkt.Length], layers.LayerTypeEthernet, gopacket.Default)

	eth, _ := frame.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	ipv6, _ := frame.Layer(layers.LayerTypeIPv6).(*layers.IPv6)
	ra, _ := frame.Layer(layers.LayerTypeICMPv6RouterAdvertisement).(*layers.ICMPv6RouterAdvertisement)
	if eth == nil || ipv6 == nil || ra == nil {
		t.Fatalf("not a router advertisement: %v", frame)
	}
	if eth.DstMAC.String() != allNodesMAC || !ipv6.DstIP.Equal(net.ParseIP("ff02::1")) {
		t.Errorf("sent to %s / %s, want all-nodes", eth.DstMAC, ipv6.DstIP)
	}
	if want := net.ParseIP("fe80::ff:fe00:6a01"); !ipv6.SrcIP.Equal(want) {
		t.Errorf("source = %s, want the EUI-64 link-local %s", ipv6.SrcIP, want)
	}
	if ipv6.HopLimit != icmpv6NDPHopLimit {
		t.Errorf("hop limit = %d, want %d", ipv6.HopLimit, icmpv6NDPHopLimit)
	}
	var prefix net.IP
	for _, option := range ra.Options {
		if option.Type == layers.ICMPv6OptPrefixInfo && len(option.Data) >= 30 {
			prefix = net.IP(option.Data[14:30])
		}
	}
	if !prefix.Equal(net.ParseIP("fd00:6a::")) {
		t.Errorf("advertised prefix = %s, want fd00:6a:: from the global address", prefix)
	}
}

func TestLinkLocalAddress(t *testing.T) {
	tests := []struct {
		name   string
		device config.Device
		want   net.IP
	}{
		{
			name:   "derived from the MAC with the universal/local bit flipped",
			device: config.Device{MACAddress: mustMAC("00:1b:54:aa:bb:cc")},
			want:   net.ParseIP("fe80::21b:54ff:feaa:bbcc"),
		},
		{
			name: "an authored link-local wins over the derived one",
			device: config.Device{
				MACAddress:  mustMAC("00:1b:54:aa:bb:cc"),
				IPAddresses: []net.IP{net.ParseIP("fd00::1"), net.ParseIP("fe80::1")},
			},
			want: net.ParseIP("fe80::1"),
		},
		{name: "no MAC, no address", device: config.Device{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := linkLocalAddress(&test.device); !got.Equal(test.want) {
				t.Errorf("linkLocalAddress = %s, want %s", got, test.want)
			}
		})
	}
}

// A host that learned the router from an advertisement resolves its link-local
// source before sending to it; an unanswered solicitation leaves the default
// route unusable.
func TestNeighborSolicitationForDerivedLinkLocal(t *testing.T) {
	tests := []struct {
		name     string
		ips      []string
		target   string
		answered bool
	}{
		{
			name: "an IPv6 device answers for its derived address", ips: []string{"fd00:6a::21"},
			target: "fe80::ff:fe00:6a01", answered: true,
		},
		{
			name: "an IPv4-only device does not speak IPv6", ips: []string{"10.66.200.21"},
			target: "fe80::ff:fe00:6a01",
		},
		{
			name: "an authored link-local replaces the derived one", ips: []string{"fd00:6a::21", "fe80::6a:21"},
			target: "fe80::ff:fe00:6a01",
		},
		{
			name: "another MAC's derived address is not ours", ips: []string{"fd00:6a::21"},
			target: "fe80::ff:fe00:6a02",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			device := config.Device{
				Name: "v6-probe-sw01", Type: "switch",
				MACAddress: net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x6a, 0x01},
			}
			for _, ip := range test.ips {
				device.IPAddresses = append(device.IPAddresses, net.ParseIP(ip))
			}
			stack := NewStack(nil, &config.Config{
				Segments: []config.Segment{{Tag: ndpTestVLAN, Devices: []config.Device{device}}},
			}, logging.NewDebugConfig(0))

			solicit(t, stack, &device, net.ParseIP(test.target))

			if answered := len(stack.sendQueue) > 0; answered != test.answered {
				t.Errorf("answered = %v, want %v", answered, test.answered)
			}
		})
	}
}
