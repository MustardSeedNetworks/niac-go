package scenario

import (
	"fmt"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

func labEdge(request Request, links linkMap) converter.Device {
	routes := make([]converter.Route, 0, len(request.Sites)+1)
	for _, site := range request.Sites {
		routes = append(routes, route(
			fmt.Sprintf("10.%d.0.0/16", site.Octet),
			"HundredGigabitEthernet0/0/1", "203.0.113.2",
		))
	}
	routes = append(routes, route("0.0.0.0/0", "HundredGigabitEthernet0/0/1", "203.0.113.2"))
	return managedDevice(request, deviceSpec{
		name: "LAB-EDGE-R1", role: "lab", index: 1,
		ips:      []string{transitGateway, "203.0.113.1"},
		sysDescr: "Cisco Catalyst 8500L isolated lab edge",
		interfaces: []converter.Interface{
			newInterface(
				"TenGigabitEthernet0/0/0",
				"lab-transit",
				transitGateway+"/24",
				speedTenGigabit,
				"CyberScope VLAN 200 attachment",
			),
			newInterface(
				"HundredGigabitEthernet0/0/1",
				"lab-wan",
				"203.0.113.1/29",
				speedHundredGigabit,
				"Provider WAN uplink",
			),
		},
		routes: routes,
		dhcp: &converter.DhcpServer{
			ServerIdentifier: transitGateway, Router: transitGateway, SubnetMask: "255.255.255.0",
			DomainNameServer: transitGateway, PoolStart: "10.254.200.100", PoolEnd: "10.254.200.199",
		},
		dns: &converter.DNSServer{ForwardRecords: internetDNSRecords()},
	}, links)
}

func providerWANRouter(request Request, index int, links linkMap) converter.Device {
	labAddress := fmt.Sprintf("203.0.113.%d", index+1)
	interfaces := []converter.Interface{
		newInterface(
			"HundredGigabitEthernet0/0/1",
			"lab-wan",
			labAddress+"/29",
			speedHundredGigabit,
			"Lab edge transit",
		),
	}
	routes := []converter.Route{route(transitSubnet, "HundredGigabitEthernet0/0/1", "203.0.113.1")}
	ips := []string{labAddress}
	for siteIndex, site := range request.Sites {
		if index > request.Counts.SiteWANRouters {
			continue
		}
		network, _, base := transit(site, "wan", siteIndex)
		address := fmt.Sprintf("203.0.113.%d", base+index)
		interfaceName := fmt.Sprintf("HundredGigabitEthernet0/0/%d", siteIndex+firstSiteInterfaceIndex)
		interfaces = append(interfaces, newInterface(
			interfaceName, network, address+"/29", speedHundredGigabit, "Site WAN transit",
		))
		routes = append(routes, route(
			fmt.Sprintf("10.%d.0.0/16", site.Octet), interfaceName,
			fmt.Sprintf("203.0.113.%d", base+index+transitPeerOffset),
		))
		ips = append(ips, address)
	}

	spec := deviceSpec{
		name: fmt.Sprintf("WAN-R%d", index), role: "wan", index: index, ips: ips,
		sysDescr: fmt.Sprintf("Cisco 8201 global WAN provider edge %d", index),
		model:    "8201-32FH", platform: "Cisco 8201-32FH", software: "IOS XR 24.4",
		interfaces: interfaces, routes: routes,
	}
	if index == 1 {
		spec.ips = append(spec.ips, internetLoopback)
		spec.interfaces = append(spec.interfaces, newInterface(
			"Loopback0",
			"internet-loopback",
			internetLoopback+"/32",
			speedHundredGigabit,
			"Internet reachability target",
		))
		spec.http = &converter.HTTPConfig{Enabled: true, ServerName: "nginx"}
		spec.dns = &converter.DNSServer{ForwardRecords: internetDNSRecords()}
	}
	return managedDevice(request, spec, links)
}

func siteWANRouter(site Site, siteIndex, index int) deviceSpec {
	wanNetwork, _, wanBase := transit(site, "wan", siteIndex)
	securityNetwork, _, securityBase := transit(site, "security", siteIndex)
	management := siteIP(site, vlanManagement, wanManagementHostOffset+index)
	return deviceSpec{
		name: numberedName(site.Code+"-WAN-R", index), role: "wan",
		index: siteIndex*maxRedundantPeers + index + wanIdentityOffset,
		ips: []string{
			management,
			fmt.Sprintf("203.0.113.%d", wanBase+index+transitPeerOffset),
			fmt.Sprintf("203.0.113.%d", securityBase+index),
		},
		site: &site, sysDescr: fmt.Sprintf("Cisco Catalyst 8500-12X %s WAN edge %d", site.Code, index),
		interfaces: []converter.Interface{
			newInterface(
				"TenGigabitEthernet0/0/0",
				siteNetworkName(site, "mgmt"),
				management+"/24",
				speedTenGigabit,
				"Network management",
			),
			newInterface(
				"HundredGigabitEthernet0/0/1",
				wanNetwork,
				fmt.Sprintf("203.0.113.%d/29", wanBase+index+transitPeerOffset),
				speedHundredGigabit,
				"Provider WAN",
			),
			newInterface(
				"HundredGigabitEthernet0/0/2",
				securityNetwork,
				fmt.Sprintf("203.0.113.%d/29", securityBase+index),
				speedHundredGigabit,
				"Firewall transit",
			),
		},
		routes: []converter.Route{
			route(
				fmt.Sprintf("10.%d.0.0/16", site.Octet),
				"HundredGigabitEthernet0/0/2",
				fmt.Sprintf("203.0.113.%d", securityBase+index+transitPeerOffset),
			),
			route("0.0.0.0/0", "HundredGigabitEthernet0/0/1", fmt.Sprintf("203.0.113.%d", wanBase+index)),
		},
		vlan: vlanManagement,
	}
}

func siteFirewall(site Site, siteIndex, index int) deviceSpec {
	securityNetwork, _, securityBase := transit(site, "security", siteIndex)
	coreNetwork, _, coreBase := transit(site, "core", siteIndex)
	management := siteIP(site, vlanManagement, firewallManagementHostOffset+index)
	return deviceSpec{
		name: numberedName(site.Code+"-FW", index), role: "firewall", index: siteIndex*2 + index,
		ips: []string{
			management,
			fmt.Sprintf("203.0.113.%d", securityBase+index+transitPeerOffset),
			fmt.Sprintf("203.0.113.%d", coreBase+index),
		},
		site: &site, sysDescr: fmt.Sprintf("Palo Alto PA-5450 %s perimeter firewall %d", site.Code, index),
		interfaces: []converter.Interface{
			newInterface(
				"management",
				siteNetworkName(site, "mgmt"),
				management+"/24",
				speedOneGigabit,
				"Network management",
			),
			newInterface(
				"ethernet1/1",
				securityNetwork,
				fmt.Sprintf("203.0.113.%d/29", securityBase+index+transitPeerOffset),
				speedHundredGigabit,
				"WAN security zone",
			),
			newInterface(
				"ethernet1/2",
				coreNetwork,
				fmt.Sprintf("203.0.113.%d/29", coreBase+index),
				speedHundredGigabit,
				"Trusted core zone",
			),
		},
		routes: []converter.Route{
			route(
				fmt.Sprintf("10.%d.0.0/16", site.Octet),
				"ethernet1/2",
				fmt.Sprintf("203.0.113.%d", coreBase+index+transitPeerOffset),
			),
			route("0.0.0.0/0", "ethernet1/1", fmt.Sprintf("203.0.113.%d", securityBase+index)),
		},
		vlan: vlanManagement,
	}
}
