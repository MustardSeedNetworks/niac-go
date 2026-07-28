package scenario

import (
	"fmt"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

func buildSiteLAN(request Request, site Site, siteIndex int, links linkMap) []converter.Device {
	specs := make([]deviceSpec, 0,
		request.Counts.CoreSwitches+request.Counts.DistributionSwitches+
			request.Counts.AccessSwitches+request.Counts.ServerSwitches+
			request.Counts.AccessSwitches*request.Counts.AccessPointsPerAccess)
	for index := 1; index <= request.Counts.CoreSwitches; index++ {
		specs = append(specs, coreSwitch(site, siteIndex, index))
	}
	for index := 1; index <= request.Counts.DistributionSwitches; index++ {
		specs = append(specs, distributionSwitch(site, index))
	}
	for index := 1; index <= request.Counts.AccessSwitches; index++ {
		specs = append(specs, accessSwitch(site, index))
	}
	for index := 1; index <= request.Counts.ServerSwitches; index++ {
		specs = append(specs, serverSwitch(site, index))
	}
	accessPointCount := request.Counts.AccessSwitches * request.Counts.AccessPointsPerAccess
	for index := 1; index <= accessPointCount; index++ {
		specs = append(specs, accessPoint(site, index, request.Counts.AccessPointsPerAccess))
	}

	devices := make([]converter.Device, len(specs))
	for index, spec := range specs {
		devices[index] = managedDevice(request, spec, links)
	}
	return devices
}

func coreSwitch(site Site, siteIndex, index int) deviceSpec {
	coreNetwork, _, coreBase := transit(site, "core", siteIndex)
	addresses := make([]string, 0, len(vlanDefinitions()))
	interfaces := make([]converter.Interface, 0, len(vlanDefinitions())+1)
	for _, vlan := range vlanDefinitions() {
		address := siteIP(site, vlan.thirdOctet, index+1)
		addresses = append(addresses, address)
		interfaces = append(interfaces, newInterface(
			fmt.Sprintf("Vlan%d", vlan.id), siteNetworkName(site, vlan.slug), address+"/24",
			speedHundredGigabit, "Gateway for "+vlan.slug,
		))
	}
	transitAddress := fmt.Sprintf("203.0.113.%d", coreBase+index+transitPeerOffset)
	interfaces = append(interfaces, newInterface(
		"HundredGigabitEthernet0/0/1", coreNetwork, transitAddress+"/29", speedHundredGigabit,
		"Firewall transit",
	))
	return deviceSpec{
		name: numberedName(site.Code+"-CORE-SW", index), role: "core", index: index,
		ips: append(addresses, transitAddress), site: &site,
		sysDescr:   fmt.Sprintf("Cisco Nexus 9508 %s modular core %d", site.Code, index),
		interfaces: interfaces,
		routes: []converter.Route{route(
			"0.0.0.0/0", "HundredGigabitEthernet0/0/1",
			fmt.Sprintf("203.0.113.%d", coreBase+index),
		)},
		vlan: vlanManagement,
	}
}

func distributionSwitch(site Site, index int) deviceSpec {
	address := siteIP(site, vlanManagement, distributionHostOffset+index)
	return deviceSpec{
		name: numberedName(site.Code+"-DIST-SW", index), role: "distribution", index: index,
		ips: []string{address}, site: &site,
		sysDescr: fmt.Sprintf("Cisco Catalyst C9606R %s distribution block %d", site.Code, index),
		interfaces: []converter.Interface{newInterface(
			"Vlan200", siteNetworkName(site, "mgmt"), address+"/24", speedHundredGigabit, "Network management",
		)},
		vlan: vlanManagement,
	}
}

func accessSwitch(site Site, index int) deviceSpec {
	address := siteIP(site, vlanManagement, accessHostOffset+index)
	return deviceSpec{
		name: accessName(site, index), role: "access", index: index,
		ips: []string{address}, site: &site,
		sysDescr: fmt.Sprintf("Cisco C9350-48HX %s multigigabit access %d", site.Code, index),
		interfaces: []converter.Interface{newInterface(
			"Vlan200", siteNetworkName(site, "mgmt"), address+"/24", speedHundredGigabit, "Network management",
		)},
		vlan: vlanManagement,
	}
}

func serverSwitch(site Site, index int) deviceSpec {
	address := siteIP(site, vlanManagement, serverSwitchHostOffset+index)
	return deviceSpec{
		name: numberedName(site.Code+"-SRV-SW", index), role: "server-switch", index: index,
		ips: []string{address}, site: &site,
		sysDescr: fmt.Sprintf("Cisco Nexus 93180YC-FX3 %s server leaf %d", site.Code, index),
		interfaces: []converter.Interface{newInterface(
			"Vlan200", siteNetworkName(site, "mgmt"), address+"/24", speedHundredGigabit, "Network management",
		)},
		vlan: vlanManagement,
	}
}

func accessPoint(site Site, index, perAccess int) deviceSpec {
	accessIndex := (index-1)/perAccess + 1
	slot := (index-1)%perAccess + 1
	address := siteIP(site, vlanManagement, accessPointHostOffset+index)
	return deviceSpec{
		name: accessPointName(site, accessIndex, slot), role: "ap", index: index,
		ips: []string{address}, site: &site,
		sysDescr: fmt.Sprintf("Cisco Wireless CW9178I Wi-Fi 7 access point %d", index),
		interfaces: []converter.Interface{newInterface(
			"mGigabitEthernet0", siteNetworkName(site, "mgmt"), address+"/24", speedTenGigabit,
			"Multigigabit PoE uplink",
		)},
		vlan: vlanManagement,
	}
}
