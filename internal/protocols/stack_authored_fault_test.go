package protocols

import (
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// An authored fault is the condition a scenario starts in, so it has to be true
// before anything is polled. Until this landed the only way to author one was a
// behavior phase, which meant a scenario could describe a broken network only as
// something that breaks a moment after the simulation starts -- and a pack whose
// whole job is to give a discovery tool something to find shipped healthy.
func TestAuthoredInterfaceFaultIsArmedBeforeStart(t *testing.T) {
	stack, device := newAuthoredFaultStack(
		[]config.InterfaceFault{{Type: "fcs_errors", Value: 25}},
		nil,
	)

	active := stack.ActiveInterfaceFaults()[device.Name]["Gi0/1"]
	if active[devicestate.FaultFCS] != 25 {
		t.Fatalf("authored fcs_errors = %d, want 25 with no Start() and no injection", active[devicestate.FaultFCS])
	}
}

func TestAuthoredDeviceFaultIsArmedBeforeStart(t *testing.T) {
	stack, device := newAuthoredFaultStack(nil, []config.DeviceFault{{Type: "latency", Value: 250}})

	active := stack.ActiveDeviceFaults()[device.Name]
	if active[devicestate.FaultLatency].Value != 250 {
		t.Fatalf("authored latency = %#v, want 250 ms", active[devicestate.FaultLatency])
	}
}

// A reload rebuilds every device store from scratch, so an authored fault that
// is only applied in NewStack silently disappears the first time an operator
// reloads the configuration -- the scenario would then differ from its own YAML.
func TestAuthoredFaultSurvivesReload(t *testing.T) {
	stack, device := newAuthoredFaultStack(
		[]config.InterfaceFault{{Type: "fcs_errors", Value: 25}},
		nil,
	)
	cfg := &config.Config{Devices: []config.Device{*device}}
	if err := stack.ReloadConfig(cfg); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}

	active := stack.ActiveInterfaceFaults()[device.Name]["Gi0/1"]
	if active[devicestate.FaultFCS] != 25 {
		t.Fatalf("after reload fcs_errors = %d, want the authored 25", active[devicestate.FaultFCS])
	}
}

func newAuthoredFaultStack(
	interfaceFaults []config.InterfaceFault,
	deviceFaults []config.DeviceFault,
) (*Stack, *config.Device) {
	device := config.Device{
		Name: "edge-1", IPAddresses: []net.IP{{192, 0, 2, 1}},
		Interfaces: []config.Interface{{
			Name: "Gi0/1", Address: "192.0.2.1/24", Speed: 100, Faults: interfaceFaults,
		}},
		TrunkPorts: []config.TrunkPort{{Interface: "Gi0/1"}},
		SNMPConfig: config.SNMPConfig{Community: "public"},
		Faults:     deviceFaults,
	}
	cfg := &config.Config{Devices: []config.Device{device}}

	return NewStack(nil, cfg, logging.NewDebugConfig(0)), &cfg.Devices[0]
}
