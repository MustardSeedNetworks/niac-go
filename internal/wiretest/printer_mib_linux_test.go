//go:build linux && integration

package wiretest_test

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// P4-4: the probe an EtherScope sends every device it discovers is one GETNEXT
// at the Printer-MIB root (docs/design/consumer-oid-demand.tsv). A pack printer
// answering inside the subtree is filed as a printer; the switch it hangs off
// answering outside it is filed as not one. Both ends are asserted, because a
// fabric that answered .43 for every device would pass the first alone.
const (
	printerMIBRootOID  = ".1.3.6.1.2.1.43"
	hrDeviceTypeOID    = ".1.3.6.1.2.1.25.3.2.1.2.1"
	hrDevicePrinterOID = ".1.3.6.1.2.1.25.3.1.5"
)

func TestPackPrinterAnswersThePrinterProbeOnTheWire(t *testing.T) {
	authored := startPack(t, "hospital")
	printer := firstDeviceOfType(t, authored, "printer")
	other := firstDeviceOfType(t, authored, "switch")

	client := dialDevice(t, authored, printer)
	probe, err := client.GetNext([]string{printerMIBRootOID})
	if err != nil || len(probe.Variables) != 1 {
		t.Fatalf("GETNEXT %s on %s: %v", printerMIBRootOID, printer, err)
	}
	if name := probe.Variables[0].Name; !strings.HasPrefix(name, printerMIBRootOID+".") {
		t.Fatalf("%s answered GETNEXT %s with %s, outside the Printer-MIB", printer, printerMIBRootOID, name)
	}
	kind, err := client.Get([]string{hrDeviceTypeOID})
	if err != nil || len(kind.Variables) != 1 {
		t.Fatalf("GET %s on %s: %v", hrDeviceTypeOID, printer, err)
	}
	if value, _ := kind.Variables[0].Value.(string); value != hrDevicePrinterOID {
		t.Errorf("%s hrDeviceType.1 = %v, want hrDevicePrinter %s", printer, kind.Variables[0].Value, hrDevicePrinterOID)
	}
	t.Logf("%s: GETNEXT %s -> %s; hrDeviceType.1 = %v",
		printer, printerMIBRootOID, probe.Variables[0].Name, kind.Variables[0].Value)

	switchClient := dialDevice(t, authored, other)
	notPrinter, err := switchClient.GetNext([]string{printerMIBRootOID})
	if err != nil || len(notPrinter.Variables) != 1 {
		t.Fatalf("GETNEXT %s on %s: %v", printerMIBRootOID, other, err)
	}
	if name := notPrinter.Variables[0].Name; strings.HasPrefix(name, printerMIBRootOID+".") {
		t.Errorf("%s is a switch and answered GETNEXT %s with %s", other, printerMIBRootOID, name)
	}
}

func firstDeviceOfType(t *testing.T, authored *config.Config, kind string) string {
	t.Helper()
	for _, device := range authored.Devices {
		if device.Type == kind && len(device.IPAddresses) > 0 {
			return device.Name
		}
	}
	t.Fatalf("the pack has no addressed %s", kind)

	return ""
}
