package scenario

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

const (
	snmpTypeInteger                                 = "INTEGER"
	snmpTypeGauge32                                 = "Gauge32"
	snmpTypeHexString                               = "Hex-STRING"
	snmpTypeCounter32                               = "Counter32"
	snmpTypeString                                  = "STRING"
	apRadioCount                                    = 4
	dot11PhyColumns                                 = 3
	dot11PHYTypeOFDM                                = 4
	dot11StationMediumOccupancyLimitColumn          = 2
	dot11StationCFPollableColumn                    = 3
	dot11StationCFPPeriodColumn                     = 4
	dot11StationCFPMaxDurationColumn                = 5
	dot11StationAuthenticationResponseTimeoutColumn = 6
	dot11StationPrivacyOptionImplementedColumn      = 7
	dot11StationPowerManagementModeColumn           = 8
	dot11StationDesiredSSIDColumn                   = 9
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

// apDot11MIBs returns the sanitized IEEE 802.11 identity and capability tables used for portable AP discovery.
func apDot11MIBs(site string) []converter.AddMib {
	mibs := apDot11StationMIBs(site)

	return append(mibs, apDot11PhyMIBs(site)...)
}

func apDot11StationMIBs(site string) []converter.AddMib {
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
		{
			column: dot11StationDesiredSSIDColumn,
			typ:    snmpTypeString,
			value:  strings.ToUpper(site) + "-CORP",
		},
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
	mibs := make([]converter.AddMib, 0, len(fields)*apRadioCount)
	for radio := 1; radio <= apRadioCount; radio++ {
		for _, field := range fields {
			mibs = append(mibs, converter.AddMib{
				OID:  fmt.Sprintf("1.2.840.10036.1.1.1.%d.%d", field.column, radio),
				Type: field.typ, Value: field.value,
			})
		}
	}
	return append(mibs, apDot11CapabilityMIBs()...)
}

func apDot11CapabilityMIBs() []converter.AddMib {
	return []converter.AddMib{
		{OID: "1.2.840.10036.1.2.1.2.1.1", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.1.2.1.2.1.2", Type: snmpTypeInteger, Value: "2"},
		{OID: "1.2.840.10036.1.2.1.2.1.3", Type: snmpTypeInteger, Value: "129"},
		{OID: "1.2.840.10036.1.2.1.2.2.1", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.1.2.1.2.2.2", Type: snmpTypeInteger, Value: "2"},
		{OID: "1.2.840.10036.1.2.1.2.2.3", Type: snmpTypeInteger, Value: "129"},
		{OID: "1.2.840.10036.1.2.1.3.1.1", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.1.2.1.3.1.2", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.1.2.1.3.1.3", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.1.2.1.3.2.1", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.1.2.1.3.2.2", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.1.2.1.3.2.3", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.1.5.1.1.1", Type: snmpTypeInteger, Value: "2"},
		{OID: "1.2.840.10036.1.5.1.1.2", Type: snmpTypeInteger, Value: "2"},
		{OID: "1.2.840.10036.1.5.1.4.1", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.1.5.1.4.2", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.1.5.1.5.1", Type: snmpTypeCounter32, Value: "5"},
		{OID: "1.2.840.10036.1.5.1.5.2", Type: snmpTypeCounter32, Value: "0"},
		{OID: "1.2.840.10036.1.5.1.6.1", Type: snmpTypeCounter32, Value: "0"},
		{OID: "1.2.840.10036.1.5.1.6.2", Type: snmpTypeCounter32, Value: "0"},
		{OID: "1.2.840.10036.1.7.1.2.1.1", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.1.7.1.2.1.2", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.1.7.1.2.1.3", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.1.7.1.2.1.4", Type: snmpTypeInteger, Value: "3"},
		{OID: "1.2.840.10036.1.7.1.2.1.5", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.1.7.1.2.2.1", Type: snmpTypeInteger, Value: "36"},
		{OID: "1.2.840.10036.1.7.1.2.2.2", Type: snmpTypeInteger, Value: "36"},
		{OID: "1.2.840.10036.1.7.1.2.2.3", Type: snmpTypeInteger, Value: "34"},
		{OID: "1.2.840.10036.1.7.1.2.2.4", Type: snmpTypeInteger, Value: "52"},
		{OID: "1.2.840.10036.1.7.1.2.2.5", Type: snmpTypeInteger, Value: "149"},
		{OID: "1.2.840.10036.1.7.1.3.1.1", Type: snmpTypeInteger, Value: "11"},
		{OID: "1.2.840.10036.1.7.1.3.1.2", Type: snmpTypeInteger, Value: "13"},
		{OID: "1.2.840.10036.1.7.1.3.1.3", Type: snmpTypeInteger, Value: "14"},
		{OID: "1.2.840.10036.1.7.1.3.1.4", Type: snmpTypeInteger, Value: "7"},
		{OID: "1.2.840.10036.1.7.1.3.1.5", Type: snmpTypeInteger, Value: "11"},
		{OID: "1.2.840.10036.1.7.1.3.2.1", Type: snmpTypeInteger, Value: "8"},
		{OID: "1.2.840.10036.1.7.1.3.2.2", Type: snmpTypeInteger, Value: "8"},
		{OID: "1.2.840.10036.1.7.1.3.2.3", Type: snmpTypeInteger, Value: "4"},
		{OID: "1.2.840.10036.1.7.1.3.2.4", Type: snmpTypeInteger, Value: "4"},
		{OID: "1.2.840.10036.1.7.1.3.2.5", Type: snmpTypeInteger, Value: "4"},
		{OID: "1.2.840.10036.1.7.1.4.1.1", Type: snmpTypeInteger, Value: "20"},
		{OID: "1.2.840.10036.1.7.1.4.1.2", Type: snmpTypeInteger, Value: "17"},
		{OID: "1.2.840.10036.1.7.1.4.1.3", Type: snmpTypeInteger, Value: "15"},
		{OID: "1.2.840.10036.1.7.1.4.1.4", Type: snmpTypeInteger, Value: "17"},
		{OID: "1.2.840.10036.1.7.1.4.1.5", Type: snmpTypeInteger, Value: "8"},
		{OID: "1.2.840.10036.1.7.1.4.2.1", Type: snmpTypeInteger, Value: "16"},
		{OID: "1.2.840.10036.1.7.1.4.2.2", Type: snmpTypeInteger, Value: "16"},
		{OID: "1.2.840.10036.1.7.1.4.2.3", Type: snmpTypeInteger, Value: "16"},
		{OID: "1.2.840.10036.1.7.1.4.2.4", Type: snmpTypeInteger, Value: "16"},
		{OID: "1.2.840.10036.1.7.1.4.2.5", Type: snmpTypeInteger, Value: "16"},
	}
}

func apDot11PhyMIBs(site string) []converter.AddMib {
	mibs := make([]converter.AddMib, 0, apRadioCount*dot11PhyColumns)
	for radio := 1; radio <= apRadioCount; radio++ {
		suffix := fmt.Sprintf(".%d", radio)
		mibs = append(
			mibs,
			converter.AddMib{
				OID:   "1.2.840.10036.2.1.1.1" + suffix,
				Type:  snmpTypeInteger,
				Value: strconv.Itoa(dot11PHYTypeOFDM),
			},
			converter.AddMib{
				OID:   "1.2.840.10036.2.1.1.2" + suffix,
				Type:  snmpTypeInteger,
				Value: apRegulatoryDomain(site),
			},
			converter.AddMib{
				OID:   "1.2.840.10036.2.1.1.3" + suffix,
				Type:  snmpTypeInteger,
				Value: "1",
			},
		)
	}
	return append(mibs, []converter.AddMib{
		{OID: "1.2.840.10036.3.1.1.0", Type: snmpTypeString, Value: "RTID"},
		{OID: "1.2.840.10036.4.1.1.1.1", Type: snmpTypeInteger, Value: "2"},
		{OID: "1.2.840.10036.4.1.1.1.2", Type: snmpTypeInteger, Value: "4"},
		{OID: "1.2.840.10036.4.1.1.2.1", Type: snmpTypeInteger, Value: "16"},
		{OID: "1.2.840.10036.4.1.1.2.2", Type: snmpTypeInteger, Value: "16"},
		{OID: "1.2.840.10036.4.1.1.3.1", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.4.1.1.3.2", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.4.2.1.1.1", Type: snmpTypeInteger, Value: "3"},
		{OID: "1.2.840.10036.4.2.1.1.2", Type: snmpTypeInteger, Value: "3"},
		{OID: "1.2.840.10036.4.2.1.2.1", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.4.2.1.2.2", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.4.2.1.3.1", Type: snmpTypeInteger, Value: "3"},
		{OID: "1.2.840.10036.4.2.1.3.2", Type: snmpTypeInteger, Value: "3"},
		{OID: "1.2.840.10036.4.3.1.1.1", Type: snmpTypeInteger, Value: "6"},
		{OID: "1.2.840.10036.4.3.1.1.2", Type: snmpTypeInteger, Value: "4"},
		{OID: "1.2.840.10036.4.3.1.2.1", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.4.3.1.2.2", Type: snmpTypeInteger, Value: "5"},
		{OID: "1.2.840.10036.4.3.1.3.1", Type: snmpTypeInteger, Value: "5"},
		{OID: "1.2.840.10036.4.3.1.3.2", Type: snmpTypeInteger, Value: "10"},
		{OID: "1.2.840.10036.4.3.1.4.1", Type: snmpTypeInteger, Value: "20"},
		{OID: "1.2.840.10036.4.3.1.4.2", Type: snmpTypeInteger, Value: "20"},
		{OID: "1.2.840.10036.4.3.1.5.1", Type: snmpTypeInteger, Value: "30"},
		{OID: "1.2.840.10036.4.3.1.5.2", Type: snmpTypeInteger, Value: "40"},
		{OID: "1.2.840.10036.4.3.1.6.1", Type: snmpTypeInteger, Value: "50"},
		{OID: "1.2.840.10036.4.3.1.6.2", Type: snmpTypeInteger, Value: "0"},
		{OID: "1.2.840.10036.4.3.1.7.1", Type: snmpTypeInteger, Value: "100"},
		{OID: "1.2.840.10036.4.3.1.7.2", Type: snmpTypeInteger, Value: "0"},
		{OID: "1.2.840.10036.4.3.1.8.1", Type: snmpTypeInteger, Value: "0"},
		{OID: "1.2.840.10036.4.3.1.8.2", Type: snmpTypeInteger, Value: "0"},
		{OID: "1.2.840.10036.4.3.1.9.1", Type: snmpTypeInteger, Value: "0"},
		{OID: "1.2.840.10036.4.3.1.9.2", Type: snmpTypeInteger, Value: "0"},
		{OID: "1.2.840.10036.4.3.1.10.1", Type: snmpTypeInteger, Value: "5"},
		{OID: "1.2.840.10036.4.3.1.10.2", Type: snmpTypeInteger, Value: "4"},
		{OID: "1.2.840.10036.4.5.1.2.1", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.4.5.1.3.1", Type: snmpTypeInteger, Value: "1"},
		{OID: "1.2.840.10036.4.5.1.4.1", Type: snmpTypeInteger, Value: "0"},
	}...)
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
