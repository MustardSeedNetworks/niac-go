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

// holdPort returns a listener occupying an ephemeral port, and that port.
func holdPort(t *testing.T) (net.Listener, int) {
	t.Helper()

	hold, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("hold listen: %v", err)
	}
	return hold, hold.Addr().(*net.TCPAddr).Port
}

// TestBindWithFallback_StepsPastAnOccupiedPort holds a port open and expects
// bindWithFallback to walk past it rather than fail. Regression test for
// #1537: on Windows, before the platform split, the predicate never recognised
// WSAEADDRINUSE, so this fell through to a fatal error instead of stepping.
//
// It asserts the offset is somewhere in 1..portFallbackMaxOffset, not that it
// is exactly 1, because the exact landing port is not a property of the code
// under test. Which successor is free belongs to the machine: a WinNAT-reserved
// port sends the walk to taken+2 (that is #1682's fix working, not a bug), and
// a port probed as free can be claimed by something else before the walk
// reaches it. Asserting taken+1 turned both into failures — one of them ejected
// a PR from the merge queue on a windows-latest run at 127.0.0.1:49664.
//
// The old helper probed taken+1 first and skipped the test after 20 failed
// attempts, which narrowed the race without closing it and could silently skip
// the only coverage of this path. Both are gone: with a range assertion there
// is nothing to pre-probe.
func TestBindWithFallback_StepsPastAnOccupiedPort(t *testing.T) {
	hold, taken := holdPort(t)
	defer func() { _ = hold.Close() }()

	addr := "127.0.0.1:" + strconv.Itoa(taken)

	ln, bound, err := bindWithFallback(context.Background(), silentLogger(), addr)
	if err != nil {
		t.Fatalf("bindWithFallback fell through instead of stepping past %d: %v", taken, err)
	}
	defer func() { _ = ln.Close() }()

	boundPort := ln.Addr().(*net.TCPAddr).Port
	offset := boundPort - taken
	if offset < 1 || offset > portFallbackMaxOffset {
		t.Fatalf("expected a port in %d+1..%d (offset 1..%d), got %q (offset %d)",
			taken, taken+portFallbackMaxOffset, portFallbackMaxOffset, bound, offset)
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
