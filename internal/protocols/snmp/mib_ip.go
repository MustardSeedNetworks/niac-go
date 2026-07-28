package snmp

import (
	"log/slog"
	"net"
	"strconv"
	"sync/atomic"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const (
	ipForwarding      = ipMIBBase + ".1.0"
	ipDefaultTTL      = ipMIBBase + ".2.0"
	ipInReceives      = ipMIBBase + ".3.0"
	ipInHdrErrors     = ipMIBBase + ".4.0"
	ipInAddrErrors    = ipMIBBase + ".5.0"
	ipForwDatagrams   = ipMIBBase + ".6.0"
	ipInUnknownProtos = ipMIBBase + ".7.0"
	ipInDiscards      = ipMIBBase + ".8.0"
	ipInDelivers      = ipMIBBase + ".9.0"
	ipOutRequests     = ipMIBBase + ".10.0"
	ipOutDiscards     = ipMIBBase + ".11.0"
	ipOutNoRoutes     = ipMIBBase + ".12.0"
	ipReasmTimeout    = ipMIBBase + ".13.0"
	ipReasmReqds      = ipMIBBase + ".14.0"
	ipReasmOKs        = ipMIBBase + ".15.0"
	ipReasmFails      = ipMIBBase + ".16.0"
	ipFragOKs         = ipMIBBase + ".17.0"
	ipFragFails       = ipMIBBase + ".18.0"
	ipFragCreates     = ipMIBBase + ".19.0"

	atEntry       = "1.3.6.1.2.1.3.1.1"
	atIfIndex     = atEntry + ".1"
	atPhysAddress = atEntry + ".2"
	atNetAddress  = atEntry + ".3"

	ipForwardingEnabled  = 1
	ipForwardingDisabled = 2
	defaultIPTTL         = 64
	defaultReasmTimeout  = 60
	unusedRouteMetric    = -1
)

func (a *Agent) initializeIPMIB() {
	logger := slog.Default()
	device := a.device
	if device == nil {
		return
	}

	a.registerIPScalars(device)

	a.registerConfiguredRoutes(device)
	a.registerARPTableEntries(device)

	if a.debugLevel >= DebugLevelMinimum {
		logger.Info("Initialized IP-MIB", "device", device.Name, "interfaces", len(device.Interfaces))
	}
}

func (a *Agent) registerIPScalars(device *config.Device) {
	forwarding := ipForwardingDisabled
	if isForwardingDevice(device.Type) {
		forwarding = ipForwardingEnabled
	}
	ttl := defaultIPTTL
	if device.OSFingerprintConfig != nil && device.OSFingerprintConfig.TTL > 0 {
		ttl = int(device.OSFingerprintConfig.TTL)
	}
	a.mib.Set(ipForwarding, &OIDValue{Type: gosnmp.Integer, Value: forwarding})
	a.mib.Set(ipDefaultTTL, &OIDValue{Type: gosnmp.Integer, Value: ttl})
	for _, oid := range []string{
		ipInReceives, ipInHdrErrors, ipInAddrErrors, ipForwDatagrams, ipInUnknownProtos,
		ipInDiscards, ipInDelivers, ipOutRequests, ipOutDiscards, ipOutNoRoutes,
		ipReasmReqds, ipReasmOKs, ipReasmFails, ipFragOKs, ipFragFails, ipFragCreates,
	} {
		a.mib.Set(oid, &OIDValue{Type: gosnmp.Counter32, Value: uint32(0)})
	}
	a.mib.Set(ipReasmTimeout, &OIDValue{Type: gosnmp.Integer, Value: defaultReasmTimeout})
	a.registerLiveIPScalars()
}

func (a *Agent) registerLiveIPScalars() {
	for oid, counter := range map[string]*atomic.Uint32{
		ipInReceives:      &a.protocolStats.ipInReceives,
		ipForwDatagrams:   &a.protocolStats.ipForwDatagrams,
		ipInUnknownProtos: &a.protocolStats.ipInUnknownProtos,
		ipInDelivers:      &a.protocolStats.ipInDelivers,
		ipOutRequests:     &a.protocolStats.ipOutRequests,
		ipReasmReqds:      &a.protocolStats.ipReasmReqds,
		ipReasmOKs:        &a.protocolStats.ipReasmOKs,
		ipReasmFails:      &a.protocolStats.ipReasmFails,
		ipFragCreates:     &a.protocolStats.ipFragCreates,
	} {
		a.mib.SetDynamic(oid, func() *OIDValue {
			return &OIDValue{Type: gosnmp.Counter32, Value: counter.Load()}
		})
	}
}

// registerConfiguredRoutes exposes the scenario's connected and static routes
// through the MIB-II ipRouteTable.  CyberScope uses this table for path and
// gateway presentation, so the replay must describe the same routing fabric as
// the packet resolver.
func (a *Agent) registerConfiguredRoutes(device *config.Device) {
	a.registerIPAddrTableEntries(device)
	for _, iface := range device.Interfaces {
		_, network, err := net.ParseCIDR(iface.Address)
		if err != nil {
			continue
		}
		destination := network.IP.String()
		a.registerRoute(destination, network.Mask, iface.Name, "0.0.0.0", IPRouteTypeDirect, IPRouteProtoLocal)
	}
	for _, route := range device.Routes {
		_, network, err := net.ParseCIDR(route.Destination)
		if err != nil {
			continue
		}
		a.registerRoute(
			network.IP.String(),
			network.Mask,
			route.Via,
			route.NextHop,
			IPRouteTypeIndirect,
			IPRouteProtoNetMgmt,
		)
	}
}

func (a *Agent) registerRoute(
	destination string,
	mask net.IPMask,
	interfaceName string,
	nextHop string,
	routeType int,
	proto int,
) {
	ifIndex, ok := a.InterfaceIndex(interfaceName)
	if !ok {
		return
	}
	a.mib.Set(ipRouteDest+"."+destination, &OIDValue{Type: gosnmp.IPAddress, Value: destination})
	a.mib.Set(ipRouteIfIndex+"."+destination, &OIDValue{Type: gosnmp.Integer, Value: ifIndex})
	a.mib.Set(ipRouteMetric1+"."+destination, &OIDValue{Type: gosnmp.Integer, Value: 1})
	for _, oid := range []string{ipRouteMetric2, ipRouteMetric3, ipRouteMetric4, ipRouteMetric5} {
		a.mib.Set(oid+"."+destination, &OIDValue{Type: gosnmp.Integer, Value: unusedRouteMetric})
	}
	a.mib.Set(ipRouteNextHop+"."+destination, &OIDValue{Type: gosnmp.IPAddress, Value: nextHop})
	a.mib.Set(ipRouteType+"."+destination, &OIDValue{Type: gosnmp.Integer, Value: routeType})
	a.mib.Set(ipRouteMask+"."+destination, &OIDValue{Type: gosnmp.IPAddress, Value: net.IP(mask).String()})
	a.mib.Set(ipRouteProto+"."+destination, &OIDValue{Type: gosnmp.Integer, Value: proto})
	a.mib.Set(ipRouteAge+"."+destination, &OIDValue{Type: gosnmp.Integer, Value: 0})
	a.mib.Set(ipRouteInfo+"."+destination, &OIDValue{Type: gosnmp.ObjectIdentifier, Value: "0.0"})
}

// InterfaceIndex resolves an interface name through the active IF-MIB.
func (a *Agent) InterfaceIndex(name string) (int, bool) {
	if name == "" {
		return 0, false
	}
	index, ok := a.mib.IndexSuffixForValue(ifName, name)
	if !ok {
		index, ok = a.mib.IndexSuffixForValue(ifDescr, name)
	}
	if !ok {
		return 0, false
	}
	value, err := strconv.Atoi(index)
	return value, err == nil
}

// registerIPAddrTableEntries registers ipAddrTable entries for each IPv4 address.
func (a *Agent) registerIPAddrTableEntries(device *config.Device) {
	for _, iface := range device.Interfaces {
		ip, network, err := net.ParseCIDR(iface.Address)
		if err != nil || ip.To4() == nil {
			continue
		}
		ifIndex, ok := a.InterfaceIndex(iface.Name)
		if !ok {
			continue
		}
		a.registerIPAddrEntry(ip.String(), ifIndex, network.Mask)
	}
}

// registerIPAddrEntry registers a single ipAddrTable entry.
func (a *Agent) registerIPAddrEntry(ipStr string, ifIndex int, mask net.IPMask) {
	a.mib.Set(ipAdEntAddr+"."+ipStr, &OIDValue{Type: gosnmp.IPAddress, Value: ipStr})
	a.mib.Set(ipAdEntIfIndex+"."+ipStr, &OIDValue{Type: gosnmp.Integer, Value: ifIndex})
	a.mib.Set(ipAdEntNetMask+"."+ipStr, &OIDValue{Type: gosnmp.IPAddress, Value: net.IP(mask).String()})
	ones, bits := mask.Size()
	broadcastLSB := 0
	if ones >= 0 && ones < bits {
		broadcastLSB = 1
	}
	a.mib.Set(ipAdEntBcastAddr+"."+ipStr, &OIDValue{Type: gosnmp.Integer, Value: broadcastLSB})
	a.mib.Set(ipAdEntReasmMaxSize+"."+ipStr, &OIDValue{Type: gosnmp.Integer, Value: MaxIPPort})
}

// registerARPTableEntries registers ipNetToMediaTable (ARP) entries for neighbors.
func (a *Agent) registerARPTableEntries(_ *config.Device) {
	// Device topology records names and ports, but no peer IP/MAC binding. Leave
	// both ARP tables empty until an authoritative binding is available.
}

func (a *Agent) registerARPEntry(ifIndex int, ip net.IP, mac net.HardwareAddr) {
	ipv4 := ip.To4()
	if ifIndex <= 0 || ipv4 == nil || len(mac) == 0 {
		return
	}
	ipString := ipv4.String()
	index := strconv.Itoa(ifIndex) + "." + ipString
	a.mib.Set(ipNetToMediaIfIndex+"."+index, &OIDValue{Type: gosnmp.Integer, Value: ifIndex})
	a.mib.Set(ipNetToMediaPhysAddress+"."+index, &OIDValue{Type: gosnmp.OctetString, Value: []byte(mac)})
	a.mib.Set(ipNetToMediaNetAddress+"."+index, &OIDValue{Type: gosnmp.IPAddress, Value: ipString})
	a.mib.Set(ipNetToMediaType+"."+index, &OIDValue{Type: gosnmp.Integer, Value: ARPTypeDynamic})

	legacyIndex := strconv.Itoa(ifIndex) + ".1." + ipString
	a.mib.Set(atIfIndex+"."+legacyIndex, &OIDValue{Type: gosnmp.Integer, Value: ifIndex})
	a.mib.Set(atPhysAddress+"."+legacyIndex, &OIDValue{Type: gosnmp.OctetString, Value: []byte(mac)})
	a.mib.Set(atNetAddress+"."+legacyIndex, &OIDValue{Type: gosnmp.IPAddress, Value: ipString})
}
