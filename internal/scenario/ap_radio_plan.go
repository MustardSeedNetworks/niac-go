package scenario

// The radios a simulated access point carries, declared once.
//
// The interface list and the IEEE802dot11-MIB rows both describe the same four
// radios, and they disagreed: every radio reported dot11PHYType OFDM — 802.11a,
// 54 Mbps, 1999 — while its interface advertised up to 11.5 Gbps and the
// device's own sysDescr called it a Wi-Fi 7 access point. A scanner reading the
// MIB and a scanner reading ifSpeed came away with different hardware.
//
// Band, speed and PHY now come from one place, so a radio cannot describe
// itself two ways.
type apRadio struct {
	// Band is what an operator calls it, for the interface description.
	band string
	// SpeedMbps is the interface's advertised rate.
	speedMbps int
	// PHYType is the IEEE802dot11-MIB dot11PHYType value for this radio.
	phyType int
}

// IEEE802dot11-MIB dot11PHYType values.
//
// The enumeration stops at vht: it has no value for HE (Wi-Fi 6) or EHT
// (Wi-Fi 7), which is why real vendors publish their own MIBs for those radios.
// A Wi-Fi 6E or Wi-Fi 7 radio answering this MIB reports the highest value the
// MIB can express, and so does this one — the alternative is claiming 802.11a.
const (
	dot11PHYTypeHT  = 7 // 802.11n
	dot11PHYTypeVHT = 8 // 802.11ac, and the ceiling of this MIB
)

// Per-radio rates the CW9178I advertises, in Mbps. The 2.4 GHz radio is the
// slow one by physics; the 6 GHz radio carries the wide channels that make a
// Wi-Fi 7 access point worth buying.
const (
	apRadioSpeed24GHz = 1_400
	apRadioSpeed5GHz  = 5_800
	apRadioSpeed6GHz  = 11_500
)

// apRadioPlan is the CW9178I's radio layout: one 2.4 GHz, two 5 GHz and one
// 6 GHz, which is what a tri-band Wi-Fi 7 access point presents.
func apRadioPlan() []apRadio {
	return []apRadio{
		{band: "2.4 GHz", speedMbps: apRadioSpeed24GHz, phyType: dot11PHYTypeHT},
		{band: "5 GHz", speedMbps: apRadioSpeed5GHz, phyType: dot11PHYTypeVHT},
		{band: "5 GHz", speedMbps: apRadioSpeed5GHz, phyType: dot11PHYTypeVHT},
		{band: "6 GHz", speedMbps: apRadioSpeed6GHz, phyType: dot11PHYTypeVHT},
	}
}

// APRadioPHYTypeForTest reports the dot11PHYType a given radio announces.
//
// Exported for the topology-truth test, which asserts the MIB rows match the
// plan rather than restating the values — restating them is how the previous
// assertion came to pin OFDM on a Wi-Fi 7 radio.
func APRadioPHYTypeForTest(index int) int {
	plan := apRadioPlan()
	if index < 0 || index >= len(plan) {
		return 0
	}
	return plan[index].phyType
}
