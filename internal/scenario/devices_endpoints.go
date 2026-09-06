package scenario

import (
	"fmt"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

func serviceRoles() []string {
	return []string{"DNS", "DHCP", "APP", "FILE", "NMS", "PERF"}
}

func buildSiteEndpoints(request Request, site Site, links linkMap) []converter.Device {
	workstationCount := request.Counts.AccessSwitches * request.Counts.WorkstationsPerAccess
	devices := make([]converter.Device, 0,
		workstationCount+len(serviceRoles())+request.Counts.WirelessControllers)
	for accessIndex := 1; accessIndex <= request.Counts.AccessSwitches; accessIndex++ {
		for slot := 1; slot <= request.Counts.WorkstationsPerAccess; slot++ {
			index := (accessIndex-1)*request.Counts.WorkstationsPerAccess + slot
			devices = append(devices, endpointDevice(
				request, site, index, accessIndex, slot,
				siteIP(site, vlanData, workstationHostOffset+index),
			))
		}
	}
	devices = append(devices, medEndpoints(request, site, workstationCount)...)
	for index, role := range serviceRoles() {
		devices = append(devices, serviceServer(request, site, role, index+1, links))
	}
	for index := 1; index <= request.Counts.WirelessControllers; index++ {
		devices = append(devices, wirelessController(request, site, index, links))
	}
	return devices
}

func serviceServer(request Request, site Site, role string, index int, links linkMap) converter.Device {
	vlan, network := servicePlacement(role)
	address := siteIP(site, vlan, serviceHostOffset+index)
	spec := deviceSpec{
		name: site.Code + "-" + role + "01", role: "server", index: index,
		ips: []string{address}, site: &site,
		sysDescr: fmt.Sprintf("Dell PowerEdge R660 %s %s service", site.Code, role),
		interfaces: []converter.Interface{newInterface(
			"eth0", siteNetworkName(site, network), address+"/24", speedTenGigabit, role+" service uplink",
		)},
		vlan: vlan,
	}
	applyServiceCapabilities(request, site, role, &spec)
	return managedDevice(request, spec, links)
}

func servicePlacement(role string) (int, string) {
	if role == "DHCP" {
		return vlanData, "data"
	}
	return vlanServers, "servers"
}

func applyServiceCapabilities(request Request, site Site, role string, spec *deviceSpec) {
	switch role {
	case "DNS":
		records := siteDNSRecords(request, site)
		spec.dns = &converter.DNSServer{ForwardRecords: records, ReverseRecords: records}
	case "DHCP":
		spec.dhcp = &converter.DhcpServer{
			ServerIdentifier: siteIP(site, vlanData, dhcpServerHost),
			Router:           siteIP(site, vlanData, primaryCoreGatewayHost),
			SubnetMask:       "255.255.255.0",
			DomainNameServer: siteIP(site, vlanServers, dnsServerHost),
			PoolStart:        siteIP(site, vlanData, dhcpPoolStartHost),
			PoolEnd:          siteIP(site, vlanData, dhcpPoolEndHost),
		}
	case "APP":
		spec.http = &converter.HTTPConfig{Enabled: true, ServerName: "Microsoft-IIS/10.0"}
	case "FILE":
		spec.netbios = &converter.NetbiosConfig{
			Enabled: true, Name: site.Code + "-FILE01", Workgroup: "DEMO", NodeType: "H",
			Services: []string{"workstation", "fileserver"},
		}
	case "NMS":
		spec.http = &converter.HTTPConfig{Enabled: true, ServerName: "Grafana"}
	case "PERF":
		spec.iperf3 = &converter.IPerf3Config{
			Enabled: true, Port: performanceTestPort, MaxBandwidthMbps: performanceBandwidthMbps,
			TypicalLatencyMs: performanceLatencyMillis, JitterMs: performanceJitterMillis,
			PacketLossPercent: performancePacketLoss,
		}
		spec.reflector = &converter.ReflectorConfig{
			LatencyMs: performanceLatencyMillis, JitterMs: performanceJitterMillis, DSCP: true,
		}
	default:
		panic("unknown service role: " + role)
	}
}

func siteDNSRecords(request Request, site Site) []converter.DNSRecord {
	roles := serviceRoles()
	workstationCount := request.Counts.AccessSwitches * request.Counts.WorkstationsPerAccess
	records := make([]converter.DNSRecord, 0, len(roles)+workstationCount)
	for index, role := range roles {
		vlan, _ := servicePlacement(role)
		records = append(records, converter.DNSRecord{
			Name: fmt.Sprintf("%s-%s01.%s", strings.ToLower(site.Code), strings.ToLower(role), request.Domain),
			IP:   siteIP(site, vlan, serviceHostOffset+index+1), TTL: dnsRecordTTL,
		})
	}
	for accessIndex := 1; accessIndex <= request.Counts.AccessSwitches; accessIndex++ {
		for slot := 1; slot <= request.Counts.WorkstationsPerAccess; slot++ {
			index := (accessIndex-1)*request.Counts.WorkstationsPerAccess + slot
			records = append(records, converter.DNSRecord{
				Name: strings.ToLower(wiredEndpointName(request, site, accessIndex, slot)) + "." + request.Domain,
				IP:   siteIP(site, vlanData, workstationHostOffset+index), TTL: dnsRecordTTL,
			})
		}
	}
	return records
}

func wirelessController(request Request, site Site, index int, links linkMap) converter.Device {
	address := siteIP(site, vlanServers, controllerHostOffset+index)
	return managedDevice(request, deviceSpec{
		name: numberedName(site.Code+"-WLC", index), role: "controller", index: index,
		ips: []string{address}, site: &site,
		sysDescr: fmt.Sprintf("Cisco Catalyst 9800-L %s wireless controller %d", site.Code, index),
		interfaces: []converter.Interface{newInterface(
			"TenGigabitEthernet0/0/0", siteNetworkName(site, "servers"), address+"/24", speedTenGigabit,
			"Wireless control plane",
		)},
		vlan: vlanServers, http: &converter.HTTPConfig{Enabled: true, ServerName: "Cisco IOS XE"},
	}, links)
}

// medEndpoints appends the phones and cameras that make LLDP-MED visible.
//
// They are appended rather than mixed into the wired round-robin on purpose.
// Adding two kinds to that rotation changes which kind every later slot gets,
// so 13 of the hospital pack's existing devices became different devices --
// and the pack's Link-Live acceptance was verified against those exact 75.
// Appending leaves every verified device where it was and makes the diff
// purely additive.
func medEndpoints(request Request, site Site, workstationCount int) []converter.Device {
	counts := map[string]int{"voip-phone": medPhonesPerSite, "ip-camera": medCamerasPerSite}
	order := []string{"voip-phone", "ip-camera"}

	devices := make([]converter.Device, 0, medPhonesPerSite+medCamerasPerSite)
	offset := workstationCount
	for _, role := range order {
		for index := 1; index <= counts[role]; index++ {
			offset++
			devices = append(devices, medEndpoint(request, site, role, index, offset))
		}
	}

	return devices
}

const (
	// medPhonesPerSite and medCamerasPerSite are deliberately small: they exist
	// so a discovery tool has something to classify by MED, not to model a real
	// handset count. Growing them re-signs every pack manifest.
	// Two and one, not three and two: the presentation packs are capped at 160
	// devices for the Link-Live map, and campus has four sites, so five per
	// site would put it at 167 and over the budget.
	medPhonesPerSite  = 2
	medCamerasPerSite = 1
)

// medEndpoint builds one MED-advertising endpoint.
func medEndpoint(request Request, site Site, role string, index, hostOffset int) converter.Device {
	profile := profileByRole(role)
	prefix := "PHONE"
	if role == "ip-camera" {
		prefix = "CAM"
	}
	name := fmt.Sprintf("%s-%s-%02d", site.Code, prefix, index)
	address := siteIP(site, vlanVoiceIoT, workstationHostOffset+hostOffset)
	access := newInterface(
		"eth0",
		siteNetworkName(site, "voice-iot"),
		address+"/24",
		speedOneGigabit,
		profile.Platform+" access",
	)
	access.InUtilization, access.OutUtilization = utilization(name + "/eth0")

	device := converter.Device{
		Name: name, Type: profile.DeviceType, Vendor: profile.Vendor,
		// hostOffset, not index: it continues the wired endpoints' numbering, so
		// a phone and a camera at index 1 cannot land on the same MAC as each
		// other or on the first workstation's.
		MACSuffix: identitySuffix(&site, role, hostOffset), IPs: []string{address}, VLAN: vlanVoiceIoT,
		Interfaces: []converter.Interface{access},
		Icmp:       &converter.IcmpConfig{Enabled: true, TTL: unixTTL},
		SnmpAgent: &converter.SnmpAgent{
			Community: request.SNMPCommunity, SysName: name,
			SysDescr:    profile.Platform + ", " + profile.Software,
			SysLocation: site.Location, SysContact: "netops@" + request.Domain,
		},
		Mdns: &converter.MdnsConfig{
			Enabled: true, Hostname: strings.ToLower(name),
			Services: endpointMDNSServices(profile.DeviceType),
		},
		Properties: map[string]string{
			"role": role, "site": site.Code, "model": profile.Model,
			"platform": profile.Platform, "software": profile.Software,
		},
		Lldp: &converter.LldpConfig{
			Enabled:           true,
			SystemDescription: profile.Platform + " - " + name,
			MED:               medProfileFor(role, profile, apSerialNumber(identitySuffix(&site, role, hostOffset))),
		},
	}

	return device
}

// medProfileFor is the MED block each endpoint class advertises.
//
// A phone is a class III communication device on a tagged voice VLAN; a camera
// is a class II media endpoint on the same segment. Both draw PoE, which is
// what a switch reads to budget power.
func medProfileFor(role string, profile DeviceProfile, serial string) *converter.LldpMedConfig {
	deviceType, application, watts := "endpoint_class3", "voice", medPhoneTenthWatts
	if role == "ip-camera" {
		deviceType, application, watts = "endpoint_class2", "streaming_video", medCameraTenthWatts
	}

	return &converter.LldpMedConfig{
		DeviceType: deviceType,
		NetworkPolicies: []converter.LldpMedNetworkPolicy{{
			Application: application, Tagged: true, VLANID: vlanVoiceIoT,
			Priority: medVoicePriority, DSCP: medVoiceDSCP,
		}},
		Power: &converter.LldpMedPower{
			DeviceType: "pd", Source: "pse", Priority: "high", ValueTenthWatts: watts,
		},
		Inventory: &converter.LldpMedInventory{
			SoftwareRevision: profile.Software,
			SerialNumber:     serial,
			Manufacturer:     profile.Vendor,
			ModelName:        profile.Model,
		},
	}
}

const (
	// medPhoneTenthWatts is a class-2 handset's draw; medCameraTenthWatts a
	// class-3 PTZ camera's. Both in the TLV's own 0.1 W unit.
	medPhoneTenthWatts  = 65
	medCameraTenthWatts = 128
	medVoicePriority    = 5
	medVoiceDSCP        = 46
)
