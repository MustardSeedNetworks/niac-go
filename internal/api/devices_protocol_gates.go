// SPDX-License-Identifier: BUSL-1.1

package api

// devices_protocol_gates.go enforces per-protocol license features on
// device create and update. The gating runs after the device has been
// parsed (so we know which protocols it actually configures) but
// before save, returning 402 + FeatureGateResponse rather than the
// usual 400 validation error so UIs can render an upgrade hint.

import (
	"net/http"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// protocolFeatureCheck pairs a per-device protocol detector with its
// canonical license feature name from the keygen productCatalog.
type protocolFeatureCheck struct {
	// feature is the keygen feature key (e.g. "stp", "ftp").
	feature string
	// present reports whether the device configures the protocol.
	present func(*config.Device) bool
}

// gatedDeviceProtocolChecks enumerates the protocols that exist in the
// current Device struct and have a matching license feature. BGP,
// OSPF, and SNMPv3 are in the licensed feature catalog but not yet
// modeled on Device — tracked separately for follow-up.
//
// Returned by a function rather than held in a package-level slice so
// gochecknoglobals stays clean; the closures aren't worth caching.
func gatedDeviceProtocolChecks() []protocolFeatureCheck {
	return []protocolFeatureCheck{
		{
			feature: "stp",
			present: func(d *config.Device) bool { return d != nil && d.STPConfig != nil },
		},
		{
			feature: "ftp",
			present: func(d *config.Device) bool { return d != nil && d.FTPConfig != nil },
		},
		{
			feature: "netbios",
			present: func(d *config.Device) bool { return d != nil && d.NetBIOSConfig != nil },
		},
	}
}

// requireDeviceProtocolFeatures returns false (and writes a 402
// FeatureGateResponse) when the device's protocol set includes a
// protocol the active license doesn't cover. Nil license manager
// = pass (dev / test builds).
func (s *Server) requireDeviceProtocolFeatures(
	w http.ResponseWriter, r *http.Request, dev *config.Device,
) bool {
	if s.license == nil || dev == nil {
		return true
	}
	for _, chk := range gatedDeviceProtocolChecks() {
		if !chk.present(dev) {
			continue
		}
		if !s.license.HasFeature(chk.feature) {
			s.writeFeatureGate(w, r, chk.feature,
				"This device uses "+chk.feature+
					", which requires the Pro tier. "+
					"Start a 14-day Pro trial with `niac license trial`.")
			return false
		}
	}
	return true
}
