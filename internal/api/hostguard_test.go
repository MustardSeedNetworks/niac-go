// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api/tokenstore"
)

func TestLoopbackHostGuard(t *testing.T) {
	tokens := tokenstore.NewTokenStore([]tokenstore.ScopedToken{{Value: "t", Scope: tokenstore.ScopeAdmin}})

	tests := []struct {
		name       string
		host       string
		bound      net.IP
		tokens     *tokenstore.TokenStore
		wantStatus int
	}{
		{"127.0.0.1", "127.0.0.1:8445", net.ParseIP("127.0.0.1"), nil, http.StatusOK},
		{"localhost", "localhost:8445", net.ParseIP("127.0.0.1"), nil, http.StatusOK},
		{"localhost mixed case", "LocalHost:8445", net.ParseIP("127.0.0.1"), nil, http.StatusOK},
		{"ipv6 loopback", "[::1]:8445", net.ParseIP("::1"), nil, http.StatusOK},
		{"bound address", "127.0.0.2:8445", net.ParseIP("127.0.0.2"), nil, http.StatusOK},
		{"rebinding name", "attacker.example:8445", net.ParseIP("127.0.0.1"), nil, http.StatusForbidden},
		{"rebinding name no port", "attacker.example", net.ParseIP("127.0.0.1"), nil, http.StatusForbidden},
		{"loopback wrong port", "127.0.0.1:9999", net.ParseIP("127.0.0.1"), nil, http.StatusForbidden},
		{"loopback no port", "127.0.0.1", net.ParseIP("127.0.0.1"), nil, http.StatusForbidden},
		{"localhost trailing dot", "localhost.:8445", net.ParseIP("127.0.0.1"), nil, http.StatusForbidden},
		{"empty host", "", net.ParseIP("127.0.0.1"), nil, http.StatusForbidden},
		{"other loopback not bound", "127.0.0.3:8445", net.ParseIP("127.0.0.2"), nil, http.StatusForbidden},
		{"token configured steps aside", "niac.example:8445", net.ParseIP("127.0.0.1"), tokens, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{tokens: tt.tokens}
			h := s.loopbackHostGuard(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}), tt.bound, 8445)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
			req.Host = tt.host
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("Host %q: got %d, want %d", tt.host, rec.Code, tt.wantStatus)
			}
		})
	}
}

// A started tokenless daemon refuses a rebinding Host over the real listener,
// so the guard is wired into Start and not only correct in isolation.
func TestStartTokenlessLoopbackRefusesForeignHost(t *testing.T) {
	certDir := t.TempDir()
	s := NewServer(ServerConfig{
		Addr:                           "127.0.0.1:0",
		CertDir:                        certDir,
		SuppressUnauthenticatedWarning: true,
	})
	s.SetDaemonController(&nilDaemonController{})
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})

	certFile, _ := DefaultCertPaths(certDir)
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		t.Fatalf("read generated certificate: %v", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certPEM) {
		t.Fatal("generated certificate could not be added to root pool")
	}
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots, ServerName: "localhost"},
		},
	}
	_, port, err := net.SplitHostPort(s.BoundAddr())
	if err != nil {
		t.Fatalf("split bound address: %v", err)
	}

	for _, tt := range []struct {
		host string
		want int
	}{
		{net.JoinHostPort("localhost", port), http.StatusOK},
		{net.JoinHostPort("attacker.example", port), http.StatusForbidden},
	} {
		req, reqErr := http.NewRequestWithContext(
			t.Context(), http.MethodGet, "https://"+s.BoundAddr()+"/health/live", nil)
		if reqErr != nil {
			t.Fatal(reqErr)
		}
		req.Host = tt.host
		resp, doErr := client.Do(req)
		if doErr != nil {
			t.Fatalf("Host %q: %v", tt.host, doErr)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != tt.want {
			t.Errorf("Host %q: got %d, want %d", tt.host, resp.StatusCode, tt.want)
		}
	}
}
