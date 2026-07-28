package linklive

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"unicode"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const macHexLength = 12

// FromConfig creates stable device and link truth from a NIAC configuration.
func FromConfig(cfg *config.Config) AuthoredSnapshot {
	devices := allDevices(cfg)
	snapshot := AuthoredSnapshot{Devices: make([]AuthoredDevice, 0, len(devices))}
	byName := make(map[string]config.Device, len(devices))
	for _, device := range devices {
		byName[device.Name] = device
		snapshot.Devices = append(snapshot.Devices, authoredDevice(device))
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

func authoredDevice(device config.Device) AuthoredDevice {
	addresses := make([]string, 0, len(device.IPAddresses))
	for _, address := range device.IPAddresses {
		if address.To4() != nil {
			addresses = append(addresses, address.String())
		}
	}
	return AuthoredDevice{
		Name: device.Name,
		Type: device.Type,
		MAC:  normalizeMAC(device.MACAddress.String()),
		IPv4: addresses,
	}
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
