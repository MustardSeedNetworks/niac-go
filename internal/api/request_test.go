package api

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"golang.org/x/time/rate"

	"github.com/MustardSeedNetworks/niac-go/internal/api/ratelimit"
)

func TestParseTrustedProxies(t *testing.T) {
	tests := []struct {
		name    string
		list    string
		want    []string
		wantErr bool
	}{
		{name: "unset is loopback-only", list: "", want: nil},
		{name: "whitespace only", list: "  ,\t, ", want: nil},
		{name: "single CIDR", list: "10.0.0.0/24", want: []string{"10.0.0.0/24"}},
		{
			name: "several, spaced",
			list: "10.0.0.0/24, 192.168.7.5/32 ,fd00::/64",
			want: []string{"10.0.0.0/24", "192.168.7.5/32", "fd00::/64"},
		},
		{name: "host bits are masked", list: "10.0.0.9/24", want: []string{"10.0.0.0/24"}},
		{name: "bare address is not a CIDR", list: "10.0.0.1", wantErr: true},
		{name: "not an address", list: "proxy.example.com/32", wantErr: true},
		// A zero-bit prefix trusts every peer, which is the spoofable
		// behaviour #2174 removed written a different way.
		{name: "IPv4 default route refused", list: "0.0.0.0/0", wantErr: true},
		{name: "IPv6 default route refused", list: "::/0", wantErr: true},
		{name: "one bad entry fails the whole list", list: "10.0.0.0/24,::/0", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTrustedProxies(tt.list)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseTrustedProxies(%q) = %v, want an error", tt.list, got)
				}

				return
			}
			if err != nil {
				t.Fatalf("ParseTrustedProxies(%q): %v", tt.list, err)
			}
			var gotStrings []string
			for _, prefix := range got {
				gotStrings = append(gotStrings, prefix.String())
			}
			if !slices.Equal(gotStrings, tt.want) {
				t.Errorf("ParseTrustedProxies(%q) = %v, want %v", tt.list, gotStrings, tt.want)
			}
		})
	}
}

// TestServerClientIP is the security invariant of #2174: a forwarding header
// is believed only from a hop the operator named, never from "any private
// address". niac terminates its own TLS and ships no proxy, so before this
// every real client was a "trusted proxy" and could mint a fresh rate-limit
// bucket per request by varying the header.
func TestServerClientIP(t *testing.T) {
	tests := []struct {
		name       string
		trusted    string
		remoteAddr string
		xForwarded string
		xRealIP    string
		want       string
	}{
		{
			name:       "no headers, the peer is the key",
			remoteAddr: "192.168.1.100:54321",
			want:       "192.168.1.100",
		},
		{
			name:       "no port in RemoteAddr",
			remoteAddr: "192.168.1.1",
			want:       "192.168.1.1",
		},
		{
			name:       "public peer cannot forge",
			remoteAddr: "8.8.8.8:1234",
			xForwarded: "192.168.1.100",
			want:       "8.8.8.8",
		},
		{
			// The defect itself: an unlisted private peer is not a proxy.
			name:       "unlisted private peer cannot forge",
			remoteAddr: "10.44.40.9:1234",
			xForwarded: "203.0.113.50",
			want:       "10.44.40.9",
		},
		{
			name:       "unlisted private peer cannot forge X-Real-IP either",
			remoteAddr: "192.168.1.5:1234",
			xRealIP:    "203.0.113.50",
			want:       "192.168.1.5",
		},
		{
			// Loopback is the single-host reverse-proxy deployment and is
			// trusted with no configuration.
			name:       "loopback is trusted by default",
			remoteAddr: "127.0.0.1:1234",
			xForwarded: "203.0.113.50",
			want:       "203.0.113.50",
		},
		{
			name:       "IPv6 loopback is trusted by default",
			remoteAddr: "[::1]:1234",
			xRealIP:    "203.0.113.100",
			want:       "203.0.113.100",
		},
		{
			name:       "listed proxy is believed",
			trusted:    "10.0.0.0/24",
			remoteAddr: "10.0.0.1:1234",
			xForwarded: "203.0.113.50",
			want:       "203.0.113.50",
		},
		{
			name:       "a private peer outside the listed range is not",
			trusted:    "10.0.0.0/24",
			remoteAddr: "10.9.9.9:1234",
			xForwarded: "203.0.113.50",
			want:       "10.9.9.9",
		},
		{
			// The chain is read right to left: each hop appends the peer it
			// saw, so the left-most entry is whatever the client chose to
			// send. Taking it (which is what this code did) lets a client
			// behind a real proxy forge its own key.
			name:       "forged left-most entry is skipped",
			trusted:    "10.0.0.0/24",
			remoteAddr: "10.0.0.1:1234",
			xForwarded: "1.2.3.4, 203.0.113.50",
			want:       "203.0.113.50",
		},
		{
			name:       "the walk stops at the last untrusted hop",
			trusted:    "10.0.0.0/24",
			remoteAddr: "10.0.0.1:1234",
			xForwarded: "1.2.3.4, 203.0.113.50, 10.0.0.2",
			want:       "203.0.113.50",
		},
		{
			name:       "a malformed entry cannot be reached past",
			trusted:    "10.0.0.0/24",
			remoteAddr: "10.0.0.1:1234",
			xForwarded: "1.2.3.4, not-an-ip, 10.0.0.2",
			want:       "not-an-ip",
		},
		{
			// A dual-stack listener reports an IPv4 peer as ::ffff:a.b.c.d,
			// which no IPv4 prefix contains unless it is unmapped first.
			name:       "IPv4-mapped peer matches an IPv4 prefix",
			trusted:    "10.0.0.0/24",
			remoteAddr: "[::ffff:10.0.0.1]:1234",
			xForwarded: "203.0.113.50",
			want:       "203.0.113.50",
		},
		{
			name:       "X-Forwarded-For wins over X-Real-IP",
			trusted:    "10.0.0.0/24",
			remoteAddr: "10.0.0.1:1234",
			xForwarded: "203.0.113.50",
			xRealIP:    "203.0.113.99",
			want:       "203.0.113.50",
		},
		{
			name:       "trusted hop with no usable header falls back to the peer",
			trusted:    "10.0.0.0/24",
			remoteAddr: "10.0.0.1:1234",
			xForwarded: " , ",
			want:       "10.0.0.1",
		},
		{
			// Fail closed: an address we cannot parse is in no CIDR.
			name:       "unparseable peer is never a trusted hop",
			remoteAddr: "not-an-address",
			xForwarded: "203.0.113.50",
			want:       "not-an-address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := serverWithTrustedProxies(t, tt.trusted)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwarded)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			if got := srv.clientIP(req); got != tt.want {
				t.Errorf("clientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestForgedForwardedHeaderSharesOneRateLimitBucket is the defect stated as
// the consequence that matters, through the real limiter rather than through
// clientIP alone: the limiter is niac's only brute-force defence on the token
// path, and before #2174 a client varying the header per request was never
// throttled. Asserted on the bucket identity, not on a 429, so it does not
// depend on the limit's configured rate.
func TestForgedForwardedHeaderSharesOneRateLimitBucket(t *testing.T) {
	srv := serverWithTrustedProxies(t, "")
	limiter := ratelimit.NewRateLimiter(rate.Limit(1), 1)

	buckets := make(map[*rate.Limiter]struct{})
	for _, forged := range []string{"203.0.113.1", "203.0.113.2", "203.0.113.3"} {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.44.40.9:1234" // one real client, a private peer
		req.Header.Set("X-Forwarded-For", forged)
		buckets[limiter.GetLimiter(srv.clientIP(req))] = struct{}{}
	}

	if len(buckets) != 1 {
		t.Errorf("one client got %d rate-limit buckets by varying X-Forwarded-For, want 1", len(buckets))
	}
}

// serverWithTrustedProxies builds the minimum Server the client-IP policy
// needs. NewServer is deliberately not used: it opens storage and a content
// library, and clientIP reads one field.
func serverWithTrustedProxies(t *testing.T, list string) *Server {
	t.Helper()

	trusted, err := ParseTrustedProxies(list)
	if err != nil {
		t.Fatalf("ParseTrustedProxies(%q): %v", list, err)
	}

	return &Server{cfg: ServerConfig{TrustedProxies: trusted}}
}
