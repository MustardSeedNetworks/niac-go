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

func displayedTypeMatches(deviceType, actual string) bool {
	if deviceType == "layer3-switch" {
		return actual == "Switch" || actual == "Router"
	}
	return displayedType(deviceType) == actual
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
