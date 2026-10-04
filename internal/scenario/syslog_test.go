package scenario_test

import (
	"net"
	"net/netip"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// P2-3 added syslog emission and nothing shipped turned it on, so the feature
// was unreachable from every pack (#2105). Every pack must now point its
// managed devices at a collector, or an operator who injects a fault sees
// nothing.
func TestEveryPackPointsItsDevicesAtACollector(t *testing.T) {
	t.Parallel()

	for _, pack := range scenario.Packs() {
		cfg := generatePack(t, pack.ID)
		emitting := 0
		for index := range cfg.Devices {
			if cfg.Devices[index].SyslogConfig != nil &&
				cfg.Devices[index].SyslogConfig.Enabled {
				emitting++
			}
		}
		if emitting == 0 {
			t.Errorf("%s: no device emits syslog, so an injected fault is invisible", pack.ID)
		}
	}
}

// A receiver that is not the site's own collector sends every event to the
// wrong place, which is worse than sending none: the operator sees a healthy
// collector and a broken network.
func TestDevicesReportToTheirOwnSiteCollector(t *testing.T) {
	t.Parallel()

	cfg := generatePack(t, "campus")
	nms := map[string]string{}
	for index := range cfg.Devices {
		if site, found := strings.CutSuffix(cfg.Devices[index].Name, "-NMS01"); found {
			nms[site] = cfg.Devices[index].IPAddresses[0].String()
		}
	}
	if len(nms) == 0 {
		t.Fatal("campus generated no NMS server to collect anything")
	}

	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		if device.SyslogConfig == nil {
			continue
		}
		site, _, found := strings.Cut(device.Name, "-")
		if !found {
			continue
		}
		want, known := nms[site]
		if !known {
			continue
		}
		for _, receiver := range device.SyslogConfig.Receivers {
			if !strings.HasPrefix(receiver, want+":") {
				t.Errorf("%s reports to %s, want its own site collector %s:514",
					device.Name, receiver, want)
			}
		}
	}
}

// The collector does not report to itself: an NMS logging its own faults to
// itself tells an operator nothing they cannot already see.
func TestTheCollectorDoesNotReportToItself(t *testing.T) {
	t.Parallel()

	cfg := generatePack(t, "hospital")
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		if strings.HasSuffix(device.Name, "-NMS01") && device.SyslogConfig != nil {
			t.Errorf("%s reports syslog to itself", device.Name)
		}
	}
}

// A device with no route to its collector never sends: the emission path fails
// with "no active local egress route" before a frame is built, so every fault
// on a pack switch or access point was silent (#2410). Each receiver must sit
// on one of the device's own subnets or behind a static route out of one.
func TestEverySyslogSenderCanRouteToItsCollector(t *testing.T) {
	t.Parallel()

	for _, pack := range scenario.Packs() {
		cfg := generatePack(t, pack.ID)
		for index := range cfg.Devices {
			device := &cfg.Devices[index]
			if device.SyslogConfig == nil || !device.SyslogConfig.Enabled {
				continue
			}
			for _, receiver := range device.SyslogConfig.Receivers {
				host, _, err := net.SplitHostPort(receiver)
				if err != nil {
					t.Fatalf("%s/%s: receiver %q: %v", pack.ID, device.Name, receiver, err)
				}
				if !routesTo(device, netip.MustParseAddr(host)) {
					t.Errorf("%s/%s has no route to its collector %s", pack.ID, device.Name, host)
				}
			}
		}
	}
}

func routesTo(device *config.Device, destination netip.Addr) bool {
	addressed := map[string]bool{}
	for _, iface := range device.Interfaces {
		prefix, err := netip.ParsePrefix(iface.Address)
		if err != nil {
			continue
		}
		addressed[iface.Name] = true
		if prefix.Masked().Contains(destination) {
			return true
		}
	}
	for _, route := range device.Routes {
		prefix, err := netip.ParsePrefix(route.Destination)
		if err == nil && prefix.Contains(destination) && addressed[route.Via] {
			return true
		}
	}
	return false
}
