package api

import (
	"errors"
	"fmt"
	"net/netip"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// deviceFaultTargetResponse advertises which service outcomes one device can
// serve; a device with no DHCP or DNS server offers only the outcomes that
// need no service of their own, and one with no address at all is absent.
type deviceFaultTargetResponse struct {
	Device     string   `json:"device"`
	Address    string   `json:"address,omitempty"`
	ErrorTypes []string `json:"errorTypes"`
}

type interfaceFaultTargetResponse struct {
	Device     string              `json:"device"`
	Address    string              `json:"address,omitempty"`
	Interfaces []string            `json:"interfaces"`
	ErrorTypes map[string][]string `json:"errorTypes"`
}

var errInterfaceFaultTypeInvalid = errors.New("unsupported fault type")

const duplicateIPLabel = "Duplicate IP"

const badMaskLabel = "Bad Subnet Mask"

func availableErrorTypes() []map[string]string {
	descriptions := map[devicestate.FaultType]string{
		devicestate.FaultFCS:         "Frame Check Sequence errors (0-100)",
		devicestate.FaultDiscards:    "Dropped packets (0-100)",
		devicestate.FaultInterface:   "Generic interface errors (0-100)",
		devicestate.FaultUtilization: "Interface bandwidth saturation (0-100%)",
		devicestate.FaultLinkDown:    "Drop the link (non-zero takes the interface down)",
		devicestate.FaultPoELoss:     "Cut the power this port supplies (non-zero faults the PSE port)",
	}
	result := make([]map[string]string, 0, len(descriptions))
	for _, definition := range devicestate.InterfaceFaultDefinitions() {
		result = append(result, map[string]string{
			"type": definition.Label, "description": descriptions[definition.Type], "valueKind": "number",
		})
	}
	result = append(result, map[string]string{
		"type":        duplicateIPLabel,
		"description": "Answer ARP for an IPv4 address owned by a peer on the selected segment",
		"valueKind":   "address",
	})
	result = append(result, map[string]string{
		"type":        badMaskLabel,
		"description": "Change the IPv4 host mask without changing its canonical address",
		"valueKind":   "prefix",
	})
	return result
}

type deviceFaultTypeResponse struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	MaxValue    *int   `json:"maxValue,omitempty"`
	ValueKind   string `json:"valueKind"`
}

// availableDeviceErrorTypes describes the device-scoped catalog: service
// outcomes that change what a device answers rather than what its interface
// counters report.
func availableDeviceErrorTypes() []deviceFaultTypeResponse {
	descriptions := map[devicestate.DeviceFaultType]string{
		devicestate.FaultDuplicateDHCPOffer: "Offer an IPv4 address already owned by a peer on the DHCP server's network",
		devicestate.FaultDHCPNoOffer:        "DHCP server consumes the Discover and sends no Offer",
		devicestate.FaultDNSNXDomain:        "DNS server answers every query with NXDOMAIN",
		devicestate.FaultDNSTimeout:         "DNS server answers nothing at all",
		devicestate.FaultLatency:            "Delay every ICMP echo reply (0-60000 ms)",
		devicestate.FaultCPUPercent:         "Set processor utilization (1-100%; zero clears)",
		devicestate.FaultMemoryPercent:      "Set used memory as a percentage of capacity (1-100%; zero clears)",
		devicestate.FaultDiskPercent:        "Set used disk storage as a percentage of capacity (1-100%; zero clears)",
		devicestate.FaultCaptivePortal:      "Redirect HTTP requests to a local portal (1 enables; zero clears)",
	}
	result := make([]deviceFaultTypeResponse, 0, len(descriptions))
	for _, definition := range devicestate.DeviceFaultDefinitions() {
		result = append(result, deviceFaultTypeResponse{
			Type: definition.Label, Description: descriptions[definition.Type],
			MaxValue: new(definition.MaxValue), ValueKind: deviceFaultValueKind(definition.Type),
		})
	}
	result = append(result, deviceFaultTypeResponse{
		Type:        devicestate.FaultDuplicateDHCPOffer.Label(),
		Description: descriptions[devicestate.FaultDuplicateDHCPOffer],
		ValueKind:   deviceFaultValueKind(devicestate.FaultDuplicateDHCPOffer),
	})
	return result
}

func deviceFaultValueKind(kind devicestate.DeviceFaultType) string {
	switch kind {
	case devicestate.FaultDuplicateDHCPOffer:
		return "address"
	case devicestate.FaultLatency:
		return "milliseconds"
	case devicestate.FaultCPUPercent, devicestate.FaultMemoryPercent, devicestate.FaultDiskPercent:
		return "percent"
	case devicestate.FaultDHCPNoOffer, devicestate.FaultDNSNXDomain,
		devicestate.FaultDNSTimeout, devicestate.FaultCaptivePortal:
		return "toggle"
	}
	return ""
}

func deviceFaultResponse(
	active map[string]map[devicestate.DeviceFaultType]devicestate.DeviceFault,
) map[string]map[string]deviceFaultPayload {
	result := make(map[string]map[string]deviceFaultPayload, len(active))
	for device, faults := range active {
		result[device] = make(map[string]deviceFaultPayload, len(faults))
		for faultType, fault := range faults {
			payload := deviceFaultPayload{}
			if faultType == devicestate.FaultDuplicateDHCPOffer {
				payload.Address = new(fault.Address.String())
			} else {
				payload.Value = new(fault.Value)
			}
			result[device][faultType.Label()] = payload
		}
	}
	return result
}

type deviceFaultPayload struct {
	Value      *int    `json:"value,omitempty"`
	Address    *string `json:"address,omitempty"`
	PrefixBits *int    `json:"prefixBits,omitempty"`
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
	addressed map[string][]devicestate.InterfaceAddressFault,
	prefixes map[string][]devicestate.InterfacePrefixFault,
) map[string]map[string]map[string]deviceFaultPayload {
	result := make(map[string]map[string]map[string]deviceFaultPayload, len(active))
	for deviceIP, interfaces := range active {
		result[deviceIP] = make(map[string]map[string]deviceFaultPayload, len(interfaces))
		for interfaceName, faults := range interfaces {
			result[deviceIP][interfaceName] = make(map[string]deviceFaultPayload, len(faults))
			for faultType, value := range faults {
				result[deviceIP][interfaceName][faultType.Label()] = deviceFaultPayload{Value: new(value)}
			}
		}
	}
	for device, faults := range addressed {
		if result[device] == nil {
			result[device] = make(map[string]map[string]deviceFaultPayload)
		}
		for _, fault := range faults {
			if result[device][fault.Interface] == nil {
				result[device][fault.Interface] = make(map[string]deviceFaultPayload)
			}
			result[device][fault.Interface][duplicateIPLabel] = deviceFaultPayload{Address: new(fault.Address.String())}
		}
	}
	for device, faults := range prefixes {
		appendPrefixFaultResponse(result, device, faults)
	}
	return result
}

func appendPrefixFaultResponse(
	result map[string]map[string]map[string]deviceFaultPayload,
	device string,
	faults []devicestate.InterfacePrefixFault,
) {
	if result[device] == nil {
		result[device] = make(map[string]map[string]deviceFaultPayload)
	}
	for _, fault := range faults {
		if result[device][fault.Interface] == nil {
			result[device][fault.Interface] = make(map[string]deviceFaultPayload)
		}
		result[device][fault.Interface][badMaskLabel] = deviceFaultPayload{PrefixBits: new(fault.PrefixBits)}
	}
}

func interfaceFaultTargetsResponse(
	targets []protocols.InterfaceFaultTarget,
) []interfaceFaultTargetResponse {
	result := make([]interfaceFaultTargetResponse, 0, len(targets))
	for _, target := range targets {
		result = append(result, interfaceFaultTargetResponse{
			Device:     target.Device,
			Address:    target.Address,
			Interfaces: target.Interfaces,
			ErrorTypes: target.ErrorTypes,
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
	default:
		return req.payloadValidationMessage()
	}
}

func (req *errorInjectionRequest) payloadValidationMessage() string {
	if req.ErrorType == badMaskLabel {
		return validateMaskPayload(req.PrefixBits, req.Value, req.Address)
	}
	if req.PrefixBits != nil {
		return "prefixBits requires a mask fault"
	}
	kind, _ := devicestate.ParseDeviceFaultLabel(req.ErrorType)
	addressed := kind == devicestate.FaultDuplicateDHCPOffer || req.ErrorType == duplicateIPLabel
	if message := validateFaultPayload(addressed, req.Value, req.Address); message != "" {
		return message
	}
	if addressed {
		return ""
	}
	if *req.Value < 0 || *req.Value > errorInjectionMaximum(req.ErrorType) {
		return fmt.Sprintf("value must be between 0 and %d", errorInjectionMaximum(req.ErrorType))
	}
	return ""
}

func validateMaskPayload(prefix, value *int, address *string) string {
	if prefix == nil || value != nil || address != nil {
		return "mask fault requires prefixBits and no value or address"
	}
	if *prefix < 0 || *prefix > 32 {
		return "prefixBits must be between 0 and 32"
	}
	return ""
}

func validateFaultPayload(addressed bool, value *int, addressText *string) string {
	if !addressed {
		if addressText != nil || value == nil {
			return "numeric fault requires value and no address"
		}
		return ""
	}
	if addressText == nil || value != nil {
		return "address fault requires address and no value"
	}
	address, err := netip.ParseAddr(*addressText)
	if err != nil || !devicestate.ValidFaultAddress(address) {
		return "address must be a unicast IPv4 address"
	}
	return ""
}

func errorInjectionMaximum(label string) int {
	for _, definition := range devicestate.DeviceFaultDefinitions() {
		if definition.Label == label {
			return definition.MaxValue
		}
	}
	return defaultErrorInjectionMaximum
}

const defaultErrorInjectionMaximum = 100

// deviceScoped reports whether the named error type belongs to the
// device-service catalog.
func (req *errorInjectionRequest) deviceScoped() bool {
	_, ok := devicestate.ParseDeviceFaultLabel(req.ErrorType)
	return ok
}

func parseDeviceFaultType(value string) (devicestate.DeviceFaultType, error) {
	faultType, ok := devicestate.ParseDeviceFaultLabel(value)
	if !ok {
		return "", errInterfaceFaultTypeInvalid
	}
	return faultType, nil
}
