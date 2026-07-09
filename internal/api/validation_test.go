package api

import (
	"net"
	"testing"
)

func TestNormalizeAndParseIP(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantIP  string
	}{
		{name: "valid IPv4", input: "192.168.1.1", wantNil: false, wantIP: "192.168.1.1"},
		{name: "valid IPv6", input: "::1", wantNil: false, wantIP: "::1"},
		{name: "IPv6 with zone", input: "::1%eth0", wantNil: false, wantIP: "::1"},
		{name: "decimal IPv4", input: "2130706433", wantNil: false, wantIP: "127.0.0.1"},
		{name: "invalid IP", input: "not-an-ip", wantNil: true},
		{name: "empty string", input: "", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAndParseIP(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("normalizeAndParseIP(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Errorf("normalizeAndParseIP(%q) = nil, want %s", tt.input, tt.wantIP)
				return
			}
			// For decimal IP, check the result
			if tt.input == "2130706433" {
				if got.String() != tt.wantIP {
					t.Errorf("normalizeAndParseIP(%q) = %s, want %s", tt.input, got.String(), tt.wantIP)
				}
			}
		})
	}
}

func TestIsIPv6MappedLoopback(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		want bool
	}{
		{
			name: "IPv4 loopback parsed as 16-byte returns true",
			ip:   net.ParseIP("127.0.0.1"), // ParseIP returns 16-byte representation
			want: true,
		},
		{
			name: "IPv6 loopback",
			ip:   net.ParseIP("::1"),
			want: false,
		},
		{
			name: "IPv6-mapped IPv4 loopback",
			ip:   net.ParseIP("::ffff:127.0.0.1"),
			want: true,
		},
		{
			name: "IPv6-mapped public IP",
			ip:   net.ParseIP("::ffff:8.8.8.8"),
			want: false,
		},
		{
			name: "regular IPv6",
			ip:   net.ParseIP("2001:db8::1"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIPv6MappedLoopback(tt.ip)
			if got != tt.want {
				t.Errorf("isIPv6MappedLoopback(%v) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestIsBlockedHostname(t *testing.T) {
	tests := []struct {
		name string
		host string
		want bool
	}{
		{name: "localhost blocked", host: "localhost", want: true},
		{name: "LOCALHOST blocked", host: "LOCALHOST", want: true},
		{name: "127.0.0.1 blocked", host: "127.0.0.1", want: true},
		{name: "127.x.x.x blocked", host: "127.0.0.2", want: true},
		{name: "::1 blocked", host: "::1", want: true},
		{name: "[::1] blocked", host: "[::1]", want: true},
		{name: "0.0.0.0 blocked", host: "0.0.0.0", want: true},
		{name: "metadata service blocked", host: "169.254.169.254", want: true},
		{name: "google metadata blocked", host: "metadata.google.internal", want: true},
		{name: "public hostname allowed", host: "example.com", want: false},
		{name: "public IP allowed", host: "8.8.8.8", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBlockedHostname(tt.host)
			if got != tt.want {
				t.Errorf("isBlockedHostname(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{name: "loopback blocked", ip: "127.0.0.1", wantErr: true},
		{name: "private 10.x blocked", ip: "10.0.0.1", wantErr: true},
		{name: "private 172.16.x blocked", ip: "172.16.0.1", wantErr: true},
		{name: "private 192.168.x blocked", ip: "192.168.1.1", wantErr: true},
		{name: "link-local blocked", ip: "169.254.1.1", wantErr: true},
		{name: "unspecified blocked", ip: "0.0.0.0", wantErr: true},
		{name: "public IP allowed", ip: "8.8.8.8", wantErr: false},
		{name: "public IP 2 allowed", ip: "93.184.216.34", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			err := isBlockedIP(ip)
			if tt.wantErr && err == nil {
				t.Errorf("isBlockedIP(%q) = nil, want error", tt.ip)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("isBlockedIP(%q) = %v, want nil", tt.ip, err)
			}
		})
	}
}

func TestIsBlockedIPv6Mapped(t *testing.T) {
	tests := []struct {
		name    string
		ip      net.IP
		wantErr bool
	}{
		{
			name:    "private IPv4 parsed as 16-byte blocked",
			ip:      net.ParseIP("192.168.1.1"), // ParseIP returns 16-byte representation
			wantErr: true,
		},
		{
			name:    "IPv6 returns nil",
			ip:      net.ParseIP("2001:db8::1"),
			wantErr: false,
		},
		{
			name:    "IPv6-mapped private blocked",
			ip:      net.ParseIP("::ffff:192.168.1.1"),
			wantErr: true,
		},
		{
			name:    "IPv6-mapped loopback blocked",
			ip:      net.ParseIP("::ffff:127.0.0.1"),
			wantErr: true,
		},
		{
			name:    "IPv6-mapped public allowed",
			ip:      net.ParseIP("::ffff:8.8.8.8"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isBlockedIPv6Mapped(tt.ip)
			if tt.wantErr && err == nil {
				t.Errorf("isBlockedIPv6Mapped(%v) = nil, want error", tt.ip)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("isBlockedIPv6Mapped(%v) = %v, want nil", tt.ip, err)
			}
		})
	}
}

func TestValidateWebhookURLSSRF(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "valid public URL", url: "https://example.com/webhook", wantErr: false},
		{name: "localhost blocked", url: "http://localhost/webhook", wantErr: true},
		{name: "127.0.0.1 blocked", url: "http://127.0.0.1/webhook", wantErr: true},
		{name: "private IP blocked", url: "http://192.168.1.1/webhook", wantErr: true},
		{name: "metadata service blocked", url: "http://169.254.169.254/latest/meta-data/", wantErr: true},
		{name: "invalid URL format", url: "not a url", wantErr: true},
		{name: "no hostname", url: "http:///path", wantErr: true},
		{name: "public IP allowed", url: "https://93.184.216.34/webhook", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURLSSRF(tt.url)
			if tt.wantErr && err == nil {
				t.Errorf("validateWebhookURLSSRF(%q) = nil, want error", tt.url)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateWebhookURLSSRF(%q) = %v, want nil", tt.url, err)
			}
		})
	}
}

func TestIsValidInterfaceChar(t *testing.T) {
	tests := []struct {
		char rune
		want bool
	}{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'-', true},
		{'_', true},
		{'.', true},
		{'/', false},
		{'\\', false},
		{'@', false},
		{' ', false},
		{'!', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			got := isValidInterfaceChar(tt.char)
			if got != tt.want {
				t.Errorf("isValidInterfaceChar(%q) = %v, want %v", tt.char, got, tt.want)
			}
		})
	}
}

func TestValidateInterfaceName(t *testing.T) {
	tests := []struct {
		name    string
		iface   string
		wantErr bool
	}{
		{name: "valid eth0", iface: "eth0", wantErr: false},
		{name: "valid lo", iface: "lo", wantErr: false},
		{name: "valid wlan0", iface: "wlan0", wantErr: false},
		{name: "valid with dash", iface: "eth-0", wantErr: false},
		{name: "valid with underscore", iface: "eth_0", wantErr: false},
		{name: "valid with dot", iface: "eth0.100", wantErr: false},
		{name: "empty string", iface: "", wantErr: true},
		{name: "starts with dash", iface: "-eth0", wantErr: true},
		{name: "starts with dot", iface: ".eth0", wantErr: true},
		{name: "starts with number", iface: "0eth", wantErr: true},
		{name: "path traversal", iface: "../etc/passwd", wantErr: true},
		{name: "contains ..", iface: "eth..0", wantErr: true},
		{name: "too long", iface: "abcdefghijklmnop", wantErr: true}, // 16 chars > 15
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateInterfaceName(tt.iface)
			if tt.wantErr && got == nil {
				t.Errorf("validateInterfaceName(%q) = nil, want error", tt.iface)
			}
			if !tt.wantErr && got != nil {
				t.Errorf("validateInterfaceName(%q) = %v, want nil", tt.iface, got)
			}
		})
	}
}

func TestValidateAlertConfig(t *testing.T) {
	tests := []struct {
		name     string
		cfg      AlertConfig
		wantErrs int
	}{
		{
			name:     "valid config",
			cfg:      AlertConfig{PacketsThreshold: 1000, WebhookURL: "https://example.com/hook"},
			wantErrs: 0,
		},
		{
			name:     "empty config is valid",
			cfg:      AlertConfig{},
			wantErrs: 0,
		},
		{
			name:     "threshold too high",
			cfg:      AlertConfig{PacketsThreshold: 2000000000},
			wantErrs: 1,
		},
		{
			name:     "invalid webhook URL protocol",
			cfg:      AlertConfig{WebhookURL: "ftp://example.com/hook"},
			wantErrs: 1,
		},
		{
			name:     "localhost webhook blocked",
			cfg:      AlertConfig{WebhookURL: "http://localhost/hook"},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateAlertConfig(tt.cfg)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateAlertConfig() returned %d errors, want %d", len(errs), tt.wantErrs)
				for _, e := range errs {
					t.Logf("  error: field=%s issue=%s", e.Field, e.Issue)
				}
			}
		})
	}
}

func TestValidateReplayRequest_Rate(t *testing.T) {
	tests := []struct {
		name     string
		req      ReplayRequest
		wantErrs int
	}{
		{name: "timing default", req: ReplayRequest{}, wantErrs: 0},
		{name: "topspeed", req: ReplayRequest{RateMode: "topspeed"}, wantErrs: 0},
		{name: "pps valid", req: ReplayRequest{RateMode: "pps", Pps: 1000}, wantErrs: 0},
		{name: "pps missing rate", req: ReplayRequest{RateMode: "pps"}, wantErrs: 1},
		{name: "pps negative", req: ReplayRequest{RateMode: "pps", Pps: -5}, wantErrs: 1},
		{name: "pps too high", req: ReplayRequest{RateMode: "pps", Pps: maxReplayPPS + 1}, wantErrs: 1},
		{name: "mbps valid", req: ReplayRequest{RateMode: "mbps", MbpsCap: 100}, wantErrs: 0},
		{name: "mbps missing cap", req: ReplayRequest{RateMode: "mbps"}, wantErrs: 1},
		{name: "mbps too high", req: ReplayRequest{RateMode: "mbps", MbpsCap: maxReplayMbps + 1}, wantErrs: 1},
		{name: "unknown mode", req: ReplayRequest{RateMode: "warpspeed"}, wantErrs: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateReplayRequest(tt.req)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateReplayRequest() returned %d errors, want %d", len(errs), tt.wantErrs)
				for _, e := range errs {
					t.Logf("  error: field=%s issue=%s", e.Field, e.Issue)
				}
			}
		})
	}
}

func TestValidateReplayRequest_LoopCount(t *testing.T) {
	tests := []struct {
		name     string
		req      ReplayRequest
		wantErrs int
	}{
		{name: "default zero", req: ReplayRequest{}, wantErrs: 0},
		{name: "positive count", req: ReplayRequest{LoopCount: 5}, wantErrs: 0},
		{name: "negative count", req: ReplayRequest{LoopCount: -1}, wantErrs: 1},
		{name: "over max", req: ReplayRequest{LoopCount: maxReplayLoopCount + 1}, wantErrs: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateReplayRequest(tt.req)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateReplayRequest() returned %d errors, want %d", len(errs), tt.wantErrs)
				for _, e := range errs {
					t.Logf("  error: field=%s issue=%s", e.Field, e.Issue)
				}
			}
		})
	}
}
