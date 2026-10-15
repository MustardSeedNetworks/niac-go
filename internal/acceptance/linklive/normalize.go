package linklive

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"unicode"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
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
	pool := attachmentPoolPorts(cfg.Attachments)
	testerNetworks := attachmentNetworks(cfg)
	snapshot := AuthoredSnapshot{Devices: make([]AuthoredDevice, 0, len(devices))}
	byName := make(map[string]config.Device, len(devices))
	for _, device := range devices {
		byName[device.Name] = device
		authored := authoredDevice(device, faults, pool)
		authored.NameOffLink = announcesOnlyOverMDNS(device) && len(testerNetworks) > 0 &&
			!onAnyNetwork(device, testerNetworks)
		snapshot.Devices = append(snapshot.Devices, authored)
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

func authoredDevice(
	device config.Device,
	faults map[string]authoredFaultExpectation,
	pool map[string]bool,
) AuthoredDevice {
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
			iface := &authored.Interfaces[index]
			key := interfaceKey(device.Name, iface.Name)
			fault := faults[key]
			iface.UtilizationDynamic = fault.utilization
			iface.ExpectZeroErrors = iface.ExpectZeroErrors && !fault.errors
			iface.ExpectZeroDiscards = iface.ExpectZeroDiscards && !fault.discards
			iface.AttachmentPool = pool[key]
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
				key := interfaceKey(traffic.Device, traffic.Interface)
				expectation := result[key]
				expectation.utilization = true
				result[key] = expectation
			}
			for _, fault := range phase.Faults {
				key := interfaceKey(fault.Device, fault.Interface)
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

func attachmentPoolPorts(attachments []config.LogicalAttachment) map[string]bool {
	result := make(map[string]bool)
	for _, attachment := range attachments {
		for _, at := range attachment.At {
			for _, port := range at.Ports {
				result[interfaceKey(at.Device, port)] = true
			}
		}
	}
	return result
}

// attachmentNetworks lists the networks testers land on, as the fabric
// compiler resolves each attachment's pool.
func attachmentNetworks(cfg *config.Config) []string {
	if !fabric.IsRouted(cfg) {
		return nil
	}
	var networks []string
	for _, attachment := range fabric.CompileConfig(cfg).Topology.Attachments {
		if attachment.Network != "" && !slices.Contains(networks, attachment.Network) {
			networks = append(networks, attachment.Network)
		}
	}
	return networks
}

// announcesOnlyOverMDNS reports a device whose only name source is multicast
// DNS. A discovery tool reads sysName first and asks NetBIOS next, and both of
// those cross a router.
func announcesOnlyOverMDNS(device config.Device) bool {
	return device.SNMPConfig.SysName == "" &&
		(device.NetBIOSConfig == nil || !device.NetBIOSConfig.Enabled) &&
		device.MDNSConfig != nil && device.MDNSConfig.Enabled
}

func onAnyNetwork(device config.Device, networks []string) bool {
	for _, iface := range device.Interfaces {
		if slices.Contains(networks, iface.Network) {
			return true
		}
	}
	return false
}

func interfaceKey(device, iface string) string {
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
	// The agent serves dot3StatsDuplexStatus on ethernet interfaces only, so an
	// authored duplex anywhere else never reaches Link-Live.
	duplex := ""
	if interfaceType == "ethernet" {
		duplex = iface.Duplex
		if applyDefaults && duplex == "" {
			duplex = "Full"
		}
	}
	authored := AuthoredInterface{
		Name: iface.Name, Type: interfaceType, Status: status,
		SpeedMbps: speed, Duplex: duplex, MTU: mtu,
		UtilizationPercent: max(iface.InUtilization, iface.OutUtilization),
		ExpectZeroErrors:   applyDefaults, ExpectZeroDiscards: applyDefaults,
	}
	applyInterfaceFaults(&authored, iface.Faults)
	return authored
}

// applyInterfaceFaults carries the conditions a pack starts in onto the
// expectation. The runtime serves them from the first poll: a utilization
// fault adds its rate to the authored band, a carrier fault takes the port
// down, and an error or discard fault moves its counters, which Link-Live
// flags.
func applyInterfaceFaults(authored *AuthoredInterface, faults []config.InterfaceFault) {
	for _, fault := range faults {
		if fault.Value <= 0 {
			continue
		}
		switch devicestate.FaultType(fault.Type) {
		case devicestate.FaultUtilization:
			authored.UtilizationPercent += float64(fault.Value)
		case devicestate.FaultLinkDown, devicestate.FaultPoELoss:
			authored.Status = "Down"
		case devicestate.FaultFCS, devicestate.FaultInterface:
			authored.CounterFault = true
			authored.ExpectZeroErrors = false
		case devicestate.FaultDiscards:
			authored.CounterFault = true
			authored.ExpectZeroDiscards = false
		}
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
	case strings.HasPrefix(lowerName, "port-channel"):
		// The agent types a bundle ieee8023adLag and, like IOS, serves it no
		// dot3StatsDuplexStatus, so Link-Live reads its duplex as Unknown.
		return "ieee8023adlag"
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
	link.SourceMembers = bundleMembers(&source, trunk.Interface)
	link.TargetMembers = bundleMembers(&target, trunk.RemoteInterface)
	return link
}

func bundleMembers(device *config.Device, interfaceName string) []AuthoredMember {
	channel, found := config.PortChannelFor(device, interfaceName)
	if !found {
		return nil
	}
	members := make([]AuthoredMember, 0, len(channel.Members))
	for _, name := range channel.Members {
		speed := effectiveInterfaceSpeed(name)
		for _, iface := range device.Interfaces {
			if iface.Name == name && iface.Speed > 0 {
				speed = iface.Speed
				break
			}
		}
		members = append(members, AuthoredMember{Name: name, SpeedMbps: speed})
	}
	return members
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
