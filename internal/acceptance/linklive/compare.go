package linklive

import (
	"strconv"
	"strings"
)

const namePrefixParts = 2

// Compare reports every deterministic mismatch in NIAC-authored topology.
func Compare(authored AuthoredSnapshot, observed ObservedSnapshot) []Finding {
	byMAC := observedByMAC(observed.Hosts)
	var findings []Finding
	for _, device := range authored.Devices {
		findings = append(findings, compareDevice(device, byMAC[device.MAC])...)
	}
	for _, link := range authored.Links {
		findings = append(findings, compareLink(link, byMAC)...)
	}
	findings = append(findings, compareUnexpectedDevices(authored.Devices, observed.Hosts)...)
	findings = append(findings, compareUnexpectedLinks(authored, observed.Hosts)...)
	return findings
}

func observedByMAC(hosts []ObservedHost) map[string]ObservedHost {
	byMAC := make(map[string]ObservedHost, len(hosts))
	for _, host := range hosts {
		byMAC[normalizeMAC(host.MAC)] = host
	}
	return byMAC
}

func compareDevice(expected AuthoredDevice, actual ObservedHost) []Finding {
	if actual.MAC == "" {
		return []Finding{
			{Kind: FindingMissingDevice, Device: expected.Name, Expected: expected.MAC},
		}
	}
	var findings []Finding
	if !namesMatch(actual.Name, expected.Name) {
		findings = append(
			findings,
			conflict(FindingNameConflict, expected.Name, expected.Name, actual.Name),
		)
	}
	wantType := displayedType(expected.Type)
	if wantType != "" && !displayedTypeMatches(expected, actual.Type) {
		findings = append(
			findings,
			conflict(FindingTypeConflict, expected.Name, wantType, actual.Type),
		)
	}
	if len(expected.IPv4) > 0 && !contains(expected.IPv4, actual.IPv4) {
		findings = append(
			findings,
			conflict(
				FindingAddressConflict,
				expected.Name,
				strings.Join(expected.IPv4, ","),
				actual.IPv4,
			),
		)
	}
	if strings.TrimSpace(actual.WorstProblem) != "" {
		findings = append(
			findings,
			conflict(FindingProblemConflict, expected.Name, "", actual.WorstProblem),
		)
	}
	findings = append(findings, compareInterfaces(expected, actual.Interfaces)...)
	return findings
}

func compareLink(expected AuthoredLink, hosts map[string]ObservedHost) []Finding {
	connection, found := findConnection(hosts[expected.SourceMAC], expected.TargetMAC)
	if !found {
		connection, found = findConnection(hosts[expected.TargetMAC], expected.SourceMAC)
	}
	if !found {
		return []Finding{{Kind: FindingMissingLink, Device: expected.Source, Peer: expected.Target}}
	}
	var findings []Finding
	if !portsMatch(connection.Edge.Port, expected.SourcePort, expected.TargetPort) {
		findings = append(
			findings,
			linkConflict(
				FindingPortConflict,
				expected,
				expected.SourcePort+" / "+expected.TargetPort,
				connection.Edge.Port,
			),
		)
	}
	if expected.Duplex != "" && !strings.EqualFold(expected.Duplex, connection.Edge.Duplex) {
		findings = append(
			findings,
			linkConflict(FindingDuplexConflict, expected, expected.Duplex, connection.Edge.Duplex),
		)
	}
	if expected.SpeedMbps > 0 && parseSpeedMbps(connection.Edge.Speed) != expected.SpeedMbps {
		findings = append(
			findings,
			linkConflict(
				FindingSpeedConflict,
				expected,
				strconv.Itoa(expected.SpeedMbps),
				connection.Edge.Speed,
			),
		)
	}
	if expected.NativeVLAN > 0 && parseVLAN(connection.Edge.VLAN) != expected.NativeVLAN {
		findings = append(
			findings,
			linkConflict(
				FindingVLANConflict,
				expected,
				strconv.Itoa(expected.NativeVLAN),
				connection.Edge.VLAN,
			),
		)
	}
	return findings
}

func compareUnexpectedDevices(authored []AuthoredDevice, observed []ObservedHost) []Finding {
	known := make(map[string]bool, len(authored))
	prefixes := make(map[string]bool)
	for _, device := range authored {
		known[device.MAC] = true
		prefixes[strings.SplitN(device.Name, "-", namePrefixParts)[0]] = true
	}
	var findings []Finding
	for _, host := range observed {
		if !known[normalizeMAC(host.MAC)] &&
			prefixes[strings.SplitN(host.Name, "-", namePrefixParts)[0]] {
			findings = append(findings, Finding{
				Kind: FindingUnexpectedDevice, Device: host.Name, Observed: normalizeMAC(host.MAC),
			})
		}
	}
	return findings
}

func compareUnexpectedLinks(authored AuthoredSnapshot, observed []ObservedHost) []Finding {
	devices := make(map[string]AuthoredDevice, len(authored.Devices))
	for _, device := range authored.Devices {
		devices[device.MAC] = device
	}
	expected := make(map[string]bool, len(authored.Links))
	for _, link := range authored.Links {
		expected[linkKey(link.SourceMAC, link.TargetMAC)] = true
	}
	return findUnexpectedLinks(observed, devices, expected)
}

func findUnexpectedLinks(
	hosts []ObservedHost,
	devices map[string]AuthoredDevice,
	expected map[string]bool,
) []Finding {
	seen := make(map[string]bool)
	var findings []Finding
	for _, host := range hosts {
		source, sourceKnown := devices[normalizeMAC(host.MAC)]
		for _, connection := range host.Connections {
			target, targetKnown := devices[normalizeMAC(connection.MAC)]
			key := linkKey(source.MAC, target.MAC)
			if sourceKnown && targetKnown && !expected[key] && !seen[key] {
				findings = append(findings, Finding{
					Kind: FindingUnexpectedLink, Device: source.Name, Peer: target.Name,
				})
				seen[key] = true
			}
		}
	}
	return findings
}

func findConnection(host ObservedHost, targetMAC string) (ObservedConnection, bool) {
	for _, connection := range host.Connections {
		if normalizeMAC(connection.MAC) == targetMAC {
			return connection, true
		}
	}
	return ObservedConnection{}, false
}

func conflict(kind FindingKind, device, expected, observed string) Finding {
	return Finding{Kind: kind, Device: device, Expected: expected, Observed: observed}
}

func linkConflict(kind FindingKind, link AuthoredLink, expected, observed string) Finding {
	return Finding{
		Kind:     kind,
		Device:   link.Source,
		Peer:     link.Target,
		Expected: expected,
		Observed: observed,
	}
}
