package converter

import (
	"net/netip"
	"reflect"

	"github.com/go-playground/validator/v10"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func validateBehaviorFaultPayload(validation validator.StructLevel) {
	fault, ok := reflect.TypeAssert[BehaviorFault](validation.Current())
	if !ok {
		return
	}
	if fault.Type == "bad_mask" {
		if fault.PrefixBits == nil {
			validation.ReportError(fault.PrefixBits, "prefix_bits", "PrefixBits", "required", "")
		}
		if fault.Value != nil {
			validation.ReportError(fault.Value, "value", "Value", "excluded", "")
		}
		if fault.Address != nil {
			validation.ReportError(fault.Address, "address", "Address", "excluded", "")
		}
		return
	}
	if fault.PrefixBits != nil {
		validation.ReportError(fault.PrefixBits, "prefix_bits", "PrefixBits", "excluded", "")
	}
	if fault.Type == "duplicate_dhcp_offer" || fault.Type == "duplicate_ip" {
		if fault.Value != nil {
			validation.ReportError(fault.Value, "value", "Value", "excluded", "")
		}
		if fault.Address == nil {
			validation.ReportError(fault.Address, "address", "Address", "required", "")
		} else if address, err := netip.ParseAddr(*fault.Address); err != nil || !devicestate.ValidFaultAddress(address) {
			validation.ReportError(fault.Address, "address", "Address", "ipv4", "")
		}
		return
	}
	if fault.Value == nil {
		validation.ReportError(fault.Value, "value", "Value", "required", "")
	}
	if fault.Address != nil {
		validation.ReportError(fault.Address, "address", "Address", "excluded", "")
	}
}

// validateAuthoredInterfaceFault enforces the one thing the type tags cannot:
// link_down and poe_loss have no magnitude, so authoring a value for either
// would invent one. Every other interface fault is a rate and needs one.
func validateAuthoredInterfaceFault(validation validator.StructLevel) {
	fault, ok := reflect.TypeAssert[InterfaceFault](validation.Current())
	if !ok {
		return
	}
	if fault.Type == string(devicestate.FaultLinkDown) || fault.Type == string(devicestate.FaultPoELoss) {
		if fault.Value != nil {
			validation.ReportError(fault.Value, "value", "Value", "excluded", "")
		}

		return
	}
	if fault.Value == nil {
		validation.ReportError(fault.Value, "value", "Value", "required", "")
	}
}
