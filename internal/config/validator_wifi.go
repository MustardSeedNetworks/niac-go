package config

import (
	"fmt"
	"net"
	"strings"
)

// Transmit-power bounds in dBm. The floor is one milliwatt, which is the
// weakest level IEEE802dot11-MIB can report, and the ceiling is the strongest
// any regulatory domain permits an indoor AP, so a mistyped power is caught
// rather than replayed.
const (
	minWiFiTxPowerDBM = 1
	maxWiFiTxPowerDBM = 30

	// maxSSIDOctets is the size of dot11DesiredSSID.
	maxSSIDOctets = 32

	// wifiRadioInterfaceType is the interface type a radio must be declared
	// as. It is the vocabulary IF-MIB reports as ieee80211(71); the dot11
	// tables are indexed by that interface's ifIndex.
	wifiRadioInterfaceType = "ieee80211"
)

// Bounds on what an associated station reports. The signal floor is below any
// usable receive sensitivity and the ceiling is a station on top of the AP, so
// a sign slip or a milliwatt value is caught rather than replayed.
const (
	minClientSignalDBM = -100
	maxClientSignalDBM = -20

	// maxSignalQualityPct is the percentage cDot11ClientSigQuality is defined
	// in.
	maxSignalQualityPct = 100
)

// Channel numbering per band. A band's range is the whole numbering, not a
// per-domain channel plan: which channels a country permits is a regulatory
// question the scenario does not model, and rejecting a channel that is legal
// somewhere would be worse than accepting one that is illegal here.
const (
	firstChannel24GHz = 1
	lastChannel24GHz  = 14
	firstChannel5GHz  = 32
	lastChannel5GHz   = 177
	firstChannel6GHz  = 1
	lastChannel6GHz   = 233
)

// channelRange is the first and last channel number of one band.
type channelRange struct {
	first int
	last  int
}

func wifiChannelRange(band string) (channelRange, bool) {
	switch band {
	case "2.4GHz":
		return channelRange{first: firstChannel24GHz, last: lastChannel24GHz}, true
	case "5GHz":
		return channelRange{first: firstChannel5GHz, last: lastChannel5GHz}, true
	case "6GHz":
		return channelRange{first: firstChannel6GHz, last: lastChannel6GHz}, true
	default:
		return channelRange{}, false
	}
}

// WiFiBands returns the bands a radio may be authored in, in the order the
// schema publishes them.
func WiFiBands() []string {
	return []string{"2.4GHz", "5GHz", "6GHz"}
}

// validateWiFi checks one device's radios. BSSID uniqueness is not here: it
// spans the whole scenario, so validateWiFiBSSIDs does it once the roster is
// known.
func (v *Validator) validateWiFi(device *Device, prefix string) {
	wifi := device.WiFiConfig
	if wifi == nil {
		return
	}

	prefix += ".wifi"
	if len(wifi.Radios) == 0 {
		v.addError(prefix+".radios",
			"a wifi block declares an access point, so it needs at least one radio")

		return
	}

	radioInterfaces := radioInterfacesByName(device)
	claimed := make(map[string]int, len(wifi.Radios))
	for index := range wifi.Radios {
		v.validateWiFiRadio(
			&wifi.Radios[index],
			fmt.Sprintf("%s.radios[%d]", prefix, index),
			radioInterfaces,
			claimed,
		)
	}
}

func (v *Validator) validateWiFiRadio(
	radio *WiFiRadio,
	prefix string,
	radioInterfaces map[string]bool,
	claimed map[string]int,
) {
	v.validateWiFiRadioInterface(radio, prefix, radioInterfaces, claimed)

	if ssid := len(radio.SSID); ssid == 0 || ssid > maxSSIDOctets {
		v.addError(prefix+".ssid", fmt.Sprintf(
			"SSID must be 1 to %d octets, got %d", maxSSIDOctets, ssid))
	}

	if address, err := net.ParseMAC(radio.BSSID); err != nil {
		v.addError(prefix+".bssid",
			fmt.Sprintf("BSSID %q is not a MAC address", radio.BSSID))
	} else if address[0]&1 == 1 {
		v.addError(prefix+".bssid", fmt.Sprintf(
			"BSSID %s is a group address; a radio answers on its own unicast MAC",
			radio.BSSID))
	}

	v.validateWiFiRadioChannel(radio, prefix)

	if radio.TxPowerDBM < minWiFiTxPowerDBM || radio.TxPowerDBM > maxWiFiTxPowerDBM {
		v.addError(prefix+".tx_power_dbm", fmt.Sprintf(
			"transmit power must be between %d and %d dBm, got %d",
			minWiFiTxPowerDBM, maxWiFiTxPowerDBM, radio.TxPowerDBM))
	}

	v.validateWiFiClients(radio, prefix)
}

// validateWiFiClients checks the stations associated to one radio. Their MACs
// have to differ: the MAC is part of the row index, so two clients sharing one
// makes a single row that reports whichever was authored last.
func (v *Validator) validateWiFiClients(radio *WiFiRadio, prefix string) {
	associated := make(map[string]int, len(radio.Clients))
	for index := range radio.Clients {
		v.validateWiFiClient(
			&radio.Clients[index],
			fmt.Sprintf("%s.clients[%d]", prefix, index),
			associated,
		)
	}
}

func (v *Validator) validateWiFiClient(
	client *WiFiClient,
	prefix string,
	associated map[string]int,
) {
	v.validateWiFiClientAddress(client, prefix, associated)

	if client.AssociatedSeconds < 1 {
		v.addError(prefix+".associated_seconds", fmt.Sprintf(
			"a station that is associated has been so for at least a second, got %d",
			client.AssociatedSeconds))
	}

	if client.SignalDBM < minClientSignalDBM || client.SignalDBM > maxClientSignalDBM {
		v.addError(prefix+".signal_dbm", fmt.Sprintf(
			"signal strength must be between %d and %d dBm, got %d",
			minClientSignalDBM, maxClientSignalDBM, client.SignalDBM))
	}

	if client.SignalQualityPct < 1 || client.SignalQualityPct > maxSignalQualityPct {
		v.addError(prefix+".signal_quality_pct", fmt.Sprintf(
			"signal quality is a percentage of 1 to %d, got %d",
			maxSignalQualityPct, client.SignalQualityPct))
	}
}

func (v *Validator) validateWiFiClientAddress(
	client *WiFiClient,
	prefix string,
	associated map[string]int,
) {
	address, err := net.ParseMAC(client.MAC)
	switch {
	case err != nil:
		v.addError(prefix+".mac",
			fmt.Sprintf("MAC %q is not a MAC address", client.MAC))
	case address[0]&1 == 1:
		v.addError(prefix+".mac", fmt.Sprintf(
			"MAC %s is a group address; a station associates from its own unicast MAC",
			client.MAC))
	default:
		key := address.String()
		if first, taken := associated[key]; taken {
			v.addError(prefix+".mac", fmt.Sprintf(
				"MAC %s is already the station at index %d of this radio", key, first))
		} else {
			associated[key] = len(associated)
		}
	}

	if ip := net.ParseIP(client.IPAddress); ip == nil || ip.To4() == nil {
		v.addError(prefix+".ip_address", fmt.Sprintf(
			"IP address %q is not an IPv4 address", client.IPAddress))
	}
}

func (v *Validator) validateWiFiRadioInterface(
	radio *WiFiRadio,
	prefix string,
	radioInterfaces map[string]bool,
	claimed map[string]int,
) {
	isRadio, exists := radioInterfaces[radio.Interface]
	switch {
	case !exists:
		v.addError(prefix+".interface", fmt.Sprintf(
			"device has no interface %q", radio.Interface))
	case !isRadio:
		v.addError(prefix+".interface", fmt.Sprintf(
			"interface %q is not of type %s, so it has no radio to configure",
			radio.Interface, wifiRadioInterfaceType))
	default:
		if first, taken := claimed[radio.Interface]; taken {
			v.addError(prefix+".interface", fmt.Sprintf(
				"interface %q already carries the radio at index %d",
				radio.Interface, first))
		}
	}
	if _, taken := claimed[radio.Interface]; !taken {
		claimed[radio.Interface] = len(claimed)
	}
}

func (v *Validator) validateWiFiRadioChannel(radio *WiFiRadio, prefix string) {
	limits, known := wifiChannelRange(radio.Band)
	if !known {
		v.addError(prefix+".band", fmt.Sprintf(
			"band must be one of %s, got %q",
			strings.Join(WiFiBands(), ", "), radio.Band))

		return
	}
	if radio.Channel < limits.first || radio.Channel > limits.last {
		v.addError(prefix+".channel", fmt.Sprintf(
			"channel %d is outside the %s band's numbering (%d-%d)",
			radio.Channel, radio.Band, limits.first, limits.last))
	}
}

// validateWiFiBSSIDs reports a BSSID authored on more than one radio. The BSSID
// is what a wireless tester uses to tell two radios apart -- it is the address
// in every frame they send -- so the same one twice makes one of them
// unreportable rather than merely duplicated.
func (v *Validator) validateWiFiBSSIDs(cfg *Config) {
	owners := make(map[string]string)
	for _, segment := range cfg.NormalizedSegments() {
		for index := range segment.Devices {
			device := &segment.Devices[index]
			if device.WiFiConfig == nil {
				continue
			}
			for radio := range device.WiFiConfig.Radios {
				v.claimBSSID(device, radio, owners)
			}
		}
	}
}

func (v *Validator) claimBSSID(device *Device, radio int, owners map[string]string) {
	authored := device.WiFiConfig.Radios[radio].BSSID
	address, err := net.ParseMAC(authored)
	if err != nil {
		return
	}
	key := address.String()
	field := fmt.Sprintf("%s.wifi.radios[%d].bssid", device.Name, radio)
	if owner, taken := owners[key]; taken {
		v.addError(field, fmt.Sprintf("BSSID %s is already a radio of %s", key, owner))

		return
	}
	owners[key] = device.Name
}

// radioInterfacesByName maps every interface of the device to whether it is a
// radio, so an interface that is missing and one that is the wrong type get
// different errors.
func radioInterfacesByName(device *Device) map[string]bool {
	interfaces := make(map[string]bool, len(device.Interfaces))
	for _, iface := range device.Interfaces {
		interfaces[iface.Name] = strings.EqualFold(iface.Type, wifiRadioInterfaceType)
	}

	return interfaces
}
