package scenario

import (
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// packJetDirect opens the raw print port on every printer, which the printers
// already advertise as _pdl-datastream._tcp on 9100. Only the HP speaks PJL,
// and it names itself the way FutureSmart firmware does, without the
// "Enterprise" of the marketing name; the Zebra ZPL and Epson ESC/POS printers
// take raw jobs but answer no PJL query.
func packJetDirect(profile DeviceProfile) *converter.JetDirectConfig {
	if profile.DeviceType != "printer" {
		return nil
	}
	jetDirect := &converter.JetDirectConfig{Enabled: true}
	if profile.Vendor == "hewlett packard" {
		jetDirect.PJLID = "HP " + strings.Replace(profile.Model, "Enterprise ", "", 1)
	}

	return jetDirect
}
