package snmp

import (
	"bytes"
	"strings"
)

// NormalizeKnownWalkOIDs converts symbolic objects used by profile inference
// to numeric OIDs while preserving every value and unknown vendor object.
func NormalizeKnownWalkOIDs(content []byte) []byte {
	lines := bytes.Split(content, []byte("\n"))
	for index, line := range lines {
		separator := bytes.IndexByte(line, '=')
		if separator < 0 {
			continue
		}
		oid := strings.TrimSpace(string(line[:separator]))
		numeric, resolved := NormalizeWalkOID(oid)
		if !resolved || numeric == oid {
			continue
		}
		lines[index] = append([]byte(numeric+" "), line[separator:]...)
	}
	return bytes.Join(lines, []byte("\n"))
}

// NormalizeWalkOID resolves a walk line's object to numeric form, and reports
// whether it could be resolved at all.
//
// A walk taken with `snmpwalk -On` is already numeric and passes through. One
// taken without it names objects symbolically, and only two kinds can be
// resolved without a MIB compiler: an object in the fixed set below, and an SMI
// anchor — the ancestor net-snmp falls back to when it has no MIB for the
// subtree, always followed by a numeric tail.
//
// Anything else must be refused rather than stored. A non-numeric key is
// unusable in both directions: parseOIDParts silently drops every arc that is
// not an integer, so two unrelated symbolic OIDs can compare equal and break
// the ordering GET-NEXT depends on, and gosnmp cannot marshal one onto the wire
// at all — one such varbind fails the whole response.
func NormalizeWalkOID(oid string) (string, bool) {
	if IsValidOID(oid) {
		return oid, true
	}

	name, suffix, found := strings.Cut(oid, ".")
	if !found {
		name = oid
	}

	base, known := knownWalkOIDBase(strings.ToLower(name))
	if !known {
		return "", false
	}

	if suffix == "" {
		return base, true
	}

	// A symbolic INDEX — `IP-MIB::icmpMsgStatsOutPkts.ipv6.3` — names an
	// enumeration no name table resolves. The object is known; the row is not.
	if !IsValidOID(suffix) {
		return "", false
	}

	return base + "." + suffix, true
}

func knownWalkOIDBase(name string) (string, bool) {
	switch name {
	case "snmpv2-mib::sysdescr":
		return ".1.3.6.1.2.1.1.1", true
	case "snmpv2-mib::sysobjectid":
		return ".1.3.6.1.2.1.1.2", true
	case "snmpv2-mib::syscontact":
		return ".1.3.6.1.2.1.1.4", true
	case "snmpv2-mib::sysname":
		return ".1.3.6.1.2.1.1.5", true
	case "snmpv2-mib::syslocation":
		return ".1.3.6.1.2.1.1.6", true
	case "if-mib::ifdescr":
		return ".1.3.6.1.2.1.2.2.1.2", true
	case "if-mib::iftype":
		return ".1.3.6.1.2.1.2.2.1.3", true
	case "if-mib::ifmtu":
		return ".1.3.6.1.2.1.2.2.1.4", true
	case "if-mib::ifspeed":
		return ".1.3.6.1.2.1.2.2.1.5", true
	case "if-mib::ifphysaddress":
		return ".1.3.6.1.2.1.2.2.1.6", true
	case "if-mib::ifadminstatus":
		return ".1.3.6.1.2.1.2.2.1.7", true
	case "if-mib::ifoperstatus":
		return ".1.3.6.1.2.1.2.2.1.8", true
	case "if-mib::ifname":
		return ".1.3.6.1.2.1.31.1.1.1.1", true
	case "if-mib::ifhighspeed":
		return ".1.3.6.1.2.1.31.1.1.1.15", true
	case "if-mib::ifalias":
		return ".1.3.6.1.2.1.31.1.1.1.18", true
	default:
		return knownDiscoveryWalkOIDBase(name)
	}
}

// smiAnchorOIDBase resolves the RFC 2578 registration anchors. net-snmp prints
// these when it has no MIB for a subtree, so the name is always followed by a
// numeric tail — `SNMPv2-SMI::enterprises.9.12.3.1.3.1008`. They are a fixed
// part of the SMI rather than a per-vendor object list.
func smiAnchorOIDBase(name string) (string, bool) {
	// net-snmp with no MIBs loaded at all drops the module prefix and prints the
	// bare anchor — `iso.3.6.1.2.1.1.5.0`. Same registration, same fix.
	if bare, _, found := strings.Cut(name, "::"); !found {
		name = "snmpv2-smi::" + bare
	}

	switch name {
	case "snmpv2-smi::iso":
		return ".1", true
	case "snmpv2-smi::org":
		return ".1.3", true
	case "snmpv2-smi::dod":
		return ".1.3.6", true
	case "snmpv2-smi::internet":
		return ".1.3.6.1", true
	case "snmpv2-smi::directory":
		return ".1.3.6.1.1", true
	case "snmpv2-smi::mgmt":
		return ".1.3.6.1.2", true
	case "snmpv2-smi::mib-2", "rfc1213-mib::mib-2":
		return ".1.3.6.1.2.1", true
	case "snmpv2-smi::transmission":
		return ".1.3.6.1.2.1.10", true
	case "snmpv2-smi::experimental":
		return ".1.3.6.1.3", true
	case "snmpv2-smi::private":
		return ".1.3.6.1.4", true
	case "snmpv2-smi::enterprises":
		return ".1.3.6.1.4.1", true
	case "snmpv2-smi::security":
		return ".1.3.6.1.5", true
	case "snmpv2-smi::snmpv2":
		return ".1.3.6.1.6", true
	case "snmpv2-smi::snmpdomains":
		return ".1.3.6.1.6.1", true
	case "snmpv2-smi::snmpproxys":
		return ".1.3.6.1.6.2", true
	case "snmpv2-smi::snmpmodules":
		return ".1.3.6.1.6.3", true
	default:
		return "", false
	}
}

func knownDiscoveryWalkOIDBase(name string) (string, bool) {
	switch name {
	case "lldp-mib::lldplocportid":
		return ".1.0.8802.1.1.2.1.3.7.1.3", true
	case "lldp-mib::lldpremchassisid":
		return ".1.0.8802.1.1.2.1.4.1.1.5", true
	case "lldp-mib::lldpremportid":
		return ".1.0.8802.1.1.2.1.4.1.1.7", true
	case "lldp-mib::lldpremportdesc":
		return ".1.0.8802.1.1.2.1.4.1.1.8", true
	case "lldp-mib::lldpremsysname":
		return ".1.0.8802.1.1.2.1.4.1.1.9", true
	case "cisco-cdp-mib::cdpcachedeviceid":
		return ".1.3.6.1.4.1.9.9.23.1.2.1.1.6", true
	case "cisco-cdp-mib::cdpcachedeviceport":
		return ".1.3.6.1.4.1.9.9.23.1.2.1.1.7", true
	default:
		return smiAnchorOIDBase(name)
	}
}
