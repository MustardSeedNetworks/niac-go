package config

// IsRoutedTopologyLink reports whether a peer link carries no Layer 2 VLAN state.
func IsRoutedTopologyLink(deviceType string, port TrunkPort) bool {
	if port.FDBOnly || len(port.VLANs) != 0 || port.NativeVLAN != 0 {
		return false
	}
	switch deviceType {
	case "router", "firewall", "layer3-switch":
		return true
	default:
		return false
	}
}
