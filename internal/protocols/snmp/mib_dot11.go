package snmp

import (
	"math"
	"net"
	"strings"

	"github.com/gosnmp/gosnmp"
)

// IEEE802dot11-MIB object prefixes. The tree hangs off the IEEE arc
// (1.2.840.10036), not under mib-2, and every table below is indexed by the
// ifIndex of the radio interface -- the MIB says so, and the AIR-AP1200
// capture in the walk corpus answers them that way.
//
// Only the objects an authored radio has something true to say are registered.
// The regulatory domain, the supported-rate sets and the antenna tables are
// not authored anywhere, and a plausible-looking default in a discovery tool's
// report is worse than a gap.
const (
	dot11Root = "1.2.840.10036"

	// dot11StationConfigTable (1.2.840.10036.1.1), indexed by ifIndex.
	dot11StationConfigEntry = dot11Root + ".1.1.1"
	dot11StationID          = dot11StationConfigEntry + ".1"
	dot11DesiredSSID        = dot11StationConfigEntry + ".9"

	// dot11OperationTable (1.2.840.10036.2.1), indexed by ifIndex.
	dot11MACAddress = dot11Root + ".2.1.1.1"

	// dot11PhyOperationTable (1.2.840.10036.4.1), indexed by ifIndex.
	dot11PHYType = dot11Root + ".4.1.1.1"

	// dot11PhyTxPowerTable (1.2.840.10036.4.3), indexed by ifIndex.
	dot11PhyTxPowerEntry            = dot11Root + ".4.3.1"
	dot11NumberSupportedPowerLevels = dot11PhyTxPowerEntry + ".1"
	dot11TxPowerLevel1              = dot11PhyTxPowerEntry + ".2"
	dot11CurrentTxPowerLevel        = dot11PhyTxPowerEntry + ".10"

	// dot11CurrentChannel is in the DSSS table (1.2.840.10036.4.5) and
	// dot11CurrentFrequency in the OFDM table (1.2.840.10036.4.11). Which one
	// answers is the band: the MIB gives DSSS a channel and OFDM a frequency
	// channel number, and the corpus AP's own 5 GHz radio returns no DSSS row.
	dot11CurrentChannel   = dot11Root + ".4.5.1.1"
	dot11CurrentFrequency = dot11Root + ".4.11.1.1"
)

// dot11PHYType values (IEEE802dot11-MIB). The enumeration was written for the
// 1999 PHYs and never grew a 6 GHz member, so a 6 GHz radio reports the OFDM it
// is: the generation detail (HE, EHT) is a vendor-MIB matter.
const (
	dot11PHYTypeDSSS = 2
	dot11PHYTypeOFDM = 4

	// dot11CurrentTxPowerLevel indexes the dot11TxPowerLevelN columns. NIAC
	// authors one power per radio, so the level in use is always the first.
	dot11SingleTxPowerLevel = 1

	// decibelDecade is the ten of dBm's own definition: ten decibels is one
	// factor of ten in power.
	decibelDecade = 10
)

// initializeDot11MIB registers IEEE802dot11-MIB for a device with authored
// radios. A walk-backed device gets nothing here: the capture owns its own
// radios and the interface indexes these rows are keyed by, and
// refreshWalkedDot11MIB decides once the walk has loaded.
func (a *Agent) initializeDot11MIB() {
	if a.device == nil || a.device.WiFiConfig == nil || a.hasWalkContent() {
		return
	}

	a.registerDot11MIB()
}

// refreshWalkedDot11MIB serves the authored radios of a device whose capture
// carried no IEEE802dot11-MIB. A capture that did keeps it untouched: a real AP
// reports its own radio numbering and rates, and replacing that with the
// authored pair would both lie and count as an unclassified substitution in the
// replay-fidelity contract.
func (a *Agent) refreshWalkedDot11MIB(walkOwnsDot11 bool) {
	if a.device == nil || a.device.WiFiConfig == nil || walkOwnsDot11 {
		return
	}

	a.registerDot11MIB()
}

// walkOwnsDot11 reports whether a parsed capture carries IEEE802dot11-MIB of
// its own. Any object under the tree counts, as it does for POWER-ETHERNET-MIB:
// an AP that answers part of the MIB is still the authority on all of it.
func walkOwnsDot11(entries []WalkEntry) bool {
	for _, entry := range entries {
		if strings.HasPrefix(strings.TrimPrefix(entry.OID, "."), dot11Root+".") {
			return true
		}
	}

	return false
}

func (a *Agent) registerDot11MIB() {
	for _, radio := range a.device.WiFiConfig.Radios {
		index, ok := a.ifIndexForInterface(radio.Interface)
		if !ok {
			continue
		}
		bssid, err := net.ParseMAC(radio.BSSID)
		if err != nil {
			continue
		}
		suffix := "." + index

		a.mib.Set(dot11StationID+suffix, macValue(bssid))
		a.mib.Set(dot11MACAddress+suffix, macValue(bssid))
		// The radio answers on the BSSID, so IF-MIB reports the same address:
		// the two disagreeing is a finding a tester would raise against the
		// simulated AP rather than a difference anyone authored.
		a.mib.Set(ifPhysAddress+suffix, macValue(bssid))
		a.mib.Set(dot11DesiredSSID+suffix,
			&OIDValue{Type: gosnmp.OctetString, Value: radio.SSID})

		a.registerDot11Phy(radio.Band, radio.Channel, radio.TxPowerDBM, suffix)
	}
}

func (a *Agent) registerDot11Phy(band string, channel, txPowerDBM int, suffix string) {
	phyType, channelOID := dot11PhyForBand(band)
	if channelOID == "" {
		return
	}
	a.mib.Set(dot11PHYType+suffix, &OIDValue{Type: gosnmp.Integer, Value: phyType})
	a.mib.Set(channelOID+suffix, &OIDValue{Type: gosnmp.Integer, Value: channel})

	a.mib.Set(dot11NumberSupportedPowerLevels+suffix,
		&OIDValue{Type: gosnmp.Integer, Value: dot11SingleTxPowerLevel})
	a.mib.Set(dot11TxPowerLevel1+suffix,
		&OIDValue{Type: gosnmp.Integer, Value: milliwattsFromDBM(txPowerDBM)})
	a.mib.Set(dot11CurrentTxPowerLevel+suffix,
		&OIDValue{Type: gosnmp.Integer, Value: dot11SingleTxPowerLevel})
}

// dot11PhyForBand returns the PHY type a band reports and the object that
// carries its channel. An unknown band returns no object: the validator refuses
// one, so reaching here means the config was built in memory rather than
// authored, and inventing a PHY for it would put a guess on the wire.
func dot11PhyForBand(band string) (int, string) {
	switch band {
	case "2.4GHz":
		return dot11PHYTypeDSSS, dot11CurrentChannel
	case "5GHz", "6GHz":
		return dot11PHYTypeOFDM, dot11CurrentFrequency
	default:
		return 0, ""
	}
}

// milliwattsFromDBM converts authored transmit power into the unit
// dot11TxPowerLevelN is defined in. The validator caps the input at 30 dBm, so
// the result stays inside the MIB's 0..10000 mW range.
func milliwattsFromDBM(dBm int) int {
	return int(math.Round(math.Pow(decibelDecade, float64(dBm)/decibelDecade)))
}

// authoredRadioAddresses maps the ifIndex of every authored radio to its BSSID.
// Anything that rewrites interface addresses consults this first: the BSSID is
// authored truth about one interface, while a derived address is a stand-in for
// an interface nobody said anything about, so the radio keeps its own.
func (a *Agent) authoredRadioAddresses() map[string][]byte {
	if a.device == nil || a.device.WiFiConfig == nil {
		return nil
	}
	radios := make(map[string][]byte, len(a.device.WiFiConfig.Radios))
	for _, radio := range a.device.WiFiConfig.Radios {
		index, ok := a.ifIndexForInterface(radio.Interface)
		if !ok {
			continue
		}
		if bssid, err := net.ParseMAC(radio.BSSID); err == nil {
			radios[index] = bssid
		}
	}

	return radios
}

func macValue(address net.HardwareAddr) *OIDValue {
	return &OIDValue{Type: gosnmp.OctetString, Value: []byte(address)}
}
