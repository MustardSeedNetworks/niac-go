package snmp

import (
	"net"
	"slices"
	"strconv"
	"sync/atomic"

	"github.com/gosnmp/gosnmp"
)

const (
	tcpMIBRoot          = "1.3.6.1.2.1.6"
	udpMIBRoot          = "1.3.6.1.2.1.7"
	tcpListenState      = 2
	defaultHTTPPort     = 80
	defaultFTPPort      = 21
	defaultSNMPPort     = 161
	defaultIPerfPort    = 5201
	dnsListenerPort     = 53
	dhcpListenerPort    = 67
	netBIOSNamePort     = 137
	netBIOSDataPort     = 138
	netBIOSSessionPort  = 139
	tcpListenerCapacity = 4
)

func (a *Agent) initializeTransportMIBs() {
	a.initializeTCPMIB()
	a.initializeUDPMIB()
}

func (a *Agent) initializeTCPMIB() {
	values := []struct {
		id     int
		typeID gosnmp.Asn1BER
		value  any
	}{
		{1, gosnmp.Integer, 4},
		{2, gosnmp.Integer, 200},
		{3, gosnmp.Integer, 120000},
		{4, gosnmp.Integer, -1},
		{5, gosnmp.Counter32, uint32(0)},
		{6, gosnmp.Counter32, uint32(0)},
		{7, gosnmp.Counter32, uint32(0)},
		{8, gosnmp.Counter32, uint32(0)},
		{9, gosnmp.Gauge32, uint32(0)},
		{10, gosnmp.Counter32, uint32(0)},
		{11, gosnmp.Counter32, uint32(0)},
		{12, gosnmp.Counter32, uint32(0)},
		{14, gosnmp.Counter32, uint32(0)},
		{15, gosnmp.Counter32, uint32(0)},
	}
	for _, scalar := range values {
		a.mib.Set(tcpMIBRoot+"."+strconv.Itoa(scalar.id)+".0", &OIDValue{Type: scalar.typeID, Value: scalar.value})
	}
	a.registerLiveTCPCounters()

	for _, ip := range mibIIIPv4Addresses(a.device.IPAddresses) {
		for _, port := range a.configuredTCPPorts() {
			index := ip + "." + strconv.Itoa(port) + ".0.0.0.0.0"
			entry := tcpMIBRoot + ".13.1"
			a.mib.Set(entry+".1."+index, &OIDValue{Type: gosnmp.Integer, Value: tcpListenState})
			a.mib.Set(entry+".2."+index, &OIDValue{Type: gosnmp.IPAddress, Value: ip})
			a.mib.Set(entry+".3."+index, &OIDValue{Type: gosnmp.Integer, Value: port})
			a.mib.Set(entry+".4."+index, &OIDValue{Type: gosnmp.IPAddress, Value: "0.0.0.0"})
			a.mib.Set(entry+".5."+index, &OIDValue{Type: gosnmp.Integer, Value: 0})
		}
	}
}

func (a *Agent) registerLiveTCPCounters() {
	for oid, counter := range map[string]*atomic.Uint32{
		tcpMIBRoot + ".5.0":  &a.protocolStats.tcpActiveOpens,
		tcpMIBRoot + ".6.0":  &a.protocolStats.tcpPassiveOpens,
		tcpMIBRoot + ".7.0":  &a.protocolStats.tcpAttemptFails,
		tcpMIBRoot + ".8.0":  &a.protocolStats.tcpEstabResets,
		tcpMIBRoot + ".10.0": &a.protocolStats.tcpInSegs,
		tcpMIBRoot + ".11.0": &a.protocolStats.tcpOutSegs,
		tcpMIBRoot + ".15.0": &a.protocolStats.tcpOutRsts,
	} {
		a.setProtocolCounter(oid, counter)
	}
	a.mib.SetDynamic(tcpMIBRoot+".9.0", func() *OIDValue {
		return &OIDValue{Type: gosnmp.Gauge32, Value: a.protocolStats.tcpFlows.currentEstablished()}
	})
}

func (a *Agent) initializeUDPMIB() {
	for id := 1; id <= 4; id++ {
		a.mib.Set(udpMIBRoot+"."+strconv.Itoa(id)+".0", &OIDValue{Type: gosnmp.Counter32, Value: uint32(0)})
	}
	a.registerLiveUDPCounters()
	for _, ip := range mibIIIPv4Addresses(a.device.IPAddresses) {
		for _, port := range a.configuredUDPPorts() {
			index := ip + "." + strconv.Itoa(port)
			entry := udpMIBRoot + ".5.1"
			a.mib.Set(entry+".1."+index, &OIDValue{Type: gosnmp.IPAddress, Value: ip})
			a.mib.Set(entry+".2."+index, &OIDValue{Type: gosnmp.Integer, Value: port})
		}
	}
}

func (a *Agent) registerLiveUDPCounters() {
	a.setProtocolCounter(udpMIBRoot+".1.0", &a.protocolStats.udpInDatagrams)
	a.setProtocolCounter(udpMIBRoot+".2.0", &a.protocolStats.udpNoPorts)
	a.setProtocolCounter(udpMIBRoot+".4.0", &a.protocolStats.udpOutDatagrams)
}

func (a *Agent) configuredTCPPorts() []int {
	ports := make([]int, 0, tcpListenerCapacity)
	if a.device.HTTPConfig != nil && a.device.HTTPConfig.Enabled {
		ports = append(ports, defaultHTTPPort)
	}
	if a.device.FTPConfig != nil && a.device.FTPConfig.Enabled {
		ports = append(ports, defaultFTPPort)
	}
	if a.device.NetBIOSConfig != nil && a.device.NetBIOSConfig.Enabled {
		ports = append(ports, netBIOSSessionPort)
	}
	if a.device.IPerf3 != nil && a.device.IPerf3.Enabled {
		port := int(a.device.IPerf3.Port)
		if port == 0 {
			port = defaultIPerfPort
		}
		ports = append(ports, port)
	}

	return ports
}

func (a *Agent) configuredUDPPorts() []int {
	ports := []int{defaultSNMPPort}
	if a.device.DHCPConfig != nil {
		ports = append(ports, dhcpListenerPort)
	}
	if a.device.DNSConfig != nil {
		ports = append(ports, dnsListenerPort)
	}
	if a.device.NetBIOSConfig != nil && a.device.NetBIOSConfig.Enabled {
		ports = append(ports, netBIOSNamePort, netBIOSDataPort)
	}

	return ports
}

func mibIIIPv4Addresses(addresses []net.IP) []string {
	result := make([]string, 0, len(addresses))
	for _, address := range addresses {
		if ipv4 := address.To4(); ipv4 != nil {
			value := ipv4.String()
			if !slices.Contains(result, value) {
				result = append(result, value)
			}
		}
	}

	return result
}
