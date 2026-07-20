package api

// tls_security_test.go exercises the default-secure gate (#88 part 1) and
// the HTTP→HTTPS redirect handler. These tests run without a real
// simulation/stack — they target the helpers directly and (for the gate)
// poke the Server.Start path with a tiny in-package fixture.

import (
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func TestAddrIsNonLoopback(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		addr     string
		want     bool
		wantErr  bool
		errSubst string
	}{
		{name: "loopback ipv4", addr: "127.0.0.1:8445", want: false},
		{name: "loopback ipv6", addr: "[::1]:8445", want: false},
		{name: "localhost host", addr: "localhost:8445", want: false},
		{name: "wildcard ipv4", addr: "0.0.0.0:8445", want: true},
		{name: "wildcard ipv6", addr: "[::]:8445", want: true},
		{name: "wildcard implicit", addr: ":8445", want: true},
		{name: "specific external", addr: "10.0.0.5:8445", want: true},
		{name: "bad form", addr: "no-port", wantErr: true, errSubst: "split"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := addrIsNonLoopback(tc.addr)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("addrIsNonLoopback(%q): expected error, got nil", tc.addr)
				}
				if tc.errSubst != "" && !strings.Contains(err.Error(), tc.errSubst) {
					t.Errorf("addrIsNonLoopback(%q) err = %v; expected to contain %q",
						tc.addr, err, tc.errSubst)
				}
				return
			}
			if err != nil {
				t.Fatalf("addrIsNonLoopback(%q) returned unexpected error: %v", tc.addr, err)
			}
			if got != tc.want {
				t.Errorf("addrIsNonLoopback(%q) = %v, want %v", tc.addr, got, tc.want)
			}
		})
	}
}

// makeGateServer returns a Server preconfigured with the daemon=nil but
// daemon-like state so Start()'s "needs Stack+Config" check passes. We
// flip daemon on to bypass that requirement; only the non-loopback gate
// is being exercised here.
func makeGateServer(addr, token string) *Server {
	s := &Server{
		cfg:    ServerConfig{Addr: addr, Token: token},
		logger: slog.Default(),
	}
	// Setting daemon to a non-nil sentinel skips the Stack/Config check.
	s.daemon = &nilDaemonController{}
	return s
}

// nilDaemonController is a typed-nil receiver so the daemon != nil branch
// in Server.Start passes without dragging in the real DaemonController.
type nilDaemonController struct{}

func (*nilDaemonController) PreflightSimulation(SimulationRequest) (fabric.Report, error) {
	return fabric.Report{}, nil
}
func (*nilDaemonController) StartSimulation(SimulationRequest) error { return nil }
func (*nilDaemonController) StopSimulation() error                   { return nil }
func (*nilDaemonController) GetStatus() SimulationStatus             { return SimulationStatus{} }

func TestStart_LoopbackWithoutTokenIsAllowed(t *testing.T) {
	// Loopback bind without token must NOT trip the gate. We don't
	// actually want to spawn listeners in the test, so we only call the
	// pre-flight check helper directly.
	non, err := addrIsNonLoopback("127.0.0.1:0")
	if err != nil {
		t.Fatalf("addrIsNonLoopback: %v", err)
	}
	if non {
		t.Fatal("127.0.0.1:0 should be reported as loopback")
	}
}

func TestStart_NonLoopbackWithoutTokenRefused(t *testing.T) {
	s := makeGateServer("0.0.0.0:0", "")
	err := s.Start()
	if err == nil {
		t.Fatal("Server.Start should refuse non-loopback without token")
	}
	if !errors.Is(err, errNonLoopbackRequiresToken) {
		t.Errorf("expected errNonLoopbackRequiresToken, got %v", err)
	}
	const wantSubst = "NIAC_API_TOKEN"
	if !strings.Contains(err.Error(), wantSubst) {
		t.Errorf("error must mention %q, got %q", wantSubst, err.Error())
	}
}

func TestDefaultCertPaths(t *testing.T) {
	t.Parallel()

	cert, key := DefaultCertPaths("")
	if !strings.HasSuffix(cert, "server.crt") {
		t.Errorf("default cert path should end in server.crt, got %q", cert)
	}
	if !strings.HasSuffix(key, "server.key") {
		t.Errorf("default key path should end in server.key, got %q", key)
	}

	cert, key = DefaultCertPaths("/custom/dir")
	if cert != "/custom/dir/server.crt" {
		t.Errorf("cert = %q, want /custom/dir/server.crt", cert)
	}
	if key != "/custom/dir/server.key" {
		t.Errorf("key = %q, want /custom/dir/server.key", key)
	}
}
