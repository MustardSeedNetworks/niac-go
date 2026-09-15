package config

import (
	"strings"
	"testing"
)

const wifiScenarioYAML = `devices:
  - name: MED-AP-01
    type: ap
    mac: "00:0c:ce:88:23:c0"
    ips: ["10.20.220.11"]
    interfaces:
      - name: Dot11Radio0
        type: ieee80211
      - name: Dot11Radio1
        type: ieee80211
    wifi:
      radios:
        - interface: Dot11Radio0
          ssid: corp-wifi
          bssid: "00:0c:ce:88:23:c7"
          band: 2.4GHz
          channel: 6
          tx_power_dbm: 17
        - interface: Dot11Radio1
          ssid: corp-wifi
          bssid: "00:0c:ce:88:23:c8"
          band: 5GHz
          channel: 149
          tx_power_dbm: 20
`

// TestWiFiRadiosSurviveASaveAndLoad: the editor loads a scenario and writes it
// back, so a radio the marshaller drops is a radio the author loses by opening
// the file.
func TestWiFiRadiosSurviveASaveAndLoad(t *testing.T) {
	loaded, err := LoadYAMLBytes([]byte(wifiScenarioYAML))
	if err != nil {
		t.Fatalf("LoadYAMLBytes() = %v", err)
	}
	assertAuthoredRadios(t, loaded)

	saved, err := MarshalConfigYAML(loaded)
	if err != nil {
		t.Fatalf("MarshalConfigYAML() = %v", err)
	}
	if !strings.Contains(string(saved), "tx_power_dbm: 20") {
		t.Fatalf("radios lost on save:\n%s", saved)
	}

	reloaded, err := LoadYAMLBytes(saved)
	if err != nil {
		t.Fatalf("reload after save: %v", err)
	}
	if second := reloaded.Devices[0].WiFiConfig.Radios[1]; second.Channel != 149 ||
		second.Band != "5GHz" {
		t.Errorf("second radio survived as %#v", second)
	}
	assertAuthoredRadios(t, reloaded)
}

// TestWiFiParserRejectsABandItCannotServe: the `oneof` vocabulary is what the
// schema publishes, so a band outside it must fail at load rather than reach a
// validator that has no channel range for it.
func TestWiFiParserRejectsABandItCannotServe(t *testing.T) {
	_, err := LoadYAMLBytes([]byte(strings.Replace(wifiScenarioYAML, "2.4GHz", "60GHz", 1)))
	if err == nil {
		t.Fatal("LoadYAMLBytes() accepted a 60GHz radio")
	}
}

func assertAuthoredRadios(t *testing.T, cfg *Config) {
	t.Helper()
	wifi := cfg.Devices[0].WiFiConfig
	if wifi == nil || len(wifi.Radios) != 2 {
		t.Fatalf("wifi block = %#v, want two radios", wifi)
	}
	first := wifi.Radios[0]
	want := WiFiRadio{
		Interface:  "Dot11Radio0",
		SSID:       "corp-wifi",
		BSSID:      "00:0c:ce:88:23:c7",
		Band:       "2.4GHz",
		Channel:    6,
		TxPowerDBM: 17,
	}
	if first != want {
		t.Errorf("first radio = %#v, want %#v", first, want)
	}
}
