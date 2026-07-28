package scenario_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func assertUniqueIdentityAndRoutes(t *testing.T, cfg *config.Config) {
	t.Helper()
	addresses := collectUniqueAddresses(t, cfg)
	assertUniqueMACs(t, cfg)
	assertOwnedRoutes(t, cfg, addresses)
}

func collectUniqueAddresses(t *testing.T, cfg *config.Config) map[string]string {
	t.Helper()
	addresses := make(map[string]string)
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		for _, address := range device.IPAddresses {
			key := address.String()
			if owner, exists := addresses[key]; exists {
				t.Errorf("IP %s belongs to both %s and %s", key, owner, device.Name)
			}
			addresses[key] = device.Name
		}
	}
	return addresses
}

func assertUniqueMACs(t *testing.T, cfg *config.Config) {
	t.Helper()
	macs := make(map[string]string)
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		mac := device.MACAddress.String()
		if owner, exists := macs[mac]; exists {
			t.Errorf("MAC %s belongs to both %s and %s", mac, owner, device.Name)
		}
		macs[mac] = device.Name
	}
}

func assertOwnedRoutes(t *testing.T, cfg *config.Config, addresses map[string]string) {
	t.Helper()
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		if device.DHCPConfig != nil && device.DHCPConfig.Router != nil {
			gateway := device.DHCPConfig.Router.String()
			if _, exists := addresses[gateway]; !exists {
				t.Errorf("%s advertises unowned DHCP gateway %s", device.Name, gateway)
			}
		}
		for _, route := range device.Routes {
			if findInterface(device, route.Via) == nil {
				t.Errorf("%s route %s uses missing interface %s", device.Name, route.Destination, route.Via)
			}
			if _, exists := addresses[route.NextHop]; !exists {
				t.Errorf("%s route %s uses unowned next hop %s", device.Name, route.Destination, route.NextHop)
			}
		}
	}
}

func assertServiceDNS(t *testing.T, cfg *config.Config) {
	t.Helper()
	for _, site := range []string{"COS", "EVT", "EHV", "LON"} {
		dns := findDevice(cfg, site+"-DNS01")
		dhcp := findDevice(cfg, site+"-DHCP01")
		if dns == nil || dhcp == nil || dns.DNSConfig == nil || len(dhcp.IPAddresses) == 0 {
			t.Fatalf("%s service DNS prerequisites are missing", site)
		}
		name := strings.ToLower(site) + "-dhcp01.demo.lab"
		var resolved string
		for _, record := range dns.DNSConfig.ForwardRecords {
			if record.Name == name {
				resolved = record.IP.String()
				break
			}
		}
		if got, want := resolved, dhcp.IPAddresses[0].String(); got != want {
			t.Errorf("%s resolves to %s, want DHCP address %s", name, got, want)
		}
	}
}

func countNamed(cfg *config.Config, prefix string) int {
	count := 0
	for _, device := range cfg.Devices {
		if strings.HasPrefix(device.Name, prefix) {
			count++
		}
	}
	return count
}

func findDevice(cfg *config.Config, name string) *config.Device {
	for index := range cfg.Devices {
		if cfg.Devices[index].Name == name {
			return &cfg.Devices[index]
		}
	}
	return nil
}
