package protocols

import (
	"slices"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestV3OnlyResourceFaultIsObservable(t *testing.T) {
	device := faultTestDevice("v3-resource-host")
	v2Enabled := false
	device.SNMPConfig.Enabled = &v2Enabled
	device.SNMPConfig.AddMibs = resourceFaultTestMIBs()
	device.SNMPv3Config = &config.SNMPv3Config{
		Enabled: true, Users: []config.SNMPv3User{{Username: "resource-reader"}},
	}
	cfg := &config.Config{Devices: []config.Device{device}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	group := stack.snmpAgents[&cfg.Devices[0]]
	if group == nil || group.v3 == nil || group.v3Agent == nil {
		t.Fatal("v3-only agent was not initialized")
	}
	targets := stack.DeviceFaultTargets()
	if len(targets) != 1 || !slices.Contains(targets[0].Faults, devicestate.FaultCPUPercent) {
		t.Fatalf("v3-only resource not advertised: %#v", targets)
	}
	if err := stack.SetDeviceFault(device.Name, devicestate.FaultCPUPercent, 90); err != nil {
		t.Fatal(err)
	}
	value, err := group.v3Agent.HandleGet("1.3.6.1.2.1.25.3.3.1.2.1")
	if err != nil || value == nil || value.Type != gosnmp.Integer || value.Value != 90 {
		t.Fatalf("v3-only resource = %#v, err=%v", value, err)
	}
}
