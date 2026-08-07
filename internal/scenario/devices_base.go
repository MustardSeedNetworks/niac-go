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
	macSuffix := identitySuffix(spec.site, spec.role, spec.index)
	device := converter.Device{
		Name: spec.name, Type: profile.DeviceType, Vendor: profile.Vendor,
		MACSuffix: macSuffix, IPs: spec.ips,
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
	if spec.role == "ap" {
		device.SnmpAgent.AddMibs = apDiscoveryMIBs(spec.name, spec.site.Code, macSuffix)
	}
	return device
}

func endpointDevice(
	request Request,
	site Site,
	index, accessIndex, slot int,
	address string,
) converter.Device {
	kind := wiredEndpointKind(request, accessIndex, slot)
	profile := profileByRole(kind.role)
	name := wiredEndpointName(request, site, accessIndex, slot)
	device := converter.Device{
		Name: name, Type: profile.DeviceType, Vendor: profile.Vendor,
		MACSuffix: identitySuffix(&site, kind.role, index), IPs: []string{address}, VLAN: vlanData,
		Interfaces: []converter.Interface{newInterface(
			"eth0",
			siteNetworkName(site, "data"),
			address+"/24",
			speedOneGigabit,
			"Wired client access",
		)},
		Icmp: &converter.IcmpConfig{Enabled: true, TTL: kind.ttl},
		OSFingerprint: &converter.OSFingerprintConfig{
			OSType: kind.osType, TTL: kind.ttl, WindowSize: kind.windowSize,
			MSS: windowsMSS, DontFragment: true,
		},
		Properties: map[string]string{
			"role": kind.role, "site": site.Code, "model": profile.Model,
			"platform": profile.Platform, "software": profile.Software,
		},
		// Endpoints answer SNMP for their own identity. A discovery tool names a
		// host from sysName first and only falls back to a reverse lookup, which
		// it resolves through its own resolver rather than the simulated one —
		// so without an agent here these devices render as bare IP addresses on
		// the map. Managed endpoints (infusion pumps, imaging, patient monitors,
		// clinical workstations) carry an agent in the real world too.
		SnmpAgent: &converter.SnmpAgent{
			Community: request.SNMPCommunity, SysName: name,
			// Platform and software already read the way a real agent reports
			// itself ("Siemens Healthineers MR imaging system, Embedded clinical
			// software"). The raw vendor field is a lowercase lookup key and
			// renders as "siemens MAGNETOM Vida" on a discovery map.
			SysDescr:    profile.Platform + ", " + profile.Software,
			SysLocation: site.Location, SysContact: "netops@" + request.Domain,
		},
	}
	if kind.osType == "windows" {
		device.Netbios = &converter.NetbiosConfig{
			Enabled: true, Name: name, Workgroup: "DEMO", NodeType: "H", Services: []string{"workstation"},
		}
	}
	return device
}

func newInterface(
	name, network, address string,
	speed int,
	description string,
) converter.Interface {
	interfaceType := "ethernet"
	mtu := standardMTU
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

// utilization derives a busy-but-plausible load for one interface. A production
// network under load sits mostly in the 50-70% band, peaks higher on a minority
// of links, and leaves a few quiet. Keying it off the interface name keeps every
// run reproducible, which the Link-Live comparator depends on.
//
// Peaks stop below utilizationWarningFloor: Link-Live raises an interface
// Warning above 80% (measured — clean up to 78.7%, warned from 81.8%), and a
// demo map should read healthy rather than scattered with amber icons.
func utilization(name string) (float64, float64) {
	const outSeedFactor = 7
	sum := 0
	for _, value := range []byte(name) {
		sum += int(value)
	}
	return utilizationBand(sum), utilizationBand(sum * outSeedFactor)
}

func utilizationBand(seed int) float64 {
	const (
		steadyShare   = 70 // of 100 interfaces, this many sit in the steady band
		peakShare     = 90 // steadyShare..peakShare peak; the remainder stay quiet
		steadyFloor   = 50
		peakFloor     = 70
		quietFloor    = 25
		steadySpread  = 21 // inclusive width of the steady band: 50-70
		peakSpread    = 9  // inclusive width of the peak band: 70-78
		quietSpread   = 25
		bandSelector  = 100
		mixMultiplier = 31 // odd multiplier and shift decorrelate band from value
		mixShift      = 7
	)
	// Mix before selecting so a name's band and its value inside that band are
	// not derived from the same low bits.
	mixed := seed*mixMultiplier + seed/mixShift
	switch band := mixed % bandSelector; {
	case band < steadyShare:
		return float64(steadyFloor + seed%steadySpread)
	case band < peakShare:
		return float64(peakFloor + seed%peakSpread)
	default:
		return float64(quietFloor + seed%quietSpread)
	}
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
	if len(vlans) == 0 {
		return 0
	}
	for _, vlan := range vlans {
		if vlan == vlanManagement {
			return vlan
		}
	}
	return vlans[0]
}
