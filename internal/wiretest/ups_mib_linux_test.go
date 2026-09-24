//go:build linux && integration

package wiretest_test

import (
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// P4-4: a manager that finds an APC Network Management Card walks RFC 1628 next.
// A UPS answering inside .33 is filed as power equipment; a switch answering
// outside it is not. Both ends are asserted, because an agent that answered
// .33 for every device would pass the first alone.
//
// Both are pack devices, probed from the pack's own attachment port: the site's
// UPS across the data VLAN, and the access switch the tester is plugged into.
const (
	upsMIBRootOID     = ".1.3.6.1.2.1.33"
	upsIdentManufOID  = ".1.3.6.1.2.1.33.1.1.1.0"
	upsBatteryStatOID = ".1.3.6.1.2.1.33.1.2.1.0"
	batteryNormal     = 2
)

func TestPackUPSAnswersTheUPSMIBOnTheWire(t *testing.T) {
	authored, _ := startPack(t, "hospital")
	upsName := firstDeviceWithRole(t, authored, "ups")
	switchName := authored.Attachments[0].At.Device

	ups := dialDevice(t, authored, upsName)
	probe, err := ups.GetNext([]string{upsMIBRootOID})
	if err != nil || len(probe.Variables) != 1 {
		t.Fatalf("GETNEXT %s on %s: %v", upsMIBRootOID, upsName, err)
	}
	if name := probe.Variables[0].Name; name != upsIdentManufOID {
		t.Fatalf("%s answered GETNEXT %s with %s, want %s", upsName, upsMIBRootOID, name, upsIdentManufOID)
	}
	battery, err := ups.Get([]string{upsBatteryStatOID})
	if err != nil || len(battery.Variables) != 1 {
		t.Fatalf("GET %s on %s: %v", upsBatteryStatOID, upsName, err)
	}
	if value := gosnmp.ToBigInt(battery.Variables[0].Value).Int64(); value != batteryNormal {
		t.Errorf("%s upsBatteryStatus = %d, want batteryNormal %d", upsName, value, batteryNormal)
	}
	t.Logf("%s: GETNEXT %s -> %s = %s; upsBatteryStatus = %v", upsName, upsMIBRootOID,
		probe.Variables[0].Name, probe.Variables[0].Value, battery.Variables[0].Value)

	notUPS, err := dialDevice(t, authored, switchName).GetNext([]string{upsMIBRootOID})
	if err != nil || len(notUPS.Variables) != 1 {
		t.Fatalf("GETNEXT %s on %s: %v", upsMIBRootOID, switchName, err)
	}
	if name := notUPS.Variables[0].Name; strings.HasPrefix(name, upsMIBRootOID+".") {
		t.Errorf("%s answered GETNEXT %s with %s", switchName, upsMIBRootOID, name)
	}
	t.Logf("%s: GETNEXT %s -> %s", switchName, upsMIBRootOID, notUPS.Variables[0].Name)
}

// firstDeviceWithRole finds a device by its generator role. A UPS is typed iot
// like the pack's clinical devices, so its type alone does not pick it out.
func firstDeviceWithRole(t *testing.T, cfg *config.Config, role string) string {
	t.Helper()
	for index := range cfg.Devices {
		if cfg.Devices[index].Properties["role"] == role {
			return cfg.Devices[index].Name
		}
	}
	t.Fatalf("the generated pack has no device with role %q", role)
	return ""
}
