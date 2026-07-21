package snmp

import (
	"strconv"

	"github.com/gosnmp/gosnmp"
)

const egpMIBRoot = "1.3.6.1.2.1.8"

// initializeEGPMIB registers RFC 1213's mandatory EGP scalars. NIAC has no EGP
// behavior or configured peers, so the neighbor table correctly remains empty.
func (a *Agent) initializeEGPMIB() {
	for id := 1; id <= 4; id++ {
		a.mib.Set(egpMIBRoot+"."+strconv.Itoa(id)+".0", &OIDValue{Type: gosnmp.Counter32, Value: uint32(0)})
	}
	a.mib.Set(egpMIBRoot+".6.0", &OIDValue{Type: gosnmp.Integer, Value: 0})
}

func (a *Agent) initializeMIBIIProtocolGroups() {
	a.initializeICMPMIB()
	a.initializeTransportMIBs()
	a.initializeEGPMIB()
}
