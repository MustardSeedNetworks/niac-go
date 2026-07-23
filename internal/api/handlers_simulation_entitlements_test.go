package api

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestFreeTierDeviceCountMatchesProductContract(t *testing.T) {
	if FreeTierDeviceCount != 10 {
		t.Fatalf("FreeTierDeviceCount = %d, want 10", FreeTierDeviceCount)
	}
}

func TestValidateConfigEntitlementsDeviceLimits(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *config.Config
		entitlements SimulationEntitlements
		wantErr      error
	}{
		{name: "flat Free 10", cfg: flatConfig(10)},
		{
			name: "flat Free 11", cfg: flatConfig(11),
			wantErr: ErrUnlimitedDevicesLicenseRequired,
		},
		{
			name: "flat Pro 11", cfg: flatConfig(11),
			entitlements: SimulationEntitlements{UnlimitedDevices: true},
		},
		{
			name: "flat Pro 1001", cfg: flatConfig(1001),
			entitlements: SimulationEntitlements{UnlimitedDevices: true},
			wantErr:      ErrSimulationDeviceLimitExceeded,
		},
		{name: "segmented Free 10", cfg: segmentedConfig(4, 6)},
		{
			name: "segmented Free 11", cfg: segmentedConfig(5, 6),
			wantErr: ErrUnlimitedDevicesLicenseRequired,
		},
		{
			name: "segmented Pro 11", cfg: segmentedConfig(5, 6),
			entitlements: SimulationEntitlements{UnlimitedDevices: true},
		},
		{
			name: "segmented Pro 1001", cfg: segmentedConfig(500, 501),
			entitlements: SimulationEntitlements{UnlimitedDevices: true},
			wantErr:      ErrSimulationDeviceLimitExceeded,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateConfigEntitlements(test.cfg, test.entitlements)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("ValidateConfigEntitlements() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func flatConfig(count int) *config.Config {
	return &config.Config{Devices: make([]config.Device, count)}
}

func segmentedConfig(counts ...int) *config.Config {
	cfg := &config.Config{Segments: make([]config.Segment, len(counts))}
	for index, count := range counts {
		cfg.Segments[index] = config.Segment{Tag: index + 1, Devices: make([]config.Device, count)}
	}
	return cfg
}
