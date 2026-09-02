//go:build linux && integration

// Package wiretest drives a running simulation over a real Linux wire.
//
// Every other test in this repo asserts on structs. This one asserts on frames:
// the daemon serves a generated pack on one end of a veth pair, and the test
// reads what actually arrives on the other end. A responder that satisfies its
// unit tests and emits nothing — or emits the wrong field — fails here and
// nowhere else.
package wiretest

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The simulated side lives in its own network namespace because the test host
// runs snmpd on 0.0.0.0:161. In the root namespace the daemon's SNMP responder
// would lose the bind, or worse, the walk would be answered by snmpd and the
// test would pass against the wrong process.
const (
	wireNS      = "niacwire"
	simIface    = "nw-sim"
	testIface   = "nw-test"
	simAddrCIDR = "10.99.0.2/24"
	simAddr     = "10.99.0.2"
	testAddr    = "10.99.0.1"
)

type wire struct {
	t *testing.T
}

// run executes a command and fails the test with the combined output, which is
// where `ip` puts its actual reason for refusing.
func run(t *testing.T, name string, args ...string) string {
	t.Helper()
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return string(out)
}

// nsRun executes a command inside the simulated namespace.
func (w *wire) nsRun(name string, args ...string) string {
	w.t.Helper()
	return run(w.t, "ip", append([]string{"netns", "exec", wireNS, name}, args...)...)
}

// setupWire builds the veth pair and returns with both ends up. The namespace
// and link are torn down even when the test fails, so a crashed run does not
// leave the next one to collide with a stale nw-test.
func setupWire(t *testing.T) *wire {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Skip("wire tests need root to create network namespaces; run under sudo")
	}
	if _, err := exec.LookPath("ip"); err != nil {
		t.Skip("iproute2 not installed")
	}

	teardown(t)
	t.Cleanup(func() { teardown(t) })

	run(t, "ip", "netns", "add", wireNS)
	run(t, "ip", "link", "add", testIface, "type", "veth", "peer", "name", simIface)
	run(t, "ip", "link", "set", simIface, "netns", wireNS)
	run(t, "ip", "addr", "add", fmt.Sprintf("%s/24", testAddr), "dev", testIface)
	run(t, "ip", "link", "set", testIface, "up")

	w := &wire{t: t}
	w.nsRun("ip", "addr", "add", simAddrCIDR, "dev", simIface)
	w.nsRun("ip", "link", "set", simIface, "up")
	w.nsRun("ip", "link", "set", "lo", "up")
	return w
}

// teardown is best-effort: a missing namespace or link is the normal case on a
// clean run, so failures here are deliberately ignored.
func teardown(t *testing.T) {
	t.Helper()
	_ = exec.Command("ip", "netns", "del", wireNS).Run()
	_ = exec.Command("ip", "link", "del", testIface).Run()
}
