package snmp

import (
	"net"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func (a *Agent) refreshDeviceStateIPMIBs(snapshot devicestate.Snapshot) {
	for oid := range a.stateIPOIDs {
		a.mib.Delete(oid)
	}
	a.stateIPOIDs = make(map[string]struct{})
	for _, iface := range snapshot.Network.Interfaces {
		if !iface.Address.IsValid() || !iface.Address.Addr().Is4() {
			continue
		}
		index, ok := a.InterfaceIndex(iface.Name)
		if !ok {
			continue
		}
		address := iface.Address.Addr().String()
		mask := net.CIDRMask(iface.Address.Bits(), iface.Address.Addr().BitLen())
		a.registerIPAddrEntry(address, index, mask)
		for _, column := range []string{
			ipAdEntAddr, ipAdEntIfIndex, ipAdEntNetMask, ipAdEntBcastAddr, ipAdEntReasmMaxSize,
		} {
			a.stateIPOIDs[column+"."+address] = struct{}{}
		}
	}
	for _, route := range snapshot.Network.Routes {
		if !route.Destination.IsValid() || !route.Destination.Addr().Is4() {
			continue
		}
		destination := route.Destination.Masked().Addr().String()
		nextHop := "0.0.0.0"
		routeType := IPRouteTypeDirect
		protocol := IPRouteProtoLocal
		if !route.Connected {
			nextHop = route.NextHop.String()
			routeType = IPRouteTypeIndirect
			protocol = IPRouteProtoNetMgmt
		}
		a.registerRoute(
			destination,
			net.CIDRMask(route.Destination.Bits(), route.Destination.Addr().BitLen()),
			route.Via,
			nextHop,
			routeType,
			protocol,
		)
		for _, column := range []string{
			ipRouteDest, ipRouteIfIndex, ipRouteMetric1, ipRouteMetric2, ipRouteMetric3,
			ipRouteMetric4, ipRouteNextHop, ipRouteType, ipRouteProto, ipRouteAge,
			ipRouteMask, ipRouteMetric5, ipRouteInfo,
		} {
			a.stateIPOIDs[column+"."+destination] = struct{}{}
		}
	}
}
