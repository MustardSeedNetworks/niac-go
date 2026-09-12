package snmp

import (
	"encoding/binary"
	"encoding/hex"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/deviceclass"
)

const (
	cdpCapabilityRouter      = 0x01
	cdpCapabilitySwitch      = 0x08
	cdpCapabilityHost        = 0x10
	cdpCapabilityIGMP        = 0x20
	cdpCapabilityPhone       = 0x80
	cdpCapabilitiesByteCount = 4
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
	switch deviceclass.Parse(deviceType) {
	case deviceclass.Router, deviceclass.Layer3Switch, deviceclass.Firewall:
		return CapabilityRouterBridge // Router + Bridge
	case deviceclass.Switch:
		return CapabilityBridge // Bridge
	case deviceclass.AP, deviceclass.AccessPoint:
		return CapabilityWLANAP // WLAN AP
	case deviceclass.VoipPhone:
		return CapabilityTelephoneStation // Telephone + Station Only
	case deviceclass.Server, deviceclass.Host, deviceclass.Workstation,
		deviceclass.IoT, deviceclass.Printer:
		return CapabilityStationOnly // Station Only
	case deviceclass.Unknown:
		return LLDPCapabilityOther
	default:
		return LLDPCapabilityOther
	}
}

func cdpCapabilities(deviceType string) []byte {
	var capabilities uint32
	switch deviceclass.Parse(deviceType) {
	case deviceclass.Router:
		capabilities = cdpCapabilityRouter | cdpCapabilityIGMP
	case deviceclass.Layer3Switch:
		capabilities = cdpCapabilityRouter | cdpCapabilitySwitch | cdpCapabilityIGMP
	case deviceclass.Switch, deviceclass.AP, deviceclass.AccessPoint:
		capabilities = cdpCapabilitySwitch | cdpCapabilityIGMP
	case deviceclass.Firewall:
		// Had no case here and fell to host, the same hole the LLDP and CDP
		// TLVs carried (#2096). A firewall routes.
		capabilities = cdpCapabilityRouter
	case deviceclass.VoipPhone:
		capabilities = cdpCapabilityPhone | cdpCapabilityHost
	case deviceclass.Server, deviceclass.Host, deviceclass.Workstation,
		deviceclass.IoT, deviceclass.Printer, deviceclass.Unknown:
		capabilities = cdpCapabilityHost
	}
	encoded := make([]byte, cdpCapabilitiesByteCount)
	binary.BigEndian.PutUint32(encoded, capabilities)
	return encoded
}
