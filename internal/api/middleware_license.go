// SPDX-License-Identifier: BUSL-1.1

package api

// middleware_license.go provides per-route license-tier gating for NIAC.
// Wrap any handler that exposes a paid feature with s.requireFeature
// so the route returns 402 Payment Required when the active license
// doesn't cover the feature. Mirrors seed's gating framework.

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/MustardSeedNetworks/niac-go/internal/license"
)

// errCodeTierTooLow is the standard error code for routes blocked by
// the license tier. UIs key off this code to render the upgrade hint.
const errCodeTierTooLow = "TIER_TOO_LOW"

// FeatureGateResponse is the JSON body returned with a 402 Payment
// Required when a request hits a feature it isn't licensed for.
type FeatureGateResponse struct {
	Error           string `json:"error"`
	Code            string `json:"code"`
	RequiredFeature string `json:"requiredFeature"`
	CurrentTier     string `json:"currentTier"`
	UpgradeMessage  string `json:"upgradeMessage"`
}

// defaultUpgradeMessage is the standard upgrade hint surfaced when a
// caller doesn't supply a feature-specific message.
const defaultUpgradeMessage = "Start a 14-day Pro trial with `niac license trial` " +
	"or activate a Pro key with `niac license activate -k <KEY>`."

// writeFeatureGate writes the canonical 402 Payment Required response
// for a feature-gated request. Shared between requireFeature (route-
// level) and handler-level soft caps (e.g. device count). Pass an
// empty upgradeMessage to use defaultUpgradeMessage.
func (s *Server) writeFeatureGate(
	w http.ResponseWriter, r *http.Request,
	feature, upgradeMessage string,
) {
	tierName := license.TierFree.String()
	if s.license != nil {
		if st := s.license.GetState(); st != nil {
			tierName = st.Tier.String()
		}
	}
	if upgradeMessage == "" {
		upgradeMessage = defaultUpgradeMessage
	}

	resp := FeatureGateResponse{
		Error:           "Feature requires the Pro tier",
		Code:            errCodeTierTooLow,
		RequiredFeature: feature,
		CurrentTier:     tierName,
		UpgradeMessage:  upgradeMessage,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)
	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
		slog.ErrorContext(r.Context(),
			"failed to encode feature-gate response", "error", encErr)
	}
}

// requireFeature wraps `next` so it only runs when the active license
// includes `feature`. Returns 402 Payment Required with a structured
// upgrade hint otherwise.
//
// A nil license manager is treated as "license disabled" (dev / test
// builds) and permits all features so the gating layer never breaks
// local workflows.
func (s *Server) requireFeature(feature string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.license == nil || s.license.HasFeature(feature) {
			next(w, r)
			return
		}
		s.writeFeatureGate(w, r, feature, "")
	}
}
