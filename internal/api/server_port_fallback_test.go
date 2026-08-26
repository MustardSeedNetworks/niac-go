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

// TestBindWithFallback_FallsBackOneStep grabs a port, holds it open, then
// expects bindWithFallback to land on requested+1. This is the regression
// test for #1537: on Windows, before the platform split, isAddrInUse never
// recognised WSAEADDRINUSE and this fell through to a fatal error instead.
func TestBindWithFallback_FallsBackOneStep(t *testing.T) {
	hold, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("hold listen: %v", err)
	}
	defer func() { _ = hold.Close() }()

	taken := hold.Addr().(*net.TCPAddr).Port
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
// Platform-agnostic: the two isAddrInUse implementations agree on this case,
// so it needs no build tag. Each implementation's positive case (a real
// EADDRINUSE/WSAEADDRINUSE match) lives next to it in
// server_port_fallback_unix_test.go / server_port_fallback_windows_test.go.
func TestIsAddrInUse_RejectsOtherErrors(t *testing.T) {
	if isAddrInUse(errors.New("some unrelated failure")) {
		t.Fatalf("expected isAddrInUse to reject unrelated error")
	}
}

// TestBindWithFallback_BadAddrErrors confirms a malformed addr is rejected.
func TestBindWithFallback_BadAddrErrors(t *testing.T) {
	_, _, err := bindWithFallback(context.Background(), silentLogger(), "not-a-real-addr")
	if err == nil {
		t.Fatalf("expected error parsing bad addr")
	}
}
