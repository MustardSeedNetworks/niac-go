package api

import (
	"net"
	"testing"

	"golang.org/x/sys/windows"
)

// TestIsAddrInUse_RecognisesWSAEADDRINUSE confirms isAddrInUse matches a
// wrapped WSAEADDRINUSE — the Winsock errno, not syscall.EADDRINUSE (#1537).
func TestIsAddrInUse_RecognisesWSAEADDRINUSE(t *testing.T) {
	wrapped := &net.OpError{Op: "listen", Err: windows.WSAEADDRINUSE}
	if !isAddrInUse(wrapped) {
		t.Fatalf("expected isAddrInUse to match WSAEADDRINUSE")
	}
}
