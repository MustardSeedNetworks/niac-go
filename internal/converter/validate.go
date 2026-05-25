package converter

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ErrConfigInvalid is returned when struct-tag validation rejects a Config.
// Use errors.Is(err, ErrConfigInvalid) to distinguish validator failures
// from the sentinel errors (ErrDeviceMissingMAC etc.) raised by the manual
// ValidateConfig pre-checks.
var ErrConfigInvalid = errors.New("config validation failed")

// configValidator is the package-level validator. Initialised once at
// package load; immutable thereafter (validator.Validate is goroutine-safe
// once registration is complete). Exempted from gochecknoglobals via
// .golangci.yml — same pattern as other registry singletons in the repo.
var configValidator = newConfigValidator()

func newConfigValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())

	// Use yaml struct-tag names in error namespaces so messages read
	// `devices[0].mac` instead of `Devices[0].MAC`.
	v.RegisterTagNameFunc(yamlFieldName)

	v.RegisterStructValidation(deviceStructValidation, Device{})

	return v
}

func yamlFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("yaml")
	if tag == "" || tag == "-" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}

// deviceStructValidation enforces cross-field rules that single-field tags
// can't express. Today: Device.IP and Device.IPs are mutually exclusive.
func deviceStructValidation(sl validator.StructLevel) {
	d, ok := sl.Current().Interface().(Device)
	if !ok {
		return
	}
	if d.IP != "" && len(d.IPs) > 0 {
		sl.ReportError(d.IPs, "ips", "IPs", "excluded_with_ip", "")
	}
}

func validateConfigStruct(cfg *Config) error {
	err := configValidator.Struct(cfg)
	if err == nil {
		return nil
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return fmt.Errorf("%w: %w", ErrConfigInvalid, err)
	}
	msgs := make([]string, 0, len(verrs))
	for _, fe := range verrs {
		msgs = append(msgs, formatFieldError(fe))
	}
	return fmt.Errorf("%w:\n  - %s", ErrConfigInvalid, strings.Join(msgs, "\n  - "))
}

func formatFieldError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Namespace())
	case "ip":
		return fmt.Sprintf("%s: %q is not a valid IP address", fe.Namespace(), fe.Value())
	case "mac":
		return fmt.Sprintf("%s: %q is not a valid MAC address", fe.Namespace(), fe.Value())
	case "oneof":
		return fmt.Sprintf("%s: %q is not one of [%s]", fe.Namespace(), fe.Value(), fe.Param())
	case "gte", "lte", "gt", "lt":
		return fmt.Sprintf("%s: %v fails rule %s=%s", fe.Namespace(), fe.Value(), fe.Tag(), fe.Param())
	case "excluded_with_ip":
		return fmt.Sprintf("%s: cannot be set together with `ip` (use one or the other)", fe.Namespace())
	default:
		return fmt.Sprintf("%s: failed rule %q", fe.Namespace(), fe.Tag())
	}
}
