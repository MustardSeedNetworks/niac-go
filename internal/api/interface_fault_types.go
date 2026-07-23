package api

import (
	"errors"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

var errInterfaceFaultTypeInvalid = errors.New("unsupported interface fault type")

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
