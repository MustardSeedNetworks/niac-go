package snmp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

const resourceWalk = `
.1.3.6.1.2.1.25.3.3.1.2.7 = INTEGER: 18
.1.3.6.1.2.1.25.3.3.1.2.8 = INTEGER: 23
.1.3.6.1.2.1.25.2.3.1.2.10 = OID: .1.3.6.1.2.1.25.2.1.2
.1.3.6.1.2.1.25.2.3.1.4.10 = INTEGER: 4096
.1.3.6.1.2.1.25.2.3.1.5.10 = INTEGER: 1003
.1.3.6.1.2.1.25.2.3.1.6.10 = INTEGER: 201
.1.3.6.1.2.1.25.2.3.1.2.20 = OID: .1.3.6.1.2.1.25.2.1.4
.1.3.6.1.2.1.25.2.3.1.4.20 = INTEGER: 4096
.1.3.6.1.2.1.25.2.3.1.5.20 = INTEGER: 2147483647
.1.3.6.1.2.1.25.2.3.1.6.20 = INTEGER: 900
.1.3.6.1.2.1.25.2.3.1.2.30 = OID: .1.3.6.1.2.1.25.2.1.3
.1.3.6.1.2.1.25.2.3.1.5.30 = INTEGER: 500
.1.3.6.1.2.1.25.2.3.1.6.30 = INTEGER: 50
`

func TestCapturedResourceFaultsRestoreExactBaseline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resources.walk")
	if err := os.WriteFile(path, []byte(resourceWalk), 0o600); err != nil {
		t.Fatal(err)
	}
	device := &config.Device{Name: "resource-host", SNMPConfig: config.SNMPConfig{WalkFile: path}}
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	agent := NewAgentWithState(device, state, AgentOptions{})
	if err := agent.LoadWalkFile(path); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		fault    devicestate.DeviceFaultType
		oid      string
		baseline string
		armed    string
	}{
		{devicestate.FaultCPUPercent, "1.3.6.1.2.1.25.3.3.1.2.7", "18", "99"},
		{devicestate.FaultCPUPercent, "1.3.6.1.2.1.25.3.3.1.2.8", "23", "99"},
		{devicestate.FaultMemoryPercent, "1.3.6.1.2.1.25.2.3.1.6.10", "201", "992"},
		{devicestate.FaultDiskPercent, "1.3.6.1.2.1.25.2.3.1.6.20", "900", "2126008810"},
	} {
		t.Run(tc.oid, func(t *testing.T) {
			assertResourceValue(t, agent, tc.oid, tc.baseline)
			if err := state.SetDeviceFault(tc.fault, 99); err != nil {
				t.Fatal(err)
			}
			assertResourceValue(t, agent, tc.oid, tc.armed)
			assertResourceValue(t, agent, "1.3.6.1.2.1.25.2.3.1.6.30", "50")
			assertResourceValue(t, agent, "1.3.6.1.2.1.25.2.3.1.5.20", "2147483647")
			if err := state.SetDeviceFault(tc.fault, 0); err != nil {
				t.Fatal(err)
			}
			assertResourceValue(t, agent, tc.oid, tc.baseline)
		})
	}
}

func assertResourceValue(t *testing.T, agent *Agent, oid, want string) {
	t.Helper()
	value, err := agent.HandleGet(oid)
	if err != nil {
		t.Fatal(err)
	}
	if value == nil || value.Type != gosnmp.Integer || oidValueString(value) != want {
		t.Fatalf("%s = %#v, want INTEGER %s", oid, value, want)
	}
}

func TestResourceFaultFinalizationHonorsAddMibOverrides(t *testing.T) {
	for _, tc := range []struct {
		name     string
		oid      string
		fault    devicestate.DeviceFaultType
		baseline string
		armed    string
	}{
		{"cpu", hrProcessorLoadPrefix + "7", devicestate.FaultCPUPercent, "31", "50"},
		{"memory", hrStorageUsedPrefix + "10", devicestate.FaultMemoryPercent, "77", "1001"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			agent, state := resourceTestAgent(t)
			if err := state.SetDeviceFault(tc.fault, 50); err != nil {
				t.Fatal(err)
			}
			if err := agent.AddMib(hrStorageSizePrefix+"10", "INTEGER", "2003"); err != nil {
				t.Fatal(err)
			}
			if err := agent.AddMib(tc.oid, "INTEGER", tc.baseline); err != nil {
				t.Fatal(err)
			}
			agent.Reindex()
			assertResourceValue(t, agent, tc.oid, tc.armed)
			agent.Reindex()
			assertResourceValue(t, agent, tc.oid, tc.armed)
			if err := state.SetDeviceFault(tc.fault, 0); err != nil {
				t.Fatal(err)
			}
			assertResourceValue(t, agent, tc.oid, tc.baseline)
		})
	}
}

func TestResourceFaultFinalizationRefreshesAuthoredCapacity(t *testing.T) {
	agent, state := resourceTestAgent(t)
	if err := state.SetDeviceFault(devicestate.FaultMemoryPercent, 50); err != nil {
		t.Fatal(err)
	}
	if err := agent.AddMib(hrStorageSizePrefix+"10", "INTEGER", "2003"); err != nil {
		t.Fatal(err)
	}
	agent.Reindex()
	assertResourceValue(t, agent, hrStorageUsedPrefix+"10", "1001")
	if err := state.SetDeviceFault(devicestate.FaultMemoryPercent, 0); err != nil {
		t.Fatal(err)
	}
	assertResourceValue(t, agent, hrStorageUsedPrefix+"10", "201")
}

func TestResourceFaultClearResumesAuthoredDynamicCallback(t *testing.T) {
	agent, state := resourceTestAgent(t)
	const oid = hrProcessorLoadPrefix + "7"
	if err := agent.AddMib(oid, "INTEGER", "varimib((100000 100))"); err != nil {
		t.Fatal(err)
	}
	agent.startTime = time.Now()
	agent.uptimeBase = 310 * time.Second
	agent.Reindex()
	assertResourceValue(t, agent, oid, "31")
	if err := state.SetDeviceFault(devicestate.FaultCPUPercent, 99); err != nil {
		t.Fatal(err)
	}
	agent.uptimeBase = 710 * time.Second
	agent.Reindex()
	agent.Reindex()
	assertResourceValue(t, agent, oid, "99")
	if err := state.SetDeviceFault(devicestate.FaultCPUPercent, 0); err != nil {
		t.Fatal(err)
	}
	assertResourceValue(t, agent, oid, "71")
	agent.uptimeBase = 310 * time.Second
	assertResourceValue(t, agent, oid, "31")
}

func resourceTestAgent(t *testing.T) (*Agent, *devicestate.Store) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "resources.walk")
	if err := os.WriteFile(path, []byte(resourceWalk), 0o600); err != nil {
		t.Fatal(err)
	}
	state := devicestate.NewStore(devicestate.Identity{Hostname: "resource-host"})
	agent := NewAgentWithState(&config.Device{Name: "resource-host"}, state, AgentOptions{})
	if err := agent.LoadWalkFile(path); err != nil {
		t.Fatal(err)
	}
	return agent, state
}

func TestResourceFaultCombinesSplitWalkInventory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resources.walk")
	state := devicestate.NewStore(devicestate.Identity{Hostname: "split-host"})
	agent := NewAgentWithState(&config.Device{Name: "split-host"}, state, AgentOptions{})
	const usedOID = "1.3.6.1.2.1.25.2.3.1.6.10"
	parts := []string{
		"." + usedOID + " = INTEGER: 201\n",
		".1.3.6.1.2.1.25.2.3.1.2.10 = OID: .1.3.6.1.2.1.25.2.1.2\n.1.3.6.1.2.1.25.2.3.1.5.10 = INTEGER: 1003\n",
	}
	for _, part := range parts {
		if err := os.WriteFile(path, []byte(part), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := agent.LoadWalkFile(path); err != nil {
			t.Fatal(err)
		}
	}
	if err := state.SetDeviceFault(devicestate.FaultMemoryPercent, 50); err != nil {
		t.Fatal(err)
	}
	assertResourceValue(t, agent, usedOID, "501")
	updated := strings.Replace(parts[1], "1003", "2003", 1)
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := agent.LoadWalkFile(path); err != nil {
		t.Fatal(err)
	}
	assertResourceValue(t, agent, usedOID, "1001")
	if err := state.SetDeviceFault(devicestate.FaultMemoryPercent, 0); err != nil {
		t.Fatal(err)
	}
	assertResourceValue(t, agent, usedOID, "201")
}
