package linklive

import (
	"strconv"
	"strings"
)

func compareInterfaces(expected AuthoredDevice, actual []ObservedInterface) []Finding {
	if expected.Interfaces == nil {
		return nil
	}
	observed := indexObservedInterfaces(actual)
	hasUtilization := deviceHasUtilizationSample(actual)
	wanted, findings := compareExpectedInterfaces(
		expected,
		observed,
		hasUtilization,
	)
	if !hasUtilization {
		findings = append(findings, missingUtilizationFindings(expected)...)
	}
	if !expected.InterfaceInventoryComplete {
		return findings
	}
	return append(findings, compareUnexpectedInterfaces(expected.Name, wanted, actual)...)
}

func missingUtilizationFindings(device AuthoredDevice) []Finding {
	var findings []Finding
	for _, iface := range device.Interfaces {
		if iface.UtilizationPercent <= 0 && !iface.UtilizationDynamic {
			continue
		}
		findings = append(findings, interfaceFinding(
			FindingInterfaceUtilizationConflict,
			device.Name,
			iface.Name,
			"utilization sample",
			"none",
		))
	}
	return findings
}

func indexObservedInterfaces(interfaces []ObservedInterface) map[string]ObservedInterface {
	result := make(map[string]ObservedInterface, len(interfaces))
	for _, iface := range interfaces {
		result[normalizeInterfaceName(iface.Interface.Name)] = iface
	}
	return result
}

func compareExpectedInterfaces(
	expected AuthoredDevice,
	observed map[string]ObservedInterface,
	compareUtilization bool,
) (map[string]bool, []Finding) {
	wanted := make(map[string]bool, len(expected.Interfaces))
	var findings []Finding
	for _, iface := range expected.Interfaces {
		key := normalizeInterfaceName(iface.Name)
		wanted[key] = true
		got, found := observed[key]
		if !found {
			findings = append(findings, interfaceFinding(
				FindingMissingInterface, expected.Name, iface.Name, iface.Name, "",
			))
			continue
		}
		findings = append(
			findings,
			compareInterface(expected.Name, iface, got, compareUtilization)...,
		)
	}
	return wanted, findings
}

func compareUnexpectedInterfaces(
	device string,
	wanted map[string]bool,
	actual []ObservedInterface,
) []Finding {
	var findings []Finding
	for _, iface := range actual {
		if !wanted[normalizeInterfaceName(iface.Interface.Name)] {
			findings = append(findings, interfaceFinding(
				FindingUnexpectedInterface, device, iface.Interface.Name, "", iface.Interface.Name,
			))
		}
	}
	return findings
}

func compareInterface(
	device string,
	expected AuthoredInterface,
	actual ObservedInterface,
	compareUtilization bool,
) []Finding {
	findings := compareInterfaceState(device, expected, actual.Interface)
	findings = append(
		findings,
		compareInterfaceTelemetry(device, expected, actual.Interface, compareUtilization)...,
	)
	if strings.TrimSpace(actual.WorstProblem) != "" &&
		!problemMatchesAuthoredState(expected, actual.WorstProblem) {
		findings = append(findings, interfaceFinding(
			FindingInterfaceProblemConflict, device, expected.Name, "", actual.WorstProblem,
		))
	}
	return findings
}

func problemMatchesAuthoredState(expected AuthoredInterface, problem string) bool {
	return strings.EqualFold(expected.Status, "down") &&
		strings.Contains(strings.ToLower(problem), "down")
}

func deviceHasUtilizationSample(interfaces []ObservedInterface) bool {
	for _, iface := range interfaces {
		if iface.Interface.Utilization.Percent > 0 {
			return true
		}
	}
	return false
}

func compareInterfaceState(
	device string,
	expected AuthoredInterface,
	actual ObservedInterfaceDetails,
) []Finding {
	var findings []Finding
	if expected.Status != "" && !strings.EqualFold(expected.Status, actual.Status) {
		findings = append(findings, interfaceFinding(
			FindingInterfaceStatusConflict, device, expected.Name, expected.Status, actual.Status,
		))
	}
	findings = append(findings, compareInterfaceSpeed(device, expected, actual)...)
	findings = append(findings, compareInterfaceMTU(device, expected, actual)...)
	return findings
}

func compareInterfaceSpeed(
	device string,
	expected AuthoredInterface,
	actual ObservedInterfaceDetails,
) []Finding {
	var findings []Finding
	if expected.SpeedMbps > 0 && parseSpeedMbps(actual.Speed) != expected.SpeedMbps {
		findings = append(findings, interfaceFinding(
			FindingInterfaceSpeedConflict, device, expected.Name,
			strconv.Itoa(expected.SpeedMbps), actual.Speed,
		))
	}
	if interfaceHasDuplex(expected) && !strings.EqualFold(expected.Duplex, actual.Duplex) {
		findings = append(findings, interfaceFinding(
			FindingInterfaceDuplexConflict, device, expected.Name, expected.Duplex, actual.Duplex,
		))
	}
	return findings
}

func compareInterfaceMTU(
	device string,
	expected AuthoredInterface,
	actual ObservedInterfaceDetails,
) []Finding {
	if expected.MTU <= 0 || expected.MTU == actual.MTU {
		return nil
	}
	return []Finding{interfaceFinding(
		FindingInterfaceMTUConflict, device, expected.Name,
		strconv.Itoa(expected.MTU), strconv.Itoa(actual.MTU),
	)}
}

func compareInterfaceTelemetry(
	device string,
	expected AuthoredInterface,
	actual ObservedInterfaceDetails,
	compareUtilization bool,
) []Finding {
	var findings []Finding
	utilization := actual.Utilization.Percent
	if compareUtilization {
		findings = append(
			findings,
			compareInterfaceUtilization(device, expected, utilization)...,
		)
	}
	if expected.ExpectZeroErrors {
		findings = append(findings, compareZeroPacketRate(
			FindingInterfaceErrorConflict, device, expected.Name, actual.Errors.Percent,
		)...)
	}
	if expected.ExpectZeroDiscards {
		findings = append(findings, compareZeroPacketRate(
			FindingInterfaceDiscardConflict, device, expected.Name, actual.Discards.Percent,
		)...)
	}
	return findings
}

func compareInterfaceUtilization(
	device string,
	expected AuthoredInterface,
	actual float64,
) []Finding {
	if expected.UtilizationDynamic || expected.UtilizationPercent <= 0 ||
		utilizationMatches(expected.UtilizationPercent, actual) {
		return nil
	}
	return []Finding{interfaceFinding(
		FindingInterfaceUtilizationConflict, device, expected.Name,
		formatPercent(expected.UtilizationPercent), formatPercent(actual),
	)}
}

func compareZeroPacketRate(kind FindingKind, device, iface string, rate float64) []Finding {
	if rate == 0 {
		return nil
	}
	return []Finding{interfaceFinding(kind, device, iface, "0", formatPercent(rate))}
}
