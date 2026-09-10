package snmp

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestResourceFaultRequiresObservableInventory(t *testing.T) {
	for _, tc := range []struct {
		name  string
		fault devicestate.DeviceFaultType
		rows  [][3]string
		want  bool
	}{
		{"absent CPU", devicestate.FaultCPUPercent, nil, false},
		{"CPU", devicestate.FaultCPUPercent, [][3]string{{hrProcessorLoadPrefix + "1", "INTEGER", "18"}}, true},
		{"wrong CPU type", devicestate.FaultCPUPercent, [][3]string{{hrProcessorLoadPrefix + "1", "STRING", "18"}}, false},
		{"invalid CPU index", devicestate.FaultCPUPercent, [][3]string{{hrProcessorLoadPrefix + "0", "INTEGER", "18"}}, false},
		{"memory missing capacity", devicestate.FaultMemoryPercent, [][3]string{
			{hrStorageTypePrefix + "2", "OID", hrStorageRAM},
			{hrStorageUsedPrefix + "2", "INTEGER", "20"},
		}, false},
		{"memory", devicestate.FaultMemoryPercent, [][3]string{
			{hrStorageTypePrefix + "2", "OID", hrStorageRAM},
			{hrStorageSizePrefix + "2", "INTEGER", "100"},
			{hrStorageUsedPrefix + "2", "INTEGER", "20"},
		}, true},
		{"memory is not disk", devicestate.FaultDiskPercent, [][3]string{
			{hrStorageTypePrefix + "2", "OID", hrStorageRAM},
			{hrStorageSizePrefix + "2", "INTEGER", "100"},
			{hrStorageUsedPrefix + "2", "INTEGER", "20"},
		}, false},
		{"disk", devicestate.FaultDiskPercent, [][3]string{
			{hrStorageTypePrefix + "2", "OID", hrStorageFixedDisk},
			{hrStorageSizePrefix + "2", "INTEGER", "100"},
			{hrStorageUsedPrefix + "2", "INTEGER", "20"},
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			device := &config.Device{Name: "resource-host"}
			state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
			agent := NewAgentWithState(device, state, AgentOptions{})
			for _, row := range tc.rows {
				if err := agent.AddMib(row[0], row[1], row[2]); err != nil {
					t.Fatal(err)
				}
			}
			agent.Reindex()
			if got := agent.ResourceFaultObservable(tc.fault); got != tc.want {
				t.Fatalf("ResourceFaultObservable(%s) = %v, want %v", tc.fault, got, tc.want)
			}
		})
	}
}
