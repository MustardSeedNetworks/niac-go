package scenario

import (
	"fmt"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

const (
	snmpTypeInteger   = "INTEGER"
	snmpTypeGauge32   = "Gauge32"
	snmpTypeHexString = "Hex-STRING"
	snmpTypeCounter32 = "Counter32"
	snmpTypeString    = "STRING"

	// dot11 columns an access point's radios answer that the authored `wifi`
	// block does not carry. The ones it does carry -- station ID, desired
	// SSID, MAC address, PHY type, channel and transmit power -- are served
	// once, by internal/protocols/snmp from the block itself, and must not
	// appear here: two sources for one object is #2163.
	dot11StationMediumOccupancyLimitColumn          = 2
	dot11StationCFPollableColumn                    = 3
	dot11StationCFPPeriodColumn                     = 4
	dot11StationCFPMaxDurationColumn                = 5
	dot11StationAuthenticationResponseTimeoutColumn = 6
	dot11StationPrivacyOptionImplementedColumn      = 7
	dot11StationPowerManagementModeColumn           = 8
	dot11StationDesiredBSSTypeColumn                = 10
	dot11StationOperationalRateSetColumn            = 11
	dot11StationBeaconPeriodColumn                  = 12
	dot11StationDTIMPeriodColumn                    = 13
	dot11StationAssociationResponseTimeoutColumn    = 14
	dot11StationDisassociateReasonColumn            = 15
	dot11StationDisassociateStationColumn           = 16
	dot11StationDeauthenticateReasonColumn          = 17
	dot11StationDeauthenticateStationColumn         = 18
	dot11StationAuthenticateFailStatusColumn        = 19
	dot11StationAuthenticateFailStationColumn       = 20
	dot11StationMultiDomainImplementedColumn        = 21
	dot11StationMultiDomainActivatedColumn          = 22
	dot11StationCountryStringColumn                 = 23
)

// Values read off the Cisco AIR-AP1200 capture in the walk corpus
// (`walks/raw/cisco/cisco-c1200-01.walk`), which is the only real AP walk we
// have. dot11OperationTable is the MAC table -- its columns are the radio
// address, the RTS threshold and the short-retry limit -- and the generator
// used to write the PHY triple here instead, so a consumer reading a generated
// AP's RTS threshold got a regulatory domain.
const (
	dot11RTSThreshold    = "2312"
	dot11ShortRetryLimit = "64"
)

// dot11BandRow is one row template: an OID carrying a single %d for the
// radio's ifIndex, and the value the capture's 2.4 GHz and 5 GHz radios gave.
type dot11BandRow struct {
	oid          string
	typ          string
	lower, upper string
}

// apDot11MIBs returns the IEEE 802.11 capability tables a generated access
// point answers beyond its authored radios. Every table is indexed by the
// ifIndex of a radio interface, which is the agent's own numbering and not the
// radio's position in the plan: the access point's trunked uplink takes
// ifIndex 1, so its radios start at 2.
func apDot11MIBs(site string, radios []apRadioIndex) []converter.AddMib {
	mibs := apDot11StationMIBs(site, radios)
	mibs = append(mibs, apDot11OperationMIBs(radios)...)

	return append(mibs, apDot11PhyMIBs(site, radios)...)
}

// apRadioIndex pairs one radio of the plan with the ifIndex its interface was
// given.
type apRadioIndex struct {
	plan    apRadio
	ifIndex int
}

// apRadioIndexes resolves each radio of the plan to its ifIndex, through the
// same ordering the agent uses. A radio whose interface is missing is dropped
// rather than guessed at.
func apRadioIndexes(trunkPorts []converter.TrunkPort, interfaces []converter.Interface) []apRadioIndex {
	trunks := make([]string, 0, len(trunkPorts))
	for _, trunk := range trunkPorts {
		trunks = append(trunks, trunk.Interface)
	}
	authored := make([]string, 0, len(interfaces))
	for _, iface := range interfaces {
		authored = append(authored, iface.Name)
	}
	order := make(map[string]int, len(trunks)+len(authored))
	for position, name := range snmp.SynthesizedInterfaceOrder(trunks, authored) {
		order[name] = position + 1
	}

	plan := apRadioPlan()
	radios := make([]apRadioIndex, 0, len(plan))
	for index, radio := range plan {
		ifIndex, ok := order[fmt.Sprintf("Dot11Radio%d", index)]
		if !ok {
			continue
		}
		radios = append(radios, apRadioIndex{plan: radio, ifIndex: ifIndex})
	}

	return radios
}

func apDot11StationMIBs(site string, radios []apRadioIndex) []converter.AddMib {
	fields := []struct {
		column int
		typ    string
		value  string
	}{
		{column: dot11StationMediumOccupancyLimitColumn, typ: snmpTypeGauge32, value: "100"},
		{column: dot11StationCFPollableColumn, typ: snmpTypeInteger, value: "2"},
		{column: dot11StationCFPPeriodColumn, typ: snmpTypeGauge32, value: "0"},
		{column: dot11StationCFPMaxDurationColumn, typ: snmpTypeGauge32, value: "0"},
		{
			column: dot11StationAuthenticationResponseTimeoutColumn,
			typ:    snmpTypeGauge32,
			value:  "60",
		},
		{column: dot11StationPrivacyOptionImplementedColumn, typ: snmpTypeInteger, value: "2"},
		{column: dot11StationPowerManagementModeColumn, typ: snmpTypeInteger, value: "1"},
		{column: dot11StationDesiredBSSTypeColumn, typ: snmpTypeInteger, value: "1"},
		{
			column: dot11StationOperationalRateSetColumn,
			typ:    snmpTypeHexString,
			value:  "0C 12 18 24 30 48 60 6C",
		},
		{column: dot11StationBeaconPeriodColumn, typ: snmpTypeGauge32, value: "100"},
		{column: dot11StationDTIMPeriodColumn, typ: snmpTypeGauge32, value: "2"},
		{column: dot11StationAssociationResponseTimeoutColumn, typ: snmpTypeGauge32, value: "60"},
		{column: dot11StationDisassociateReasonColumn, typ: snmpTypeGauge32, value: "0"},
		{
			column: dot11StationDisassociateStationColumn,
			typ:    snmpTypeHexString,
			value:  "00 00 00 00 00 00",
		},
		{column: dot11StationDeauthenticateReasonColumn, typ: snmpTypeGauge32, value: "0"},
		{
			column: dot11StationDeauthenticateStationColumn,
			typ:    snmpTypeHexString,
			value:  "00 00 00 00 00 00",
		},
		{column: dot11StationAuthenticateFailStatusColumn, typ: snmpTypeGauge32, value: "0"},
		{
			column: dot11StationAuthenticateFailStationColumn,
			typ:    snmpTypeHexString,
			value:  "00 00 00 00 00 00",
		},
		{column: dot11StationMultiDomainImplementedColumn, typ: snmpTypeInteger, value: "1"},
		{column: dot11StationMultiDomainActivatedColumn, typ: snmpTypeInteger, value: "1"},
		{
			column: dot11StationCountryStringColumn,
			typ:    snmpTypeHexString,
			value:  apCountryString(site),
		},
	}
	mibs := make([]converter.AddMib, 0, len(fields)*len(radios))
	for _, radio := range radios {
		for _, field := range fields {
			mibs = append(mibs, converter.AddMib{
				OID:  fmt.Sprintf("1.2.840.10036.1.1.1.%d.%d", field.column, radio.ifIndex),
				Type: field.typ, Value: field.value,
			})
		}
	}

	return append(mibs, apDot11CapabilityMIBs(radios)...)
}

// apDot11OperationMIBs answers the two dot11OperationTable columns the
// authored radio does not carry. The third, dot11MACAddress, is the radio's
// BSSID and is served from the `wifi` block.
func apDot11OperationMIBs(radios []apRadioIndex) []converter.AddMib {
	const columnsPerRadio = 2
	mibs := make([]converter.AddMib, 0, columnsPerRadio*len(radios))
	for _, radio := range radios {
		suffix := fmt.Sprintf(".%d", radio.ifIndex)
		mibs = append(mibs,
			converter.AddMib{
				OID: "1.2.840.10036.2.1.1.2" + suffix, Type: snmpTypeInteger,
				Value: dot11RTSThreshold,
			},
			converter.AddMib{
				OID: "1.2.840.10036.2.1.1.3" + suffix, Type: snmpTypeInteger,
				Value: dot11ShortRetryLimit,
			},
		)
	}

	return mibs
}

// apDot11CapabilityMIBs answers the authentication, privacy and supported-rate
// tables. The capture has two radios and this access point has four, so each
// radio answers the row the capture's radio of the same band answered: the
// values are a property of the band, not of a radio's position in the list.
func apDot11CapabilityMIBs(radios []apRadioIndex) []converter.AddMib {
	rows := []dot11BandRow{
		{oid: "1.2.840.10036.1.2.1.2.%d.1", typ: snmpTypeInteger, lower: "1", upper: "1"},
		{oid: "1.2.840.10036.1.2.1.2.%d.2", typ: snmpTypeInteger, lower: "2", upper: "2"},
		{oid: "1.2.840.10036.1.2.1.2.%d.3", typ: snmpTypeInteger, lower: "129", upper: "129"},
		{oid: "1.2.840.10036.1.2.1.3.%d.1", typ: snmpTypeInteger, lower: "1", upper: "1"},
		{oid: "1.2.840.10036.1.2.1.3.%d.2", typ: snmpTypeInteger, lower: "1", upper: "1"},
		{oid: "1.2.840.10036.1.2.1.3.%d.3", typ: snmpTypeInteger, lower: "1", upper: "1"},
		{oid: "1.2.840.10036.1.5.1.1.%d", typ: snmpTypeInteger, lower: "2", upper: "2"},
		{oid: "1.2.840.10036.1.5.1.4.%d", typ: snmpTypeInteger, lower: "1", upper: "1"},
		{oid: "1.2.840.10036.1.5.1.5.%d", typ: snmpTypeCounter32, lower: "5", upper: "0"},
		{oid: "1.2.840.10036.1.5.1.6.%d", typ: snmpTypeCounter32, lower: "0", upper: "0"},
		{oid: "1.2.840.10036.1.7.1.2.%d.1", typ: snmpTypeInteger, lower: "1", upper: "36"},
		{oid: "1.2.840.10036.1.7.1.2.%d.2", typ: snmpTypeInteger, lower: "1", upper: "36"},
		{oid: "1.2.840.10036.1.7.1.2.%d.3", typ: snmpTypeInteger, lower: "1", upper: "34"},
		{oid: "1.2.840.10036.1.7.1.2.%d.4", typ: snmpTypeInteger, lower: "3", upper: "52"},
		{oid: "1.2.840.10036.1.7.1.2.%d.5", typ: snmpTypeInteger, lower: "1", upper: "149"},
		{oid: "1.2.840.10036.1.7.1.3.%d.1", typ: snmpTypeInteger, lower: "11", upper: "8"},
		{oid: "1.2.840.10036.1.7.1.3.%d.2", typ: snmpTypeInteger, lower: "13", upper: "8"},
		{oid: "1.2.840.10036.1.7.1.3.%d.3", typ: snmpTypeInteger, lower: "14", upper: "4"},
		{oid: "1.2.840.10036.1.7.1.3.%d.4", typ: snmpTypeInteger, lower: "7", upper: "4"},
		{oid: "1.2.840.10036.1.7.1.3.%d.5", typ: snmpTypeInteger, lower: "11", upper: "4"},
		{oid: "1.2.840.10036.1.7.1.4.%d.1", typ: snmpTypeInteger, lower: "20", upper: "16"},
		{oid: "1.2.840.10036.1.7.1.4.%d.2", typ: snmpTypeInteger, lower: "17", upper: "16"},
		{oid: "1.2.840.10036.1.7.1.4.%d.3", typ: snmpTypeInteger, lower: "15", upper: "16"},
		{oid: "1.2.840.10036.1.7.1.4.%d.4", typ: snmpTypeInteger, lower: "17", upper: "16"},
		{oid: "1.2.840.10036.1.7.1.4.%d.5", typ: snmpTypeInteger, lower: "8", upper: "16"},
	}

	return apDot11BandRows(rows, radios)
}

func apDot11PhyMIBs(site string, radios []apRadioIndex) []converter.AddMib {
	rows := []dot11BandRow{
		// dot11PhyOperationTable: the PHY type is the authored radio's, so
		// only the regulatory domain and the temperature type are here.
		{
			oid: "1.2.840.10036.4.1.1.2.%d", typ: snmpTypeInteger,
			lower: apRegulatoryDomain(site), upper: apRegulatoryDomain(site),
		},
		{oid: "1.2.840.10036.4.1.1.3.%d", typ: snmpTypeInteger, lower: "1", upper: "1"},
		{oid: "1.2.840.10036.4.2.1.1.%d", typ: snmpTypeInteger, lower: "3", upper: "3"},
		{oid: "1.2.840.10036.4.2.1.2.%d", typ: snmpTypeInteger, lower: "1", upper: "1"},
		{oid: "1.2.840.10036.4.2.1.3.%d", typ: snmpTypeInteger, lower: "3", upper: "3"},
		// dot11PhyTxPowerTable: columns 1, 2 and 10 are the authored radio's
		// power, so the remaining level columns are what is left to say.
		{oid: "1.2.840.10036.4.3.1.3.%d", typ: snmpTypeInteger, lower: "5", upper: "10"},
		{oid: "1.2.840.10036.4.3.1.4.%d", typ: snmpTypeInteger, lower: "20", upper: "20"},
		{oid: "1.2.840.10036.4.3.1.5.%d", typ: snmpTypeInteger, lower: "30", upper: "40"},
		{oid: "1.2.840.10036.4.3.1.6.%d", typ: snmpTypeInteger, lower: "50", upper: "0"},
		{oid: "1.2.840.10036.4.3.1.7.%d", typ: snmpTypeInteger, lower: "100", upper: "0"},
		{oid: "1.2.840.10036.4.3.1.8.%d", typ: snmpTypeInteger, lower: "0", upper: "0"},
		{oid: "1.2.840.10036.4.3.1.9.%d", typ: snmpTypeInteger, lower: "0", upper: "0"},
	}
	mibs := apDot11BandRows(rows, radios)
	// dot11ResourceTypeIDName, a scalar: the capture answers "RTID".
	mibs = append(mibs, converter.AddMib{
		OID: "1.2.840.10036.3.1.1.0", Type: snmpTypeString, Value: "RTID",
	})

	return append(mibs, apDot11DSSSMIBs(radios)...)
}

// apDot11DSSSMIBs answers dot11PhyDSSSTable for the 2.4 GHz radio alone. The
// capture's 5 GHz radio answers no DSSS row at all, and its current channel
// column is the authored radio's, so what is left is the three sub-band
// columns beside it.
func apDot11DSSSMIBs(radios []apRadioIndex) []converter.AddMib {
	const subBandColumns = 3
	mibs := make([]converter.AddMib, 0, subBandColumns)
	for _, radio := range radios {
		if radio.plan.band != apBand24GHz {
			continue
		}
		suffix := fmt.Sprintf(".%d", radio.ifIndex)
		mibs = append(mibs,
			converter.AddMib{OID: "1.2.840.10036.4.5.1.2" + suffix, Type: snmpTypeInteger, Value: "1"},
			converter.AddMib{OID: "1.2.840.10036.4.5.1.3" + suffix, Type: snmpTypeInteger, Value: "1"},
			converter.AddMib{OID: "1.2.840.10036.4.5.1.4" + suffix, Type: snmpTypeInteger, Value: "0"},
		)
	}

	return mibs
}

// apDot11BandRows expands one row template per radio, choosing the capture's
// 2.4 GHz value for the 2.4 GHz radio and its 5 GHz value for the rest. The
// template's OID carries a single %d, which is the radio's ifIndex.
func apDot11BandRows(rows []dot11BandRow, radios []apRadioIndex) []converter.AddMib {
	mibs := make([]converter.AddMib, 0, len(rows)*len(radios))
	for _, radio := range radios {
		for _, row := range rows {
			value := row.upper
			if radio.plan.band == apBand24GHz {
				value = row.lower
			}
			mibs = append(mibs, converter.AddMib{
				OID: fmt.Sprintf(row.oid, radio.ifIndex), Type: row.typ, Value: value,
			})
		}
	}

	return mibs
}

func apCountryString(site string) string {
	switch strings.ToUpper(site) {
	case "EHV":
		return "4E 4C 20"
	case "LON":
		return "47 42 20"
	default:
		return "55 53 20"
	}
}

func apRegulatoryDomain(site string) string {
	if strings.EqualFold(site, "EHV") || strings.EqualFold(site, "LON") {
		return "48"
	}

	return "16"
}
