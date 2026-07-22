// SPDX-License-Identifier: BUSL-1.1

package api

// handlers_license.go exposes the active license state to the UI so
// the React layer can render TierGate / RequireFeature primitives
// without needing to hit a gated endpoint and parse a 402.

import (
	"net/http"
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/license"
)

// LicenseFeature is a single entry in LicenseStatusResponse.Features — the
// Pro feature catalog paired with whether the active license grants it.
// Label and Description come from license.FeatureCatalog() and are
// English-only product copy (see that package for the i18n boundary).
type LicenseFeature struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	// Granted is true when the active license includes this feature.
	Granted bool `json:"granted"`
}

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
	// Features is the full Pro feature catalog (license.FeatureCatalog()),
	// each entry carrying a human-readable label + description plus
	// whether the active license grants it. Always non-empty — even on
	// Free tier — so the UI can show what upgrading unlocks instead of
	// just what's currently active.
	Features []LicenseFeature `json:"features"`
	// LicenseEnforced reports that paid feature gates are active. It remains
	// true when manager initialization fails because those gates fail closed.
	LicenseEnforced bool `json:"licenseEnforced"`
}

// handleLicenseStatus serves GET /api/v1/license. No auth gate beyond
// the shared session auth that the route is already wrapped in — the
// UI needs to know the tier on every page, including the login-
// adjacent ones, so this stays on the cheapest path.
func (s *Server) handleLicenseStatus(w http.ResponseWriter, _ *http.Request) {
	resp := LicenseStatusResponse{
		Tier:            license.TierFree.String(),
		LicenseEnforced: true,
	}

	var granted []string
	if s.license != nil {
		resp.IsActivated = s.license.IsActivated()
		if st := s.license.GetState(); st != nil {
			resp.Tier = license.Tier(st.Tier).String()
			resp.IsTrialMode = st.IsTrialMode
			granted = st.Features
		}
		resp.TrialDaysRemaining = s.license.TrialDaysRemaining()
	}

	catalog := license.FeatureCatalog()
	resp.Features = make([]LicenseFeature, 0, len(catalog))
	for _, meta := range catalog {
		resp.Features = append(resp.Features, LicenseFeature{
			ID:          meta.ID,
			Label:       meta.Label,
			Description: meta.Description,
			Granted:     slices.Contains(granted, meta.ID),
		})
	}

	s.writeJSON(w, resp)
}
