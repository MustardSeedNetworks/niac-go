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

// P4-4: the probe an EtherScope sends every device it discovers is one GETNEXT
// at the Printer-MIB root (docs/design/consumer-oid-demand.tsv). A printer
// answering inside the subtree is filed as a printer; a switch answering
// outside it is filed as not one. Both ends are asserted, because an agent that
// answered .43 for every device would pass the first alone.
const (
	printerTarget      = "10.254.200.70"
	printerSwitch      = "10.254.200.71"
	printerCommunity   = "wire_printer"
	printerMIBRootOID  = ".1.3.6.1.2.1.43"
	hrDeviceTypeOID    = ".1.3.6.1.2.1.25.3.2.1.2.1"
	hrDevicePrinterOID = ".1.3.6.1.2.1.25.3.1.5"
)

func TestPrinterAnswersThePrinterProbeOnTheWire(t *testing.T) {
	startPrinterWire(t)

	printer := dialPrinterWire(t, printerTarget)
	probe, err := printer.GetNext([]string{printerMIBRootOID})
	if err != nil || len(probe.Variables) != 1 {
		t.Fatalf("GETNEXT %s on the printer: %v", printerMIBRootOID, err)
	}
	if name := probe.Variables[0].Name; !strings.HasPrefix(name, printerMIBRootOID+".") {
		t.Fatalf("the printer answered GETNEXT %s with %s, outside the Printer-MIB", printerMIBRootOID, name)
	}
	kind, err := printer.Get([]string{hrDeviceTypeOID})
	if err != nil || len(kind.Variables) != 1 {
		t.Fatalf("GET %s on the printer: %v", hrDeviceTypeOID, err)
	}
	if value, _ := kind.Variables[0].Value.(string); value != hrDevicePrinterOID {
		t.Errorf("hrDeviceType.1 = %v, want hrDevicePrinter %s", kind.Variables[0].Value, hrDevicePrinterOID)
	}
	t.Logf("printer: GETNEXT %s -> %s = %v; hrDeviceType.1 = %v", printerMIBRootOID,
		probe.Variables[0].Name, probe.Variables[0].Value, kind.Variables[0].Value)

	notPrinter, err := dialPrinterWire(t, printerSwitch).GetNext([]string{printerMIBRootOID})
	if err != nil || len(notPrinter.Variables) != 1 {
		t.Fatalf("GETNEXT %s on the switch: %v", printerMIBRootOID, err)
	}
	if name := notPrinter.Variables[0].Name; strings.HasPrefix(name, printerMIBRootOID+".") {
		t.Errorf("the switch answered GETNEXT %s with %s", printerMIBRootOID, name)
	}
	t.Logf("switch: GETNEXT %s -> %s", printerMIBRootOID, notPrinter.Variables[0].Name)
}

func startPrinterWire(t *testing.T) {
	t.Helper()
	requireWire(t)

	root := t.TempDir()
	configPath := filepath.Join(root, "printer-wire.yaml")
	copyFile(t, filepath.Join("testdata", "printer-wire.yaml"), configPath)
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
		SessionID:      "wiretest-printer",
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

func dialPrinterWire(t *testing.T, target string) *gosnmp.GoSNMP {
	t.Helper()
	client := &gosnmp.GoSNMP{
		Target: target, Port: 161, Community: printerCommunity,
		Version: gosnmp.Version2c, Timeout: 5 * time.Second, Retries: 3,
	}
	if err := client.Connect(); err != nil {
		t.Fatalf("connect to %s: %v", target, err)
	}
	t.Cleanup(func() { _ = client.Conn.Close() })

	return client
}
