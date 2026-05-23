package api

// server_port_fallback.go provides bindWithFallback, a helper that opens
// a TCP listener on a desired address and walks port+1..+9 if the
// canonical port is already in use. This keeps `niac daemon` runnable
// for developers who have another service squatting on 8080 without
// changing the documented default port (see #69).

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"syscall"
)

// portFallbackMaxOffset is the maximum offset above the requested port that
// bindWithFallback will probe. Probes are requested..requested+portFallbackMaxOffset.
const portFallbackMaxOffset = 9

// bindWithFallback opens a TCP listener on addr (host:port form). If that
// port is in use it walks port+1..+portFallbackMaxOffset and returns the
// first listener that binds, logging a WARN with the requested and actual
// port via the supplied logger.
//
// Non-EADDRINUSE errors are returned immediately — the caller must treat
// them as fatal (permission denied, invalid address, etc.).
//
// The returned address string is the actual host:port the listener is
// bound on (suitable for assigning back into [http.Server.Addr] so
// /__version log lines reflect reality). The caller is responsible for
// closing the returned listener (typically by passing it to
// [http.Server.Serve] which closes on shutdown).
func bindWithFallback(
	ctx context.Context,
	logger *slog.Logger,
	addr string,
) (net.Listener, string, error) {
	host, portStr, splitErr := net.SplitHostPort(addr)
	if splitErr != nil {
		return nil, "", fmt.Errorf("parse listen address %q: %w", addr, splitErr)
	}
	port, parseErr := strconv.Atoi(portStr)
	if parseErr != nil {
		return nil, "", fmt.Errorf("parse port %q from %q: %w", portStr, addr, parseErr)
	}

	var lc net.ListenConfig

	// Port 0 means "let the OS pick" — no fallback needed.
	if port == 0 {
		ln, err := lc.Listen(ctx, "tcp", addr)
		if err != nil {
			return nil, "", fmt.Errorf("bind %s: %w", addr, err)
		}
		return ln, ln.Addr().String(), nil
	}

	for offset := 0; offset <= portFallbackMaxOffset; offset++ {
		actual := port + offset
		candidate := net.JoinHostPort(host, strconv.Itoa(actual))
		ln, err := lc.Listen(ctx, "tcp", candidate)
		if err == nil {
			if offset > 0 && logger != nil {
				logger.WarnContext(ctx,
					"requested port is in use, bound fallback port instead",
					"requested", port,
					"bound", actual,
				)
			}
			return ln, candidate, nil
		}
		if !isAddrInUse(err) {
			return nil, "", fmt.Errorf("bind %s: %w", candidate, err)
		}
	}
	return nil, "", fmt.Errorf(
		"bind %s and +1..+%d all in use",
		addr, portFallbackMaxOffset,
	)
}

// isAddrInUse reports whether err indicates the address-in-use condition.
// It checks [syscall.EADDRINUSE] via [errors.Is] (works on Linux/macOS) and
// falls back to a string match for platforms whose listener wrapping does
// not unwrap to the syscall errno.
func isAddrInUse(err error) bool {
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Err != nil {
		return containsAddrInUse(opErr.Err.Error())
	}
	return false
}

// containsAddrInUse looks for the canonical address-in-use substring.
// Split out so it can be unit-tested independently of platform errno
// behaviour.
func containsAddrInUse(msg string) bool {
	const needle = "address already in use"
	if len(msg) < len(needle) {
		return false
	}
	for i := 0; i+len(needle) <= len(msg); i++ {
		if msg[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
