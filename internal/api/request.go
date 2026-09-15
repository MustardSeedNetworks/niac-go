package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
)

// TrustedProxiesEnv names the environment variable carrying the trusted-proxy
// CIDR list, comma separated (for example "10.0.0.0/24,192.168.7.5/32").
// Unset — the default — means loopback is the only trusted hop.
const TrustedProxiesEnv = "NIAC_TRUSTED_PROXIES"

// ParseTrustedProxies parses a comma-separated CIDR list into prefixes.
//
// A prefix covering every address ("0.0.0.0/0", "::/0", or any other zero-bit
// spelling) is refused: trusting every peer is the spoofable behaviour #2174
// removes, written a different way. An empty or whitespace-only list parses to
// nil, which keeps the loopback-only default.
func ParseTrustedProxies(list string) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for entry := range strings.SplitSeq(list, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			return nil, fmt.Errorf("trusted proxy %q: %w", entry, err)
		}
		if prefix.Bits() == 0 {
			return nil, fmt.Errorf("trusted proxy %q trusts every peer, which makes "+
				"X-Forwarded-For spoofable by any client", entry)
		}
		prefixes = append(prefixes, prefix.Masked())
	}

	return prefixes, nil
}

// clientIP returns the client address that security decisions are keyed on:
// the per-IP rate-limit buckets and the addresses the authz denials log.
//
// It is the immediate TCP peer, and honours X-Forwarded-For / X-Real-IP only
// when that peer is a trusted hop: a loopback address (the single-host
// reverse-proxy deployment) or a member of the operator's [TrustedProxiesEnv]
// list. Before #2174 every RFC 1918 peer was trusted, and since niac
// terminates its own TLS and ships no proxy that meant every real client:
// varying the header per request minted a fresh bucket each time and the only
// brute-force defence on the token path never fired.
//
// It fails closed. An unparseable peer address is not loopback and is in no
// CIDR, so forwarding headers are ignored rather than trusted from an unknown
// source, and an unset list leaves loopback as the only trusted hop.
func (s *Server) clientIP(r *http.Request) string {
	peer := r.RemoteAddr
	if host, _, err := net.SplitHostPort(peer); err == nil {
		// RemoteAddr does not always carry a port (httptest, unix sockets).
		peer = host
	}

	addr, err := netip.ParseAddr(peer)
	if err != nil || !isTrustedHop(addr, s.cfg.TrustedProxies) {
		return peer
	}
	if forwarded := forwardedClientIP(r, s.cfg.TrustedProxies); forwarded != "" {
		return forwarded
	}

	return peer
}

// forwardedClientIP returns the client IP conveyed by X-Forwarded-For or
// X-Real-IP, or "" if neither carries a usable value. Callers MUST gate its
// use on the immediate peer being a trusted hop.
//
// The header is read right to left, not left to right. Every proxy in the
// chain appends the peer it saw (nginx's $proxy_add_x_forwarded_for, and every
// load balancer that follows RFC 7239's model), so the rightmost entries are
// the ones our own trusted hops observed and the leftmost is whatever the
// original client chose to send — a client that pre-populates the header
// prepends an address of its choosing. Walking from the right and discarding
// entries that are themselves trusted hops stops at the last address the
// trusted chain actually vouched for, which is as far back as it can be
// believed. An entry that will not parse also stops the walk, so a malformed
// header cannot be used to reach past it.
func forwardedClientIP(r *http.Request, trusted []netip.Prefix) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for _, raw := range slices.Backward(strings.Split(xff, ",")) {
			entry := strings.TrimSpace(raw)
			if entry == "" {
				continue
			}
			addr, err := netip.ParseAddr(entry)
			if err != nil || !isTrustedHop(addr, trusted) {
				return entry
			}
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}

	return ""
}

// isTrustedHop reports whether addr is a hop whose forwarding headers may be
// believed: a loopback address, or a member of the configured list.
// An invalid address is not trusted, so the caller fails closed.
func isTrustedHop(addr netip.Addr, trusted []netip.Prefix) bool {
	if !addr.IsValid() {
		return false
	}
	if addr.IsLoopback() {
		return true
	}
	// A dual-stack listener reports an IPv4 peer as ::ffff:a.b.c.d, which no
	// IPv4 prefix contains, and Prefix.Contains rejects any zoned address.
	addr = addr.Unmap().WithZone("")
	for _, prefix := range trusted {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}

// generateRequestID creates a unique request ID for tracing.
func generateRequestID() string {
	b := make([]byte, requestIDBytes)
	_, _ = rand.Read(b) // crypto/rand read errors will result in zero bytes, still usable

	return hex.EncodeToString(b)
}
