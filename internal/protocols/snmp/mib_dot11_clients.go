package snmp

import (
	"net"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// CISCO-DOT11-ASSOCIATION-MIB object prefixes. IEEE802dot11-MIB has no client
// table -- it says what a radio is, never who is on it -- so the associated
// stations come from the vendor family, and the AIR-AP1200 capture in the walk
// corpus (walks/raw/cisco/cisco-c1200-01.walk) is what these arcs are read off.
//
// Only the objects an authored client has something true to say are
// registered. The capture answers the rate sets, the cipher columns, the
// association identifier and twenty more, and nothing in a scenario authors
// any of them.
const (
	cDot11AssociationRoot = "1.3.6.1.4.1.9.9.273"

	// cDot11ClientConfigInfoTable (…273.1.2.1), indexed by ifIndex, the SSID
	// the station associated on, and the station's MAC.
	cDot11ClientConfigEntry   = cDot11AssociationRoot + ".1.2.1.1"
	cDot11ClientParentAddress = cDot11ClientConfigEntry + ".2"
	cDot11ClientIPAddressType = cDot11ClientConfigEntry + ".15"
	cDot11ClientIPAddress     = cDot11ClientConfigEntry + ".16"

	// cDot11ClientStatisticTable (…273.1.3.1) augments the table above, so it
	// carries the same index.
	cDot11ClientStatisticEntry = cDot11AssociationRoot + ".1.3.1.1"
	cDot11ClientUpTime         = cDot11ClientStatisticEntry + ".2"
	cDot11ClientSignalStrength = cDot11ClientStatisticEntry + ".3"
	cDot11ClientSigQuality     = cDot11ClientStatisticEntry + ".4"
)

// initializeDot11ClientMIB registers the associated stations of a device with
// authored radios. A walk-backed device gets nothing here, for the reason the
// radios themselves do: the capture owns the ifIndexes the rows are keyed by,
// so refreshWalkedDot11ClientMIB decides once it has loaded.
func (a *Agent) initializeDot11ClientMIB() {
	if a.device == nil || a.device.WiFiConfig == nil || a.hasWalkContent() {
		return
	}

	a.registerDot11ClientMIB()
}

// refreshWalkedDot11ClientMIB serves the authored stations of a device whose
// capture carried no association table. A capture that has one keeps it: a real
// AP is the authority on who is associated to it, and replacing that would
// count as an unclassified substitution in the replay-fidelity contract.
//
// This is a separate decision from the radios': an AP can answer
// IEEE802dot11-MIB and no association table, and then its authored clients are
// the only thing that can fill it.
func (a *Agent) refreshWalkedDot11ClientMIB(walkOwnsClients bool) {
	if a.device == nil || a.device.WiFiConfig == nil || walkOwnsClients {
		return
	}

	a.registerDot11ClientMIB()
}

// walkOwnsDot11Clients reports whether a parsed capture carries an association
// table of its own. Any object under the tree counts, as it does for the radios
// and for POWER-ETHERNET-MIB: an AP that answers part of the MIB is still the
// authority on all of it.
func walkOwnsDot11Clients(entries []WalkEntry) bool {
	for _, entry := range entries {
		if strings.HasPrefix(strings.TrimPrefix(entry.OID, "."), cDot11AssociationRoot+".") {
			return true
		}
	}

	return false
}

func (a *Agent) registerDot11ClientMIB() {
	for _, radio := range a.device.WiFiConfig.Radios {
		index, ok := a.ifIndexForInterface(radio.Interface)
		if !ok {
			continue
		}
		bssid, err := net.ParseMAC(radio.BSSID)
		if err != nil {
			continue
		}
		for _, client := range radio.Clients {
			a.registerDot11Client(client, index, radio.SSID, bssid)
		}
	}
}

func (a *Agent) registerDot11Client(
	client config.WiFiClient,
	ifIndex, ssid string,
	bssid net.HardwareAddr,
) {
	station, err := net.ParseMAC(client.MAC)
	if err != nil {
		return
	}
	address := net.ParseIP(client.IPAddress).To4()
	if address == nil {
		return
	}
	suffix := "." + dot11ClientIndex(ifIndex, ssid, station)

	// The AP a station is on is its radio's BSSID -- the same address
	// dot11MACAddress reports for that radio, so an NMS reading both tables
	// sees one AP rather than two.
	a.mib.Set(cDot11ClientParentAddress+suffix, macValue(bssid))
	a.mib.Set(cDot11ClientIPAddressType+suffix,
		&OIDValue{Type: gosnmp.Integer, Value: inetAddressTypeIPv4})
	a.mib.Set(cDot11ClientIPAddress+suffix,
		&OIDValue{Type: gosnmp.OctetString, Value: []byte(address)})
	a.mib.Set(cDot11ClientUpTime+suffix,
		&OIDValue{Type: gosnmp.Gauge32, Value: client.AssociatedSeconds})
	a.mib.Set(cDot11ClientSignalStrength+suffix,
		&OIDValue{Type: gosnmp.Integer, Value: client.SignalDBM})
	a.mib.Set(cDot11ClientSigQuality+suffix,
		&OIDValue{Type: gosnmp.Gauge32, Value: client.SignalQualityPct})
}

// dot11ClientIndex builds the index both association tables are keyed by: the
// radio's ifIndex, then the SSID as a length-prefixed octet string, then the
// station's MAC as six bare octets. MacAddress is fixed-length, so it carries
// no length of its own -- the capture's own row encodes it exactly this way.
func dot11ClientIndex(ifIndex, ssid string, station net.HardwareAddr) string {
	arcs := make([]string, 0, 1+1+len(ssid)+len(station))
	arcs = append(arcs, ifIndex, strconv.Itoa(len(ssid)))
	for _, octet := range []byte(ssid) {
		arcs = append(arcs, strconv.Itoa(int(octet)))
	}
	for _, octet := range station {
		arcs = append(arcs, strconv.Itoa(int(octet)))
	}

	return strings.Join(arcs, ".")
}
