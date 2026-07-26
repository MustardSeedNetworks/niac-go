package protocols

import (
	"net"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

func (s *Stack) arpBindings() []snmp.ARPBinding {
	bindings := make([]snmp.ARPBinding, 0, len(s.config.Devices))
	for i := range s.config.Devices {
		device := &s.config.Devices[i]
		if len(device.MACAddress) == 0 {
			continue
		}
		for _, iface := range device.Interfaces {
			address, _, err := net.ParseCIDR(iface.Address)
			if err == nil && address.To4() != nil {
				bindings = append(bindings, snmp.ARPBinding{
					Address: address.To4(), MAC: device.MACAddress,
				})
			}
		}
	}
	return bindings
}
