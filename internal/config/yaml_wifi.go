package config

import "github.com/MustardSeedNetworks/niac-go/internal/converter"

// parseWiFiConfig parses the radios of an access point from YAML.
func parseWiFiConfig(yamlWiFi *converter.WifiConfig) *WiFiConfig {
	if yamlWiFi == nil {
		return nil
	}

	cfg := &WiFiConfig{Radios: make([]WiFiRadio, 0, len(yamlWiFi.Radios))}
	for _, radio := range yamlWiFi.Radios {
		cfg.Radios = append(cfg.Radios, WiFiRadio{
			Interface:  radio.Interface,
			SSID:       radio.SSID,
			BSSID:      radio.BSSID,
			Band:       radio.Band,
			Channel:    radio.Channel,
			TxPowerDBM: radio.TxPowerDBM,
			Clients:    parseWiFiClients(radio.Clients),
		})
	}

	return cfg
}

func parseWiFiClients(yamlClients []converter.WifiClient) []WiFiClient {
	if len(yamlClients) == 0 {
		return nil
	}

	clients := make([]WiFiClient, 0, len(yamlClients))
	for _, client := range yamlClients {
		clients = append(clients, WiFiClient(client))
	}

	return clients
}
