package api

import (
	"errors"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

type interfaceFaultTargetResponse struct {
	Device     string   `json:"device"`
	Address    string   `json:"address,omitempty"`
	Interfaces []string `json:"interfaces"`
}

var errInterfaceFaultTypeInvalid = errors.New("unsupported interface fault type")

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
	case req.Interface == "":
		return "interface is required"
	case req.ErrorType == "":
		return "errorType is required"
	case req.Value < 0 || req.Value > 100:
		return "value must be between 0 and 100"
	default:
		return ""
	}
}
