package scenario_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

func assertEnterpriseDeviceMix(t *testing.T, cfg *config.Config) {
	t.Helper()
	for _, site := range []string{"COS", "EVT", "EHV", "LON"} {
		checks := map[string]int{
			site + "-WAN-R": 2, site + "-FW": 2, site + "-CORE-SW": 2, site + "-DIST-SW": 4,
			site + "-ACC-SW": 16, site + "-SRV-SW": 2, site + "-WAP-": 32,
		}
		// A printer fills an endpoint slot like any other wired client, so it
		// counts here — otherwise adding one to the rotation reads as the site
		// having lost clients rather than gained a printer.
		wiredClients := countNamed(cfg, site+"-WS-") + countNamed(cfg, site+"-LAP-") +
			countNamed(cfg, site+"-MBP-") + countNamed(cfg, site+"-PRN-")
		if wiredClients != 64 {
			t.Errorf("%s wired clients = %d, want 64", site, wiredClients)
		}
		for prefix, want := range checks {
			if got := countNamed(cfg, prefix); got != want {
				t.Errorf("devices matching %q = %d, want %d", prefix, got, want)
			}
		}
		for _, role := range []string{"DNS01", "DHCP01", "APP01", "FILE01", "NMS01", "PERF01"} {
			if findDevice(cfg, site+"-"+role) == nil {
				t.Errorf("missing %s-%s", site, role)
			}
		}
		assertCoreTypes(t, cfg, site)
	}
	for _, device := range cfg.Devices {
		if strings.Contains(device.Name, "-WAP-") &&
			(device.Properties["wifiStandard"] != "Wi-Fi 7" || !strings.Contains(device.SNMPConfig.SysDescr, "Wi-Fi 7")) {
			t.Errorf("%s does not identify as Wi-Fi 7", device.Name)
		}
	}
}

func assertCoreTypes(t *testing.T, cfg *config.Config, site string) {
	t.Helper()
	for _, suffix := range []string{"CORE-SW01", "CORE-SW02"} {
		device := findDevice(cfg, site+"-"+suffix)
		if device == nil || device.Type != "layer3-switch" {
			t.Errorf("%s-%s type = %q, want layer3-switch", site, suffix, deviceType(device))
		}
	}
}

func assertAPDiscoveryIdentity(t *testing.T, cfg *config.Config) {
	t.Helper()
	// The radios' own identity is the authored wifi block's, served by the
	// runtime; what stays an add_mibs row is the chassis ENTITY-MIB row a
	// discovery tool reads the model from.
	const apModelRow = "1.3.6.1.2.1.47.1.1.1.1.13.1"
	bssids := make(map[string]string)
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		if !strings.Contains(device.Name, "-WAP-") {
			continue
		}
		if !hasMIB(device, apModelRow) {
			t.Errorf("%s omits its ENTITY-MIB model row", device.Name)
		}
		for radio := range 4 {
			name := fmt.Sprintf("Dot11Radio%d", radio)
			iface := findInterface(device, name)
			if iface == nil || iface.Type != "ieee80211" {
				t.Errorf("%s omits IEEE 802.11 interface %s", device.Name, name)
			}
		}
		assertCompleteAPRadioMIBs(t, device, bssids)
	}
}

func assertCompleteAPRadioMIBs(t *testing.T, device *config.Device, bssids map[string]string) {
	t.Helper()
	const (
		radioCount = 4
		// Station columns 1 and 9 -- the station ID and the desired SSID --
		// belong to the authored radio and are served from the wifi block, so
		// they are not add_mibs rows.
		firstStationColumn = 2
		lastStationColumn  = 23
		desiredSSIDColumn  = 9
		// dot11OperationTable's first column is the radio's MAC and the PHY
		// table's is the PHY type, both the authored radio's; what remains on
		// both is columns 2 and 3.
		firstPairedColumn = 2
		lastPairedColumn  = 3
	)
	if device.WiFiConfig == nil || len(device.WiFiConfig.Radios) != radioCount {
		t.Fatalf("%s authors %d radios, want %d",
			device.Name, len(wifiRadios(device)), radioCount)
	}
	mibs := make(map[string]config.AddMib, len(device.SNMPConfig.AddMibs))
	for _, mib := range device.SNMPConfig.AddMibs {
		mibs[strings.TrimPrefix(mib.OID, ".")] = mib
	}
	agent := snmp.NewAgent(device, 0)
	for radio, authored := range device.WiFiConfig.Radios {
		// The rows are keyed by the interface's ifIndex, which is not the
		// radio's position: the access point's trunked uplink takes ifIndex 1.
		index := radioIfIndex(t, agent, device, authored.Interface)
		for column := firstStationColumn; column <= lastStationColumn; column++ {
			if column == desiredSSIDColumn {
				continue
			}
			assertRadioRow(t, device, mibs, "1.2.840.10036.1.1.1", column, index)
		}
		for column := firstPairedColumn; column <= lastPairedColumn; column++ {
			assertRadioRow(t, device, mibs, "1.2.840.10036.2.1.1", column, index)
			assertRadioRow(t, device, mibs, "1.2.840.10036.4.1.1", column, index)
		}
		if owner, exists := bssids[authored.BSSID]; exists {
			t.Errorf("BSSID %s belongs to both %s and %s", authored.BSSID, owner, device.Name)
		}
		bssids[authored.BSSID] = device.Name
		// Each radio reports the band of the same plan that sets its interface
		// speed. The PHY type used to be restated here as a value the plan
		// declared -- vht(8) on a table whose first column is the radio's MAC
		// -- so the assertion pinned the contradiction rather than catching it.
		if want := scenario.APRadioBandForTest(radio); authored.Band != want {
			t.Errorf("%s radio %d band = %q, want %q",
				device.Name, radio, authored.Band, want)
		}
	}
}

func assertRadioRow(
	t *testing.T,
	device *config.Device,
	mibs map[string]config.AddMib,
	table string,
	column int,
	ifIndex string,
) {
	t.Helper()
	oid := fmt.Sprintf("%s.%d.%s", table, column, ifIndex)
	if _, exists := mibs[oid]; !exists {
		t.Errorf("%s ifIndex %s omits %s column %d", device.Name, ifIndex, table, column)
	}
}

func wifiRadios(device *config.Device) []config.WiFiRadio {
	if device.WiFiConfig == nil {
		return nil
	}

	return device.WiFiConfig.Radios
}

func hasMIB(device *config.Device, oid string) bool {
	for _, mib := range device.SNMPConfig.AddMibs {
		if strings.TrimPrefix(mib.OID, ".") == oid {
			return true
		}
	}
	return false
}

func deviceType(device *config.Device) string {
	if device == nil {
		return ""
	}
	return device.Type
}

func assertAuthoredInterfacesAndLinks(t *testing.T, cfg *config.Config) {
	t.Helper()
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		for _, iface := range device.Interfaces {
			if iface.MTU == 0 || iface.Speed == 0 ||
				(iface.Type == "ethernet" && iface.Duplex != "full") ||
				iface.AdminStatus != "up" || iface.OperStatus != "up" {
				t.Errorf("%s %s lacks explicit link state: %+v", device.Name, iface.Name, iface)
			}
			if iface.Type == "ethernet" && (iface.InUtilization <= 0 || iface.OutUtilization <= 0) {
				t.Errorf("%s %s lacks live utilization: %+v", device.Name, iface.Name, iface)
			}
		}
	}
	assertAuthoredLinks(t, cfg)
	assertRoutedEdgesUntagged(t, cfg)
}

func assertRoutedEdgesUntagged(t *testing.T, cfg *config.Config) {
	t.Helper()
	pairs := [][2]string{
		{"LAB-EDGE-R1", "WAN-R1"},
		{"WAN-R1", "COS-WAN-R01"},
		{"COS-WAN-R01", "COS-FW01"},
		{"COS-FW01", "COS-CORE-SW01"},
	}
	for _, pair := range pairs {
		device := findDevice(cfg, pair[0])
		port := findRemotePort(device, pair[1])
		if port == nil || port.NativeVLAN != 0 || len(port.VLANs) != 0 {
			t.Errorf("routed edge %s -> %s carries VLAN metadata: %+v", pair[0], pair[1], port)
		}
	}
}

func findRemotePort(device *config.Device, remote string) *config.TrunkPort {
	for index := range device.TrunkPorts {
		if device.TrunkPorts[index].RemoteDevice == remote {
			return &device.TrunkPorts[index]
		}
	}
	return nil
}

func assertAuthoredLinks(t *testing.T, cfg *config.Config) {
	t.Helper()
	devices := make(map[string]*config.Device, len(cfg.Devices))
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		devices[device.Name] = device
	}
	for deviceIndex := range cfg.Devices {
		device := &cfg.Devices[deviceIndex]
		seen := make(map[string]bool)
		for _, port := range device.TrunkPorts {
			if seen[port.Interface] {
				t.Errorf("%s interface %s has multiple physical peers", device.Name, port.Interface)
			}
			seen[port.Interface] = true
			if findInterface(device, port.Interface) == nil {
				t.Errorf(
					"%s trunk %s is absent from authored interfaces",
					device.Name,
					port.Interface,
				)
			}
			remote := devices[port.RemoteDevice]
			if remote == nil {
				t.Errorf("%s trunk references missing device %s", device.Name, port.RemoteDevice)
				continue
			}
			if findInterface(remote, port.RemoteInterface) == nil {
				t.Errorf(
					"%s trunk references missing interface %s %s",
					device.Name,
					remote.Name,
					port.RemoteInterface,
				)
			}
			if !port.FDBOnly &&
				!hasReciprocalPort(remote, device.Name, port.RemoteInterface, port.Interface) {
				t.Errorf(
					"%s %s has no reciprocal link on %s %s",
					device.Name, port.Interface, remote.Name, port.RemoteInterface,
				)
			}
		}
	}
}

func findInterface(device *config.Device, name string) *config.Interface {
	for index := range device.Interfaces {
		if device.Interfaces[index].Name == name {
			return &device.Interfaces[index]
		}
	}
	return nil
}

func hasReciprocalPort(device *config.Device, remote, localInterface, remoteInterface string) bool {
	for _, port := range device.TrunkPorts {
		if port.Interface == localInterface && port.RemoteDevice == remote &&
			port.RemoteInterface == remoteInterface {
			return true
		}
	}
	return false
}
