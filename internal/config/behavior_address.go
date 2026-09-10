package config

import (
	"errors"
	"fmt"
	"net/netip"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// ErrBehaviorAddressTarget means an address fault lacks an eligible target or a peer owner in its network.
var ErrBehaviorAddressTarget = errors.New(
	"address fault requires an eligible target and a peer-owned address in the same network",
)

func validateBehaviorAddressFault(targets map[string]behaviorTarget, fault BehaviorFault) error {
	if fault.Interface != "" {
		return fmt.Errorf("%w: %q is device-scoped", ErrBehaviorFaultScope, fault.Type)
	}
	if fault.Value != 0 || !devicestate.ValidFaultAddress(fault.Address) {
		return fmt.Errorf("%w: %q requires only a unicast IPv4 address", ErrBehaviorFaultValue, fault.Type)
	}
	if err := validateBehaviorDevice(targets, fault.Device); err != nil {
		return err
	}
	server := targets[fault.Device]
	if server.device.DHCPConfig == nil {
		return fmt.Errorf("%w: %s does not serve DHCP", ErrBehaviorAddressTarget, fault.Device)
	}
	for name, peer := range targets {
		if name != fault.Device && peer.vlan == server.vlan && behaviorAddressPeer(server, peer, fault.Address) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrBehaviorAddressTarget, fault.Address)
}

func behaviorAddressPeer(server, peer behaviorTarget, address netip.Addr) bool {
	if !server.routed {
		for index := range max(len(peer.device.Interfaces), len(peer.device.IPAddresses)) {
			prefix := DeviceInterfaceAddress(peer.device, index)
			if prefix.Addr() == address && ValidIPv4Host(prefix, address) {
				return true
			}
		}
		return false
	}
	for _, iface := range peer.device.Interfaces {
		prefix, err := netip.ParsePrefix(iface.Address)
		if err != nil || prefix.Addr() != address || !ValidIPv4Host(prefix, address) {
			continue
		}
		if behaviorDeviceOnNetwork(server.device, iface.Network) {
			return true
		}
	}
	return false
}

func behaviorDeviceOnNetwork(device Device, network string) bool {
	if network == "" {
		return false
	}
	for _, iface := range device.Interfaces {
		if iface.Network == network {
			return true
		}
	}
	return false
}
