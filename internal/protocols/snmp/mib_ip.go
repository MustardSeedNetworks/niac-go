package snmp

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

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
	macBytes := []byte{0x00, 0x00, 0x00, 0x00, 0x00, safeconv.Byte(arpIndex)}
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
