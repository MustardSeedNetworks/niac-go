package scenario

import (
	"fmt"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// apDiscoveryMIBs gives an access point the entity and per-radio resource
// identity a discovery tool needs to tell its radios apart. The radios'
// own identity -- station ID, SSID, MAC address, channel and power -- is the
// authored `wifi` block's, served once from there.
func apDiscoveryMIBs(name, site string, radios []apRadioIndex) []converter.AddMib {
	mibs := append(apDot11MIBs(site, radios), []converter.AddMib{
		{OID: "1.3.6.1.2.1.47.1.1.1.1.2.1", Type: snmpTypeString, Value: "Cisco CW9178I Wi-Fi 7 Access Point"},
		{OID: "1.3.6.1.2.1.47.1.1.1.1.5.1", Type: snmpTypeInteger, Value: "3"},
		{OID: "1.3.6.1.2.1.47.1.1.1.1.7.1", Type: snmpTypeString, Value: name},
		{OID: "1.3.6.1.2.1.47.1.1.1.1.12.1", Type: snmpTypeString, Value: "Cisco Systems, Inc."},
		{OID: "1.3.6.1.2.1.47.1.1.1.1.13.1", Type: snmpTypeString, Value: "CW9178I"},
	}...)
	for _, radio := range radios {
		suffix := fmt.Sprintf(".%d", radio.ifIndex)
		mibs = append(
			mibs,
			converter.AddMib{OID: "1.2.840.10036.2.1.1.8" + suffix, Type: snmpTypeString, Value: "Cisco Systems"},
			converter.AddMib{
				OID:   "1.2.840.10036.2.1.1.9" + suffix,
				Type:  snmpTypeString,
				Value: "CW9178I Wi-Fi 7 radio",
			},
			converter.AddMib{OID: "1.2.840.10036.3.1.2.1.1" + suffix, Type: snmpTypeHexString, Value: "00 40 96"},
			converter.AddMib{OID: "1.2.840.10036.3.1.2.1.2" + suffix, Type: snmpTypeString, Value: "Cisco Systems"},
			converter.AddMib{
				OID:   "1.2.840.10036.3.1.2.1.3" + suffix,
				Type:  snmpTypeString,
				Value: "CW9178I Wi-Fi 7 radio",
			},
			converter.AddMib{OID: "1.2.840.10036.3.1.2.1.4" + suffix, Type: snmpTypeString, Value: "IOS XE 17.15.3"},
		)
	}

	return mibs
}
