//go:build linux && integration

package wiretest

import (
	"strings"
	"testing"
)

// The wire itself is a precondition for every assertion in this package, so it
// gets its own test: when the veth pair is the thing that broke, a protocol
// test failing on "no response" points at the responder and wastes the hour.
func TestWireIsReachable(t *testing.T) {
	w := setupWire(t)

	out := run(t, "ping", "-c", "2", "-W", "2", simAddr)
	if !strings.Contains(out, "0% packet loss") {
		t.Fatalf("ping %s across the veth pair did not reach the simulated end:\n%s", simAddr, out)
	}
	_ = w
}

// The test host runs snmpd on 0.0.0.0:161. If the namespace ever stopped
// isolating us from it, an SNMP walk would be answered by snmpd and the suite
// would report a passing walk against the wrong process — the worst kind of
// green. Assert the port is free on the simulated side before anything binds it.
func TestSimulatedNamespaceHasNoInheritedListeners(t *testing.T) {
	w := setupWire(t)

	out := w.nsRun("ss", "-lntup")
	if strings.Contains(out, ":161") {
		t.Fatalf("something already listens on :161 inside %s; the namespace is not isolating the host's snmpd:\n%s", wireNS, out)
	}
}
