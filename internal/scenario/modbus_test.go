package scenario_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// The manufacturing plant's PLCs answer Modbus/TCP (AI-6), and nothing else in
// any pack opens the port.
func TestPacksServeModbusOnPLCs(t *testing.T) {
	t.Parallel()

	for _, pack := range scenario.Packs() {
		cfg := generatePack(t, pack.ID)
		controllers := 0
		for index := range cfg.Devices {
			device := &cfg.Devices[index]
			plc := device.Properties["role"] == "plc"
			modbus := device.ModbusConfig
			serving := modbus != nil && modbus.Enabled
			if serving != plc {
				t.Errorf(
					"%s: %s (role %q) serves Modbus = %v",
					pack.ID,
					device.Name,
					device.Properties["role"],
					serving,
				)
			}
			if !serving {
				continue
			}
			controllers++
			if modbus.HoldingRegisters != 100 || modbus.InputRegisters != 100 {
				t.Errorf("%s: %s has %d holding and %d input registers, want 100 of each",
					pack.ID, device.Name, modbus.HoldingRegisters, modbus.InputRegisters)
			}
		}
		if pack.ID == "manufacturing" && controllers == 0 {
			t.Error("manufacturing: no PLC answers Modbus")
		}
	}
}
