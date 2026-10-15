package scenario_test

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// The manufacturing plant's PLCs answer EtherNet/IP List Identity (AI-7) as
// ControlLogix controllers, each with its own serial number, and nothing else
// in any pack opens the port.
func testPacksServeEtherNetIPOnPLCs(t *testing.T, fixture *packFixture) {
	t.Parallel()

	for _, pack := range scenario.Packs() {
		cfg := fixture.configByID(t, pack.ID)
		serials := map[uint32]string{}
		for index := range cfg.Devices {
			device := &cfg.Devices[index]
			plc := device.Properties["role"] == "plc"
			enip := device.EtherNetIPConfig
			serving := enip != nil && enip.Enabled
			if serving != plc {
				t.Errorf("%s: %s (role %q) serves EtherNet/IP = %v",
					pack.ID, device.Name, device.Properties["role"], serving)
			}
			if !serving {
				continue
			}
			if enip.VendorID != 1 || enip.DeviceType != 14 || enip.ProductName != "1756-L83E/B" {
				t.Errorf("%s: %s identifies as vendor %d, type %d, %q; want a Rockwell PLC, 1756-L83E/B",
					pack.ID, device.Name, enip.VendorID, enip.DeviceType, enip.ProductName)
			}
			if other, taken := serials[enip.SerialNumber]; taken {
				t.Errorf("%s: %s and %s share serial number %08x", pack.ID, other, device.Name, enip.SerialNumber)
			}
			serials[enip.SerialNumber] = device.Name
		}
		if pack.ID == "manufacturing" && len(serials) == 0 {
			t.Error("manufacturing: no PLC answers EtherNet/IP")
		}
	}
}
