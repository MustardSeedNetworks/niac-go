package api

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"testing"
)

// silentLogger returns a slog.Logger that discards output so tests don't
// spam the buffer with the expected fallback WARN.
func silentLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// TestBindWithFallback_FreePort confirms a free port is bound at offset 0.
func TestBindWithFallback_FreePort(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("probe listen: %v", err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	_ = probe.Close()

	addr := "127.0.0.1:" + strconv.Itoa(port)
	ln, bound, err := bindWithFallback(context.Background(), silentLogger(), addr)
	if err != nil {
		t.Fatalf("bindWithFallback returned error: %v", err)
	}
	defer func() { _ = ln.Close() }()

	if bound != addr {
		t.Fatalf("expected bound addr %q, got %q", addr, bound)
	}
}

// holdPortWithFreeSuccessor returns a listener occupying some port P, having
// first confirmed P+1 is bindable, and the value of P.
//
// The confirmation is the point, and it is still needed after #1682. That fix
// made the walk STEP OVER a WinNAT-reserved port rather than stop at it — but
// this test asserts the listener lands on exactly taken+1, so a reserved
// successor now sends the walk to taken+2 and fails the assertion just the
// same. The failure mode changed; the flakiness did not.
//
// Observed on a windows-latest merge_group run at 127.0.0.1:49664, which
// ejected a PR from the merge queue.
func holdPortWithFreeSuccessor(t *testing.T) (net.Listener, int) {
	t.Helper()

	const attempts = 20
	for range attempts {
		hold, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("hold listen: %v", err)
		}
		port := hold.Addr().(*net.TCPAddr).Port

		// Probe the successor and release it again immediately. A port that
		// binds now can in principle be taken before bindWithFallback reaches
		// it, but nothing else on the runner is claiming single ports in this
		// range, and a reserved block — the failure being guarded against —
		// stays reserved.
		probe, probeErr := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port+1))
		if probeErr == nil {
			_ = probe.Close()
			return hold, port
		}
		_ = hold.Close()
	}

	t.Skipf("no ephemeral port with a bindable successor after %d attempts", attempts)
	return nil, 0
}

// TestBindWithFallback_FallsBackOneStep grabs a port, holds it open, then
// expects bindWithFallback to land on requested+1. This is the regression
// test for #1537: on Windows, before the platform split, the predicate never
// recognised WSAEADDRINUSE and this fell through to a fatal error instead.
func TestBindWithFallback_FallsBackOneStep(t *testing.T) {
	hold, taken := holdPortWithFreeSuccessor(t)
	defer func() { _ = hold.Close() }()

	addr := "127.0.0.1:" + strconv.Itoa(taken)

	ln, bound, err := bindWithFallback(context.Background(), silentLogger(), addr)
	if err != nil {
		t.Fatalf("bindWithFallback fell through: %v", err)
	}
	defer func() { _ = ln.Close() }()

	wantSuffix := ":" + strconv.Itoa(taken+1)
	if !strings.HasSuffix(bound, wantSuffix) {
		t.Fatalf("expected bound addr to end with %q, got %q", wantSuffix, bound)
	}
}

// TestBindWithFallback_PortZeroUsesEphemeral confirms ":0" passes through
// without any fallback (OS picks the ephemeral port).
func TestBindWithFallback_PortZeroUsesEphemeral(t *testing.T) {
	ln, bound, err := bindWithFallback(context.Background(), silentLogger(), "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bindWithFallback returned error: %v", err)
	}
	defer func() { _ = ln.Close() }()

	if strings.HasSuffix(bound, ":0") {
		t.Fatalf("expected resolved port, got %q", bound)
	}
}

// TestIsAddrInUse_RejectsOtherErrors ensures unrelated errors don't match.
// Platform-agnostic: the two isPortUnavailable implementations agree on this case,
// so it needs no build tag. Each implementation's positive case (a real
// EADDRINUSE/WSAEADDRINUSE match) lives next to it in
// server_port_fallback_unix_test.go / server_port_fallback_windows_test.go.
func TestIsAddrInUse_RejectsOtherErrors(t *testing.T) {
	if isPortUnavailable(errors.New("some unrelated failure")) {
		t.Fatalf("expected isPortUnavailable to reject unrelated error")
	}
}

// TestBindWithFallback_BadAddrErrors confirms a malformed addr is rejected.
func TestBindWithFallback_BadAddrErrors(t *testing.T) {
	_, _, err := bindWithFallback(context.Background(), silentLogger(), "not-a-real-addr")
	if err == nil {
		t.Fatalf("expected error parsing bad addr")
	}
}
