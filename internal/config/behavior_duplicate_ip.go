package config

import (
	"fmt"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func validateBehaviorDuplicateIP(targets map[string]behaviorTarget, fault BehaviorFault) error {
	if err := validateBehaviorTarget(targets, fault.Device, fault.Interface); err != nil {
		return err
	}
	if fault.Value != 0 || !devicestate.ValidFaultAddress(fault.Address) {
		return fmt.Errorf("%w: duplicate_ip requires only a unicast IPv4 address", ErrBehaviorFaultValue)
	}
	target := targets[fault.Device]
	for _, iface := range target.device.Interfaces {
		if iface.Name == fault.Interface {
			target.device.Interfaces = []Interface{iface}
			break
		}
	}
	for name, peer := range targets {
		if name != fault.Device && peer.vlan == target.vlan && behaviorAddressPeer(target, peer, fault.Address) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s has no peer on interface %s", ErrBehaviorAddressTarget, fault.Address, fault.Interface)
}

func behaviorUsesAddress(kind string) bool {
	return kind == string(devicestate.FaultDuplicateDHCPOffer) || kind == string(devicestate.FaultDuplicateIP)
}
