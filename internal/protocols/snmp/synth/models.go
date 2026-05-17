package synth

import (
	"fmt"
	"sort"
)

// Model identifies a specific platform within a vendor. When the
// operator picks a model in the Generate-walk modal, the synthesised
// walk gets the model's real sysObjectID + a sysDescr that matches
// what the real gear announces — closer to "indistinguishable from
// real hardware" than the per-vendor generic profile.
//
// The string values follow the kebab-case slug convention so they
// can ride straight through the JSON API + the UI dropdown without
// any per-vendor escaping rules.
type Model string

// ModelDescriptor is the UI-facing view of a model. Returned by
// Models() / ModelsForType() so the Generate-walk dropdown can
// render labels + group by vendor without knowing the per-model
// Profile fields directly.
type ModelDescriptor struct {
	Vendor    Vendor     `json:"vendor"`
	Model     Model      `json:"model"`
	Label     string     `json:"label"`     // "Catalyst 9300-48P (48× 1G)"
	Type      DeviceType `json:"type"`      // type implied by the model
	IfCount   int        `json:"ifCount"`   // default ifTable rows
	SpeedMbps int        `json:"speedMbps"` // per-port speed
}

// modelEntry is the canonical record stored in modelTable. It builds
// a Profile on demand so the per-model facts (sysDescr template,
// sysObjectID, interface naming convention) stay co-located.
type modelEntry struct {
	descriptor   ModelDescriptor
	sysDescr     string
	sysObjectID  string
	ifNameFormat string
	// Per-model overrides for the include* flags. Default values
	// inherit from the vendor's per-type profile via deriveFlags so
	// new models don't need to think about every flag.
	overrideFlags *profileFlags
}

type profileFlags struct {
	includeIPMIB  bool
	includeBridge bool
	includeLLDP   bool
	ipForwarding  bool
}

// Models returns every model defined for the vendor, sorted by
// (Type, Label) for stable dropdown rendering. Used by the UI to
// build the cascading vendor → model picker.
func Models(v Vendor) []ModelDescriptor {
	var out []ModelDescriptor
	for _, e := range modelTable {
		if e.descriptor.Vendor == v {
			out = append(out, e.descriptor)
		}
	}
	sortDescriptors(out)
	return out
}

// ModelsForType returns models compatible with a device's type.
// Drives the dropdown filtering — picking "Cisco IOS" for a router
// only shows routers, not switches or APs.
func ModelsForType(v Vendor, t DeviceType) []ModelDescriptor {
	var out []ModelDescriptor
	for _, e := range modelTable {
		if e.descriptor.Vendor == v && e.descriptor.Type == t {
			out = append(out, e.descriptor)
		}
	}
	sortDescriptors(out)
	return out
}

// AllModels returns every defined model across every vendor, sorted
// by (Vendor, Type, Label). Used by the daemon's
// /api/v1/synthesize-walk/models endpoint so the UI can build the
// full picker in one round-trip.
func AllModels() []ModelDescriptor {
	out := make([]ModelDescriptor, 0, len(modelTable))
	for _, e := range modelTable {
		out = append(out, e.descriptor)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Vendor != out[j].Vendor {
			return out[i].Vendor < out[j].Vendor
		}
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].Label < out[j].Label
	})
	return out
}

func sortDescriptors(d []ModelDescriptor) {
	sort.Slice(d, func(i, j int) bool {
		if d[i].Type != d[j].Type {
			return d[i].Type < d[j].Type
		}
		return d[i].Label < d[j].Label
	})
}

// PickModel returns the resolved Profile for a specific model. The
// model implies the device type, so callers don't pass a separate
// type — the Profile.Type field is set from the model's descriptor.
//
// Falls back to the vendor's per-type Profile for flags the model
// doesn't override (e.g. a Cisco switch model inherits BRIDGE-MIB
// inclusion from the cisco-ios + switch base profile).
func PickModel(v Vendor, m Model) (Profile, error) {
	e, ok := modelTable[modelKey{v, m}]
	if !ok {
		return Profile{}, fmt.Errorf("synth: no profile for vendor=%s model=%s", v, m)
	}

	flags := deriveFlags(v, e.descriptor.Type, e.overrideFlags)

	return Profile{
		Vendor:         v,
		Type:           e.descriptor.Type,
		SysDescr:       e.sysDescr,
		SysObjectID:    e.sysObjectID,
		IfNameFormat:   e.ifNameFormat,
		DefaultIfCount: e.descriptor.IfCount,
		IfSpeedMbps:    e.descriptor.SpeedMbps,
		IncludeIPMIB:   flags.includeIPMIB,
		IncludeBridge:  flags.includeBridge,
		IncludeLLDP:    flags.includeLLDP,
		IPForwarding:   flags.ipForwarding,
	}, nil
}

// deriveFlags resolves the include* / ipForwarding flags for a
// (vendor, type) — defaults from the per-vendor profile table, then
// applies any per-model override.
func deriveFlags(v Vendor, t DeviceType, override *profileFlags) profileFlags {
	base, ok := lookupProfile(v, t)
	out := profileFlags{}
	if ok {
		out = profileFlags{
			includeIPMIB:  base.IncludeIPMIB,
			includeBridge: base.IncludeBridge,
			includeLLDP:   base.IncludeLLDP,
			ipForwarding:  base.IPForwarding,
		}
	}
	if override != nil {
		out = *override
	}
	return out
}

type modelKey struct {
	v Vendor
	m Model
}

// modelTable is the canonical (vendor, model) → modelEntry map. Each
// entry carries a model-specific sysDescr / sysObjectID; the include
// flags fall back to the per-vendor profile via deriveFlags.
//
// sysObjectID values are real-world identifiers taken from each
// vendor's published MIB. For models where the exact suffix isn't
// public, we use a representative platform-family OID — the daemon
// is a simulator, not a stand-in for a regulatory test rig.
//
//nolint:gochecknoglobals // static lookup
var modelTable = func() map[modelKey]modelEntry {
	out := map[modelKey]modelEntry{}
	add := func(e modelEntry) {
		out[modelKey{e.descriptor.Vendor, e.descriptor.Model}] = e
	}

	// ===== Cisco IOS =====
	// Catalyst 9300 series — 48-port stackable access switches. The
	// 48P SKU is 48× 1G PoE+; the 48Y SKU on 9500 is 48× 25G/100G uplink.
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorCiscoIOS, Model: "catalyst-9300-48p", Type: TypeSwitch,
			Label:   "Catalyst 9300-48P (48× 1G PoE+)",
			IfCount: ports48, SpeedMbps: speed1G,
		},
		sysDescr:     "Cisco IOS Software [Cupertino], Catalyst L3 Switch Software (CAT9K_IOSXE), Version 17.9.4, RELEASE SOFTWARE (fc4)",
		sysObjectID:  "1.3.6.1.4.1.9.1.2238",
		ifNameFormat: "GigabitEthernet1/0/%d",
	})
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorCiscoIOS, Model: "catalyst-9500-48y4c", Type: TypeSwitch,
			Label:   "Catalyst 9500-48Y4C (48× 25G + 4× 100G)",
			IfCount: ports48, SpeedMbps: speed25G,
		},
		sysDescr:     "Cisco IOS Software [Cupertino], Catalyst L3 Switch Software (CAT9K_IOSXE), Version 17.9.4, RELEASE SOFTWARE (fc4)",
		sysObjectID:  "1.3.6.1.4.1.9.1.2779",
		ifNameFormat: "TwentyFiveGigE1/0/%d",
	})
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorCiscoIOS, Model: "nexus-9336c-fx2", Type: TypeSwitch,
			Label:   "Nexus 9336C-FX2 (36× 100G)",
			IfCount: ports36, SpeedMbps: speed100G,
		},
		sysDescr:     "Cisco Nexus Operating System (NX-OS) Software, Version 10.3(2)F",
		sysObjectID:  "1.3.6.1.4.1.9.12.3.1.3.1858",
		ifNameFormat: "Ethernet1/%d",
	})
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorCiscoIOS, Model: "isr-4451-x", Type: TypeRouter,
			Label:   "ISR 4451-X (4× 1G WAN/LAN)",
			IfCount: ports4, SpeedMbps: speed1G,
		},
		sysDescr:     "Cisco IOS Software [Fuji], ISR Software (X86_64_LINUX_IOSD-UNIVERSALK9-M), Version 16.9.4",
		sysObjectID:  "1.3.6.1.4.1.9.1.1990",
		ifNameFormat: "GigabitEthernet0/0/%d",
	})
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorCiscoIOS, Model: "asr-9001", Type: TypeRouter,
			Label:   "ASR 9001 (8× 10G)",
			IfCount: ports8, SpeedMbps: speed10G,
		},
		sysDescr:     "Cisco IOS XR Software, ASR9K Series, Version 7.5.2",
		sysObjectID:  "1.3.6.1.4.1.9.1.1240",
		ifNameFormat: "TenGigE0/0/0/%d",
	})
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorCiscoIOS, Model: "asa-5525-x", Type: TypeFirewall,
			Label:   "ASA 5525-X (8× 1G)",
			IfCount: ports8, SpeedMbps: speed1G,
		},
		sysDescr:     "Cisco Adaptive Security Appliance Software Version 9.16(4)27",
		sysObjectID:  "1.3.6.1.4.1.9.1.915",
		ifNameFormat: "GigabitEthernet0/%d",
	})
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorCiscoIOS, Model: "ftd-2110", Type: TypeFirewall,
			Label:   "Firepower 2110 (12× 1G + 4× 10G)",
			IfCount: ports12, SpeedMbps: speed1G,
		},
		sysDescr:     "Cisco Firepower Threat Defense for the 2100 Series Version 7.2.5",
		sysObjectID:  "1.3.6.1.4.1.9.1.2253",
		ifNameFormat: "Ethernet1/%d",
	})
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorCiscoIOS, Model: "aironet-iw9165e", Type: TypeAccessPoint,
			Label:   "Aironet IW-9165E (Wi-Fi 6E)",
			IfCount: defaultIfAP, SpeedMbps: speed1G,
		},
		sysDescr:     "Cisco Aironet IW-9165E Wi-Fi 6E Access Point, Cisco IOS XE Software, Version 17.13.1",
		sysObjectID:  "1.3.6.1.4.1.9.1.2879",
		ifNameFormat: "Dot11Radio%d",
	})

	// ===== Juniper (Junos) =====
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorJunos, Model: "ex4300-48t", Type: TypeSwitch,
			Label:   "EX4300-48T (48× 1G)",
			IfCount: ports48, SpeedMbps: speed1G,
		},
		sysDescr:     "Juniper Networks, Inc. ex4300-48t Ethernet Switch, kernel JUNOS 22.4R3.25",
		sysObjectID:  "1.3.6.1.4.1.2636.1.1.1.2.29",
		ifNameFormat: "ge-0/0/%d",
	})
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorJunos, Model: "ex4400-48t", Type: TypeSwitch,
			Label:   "EX4400-48T (48× 1G + 4× 25G uplink)",
			IfCount: ports48, SpeedMbps: speed1G,
		},
		sysDescr:     "Juniper Networks, Inc. ex4400-48t Ethernet Switch, kernel JUNOS 22.4R3.25",
		sysObjectID:  "1.3.6.1.4.1.2636.1.1.1.2.143",
		ifNameFormat: "ge-0/0/%d",
	})
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorJunos, Model: "qfx5100-48s", Type: TypeSwitch,
			Label:   "QFX5100-48S (48× 10G + 6× 40G uplink)",
			IfCount: ports48, SpeedMbps: speed10G,
		},
		sysDescr:     "Juniper Networks, Inc. qfx5100-48s Internet Router, kernel JUNOS 21.4R3.15",
		sysObjectID:  "1.3.6.1.4.1.2636.1.1.1.2.71",
		ifNameFormat: "xe-0/0/%d",
	})
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorJunos, Model: "mx204", Type: TypeRouter,
			Label:   "MX204 (4× 100G + 8× 10G)",
			IfCount: ports4, SpeedMbps: speed100G,
		},
		sysDescr:     "Juniper Networks, Inc. mx204 Internet router, kernel JUNOS 21.4R3.15",
		sysObjectID:  "1.3.6.1.4.1.2636.1.1.1.2.129",
		ifNameFormat: "et-0/0/%d",
	})
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorJunos, Model: "mx480", Type: TypeRouter,
			Label:   "MX480 (modular chassis, 100G capable)",
			IfCount: ports16, SpeedMbps: speed100G,
		},
		sysDescr:     "Juniper Networks, Inc. mx480 Internet router, kernel JUNOS 21.4R3.15",
		sysObjectID:  "1.3.6.1.4.1.2636.1.1.1.2.25",
		ifNameFormat: "xe-2/0/%d",
	})
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorJunos, Model: "srx340", Type: TypeFirewall,
			Label:   "SRX340 (16× 1G)",
			IfCount: ports16, SpeedMbps: speed1G,
		},
		sysDescr:     "Juniper Networks, Inc. srx340 Services Gateway, kernel JUNOS 21.4R3.15",
		sysObjectID:  "1.3.6.1.4.1.2636.1.1.1.2.82",
		ifNameFormat: "ge-0/0/%d",
	})
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorJunos, Model: "srx1500", Type: TypeFirewall,
			Label:   "SRX1500 (16× 1G + 4× 10G)",
			IfCount: ports16, SpeedMbps: speed1G,
		},
		sysDescr:     "Juniper Networks, Inc. srx1500 Services Gateway, kernel JUNOS 21.4R3.15",
		sysObjectID:  "1.3.6.1.4.1.2636.1.1.1.2.114",
		ifNameFormat: "ge-0/0/%d",
	})
	// Juniper Mist AP43 — API-first, minimal SNMP per the design res.
	// Overrides default flags so LLDP / BRIDGE / IP-MIB are all off.
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorJuniperMist, Model: "mist-ap43", Type: TypeAccessPoint,
			Label:   "Mist AP43 (Wi-Fi 6, API-first)",
			IfCount: defaultIfAP, SpeedMbps: speed1G,
		},
		sysDescr:      "Juniper Mist AP43, Mist OS 0.13.x",
		sysObjectID:   "1.3.6.1.4.1.2636.1.1.1.4.42",
		ifNameFormat:  "wifi%d",
		overrideFlags: &profileFlags{},
	})

	// ===== Arista EOS =====
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorAristaEOS, Model: "dcs-7050sx", Type: TypeSwitch,
			Label:   "DCS-7050SX (32× 10G/25G ToR)",
			IfCount: ports32, SpeedMbps: speed25G,
		},
		sysDescr:     "Arista Networks EOS version 4.30.4M running on an Arista Networks DCS-7050SX-72",
		sysObjectID:  "1.3.6.1.4.1.30065.1.3.1.32",
		ifNameFormat: "Ethernet%d",
	})
	add(modelEntry{
		descriptor: ModelDescriptor{
			Vendor: VendorAristaEOS, Model: "dcs-7280sr", Type: TypeRouter,
			Label:   "DCS-7280SR (48× 10G + 6× 100G spine)",
			IfCount: ports48, SpeedMbps: speed100G,
		},
		sysDescr:     "Arista Networks EOS version 4.30.4M running on an Arista Networks DCS-7280SR-48C6",
		sysObjectID:  "1.3.6.1.4.1.30065.1.3.1.60",
		ifNameFormat: "Ethernet%d",
	})

	return out
}()
