package config

import (
	"errors"
	"fmt"
	"os"
)

// ErrSSHPasswordUnavailable indicates that an enabled SSH service cannot load its password.
var ErrSSHPasswordUnavailable = errors.New("SSH password environment variable is not set")

// ValidateRuntimeRequirements checks external prerequisites needed to start a configuration.
func ValidateRuntimeRequirements(cfg *Config) error {
	for _, segment := range cfg.NormalizedSegments() {
		for index := range segment.Devices {
			device := &segment.Devices[index]
			if device.SSHConfig == nil || !device.SSHConfig.Enabled {
				continue
			}
			if password, found := os.LookupEnv(device.SSHConfig.PasswordEnv); !found || password == "" {
				return fmt.Errorf(
					"%w: device %q requires %q",
					ErrSSHPasswordUnavailable,
					device.Name,
					device.SSHConfig.PasswordEnv,
				)
			}
		}
	}
	return nil
}

// ValidateDeviceManagementRequirements checks management syntax and external prerequisites.
func ValidateDeviceManagementRequirements(device *Device) error {
	validator := NewValidator("")
	validator.validateSSH(device, "device")
	validator.validateSyslog(device, "device")
	if validation := validator.errors; validation.HasErrors() {
		return validation
	}
	return ValidateRuntimeRequirements(&Config{Devices: []Device{*device}})
}
