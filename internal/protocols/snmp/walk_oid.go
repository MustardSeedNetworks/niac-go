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
		name, suffix, found := strings.Cut(oid, ".")
		if !found {
			name = oid
		}
		base, known := knownWalkOIDBase(strings.ToLower(name))
		if !known {
			continue
		}
		if suffix != "" {
			base += "." + suffix
		}
		lines[index] = append([]byte(base+" "), line[separator:]...)
	}
	return bytes.Join(lines, []byte("\n"))
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
		return "", false
	}
}
