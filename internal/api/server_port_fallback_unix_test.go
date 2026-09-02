//go:build !windows

package api

import (
	"net"
	"syscall"
	"testing"
)

// TestIsPortUnavailable_RecognisesSyscall confirms the predicate matches a
// wrapped EADDRINUSE.
func TestIsPortUnavailable_RecognisesSyscall(t *testing.T) {
	wrapped := &net.OpError{Op: "listen", Err: syscall.EADDRINUSE}
	if !isPortUnavailable(wrapped) {
		t.Fatalf("expected isPortUnavailable to match EADDRINUSE")
	}
}

// TestIsPortUnavailable_RejectsEACCES pins the deliberate asymmetry with the
// Windows sibling. On unix EACCES means a privileged port below 1024 that needs
// root, so the walk must NOT step over it: doing so would turn one clear
// "permission denied" on port 80 into "tried 10 ports and gave up" (#1682).
func TestIsPortUnavailable_RejectsEACCES(t *testing.T) {
	wrapped := &net.OpError{Op: "listen", Err: syscall.EACCES}
	if isPortUnavailable(wrapped) {
		t.Fatalf("expected isPortUnavailable to reject EACCES on unix")
	}
}
