// Package synth generates baseline SNMP walk files from a device
// type and a vendor selector. See docs/design/2026-05-baseline-walk-synthesis.md
// for the contract; tl;dr — POST /api/v1/devices/{hostname}/synthesize-walk
// gives an operator-clicked device a usable SNMP responder without
// requiring them to capture a real walk from hardware.
//
// The package is intentionally small. Three files:
//   - profiles.go (this) — vendor + per-type platform tables
//   - build.go            — walk-line emission for the system group,
//     ifTable, ipAddrTable, bridge / lldp tables
//   - format.go           — single-line emit helpers
package synth

import (
	"fmt"
	"strings"
)

// Vendor identifies the chosen vendor profile. Strings match the
// design doc + the UI dropdown values.
type Vendor string

const (
	VendorCiscoIOS  Vendor = "cisco-ios"
	VendorJunos     Vendor = "junos"
	VendorAristaEOS Vendor = "arista-eos"
	VendorGeneric   Vendor = "generic"
	// VendorJuniperMist is the Mist-acquisition AP line. API-first, so
	// the synthesised walk is minimal (system group + bare ifTable).
	// Selected automatically when the operator picks Junos + access_point;
	// not a top-level UI option in its own right today.
	VendorJuniperMist Vendor = "junos-mist"
)

// Default per-type interface counts + speeds. Pulled out as named
// constants so the per-row add() lines stay readable and the mnd
// linter doesn't fire on every literal.
const (
	defaultIfSwitch   = 24
	defaultIfRouter   = 4
	defaultIfFirewall = 4
	defaultIfAP       = 2
	defaultIfHost     = 1
	defaultIfServer   = 1
	defaultIfPhone    = 2
	defaultIfPrinter  = 1

	// Link speeds in Mbps. Full range from 100M up to 100G so DC-class
	// gear (40G/100G uplinks, 25G/50G server NICs) gets the right
	// ifSpeed gauge — the simulator surfaces this raw on the wire and
	// pollers compare it against their alarm thresholds.
	speed100M = 100
	speed1G   = 1000
	speed10G  = 10_000
	speed25G  = 25_000
	speed40G  = 40_000
	speed50G  = 50_000
	speed100G = 100_000
)

// AllVendors lists every operator-selectable vendor in UI dropdown order.
// Mist is selected automatically when needed, not surfaced separately.
//
//nolint:gochecknoglobals // static lookup
var AllVendors = []Vendor{
	VendorCiscoIOS, VendorJunos, VendorAristaEOS, VendorGeneric,
}

// DeviceType is the niac device.type string. Kept as a typed alias
// so map keys are self-documenting.
type DeviceType string

const (
	TypeSwitch      DeviceType = "switch"
	TypeRouter      DeviceType = "router"
	TypeFirewall    DeviceType = "firewall"
	TypeAccessPoint DeviceType = "access_point"
	TypeHost        DeviceType = "host"
	TypeServer      DeviceType = "server"
	TypeVoIPPhone   DeviceType = "voip_phone"
	TypePrinter     DeviceType = "printer"
)

// Profile captures everything the build pass needs to know about a
// (vendor, type) pair: the sysDescr / sysObjectID stamping, the
// interface naming convention, and the default interface count.
type Profile struct {
	Vendor         Vendor
	Type           DeviceType
	SysDescr       string // already substituted; ready to emit
	SysObjectID    string // OID, e.g. "1.3.6.1.4.1.9.1.745"
	IfNameFormat   string // printf'd with the 1-based interface index
	DefaultIfCount int
	IfSpeedMbps    int
	IncludeIPMIB   bool // emit ipAddrTable + ipForwarding=1 (routers / firewalls)
	IncludeBridge  bool // emit dot1dBridge group (switches)
	IncludeLLDP    bool // emit minimal lldpRemTable (switches / routers / APs / phones)
	IPForwarding   bool // true for routers + firewalls
}

// Synthesizable returns true when this (vendor, type) is supported.
// Used by the daemon's 422 safety net and the UI dropdown disable
// logic. Mist is the only auto-routed entry — operators don't pick
// it directly.
func Synthesizable(vendor Vendor, t DeviceType) bool {
	_, ok := lookupProfile(vendor, t)
	return ok
}

// Pick returns the resolved Profile for a (vendor, type) request,
// auto-routing Junos + access_point to the Mist sub-profile. Returns
// an error rather than silently falling back to generic — per the
// #4 resolution in the design doc, the daemon never substitutes a
// different vendor.
func Pick(vendor Vendor, t DeviceType) (Profile, error) {
	if vendor == VendorJunos && t == TypeAccessPoint {
		vendor = VendorJuniperMist
	}
	p, ok := lookupProfile(vendor, t)
	if !ok {
		return Profile{}, fmt.Errorf("synth: no profile for vendor=%s type=%s", vendor, t)
	}
	return p, nil
}

// lookupProfile is the raw table lookup; callers should prefer Pick
// so the Junos→Mist routing happens consistently.
func lookupProfile(vendor Vendor, t DeviceType) (Profile, bool) {
	key := profileKey{vendor, t}
	p, ok := profileTable[key]
	return p, ok
}

type profileKey struct {
	v Vendor
	t DeviceType
}

// profileTable is the canonical (vendor, type) → Profile map. Built
// from the design-doc tables so missing entries are deliberate
// rather than oversights — see the table in
// docs/design/2026-05-baseline-walk-synthesis.md.
//
//nolint:gochecknoglobals // static lookup
var profileTable = func() map[profileKey]Profile {
	out := map[profileKey]Profile{}

	add := func(v Vendor, t DeviceType, p Profile) {
		p.Vendor = v
		p.Type = t
		out[profileKey{v, t}] = p
	}

	// ---- Cisco IOS ----
	cisco := func(platform, oidSuffix, ifFmt string, n int) Profile {
		return Profile{
			SysDescr: fmt.Sprintf(
				"Cisco IOS Software, %s Software (XXXX-K9), Version 15.7(3)M3, RELEASE SOFTWARE (fc1)",
				platform,
			),
			SysObjectID:    "1.3.6.1.4.1.9.1." + oidSuffix,
			IfNameFormat:   ifFmt,
			DefaultIfCount: n,
			IfSpeedMbps:    speed1G,
		}
	}
	add(
		VendorCiscoIOS,
		TypeSwitch,
		withFlags(cisco("Catalyst 3850", "1641", "GigabitEthernet1/0/%d", defaultIfSwitch), false, true, true, false),
	)
	add(
		VendorCiscoIOS,
		TypeRouter,
		withFlags(cisco("ISR 4451-X", "1990", "GigabitEthernet0/0/%d", defaultIfRouter), true, false, true, true),
	)
	add(
		VendorCiscoIOS,
		TypeFirewall,
		withFlags(cisco("ASA 5525-X", "915", "GigabitEthernet0/%d", defaultIfFirewall), true, false, false, true),
	)
	add(
		VendorCiscoIOS,
		TypeAccessPoint,
		withFlags(cisco("Aironet 2802i", "2659", "Dot11Radio%d", defaultIfAP), false, false, true, false),
	)
	add(
		VendorCiscoIOS,
		TypeVoIPPhone,
		withFlags(cisco("IP Phone 7945", "404", "Port-channel%d", defaultIfPhone), false, false, true, false),
	)

	// ---- Juniper (Junos) ----
	// Per-platform default speeds: EX4300 is 1G access switch, MX204
	// is a 100G universal router, SRX340 is a 1G branch firewall.
	junos := func(platform, oidSuffix, ifFmt string, n, speedMbps int) Profile {
		return Profile{
			SysDescr: fmt.Sprintf(
				"Juniper Networks, Inc. %s Internet router, kernel JUNOS 21.4R3.15 #0 Build date: 2024-01-15",
				platform,
			),
			SysObjectID:    "1.3.6.1.4.1.2636.1.1.1.2." + oidSuffix,
			IfNameFormat:   ifFmt,
			DefaultIfCount: n,
			IfSpeedMbps:    speedMbps,
		}
	}
	add(
		VendorJunos,
		TypeSwitch,
		withFlags(
			junos("EX4300", "29", "ge-0/0/%d", defaultIfSwitch, speed1G),
			false, true, true, false,
		),
	)
	add(
		VendorJunos,
		TypeRouter,
		withFlags(
			junos("MX204", "129", "ge-0/0/%d", defaultIfRouter, speed100G),
			true, false, true, true,
		),
	)
	add(
		VendorJunos,
		TypeFirewall,
		withFlags(
			junos("SRX340", "82", "ge-0/0/%d", defaultIfRouter, speed1G),
			true, false, false, true,
		),
	)

	// ---- Juniper Mist (AP only — domain-noted per design doc res.) ----
	add(VendorJuniperMist, TypeAccessPoint, Profile{
		SysDescr:       "Juniper Mist AP43, Mist OS 0.13.x",
		SysObjectID:    "1.3.6.1.4.1.2636.1.1.1.4.42",
		IfNameFormat:   "wifi%d",
		DefaultIfCount: defaultIfAP,
		IfSpeedMbps:    speed1G,
		IncludeLLDP:    false, // Mist is API-first; SNMP is minimal
	})

	// ---- Arista EOS ----
	// Arista is data-centre gear — defaults match what the platform
	// actually ships: 7050SX = 25/10G ToR, 7280SR = 100G spine.
	arista := func(platform, oidSuffix, ifFmt string, n, speedMbps int) Profile {
		return Profile{
			SysDescr: fmt.Sprintf(
				"Arista Networks EOS version 4.30.4M running on an Arista Networks %s",
				platform,
			),
			SysObjectID:    "1.3.6.1.4.1.30065.1.3.1." + oidSuffix,
			IfNameFormat:   ifFmt,
			DefaultIfCount: n,
			IfSpeedMbps:    speedMbps,
		}
	}
	add(
		VendorAristaEOS,
		TypeSwitch,
		withFlags(
			arista("DCS-7050SX", "32", "Ethernet%d", defaultIfSwitch, speed25G),
			false, true, true, false,
		),
	)
	add(
		VendorAristaEOS,
		TypeRouter,
		withFlags(
			arista("DCS-7280SR", "60", "Ethernet%d", defaultIfRouter, speed100G),
			true, false, true, true,
		),
	)

	// ---- Generic (Net-SNMP) — host / server / printer / generic-anything ----
	generic := func(descr, ifFmt string, n int, speed int) Profile {
		return Profile{
			SysDescr:       descr,
			SysObjectID:    "1.3.6.1.4.1.8072.3.2.10", // Net-SNMP "linux"
			IfNameFormat:   ifFmt,
			DefaultIfCount: n,
			IfSpeedMbps:    speed,
		}
	}
	add(
		VendorGeneric,
		TypeSwitch,
		withFlags(
			generic("Linux switch 6.1.0 #1 SMP x86_64 GNU/Linux", "eth%d", defaultIfSwitch, speed1G),
			false,
			true,
			false,
			false,
		),
	)
	add(
		VendorGeneric,
		TypeRouter,
		withFlags(
			generic("Linux router 6.1.0 #1 SMP x86_64 GNU/Linux", "eth%d", defaultIfRouter, speed1G),
			true,
			false,
			false,
			true,
		),
	)
	add(
		VendorGeneric,
		TypeFirewall,
		withFlags(
			generic("Linux firewall 6.1.0 #1 SMP x86_64 GNU/Linux", "eth%d", defaultIfRouter, speed1G),
			true,
			false,
			false,
			true,
		),
	)
	add(
		VendorGeneric,
		TypeAccessPoint,
		withFlags(
			generic("Linux AP 6.1.0 #1 SMP armv7l GNU/Linux", "wlan%d", defaultIfAP, speed1G),
			false,
			false,
			false,
			false,
		),
	)
	add(
		VendorGeneric,
		TypeHost,
		withFlags(
			generic("Linux host 6.1.0 #1 SMP x86_64 GNU/Linux", "eth%d", defaultIfHost, speed1G),
			false,
			false,
			false,
			false,
		),
	)
	add(
		VendorGeneric,
		TypeServer,
		withFlags(
			generic("Linux server 6.1.0 #1 SMP x86_64 GNU/Linux", "eth%d", defaultIfServer, speed10G),
			false,
			false,
			false,
			false,
		),
	)
	add(
		VendorGeneric,
		TypeVoIPPhone,
		withFlags(
			generic("Generic SIP phone, firmware 1.0", "eth%d", defaultIfPhone, speed100M),
			false,
			false,
			false,
			false,
		),
	)
	add(
		VendorGeneric,
		TypePrinter,
		withFlags(
			generic("Linux printer 6.1.0 #1 SMP arm GNU/Linux", "eth%d", defaultIfPrinter, speed100M),
			false,
			false,
			false,
			false,
		),
	)

	return out
}()

// withFlags is a tiny constructor helper so each add() call reads
// cleanly without repeating field names. Order matches Profile's
// remaining bool flags.
func withFlags(p Profile, ipMIB, bridge, lldp, ipForward bool) Profile {
	p.IncludeIPMIB = ipMIB
	p.IncludeBridge = bridge
	p.IncludeLLDP = lldp
	p.IPForwarding = ipForward
	return p
}

// IfName renders the interface name for index i (1-based — matches
// ifIndex convention).
func (p Profile) IfName(i int) string {
	// %d-only formats just substitute; formats with a slash (Cisco's
	// GigabitEthernet1/0/%d) work the same way.
	if strings.Contains(p.IfNameFormat, "%d") {
		return fmt.Sprintf(p.IfNameFormat, i-1)
	}
	return fmt.Sprintf("%s%d", p.IfNameFormat, i)
}
