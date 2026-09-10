package config_test

import (
	"net"
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestDeviceInterfaceAddress(t *testing.T) {
	device := config.Device{
		Interfaces: []config.Interface{{Address: " 192.0.2.30/24 "}, {Address: "invalid"}},
		IPAddresses: []net.IP{
			net.ParseIP("192.0.2.20"), net.ParseIP("192.0.2.21"),
			net.ParseIP("2001:db8::1"), nil,
		},
	}
	for _, tc := range []struct {
		index int
		want  string
	}{
		{0, "192.0.2.30/24"},
		{1, "192.0.2.21/32"},
		{2, "2001:db8::1/128"},
		{3, "invalid Prefix"},
		{4, "invalid Prefix"},
		{-1, "invalid Prefix"},
	} {
		if got := config.DeviceInterfaceAddress(device, tc.index).String(); got != tc.want {
			t.Errorf("index %d = %s, want %s", tc.index, got, tc.want)
		}
	}
}

func TestValidIPv4Host(t *testing.T) {
	for _, tc := range []struct {
		prefix, address string
		want            bool
	}{
		{"192.0.2.20/24", "192.0.2.20", true},
		{"192.0.2.20/24", "192.0.2.0", false},
		{"192.0.2.20/24", "192.0.2.255", false},
		{"192.0.2.20/24", "198.51.100.20", false},
		{"192.0.2.20/31", "192.0.2.20", true},
		{"192.0.2.20/31", "192.0.2.21", true},
		{"192.0.2.20/32", "192.0.2.20", true},
		{"192.0.2.20/24", "2001:db8::1", false},
		{"2001:db8::/64", "2001:db8::1", false},
	} {
		prefix, address := netip.MustParsePrefix(tc.prefix), netip.MustParseAddr(tc.address)
		if got := config.ValidIPv4Host(prefix, address); got != tc.want {
			t.Errorf("ValidIPv4Host(%s, %s) = %v, want %v", tc.prefix, tc.address, got, tc.want)
		}
	}
	if config.ValidIPv4Host(netip.Prefix{}, netip.Addr{}) {
		t.Fatal("invalid prefix/address accepted")
	}
}
