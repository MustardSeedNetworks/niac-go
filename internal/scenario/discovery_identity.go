package scenario

import (
	"fmt"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// apDiscoveryMIBs gives an access point the entity and per-radio dot11 identity
// a discovery tool needs to tell its radios apart. The station ID embeds both
// the radio index and the device's own MAC suffix, so no two radios in a fleet
// share one — asserted fleet-wide by assertCompleteAPRadioMIBs.
func apDiscoveryMIBs(name, site string, macSuffix uint32) []converter.AddMib {
	mibs := append(apDot11MIBs(site), []converter.AddMib{
		{OID: "1.2.840.10036.4.5.1.1.1", Type: "INTEGER", Value: "1"},
		{OID: "1.3.6.1.2.1.47.1.1.1.1.2.1", Type: "STRING", Value: "Cisco CW9178I Wi-Fi 7 Access Point"},
		{OID: "1.3.6.1.2.1.47.1.1.1.1.5.1", Type: "INTEGER", Value: "3"},
		{OID: "1.3.6.1.2.1.47.1.1.1.1.7.1", Type: "STRING", Value: name},
		{OID: "1.3.6.1.2.1.47.1.1.1.1.12.1", Type: "STRING", Value: "Cisco Systems, Inc."},
		{OID: "1.3.6.1.2.1.47.1.1.1.1.13.1", Type: "STRING", Value: "CW9178I"},
	}...)
	for radio := 1; radio <= apRadioCount; radio++ {
		stationID := fmt.Sprintf("02 00 %02X %02X %02X %02X",
			radio,
			safeconv.ByteFromUint32(macSuffix>>macHighByteShift),
			safeconv.ByteFromUint32(macSuffix>>macMiddleByteShift),
			safeconv.ByteFromUint32(macSuffix),
		)
		suffix := fmt.Sprintf(".%d", radio)
		mibs = append(mibs,
			converter.AddMib{OID: "1.2.840.10036.1.1.1.1" + suffix, Type: "Hex-STRING", Value: stationID},
			converter.AddMib{OID: "1.2.840.10036.2.1.1.8" + suffix, Type: "STRING", Value: "Cisco Systems"},
			converter.AddMib{OID: "1.2.840.10036.2.1.1.9" + suffix, Type: "STRING", Value: "CW9178I Wi-Fi 7 radio"},
			converter.AddMib{OID: "1.2.840.10036.3.1.2.1.1" + suffix, Type: "Hex-STRING", Value: "00 40 96"},
			converter.AddMib{OID: "1.2.840.10036.3.1.2.1.2" + suffix, Type: "STRING", Value: "Cisco Systems"},
			converter.AddMib{OID: "1.2.840.10036.3.1.2.1.3" + suffix, Type: "STRING", Value: "CW9178I Wi-Fi 7 radio"},
			converter.AddMib{OID: "1.2.840.10036.3.1.2.1.4" + suffix, Type: "STRING", Value: "IOS XE 17.15.3"},
		)
	}
	return mibs
}
