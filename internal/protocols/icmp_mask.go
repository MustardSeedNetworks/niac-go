package protocols

import (
	"net"
	"net/netip"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func (h *ICMPHandler) effectiveAddressMask(device *config.Device, source net.IP) net.IP {
	state := h.stack.deviceStates[device]
	address, valid := netip.AddrFromSlice(source)
	if state != nil && valid {
		snapshot := state.Snapshot()
		iface, faulted := hostFaultInterface(snapshot, address.Unmap())
		if faulted {
			for _, effective := range snapshot.EffectiveHostNetwork().Interfaces {
				if effective.Name == iface.Name {
					return net.IP(net.CIDRMask(effective.Address.Bits(), effective.Address.Addr().BitLen()))
				}
			}
		}
	}
	return device.ICMPConfig.AddressMaskReply.To4()
}
