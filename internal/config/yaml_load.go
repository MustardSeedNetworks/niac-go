package config

import (
	"fmt"
	"path/filepath"

	"github.com/krisarmstrong/niac-go/internal/converter"
)

func LoadYAML(filename string) (*Config, error) {
	yamlConfig, err := loadYAMLFile(filename)
	if err != nil {
		return nil, err
	}

	return buildConfigFromYAML(yamlConfig, filepath.Dir(filepath.Clean(filename)))
}

// LoadYAMLBytes builds a runtime config from in-memory YAML data.
func LoadYAMLBytes(data []byte) (*Config, error) {
	yamlConfig, err := loadYAMLBytes(data)
	if err != nil {
		return nil, err
	}

	return buildConfigFromYAML(yamlConfig, "")
}

// loadYAMLFile loads and validates a YAML configuration file.
func loadYAMLFile(filename string) (*converter.Config, error) {
	yamlConfig, err := converter.LoadYAMLConfig(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load YAML config: %w", err)
	}

	return validateYAMLConfig(yamlConfig)
}

func loadYAMLBytes(data []byte) (*converter.Config, error) {
	yamlConfig, err := converter.LoadYAMLConfigFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return validateYAMLConfig(yamlConfig)
}

func validateYAMLConfig(yamlConfig *converter.Config) (*converter.Config, error) {
	err := converter.ValidateConfig(yamlConfig)
	if err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return yamlConfig, nil
}

func buildConfigFromYAML(yamlConfig *converter.Config, configDir string) (*Config, error) {
	cfg := createBaseConfig(yamlConfig)
	cfg.IncludePath = resolveIncludePath(configDir, cfg.IncludePath)

	for _, yamlDevice := range yamlConfig.Devices {
		device, err := convertYAMLDevice(yamlDevice, cfg.IncludePath)
		if err != nil {
			return nil, err
		}

		cfg.Devices = append(cfg.Devices, device)
	}

	if len(cfg.Devices) == 0 {
		return nil, ErrNoDevicesDefined
	}

	return cfg, nil
}

func resolveIncludePath(configDir, includePath string) string {
	switch {
	case includePath == "":
		return configDir
	case filepath.IsAbs(includePath) || configDir == "":
		return includePath
	default:
		return filepath.Join(configDir, includePath)
	}
}

// createBaseConfig creates the base configuration with global settings.
func createBaseConfig(yamlConfig *converter.Config) *Config {
	cfg := &Config{
		Devices:     make([]Device, 0, len(yamlConfig.Devices)),
		IncludePath: yamlConfig.IncludePath,
	}

	// Copy CapturePlayback if present (use first one from array for now)
	if len(yamlConfig.CapturePlaybacks) > 0 {
		cfg.CapturePlayback = &CapturePlayback{
			FileName:  yamlConfig.CapturePlaybacks[0].FileName,
			LoopTime:  yamlConfig.CapturePlaybacks[0].LoopTime,
			ScaleTime: yamlConfig.CapturePlaybacks[0].ScaleTime,
		}
	}

	// Copy DiscoveryProtocols if present
	if yamlConfig.DiscoveryProtocols != nil {
		cfg.DiscoveryProtocols = &DiscoveryProtocols{}

		if yamlConfig.DiscoveryProtocols.LLDP != nil {
			cfg.DiscoveryProtocols.LLDP = &ProtocolConfig{
				Enabled:  yamlConfig.DiscoveryProtocols.LLDP.Enabled,
				Interval: yamlConfig.DiscoveryProtocols.LLDP.Interval,
			}
		}

		if yamlConfig.DiscoveryProtocols.CDP != nil {
			cfg.DiscoveryProtocols.CDP = &ProtocolConfig{
				Enabled:  yamlConfig.DiscoveryProtocols.CDP.Enabled,
				Interval: yamlConfig.DiscoveryProtocols.CDP.Interval,
			}
		}

		if yamlConfig.DiscoveryProtocols.EDP != nil {
			cfg.DiscoveryProtocols.EDP = &ProtocolConfig{
				Enabled:  yamlConfig.DiscoveryProtocols.EDP.Enabled,
				Interval: yamlConfig.DiscoveryProtocols.EDP.Interval,
			}
		}

		if yamlConfig.DiscoveryProtocols.FDP != nil {
			cfg.DiscoveryProtocols.FDP = &ProtocolConfig{
				Enabled:  yamlConfig.DiscoveryProtocols.FDP.Enabled,
				Interval: yamlConfig.DiscoveryProtocols.FDP.Interval,
			}
		}
	}

	return cfg
}
