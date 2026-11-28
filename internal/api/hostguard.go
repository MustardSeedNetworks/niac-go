// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"net"
	"net/http"
	"strconv"
	"strings"
)

// loopbackHostGuard rejects requests whose Host header is not one of the
// daemon's own loopback names on the port it bound, while no API token is
// configured.
//
// A tokenless daemon trusts loopback reachability alone. DNS rebinding
// defeats that: a page on attacker.example re-resolves its name to 127.0.0.1
// and drives the admin API with the browser's same-origin rights, but its
// requests still carry the attacker's name in Host. Once a token is
// configured the bearer check is the gate, so the guard steps aside; it
// consults the token store per request so a SIGHUP rotation is honoured.
//
// This is a temporary mode: it ends when human login lands (#3165).
func (s *Server) loopbackHostGuard(next http.Handler, bound net.IP, port int) http.Handler {
	allowed := map[string]bool{
		"localhost":               true,
		"127.0.0.1":               true,
		net.IPv6loopback.String(): true,
		bound.String():            true,
	}
	wantPort := strconv.Itoa(port)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.AuthConfigured() {
			next.ServeHTTP(w, r)
			return
		}
		host, hostPort, err := net.SplitHostPort(r.Host)
		if err != nil || hostPort != wantPort || !allowed[strings.ToLower(host)] {
			http.Error(w, "invalid Host header", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
