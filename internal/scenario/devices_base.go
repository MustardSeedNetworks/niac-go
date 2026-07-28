package scenario

import (
	"fmt"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

type deviceSpec struct {
	name       string
	role       string
	index      int
	ips        []string
	site       *Site
	interfaces []converter.Interface
	routes     []converter.Route
	vlan       int
	sysDescr   string
	model      string
	platform   string
	software   string
	dhcp       *converter.DhcpServer
	dns        *converter.DNSServer
	http       *converter.HTTPConfig
	netbios    *converter.NetbiosConfig
	iperf3     *converter.IPerf3Config
	reflector  *converter.ReflectorConfig
}

func managedDevice(request Request, spec deviceSpec, links linkMap) converter.Device {
	profile := profileByRole(spec.role)
	model, platform, software := profile.Model, profile.Platform, profile.Software
	if spec.model != "" {
		model = spec.model
	}
	if spec.platform != "" {
		platform = spec.platform
	}
	if spec.software != "" {
		software = spec.software
	}
	interfaces := append([]converter.Interface(nil), spec.interfaces...)
	interfaces = addLinkedInterfaces(interfaces, links[spec.name])
	properties := map[string]string{
		"role": spec.role, "model": model, "platform": platform,
		"software": software, "sysObjectID": profile.SysObjectID,
	}
	if spec.site != nil {
		properties["site"] = spec.site.Code
	}
	if spec.role == "ap" {
		properties["wifiStandard"] = "Wi-Fi 7"
	}

	location := "Global WAN"
	if spec.site != nil {
		location = spec.site.Location
	}
	device := converter.Device{
		Name: spec.name, Type: profile.DeviceType, Vendor: profile.Vendor,
		MACSuffix: identitySuffix(spec.site, spec.role, spec.index), IPs: spec.ips,
		VLAN: spec.vlan, Interfaces: interfaces, Routes: spec.routes,
		SnmpAgent: &converter.SnmpAgent{
			Community: request.SNMPCommunity, SysName: spec.name, SysDescr: spec.sysDescr,
			SysLocation: location, SysContact: "netops@" + request.Domain,
		},
		Icmp:       &converter.IcmpConfig{Enabled: true, TTL: managedDeviceTTL},
		TrunkPorts: authoredTrunkPorts(links[spec.name]), Properties: properties,
		Dhcp: spec.dhcp, DNS: spec.dns, HTTP: spec.http, Netbios: spec.netbios,
		IPerf3: spec.iperf3, Reflector: spec.reflector,
	}
	if platform != "" {
		device.Lldp = &converter.LldpConfig{
			Enabled: true, SystemDescription: platform + " - " + spec.name,
		}
		if profile.Vendor == "cisco" {
			device.Cdp = &converter.CdpConfig{
				Enabled: true, Platform: platform, SoftwareVersion: software,
			}
		}
	}
	return device
}

func endpointDevice(site Site, index int, name, address string) converter.Device {
	profile := profileByRole("workstation")
	return converter.Device{
		Name: name, Type: profile.DeviceType, Vendor: profile.Vendor,
		MACSuffix: identitySuffix(&site, "workstation", index), IPs: []string{address}, VLAN: vlanData,
		Interfaces: []converter.Interface{newInterface(
			"eth0", siteNetworkName(site, "data"), address+"/24", speedOneGigabit, "Wired client access",
		)},
		Icmp: &converter.IcmpConfig{Enabled: true, TTL: windowsTTL},
		OSFingerprint: &converter.OSFingerprintConfig{
			OSType: "windows", TTL: windowsTTL, WindowSize: windowsTCPWindowSize,
			MSS: windowsMSS, DontFragment: true,
		},
		Properties: map[string]string{
			"role": "workstation", "site": site.Code, "model": profile.Model,
			"platform": profile.Platform, "software": profile.Software,
		},
	}
}

func newInterface(name, network, address string, speed int, description string) converter.Interface {
	interfaceType := "ethernet"
	mtu := 1500
	switch {
	case strings.HasPrefix(name, "Vlan"):
		interfaceType = "l2vlan"
		mtu = 9000
	case strings.HasPrefix(name, "Loopback"):
		interfaceType = "loopback"
		mtu = 65535
	case speed >= speedTenGigabit:
		mtu = 9000
	}
	inUtilization, outUtilization := utilization(name)
	return converter.Interface{
		Name: name, Type: interfaceType, Network: network, Address: address,
		MTU: mtu, Speed: speed, Duplex: "full", AdminStatus: "up", OperStatus: "up",
		Description: description, InUtilization: inUtilization, OutUtilization: outUtilization,
	}
}

func addLinkedInterfaces(interfaces []converter.Interface, links []link) []converter.Interface {
	byName := make(map[string]int, len(interfaces))
	for index := range interfaces {
		byName[interfaces[index].Name] = index
	}
	for _, peer := range links {
		description := fmt.Sprintf("to %s %s", peer.remoteDevice, peer.remoteInterface)
		if index, found := byName[peer.localInterface]; found {
			interfaces[index].Description = description
			interfaces[index].VLANs = append([]int(nil), peer.vlans...)
			continue
		}
		speed := interfaceSpeed(peer.localInterface)
		iface := newInterface(peer.localInterface, "", "", speed, description)
		iface.VLANs = append([]int(nil), peer.vlans...)
		byName[peer.localInterface] = len(interfaces)
		interfaces = append(interfaces, iface)
	}
	return interfaces
}

func interfaceSpeed(name string) int {
	switch {
	case strings.HasPrefix(name, "HundredGigabit"):
		return speedHundredGigabit
	case strings.HasPrefix(name, "TenGigabit"), strings.HasPrefix(name, "mGigabit"):
		return speedTenGigabit
	default:
		return speedOneGigabit
	}
}

func utilization(name string) (float64, float64) {
	const (
		minimumInUtilization  = 5
		inUtilizationSpread   = 26
		minimumOutUtilization = 4
		outUtilizationFactor  = 7
		outUtilizationSpread  = 25
	)
	sum := 0
	for _, value := range []byte(name) {
		sum += int(value)
	}
	return float64(minimumInUtilization + sum%inUtilizationSpread),
		float64(minimumOutUtilization + (sum*outUtilizationFactor)%outUtilizationSpread)
}

func authoredTrunkPorts(links []link) []converter.TrunkPort {
	ports := make([]converter.TrunkPort, 0, len(links))
	for _, peer := range links {
		ports = append(ports, converter.TrunkPort{
			Interface: peer.localInterface, VLANs: append([]int(nil), peer.vlans...),
			NativeVLAN: nativeVLAN(peer.vlans), RemoteDevice: peer.remoteDevice,
			RemoteInterface: peer.remoteInterface, FDBOnly: peer.fdbOnly,
		})
	}
	return ports
}

func nativeVLAN(vlans []int) int {
	for _, vlan := range vlans {
		if vlan == vlanManagement {
			return vlan
		}
	}
	return vlans[0]
}
