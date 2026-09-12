// Package deviceclass is the one place that knows what a device type means.
//
// Twelve files used to hand-roll their own list of device types — the LLDP and
// CDP capability TLVs, the SNMP agent, the topology link rules, the Link-Live
// comparator, the CLI. Each drifted from the schema in its own way, and each
// drift was invisible until something downstream disagreed:
//
//   - eight builtin templates authored no type at all, so their core routers
//     announced themselves to every discovery tool as end stations (#2096);
//   - `firewall` had no case in either capability switch and fell to a default
//     of host, on a device whose entire job is forwarding;
//   - `voip-phone`, the schema's own spelling, fell through a switch that
//     listed `voip_phone`, so a hand-authored phone silently stopped
//     advertising the LLDP that tells it its voice VLAN.
//
// The schema's `oneof` in converter.Device is the source of truth. This package
// mirrors it, the tests read it rather than restating it, and everything that
// needs to reason about a device type asks here.
package deviceclass

import (
	"slices"
	"strings"
)

// Type is a device type as the schema spells it.
type Type string

// The vocabulary, in the schema's own order.
const (
	Router       Type = "router"
	Switch       Type = "switch"
	Layer3Switch Type = "layer3-switch"
	AP           Type = "ap"
	AccessPoint  Type = "access-point"
	Firewall     Type = "firewall"
	Server       Type = "server"
	Host         Type = "host"
	Workstation  Type = "workstation"
	IoT          Type = "iot"
	Printer      Type = "printer"
	VoipPhone    Type = "voip-phone"

	// Unknown is what an absent or unrecognised type becomes. The schema
	// permits an absent type — it is `omitempty` — so this is a real state,
	// but it is not a value anyone can author.
	Unknown Type = ""
)

// All returns every type the schema accepts, in the schema's own order.
func All() []Type {
	return []Type{
		Router, Switch, Layer3Switch, AP, AccessPoint, Firewall,
		Server, Host, Workstation, IoT, Printer, VoipPhone,
	}
}

// IsKnown reports whether the schema accepts this type.
func IsKnown(t Type) bool {
	return slices.Contains(All(), t)
}

// aliases are spellings the runtime has always tolerated. They are not schema
// values: `niac validate` rejects them. They resolve here so a configuration
// that predates the schema, or a hand-edit that reaches for the underscore,
// still means what its author meant.
func aliases() map[string]Type {
	return map[string]Type{
		"access_point": AccessPoint,
		"accesspoint":  AccessPoint,
		"wireless-ap":  AccessPoint,
		"wireless_ap":  AccessPoint,
		"voip_phone":   VoipPhone,
		"phone":        VoipPhone,
		"pc":           Workstation,
		"client":       Workstation,
	}
}

// Parse resolves a device type as authored. An unrecognised value is Unknown
// rather than an error: the daemon draws and serves a device whose type it does
// not recognise, it just cannot reason about it.
func Parse(raw string) Type {
	lowered := Type(strings.ToLower(strings.TrimSpace(raw)))
	if IsKnown(lowered) {
		return lowered
	}
	if resolved, ok := aliases()[string(lowered)]; ok {
		return resolved
	}
	return Unknown
}

// Forwards reports whether this device moves other devices' traffic.
//
// This is the distinction the wire cares about. A forwarding device announces
// itself as a bridge, a router or an access point; everything else is an end
// station, which IEEE 802.1AB calls "station only" and sets if and only if no
// other capability is set. Getting it wrong is how a firewall came to announce
// itself as a host.
func Forwards(t Type) bool {
	switch t {
	case Router, Switch, Layer3Switch, AP, AccessPoint, Firewall:
		return true
	case Server, Host, Workstation, IoT, Printer, VoipPhone, Unknown:
		return false
	default:
		return false
	}
}

// RunsLLDPByDefault reports whether this device advertises LLDP without being
// asked.
//
// Real hardware in these roles ships with LLDP on. A VoIP phone belongs in that
// group — LLDP-MED is how it learns its voice VLAN and its PoE budget — and it
// was excluded by a spelling accident rather than a decision.
func RunsLLDPByDefault(t Type) bool {
	switch t {
	case Router, Switch, Layer3Switch, AP, AccessPoint, Firewall, VoipPhone:
		return true
	case Server, Host, Workstation, IoT, Printer, Unknown:
		return false
	default:
		return false
	}
}

// RoutesIP reports whether this device forwards between IP subnets.
//
// Narrower than Forwards: a switch and an access point move frames without
// routing. Three files carried this same list of three types independently —
// the topology link rules, the bad-mask fault scope, and the host egress mask —
// which is three chances to disagree about what a layer-3 device is.
func RoutesIP(t Type) bool {
	switch t {
	case Router, Layer3Switch, Firewall:
		return true
	case Switch, AP, AccessPoint, Server, Host, Workstation, IoT, Printer,
		VoipPhone, Unknown:
		return false
	default:
		return false
	}
}

// IsEndpoint reports whether this device is something a person or a process
// uses, rather than infrastructure carrying it.
func IsEndpoint(t Type) bool {
	switch t {
	case Server, Host, Workstation, IoT, Printer, VoipPhone:
		return true
	case Router, Switch, Layer3Switch, AP, AccessPoint, Firewall, Unknown:
		return false
	default:
		return false
	}
}
