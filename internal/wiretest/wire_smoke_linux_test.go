//go:build linux && integration

package wiretest_test

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

// No simulated address may be assigned to a kernel interface. If one ever is,
// the kernel starts answering ARP and emitting ICMP unreachables on behalf of
// the simulation, and an assertion can pass against the kernel instead of
// against NIAC — the worst kind of green.
func TestNoSimulatedAddressIsOnAKernelInterface(t *testing.T) {
	requireWire(t)

	out := run(t, "ip", "-4", "-br", "addr", "show")
	if strings.Contains(out, transitGateway+"/") {
		t.Errorf(
			"the simulated gateway %s is assigned to a kernel interface, so the kernel can answer for it:\n%s",
			transitGateway,
			out,
		)
	}
}
