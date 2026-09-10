package snmp

import (
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestResourceFaultResponsesMarshal(t *testing.T) {
	for _, tc := range []struct {
		fault devicestate.DeviceFaultType
		oid   string
	}{
		{devicestate.FaultCPUPercent, hrProcessorLoadPrefix + "7"},
		{devicestate.FaultMemoryPercent, hrStorageUsedPrefix + "10"},
		{devicestate.FaultDiskPercent, hrStorageUsedPrefix + "20"},
	} {
		t.Run(string(tc.fault), func(t *testing.T) {
			agent, state := resourceTestAgent(t)
			for _, percent := range []int{1, 99, 100, 0} {
				if err := state.SetDeviceFault(tc.fault, percent); err != nil {
					t.Fatal(err)
				}
				value, err := agent.HandleGet(tc.oid)
				if err != nil {
					t.Fatal(err)
				}
				packet := &gosnmp.SnmpPacket{
					Version: gosnmp.Version2c, Community: "test", PDUType: gosnmp.GetResponse,
					Variables: []gosnmp.SnmpPDU{{Name: tc.oid, Type: value.Type, Value: value.Value}},
				}
				if _, marshalErr := packet.MarshalMsg(); marshalErr != nil {
					t.Fatalf("%s at %d%%: %v", tc.oid, percent, marshalErr)
				}
			}
		})
	}
}
