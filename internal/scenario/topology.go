package scenario

import (
	"fmt"
	"strings"
)

func allTrunkVLANs() []int {
	return []int{vlanManagement, vlanData, vlanWiFiCorp, vlanWiFiGuest, vlanServers, vlanVoiceIoT}
}

type link struct {
	localInterface  string
	remoteDevice    string
	remoteInterface string
	vlans           []int
	fdbOnly         bool
}

type linkMap map[string][]link

func buildLinks(request Request) linkMap {
	links := make(linkMap)
	addEdge(links,
		endpoint{"LAB-EDGE-R1", "HundredGigabitEthernet0/0/1"},
		endpoint{"WAN-R1", "HundredGigabitEthernet0/0/1"}, []int{})
	addEdge(links,
		endpoint{"LAB-EDGE-R1", "HundredGigabitEthernet0/0/2"},
		endpoint{"WAN-R2", "HundredGigabitEthernet0/0/1"}, []int{})
	for siteIndex, site := range request.Sites {
		addSiteBackbone(links, site, siteIndex, request.Counts)
		addSiteLAN(links, site, request)
		addSiteEndpoints(links, site, request)
	}
	return links
}

type endpoint struct {
	device        string
	interfaceName string
}

func addEdge(links linkMap, left, right endpoint, vlans []int) {
	if vlans == nil {
		vlans = allTrunkVLANs()
	}
	leftVLANs := append([]int(nil), vlans...)
	rightVLANs := append([]int(nil), vlans...)
	links[left.device] = append(links[left.device], link{
		localInterface: left.interfaceName, remoteDevice: right.device,
		remoteInterface: right.interfaceName, vlans: leftVLANs,
	})
	links[right.device] = append(links[right.device], link{
		localInterface: right.interfaceName, remoteDevice: left.device,
		remoteInterface: left.interfaceName, vlans: rightVLANs,
	})
}

func addFDB(links linkMap, sw, interfaceName, remote, remoteInterface string, vlan int) {
	links[sw] = append(links[sw], link{
		localInterface: interfaceName, remoteDevice: remote, remoteInterface: remoteInterface,
		vlans: []int{vlan}, fdbOnly: true,
	})
}

func addMesh(links linkMap, left, right []string, prefix string, vlans []int) {
	index := 1
	for _, leftName := range left {
		for _, rightName := range right {
			interfaceName := fmt.Sprintf("%s%d", prefix, index)
			addEdge(links, endpoint{leftName, interfaceName}, endpoint{rightName, interfaceName}, vlans)
			index++
		}
	}
}

func addSiteBackbone(links linkMap, site Site, siteIndex int, counts Counts) {
	wan := numberedNames(site.Code+"-WAN-R", counts.SiteWANRouters)
	firewalls := numberedNames(site.Code+"-FW", counts.Firewalls)
	cores := numberedNames(site.Code+"-CORE-SW", counts.CoreSwitches)
	for index, wanName := range wan {
		provider := index%maxRedundantPeers + 1
		addEdge(links,
			endpoint{
				fmt.Sprintf("WAN-R%d", provider),
				fmt.Sprintf("HundredGigabitEthernet0/0/%d", siteIndex+firstSiteInterfaceIndex),
			},
			endpoint{wanName, "HundredGigabitEthernet0/0/1"}, []int{})
	}
	addMesh(links, wan, firewalls, "TenGigabitEthernet0/1/", []int{})
	addMesh(links, firewalls, cores, "HundredGigabitEthernet0/2/", []int{})
}

func addSiteLAN(links linkMap, site Site, request Request) {
	counts := request.Counts
	cores := numberedNames(site.Code+"-CORE-SW", counts.CoreSwitches)
	for distribution := 1; distribution <= counts.DistributionSwitches; distribution++ {
		distName := numberedName(site.Code+"-DIST-SW", distribution)
		for coreIndex, coreName := range cores {
			addEdge(links,
				endpoint{coreName, fmt.Sprintf("HundredGigabitEthernet1/0/%d", distribution)},
				endpoint{distName, fmt.Sprintf("HundredGigabitEthernet1/0/%d", coreIndex+1)}, nil)
		}
	}

	if request.AccessLayer == AccessLayerRing {
		addAccessRing(links, site, counts)
		addServerSwitches(links, site, counts)

		return
	}

	distributionPairs := counts.DistributionSwitches / maxRedundantPeers
	accessPerPair := (counts.AccessSwitches + distributionPairs - 1) / distributionPairs
	for accessIndex := 1; accessIndex <= counts.AccessSwitches; accessIndex++ {
		pair := min((accessIndex-1)/accessPerPair, distributionPairs-1)
		firstDistribution := pair*maxRedundantPeers + 1
		distributionPort := (accessIndex-1)%accessPerPair + firstDistributionAccessPort
		for uplink := range maxRedundantPeers {
			distribution := firstDistribution + uplink
			addEdge(
				links,
				endpoint{
					numberedName(site.Code+"-DIST-SW", distribution),
					fmt.Sprintf("HundredGigabitEthernet1/0/%d", distributionPort),
				},
				endpoint{
					accessName(site, accessIndex),
					fmt.Sprintf("HundredGigabitEthernet1/0/%d", uplink+accessUplinkPortStart),
				},
				nil,
			)
		}
	}

	addServerSwitches(links, site, counts)
}

func addServerSwitches(links linkMap, site Site, counts Counts) {
	cores := numberedNames(site.Code+"-CORE-SW", counts.CoreSwitches)
	for serverSwitch := 1; serverSwitch <= counts.ServerSwitches; serverSwitch++ {
		name := numberedName(site.Code+"-SRV-SW", serverSwitch)
		for coreIndex, coreName := range cores {
			addEdge(links,
				endpoint{coreName, fmt.Sprintf("HundredGigabitEthernet1/0/%d", serverSwitch+coreServerPortOffset)},
				endpoint{name, fmt.Sprintf("HundredGigabitEthernet1/0/%d", coreIndex+1)}, nil)
		}
	}
}

// addAccessRing closes the access tier into one ring and joins it to the
// distribution tier at two opposite points, so a break anywhere on the ring
// still leaves every cell a path out. Link-Live renders this as the loop it is
// rather than reducing it to a tree, which is what makes it worth generating.
func addAccessRing(links linkMap, site Site, counts Counts) {
	for index := 1; index <= counts.AccessSwitches; index++ {
		next := index%counts.AccessSwitches + 1
		addEdge(links,
			endpoint{accessName(site, index), ringEastPort},
			endpoint{accessName(site, next), ringWestPort}, nil)
	}
	for join := range maxRedundantPeers {
		accessIndex := join*counts.AccessSwitches/maxRedundantPeers + 1
		addEdge(links,
			endpoint{
				numberedName(site.Code+"-DIST-SW", join+1),
				fmt.Sprintf("HundredGigabitEthernet1/0/%d", firstDistributionAccessPort),
			},
			endpoint{
				accessName(site, accessIndex),
				fmt.Sprintf("HundredGigabitEthernet1/0/%d", accessUplinkPortStart),
			},
			nil)
	}
}

func addSiteEndpoints(links linkMap, site Site, request Request) {
	counts := request.Counts
	for accessIndex := 1; accessIndex <= counts.AccessSwitches; accessIndex++ {
		switchName := accessName(site, accessIndex)
		for slot := 1; slot <= counts.AccessPointsPerAccess; slot++ {
			addEdge(links,
				endpoint{switchName, fmt.Sprintf("TenGigabitEthernet1/0/%d", slot)},
				endpoint{accessPointName(site, accessIndex, slot), "mGigabitEthernet0"},
				[]int{vlanManagement, vlanWiFiCorp, vlanWiFiGuest})
		}
		for slot := 1; slot <= counts.WorkstationsPerAccess; slot++ {
			addFDB(links, switchName, fmt.Sprintf("GigabitEthernet1/0/%d", slot+workstationPortOffset),
				wiredEndpointName(request, site, accessIndex, slot), "eth0", vlanData)
		}
	}

	services := []string{"DNS01", "DHCP01", "APP01", "FILE01", "NMS01", "PERF01"}
	for index := 1; index <= counts.WirelessControllers; index++ {
		services = append(services, fmt.Sprintf("WLC%02d", index))
	}
	for index, service := range services {
		switchIndex := index%counts.ServerSwitches + 1
		vlan := vlanServers
		if service == "DHCP01" {
			vlan = vlanData
		}
		remoteInterface := "eth0"
		if strings.HasPrefix(service, "WLC") {
			remoteInterface = "TenGigabitEthernet0/0/0"
		}
		addFDB(links, numberedName(site.Code+"-SRV-SW", switchIndex),
			fmt.Sprintf("TenGigabitEthernet1/0/%d", index+serverPortOffset), site.Code+"-"+service,
			remoteInterface, vlan)
	}
}
