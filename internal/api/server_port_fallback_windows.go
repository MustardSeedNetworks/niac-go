package api

import (
	"errors"

	"golang.org/x/sys/windows"
)

// isPortUnavailable reports whether a bind failed for a reason that makes THIS
// port unusable while leaving the next one worth trying.
//
// Two Winsock errnos qualify on Windows, and the walk needs both:
//
//   - WSAEADDRINUSE — something else already holds the port. Note this is not
//     the value syscall.EADDRINUSE carries here: on Windows that constant is a
//     synthesised APPLICATION_ERROR with nothing mapping the Winsock errno back
//     to it, so comparing against it is always false and the walk would never
//     happen (#1537).
//
//   - WSAEACCES — the port sits inside a range reserved by WinNAT/Hyper-V.
//     Those blocks live inside the dynamic range (49152-65535) and are present
//     on any host running Hyper-V, WSL2 or Docker Desktop. Bind reports
//     "an attempt was made to access a socket in a way forbidden by its access
//     permissions", which reads like a privilege problem and is not one — the
//     port is simply spoken for. Treating it as fatal stopped the walk dead and
//     failed the daemon's start on a port it should have stepped over (#1682).
//
// The unix sibling deliberately does NOT treat EACCES this way: there it means
// a privileged port below 1024 and genuinely needs root, so walking would turn
// one clear "permission denied" into "tried 10 ports and gave up".
func isPortUnavailable(err error) bool {
	return errors.Is(err, windows.WSAEADDRINUSE) || errors.Is(err, windows.WSAEACCES)
}
