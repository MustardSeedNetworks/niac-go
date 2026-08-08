package linklive

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

func contains(values []string, want string) bool {
	return slices.Contains(values, want)
}

func namesMatch(left, right string) bool {
	authored := normalizeHostname(right)
	observed := normalizeHostname(left)
	if strings.Contains(authored, ".") {
		return observed == authored
	}
	return hostLabel(observed) == authored
}

func hostLabel(value string) string {
	label, _, _ := strings.Cut(normalizeHostname(value), ".")
	return label
}

func normalizeHostname(value string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
}

func displayedType(deviceType string) string {
	switch deviceType {
	case "switch", "layer3-switch":
		return "Switch"
	case "router", "firewall":
		return "Router"
	case "ap", "access-point":
		return "AP"
	case "host", "workstation", "iot":
		return "Host/Client"
	case "printer":
		return "Printer"
	default:
		return ""
	}
}

// displayedTypeMatches reports whether Link-Live filed the device the way the
// authored role says it should.
//
// An endpoint that answers SNMP is filed as an SNMP Agent rather than as a
// Host/Client, and that is the correct reading of it - a clinical appliance or
// a managed printer is SNMP-managed gear, not somebody's desktop. Infrastructure
// keeps its role label, so a switch Link-Live failed to recognise as a switch is
// still a finding.
func displayedTypeMatches(expected AuthoredDevice, actual string) bool {
	if expected.Type == "layer3-switch" {
		return actual == "Switch" || actual == "Router"
	}
	if actual == snmpAgentType && expected.ServesSNMP && isEndpointType(expected.Type) {
		return true
	}

	return displayedType(expected.Type) == actual
}

// snmpAgentType is how Link-Live labels a device it identified only by its
// SNMP agent.
const snmpAgentType = "SNMP Agent"

func isEndpointType(deviceType string) bool {
	switch deviceType {
	case "host", "workstation", "iot", "printer":
		return true
	default:
		return false
	}
}

// isLeafType reports whether the device hangs off the network rather than
// forming it. Servers, wireless controllers and access points are not endpoints
// - they are managed infrastructure - but they still sit on one port, which is
// what decides whether Link-Live measures them. Access points are the same
// reading: on the hospital capture Link-Live returned util 0 for all five
// interfaces of every one of the 30 APs, wired uplink included, while the
// switch port facing that same uplink reported 71.57%.
func isLeafType(deviceType string) bool {
	return isEndpointType(deviceType) || deviceType == "server" || deviceType == "access-point"
}

func parseSpeedMbps(value string) int {
	var number float64
	var unit string
	if _, err := fmt.Sscan(value, &number, &unit); err != nil {
		return 0
	}
	switch strings.ToLower(unit) {
	case "gb", "gbps":
		number *= 1000
	case "mb", "mbps":
	default:
		return 0
	}
	return int(number)
}

func parseVLAN(value string) int {
	vlan, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return vlan
}

func portsMatch(actual, left, right string) bool {
	actual = normalizePortDisplay(actual)
	left, right = normalizePortDisplay(left), normalizePortDisplay(right)
	if actual == "" {
		return false
	}
	return left != "" && strings.EqualFold(actual, left) ||
		right != "" && strings.EqualFold(actual, right) ||
		left != "" && right != "" &&
			(strings.EqualFold(actual, left+" / "+right) ||
				strings.EqualFold(actual, right+" / "+left))
}

func normalizePortDisplay(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
