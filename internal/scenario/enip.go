package scenario

import (
	"hash/fnv"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// The Identity object of the ControlLogix 5580 the plant floor's PLCs are
// profiled as: Rockwell Automation's vendor ID, the programmable logic
// controller device type and an L83E on firmware 33.011.
const (
	packENIPVendorRockwell = 1
	packENIPDeviceTypePLC  = 14
	packENIPProductCode    = 166
	packENIPRevisionMajor  = 33
	packENIPRevisionMinor  = 11
	packENIPProductName    = "1756-L83E/B"
)

// packEtherNetIP answers List Identity on the plant floor's PLCs, the query a
// commissioning tool or an asset inventory names a controller by. The serial
// number is the device name's hash, so every controller reports its own.
func packEtherNetIP(role, name string) *converter.EtherNetIPConfig {
	if role != "plc" {
		return nil
	}
	serial := fnv.New32a()
	_, _ = serial.Write([]byte(name))

	return &converter.EtherNetIPConfig{
		Enabled: true, VendorID: packENIPVendorRockwell, DeviceType: packENIPDeviceTypePLC,
		ProductCode: packENIPProductCode, RevisionMajor: packENIPRevisionMajor,
		RevisionMinor: packENIPRevisionMinor, SerialNumber: serial.Sum32(),
		ProductName: packENIPProductName,
	}
}
