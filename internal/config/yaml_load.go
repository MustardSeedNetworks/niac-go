package config

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
	"github.com/MustardSeedNetworks/niac-go/internal/oui"
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
	var registry *oui.Registry
	if usesVendorIdentity(yamlConfig) {
		var err error
		registry, err = oui.LoadEmbedded()
		if err != nil {
			return nil, fmt.Errorf("load IEEE OUI registry: %w", err)
		}
	}
	cfg := createBaseConfig(yamlConfig)
	cfg.IncludePath = resolveIncludePath(configDir, cfg.IncludePath)

	for _, yamlDevice := range yamlConfig.Devices {
		device, deviceErr := convertYAMLDevice(yamlDevice, cfg.IncludePath, registry)
		if deviceErr != nil {
			return nil, deviceErr
		}

		cfg.Devices = append(cfg.Devices, device)
	}

	segments, err := buildSegments(yamlConfig, cfg.IncludePath, configDir, registry)
	if err != nil {
		return nil, err
	}

	cfg.Segments = segments

	// A config must describe at least one device somewhere — either flat, or
	// inline in a segment. (Segments that only reference a `config:` file are
	// resolved later by the loader, so they legitimately have no inline devices.)
	if len(cfg.Devices) == 0 && !configHasDevices(cfg.Segments) {
		return nil, ErrNoDevicesDefined
	}

	return cfg, nil
}

func usesVendorIdentity(cfg *converter.Config) bool {
	return slices.ContainsFunc(cfg.Devices, func(device converter.Device) bool {
		return strings.TrimSpace(device.Vendor) != ""
	}) || slices.ContainsFunc(cfg.Segments, func(segment converter.Segment) bool {
		return slices.ContainsFunc(segment.Devices, func(device converter.Device) bool {
			return strings.TrimSpace(device.Vendor) != ""
		})
	})
}

// buildSegments converts and validates the multi-VLAN segment bindings (ADR
// 0008). Segments and a top-level device list are mutually exclusive.
func buildSegments(
	yamlConfig *converter.Config,
	includePath, configDir string,
	registry *oui.Registry,
) ([]Segment, error) {
	if len(yamlConfig.Segments) == 0 {
		return nil, nil
	}

	if len(yamlConfig.Devices) > 0 {
		return nil, ErrSegmentsAndTopLevelDevices
	}

	segments := make([]Segment, 0, len(yamlConfig.Segments))

	for _, ySeg := range yamlConfig.Segments {
		seg, err := buildSegment(ySeg, includePath, configDir, registry)
		if err != nil {
			return nil, err
		}

		segments = append(segments, seg)
	}

	return segments, nil
}

func buildSegment(
	ySeg converter.Segment,
	includePath, configDir string,
	registry *oui.Registry,
) (Segment, error) {
	tag, err := parseSegmentTag(string(ySeg.Tag))
	if err != nil {
		return Segment{}, err
	}

	hasInline := len(ySeg.Devices) > 0
	hasConfig := ySeg.Config != ""

	if hasInline == hasConfig {
		return Segment{}, fmt.Errorf("%w: tag %q", ErrSegmentDevicesXORConfig, ySeg.Tag)
	}

	seg := Segment{Tag: tag, ConfigPath: ySeg.Config}

	if hasConfig {
		return resolveSegmentConfig(seg, ySeg.Config, configDir)
	}

	for _, yamlDevice := range ySeg.Devices {
		device, convErr := convertYAMLDevice(yamlDevice, includePath, registry)
		if convErr != nil {
			return Segment{}, convErr
		}

		seg.Devices = append(seg.Devices, device)
	}

	return seg, nil
}

// resolveSegmentConfig loads a segment's `config:` file (a whole demo) and uses
// its devices as the segment's device set. The path is relative to the parent
// config's directory. A segment config that itself declares segments is
// rejected — nesting demos is out of scope.
func resolveSegmentConfig(seg Segment, path, configDir string) (Segment, error) {
	if !filepath.IsAbs(path) && configDir != "" {
		path = filepath.Join(configDir, path)
	}

	loaded, err := LoadYAML(path)
	if err != nil {
		return Segment{}, fmt.Errorf("segment tag %d config %q: %w", seg.Tag, path, err)
	}

	if len(loaded.Segments) > 0 {
		return Segment{}, fmt.Errorf(
			"%w: segment tag %d config %q declares its own segments",
			ErrInvalidSegmentTag,
			seg.Tag,
			path,
		)
	}

	seg.Devices = loaded.Devices

	return seg, nil
}

// parseSegmentTag maps a segment tag string to a VLAN id: "untagged" -> 0, else
// a decimal VLAN id in 1..4094.
func parseSegmentTag(tag string) (int, error) {
	if strings.EqualFold(tag, "untagged") {
		return UntaggedTag, nil
	}

	vlan, err := strconv.Atoi(tag)
	if err != nil || vlan < minVLANID || vlan > maxVLANID {
		return 0, fmt.Errorf("%w: %q (want \"untagged\" or 1..4094)", ErrInvalidSegmentTag, tag)
	}

	return vlan, nil
}

func configHasDevices(segments []Segment) bool {
	return slices.ContainsFunc(segments, func(seg Segment) bool {
		return len(seg.Devices) > 0 || seg.ConfigPath != ""
	})
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
		Networks:    convertNetworks(yamlConfig.Networks),
		Attachments: convertLogicalAttachments(yamlConfig.Attachments),
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
