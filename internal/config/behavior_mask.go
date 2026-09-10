package config

import (
	"fmt"
	"net/netip"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func validateBehaviorMask(targets map[string]behaviorTarget, fault BehaviorFault) error {
	if err := validateBehaviorTarget(targets, fault.Device, fault.Interface); err != nil {
		return err
	}
	if fault.PrefixBits < 0 || fault.PrefixBits > 32 || fault.Value != 0 || fault.Address.IsValid() {
		return fmt.Errorf("%w: bad_mask requires only prefix_bits from 0 through 32", ErrBehaviorFaultValue)
	}
	device := targets[fault.Device].device
	switch device.Type {
	case "router", "layer3-switch", "firewall":
		return fmt.Errorf("%w: bad_mask requires a host interface", ErrBehaviorFaultScope)
	}
	for _, iface := range device.Interfaces {
		if iface.Name != fault.Interface {
			continue
		}
		prefix, err := netip.ParsePrefix(iface.Address)
		if err == nil && devicestate.ValidFaultAddress(prefix.Addr()) {
			return nil
		}
	}
	return fmt.Errorf("%w: bad_mask requires an IPv4 interface address", ErrBehaviorFaultScope)
}
