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
	if fault.Type == "duplicate_dhcp_offer" {
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
