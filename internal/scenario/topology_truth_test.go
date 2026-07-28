package scenario_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func assertEnterpriseDeviceMix(t *testing.T, cfg *config.Config) {
	t.Helper()
	for _, site := range []string{"COS", "EVT", "EHV", "LON"} {
		checks := map[string]int{
			site + "-WAN-R": 2, site + "-FW": 2, site + "-CORE-SW": 2, site + "-DIST-SW": 4,
			site + "-ACC-SW": 16, site + "-SRV-SW": 2, site + "-WAP-": 32, site + "-WS-": 64,
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
	}
	for _, device := range cfg.Devices {
		if strings.Contains(device.Name, "-WAP-") &&
			(device.Properties["wifiStandard"] != "Wi-Fi 7" || !strings.Contains(device.SNMPConfig.SysDescr, "Wi-Fi 7")) {
			t.Errorf("%s does not identify as Wi-Fi 7", device.Name)
		}
	}
}

func assertAuthoredInterfacesAndLinks(t *testing.T, cfg *config.Config) {
	t.Helper()
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		for _, iface := range device.Interfaces {
			if iface.MTU == 0 || iface.Speed == 0 || iface.Duplex != "full" ||
				iface.AdminStatus != "up" || iface.OperStatus != "up" {
				t.Errorf("%s %s lacks explicit link state: %+v", device.Name, iface.Name, iface)
			}
			if iface.Type == "ethernet" && (iface.InUtilization <= 0 || iface.OutUtilization <= 0) {
				t.Errorf("%s %s lacks live utilization: %+v", device.Name, iface.Name, iface)
			}
		}
	}
	assertAuthoredLinks(t, cfg)
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
				t.Errorf("%s trunk %s is absent from authored interfaces", device.Name, port.Interface)
			}
			remote := devices[port.RemoteDevice]
			if remote == nil {
				t.Errorf("%s trunk references missing device %s", device.Name, port.RemoteDevice)
				continue
			}
			if findInterface(remote, port.RemoteInterface) == nil {
				t.Errorf("%s trunk references missing interface %s %s", device.Name, remote.Name, port.RemoteInterface)
			}
			if !port.FDBOnly && !hasReciprocalPort(remote, device.Name, port.RemoteInterface, port.Interface) {
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
		if port.Interface == localInterface && port.RemoteDevice == remote && port.RemoteInterface == remoteInterface {
			return true
		}
	}
	return false
}
