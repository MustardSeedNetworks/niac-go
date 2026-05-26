// SPDX-License-Identifier: BUSL-1.1

package api

// handlers_license.go exposes the active license state to the UI so
// the React layer can render TierGate / RequireFeature primitives
// without needing to hit a gated endpoint and parse a 402.

import (
	"net/http"

	"github.com/krisarmstrong/niac-go/internal/license"
)

// LicenseStatusResponse is the JSON shape returned by
// GET /api/v1/license. It is intentionally read-only — activation,
// trial start, and deactivation go through the `niac license` CLI,
// not the HTTP API, so the only thing the UI needs is the current
// state.
type LicenseStatusResponse struct {
	// Tier is the human-readable tier name ("Free" / "Pro" /
	// "Invalid"). When the license manager is disabled (dev / test
	// builds), this is "Free" so the UI degrades gracefully.
	Tier string `json:"tier"`
	// IsActivated is true when either an activated key OR a valid
	// trial is in effect.
	IsActivated bool `json:"isActivated"`
	// IsTrialMode reports whether the active license is a trial.
	IsTrialMode bool `json:"isTrialMode"`
	// TrialDaysRemaining is the days left in the current trial; 0
	// when no trial is active.
	TrialDaysRemaining int `json:"trialDaysRemaining"`
	// Features is the active feature set keyed by the keygen
	// productCatalog feature names. Empty slice (not nil) when the
	// license is not activated, so the UI can iterate without a nil
	// check.
	Features []string `json:"features"`
	// LicenseEnforced is false when the server is running without a
	// license manager (dev / test builds). The UI should treat
	// LicenseEnforced=false as "all features available, hide upgrade
	// hints" so local development isn't cluttered with paid-tier
	// prompts.
	LicenseEnforced bool `json:"licenseEnforced"`
}

// handleLicenseStatus serves GET /api/v1/license. No auth gate beyond
// the shared session auth that the route is already wrapped in — the
// UI needs to know the tier on every page, including the login-
// adjacent ones, so this stays on the cheapest path.
func (s *Server) handleLicenseStatus(w http.ResponseWriter, _ *http.Request) {
	resp := LicenseStatusResponse{
		Tier:            license.TierFree.String(),
		Features:        []string{},
		LicenseEnforced: s.license != nil,
	}

	if s.license != nil {
		resp.IsActivated = s.license.IsActivated()
		if st := s.license.GetState(); st != nil {
			resp.Tier = st.Tier.String()
			resp.IsTrialMode = st.IsTrialMode
			if st.Features != nil {
				resp.Features = st.Features
			}
		}
		resp.TrialDaysRemaining = s.license.TrialDaysRemaining()
	}

	s.writeJSON(w, resp)
}
