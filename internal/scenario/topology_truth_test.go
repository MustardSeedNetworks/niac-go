package scenario_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func assertEnterpriseDeviceMix(t *testing.T, cfg *config.Config) {
	t.Helper()
	for _, site := range []string{"COS", "EVT", "EHV", "LON"} {
		checks := map[string]int{
			site + "-WAN-R": 2, site + "-FW": 2, site + "-CORE-SW": 2, site + "-DIST-SW": 4,
			site + "-ACC-SW": 16, site + "-SRV-SW": 2, site + "-WAP-": 32,
		}
		wiredClients := countNamed(cfg, site+"-WS-") + countNamed(cfg, site+"-LAP-") +
			countNamed(cfg, site+"-MBP-")
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
	const wirelessMarker = "1.2.840.10036.4.5.1.1.1"
	stationIDs := make(map[string]string)
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		if !strings.Contains(device.Name, "-WAP-") {
			continue
		}
		if !hasMIB(device, wirelessMarker) {
			t.Errorf("%s omits standardized wireless discovery identity", device.Name)
		}
		for radio := range 4 {
			name := fmt.Sprintf("Dot11Radio%d", radio)
			iface := findInterface(device, name)
			if iface == nil || iface.Type != "ieee80211" {
				t.Errorf("%s omits IEEE 802.11 interface %s", device.Name, name)
			}
		}
		assertCompleteAPRadioMIBs(t, device, stationIDs)
	}
}

func assertCompleteAPRadioMIBs(t *testing.T, device *config.Device, stationIDs map[string]string) {
	t.Helper()
	const (
		radioCount         = 4
		stationColumnCount = 23
		phyColumnCount     = 3
	)
	mibs := make(map[string]config.AddMib, len(device.SNMPConfig.AddMibs))
	for _, mib := range device.SNMPConfig.AddMibs {
		mibs[strings.TrimPrefix(mib.OID, ".")] = mib
	}
	for radio := 1; radio <= radioCount; radio++ {
		for column := 1; column <= stationColumnCount; column++ {
			oid := fmt.Sprintf("1.2.840.10036.1.1.1.%d.%d", column, radio)
			if _, exists := mibs[oid]; !exists {
				t.Errorf("%s radio %d omits station column %d", device.Name, radio, column)
			}
		}
		station := mibs[fmt.Sprintf("1.2.840.10036.1.1.1.1.%d", radio)].Value
		if owner, exists := stationIDs[station]; exists {
			t.Errorf("station ID %s belongs to both %s and %s", station, owner, device.Name)
		}
		stationIDs[station] = device.Name
		for column := 1; column <= phyColumnCount; column++ {
			oid := fmt.Sprintf("1.2.840.10036.2.1.1.%d.%d", column, radio)
			if _, exists := mibs[oid]; !exists {
				t.Errorf("%s radio %d omits PHY column %d", device.Name, radio, column)
			}
		}
		phyType := mibs[fmt.Sprintf("1.2.840.10036.2.1.1.1.%d", radio)]
		if phyType.Type != "INTEGER" || phyType.Value != "4" {
			t.Errorf(
				"%s radio %d PHY type = %+v, want INTEGER OFDM(4)",
				device.Name,
				radio,
				phyType,
			)
		}
	}
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
