package scenario

import "github.com/MustardSeedNetworks/niac-go/internal/converter"

// packModbusRegisters is each register block of a pack PLC: one hundred
// holding and one hundred input registers, the size of the tag arrays a
// controller maps onto the Modbus address space for its SCADA master.
const packModbusRegisters = 100

// packModbus opens Modbus/TCP on the plant floor's PLCs, which the SCADA
// historian and the HMIs poll for process values.
func packModbus(role string) *converter.ModbusConfig {
	if role != "plc" {
		return nil
	}

	return &converter.ModbusConfig{
		Enabled: true, HoldingRegisters: packModbusRegisters, InputRegisters: packModbusRegisters,
	}
}
