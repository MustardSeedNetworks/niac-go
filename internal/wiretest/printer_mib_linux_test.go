//go:build linux && integration

package wiretest_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// P4-4: the probe an EtherScope sends every device it discovers is one GETNEXT
// at the Printer-MIB root (docs/design/consumer-oid-demand.tsv). A printer
// answering inside the subtree is filed as a printer; a switch answering
// outside it is filed as not one. Both ends are asserted, because an agent that
// answered .43 for every device would pass the first alone.
//
// Both are pack devices, probed from the pack's own attachment port: the
// printer across the data VLAN, and the access switch the tester is plugged
// into across the route to its management address.
const (
	printerMIBRootOID  = ".1.3.6.1.2.1.43"
	hrDeviceTypeOID    = ".1.3.6.1.2.1.25.3.2.1.2.1"
	hrDevicePrinterOID = ".1.3.6.1.2.1.25.3.1.5"
)

func TestPackPrinterAnswersThePrinterProbeOnTheWire(t *testing.T) {
	authored, _ := startPack(t, "hospital")
	printerName := firstDeviceOfType(t, authored, "printer")
	switchName := authored.Attachments[0].At.Device

	printer := dialDevice(t, authored, printerName)
	probe, err := printer.GetNext([]string{printerMIBRootOID})
	if err != nil || len(probe.Variables) != 1 {
		t.Fatalf("GETNEXT %s on %s: %v", printerMIBRootOID, printerName, err)
	}
	if name := probe.Variables[0].Name; !strings.HasPrefix(name, printerMIBRootOID+".") {
		t.Fatalf("%s answered GETNEXT %s with %s, outside the Printer-MIB", printerName, printerMIBRootOID, name)
	}
	kind, err := printer.Get([]string{hrDeviceTypeOID})
	if err != nil || len(kind.Variables) != 1 {
		t.Fatalf("GET %s on %s: %v", hrDeviceTypeOID, printerName, err)
	}
	if value, _ := kind.Variables[0].Value.(string); value != hrDevicePrinterOID {
		t.Errorf(
			"%s hrDeviceType.1 = %v, want hrDevicePrinter %s",
			printerName,
			kind.Variables[0].Value,
			hrDevicePrinterOID,
		)
	}
	t.Logf("%s: GETNEXT %s -> %s = %v; hrDeviceType.1 = %v", printerName, printerMIBRootOID,
		probe.Variables[0].Name, probe.Variables[0].Value, kind.Variables[0].Value)

	notPrinter, err := dialDevice(t, authored, switchName).GetNext([]string{printerMIBRootOID})
	if err != nil || len(notPrinter.Variables) != 1 {
		t.Fatalf("GETNEXT %s on %s: %v", printerMIBRootOID, switchName, err)
	}
	if name := notPrinter.Variables[0].Name; strings.HasPrefix(name, printerMIBRootOID+".") {
		t.Errorf("%s answered GETNEXT %s with %s", switchName, printerMIBRootOID, name)
	}
	t.Logf("%s: GETNEXT %s -> %s", switchName, printerMIBRootOID, notPrinter.Variables[0].Name)
}

func firstDeviceOfType(t *testing.T, cfg *config.Config, deviceType string) string {
	t.Helper()
	for index := range cfg.Devices {
		if cfg.Devices[index].Type == deviceType {
			return cfg.Devices[index].Name
		}
	}
	t.Fatalf("the generated pack has no device of type %q", deviceType)
	return ""
}
