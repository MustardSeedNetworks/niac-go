package snmp

import (
	"encoding/hex"
	"strings"
)

func parseMACBytes(mac string) []byte {
	// Remove colons, dashes, etc.
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")

	bytes, err := hex.DecodeString(mac)
	if err != nil || len(bytes) != MACAddressOctets {
		return []byte{0, 0, 0, 0, 0, 0}
	}

	return bytes
}

// getInterfaceSpeed returns interface speed based on interface name.
func getInterfaceSpeed(ifName string) uint64 {
	ifNameLower := strings.ToLower(ifName)

	switch {
	case strings.Contains(ifNameLower, "hundredgig") || strings.Contains(ifNameLower, "100g"):
		return Speed100Gbps // 100 Gbps
	case strings.Contains(ifNameLower, "fortygig") || strings.Contains(ifNameLower, "40g"):
		return Speed40Gbps // 40 Gbps
	case strings.Contains(ifNameLower, "twentyfivegig") || strings.Contains(ifNameLower, "25g"):
		return Speed25Gbps // 25 Gbps
	case strings.Contains(ifNameLower, "tengig") || strings.Contains(ifNameLower, "10g"):
		return Speed10Gbps // 10 Gbps
	case strings.Contains(ifNameLower, "fivegig") || strings.Contains(ifNameLower, "5g"):
		return Speed5Gbps // 5 Gbps
	case strings.Contains(ifNameLower, "twogig") || strings.Contains(ifNameLower, "2.5g"):
		return Speed2p5Gbps // 2.5 Gbps
	case strings.Contains(ifNameLower, "gigabit") || strings.Contains(ifNameLower, "ge") || strings.Contains(ifNameLower, "1g"):
		return NanosPerSecond // 1 Gbps
	case strings.Contains(ifNameLower, "fastethernet") || strings.Contains(ifNameLower, "fa"):
		return Speed100Mbps // 100 Mbps
	case strings.Contains(ifNameLower, "ethernet"):
		return NanosPerSecond // Default to 1 Gbps
	default:
		return NanosPerSecond // Default to 1 Gbps
	}
}

// getCapabilitiesBitfield returns LLDP/CDP capability bits based on device type.
func getCapabilitiesBitfield(deviceType string) int {
	// LLDP System Capabilities bitmap:
	// Bit 0: Other
	// Bit 1: Repeater
	// Bit 2: Bridge
	// Bit 3: WLAN Access Point
	// Bit 4: Router
	// Bit 5: Telephone
	// Bit 6: DOCSIS cable device
	// Bit 7: Station Only
	deviceTypeLower := strings.ToLower(deviceType)

	switch deviceTypeLower {
	case "router", "firewall":
		return CapabilityRouterBridge // Router + Bridge
	case "switch":
		return CapabilityBridge // Bridge
	case "ap", "access-point", "access_point":
		return CapabilityWLANAP // WLAN AP
	case "phone", "voip-phone", "voip_phone":
		return CapabilityTelephoneStation // Telephone + Station Only
	case "server", "host":
		return CapabilityStationOnly // Station Only
	default:
		return LLDPCapabilityOther
	}
}
