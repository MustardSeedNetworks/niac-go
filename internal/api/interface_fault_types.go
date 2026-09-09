package api

import (
	"errors"
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// deviceFaultTargetResponse advertises which service outcomes one device can
// serve; a device with no DHCP or DNS server is absent from the list.
type deviceFaultTargetResponse struct {
	Device     string   `json:"device"`
	Address    string   `json:"address,omitempty"`
	ErrorTypes []string `json:"errorTypes"`
}

type interfaceFaultTargetResponse struct {
	Device     string   `json:"device"`
	Address    string   `json:"address,omitempty"`
	Interfaces []string `json:"interfaces"`
}

var errInterfaceFaultTypeInvalid = errors.New("unsupported fault type")

func availableErrorTypes() []map[string]string {
	descriptions := map[devicestate.FaultType]string{
		devicestate.FaultFCS:         "Frame Check Sequence errors (0-100)",
		devicestate.FaultDiscards:    "Dropped packets (0-100)",
		devicestate.FaultInterface:   "Generic interface errors (0-100)",
		devicestate.FaultUtilization: "Interface bandwidth saturation (0-100%)",
		devicestate.FaultLinkDown:    "Drop the link (non-zero takes the interface down)",
	}
	result := make([]map[string]string, 0, len(descriptions))
	for _, definition := range devicestate.InterfaceFaultDefinitions() {
		result = append(result, map[string]string{
			"type": definition.Label, "description": descriptions[definition.Type],
		})
	}
	return result
}

// availableDeviceErrorTypes describes the device-scoped catalog: service
// outcomes that change what a device answers rather than what its interface
// counters report.
func availableDeviceErrorTypes() []map[string]string {
	descriptions := map[devicestate.FaultType]string{
		devicestate.FaultDHCPNoOffer: "DHCP server consumes the Discover and sends no Offer",
		devicestate.FaultDNSNXDomain: "DNS server answers every query with NXDOMAIN",
		devicestate.FaultDNSTimeout:  "DNS server answers nothing at all",
	}
	result := make([]map[string]string, 0, len(descriptions))
	for _, definition := range devicestate.DeviceFaultDefinitions() {
		result = append(result, map[string]string{
			"type": definition.Label, "description": descriptions[definition.Type],
		})
	}
	return result
}

func deviceFaultResponse(
	active map[string]map[devicestate.FaultType]int,
) map[string]map[string]int {
	result := make(map[string]map[string]int, len(active))
	for device, faults := range active {
		result[device] = make(map[string]int, len(faults))
		for faultType, value := range faults {
			result[device][faultType.Label()] = value
		}
	}
	return result
}

func deviceFaultTargetsResponse(
	targets []protocols.DeviceFaultTarget,
) []deviceFaultTargetResponse {
	result := make([]deviceFaultTargetResponse, 0, len(targets))
	for _, target := range targets {
		labels := make([]string, 0, len(target.Faults))
		for _, faultType := range target.Faults {
			labels = append(labels, faultType.Label())
		}
		result = append(result, deviceFaultTargetResponse{
			Device: target.Device, Address: target.Address, ErrorTypes: labels,
		})
	}
	return result
}

func parseInterfaceFaultType(value string) (devicestate.FaultType, error) {
	faultType, ok := devicestate.ParseFaultLabel(value)
	if !ok {
		return "", errInterfaceFaultTypeInvalid
	}
	return faultType, nil
}

func interfaceFaultResponse(
	active map[string]map[string]map[devicestate.FaultType]int,
) map[string]map[string]map[string]int {
	result := make(map[string]map[string]map[string]int, len(active))
	for deviceIP, interfaces := range active {
		result[deviceIP] = make(map[string]map[string]int, len(interfaces))
		for interfaceName, faults := range interfaces {
			result[deviceIP][interfaceName] = make(map[string]int, len(faults))
			for faultType, value := range faults {
				result[deviceIP][interfaceName][faultType.Label()] = value
			}
		}
	}
	return result
}

func interfaceFaultTargetsResponse(
	targets []protocols.InterfaceFaultTarget,
) []interfaceFaultTargetResponse {
	result := make([]interfaceFaultTargetResponse, 0, len(targets))
	for _, target := range targets {
		result = append(result, interfaceFaultTargetResponse{
			Device: target.Device, Address: target.Address, Interfaces: target.Interfaces,
		})
	}
	return result
}

func (req *errorInjectionRequest) validationMessage() string {
	switch {
	case req.Device == "":
		return "device is required"
	case req.ErrorType == "":
		return "errorType is required"
	// A device-scoped fault names no interface: the outage belongs to the
	// device's service, not to one of its ports.
	case req.Interface == "" && !req.deviceScoped():
		return "interface is required"
	case req.Interface != "" && req.deviceScoped():
		return "a device fault takes no interface"
	case req.Value < 0 || req.Value > 100:
		return "value must be between 0 and 100"
	default:
		return ""
	}
}

// deviceScoped reports whether the named error type belongs to the
// device-service catalog.
func (req *errorInjectionRequest) deviceScoped() bool {
	faultType, ok := devicestate.ParseFaultLabel(req.ErrorType)
	if !ok {
		return false
	}
	return slices.ContainsFunc(
		devicestate.DeviceFaultDefinitions(),
		func(definition devicestate.FaultDefinition) bool { return definition.Type == faultType },
	)
}
