package snmp

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// IP-MIB OID prefixes.
const (
	// ip (1.3.6.1.2.1.4).
	ipMIBBase = "1.3.6.1.2.1.4"

	// ipAddrTable (1.3.6.1.2.1.4.20).
	ipAddrTable         = ipMIBBase + ".20"
	ipAddrEntry         = ipAddrTable + ".1"
	ipAdEntAddr         = ipAddrEntry + ".1" // IP address (index)
	ipAdEntIfIndex      = ipAddrEntry + ".2" // Interface index
	ipAdEntNetMask      = ipAddrEntry + ".3" // Subnet mask
	ipAdEntBcastAddr    = ipAddrEntry + ".4" // Broadcast address LSB (0 or 1)
	ipAdEntReasmMaxSize = ipAddrEntry + ".5" // Max reassembly size

	// ipRouteTable (1.3.6.1.2.1.4.21) - deprecated but still commonly used.
	ipRouteTable   = ipMIBBase + ".21"
	ipRouteEntry   = ipRouteTable + ".1"
	ipRouteDest    = ipRouteEntry + ".1"  // Destination
	ipRouteIfIndex = ipRouteEntry + ".2"  // Interface index
	ipRouteMetric1 = ipRouteEntry + ".3"  // Metric
	ipRouteNextHop = ipRouteEntry + ".7"  // Next hop
	ipRouteType    = ipRouteEntry + ".8"  // Route type (1=other,2=invalid,3=direct,4=indirect)
	ipRouteProto   = ipRouteEntry + ".9"  // Protocol (1=other,2=local,3=netmgmt,4=icmp,etc.)
	ipRouteMask    = ipRouteEntry + ".11" // Subnet mask

	// ipNetToMediaTable (1.3.6.1.2.1.4.22) - ARP table.
	ipNetToMediaTable       = ipMIBBase + ".22"
	ipNetToMediaEntry       = ipNetToMediaTable + ".1"
	ipNetToMediaIfIndex     = ipNetToMediaEntry + ".1" // Interface index (index)
	ipNetToMediaPhysAddress = ipNetToMediaEntry + ".2" // MAC address
	ipNetToMediaNetAddress  = ipNetToMediaEntry + ".3" // IP address (index)
	ipNetToMediaType        = ipNetToMediaEntry + ".4" // Type (1=other,2=invalid,3=dynamic,4=static)
)

// initializeIPMIB populates IP-MIB tables (ipAddrTable, ipRouteTable, ipNetToMediaTable).
func (a *Agent) initializeIPMIB() {
	logger := slog.Default()
	device := a.device
	if device == nil {
		return
	}

	ips := device.IPAddresses
	if len(ips) == 0 {
		return
	}

	// Register ipAddrTable entries
	a.registerIPAddrTableEntries(device, ips)

	// Register default route
	a.registerDefaultRouteEntry(ips)

	// Register ARP table entries
	a.registerARPTableEntries(device)

	if a.debugLevel >= DebugLevelMinimum {
		logger.Info("Initialized IP-MIB", "device", device.Name, "ipAddresses", len(ips))
	}
}

// registerIPAddrTableEntries registers ipAddrTable entries for each IPv4 address.
func (a *Agent) registerIPAddrTableEntries(device *config.Device, ips []net.IP) {
	for idx, ip := range ips {
		// Skip IPv6 addresses for ipAddrTable (IPv4 only)
		if ip.To4() == nil {
			continue
		}

		ipStr := ip.String()
		ifIndex := 1
		if idx < len(device.TrunkPorts) {
			ifIndex = idx + 1
		}

		a.registerIPAddrEntry(ipStr, ifIndex)
	}
}

// registerIPAddrEntry registers a single ipAddrTable entry.
func (a *Agent) registerIPAddrEntry(ipStr string, ifIndex int) {
	a.mib.Set(ipAdEntAddr+"."+ipStr, &OIDValue{Type: gosnmp.IPAddress, Value: ipStr})
	a.mib.Set(ipAdEntIfIndex+"."+ipStr, &OIDValue{Type: gosnmp.Integer, Value: ifIndex})
	a.mib.Set(ipAdEntNetMask+"."+ipStr, &OIDValue{Type: gosnmp.IPAddress, Value: "255.255.255.0"})
	a.mib.Set(ipAdEntBcastAddr+"."+ipStr, &OIDValue{Type: gosnmp.Integer, Value: 1})
	a.mib.Set(ipAdEntReasmMaxSize+"."+ipStr, &OIDValue{Type: gosnmp.Integer, Value: MaxIPPort})
}

// registerDefaultRouteEntry registers the default route (0.0.0.0) in ipRouteTable.
func (a *Agent) registerDefaultRouteEntry(ips []net.IP) {
	const defaultRoute = "0.0.0.0"

	a.mib.Set(ipRouteDest+"."+defaultRoute, &OIDValue{Type: gosnmp.IPAddress, Value: defaultRoute})
	a.mib.Set(ipRouteIfIndex+"."+defaultRoute, &OIDValue{Type: gosnmp.Integer, Value: 1})
	a.mib.Set(ipRouteMetric1+"."+defaultRoute, &OIDValue{Type: gosnmp.Integer, Value: 1})

	// Compute default gateway from first IPv4 address
	if len(ips) > 0 && ips[0].To4() != nil {
		ip4 := ips[0].To4()
		gateway := fmt.Sprintf("%d.%d.%d.1", ip4[0], ip4[1], ip4[2])
		a.mib.Set(
			ipRouteNextHop+"."+defaultRoute,
			&OIDValue{Type: gosnmp.IPAddress, Value: gateway},
		)
	}

	a.mib.Set(ipRouteType+"."+defaultRoute, &OIDValue{Type: gosnmp.Integer, Value: IPRouteTypeIndirect})
	a.mib.Set(ipRouteMask+"."+defaultRoute, &OIDValue{Type: gosnmp.IPAddress, Value: defaultRoute})
	a.mib.Set(ipRouteProto+"."+defaultRoute, &OIDValue{Type: gosnmp.Integer, Value: IPRouteProtoLocal})
}

// registerARPTableEntries registers ipNetToMediaTable (ARP) entries for neighbors.
func (a *Agent) registerARPTableEntries(device *config.Device) {
	arpIndex := 1
	for _, trunk := range device.TrunkPorts {
		if trunk.RemoteDevice == "" {
			continue
		}
		a.registerARPEntry(arpIndex)
		arpIndex++
	}
}

// registerARPEntry registers a single ARP table entry.
func (a *Agent) registerARPEntry(arpIndex int) {
	neighborIP := fmt.Sprintf("10.0.0.%d", arpIndex)
	indexStr := fmt.Sprintf("%d.%s", arpIndex, neighborIP)

	a.mib.Set(ipNetToMediaIfIndex+"."+indexStr, &OIDValue{Type: gosnmp.Integer, Value: arpIndex})
	macBytes := []byte{0x00, 0x00, 0x00, 0x00, 0x00, byte(arpIndex)}
	a.mib.Set(
		ipNetToMediaPhysAddress+"."+indexStr,
		&OIDValue{Type: gosnmp.OctetString, Value: macBytes},
	)
	a.mib.Set(
		ipNetToMediaNetAddress+"."+indexStr,
		&OIDValue{Type: gosnmp.IPAddress, Value: neighborIP},
	)
	a.mib.Set(ipNetToMediaType+"."+indexStr, &OIDValue{Type: gosnmp.Integer, Value: ARPTypeDynamic})
}
