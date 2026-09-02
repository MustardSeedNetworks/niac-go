//go:build linux && integration

package wiretest

import (
	"os"
	"strings"
	"testing"
)

// The wire is a precondition for every assertion in this package, so it gets
// its own test: when the veth pair is what broke, a protocol test failing on
// "no response" points at the responder and wastes the hour.
func TestWireExistsAndIsUp(t *testing.T) {
	requireWire(t)

	for _, iface := range []string{simIface, testIface} {
		state, err := os.ReadFile("/sys/class/net/" + iface + "/operstate")
		if err != nil {
			t.Fatalf("reading operstate of %s: %v", iface, err)
		}
		if got := strings.TrimSpace(string(state)); got != "up" {
			t.Errorf("%s operstate = %q, want \"up\"", iface, got)
		}
	}
}

// Neither end may carry a kernel address. If one ever does, the kernel starts
// answering ARP and emitting ICMP unreachables for the simulated network, and
// an assertion can pass against the kernel instead of against NIAC — the worst
// kind of green.
func TestWireEndsHaveNoKernelAddresses(t *testing.T) {
	requireWire(t)

	out := run(t, "ip", "-4", "-br", "addr", "show")
	for _, iface := range []string{simIface, testIface} {
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, iface) && strings.Contains(line, ".") {
				t.Errorf("%s carries a kernel IPv4 address, which lets the kernel answer for the simulation: %s", iface, line)
			}
		}
	}
}
