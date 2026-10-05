package protocols

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// A pool may span the access switches of one site (niac-go#2505), so a re-pin
// can carry a tester from one switch to another on the running session.

const neighbourPoolPort = "GigabitEthernet1/0/1"

func twoSwitchPlacementConfig(pins ...config.AttachmentPin) *config.Config {
	cfg := placementConfig(pins...)
	cfg.Attachments[0].At = append(cfg.Attachments[0].At, config.AttachmentPort{
		Device: placementNeighbour, Ports: []string{neighbourPoolPort},
	})
	for i := range cfg.Devices {
		if cfg.Devices[i].Name == placementNeighbour {
			cfg.Devices[i].Interfaces = append(cfg.Devices[i].Interfaces, config.Interface{
				Name: neighbourPoolPort, VLANs: []int{placementDataVLAN}, OperStatus: "down",
			})
		}
	}
	return cfg
}

func TestRepinMovesAClientToAnotherSwitch(t *testing.T) {
	cfg := twoSwitchPlacementConfig()
	stack := NewStackWithTransport(idleTransport{}, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(compilePlacement(t, cfg))
	first, second := placementClient(1), placementClient(2)
	sendFrom(stack, first, "10.51.210.101")
	sendFrom(stack, second, "10.51.210.102")

	pin := config.AttachmentPin{MAC: first.String(), Device: placementNeighbour, Interface: neighbourPoolPort}
	if err := stack.RepinAttachedClient(compilePlacement(t, twoSwitchPlacementConfig(pin)), first); err != nil {
		t.Fatalf("RepinAttachedClient() = %v", err)
	}

	access, neighbour := stack.agentFor(t, placementAccess), stack.agentFor(t, placementNeighbour)
	if port, found := fdbPortName(t, access, first); found {
		t.Errorf("%s still reports the moved client on %q", placementAccess, port)
	}
	if got, _ := fdbPortName(t, neighbour, first); got != neighbourPoolPort {
		t.Errorf("%s FDB port = %q, want %q", placementNeighbour, got, neighbourPoolPort)
	}
	if got, _ := fdbPortName(t, access, second); got != "GigabitEthernet1/0/44" {
		t.Errorf("the client left behind moved to %q", got)
	}

	// The moved client was placed first, so its new switch is the one at the
	// other end of the cable and the only one that may advertise.
	if got, _ := stack.fabric.placement.advertisedPort(stack.fabric.portAdminUp); got.Device != placementNeighbour ||
		got.Interface != neighbourPoolPort {
		t.Errorf("LLDP names %s %s, want %s %s", got.Device, got.Interface, placementNeighbour, neighbourPoolPort)
	}
	if !stack.fabric.advertisesAtClient(stack.fabric.devicesByName[placementNeighbour]) {
		t.Errorf("%s may not advertise after the client moved onto it", placementNeighbour)
	}
	if stack.fabric.advertisesAtClient(stack.fabric.devicesByName[placementAccess]) {
		t.Errorf("%s still advertises after the earliest client left it", placementAccess)
	}
}
