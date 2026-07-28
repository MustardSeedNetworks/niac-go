package snmp

import (
	"net"
	"strings"
)

// ARPBinding is an authoritative IP-to-MAC association from the simulated fleet.
type ARPBinding struct {
	Address net.IP
	MAC     net.HardwareAddr
}

type arpInterface struct {
	index   int
	network *net.IPNet
	prefix  int
}

// SynthesizeARPTable publishes directly connected fleet devices in a router's
// standard IP-MIB tables so managers can correlate routed IPs with bridge FDBs.
func (a *Agent) SynthesizeARPTable(bindings []ARPBinding) {
	if a == nil || a.device == nil || !isForwardingDevice(a.device.Type) {
		return
	}
	interfaces := a.connectedARPInterfaces()
	changed := false
	for _, binding := range bindings {
		iface, ok := bestARPInterface(interfaces, binding.Address)
		if !ok || a.isLocalAddress(binding.Address) || len(binding.MAC) == 0 {
			continue
		}
		a.registerARPEntry(iface.index, binding.Address, binding.MAC)
		changed = true
	}
	if changed {
		a.mib.Reindex()
	}
}

func isForwardingDevice(deviceType string) bool {
	switch strings.ToLower(strings.TrimSpace(deviceType)) {
	case "router", "layer3-switch", "firewall":
		return true
	default:
		return false
	}
}

func (a *Agent) connectedARPInterfaces() []arpInterface {
	result := make([]arpInterface, 0, len(a.device.Interfaces))
	for _, iface := range a.device.Interfaces {
		ip, network, err := net.ParseCIDR(iface.Address)
		if err != nil || ip.To4() == nil {
			continue
		}
		index, ok := a.InterfaceIndex(iface.Name)
		if !ok {
			continue
		}
		prefix, _ := network.Mask.Size()
		result = append(result, arpInterface{index: index, network: network, prefix: prefix})
	}
	return result
}

func bestARPInterface(interfaces []arpInterface, address net.IP) (arpInterface, bool) {
	var best arpInterface
	found := false
	for _, iface := range interfaces {
		if iface.network.Contains(address) && (!found || iface.prefix > best.prefix) {
			best, found = iface, true
		}
	}
	return best, found
}

func (a *Agent) isLocalAddress(address net.IP) bool {
	for _, iface := range a.device.Interfaces {
		local, _, err := net.ParseCIDR(iface.Address)
		if err == nil && local.Equal(address) {
			return true
		}
	}
	return false
}
