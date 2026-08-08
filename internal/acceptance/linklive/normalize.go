package linklive

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"unicode"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const (
	macHexLength              = 12
	defaultInterfaceMTU       = 1500
	defaultInterfaceSpeedMbps = 1_000
	fastEthernetSpeedMbps     = 100
	twoPointFiveGigSpeedMbps  = 2_500
	fiveGigSpeedMbps          = 5_000
	tenGigSpeedMbps           = 10_000
	twentyFiveGigSpeedMbps    = 25_000
	fortyGigSpeedMbps         = 40_000
	hundredGigSpeedMbps       = 100_000
)

// FromConfig creates stable device and link truth from a NIAC configuration.
func FromConfig(cfg *config.Config) AuthoredSnapshot {
	devices := allDevices(cfg)
	faults := authoredFaults(cfg.BehaviorTimelines)
	snapshot := AuthoredSnapshot{Devices: make([]AuthoredDevice, 0, len(devices))}
	byName := make(map[string]config.Device, len(devices))
	for _, device := range devices {
		byName[device.Name] = device
		snapshot.Devices = append(snapshot.Devices, authoredDevice(device, faults))
	}
	snapshot.Links = authoredLinks(devices, byName)
	return snapshot
}

// ParseTopology removes tenant-specific fields and normalizes host identity.
func ParseTopology(data []byte) (ObservedSnapshot, error) {
	var hosts []ObservedHost
	if err := json.Unmarshal(data, &hosts); err != nil {
		return ObservedSnapshot{}, errors.New("Link-Live topology returned invalid JSON")
	}
	for index := range hosts {
		hosts[index].IPv4 = hosts[index].DefaultAddr.IPv4
		hosts[index].MAC = normalizeMAC(hosts[index].MAC)
		for connection := range hosts[index].Connections {
			hosts[index].Connections[connection].MAC = normalizeMAC(
				hosts[index].Connections[connection].MAC,
			)
		}
	}
	return ObservedSnapshot{Hosts: hosts}, nil
}

func allDevices(cfg *config.Config) []config.Device {
	if len(cfg.Segments) == 0 {
		return cfg.Devices
	}
	var devices []config.Device
	for _, segment := range cfg.Segments {
		devices = append(devices, segment.Devices...)
	}
	return devices
}

func authoredDevice(device config.Device, faults map[string]authoredFaultExpectation) AuthoredDevice {
	addresses := make([]string, 0, len(device.IPAddresses))
	for _, address := range device.IPAddresses {
		if address.To4() != nil {
			addresses = append(addresses, address.String())
		}
	}
	authored := AuthoredDevice{
		Name: device.Name,
		Type: device.Type,
		MAC:  normalizeMAC(device.MACAddress.String()),
		IPv4: addresses,
	}
	if snmpEnabled(device) {
		authored.ServesSNMP = true
		authored.InterfaceInventoryComplete = !hasCapturedInterfaceInventory(device.SNMPConfig)
		authored.Interfaces = authoredInterfaces(device, authored.InterfaceInventoryComplete)
		for index := range authored.Interfaces {
			fault := faults[authoredFaultKey(device.Name, authored.Interfaces[index].Name)]
			authored.Interfaces[index].UtilizationDynamic = fault.utilization
			authored.Interfaces[index].ExpectZeroErrors = authored.InterfaceInventoryComplete && !fault.errors
			authored.Interfaces[index].ExpectZeroDiscards = authored.InterfaceInventoryComplete && !fault.discards
		}
	}
	return authored
}

type authoredFaultExpectation struct {
	utilization bool
	errors      bool
	discards    bool
}

func authoredFaults(timelines []config.BehaviorTimeline) map[string]authoredFaultExpectation {
	result := make(map[string]authoredFaultExpectation)
	for _, timeline := range timelines {
		for _, phase := range timeline.Phases {
			for _, traffic := range phase.Traffic {
				key := authoredFaultKey(traffic.Device, traffic.Interface)
				expectation := result[key]
				expectation.utilization = true
				result[key] = expectation
			}
			for _, fault := range phase.Faults {
				key := authoredFaultKey(fault.Device, fault.Interface)
				expectation := result[key]
				switch fault.Type {
				case "high_utilization":
					expectation.utilization = true
				case "fcs_errors", "interface_errors":
					expectation.errors = true
				case "packet_discards":
					expectation.discards = true
				}
				result[key] = expectation
			}
		}
	}
	return result
}

func authoredFaultKey(device, iface string) string {
	return strings.ToLower(strings.TrimSpace(device)) + "\x00" +
		strings.ToLower(strings.TrimSpace(iface))
}

func authoredInterfaces(device config.Device, complete bool) []AuthoredInterface {
	interfaces := make([]AuthoredInterface, 0, len(device.TrunkPorts)+len(device.Interfaces))
	indexes := make(map[string]int, cap(interfaces))
	if complete {
		for _, trunk := range device.TrunkPorts {
			appendAuthoredInterface(
				&interfaces,
				indexes,
				authoredInterface(config.Interface{Name: trunk.Interface}, true),
			)
		}
	}
	for _, iface := range device.Interfaces {
		appendAuthoredInterface(&interfaces, indexes, authoredInterface(iface, complete))
	}
	if complete && len(interfaces) == 0 {
		interfaces = append(
			interfaces,
			authoredInterface(config.Interface{Name: "Management"}, true),
		)
	}
	return interfaces
}

func appendAuthoredInterface(
	interfaces *[]AuthoredInterface,
	indexes map[string]int,
	iface AuthoredInterface,
) {
	if iface.Name == "" {
		return
	}
	if index, found := indexes[iface.Name]; found {
		(*interfaces)[index] = iface
		return
	}
	indexes[iface.Name] = len(*interfaces)
	*interfaces = append(*interfaces, iface)
}

func authoredInterface(iface config.Interface, applyDefaults bool) AuthoredInterface {
	interfaceType := effectiveInterfaceType(iface.Name, iface.Type)
	status := iface.OperStatus
	if applyDefaults && status == "" {
		status = "Up"
	}
	speed := iface.Speed
	if applyDefaults && speed == 0 {
		speed = effectiveInterfaceSpeed(iface.Name)
	}
	mtu := iface.MTU
	if applyDefaults && mtu == 0 {
		mtu = defaultInterfaceMTU
	}
	duplex := iface.Duplex
	if applyDefaults && duplex == "" && interfaceType == "ethernet" {
		duplex = "Full"
	}
	return AuthoredInterface{
		Name: iface.Name, Type: interfaceType, Status: status,
		SpeedMbps: speed, Duplex: duplex, MTU: mtu,
		UtilizationPercent: max(iface.InUtilization, iface.OutUtilization),
	}
}

func effectiveInterfaceType(name, configured string) string {
	if configured != "" {
		return strings.ToLower(configured)
	}
	lowerName := strings.ToLower(name)
	switch {
	case strings.HasPrefix(lowerName, "vlan"):
		return "l2vlan"
	case strings.HasPrefix(lowerName, "loopback"):
		return "loopback"
	case strings.HasPrefix(lowerName, "tunnel"):
		return "tunnel"
	default:
		return "ethernet"
	}
}

func effectiveInterfaceSpeed(name string) int {
	lowerName := strings.ToLower(name)
	switch {
	case strings.Contains(lowerName, "hundredgig"), strings.Contains(lowerName, "100g"):
		return hundredGigSpeedMbps
	case strings.Contains(lowerName, "fortygig"), strings.Contains(lowerName, "40g"):
		return fortyGigSpeedMbps
	case strings.Contains(lowerName, "twentyfivegig"), strings.Contains(lowerName, "25g"):
		return twentyFiveGigSpeedMbps
	case strings.Contains(lowerName, "tengig"), strings.Contains(lowerName, "10g"):
		return tenGigSpeedMbps
	case strings.Contains(lowerName, "twogig"), strings.Contains(lowerName, "2.5g"):
		return twoPointFiveGigSpeedMbps
	case strings.Contains(lowerName, "fivegig"), strings.Contains(lowerName, "5g"):
		return fiveGigSpeedMbps
	case strings.Contains(lowerName, "fastethernet"), strings.Contains(lowerName, "fa"):
		return fastEthernetSpeedMbps
	default:
		return defaultInterfaceSpeedMbps
	}
}

func hasCapturedInterfaceInventory(cfg config.SNMPConfig) bool {
	return cfg.SnmpAddr != nil || cfg.WalkFile != "" || len(cfg.WalkFiles) > 0 ||
		len(cfg.CommunityIncludes) > 0
}

func snmpEnabled(device config.Device) bool {
	v2Enabled := (device.SNMPConfig.Enabled == nil || *device.SNMPConfig.Enabled) &&
		strings.TrimSpace(device.SNMPConfig.Community) != ""
	v3Enabled := device.SNMPv3Config != nil && device.SNMPv3Config.Enabled &&
		len(device.SNMPv3Config.Users) > 0
	return v2Enabled || v3Enabled
}

func authoredLinks(devices []config.Device, byName map[string]config.Device) []AuthoredLink {
	var links []AuthoredLink
	seen := make(map[string]bool)
	for _, device := range devices {
		for _, trunk := range device.TrunkPorts {
			key := linkKey(device.Name, trunk.RemoteDevice)
			if trunk.RemoteDevice == "" || seen[key] {
				continue
			}
			seen[key] = true
			links = append(links, makeAuthoredLink(device, byName[trunk.RemoteDevice], trunk))
		}
	}
	return links
}

func makeAuthoredLink(source, target config.Device, trunk config.TrunkPort) AuthoredLink {
	link := AuthoredLink{
		Source: source.Name, Target: target.Name,
		SourceMAC: normalizeMAC(
			source.MACAddress.String(),
		), TargetMAC: normalizeMAC(target.MACAddress.String()),
		SourcePort: trunk.Interface, TargetPort: trunk.RemoteInterface,
		NativeVLAN: trunk.NativeVLAN,
	}
	for _, iface := range source.Interfaces {
		if iface.Name == trunk.Interface {
			link.SpeedMbps, link.Duplex = iface.Speed, iface.Duplex
			break
		}
	}
	return link
}

func linkKey(left, right string) string {
	values := []string{left, right}
	slices.Sort(values)
	return strings.Join(values, "|")
}

func normalizeMAC(value string) string {
	hex := strings.Map(func(char rune) rune {
		if unicode.Is(unicode.ASCII_Hex_Digit, char) {
			return unicode.ToLower(char)
		}
		return -1
	}, value)
	if len(hex) > macHexLength {
		return hex[len(hex)-macHexLength:]
	}
	return hex
}
