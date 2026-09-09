package config

import "fmt"

// PoE budget bounds. The floor is one 802.3af class-1 port and the ceiling is
// the largest single-chassis PSE shipping today (a 48-port 90 W 802.3bt line
// card), so a mistyped budget is caught rather than replayed.
const (
	minPoEBudgetWatts = 4
	maxPoEBudgetWatts = 4320

	defaultPoEUsageThresholdPercent = 80

	// PoETenthWattsPerWatt converts the LLDP-MED power unit, which is tenths of
	// a watt, into the watts POWER-ETHERNET-MIB reports.
	PoETenthWattsPerWatt = 10
)

// UsageThreshold returns the authored consumption alarm threshold as a
// percentage of the budget, or the default when the author left it out.
func (p *PoEConfig) UsageThreshold() int {
	if p.UsageThresholdPercent > 0 {
		return p.UsageThresholdPercent
	}

	return defaultPoEUsageThresholdPercent
}

// validatePoE checks one device's own PSE block. Over-subscription is not here:
// it needs the roster to know which powered devices sit behind the ports.
func (v *Validator) validatePoE(device *Device, prefix string) {
	poe := device.PoEConfig
	if poe == nil {
		return
	}

	if poe.BudgetWatts < minPoEBudgetWatts || poe.BudgetWatts > maxPoEBudgetWatts {
		v.addError(prefix+".poe.budget_watts", fmt.Sprintf(
			"PoE budget must be between %d and %d watts, got %d",
			minPoEBudgetWatts, maxPoEBudgetWatts, poe.BudgetWatts))
	}

	threshold := poe.UsageThresholdPercent
	if threshold != 0 && (threshold < 1 || threshold > 99) {
		v.addError(prefix+".poe.usage_threshold_percent", fmt.Sprintf(
			"PoE usage threshold must be between 1 and 99 percent, got %d", threshold))
	}
}

// validatePoEBudgets reports every PSE whose ports draw more than it can
// supply. A real switch answers such a port with a power-denied counter instead
// of powering it, so an over-subscribed pack would replay a device the author
// meant to be up as one that never comes up -- an authoring mistake worth
// naming before the simulation starts rather than after a tester reports it.
func (v *Validator) validatePoEBudgets(cfg *Config) {
	draws := poeDrawsByDevice(cfg)

	for _, segment := range cfg.NormalizedSegments() {
		for index := range segment.Devices {
			device := &segment.Devices[index]
			if device.PoEConfig == nil {
				continue
			}
			consumed := 0
			for _, trunk := range device.TrunkPorts {
				consumed += draws[trunk.RemoteDevice]
			}
			if consumed <= device.PoEConfig.BudgetWatts*PoETenthWattsPerWatt {
				continue
			}
			v.addError(device.Name+".poe.budget_watts", fmt.Sprintf(
				"attached powered devices draw %.1f W, over the %d W budget",
				float64(consumed)/PoETenthWattsPerWatt, device.PoEConfig.BudgetWatts))
		}
	}
}

// poeDrawsByDevice maps a device name to the power it advertises drawing, in
// tenths of a watt -- the LLDP-MED unit, kept until the comparison so a pack of
// class-1 endpoints does not round to nothing.
func poeDrawsByDevice(cfg *Config) map[string]int {
	draws := make(map[string]int)
	for _, segment := range cfg.NormalizedSegments() {
		for index := range segment.Devices {
			device := &segment.Devices[index]
			if draw := PoEDrawTenthWatts(device); draw > 0 {
				draws[device.Name] = draw
			}
		}
	}

	return draws
}

// PoEDrawTenthWatts returns what a device advertises drawing from a PSE, or
// zero when it is not a powered device. The advertisement is the only authored
// source: a phone's draw is written once, in the TLV a discovery tool decodes.
func PoEDrawTenthWatts(device *Device) int {
	if device == nil || device.LLDPConfig == nil || device.LLDPConfig.MED == nil {
		return 0
	}
	power := device.LLDPConfig.MED.Power
	if power == nil || power.DeviceType != "pd" {
		return 0
	}

	return power.ValueTenthWatts
}
