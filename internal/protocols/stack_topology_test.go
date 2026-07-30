package protocols

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicecli"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestRuntimeFabricTopologyTracksCLIState(t *testing.T) {
	cfg, compiled := runtimeTopologyFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(compiled)
	router := &cfg.Devices[0]
	session := devicecli.NewSession(stack.deviceStates[router], stack.staticRouteValidator(router))
	for _, command := range []string{
		"enable",
		"configure terminal",
		"interface inside",
		"ip address 10.30.0.1/24",
	} {
		if response := session.Execute(command); strings.HasPrefix(response.Output, "%") {
			t.Fatalf("%q output = %q", command, response.Output)
		}
	}

	topology, ok := stack.RuntimeFabricTopology()
	if !ok {
		t.Fatal("RuntimeFabricTopology() unavailable")
	}
	wantAddress := netip.MustParsePrefix("10.30.0.1/24")
	if topology.Interfaces[1].Address != wantAddress {
		t.Fatalf("inside address = %s, want %s", topology.Interfaces[1].Address, wantAddress)
	}
	if topology.Routes[1].Destination != netip.MustParsePrefix("10.30.0.0/24") {
		t.Fatalf("connected route = %s, want 10.30.0.0/24", topology.Routes[1].Destination)
	}
}

func TestRuntimeTopologyTracksCLIInterfaceShutdown(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{
		{
			Name: "left", Type: "switch",
			Interfaces: []config.Interface{{Name: "Gi0/1", AdminStatus: "up", OperStatus: "up"}},
			TrunkPorts: []config.TrunkPort{
				{Interface: "Gi0/1", RemoteDevice: "right", RemoteInterface: "Gi0/1"},
			},
		},
		{
			Name: "right", Type: "switch",
			Interfaces: []config.Interface{{Name: "Gi0/1", AdminStatus: "up", OperStatus: "up"}},
		},
	}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	session := devicecli.NewSession(stack.deviceStates[&cfg.Devices[0]], nil)
	for _, command := range []string{"enable", "configure terminal", "interface Gi0/1", "shutdown"} {
		if response := session.Execute(command); strings.HasPrefix(response.Output, "%") {
			t.Fatalf("%q output = %q", command, response.Output)
		}
	}

	graph := stack.RuntimeTopology()
	if len(graph.Links) != 1 || graph.Links[0].Status != "down" {
		t.Fatalf("RuntimeTopology() links = %#v", graph.Links)
	}
}

func TestRuntimeTopologyProjectsInterfaceFaultAsDegraded(t *testing.T) {
	left := faultTestDevice("left")
	left.TrunkPorts[0].RemoteDevice = "right"
	left.TrunkPorts[0].RemoteInterface = "Gi0/1"
	right := faultTestDevice("right")
	right.IPAddresses[0] = []byte{192, 0, 2, 2}
	right.Interfaces[0].Address = "192.0.2.2/24"
	cfg := &config.Config{Devices: []config.Device{left, right}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	if err := stack.SetInterfaceFault("left", "Gi0/1", devicestate.FaultUtilization, 85); err != nil {
		t.Fatalf("SetInterfaceFault() error = %v", err)
	}

	graph := stack.RuntimeTopology()
	if len(graph.Links) != 1 || graph.Links[0].Status != "degraded" ||
		graph.Links[0].Utilization != 85 {
		t.Fatalf("RuntimeTopology() links = %#v", graph.Links)
	}
}

func TestRuntimeTopologyProjectsReciprocalTargetFault(t *testing.T) {
	left := faultTestDevice("left")
	left.TrunkPorts[0].RemoteDevice = "right"
	left.TrunkPorts[0].RemoteInterface = "Gi0/1"
	right := faultTestDevice("right")
	right.IPAddresses[0] = []byte{192, 0, 2, 2}
	right.Interfaces[0].Address = "192.0.2.2/24"
	right.TrunkPorts[0].RemoteDevice = "left"
	right.TrunkPorts[0].RemoteInterface = "Gi0/1"
	cfg := &config.Config{Devices: []config.Device{left, right}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	if err := stack.SetInterfaceFault("right", "Gi0/1", devicestate.FaultUtilization, 85); err != nil {
		t.Fatalf("SetInterfaceFault() error = %v", err)
	}

	graph := stack.RuntimeTopology()
	if len(graph.Links) != 1 || graph.Links[0].Status != "degraded" ||
		graph.Links[0].Utilization != 85 {
		t.Fatalf("RuntimeTopology() links = %#v", graph.Links)
	}
}

func runtimeTopologyFixture(t *testing.T) (*config.Config, *fabric.Topology) {
	t.Helper()
	cfg := &config.Config{
		Networks: []config.Network{
			{Name: "attachment", Subnet: "10.10.200.0/24"},
			{Name: "inside", Subnet: "10.20.0.0/24"},
		},
		Attachments: []config.LogicalAttachment{{Name: "tester", Network: "attachment"}},
		Devices: []config.Device{{
			Name: "edge", Type: "router", MACAddress: mustForwardingMAC(t, "02:00:00:00:00:01"),
			Interfaces: []config.Interface{
				{Name: "outside", Network: "attachment", Address: "10.10.200.1/24"},
				{Name: "inside", Network: "inside", Address: "10.20.0.1/24"},
			},
		}},
	}
	report := fabric.Compile(cfg, fabric.Binding{
		Attachment: "tester", Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
		PolicyApproved: true,
	})
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	return cfg, &report.Topology
}
