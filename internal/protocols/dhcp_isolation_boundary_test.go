package protocols

import (
	"fmt"
	"net"
	"testing"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestDHCPMutationsRequireOneServerIdentifier(t *testing.T) {
	for _, message := range []byte{DHCPDecline, DHCPRelease} {
		for _, selector := range []string{"missing", "short", "duplicate", "unspecified"} {
			t.Run(fmt.Sprintf("%d/%s", message, selector), func(t *testing.T) {
				assertRejectedDHCPMutation(t, message, selector)
			})
		}
	}
}

func assertRejectedDHCPMutation(t *testing.T, message byte, selector string) {
	t.Helper()
	_, cfg := isolationPair(t)
	cfg.Devices[1].DHCPConfig.PoolStart = cfg.Devices[0].DHCPConfig.PoolStart
	cfg.Devices[1].DHCPConfig.PoolEnd = cfg.Devices[0].DHCPConfig.PoolEnd
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	mac := net.HardwareAddr{2, 0, 0, 0, 1, 1}
	sendIsolationDHCP(t, stack, dhcpDiscover(mac))
	if len(isolationReplies(stack)) != 2 {
		t.Fatal("both servers must first offer the overlapping address")
	}
	request := dhcpRequest(mac, net.ParseIP("10.0.0.100"), 0)
	request.dhcp.ClientIP = net.ParseIP("10.0.0.100")
	request.dhcp.Options[0].Data[0] = message
	addInvalidDHCPSelector(request, selector)
	sendIsolationDHCP(t, stack, request)
	if len(isolationReplies(stack)) != 0 {
		t.Fatal("malformed mutation received an unexpected reply")
	}
	for index := range cfg.Devices {
		handler := stack.dhcpHandlers[&cfg.Devices[index]]
		if len(handler.leases) != 1 || len(handler.declined) != 0 {
			t.Fatalf(
				"server %s mutated: leases=%d declined=%d",
				cfg.Devices[index].Name,
				len(handler.leases),
				len(handler.declined),
			)
		}
	}
}

func addInvalidDHCPSelector(info *dhcpPacketInfo, selector string) {
	switch selector {
	case "short":
		info.dhcp.Options = append(
			info.dhcp.Options,
			layers.DHCPOption{Type: layers.DHCPOptServerID, Length: 3, Data: []byte{10, 0, 0}},
		)
	case "duplicate":
		selectIsolationServer(info, "10.0.0.2")
		selectIsolationServer(info, "10.0.0.3")
	case "unspecified":
		selectIsolationServer(info, "0.0.0.0")
	}
}

func TestDHCPRoutedServerRequiresActiveAttachment(t *testing.T) {
	stack, _ := isolationRoutedDHCP(t)
	server := stack.fabric.devicesByName["a"]
	store := stack.deviceStates[server]
	network := store.Snapshot().Network
	network.Interfaces[0].AdminUp = false
	network.Interfaces[0].OperUp = false
	store.ReplaceNetwork(network)
	sendIsolationDHCP(t, stack, dhcpDiscover(net.HardwareAddr{2, 0, 0, 0, 1, 1}))
	replies := isolationReplies(stack)
	if len(replies) != 1 || !dhcpOf(t, replies[0]).YourClientIP.Equal(net.ParseIP("10.10.200.110")) {
		t.Fatalf("inactive attachment produced %d responses, want only healthy peer", len(replies))
	}
	if len(stack.dhcpHandlers[server].leases) != 0 {
		t.Fatal("inactive server allocated a lease")
	}
	network.Interfaces[0].AdminUp = true
	network.Interfaces[0].OperUp = true
	store.ReplaceNetwork(network)
	sendIsolationDHCP(t, stack, dhcpDiscover(net.HardwareAddr{2, 0, 0, 0, 1, 2}))
	if len(isolationReplies(stack)) != 2 {
		t.Fatal("restored attachment did not resume both servers")
	}
}
