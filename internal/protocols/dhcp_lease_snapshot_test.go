package protocols

import (
	"net"
	"testing"
)

func TestDHCPAllocatedLeaseIsStableSnapshot(t *testing.T) {
	_, handler, _ := newDHCPTestHandler(t)
	mac := net.HardwareAddr{2, 0, 0, 0, 1, 1}
	first, err := handler.allocateLease(mac, net.ParseIP("10.20.200.100"), "first")
	if err != nil {
		t.Fatal(err)
	}
	_, err = handler.allocateLease(mac, net.ParseIP("10.20.200.150"), "second")
	if err != nil {
		t.Fatal(err)
	}
	if !first.IP.Equal(net.ParseIP("10.20.200.100")) || first.Hostname != "first" {
		t.Fatalf("returned lease mutated after allocation lock released: %+v", first)
	}
}

func TestDHCPLeaseOwnsInputAddresses(t *testing.T) {
	_, handler, _ := newDHCPTestHandler(t)
	mac := net.HardwareAddr{2, 0, 0, 0, 1, 1}
	key := mac.String()
	address := net.ParseIP("10.20.200.100").To4()
	if _, err := handler.allocateLease(mac, address, "first"); err != nil {
		t.Fatal(err)
	}
	address[3], mac[5] = 199, 99
	if lease := handler.leases[key]; lease.IP.String() != "10.20.200.100" || lease.MAC.String() != key {
		t.Fatalf("caller changed retained lease via input slices: %+v", lease)
	}
}
