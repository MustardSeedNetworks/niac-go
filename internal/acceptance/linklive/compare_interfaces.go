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

// samplesUtilization reports whether Link-Live measures interface utilization
// for this device at all.
//
// It measures switch and router ports. A leaf node - an endpoint, a server, a
// wireless controller - is never sampled even when its agent serves
// ifHCInOctets, which every one of ours does: verified against the live
// simulation, MED-DNS01, MED-WLC01 and MED-PUMP-B01-F01-02 all return Counter64
// octet counters and Link-Live still reports no utilization for any of them.
// Expecting a sample there fails every run for something the simulation gets
// right, and drowns the findings that matter.
func samplesUtilization(device AuthoredDevice) bool {
	return !isLeafType(device.Type)
}

func missingUtilizationFindings(device AuthoredDevice) []Finding {
	if !samplesUtilization(device) {
		return nil
	}

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

// utilizationWarningPercent is where Link-Live raises an interface Warning.
// Measured against a live discovery: interfaces up to 78.7% stayed clean, 81.8%
// and above were flagged.
const utilizationWarningPercent = 80

func problemMatchesAuthoredState(expected AuthoredInterface, problem string) bool {
	if strings.EqualFold(expected.Status, "down") &&
		strings.Contains(strings.ToLower(problem), "down") {
		return true
	}

	// A pack that authors an interface above the warning line is asking for the
	// warning - that is the story a guided demo walks an engineer through, and
	// reporting the amber icon as a mismatch would fail the run for the one
	// thing the pack got right. A warning anywhere else is still a finding.
	return expected.UtilizationPercent >= utilizationWarningPercent &&
		strings.Contains(strings.ToLower(problem), "warning")
}

// expectsProblem reports whether any authored interface on the device is meant
// to be flagged, which is what the device's own worst-problem rolls up from.
func expectsProblem(device AuthoredDevice, problem string) bool {
	for _, iface := range device.Interfaces {
		if problemMatchesAuthoredState(iface, problem) {
			return true
		}
	}

	return false
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
