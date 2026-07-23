package api

import (
	"errors"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

var errInterfaceFaultTypeInvalid = errors.New("unsupported interface fault type")

func availableErrorTypes() []map[string]string {
	return []map[string]string{
		{"type": "FCS Errors", "description": "Frame Check Sequence errors (0-100)"},
		{"type": "Packet Discards", "description": "Dropped packets (0-100)"},
		{"type": "Interface Errors", "description": "Generic interface errors (0-100)"},
		{"type": "High Utilization", "description": "Interface bandwidth saturation (0-100%)"},
	}
}

func parseInterfaceFaultType(value string) (devicestate.FaultType, error) {
	switch value {
	case "FCS Errors":
		return devicestate.FaultFCS, nil
	case "Packet Discards":
		return devicestate.FaultDiscards, nil
	case "Interface Errors":
		return devicestate.FaultInterface, nil
	case "High Utilization":
		return devicestate.FaultUtilization, nil
	default:
		return "", errInterfaceFaultTypeInvalid
	}
}

func (req errorInjectionRequest) validationMessage() string {
	switch {
	case req.DeviceIP == "":
		return "device_ip is required"
	case req.Interface == "":
		return "interface is required"
	case req.ErrorType == "":
		return "error_type is required"
	case req.Value < 0 || req.Value > 100:
		return "value must be between 0 and 100"
	default:
		return ""
	}
}
