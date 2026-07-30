// Package walkprofile infers a reviewable authoring profile from sanitized
// SNMP walk data. It never persists credentials or raw capture requests.
package walkprofile

import (
	"sort"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp/synth"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
	"github.com/MustardSeedNetworks/niac-go/internal/walkanalysis"
)

// Review is the sanitized evidence and inferred profile shown to the operator
// before a reusable profile can be created.
type Review struct {
	WalkName string                 `json:"walkName"`
	Profile  scenario.DeviceProfile `json:"profile"`
	Analysis *walkanalysis.Analysis `json:"analysis"`
}

// Infer builds a conservative candidate. Exact supported sysObjectIDs win;
// otherwise sysDescr and interface evidence provide editable defaults.
func Infer(walkName string, entries []snmp.WalkEntry) Review {
	analysis := walkanalysis.Analyze(entries)
	supported := supportedData(entries)
	profile := inferProfile(analysis)
	profile.WalkName = walkName
	profile.InterfaceCount = analysis.Statistics.TotalInterfaces
	profile.SupportedSNMPData = supported
	profile.Interfaces = profileInterfaces(analysis.Interfaces)
	profile.Source = "captured"
	return Review{WalkName: walkName, Profile: profile, Analysis: analysis}
}

func profileInterfaces(interfaces []walkanalysis.Interface) []scenario.ProfileInterface {
	result := make([]scenario.ProfileInterface, 0, len(interfaces))
	for _, iface := range interfaces {
		result = append(result, scenario.ProfileInterface{
			Name: iface.Name, Type: interfaceType(iface.Type), MTU: iface.MTU, Speed: iface.Speed,
			AdminStatus: iface.AdminStatus, OperStatus: iface.OperStatus,
		})
	}
	return result
}

func interfaceType(value string) string {
	switch value {
	case "ethernetCsmacd", "fastEther", "fastEtherFX", "gigabitEthernet":
		return "ethernet"
	case "softwareLoopback":
		return "loopback"
	case "l2vlan", "l3ipvlan":
		return value
	default:
		return "other"
	}
}

func inferProfile(analysis *walkanalysis.Analysis) scenario.DeviceProfile {
	if descriptor, profile, ok := synth.MatchSysObjectID(analysis.Device.SysObjectID); ok {
		return scenario.DeviceProfile{
			Role: "captured-" + scenarioDeviceType(
				profile.Type,
			), DeviceType: scenarioDeviceType(profile.Type),
			Vendor: vendorLabel(
				profile.Vendor,
			), Model: descriptor.Label, Platform: profile.SysDescr,
			SysObjectID: profile.SysObjectID,
		}
	}

	description := strings.TrimSpace(analysis.Device.SysDescr)
	lower := strings.ToLower(description)
	deviceType := inferDeviceType(lower, analysis.Statistics.PhysicalInterfaces)
	vendor := inferVendor(lower)
	model := description
	if model == "" {
		model = vendor + " " + strings.ReplaceAll(deviceType, "-", " ")
	}
	return scenario.DeviceProfile{
		Role: "captured-" + deviceType, DeviceType: deviceType, Vendor: vendor,
		Model: model, Platform: model, SysObjectID: analysis.Device.SysObjectID,
	}
}

func inferDeviceType(description string, physicalInterfaces int) string {
	switch {
	case strings.Contains(description, "firewall"), strings.Contains(description, "pan-os"),
		strings.Contains(description, "fortigate"):
		return "firewall"
	case strings.Contains(description, "wireless"), strings.Contains(description, "access point"):
		return "access-point"
	case strings.Contains(description, "router"), strings.Contains(description, "ios xr"):
		return "router"
	case strings.Contains(description, "switch"),
		strings.Contains(description, "nx-os"),
		physicalInterfaces > 1:
		return "switch"
	default:
		return "host"
	}
}

func inferVendor(description string) string {
	switch {
	case strings.Contains(description, "cisco"):
		return "cisco"
	case strings.Contains(description, "juniper"), strings.Contains(description, "junos"):
		return "juniper"
	case strings.Contains(description, "arista"):
		return "arista"
	case strings.Contains(description, "aruba"):
		return "aruba"
	case strings.Contains(description, "extreme"):
		return "extreme"
	case strings.Contains(description, "palo alto"), strings.Contains(description, "pan-os"):
		return "palo alto"
	default:
		return "generic"
	}
}

func scenarioDeviceType(deviceType synth.DeviceType) string {
	return strings.ReplaceAll(string(deviceType), "_", "-")
}

func vendorLabel(vendor synth.Vendor) string {
	switch vendor {
	case synth.VendorCiscoIOS:
		return "cisco"
	case synth.VendorJunos, synth.VendorJuniperMist:
		return "juniper"
	case synth.VendorAristaEOS:
		return "arista"
	case synth.VendorAruba:
		return "aruba"
	case synth.VendorExtreme:
		return "extreme"
	case synth.VendorHP:
		return "hp"
	case synth.VendorPaloAlto:
		return "palo alto"
	case synth.VendorGeneric:
		return "generic"
	}
	return string(vendor)
}

func supportedData(entries []snmp.WalkEntry) []string {
	found := make(map[string]bool)
	for _, entry := range entries {
		oid := strings.TrimPrefix(entry.OID, ".")
		switch {
		case strings.HasPrefix(oid, "1.0.8802.1.1.2"):
			found["lldp"] = true
		case strings.HasPrefix(oid, "1.3.6.1.4.1.9.9.23"):
			found["cdp"] = true
		case strings.HasPrefix(oid, "1.3.6.1.2.1.17.7"):
			found["vlans"] = true
			found["bridge"] = true
		case strings.HasPrefix(oid, "1.3.6.1.2.1.17"):
			found["bridge"] = true
		case strings.HasPrefix(oid, "1.3.6.1.2.1.31.1.1"), strings.HasPrefix(oid, "1.3.6.1.2.1.2"):
			found["interfaces"] = true
		case strings.HasPrefix(oid, "1.3.6.1.2.1.4.21"), strings.HasPrefix(oid, "1.3.6.1.2.1.4.24"):
			found["routing"] = true
			found["ip"] = true
		case strings.HasPrefix(oid, "1.3.6.1.2.1.4"):
			found["ip"] = true
		case strings.HasPrefix(oid, "1.3.6.1.2.1.1"):
			found["system"] = true
		}
	}
	result := make([]string, 0, len(found))
	for capability := range found {
		result = append(result, capability)
	}
	sort.Strings(result)
	return result
}
