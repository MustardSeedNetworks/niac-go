package protocols

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func resourceFaultTestMIBs() []config.AddMib {
	return []config.AddMib{
		{OID: "1.3.6.1.2.1.25.3.3.1.2.1", Type: "INTEGER", Value: "18"},
		{OID: "1.3.6.1.2.1.25.2.3.1.2.2", Type: "OID", Value: "1.3.6.1.2.1.25.2.1.2"},
		{OID: "1.3.6.1.2.1.25.2.3.1.5.2", Type: "INTEGER", Value: "1003"},
		{OID: "1.3.6.1.2.1.25.2.3.1.6.2", Type: "INTEGER", Value: "201"},
		{OID: "1.3.6.1.2.1.25.2.3.1.2.3", Type: "OID", Value: "1.3.6.1.2.1.25.2.1.4"},
		{OID: "1.3.6.1.2.1.25.2.3.1.5.3", Type: "INTEGER", Value: "10000"},
		{OID: "1.3.6.1.2.1.25.2.3.1.6.3", Type: "INTEGER", Value: "3300"},
	}
}

func TestCommunityResourceFaultIsObservable(t *testing.T) {
	walk := filepath.Join(t.TempDir(), "cpu.walk")
	if err := os.WriteFile(walk, []byte(".1.3.6.1.2.1.25.3.3.1.2.1 = INTEGER: 18\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	device := faultTestDevice("edge-1")
	device.SNMPConfig.CommunityIncludes = []config.CommunityInclude{{Community: "resources", WalkFile: walk}}
	cfg := &config.Config{Devices: []config.Device{device}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	group := stack.snmpAgents[&cfg.Devices[0]]
	if group.resourceFaultObservable(devicestate.FaultCPUPercent, false) {
		t.Fatal("community-only resource advertised with v2 disabled")
	}
	if err := stack.SetDeviceFault("edge-1", devicestate.FaultCPUPercent, 90); err != nil {
		t.Fatal(err)
	}
	targets := stack.DeviceFaultTargets()
	if len(targets) != 1 || !slices.Contains(targets[0].Faults, devicestate.FaultCPUPercent) {
		t.Fatalf("community resource not advertised: %#v", targets)
	}
	value, err := stack.snmpAgents[&cfg.Devices[0]].Get("resources").HandleGet("1.3.6.1.2.1.25.3.3.1.2.1")
	if err != nil || value == nil || value.Type != gosnmp.Integer || value.Value != 90 {
		t.Fatalf("community resource = %#v, err=%v", value, err)
	}
}

func TestStackResourceFaultRequiresServedTelemetry(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		rows    bool
		want    bool
	}{
		{"served CPU", true, true, true},
		{"absent CPU", true, false, false},
		{"disabled SNMP", false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			device := faultTestDevice("edge-1")
			device.SNMPConfig.Enabled = &tc.enabled
			if tc.rows {
				device.SNMPConfig.AddMibs = []config.AddMib{{
					OID: "1.3.6.1.2.1.25.3.3.1.2.1", Type: "INTEGER", Value: "18",
				}}
			}
			cfg := &config.Config{Devices: []config.Device{device}}
			stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
			targets := stack.DeviceFaultTargets()
			if len(targets) != 1 || slices.Contains(targets[0].Faults, devicestate.FaultCPUPercent) != tc.want {
				t.Fatalf("advertised targets = %#v, CPU observable = %v", targets, tc.want)
			}
			err := stack.SetDeviceFault("edge-1", devicestate.FaultCPUPercent, 90)
			if tc.want && err != nil {
				t.Fatal(err)
			}
			if !tc.want && (!errors.Is(err, ErrFaultServiceAbsent) || len(stack.ActiveDeviceFaults()) != 0) {
				t.Fatalf("unobservable resource fault: error=%v active=%v", err, stack.ActiveDeviceFaults())
			}
		})
	}
}
