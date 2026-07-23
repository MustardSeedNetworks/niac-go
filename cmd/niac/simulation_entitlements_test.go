package main

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

type testFeatureChecker map[string]bool

func (f testFeatureChecker) HasFeature(feature string) bool { return f[feature] }

func TestValidateSimulationConfigRequiresRoutedLabsForSSH(t *testing.T) {
	t.Setenv("NIAC_TEST_SSH_PASSWORD", "secret")
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1", SSHConfig: &config.SSHConfig{
			Enabled: true, Username: "admin", PasswordEnv: "NIAC_TEST_SSH_PASSWORD",
		},
	}}}

	err := validateSimulationConfig(cfg, testFeatureChecker{})
	if !errors.Is(err, api.ErrRoutedLabsLicenseRequired) {
		t.Fatalf("validateSimulationConfig() error = %v, want routed-labs requirement", err)
	}
	if err = validateSimulationConfig(cfg, testFeatureChecker{"routed_labs": true}); err != nil {
		t.Fatalf("validateSimulationConfig() with grant error = %v", err)
	}
}

func TestValidateSimulationConfigRejectsMissingSSHPassword(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1", SSHConfig: &config.SSHConfig{
			Enabled: true, Username: "admin", PasswordEnv: "NIAC_MISSING_SSH_PASSWORD",
		},
	}}}

	err := validateSimulationConfig(cfg, testFeatureChecker{"routed_labs": true})
	if !errors.Is(err, config.ErrSSHPasswordUnavailable) {
		t.Fatalf("validateSimulationConfig() error = %v, want SSH password requirement", err)
	}
}

func TestValidateStoredSimulationConfigUsesCurrentLicenseState(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1", SSHConfig: &config.SSHConfig{Enabled: true},
	}}}
	loads := 0
	err := validateStoredSimulationConfig(cfg, func() (featureChecker, error) {
		loads++
		return testFeatureChecker{}, nil
	})
	if !errors.Is(err, api.ErrRoutedLabsLicenseRequired) {
		t.Fatalf("validateStoredSimulationConfig() error = %v, want routed-labs requirement", err)
	}
	if loads != 1 {
		t.Fatalf("license state loads = %d, want 1", loads)
	}
}
