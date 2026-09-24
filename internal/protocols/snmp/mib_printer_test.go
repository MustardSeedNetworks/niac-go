package snmp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	printerMIBRoot     = "1.3.6.1.2.1.43"
	officePrinterDescr = "HP LaserJet Enterprise MFP M634, HP FutureSmart"
)

func officePrinter() *Agent {
	device := createTestDevice()
	device.Name = "HQ-PRN-01"
	device.Type = "printer"
	device.SNMPConfig.SysName = "HQ-PRN-01"
	device.SNMPConfig.SysDescr = officePrinterDescr

	return NewAgent(device, 0)
}

func mustGet(t *testing.T, agent *Agent, oid string) *OIDValue {
	t.Helper()
	value := agent.mib.Get(oid)
	if value == nil {
		t.Fatalf("%s is absent", oid)
	}

	return value
}

// TestPrinterAnswersTheDiscoveryProbe is the demand itself: the EtherScope
// capture behind consumer-oid-demand.tsv sends every device it finds one
// GETNEXT at the Printer-MIB root, and a device that answers inside the subtree
// is filed as a printer. A pack printer used to answer with whatever followed
// .43 in its MIB, which is the same answer a switch gives.
func TestPrinterAnswersTheDiscoveryProbe(t *testing.T) {
	printer := officePrinter()

	next, value := printer.mib.GetNext(printerMIBRoot)
	if value == nil || !strings.HasPrefix(next, printerMIBRoot+".") {
		t.Fatalf("GETNEXT %s = %s, want an object inside the Printer-MIB", printerMIBRoot, next)
	}
}

// TestPrinterReportsItselfAsAnHrDevicePrinter pins the rows a manager reads
// after the probe. RFC 3805 indexes every Printer-MIB table by hrDeviceIndex, so
// the printer has to exist in HOST-RESOURCES-MIB's device table first, typed as
// a printer; hrPrinterStatus is what monitoring polls.
func TestPrinterReportsItselfAsAnHrDevicePrinter(t *testing.T) {
	printer := officePrinter()

	if got := oidValueString(mustGet(t, printer, hrDeviceType+".1")); got != hrDeviceTypePrinter {
		t.Errorf("hrDeviceType.1 = %s, want hrDevicePrinter %s", got, hrDeviceTypePrinter)
	}
	if got := oidValueString(mustGet(t, printer, hrDeviceDescr+".1")); got != officePrinterDescr {
		t.Errorf("hrDeviceDescr.1 = %q, want the authored sysDescr", got)
	}
	wantInt(t, mustGet(t, printer, hrDeviceStatus+".1"), hrDeviceStatusRunning, "hrDeviceStatus")
	wantInt(t, mustGet(t, printer, hrPrinterStatus+".1"), hrPrinterStatusIdle, "hrPrinterStatus")

	if got := oidValueString(mustGet(t, printer, prtGeneralPrinterName+".1")); got != "HQ-PRN-01" {
		t.Errorf("prtGeneralPrinterName.1 = %q, want the authored sysName", got)
	}
	wantInt(t, mustGet(t, printer, prtGeneralReset+".1"), prtGeneralResetNotResetting, "prtGeneralReset")
}

// TestPrinterMIBOnlyOnPrinters is the other half of the probe: a switch that
// answered inside .43 would be filed as a printer by the same instrument.
func TestPrinterMIBOnlyOnPrinters(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)

	if next, _ := agent.mib.GetNext(printerMIBRoot); strings.HasPrefix(next, printerMIBRoot+".") {
		t.Fatalf("a router answered GETNEXT %s with %s", printerMIBRoot, next)
	}
	if value := agent.mib.Get(hrDeviceType + ".1"); value != nil {
		t.Errorf("a router reported hrDeviceType.1 = %s", oidValueString(value))
	}
}

// TestPrinterMIBAfterAWalkWithoutIt covers a walk-backed printer whose capture
// carries HOST-RESOURCES devices of its own but no Printer-MIB. The printer row
// takes the next free hrDeviceIndex rather than overwriting the capture's
// processor at index 1.
func TestPrinterMIBAfterAWalkWithoutIt(t *testing.T) {
	walk := writeTestWalk(t, ""+
		".1.3.6.1.2.1.1.5.0 = STRING: HQ-PRN-02\r\n"+
		".1.3.6.1.2.1.25.3.2.1.2.1 = OID: .1.3.6.1.2.1.25.3.1.3\r\n"+
		".1.3.6.1.2.1.25.3.2.1.3.1 = STRING: ARM Cortex-A53\r\n")
	agent := walkBackedPrinter(t, walk)

	if got := oidValueString(mustGet(t, agent, hrDeviceType+".1")); got != ".1.3.6.1.2.1.25.3.1.3" &&
		got != "1.3.6.1.2.1.25.3.1.3" {
		t.Errorf("the capture's processor row was overwritten: hrDeviceType.1 = %s", got)
	}
	if got := oidValueString(mustGet(t, agent, hrDeviceType+".2")); got != hrDeviceTypePrinter {
		t.Errorf("hrDeviceType.2 = %s, want the printer at the next free index", got)
	}
	if value := agent.mib.Get(prtGeneralPrinterName + ".2"); value == nil {
		t.Error("prtGeneralPrinterName is not keyed by the printer's hrDeviceIndex")
	}
}

// TestPrinterMIBNotSynthesizedOverACaptureThatHasIt: a real printer is the
// authority on its own Printer-MIB, as a PSE is on POWER-ETHERNET-MIB.
func TestPrinterMIBNotSynthesizedOverACaptureThatHasIt(t *testing.T) {
	walk := writeTestWalk(t, ""+
		".1.3.6.1.2.1.1.5.0 = STRING: HQ-PRN-03\r\n"+
		".1.3.6.1.2.1.43.5.1.1.16.1 = STRING: captured-name\r\n")
	agent := walkBackedPrinter(t, walk)

	if got := oidValueString(mustGet(t, agent, prtGeneralPrinterName+".1")); got != "captured-name" {
		t.Errorf("prtGeneralPrinterName.1 = %q, want the capture's own value", got)
	}
	if value := agent.mib.Get(prtGeneralReset + ".1"); value != nil {
		t.Errorf("synthesized a column the capture did not carry: %s", oidValueString(value))
	}
}

func writeTestWalk(t *testing.T, content string) string {
	t.Helper()
	walk := filepath.Join(t.TempDir(), "printer.walk")
	if err := os.WriteFile(walk, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return walk
}

func walkBackedPrinter(t *testing.T, walk string) *Agent {
	t.Helper()
	device := createTestDevice()
	device.Type = "printer"
	device.SNMPConfig.WalkFile = walk

	agent := NewAgent(device, 0)
	if value := agent.mib.Get(hrDeviceType + ".1"); value != nil {
		t.Fatalf("a walk-backed printer got a device row before its walk loaded: %s",
			oidValueString(value))
	}
	if err := agent.LoadWalkFile(walk); err != nil {
		t.Fatalf("LoadWalkFile() error = %v", err)
	}

	return agent
}
