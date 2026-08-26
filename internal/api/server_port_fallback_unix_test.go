//go:build !windows

package api

import (
	"net"
	"syscall"
	"testing"
)

// TestIsAddrInUse_RecognisesSyscall confirms isAddrInUse matches a wrapped
// EADDRINUSE.
func TestIsAddrInUse_RecognisesSyscall(t *testing.T) {
	wrapped := &net.OpError{Op: "listen", Err: syscall.EADDRINUSE}
	if !isAddrInUse(wrapped) {
		t.Fatalf("expected isAddrInUse to match EADDRINUSE")
	}
}
