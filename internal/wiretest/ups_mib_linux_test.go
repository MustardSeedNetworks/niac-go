//go:build linux && integration

package wiretest_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// P4-4: a manager that finds an APC Network Management Card walks RFC 1628 next.
// A UPS answering inside .33 is filed as power equipment; a switch answering
// outside it is not. Both ends are asserted, because an agent that answered
// .33 for every device would pass the first alone.
//
// No pack carries a UPS yet, so the probe runs against an authored UPS and
// switch on the transit network.
const (
	upsTarget         = "10.254.200.72"
	upsSwitch         = "10.254.200.73"
	upsCommunity      = "wire_ups"
	upsMIBRootOID     = ".1.3.6.1.2.1.33"
	upsIdentManufOID  = ".1.3.6.1.2.1.33.1.1.1.0"
	upsBatteryStatOID = ".1.3.6.1.2.1.33.1.2.1.0"
	batteryNormal     = 2
)

func TestUPSAnswersTheUPSMIBOnTheWire(t *testing.T) {
	startUPSWire(t)

	ups := dialUPSWire(t, upsTarget)
	probe, err := ups.GetNext([]string{upsMIBRootOID})
	if err != nil || len(probe.Variables) != 1 {
		t.Fatalf("GETNEXT %s on the UPS: %v", upsMIBRootOID, err)
	}
	if name := probe.Variables[0].Name; name != upsIdentManufOID {
		t.Fatalf("the UPS answered GETNEXT %s with %s, want %s", upsMIBRootOID, name, upsIdentManufOID)
	}
	battery, err := ups.Get([]string{upsBatteryStatOID})
	if err != nil || len(battery.Variables) != 1 {
		t.Fatalf("GET %s on the UPS: %v", upsBatteryStatOID, err)
	}
	if value := gosnmp.ToBigInt(battery.Variables[0].Value).Int64(); value != batteryNormal {
		t.Errorf("upsBatteryStatus = %d, want batteryNormal %d", value, batteryNormal)
	}
	t.Logf("ups: GETNEXT %s -> %s = %s; upsBatteryStatus = %v", upsMIBRootOID,
		probe.Variables[0].Name, probe.Variables[0].Value, battery.Variables[0].Value)

	notUPS, err := dialUPSWire(t, upsSwitch).GetNext([]string{upsMIBRootOID})
	if err != nil || len(notUPS.Variables) != 1 {
		t.Fatalf("GETNEXT %s on the switch: %v", upsMIBRootOID, err)
	}
	if name := notUPS.Variables[0].Name; strings.HasPrefix(name, upsMIBRootOID+".") {
		t.Errorf("the switch answered GETNEXT %s with %s", upsMIBRootOID, name)
	}
	t.Logf("switch: GETNEXT %s -> %s", upsMIBRootOID, notUPS.Variables[0].Name)
}

func startUPSWire(t *testing.T) {
	t.Helper()
	requireWire(t)

	root := t.TempDir()
	configPath := filepath.Join(root, "ups-wire.yaml")
	copyFile(t, filepath.Join("testdata", "ups-wire.yaml"), configPath)
	t.Setenv("NIAC_CONFIGS_DIR", root)

	d, err := daemon.NewDaemon(daemon.Config{
		StoragePath: "disabled",
		AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
			Interface: simIface, Mode: fabric.ModeAccess, AccessVLAN: accessVLAN,
		}},
	})
	if err != nil {
		t.Fatalf("daemon.NewDaemon: %v", err)
	}
	if startErr := d.StartSimulation(api.SimulationRequest{
		SessionID:      "wiretest-ups",
		Interface:      simIface,
		Attachment:     "tester",
		AttachmentMode: fabric.ModeAccess,
		AccessVLAN:     accessVLAN,
		ConfigPath:     configPath,
	}); startErr != nil {
		t.Fatalf("StartSimulation on %s: %v", simIface, startErr)
	}
	t.Cleanup(func() {
		if stopErr := d.StopSimulation(""); stopErr != nil {
			t.Errorf("StopSimulation: %v", stopErr)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = d.Shutdown(ctx)
	})
}

func dialUPSWire(t *testing.T, target string) *gosnmp.GoSNMP {
	t.Helper()
	client := &gosnmp.GoSNMP{
		Target: target, Port: 161, Community: upsCommunity,
		Version: gosnmp.Version2c, Timeout: 5 * time.Second, Retries: 3,
	}
	if err := client.Connect(); err != nil {
		t.Fatalf("connect to %s: %v", target, err)
	}
	t.Cleanup(func() { _ = client.Conn.Close() })

	return client
}
