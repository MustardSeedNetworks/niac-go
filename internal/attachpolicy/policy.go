// SPDX-License-Identifier: BUSL-1.1

// Package attachpolicy holds the operator's approval of which host interfaces
// a simulation may bind, and the persistent file that records it.
//
// The approval is the operator's decision about which NIC a client may drive,
// so it has exactly two sources, both outside the API: --attachment-policy
// flags and a root-owned file at [DefaultPath]. A daemon that finds the file
// writable by anyone but root ignores it rather than trust it.
package attachpolicy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// Policy is one approved physical attachment, as the daemon enforces it.
type Policy = fabric.PhysicalAttachmentPolicy

// Syntax is the value grammar shared by the flag and one line of the file.
const Syntax = "INTERFACE=direct, INTERFACE=access:VLAN, or INTERFACE=trunk:VLAN,..."

const (
	minVLAN = 1
	maxVLAN = 4094

	// modesPerInterface is the most modes one interface can hold: a trunk
	// policy for the tagged sessions plus an access policy for the native one.
	modesPerInterface = 2
)

// Parse reads one policy in the flag syntax.
func Parse(value string) (Policy, error) {
	interfaceName, modeValue, ok := strings.Cut(value, "=")
	if !ok || strings.TrimSpace(interfaceName) != interfaceName || interfaceName == "" {
		return Policy{}, syntaxError(value)
	}
	policy := Policy{Interface: interfaceName}
	switch {
	case modeValue == string(fabric.ModeDirect):
		policy.Mode = fabric.ModeDirect
	case strings.HasPrefix(modeValue, string(fabric.ModeAccess)+":"):
		vlan, err := parseVLAN(value, strings.TrimPrefix(modeValue, "access:"), "access VLAN")
		if err != nil {
			return Policy{}, err
		}
		policy.Mode, policy.AccessVLAN = fabric.ModeAccess, vlan
	case strings.HasPrefix(modeValue, string(fabric.ModeTrunk)+":"):
		vlans, err := parseTrunkVLANs(value, strings.TrimPrefix(modeValue, "trunk:"))
		if err != nil {
			return Policy{}, err
		}
		policy.Mode, policy.AllowedVLANs = fabric.ModeTrunk, vlans
	default:
		return Policy{}, syntaxError(value)
	}
	return policy, nil
}

// Format is the canonical flag-syntax value of a policy; Parse(Format(p)) == p.
func Format(policy Policy) string {
	switch policy.Mode {
	case fabric.ModeAccess:
		return fmt.Sprintf("%s=access:%d", policy.Interface, policy.AccessVLAN)
	case fabric.ModeTrunk:
		vlans := make([]string, len(policy.AllowedVLANs))
		for i, vlan := range policy.AllowedVLANs {
			vlans[i] = strconv.FormatUint(uint64(vlan), 10)
		}
		return policy.Interface + "=trunk:" + strings.Join(vlans, ",")
	case fabric.ModeDirect:
		return policy.Interface + "=direct"
	}
	return policy.Interface + "=" + string(policy.Mode)
}

// ParseAll parses flag-syntax values and rejects a set that contradicts itself.
func ParseAll(values []string) ([]Policy, error) {
	policies := make([]Policy, 0, len(values))
	for _, value := range values {
		policy, err := Parse(value)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	if err := Validate(policies); err != nil {
		return nil, err
	}
	return policies, nil
}

// Validate rejects policies that contradict each other.
//
// One interface carries N tagged sessions plus at most one native session,
// exactly as the session registry models a trunk port with a native VLAN
// (#1426), so it needs one policy per mode rather than one policy outright
// (#1463). Direct is the exception: it is unisolated ownership of the whole
// interface, so it cannot share one.
func Validate(policies []Policy) error {
	seenModes := make(map[string]map[fabric.AttachmentMode]struct{}, len(policies))
	for _, policy := range policies {
		modes := seenModes[policy.Interface]
		if modes == nil {
			modes = make(map[fabric.AttachmentMode]struct{}, modesPerInterface)
			seenModes[policy.Interface] = modes
		}
		if _, duplicate := modes[policy.Mode]; duplicate {
			return fmt.Errorf(
				"duplicate %s policy for interface %q in attachment policy %q",
				policy.Mode, policy.Interface, Format(policy),
			)
		}
		if err := checkDirectExclusive(policy, modes); err != nil {
			return err
		}
		modes[policy.Mode] = struct{}{}
	}
	return nil
}

// Merge returns base followed by the extra policies base does not already
// hold, so the same line in a flag and in the file is one approval rather than
// a startup error. It does not check that the result is consistent; Validate
// does, and a real conflict between the two sources must be loud.
func Merge(base, extra []Policy) []Policy {
	merged := make([]Policy, 0, len(base)+len(extra))
	seen := make(map[string]struct{}, len(base)+len(extra))
	for _, policy := range append(append([]Policy{}, base...), extra...) {
		key := Format(policy)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, policy)
	}
	return merged
}

func checkDirectExclusive(policy Policy, modes map[fabric.AttachmentMode]struct{}) error {
	_, hasDirect := modes[fabric.ModeDirect]
	if !hasDirect && policy.Mode != fabric.ModeDirect {
		return nil
	}
	if hasDirect || len(modes) > 0 {
		return fmt.Errorf(
			"interface %q cannot combine direct with another attachment mode (%q)",
			policy.Interface, Format(policy),
		)
	}
	return nil
}

func syntaxError(value string) error {
	return fmt.Errorf("invalid attachment policy %q: expected %s", value, Syntax)
}

func parseVLAN(value, text, label string) (uint16, error) {
	vlan, err := strconv.ParseUint(text, 10, 16)
	if err != nil || vlan < minVLAN || vlan > maxVLAN {
		return 0, fmt.Errorf(
			"invalid attachment policy %q: %s must be between %d and %d",
			value, label, minVLAN, maxVLAN,
		)
	}
	return uint16(vlan), nil
}

func parseTrunkVLANs(value, text string) ([]uint16, error) {
	if text == "" {
		return nil, fmt.Errorf("invalid attachment policy %q: trunk requires at least one VLAN", value)
	}
	vlans := make([]uint16, 0, strings.Count(text, ",")+1)
	var previous uint16
	for item := range strings.SplitSeq(text, ",") {
		vlan, err := parseVLAN(value, item, "trunk VLANs")
		if err != nil {
			return nil, err
		}
		if vlan == previous {
			return nil, fmt.Errorf("invalid attachment policy %q: duplicate VLAN %d", value, vlan)
		}
		if vlan < previous {
			return nil, fmt.Errorf(
				"invalid attachment policy %q: trunk VLANs must be in ascending order", value,
			)
		}
		vlans = append(vlans, vlan)
		previous = vlan
	}
	return vlans, nil
}
