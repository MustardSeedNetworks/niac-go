package api

import (
	"net/http/httptest"
	"testing"
)

func TestParseValidIP(t *testing.T) {
	tests := []struct {
		name  string
		rawIP string
		want  string
	}{
		{name: "valid IPv4", rawIP: "192.168.1.1", want: "192.168.1.1"},
		{name: "valid IPv4 with spaces", rawIP: "  192.168.1.1  ", want: "192.168.1.1"},
		{name: "valid IPv6", rawIP: "::1", want: "::1"},
		{name: "valid IPv6 full", rawIP: "2001:db8::1", want: "2001:db8::1"},
		{name: "invalid IP", rawIP: "not-an-ip", want: ""},
		{name: "empty string", rawIP: "", want: ""},
		{name: "partial IP", rawIP: "192.168", want: ""},
		{name: "IP with port", rawIP: "192.168.1.1:8080", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseValidIP(tt.rawIP)
			if got != tt.want {
				t.Errorf("parseValidIP(%q) = %q, want %q", tt.rawIP, got, tt.want)
			}
		})
	}
}

func TestExtractXForwardedForIP(t *testing.T) {
	tests := []struct {
		name       string
		xForwarded string
		want       string
	}{
		{name: "single IP", xForwarded: "192.168.1.1", want: "192.168.1.1"},
		{name: "multiple IPs", xForwarded: "192.168.1.1, 10.0.0.1, 172.16.0.1", want: "192.168.1.1"},
		{name: "with spaces", xForwarded: "  192.168.1.1  ", want: "192.168.1.1"},
		{name: "invalid first IP", xForwarded: "invalid, 192.168.1.1", want: ""},
		{name: "empty header", xForwarded: "", want: ""},
		{name: "only commas", xForwarded: ", ,", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.xForwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwarded)
			}

			got := extractXForwardedForIP(req)
			if got != tt.want {
				t.Errorf("extractXForwardedForIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractXRealIP(t *testing.T) {
	tests := []struct {
		name    string
		xRealIP string
		want    string
	}{
		{name: "valid IP", xRealIP: "192.168.1.1", want: "192.168.1.1"},
		{name: "with spaces", xRealIP: "  192.168.1.1  ", want: "192.168.1.1"},
		{name: "invalid IP", xRealIP: "not-an-ip", want: ""},
		{name: "empty header", xRealIP: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			got := extractXRealIP(req)
			if got != tt.want {
				t.Errorf("extractXRealIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsTrustedProxy(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "localhost IPv4", ip: "127.0.0.1", want: true},
		{name: "localhost IPv6", ip: "::1", want: true},
		{name: "private 10.x.x.x", ip: "10.0.0.1", want: true},
		{name: "private 172.16.x.x", ip: "172.16.0.1", want: true},
		{name: "private 192.168.x.x", ip: "192.168.1.1", want: true},
		{name: "public IP", ip: "8.8.8.8", want: false},
		{name: "invalid IP", ip: "not-an-ip", want: false},
		{name: "empty string", ip: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTrustedProxy(tt.ip)
			if got != tt.want {
				t.Errorf("isTrustedProxy(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestGetClientIPDirect(t *testing.T) {
	// Test when request comes from public IP (headers not trusted)
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "8.8.8.8:1234"
	req.Header.Set("X-Forwarded-For", "192.168.1.100")

	got := getClientIP(req)
	// Should return RemoteAddr IP, not X-Forwarded-For (public IP is not trusted proxy)
	if got != "8.8.8.8" {
		t.Errorf("getClientIP() = %q, want %q", got, "8.8.8.8")
	}
}

func TestGetClientIPFromTrustedProxy(t *testing.T) {
	// Test when request comes from trusted private proxy
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")

	got := getClientIP(req)
	// Should use X-Forwarded-For since RemoteAddr is trusted proxy
	if got != "203.0.113.50" {
		t.Errorf("getClientIP() = %q, want %q", got, "203.0.113.50")
	}
}

func TestGetClientIPXRealIPFromTrustedProxy(t *testing.T) {
	// Test X-Real-IP from trusted proxy
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Real-IP", "203.0.113.100")

	got := getClientIP(req)
	// Should use X-Real-IP since RemoteAddr is localhost
	if got != "203.0.113.100" {
		t.Errorf("getClientIP() = %q, want %q", got, "203.0.113.100")
	}
}

func TestGetClientIPNoPort(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1" // No port

	got := getClientIP(req)
	if got != "192.168.1.1" {
		t.Errorf("getClientIP() = %q, want %q", got, "192.168.1.1")
	}
}

func TestGenerateRequestID(t *testing.T) {
	// Test that request ID is generated
	id1 := generateRequestID()
	if id1 == "" {
		t.Error("generateRequestID() returned empty string")
	}

	// Should be hex encoded (apply De Morgan's law for clarity)
	for _, c := range id1 {
		isDigit := c >= '0' && c <= '9'
		isHexLetter := c >= 'a' && c <= 'f'
		if !isDigit && !isHexLetter {
			t.Errorf("generateRequestID() contains non-hex character: %c", c)
		}
	}

	// Each call should return unique ID
	id2 := generateRequestID()
	if id1 == id2 {
		t.Error("generateRequestID() returned same ID twice")
	}
}
