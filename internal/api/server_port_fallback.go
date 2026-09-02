package api

// server_port_fallback.go provides bindWithFallback, a helper that opens
// a TCP listener on a desired address and walks port+1..+9 if the
// canonical port is already in use. This keeps `niac daemon` runnable
// for developers who have another service squatting on 8445 without
// changing the documented default port (see #69). isPortUnavailable itself
// lives in the platform-specific server_port_fallback_unix.go / _windows.go
// siblings — see #1537 for why the predicate can't be shared, and #1682 for
// why the two platforms classify EACCES differently.

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"
)

// portFallbackMaxOffset is the maximum offset above the requested port that
// bindWithFallback will probe. Probes are requested..requested+portFallbackMaxOffset.
const portFallbackMaxOffset = 9

// bindWithFallback opens a TCP listener on addr (host:port form). If that
// port is in use it walks port+1..+portFallbackMaxOffset and returns the
// first listener that binds, logging a WARN with the requested and actual
// port via the supplied logger.
//
// Errors that do not make the port unusable are returned immediately — the
// caller must treat them as fatal (an invalid address, or on unix a privileged
// port needing root).
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
		if !isPortUnavailable(err) {
			return nil, "", fmt.Errorf("bind %s: %w", candidate, err)
		}
	}
	return nil, "", fmt.Errorf(
		"bind %s and +1..+%d: every port is in use or reserved",
		addr, portFallbackMaxOffset,
	)
}
