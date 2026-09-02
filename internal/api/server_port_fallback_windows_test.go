package api

import (
	"net"
	"testing"

	"golang.org/x/sys/windows"
)

// TestIsPortUnavailable_RecognisesWSAEADDRINUSE confirms the predicate matches
// a wrapped WSAEADDRINUSE — the Winsock errno, not syscall.EADDRINUSE (#1537).
func TestIsPortUnavailable_RecognisesWSAEADDRINUSE(t *testing.T) {
	wrapped := &net.OpError{Op: "listen", Err: windows.WSAEADDRINUSE}
	if !isPortUnavailable(wrapped) {
		t.Fatalf("expected isPortUnavailable to match WSAEADDRINUSE")
	}
}

// TestIsPortUnavailable_RecognisesWSAEACCES is the regression test for #1682.
//
// A port inside a WinNAT/Hyper-V reserved block fails to bind with WSAEACCES,
// not WSAEADDRINUSE. Before this, the walk treated that as fatal and stopped —
// so a daemon whose canonical port had a reserved successor failed to start
// with a Winsock permissions message rather than stepping over it.
//
// Observed on a windows-latest runner at 127.0.0.1:49664, which ejected a PR
// from the merge queue.
func TestIsPortUnavailable_RecognisesWSAEACCES(t *testing.T) {
	wrapped := &net.OpError{Op: "listen", Err: windows.WSAEACCES}
	if !isPortUnavailable(wrapped) {
		t.Fatalf("expected isPortUnavailable to match WSAEACCES")
	}
}
