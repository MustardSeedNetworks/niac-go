package snmp

import (
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/deviceclass"
)

// HOST-RESOURCES-MIB (RFC 2790) device and printer tables. Printer-MIB keys
// every table by hrDeviceIndex, so a printer exists here before it exists
// there.
const (
	hrDeviceEntry  = "1.3.6.1.2.1.25.3.2.1"
	hrDeviceIndex  = hrDeviceEntry + ".1"
	hrDeviceType   = hrDeviceEntry + ".2"
	hrDeviceDescr  = hrDeviceEntry + ".3"
	hrDeviceID     = hrDeviceEntry + ".4"
	hrDeviceStatus = hrDeviceEntry + ".5"
	hrDeviceErrors = hrDeviceEntry + ".6"

	hrPrinterEntry              = "1.3.6.1.2.1.25.3.5.1"
	hrPrinterStatus             = hrPrinterEntry + ".1"
	hrPrinterDetectedErrorState = hrPrinterEntry + ".2"

	hrDeviceTypePrinter = "1.3.6.1.2.1.25.3.1.5"
	// hrDeviceID is zeroDotZero when no product ID is known, which RFC 2790
	// names as the value to use.
	zeroDotZero = "0.0"

	hrDeviceStatusRunning = 2
	hrPrinterStatusIdle   = 3
)

// Printer-MIB (RFC 3805) general table. Only columns whose value NIAC can say
// truthfully are served: the ones that point into tables NIAC does not model
// (localization, console, covers) are left out rather than left dangling.
const (
	printerMIBObjects       = "1.3.6.1.2.1.43"
	prtGeneralEntry         = printerMIBObjects + ".5.1.1"
	prtGeneralConfigChanges = prtGeneralEntry + ".1"
	prtGeneralReset         = prtGeneralEntry + ".3"
	prtGeneralPrinterName   = prtGeneralEntry + ".16"

	prtGeneralResetNotResetting = 3
)

// initializePrinterMIB registers the printer's HOST-RESOURCES device row and
// Printer-MIB general row. A walk-backed printer waits for its walk, which may
// carry the MIB itself and owns the hrDeviceIndex space either way.
func (a *Agent) initializePrinterMIB() {
	if !a.isPrinter() || a.hasWalkContent() {
		return
	}

	a.registerPrinterMIB()
}

// refreshWalkedPrinterMIB serves a printer whose capture carried no Printer-MIB.
// One that did keeps it untouched: a real printer is the authority on its own
// tables, as a PSE is on POWER-ETHERNET-MIB.
func (a *Agent) refreshWalkedPrinterMIB(walkOwnsPrinter bool) {
	if !a.isPrinter() || walkOwnsPrinter {
		return
	}

	a.registerPrinterMIB()
}

func (a *Agent) isPrinter() bool {
	return a.device != nil && deviceclass.Parse(a.device.Type) == deviceclass.Printer
}

// walkOwnsPrinter reports whether a parsed capture carries Printer-MIB of its
// own. Any object under the MIB counts.
func walkOwnsPrinter(entries []WalkEntry) bool {
	for _, entry := range entries {
		if strings.HasPrefix(strings.TrimPrefix(entry.OID, "."), printerMIBObjects+".") {
			return true
		}
	}

	return false
}

// registerPrinterMIB names the printer the way the system group already does,
// so a walk-backed printer reports its captured sysDescr and sysName here too.
func (a *Agent) registerPrinterMIB() {
	index := a.nextHrDeviceIndex()
	suffix := "." + strconv.Itoa(index)
	descr := oidValueString(a.mib.Get("1.3.6.1.2.1.1.1.0"))
	name := oidValueString(a.mib.Get("1.3.6.1.2.1.1.5.0"))

	a.mib.Set(hrDeviceIndex+suffix, &OIDValue{Type: gosnmp.Integer, Value: index})
	a.mib.Set(hrDeviceType+suffix, &OIDValue{Type: gosnmp.ObjectIdentifier, Value: hrDeviceTypePrinter})
	a.mib.Set(hrDeviceDescr+suffix, &OIDValue{Type: gosnmp.OctetString, Value: descr})
	a.mib.Set(hrDeviceID+suffix, &OIDValue{Type: gosnmp.ObjectIdentifier, Value: zeroDotZero})
	a.mib.Set(hrDeviceStatus+suffix, &OIDValue{Type: gosnmp.Integer, Value: hrDeviceStatusRunning})
	a.mib.Set(hrDeviceErrors+suffix, &OIDValue{Type: gosnmp.Counter32, Value: uint32(0)})

	a.mib.Set(hrPrinterStatus+suffix, &OIDValue{Type: gosnmp.Integer, Value: hrPrinterStatusIdle})
	// One octet with no bit set: no error condition detected.
	a.mib.Set(hrPrinterDetectedErrorState+suffix, &OIDValue{Type: gosnmp.OctetString, Value: []byte{0}})

	a.mib.Set(prtGeneralConfigChanges+suffix, &OIDValue{Type: gosnmp.Counter32, Value: uint32(0)})
	a.mib.Set(prtGeneralReset+suffix, &OIDValue{Type: gosnmp.Integer, Value: prtGeneralResetNotResetting})
	a.mib.Set(prtGeneralPrinterName+suffix, &OIDValue{Type: gosnmp.OctetString, Value: name})
}

// nextHrDeviceIndex returns the first index above every device row already in
// the MIB: 1 on a synthesized agent, the next free one after a capture's own
// processors, disks and network devices.
func (a *Agent) nextHrDeviceIndex() int {
	highest := 0
	prefix := hrDeviceType + "."
	for _, oid := range a.mib.AllOIDs() {
		index, err := strconv.Atoi(strings.TrimPrefix(oid, prefix))
		if strings.HasPrefix(oid, prefix) && err == nil && index > highest {
			highest = index
		}
	}

	return highest + 1
}
