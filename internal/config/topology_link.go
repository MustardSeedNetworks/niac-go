package config

import "github.com/MustardSeedNetworks/niac-go/internal/deviceclass"

// IsRoutedTopologyLink reports whether a peer link carries no Layer 2 VLAN state.
func IsRoutedTopologyLink(deviceType string, port TrunkPort) bool {
	if port.FDBOnly || len(port.VLANs) != 0 || port.NativeVLAN != 0 {
		return false
	}
	return deviceclass.RoutesIP(deviceclass.Parse(deviceType))
}
