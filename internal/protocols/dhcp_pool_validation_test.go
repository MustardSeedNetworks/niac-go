package protocols

import (
	"context"
	"net"
	"os"
	"os/exec"
	"runtime/debug"
	"testing"
	"time"
)

func TestDHCPPoolRejectsNonIPv4BeforeDecoding(t *testing.T) {
	handler := NewDHCPHandler(nil)
	for _, address := range []net.IP{nil, {1, 2}, net.ParseIP("2001:db8::1")} {
		if _, err := handler.generateIPPool(address, net.ParseIP("10.0.0.109")); err == nil {
			t.Errorf("accepted non-IPv4 start: %v", address)
		}
		if _, err := handler.generateIPPool(net.ParseIP("10.0.0.100"), address); err == nil {
			t.Errorf("accepted non-IPv4 end: %v", address)
		}
	}
}

func TestDHCPPoolIncludesMaximumIPv4Endpoint(t *testing.T) {
	if os.Getenv("NIAC_DHCP_ENDPOINT_CHILD") == "1" {
		debug.SetMemoryLimit(16 << 20)
		handler := NewDHCPHandler(nil)
		for _, start := range []string{"255.255.255.255", "255.255.255.254"} {
			pool, err := handler.generateIPPool(net.ParseIP(start), net.IPv4bcast)
			if err != nil || len(pool) == 0 || len(pool) > 2 || !pool[0].Equal(net.ParseIP(start)) ||
				!pool[len(pool)-1].Equal(net.IPv4bcast) {
				t.Fatalf("maximum endpoint pool = %v, error=%v", pool, err)
			}
		}
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-test.run=^TestDHCPPoolIncludesMaximumIPv4Endpoint$")
	command.Env = append(os.Environ(), "NIAC_DHCP_ENDPOINT_CHILD=1")
	if output, runErr := command.CombinedOutput(); runErr != nil {
		t.Fatalf("bounded endpoint subprocess failed: %v (context=%v): %s", runErr, ctx.Err(), output)
	}
}
