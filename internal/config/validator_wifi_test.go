package config

import (
	"net"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

const wifiAPName = "MED-AP-01"

func wifiAP(radios ...WiFiRadio) Device {
	return Device{
		Name:        wifiAPName,
		Type:        "ap",
		MACAddress:  net.HardwareAddr{0x00, 0x0c, 0xce, 0x88, 0x23, 0xc0},
		IPAddresses: []net.IP{net.ParseIP("10.20.220.11")},
		Interfaces: []Interface{
			{Name: "Dot11Radio0", Type: "ieee80211"},
			{Name: "Dot11Radio1", Type: "ieee80211"},
			{Name: "GigabitEthernet0", Type: "ethernet"},
		},
		WiFiConfig: &WiFiConfig{Radios: radios},
	}
}

func validRadio() WiFiRadio {
	return WiFiRadio{
		Interface:  "Dot11Radio0",
		SSID:       "corp-wifi",
		BSSID:      "00:0c:ce:88:23:c7",
		Band:       "2.4GHz",
		Channel:    6,
		TxPowerDBM: 17,
	}
}

func TestValidateWiFiRadio(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		radio func(WiFiRadio) WiFiRadio
		field string
	}{
		{"a 2.4 GHz radio as authored", func(r WiFiRadio) WiFiRadio { return r }, ""},
		{
			"an interface the device does not have",
			func(r WiFiRadio) WiFiRadio { r.Interface = "Dot11Radio9"; return r },
			"wifi.radios[0].interface",
		},
		{
			"an interface that is not a radio",
			func(r WiFiRadio) WiFiRadio { r.Interface = "GigabitEthernet0"; return r },
			"wifi.radios[0].interface",
		},
		{
			"a BSSID that is not a MAC",
			func(r WiFiRadio) WiFiRadio { r.BSSID = "00:0c:ce:88:23"; return r },
			"wifi.radios[0].bssid",
		},
		{
			"a multicast BSSID",
			func(r WiFiRadio) WiFiRadio { r.BSSID = "01:0c:ce:88:23:c7"; return r },
			"wifi.radios[0].bssid",
		},
		{
			"an SSID longer than the MIB can carry",
			func(r WiFiRadio) WiFiRadio {
				r.SSID = "0123456789012345678901234567890123"

				return r
			},
			"wifi.radios[0].ssid",
		},
		{
			"an empty SSID",
			func(r WiFiRadio) WiFiRadio { r.SSID = ""; return r },
			"wifi.radios[0].ssid",
		},
		{
			"a band the radio cannot be in",
			func(r WiFiRadio) WiFiRadio { r.Band = "60GHz"; return r },
			"wifi.radios[0].band",
		},
		{
			"channel 36 in the 2.4 GHz band",
			func(r WiFiRadio) WiFiRadio { r.Channel = 36; return r },
			"wifi.radios[0].channel",
		},
		{
			"channel 6 in the 5 GHz band",
			func(r WiFiRadio) WiFiRadio { r.Band = "5GHz"; r.Channel = 6; return r },
			"wifi.radios[0].channel",
		},
		{
			"5 GHz channel 149",
			func(r WiFiRadio) WiFiRadio { r.Band = "5GHz"; r.Channel = 149; return r },
			"",
		},
		{
			"6 GHz channel 213",
			func(r WiFiRadio) WiFiRadio { r.Band = "6GHz"; r.Channel = 213; return r },
			"",
		},
		{
			"transmit power a radio cannot reach",
			func(r WiFiRadio) WiFiRadio { r.TxPowerDBM = 31; return r },
			"wifi.radios[0].tx_power_dbm",
		},
		{
			"no transmit power at all",
			func(r WiFiRadio) WiFiRadio { r.TxPowerDBM = 0; return r },
			"wifi.radios[0].tx_power_dbm",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := &Config{Devices: []Device{wifiAP(testCase.radio(validRadio()))}}

			if testCase.field == "" {
				if message := anyWiFiError(t, cfg); message != "" {
					t.Errorf("rejected a valid radio: %s", message)
				}

				return
			}
			if message := errorFor(t, cfg, testCase.field); message == "" {
				t.Errorf("accepted %s, want an error on %s", testCase.name, testCase.field)
			}
		})
	}
}

func TestValidateWiFiRadiosShareAnInterface(t *testing.T) {
	second := validRadio()
	second.SSID = "guest-wifi"
	second.BSSID = "00:0c:ce:88:23:c8"
	cfg := &Config{Devices: []Device{wifiAP(validRadio(), second)}}

	if message := errorFor(t, cfg, "wifi.radios[1].interface"); message == "" {
		t.Error("accepted two radios on one interface, want an error")
	}
}

// TestValidateWiFiBSSIDIsUniqueAcrossTheScenario: a BSSID is how a tester tells
// two radios apart, so the same one twice makes one of them invisible.
func TestValidateWiFiBSSIDIsUniqueAcrossTheScenario(t *testing.T) {
	other := wifiAP(validRadio())
	other.Name = "MED-AP-02"
	other.MACAddress = net.HardwareAddr{0x00, 0x0c, 0xce, 0x88, 0x23, 0xd0}
	cfg := &Config{Devices: []Device{wifiAP(validRadio()), other}}

	if message := errorFor(t, cfg, "wifi.radios[0].bssid"); message == "" {
		t.Error("accepted one BSSID on two access points, want an error")
	}
}

// TestValidateWiFiBlockWithoutRadios: the block itself says the device is an
// access point, so an empty one is an authoring mistake rather than an AP with
// its radios off.
func TestValidateWiFiBlockWithoutRadios(t *testing.T) {
	cfg := &Config{Devices: []Device{wifiAP()}}

	if message := errorFor(t, cfg, "wifi.radios"); message == "" {
		t.Error("accepted a wifi block with no radio, want an error")
	}
}

func anyWiFiError(t *testing.T, cfg *Config) string {
	t.Helper()
	for _, err := range NewValidator("test.yaml").Validate(cfg).Errors {
		if strings.Contains(err.Field, "wifi") {
			return err.Field + ": " + err.Message
		}
	}

	return ""
}

// TestWiFiBandsMatchTheParserVocabulary: the band set is declared twice -- once
// as the `oneof` the YAML parser enforces and publishes to the schema, once as
// the ranges this validator checks a channel against -- and a band in only one
// of them is either unauthorable or unchecked.
func TestWiFiBandsMatchTheParserVocabulary(t *testing.T) {
	field, ok := reflect.TypeFor[converter.WifiRadio]().FieldByName("Band")
	if !ok {
		t.Fatal("converter.WifiRadio has no Band field")
	}
	_, vocabulary, found := strings.Cut(field.Tag.Get("validate"), "oneof=")
	if !found {
		t.Fatalf("Band carries no oneof vocabulary: %q", field.Tag.Get("validate"))
	}

	parser := strings.Fields(vocabulary)
	if !slices.Equal(parser, WiFiBands()) {
		t.Errorf("parser accepts %v, validator checks %v", parser, WiFiBands())
	}
	for _, band := range parser {
		if _, known := wifiChannelRange(band); !known {
			t.Errorf("band %s has no channel range", band)
		}
	}
}

func validClient() WiFiClient {
	return WiFiClient{
		MAC:               "02:c0:17:a4:03:6b",
		IPAddress:         "10.20.220.51",
		AssociatedSeconds: 115286,
		SignalDBM:         -55,
		SignalQualityPct:  83,
	}
}

func TestValidateWiFiClient(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		client func(WiFiClient) WiFiClient
		field  string
	}{
		{"a station as authored", func(c WiFiClient) WiFiClient { return c }, ""},
		{
			"a MAC that is not a MAC",
			func(c WiFiClient) WiFiClient { c.MAC = "02:c0:17:a4:03"; return c },
			"wifi.radios[0].clients[0].mac",
		},
		{
			"a multicast MAC",
			func(c WiFiClient) WiFiClient { c.MAC = "01:c0:17:a4:03:6b"; return c },
			"wifi.radios[0].clients[0].mac",
		},
		{
			"an address that is not IPv4",
			func(c WiFiClient) WiFiClient { c.IPAddress = "2001:db8::1"; return c },
			"wifi.radios[0].clients[0].ip_address",
		},
		{
			"no address at all",
			func(c WiFiClient) WiFiClient { c.IPAddress = ""; return c },
			"wifi.radios[0].clients[0].ip_address",
		},
		{
			"a station that has been associated for no time",
			func(c WiFiClient) WiFiClient { c.AssociatedSeconds = 0; return c },
			"wifi.radios[0].clients[0].associated_seconds",
		},
		{
			"a signal written as a positive number",
			func(c WiFiClient) WiFiClient { c.SignalDBM = 55; return c },
			"wifi.radios[0].clients[0].signal_dbm",
		},
		{
			"a signal below any receive sensitivity",
			func(c WiFiClient) WiFiClient { c.SignalDBM = -101; return c },
			"wifi.radios[0].clients[0].signal_dbm",
		},
		{
			"a quality over one hundred percent",
			func(c WiFiClient) WiFiClient { c.SignalQualityPct = 101; return c },
			"wifi.radios[0].clients[0].signal_quality_pct",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			radio := validRadio()
			radio.Clients = []WiFiClient{testCase.client(validClient())}
			cfg := &Config{Devices: []Device{wifiAP(radio)}}

			if testCase.field == "" {
				if message := anyWiFiError(t, cfg); message != "" {
					t.Errorf("rejected a valid station: %s", message)
				}

				return
			}
			if message := errorFor(t, cfg, testCase.field); message == "" {
				t.Errorf("accepted %s, want an error on %s", testCase.name, testCase.field)
			}
		})
	}
}

// TestValidateWiFiClientsShareAMAC: the MAC is part of the row index, so two
// stations sharing one collapse into a single row reporting whichever was
// authored last.
func TestValidateWiFiClientsShareAMAC(t *testing.T) {
	radio := validRadio()
	radio.Clients = []WiFiClient{validClient(), validClient()}
	cfg := &Config{Devices: []Device{wifiAP(radio)}}

	if message := errorFor(t, cfg, "wifi.radios[0].clients[1].mac"); message == "" {
		t.Error("accepted two stations with one MAC on the same radio")
	}
}

// TestValidateWiFiClientsMayRepeatAcrossRadios is the other side of that: the
// index carries the radio's ifIndex too, so the same station associated to two
// radios is two rows. It is what a roam looks like mid-flight.
func TestValidateWiFiClientsMayRepeatAcrossRadios(t *testing.T) {
	first := validRadio()
	first.Clients = []WiFiClient{validClient()}
	second := validRadio()
	second.Interface = "Dot11Radio1"
	second.BSSID = "00:0c:ce:88:23:c8"
	second.Band = "5GHz"
	second.Channel = 149
	second.Clients = []WiFiClient{validClient()}
	cfg := &Config{Devices: []Device{wifiAP(first, second)}}

	if message := anyWiFiError(t, cfg); message != "" {
		t.Errorf("rejected one station on two radios: %s", message)
	}
}
