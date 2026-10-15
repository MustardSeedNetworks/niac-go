package scenario

import (
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

// packSIP answers SIP OPTIONS on the desk phones. Without it a scanner sees a
// host with a Cisco MAC prefix; with it, the phone names its model and load
// the way the handset does ("Cisco-CP8841/12.8.1").
func packSIP(role string, profile DeviceProfile) *converter.SIPConfig {
	if role != "voip-phone" || profile.Vendor != "cisco" {
		return nil
	}
	_, load, _ := strings.Cut(profile.Software, ".")
	userAgent := "Cisco-" + strings.ReplaceAll(profile.Model, "-", "") + "/" + strings.ReplaceAll(load, "-", ".")

	return &converter.SIPConfig{Enabled: true, UserAgent: userAgent}
}
