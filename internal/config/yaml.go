package config

import (
	"errors"
	"net"
	"strconv"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// MarshalConfigYAML serializes the complete runtime configuration through the
// canonical YAML DTO used by the loader.
func MarshalConfigYAML(cfg *Config) ([]byte, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	return yaml.Marshal(configToYAML(cfg))
}

func configToYAML(cfg *Config) converter.Config {
	out := converter.Config{
		IncludePath: cfg.IncludePath,
		Devices:     devicesToYAML(cfg.Devices),
		Networks:    networksToYAML(cfg.Networks),
		Attachments: attachmentsToYAML(cfg.Attachments),
		Segments:    segmentsToYAML(cfg.Segments),
	}
	if cfg.CapturePlayback != nil {
		out.CapturePlaybacks = []converter.CapturePlayback{{
			FileName:  cfg.CapturePlayback.FileName,
			LoopTime:  cfg.CapturePlayback.LoopTime,
			ScaleTime: cfg.CapturePlayback.ScaleTime,
		}}
	}
	if cfg.DiscoveryProtocols != nil {
		out.DiscoveryProtocols = discoveryProtocolsToYAML(cfg.DiscoveryProtocols)
	}
	return out
}

func networksToYAML(networks []Network) []converter.Network {
	out := make([]converter.Network, len(networks))
	for i, network := range networks {
		out[i] = converter.Network{
			Name: network.Name, Subnet: network.Subnet, VirtualVLAN: network.VirtualVLAN,
		}
	}
	return out
}

func attachmentsToYAML(attachments []LogicalAttachment) []converter.LogicalAttachment {
	out := make([]converter.LogicalAttachment, len(attachments))
	for i, attachment := range attachments {
		out[i] = converter.LogicalAttachment{Name: attachment.Name, Connect: attachment.Network}
	}
	return out
}

func segmentsToYAML(segments []Segment) []converter.Segment {
	out := make([]converter.Segment, len(segments))
	for i, segment := range segments {
		tag := strconv.Itoa(segment.Tag)
		if segment.Tag == UntaggedTag {
			tag = "untagged"
		}
		out[i] = converter.Segment{Tag: converter.VLANTag(tag)}
		if segment.ConfigPath != "" {
			out[i].Config = segment.ConfigPath
		} else {
			out[i].Devices = devicesToYAML(segment.Devices)
		}
	}
	return out
}

func devicesToYAML(devices []Device) []converter.Device {
	out := make([]converter.Device, len(devices))
	for i := range devices {
		out[i] = deviceToYAML(&devices[i])
	}
	return out
}

func deviceToYAML(device *Device) converter.Device {
	out := converter.Device{
		Name:          device.Name,
		Type:          device.Type,
		MAC:           hardwareAddrString(device.MACAddress),
		Vendor:        device.MACVendor,
		MACSuffix:     device.MACSuffix,
		IPs:           ipStrings(device.IPAddresses),
		VLAN:          device.VLAN,
		MapToIP:       ipString(device.MapToIP),
		Babble:        device.Babble,
		TTL:           ttlToYAML(device.TTLConfig),
		SnmpAgent:     snmpToYAML(&device.SNMPConfig),
		Dhcp:          dhcpToYAML(device.DHCPConfig),
		DNS:           dnsToYAML(device.DNSConfig),
		Lldp:          lldpToYAML(device.LLDPConfig),
		Cdp:           cdpToYAML(device.CDPConfig),
		Edp:           edpToYAML(device.EDPConfig),
		Fdp:           fdpToYAML(device.FDPConfig),
		Stp:           stpToYAML(device.STPConfig),
		HTTP:          httpToYAML(device.HTTPConfig),
		Ftp:           ftpToYAML(device.FTPConfig),
		Netbios:       netbiosToYAML(device.NetBIOSConfig),
		Snmpv3:        snmpv3ToYAML(device.SNMPv3Config),
		Icmp:          icmpToYAML(device.ICMPConfig),
		Icmpv6:        icmpv6ToYAML(device.ICMPv6Config),
		Dhcpv6:        dhcpv6ToYAML(device.DHCPv6Config),
		OSFingerprint: osFingerprintToYAML(device.OSFingerprintConfig),
		SSH:           sshToYAML(device.SSHConfig),
		Syslog:        syslogToYAML(device.SyslogConfig),
		IPerf3:        iperf3ToYAML(device.IPerf3),
		Reflector:     reflectorToYAML(device.ReflectorConfig),
		Interfaces:    interfacesToYAML(device.Interfaces),
		Routes:        routesToYAML(device.Routes),
		TrunkPorts:    trunkPortsToYAML(device.TrunkPorts),
		PortChannels:  portChannelsToYAML(device.PortChannels),
		Properties:    device.Properties,
	}
	if device.MACVendor != "" {
		out.MAC = ""
	}
	return out
}

func interfacesToYAML(interfaces []Interface) []converter.Interface {
	out := make([]converter.Interface, len(interfaces))
	for i, iface := range interfaces {
		out[i] = converter.Interface{
			Name: iface.Name, Network: iface.Network, Address: iface.Address,
			Speed: iface.Speed, Duplex: iface.Duplex, AdminStatus: iface.AdminStatus,
			OperStatus: iface.OperStatus, Description: iface.Description, VLANs: iface.VLANs,
		}
	}
	return out
}

func routesToYAML(routes []Route) []converter.Route {
	out := make([]converter.Route, len(routes))
	for i, route := range routes {
		out[i] = converter.Route{
			Destination: route.Destination, Via: route.Via, NextHop: route.NextHop,
		}
	}
	return out
}

func trunkPortsToYAML(ports []TrunkPort) []converter.TrunkPort {
	out := make([]converter.TrunkPort, len(ports))
	for i, port := range ports {
		out[i] = converter.TrunkPort{
			Interface: port.Interface, VLANs: port.VLANs, NativeVLAN: port.NativeVLAN,
			RemoteDevice: port.RemoteDevice, RemoteInterface: port.RemoteInterface, FDBOnly: port.FDBOnly,
		}
	}
	return out
}

func portChannelsToYAML(channels []PortChannel) []converter.PortChannel {
	out := make([]converter.PortChannel, len(channels))
	for i, channel := range channels {
		out[i] = converter.PortChannel{ID: channel.ID, Members: channel.Members, Mode: channel.Mode}
	}
	return out
}

func discoveryProtocolsToYAML(protocols *DiscoveryProtocols) *converter.DiscoveryProtocols {
	return &converter.DiscoveryProtocols{
		LLDP: protocolToYAML(protocols.LLDP), CDP: protocolToYAML(protocols.CDP),
		EDP: protocolToYAML(protocols.EDP), FDP: protocolToYAML(protocols.FDP),
	}
}

func protocolToYAML(protocol *ProtocolConfig) *converter.ProtocolConfig {
	if protocol == nil {
		return nil
	}
	return &converter.ProtocolConfig{Enabled: protocol.Enabled, Interval: protocol.Interval}
}

func ipString(ip net.IP) string {
	if ip == nil {
		return ""
	}
	return ip.String()
}

func ipStrings(ips []net.IP) []string {
	out := make([]string, len(ips))
	for i, ip := range ips {
		out[i] = ipString(ip)
	}
	return out
}

func hardwareAddrString(address net.HardwareAddr) string {
	if address == nil {
		return ""
	}
	return address.String()
}
