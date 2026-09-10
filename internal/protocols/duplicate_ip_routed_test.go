package protocols

import (
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestDuplicateIPRoutedRequiresSelectedInterface(t *testing.T) {
	stack, _ := isolationRoutedDHCP(t)
	fault := devicestate.InterfaceAddressFault{
		Interface: "inside", Type: devicestate.FaultDuplicateIP, Address: netip.MustParseAddr("10.10.200.3"),
	}
	if err := stack.SetInterfaceAddressFault("edge", fault); err == nil {
		t.Fatal("accepted off-attachment interface on attached router")
	}
	fault.Interface = "outside"
	if err := stack.SetInterfaceAddressFault("edge", fault); err != nil {
		t.Fatal(err)
	}
	handler := NewARPHandler(stack)
	if got := handler.targetDevices(net.IP(fault.Address.AsSlice()), 0); len(got) != 2 {
		t.Fatalf("selected attachment responder count=%d", len(got))
	}
	if got := handler.targetDevices(net.IP(fault.Address.AsSlice()), 300); len(got) != 1 {
		t.Fatal("unrelated VLAN acquired extra responder")
	}
	fault.Interface = "inside"
	fault.Address = netip.MustParseAddr("10.20.0.10")
	if err := stack.SetInterfaceAddressFault("edge", fault); err == nil {
		t.Fatal("accepted unobservable remote conflict")
	}
}

func TestDuplicateIPPreflightRejectsUnobservableInterface(t *testing.T) {
	stack, cfg := isolationRoutedDHCP(t)
	cfg.BehaviorTimelines = []config.BehaviorTimeline{{
		Name: "conflict", RepeatCount: 1,
		Phases: []config.BehaviorPhase{
			{
				Name:     "duplicate",
				Duration: time.Second,
				Faults: []config.BehaviorFault{
					{
						Device:    "edge",
						Interface: "inside",
						Type:      "duplicate_ip",
						Address:   netip.MustParseAddr("10.20.0.10"),
					},
				},
			},
		},
	}}
	if err := stack.ValidateBehaviorTargets(); err == nil {
		t.Fatal("preflight accepted remote conflict")
	}
}
