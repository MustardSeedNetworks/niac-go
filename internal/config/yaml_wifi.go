package config

import "github.com/MustardSeedNetworks/niac-go/internal/converter"

// parseWiFiConfig parses the radios of an access point from YAML.
func parseWiFiConfig(yamlWiFi *converter.WifiConfig) *WiFiConfig {
	if yamlWiFi == nil {
		return nil
	}

	cfg := &WiFiConfig{Radios: make([]WiFiRadio, 0, len(yamlWiFi.Radios))}
	for _, radio := range yamlWiFi.Radios {
		cfg.Radios = append(cfg.Radios, WiFiRadio(radio))
	}

	return cfg
}
