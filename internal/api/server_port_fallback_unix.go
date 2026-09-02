//go:build !windows

package api

import (
	"errors"
	"syscall"
)

// isPortUnavailable reports whether a bind failed for a reason that makes THIS
// port unusable while leaving the next one worth trying.
//
// EADDRINUSE only. EACCES is deliberately excluded: on unix it means a
// privileged port below 1024 that needs root, and walking 80 -> 89 would turn
// one clear "permission denied" into "tried 10 ports and gave up". The Windows
// sibling does include its EACCES equivalent, because there the same errno
// means a WinNAT-reserved port rather than a privilege problem (#1682).
func isPortUnavailable(err error) bool {
	return errors.Is(err, syscall.EADDRINUSE)
}
