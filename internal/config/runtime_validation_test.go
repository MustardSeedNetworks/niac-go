package config

import (
	"errors"
	"testing"
)

func TestValidateRuntimeRequirementsSSHPassword(t *testing.T) {
	t.Setenv("NIAC_PRESENT_SSH_PASSWORD", "secret")
	t.Setenv("NIAC_EMPTY_SSH_PASSWORD", "")

	tests := []struct {
		name    string
		ssh     *SSHConfig
		wantErr bool
	}{
		{name: "disabled", ssh: &SSHConfig{Enabled: false}},
		{name: "available", ssh: &SSHConfig{
			Enabled: true, Username: "admin", PasswordEnv: "NIAC_PRESENT_SSH_PASSWORD",
		}},
		{name: "missing", ssh: &SSHConfig{
			Enabled: true, Username: "admin", PasswordEnv: "NIAC_MISSING_SSH_PASSWORD",
		}, wantErr: true},
		{name: "empty", ssh: &SSHConfig{
			Enabled: true, Username: "admin", PasswordEnv: "NIAC_EMPTY_SSH_PASSWORD",
		}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &Config{Devices: []Device{{Name: "edge-1", SSHConfig: test.ssh}}}
			err := ValidateRuntimeRequirements(cfg)
			if test.wantErr && !errors.Is(err, ErrSSHPasswordUnavailable) {
				t.Fatalf("ValidateRuntimeRequirements() error = %v, want SSH password error", err)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("ValidateRuntimeRequirements() error = %v", err)
			}
		})
	}
}

func TestValidateDeviceManagementRequirements(t *testing.T) {
	t.Setenv("NIAC_PRESENT_SSH_PASSWORD", "secret")

	tests := []struct {
		name    string
		device  Device
		wantErr bool
	}{
		{name: "valid", device: Device{SSHConfig: &SSHConfig{
			Enabled: true, Username: "admin", PasswordEnv: "NIAC_PRESENT_SSH_PASSWORD",
		}, SyslogConfig: &SyslogConfig{Enabled: true, Receivers: []string{"192.0.2.50:514"}}}},
		{name: "missing username", device: Device{SSHConfig: &SSHConfig{
			Enabled: true, PasswordEnv: "NIAC_PRESENT_SSH_PASSWORD",
		}}, wantErr: true},
		{name: "invalid receiver", device: Device{SyslogConfig: &SyslogConfig{
			Enabled: true, Receivers: []string{"192.0.2.50"},
		}}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateDeviceManagementRequirements(&test.device)
			if test.wantErr && err == nil {
				t.Fatal("ValidateDeviceManagementRequirements() error = nil")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("ValidateDeviceManagementRequirements() error = %v", err)
			}
		})
	}
}
