package linklive

import (
	"math"
	"strconv"
	"strings"
)

const utilizationTolerance = 2

func formatPercent(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func normalizeInterfaceName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func interfaceHasDuplex(iface AuthoredInterface) bool {
	return strings.EqualFold(iface.Type, "ethernet") && strings.TrimSpace(iface.Duplex) != ""
}

func utilizationMatches(expected, actual float64) bool {
	return actual > 0 && actual <= 100 && math.Abs(expected-actual) <= utilizationTolerance
}
