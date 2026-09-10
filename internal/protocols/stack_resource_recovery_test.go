package protocols

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestResourceFaultsRecoverWithoutChangingPeer(t *testing.T) {
	device, peer := faultTestDevice("resource-host"), faultTestDevice("peer")
	device.SNMPConfig.AddMibs = resourceFaultTestMIBs()
	peer.SNMPConfig.AddMibs = resourceFaultTestMIBs()
	cfg := &config.Config{Devices: []config.Device{device, peer}}
	original := NewStack(nil, cfg, logging.NewDebugConfig(0))
	for _, kind := range []devicestate.DeviceFaultType{
		devicestate.FaultCPUPercent, devicestate.FaultMemoryPercent, devicestate.FaultDiskPercent,
	} {
		if err := original.SetDeviceFault("resource-host", kind, 90); err != nil {
			t.Fatal(err)
		}
	}
	saved := original.ExportDeviceStates()
	restored := NewStack(nil, cfg, logging.NewDebugConfig(0))
	if err := restored.RestoreDeviceStates(saved); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, restored.ExportDeviceStates()) {
		t.Fatal("recovery changed durable state or replayed fault events")
	}
	assertRecoveredResourceRows(t, restored, &cfg.Devices[0], []string{"90", "902", "9000"})
	assertRecoveredResourceRows(t, restored, &cfg.Devices[1], []string{"18", "201", "3300"})
	restored.ClearAllDeviceFaults()
	assertRecoveredResourceRows(t, restored, &cfg.Devices[0], []string{"18", "201", "3300"})
	assertRecoveredResourceRows(t, restored, &cfg.Devices[1], []string{"18", "201", "3300"})
}

func assertRecoveredResourceRows(t *testing.T, stack *Stack, device *config.Device, want []string) {
	t.Helper()
	agent := stack.snmpAgents[device].baseAgent
	for i, oid := range []string{
		"1.3.6.1.2.1.25.3.3.1.2.1", "1.3.6.1.2.1.25.2.3.1.6.2", "1.3.6.1.2.1.25.2.3.1.6.3",
	} {
		value, err := agent.HandleGet(oid)
		if err != nil || value == nil || value.Type != gosnmp.Integer || fmt.Sprint(value.Value) != want[i] {
			t.Fatalf("%s %s = %#v (%v), want %s", device.Name, oid, value, err, want[i])
		}
	}
}
