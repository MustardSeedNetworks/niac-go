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
				site, index, workstationName(site, accessIndex, slot),
				siteIP(site, vlanData, workstationHostOffset+index),
			))
		}
	}
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
		spec.dns = &converter.DNSServer{ForwardRecords: siteDNSRecords(request, site)}
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
	records := make([]converter.DNSRecord, len(roles))
	for index, role := range roles {
		vlan, _ := servicePlacement(role)
		records[index] = converter.DNSRecord{
			Name: fmt.Sprintf("%s-%s01.%s", strings.ToLower(site.Code), strings.ToLower(role), request.Domain),
			IP:   siteIP(site, vlan, serviceHostOffset+index+1), TTL: dnsRecordTTL,
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
