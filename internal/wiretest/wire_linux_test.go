//go:build linux && integration

// Package wiretest drives a running simulation over a real Linux wire.
//
// Every other test in this repo asserts on structs. This one asserts on frames:
// the daemon serves a generated pack on one end of a veth pair, and the test
// reads what actually arrives on the other. A responder that satisfies its unit
// tests and emits nothing — or emits the wrong field — fails here and nowhere
// else.
package wiretest_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Both veth ends live in one dedicated namespace.
//
// NIAC does not bind sockets: capture.New opens libpcap on the interface and
// the responders match on the captured frame's fields (internal/protocols/udp.go
// keys SNMP off udp.DstPort == 161). So the host's snmpd was never a competitor
// for a bind. What the namespace actually prevents is the kernel itself
// answering for the simulated network — replying to ARP for, or returning ICMP
// unreachables about, addresses that exist only inside the simulation.
//
// The test end does carry one address, because an SNMP client needs a source
// address to be answered at. That is the client's own address and nothing
// simulated: the invariant that matters is that no *simulated* address is
// assigned to a kernel interface, which TestNoSimulatedAddressIsOnAKernelInterface
// asserts.
const (
	wireNS    = "niacwire"
	simIface  = "nw-sim"  // the daemon binds this end
	testIface = "nw-test" // the test client drives this end
	nsEnvVar  = "NIAC_WIRETEST_IN_NS"

	// The pack's transit network. clientCIDR is below the authored DHCP pool
	// (.100-.199) so it can never collide with a lease the simulation hands out.
	clientCIDR = "10.254.200.50/24"
)

// TestMain builds the namespace once, then re-executes this test binary inside
// it. Entering a namespace in-process would mean setns(2) on a thread-locked
// goroutine, and neither libpcap handles nor the daemon's goroutines would
// reliably inherit it; re-exec makes every thread in the process start there.
func TestMain(m *testing.M) {
	if os.Getenv(nsEnvVar) == "1" {
		os.Exit(m.Run())
	}
	if os.Geteuid() != 0 {
		fmt.Fprintln(
			os.Stderr,
			"wiretest: needs root to create a network namespace; run under sudo",
		)
		os.Exit(0)
	}
	if _, err := exec.LookPath("ip"); err != nil {
		fmt.Fprintln(os.Stderr, "wiretest: iproute2 not installed")
		os.Exit(0)
	}

	if err := setupNamespace(); err != nil {
		teardownNamespace()
		fmt.Fprintf(os.Stderr, "wiretest: %v\n", err)
		os.Exit(1)
	}
	// Re-run ourselves inside the namespace with the same flags. Teardown is
	// explicit rather than deferred because this function ends in os.Exit.
	args := append([]string{"netns", "exec", wireNS, os.Args[0]}, os.Args[1:]...)
	cmd := exec.Command("ip", args...)
	cmd.Env = append(os.Environ(), nsEnvVar+"=1")
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin

	code := 0
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code = exit.ExitCode()
		} else {
			fmt.Fprintf(os.Stderr, "wiretest: re-exec failed: %v\n", err)
			code = 1
		}
	}
	teardownNamespace()
	os.Exit(code)
}

// setupNamespace creates the namespace and an unaddressed veth pair inside it.
func setupNamespace() error {
	teardownNamespace()
	steps := [][]string{
		{"netns", "add", wireNS},
		{"link", "add", testIface, "type", "veth", "peer", "name", simIface},
		{"link", "set", testIface, "netns", wireNS},
		{"link", "set", simIface, "netns", wireNS},
		{"netns", "exec", wireNS, "ip", "link", "set", "lo", "up"},
		{"netns", "exec", wireNS, "ip", "link", "set", testIface, "up"},
		{"netns", "exec", wireNS, "ip", "link", "set", simIface, "up"},
		{"netns", "exec", wireNS, "ip", "addr", "add", clientCIDR, "dev", testIface},
	}
	for _, args := range steps {
		if out, err := exec.Command("ip", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("ip %s: %w\n%s", strings.Join(args, " "), err, out)
		}
	}
	return nil
}

// teardownNamespace is best-effort: on a clean run the namespace does not exist
// yet, so a failure here is the normal case rather than an error.
func teardownNamespace() {
	_ = exec.Command("ip", "netns", "del", wireNS).Run()
	_ = exec.Command("ip", "link", "del", testIface).Run()
}

// requireWire fails a test that somehow ran outside the namespace, so a missing
// interface is reported as the setup fault it is rather than as a protocol
// timeout twenty seconds later.
func requireWire(t *testing.T) {
	t.Helper()
	if os.Getenv(nsEnvVar) != "1" {
		t.Fatal("test is running outside the wire namespace; TestMain should have re-executed it")
	}
	for _, iface := range []string{simIface, testIface} {
		if _, err := os.Stat("/sys/class/net/" + iface); err != nil {
			t.Fatalf("interface %s is missing inside %s: %v", iface, wireNS, err)
		}
	}
}

// run executes a command and fails the test with the combined output, which is
// where ip puts its actual reason for refusing.
func run(t *testing.T, name string, args ...string) string {
	t.Helper()
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return string(out)
}
