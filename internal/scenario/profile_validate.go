package scenario

import (
	"fmt"
	"regexp"
	"strings"
)

var profileRoleRE = regexp.MustCompile(`^[a-z][a-z0-9-]{1,47}$`)

const (
	maxProfileSpeed = 400_000_000_000
	maxProfileText  = 256
	maxVendorText   = 64
	maxModelText    = 128
	maxInterfaces   = 4096
	maxInterfaceMTU = 1_000_000
)

// ValidateCustomProfile checks fields that enter the authoring catalog.
func ValidateCustomProfile(profile DeviceProfile) error {
	if err := validateProfileIdentity(profile); err != nil {
		return err
	}
	if err := validateProfileInterfaces(profile); err != nil {
		return err
	}
	if !validProfileWalk(profile.WalkName) {
		return fmt.Errorf("%w: walk name must reference a captured walk", ErrInvalidProfile)
	}
	if profile.Source != customSource {
		return fmt.Errorf("%w: source must be captured", ErrInvalidProfile)
	}
	return nil
}

func validateProfileIdentity(profile DeviceProfile) error {
	if !profileRoleRE.MatchString(profile.Role) {
		return fmt.Errorf("%w: role must be 2-48 lowercase letters, numbers, or hyphens", ErrInvalidProfile)
	}
	if !validProfileDeviceType(profile.DeviceType) || !validProfileText(profile.Vendor, maxVendorText) ||
		!validProfileText(profile.Model, maxModelText) || !validProfileText(profile.Platform, maxProfileText) ||
		len(profile.Software) > maxProfileText || len(profile.SysObjectID) > maxProfileText {
		return fmt.Errorf("%w: required profile fields are missing or too long", ErrInvalidProfile)
	}
	return nil
}

func validateProfileInterfaces(profile DeviceProfile) error {
	if profile.InterfaceCount < 0 || profile.InterfaceCount > maxInterfaces {
		return fmt.Errorf("%w: interface count must be between 0 and 4096", ErrInvalidProfile)
	}
	if len(profile.Interfaces) > maxInterfaces ||
		(len(profile.Interfaces) > 0 && len(profile.Interfaces) != profile.InterfaceCount) {
		return fmt.Errorf("%w: interface inventory does not match interface count", ErrInvalidProfile)
	}
	for _, iface := range profile.Interfaces {
		if !validProfileInterface(iface) {
			return fmt.Errorf("%w: interface inventory contains invalid data", ErrInvalidProfile)
		}
	}
	return nil
}

func validProfileInterface(iface ProfileInterface) bool {
	return validProfileText(iface.Name, maxModelText) && validProfileInterfaceType(iface.Type) &&
		iface.MTU >= 0 && iface.MTU <= maxInterfaceMTU && iface.Speed >= 0 && iface.Speed <= maxProfileSpeed
}

func validProfileInterfaceType(value string) bool {
	switch value {
	case "ethernet", "l2vlan", "l3ipvlan", "loopback", "tunnel", "other":
		return true
	default:
		return false
	}
}

func validProfileWalk(name string) bool {
	return strings.HasPrefix(name, "captured/") && strings.HasSuffix(name, ".walk") &&
		!strings.Contains(name, "..") && !strings.Contains(name, `\`)
}

func validProfileDeviceType(deviceType string) bool {
	switch deviceType {
	case "switch", "router", "firewall", "access-point", "host", "server", "voip-phone", "printer":
		return true
	default:
		return false
	}
}

func validProfileText(value string, maxLength int) bool {
	return value != "" && strings.TrimSpace(value) == value && len(value) <= maxLength
}
