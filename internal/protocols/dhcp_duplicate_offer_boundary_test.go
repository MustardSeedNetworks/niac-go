package protocols

import (
	"net"
	"net/netip"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestDuplicateOfferRejectsReservedAndExistingConflict(t *testing.T) {
	t.Run("existing", func(t *testing.T) { testDuplicateOfferLeaseConflict(t, false) })
	t.Run("reserved", func(t *testing.T) { testDuplicateOfferLeaseConflict(t, true) })
}

func testDuplicateOfferLeaseConflict(t *testing.T, reserved bool) {
	t.Helper()
	_, cfg := isolationPair(t)
	conflict := net.ParseIP("10.0.0.105")
	cfg.Devices[1].IPAddresses[0] = conflict
	client := net.HardwareAddr{2, 0, 0, 0, 1, 1}
	if reserved {
		cfg.Devices[0].DHCPConfig.ClientLeases = []config.DHCPLease{{ClientIP: conflict, MACAddress: client}}
	}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := stack.dhcpHandlers[&cfg.Devices[0]]
	before, err := handler.allocateLease(client, conflict, "prior")
	if err != nil {
		t.Fatal(err)
	}
	if err = stack.SetDeviceAddressFault(
		"a",
		devicestate.FaultDuplicateDHCPOffer,
		netip.MustParseAddr("10.0.0.105"),
	); err != nil {
		t.Fatal(err)
	}
	if handler.canGrantRequestedIP(client, conflict) {
		t.Fatal("conflicting reservation or existing lease remained grantable")
	}
	if _, err = handler.allocateLease(client, conflict, "changed"); err == nil {
		t.Fatal("conflicting reservation or existing lease renewed")
	}
	if !reflect.DeepEqual(before, handler.leases[client.String()]) {
		t.Fatal("rejected conflict changed the existing lease")
	}
	healthy := net.HardwareAddr{2, 0, 0, 0, 1, 2}
	if _, err = handler.allocateLease(healthy, net.ParseIP("10.0.0.106"), "healthy"); err != nil {
		t.Fatalf("unrelated client blocked: %v", err)
	}
}

func TestDuplicateOfferRejectsUnownedOrCrossSegmentAddress(t *testing.T) {
	_, pair := isolationPair(t)
	cfg := &config.Config{Segments: []config.Segment{
		{Tag: 200, Devices: pair.Devices[:1]},
		{Tag: 300, Devices: pair.Devices[1:]},
	}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	for _, address := range []string{"10.0.0.2", "10.0.0.3", "10.0.0.99", "255.255.255.255", "2001:db8::1"} {
		if err := stack.SetDeviceAddressFault(
			"a",
			devicestate.FaultDuplicateDHCPOffer,
			netip.MustParseAddr(address),
		); err == nil {
			t.Fatalf("accepted self, cross-segment, absent or unusable address %s", address)
		}
	}
	if len(stack.deviceStates[&cfg.Segments[0].Devices[0]].Snapshot().DeviceFaults) != 0 {
		t.Fatal("rejected address mutated fault state")
	}
}

func TestDuplicateOfferRechecksPeerOwnership(t *testing.T) {
	stack, cfg := isolationPair(t)
	if err := stack.SetDeviceAddressFault(
		"a",
		devicestate.FaultDuplicateDHCPOffer,
		netip.MustParseAddr("10.0.0.3"),
	); err != nil {
		t.Fatal(err)
	}
	peer := stack.deviceStates[&cfg.Devices[1]]
	network := peer.Snapshot().Network
	network.Interfaces[0].Address = netip.MustParsePrefix("10.0.0.4/24")
	peer.ReplaceNetwork(network)
	sendIsolationDHCP(t, stack, dhcpDiscover(net.HardwareAddr{2, 0, 0, 0, 1, 1}))
	assertDuplicateOffers(t, stack, map[string]string{"10.0.0.4": "10.0.0.110"})
	if len(stack.dhcpHandlers[&cfg.Devices[0]].leases) != 0 {
		t.Fatal("absent conflict silently allocated a healthy lease")
	}
}
