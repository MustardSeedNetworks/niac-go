package scenario

// The radios a simulated access point carries, declared once.
//
// The interface list, the authored `wifi` block and the IEEE802dot11-MIB rows
// all describe the same four radios, and they used to disagree: every radio
// reported dot11PHYType OFDM — 802.11a, 54 Mbps, 1999 — while its interface
// advertised up to 11.5 Gbps and the device's own sysDescr called it a Wi-Fi 7
// access point. A scanner reading the MIB and a scanner reading ifSpeed came
// away with different hardware.
//
// Band, speed and channel now come from one place, so a radio cannot describe
// itself two ways. The PHY type is no longer here at all: it follows from the
// band, and the one function that maps band to PHY lives beside the MIB it
// answers (`internal/protocols/snmp.dot11PhyForBand`).
type apRadio struct {
	// Band is the vocabulary the authored `wifi` block and the validator use.
	band string
	// Label is what an operator calls the band, for the interface description.
	label string
	// SpeedMbps is the interface's advertised rate.
	speedMbps int
	// Channel is the radio's operating channel in the band's own numbering.
	channel int
}

// Per-radio rates the CW9178I advertises, in Mbps. The 2.4 GHz radio is the
// slow one by physics; the 6 GHz radio carries the wide channels that make a
// Wi-Fi 7 access point worth buying.
const (
	apRadioSpeed24GHz = 1_400
	apRadioSpeed5GHz  = 5_800
	apRadioSpeed6GHz  = 11_500
)

// Channels the plan operates on. The two 5 GHz radios sit in different UNII
// sub-bands so one access point does not interfere with itself, and the 6 GHz
// radio takes the first 320 MHz-capable channel.
const (
	apChannel24GHz     = 6
	apChannel5GHzLower = 36
	apChannel5GHzUpper = 149
	apChannel6GHz      = 37
)

// Band names, in the vocabulary the schema publishes and the validator checks.
const (
	apBand24GHz = "2.4GHz"
	apBand5GHz  = "5GHz"
	apBand6GHz  = "6GHz"
)

// apRadioPlan is the CW9178I's radio layout: one 2.4 GHz, two 5 GHz and one
// 6 GHz, which is what a tri-band Wi-Fi 7 access point presents.
func apRadioPlan() []apRadio {
	return []apRadio{
		{band: apBand24GHz, label: "2.4 GHz", speedMbps: apRadioSpeed24GHz, channel: apChannel24GHz},
		{band: apBand5GHz, label: "5 GHz", speedMbps: apRadioSpeed5GHz, channel: apChannel5GHzLower},
		{band: apBand5GHz, label: "5 GHz", speedMbps: apRadioSpeed5GHz, channel: apChannel5GHzUpper},
		{band: apBand6GHz, label: "6 GHz", speedMbps: apRadioSpeed6GHz, channel: apChannel6GHz},
	}
}

// APRadioBandForTest reports the band a given radio of the plan operates in.
//
// Exported for the topology-truth test, which asserts the generated rows match
// the plan rather than restating the values — restating them is how the
// previous assertion came to pin OFDM on a Wi-Fi 7 radio.
func APRadioBandForTest(index int) string {
	plan := apRadioPlan()
	if index < 0 || index >= len(plan) {
		return ""
	}

	return plan[index].band
}
