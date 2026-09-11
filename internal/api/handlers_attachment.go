package api

import "net/http"

// AttachmentPolicy is one operator-approved physical binding, as the daemon
// was started with it (--attachment-policy).
//
// The wire shape is named for the request it approves, not for the compiled
// binding it produces: a caller reads a policy to fill in SimulationRequest's
// interface / attachmentMode / accessVlan, so those are the names it carries.
type AttachmentPolicy struct {
	Interface string `json:"interface"`
	Mode      string `json:"mode"`
	// AccessVLAN is the one approved VLAN in access mode. Direct mode carries
	// no VLAN and a trunk's approved set is AllowedVLANs, so both leave it 0.
	AccessVLAN uint16 `json:"accessVlan,omitempty"`
	// AllowedVLANs is the trunk's approved tag set; empty in every other mode.
	AllowedVLANs []uint16 `json:"allowedVlans,omitempty"`
}

// AttachmentPoliciesResponse lists every binding the operator approved.
type AttachmentPoliciesResponse struct {
	Policies []AttachmentPolicy `json:"policies"`
}

// SimulationAttachments names the logical attachments a prepared configuration
// declares, so a caller offers the operator the scenario's own names instead of
// guessing one. Routed distinguishes a scenario with no attachments from a flat
// one, which has no binding to choose at all.
type SimulationAttachments struct {
	Routed      bool     `json:"routed"`
	Attachments []string `json:"attachments"`
}

// handleAttachmentPolicies publishes the operator's approved physical
// bindings. Without it a client has to guess a mode and a VLAN, and a guess
// that happens to match is indistinguishable from one the operator sanctioned.
func (s *Server) handleAttachmentPolicies(w http.ResponseWriter, r *http.Request) {
	if s.daemon == nil {
		writeError(w, r, http.StatusNotImplemented, "daemon_required",
			"Daemon mode is required", nil)
		return
	}
	policies := s.daemon.AttachmentPolicies()
	response := AttachmentPoliciesResponse{Policies: make([]AttachmentPolicy, 0, len(policies))}
	for _, policy := range policies {
		response.Policies = append(response.Policies, AttachmentPolicy{
			Interface:    policy.Interface,
			Mode:         string(policy.Mode),
			AccessVLAN:   policy.AccessVLAN,
			AllowedVLANs: policy.AllowedVLANs,
		})
	}
	s.writeJSON(w, response)
}

// handleSimulationAttachments answers which logical attachments the prepared
// configuration declares. It changes no runtime state; it takes a body because
// the configuration is identified the same way a start identifies it.
func (s *Server) handleSimulationAttachments(w http.ResponseWriter, r *http.Request) {
	if s.daemon == nil {
		writeError(w, r, http.StatusNotImplemented, "daemon_required",
			"Daemon mode is required", nil)
		return
	}
	var req SimulationRequest
	if !decodeJSONStrict(w, r, &req, MaxScenarioRequestBodySize) {
		return
	}
	attachments, err := s.daemon.SimulationAttachments(req)
	if err != nil {
		if writeManagedConfigPathError(w, r, err) {
			return
		}
		s.logger.ErrorContext(r.Context(), "[API] Attachment listing failed", "error", err)
		writeError(w, r, http.StatusBadRequest, "config_load_failed",
			"Configuration could not be read", preflightErrorDetails(err))
		return
	}
	if attachments.Attachments == nil {
		attachments.Attachments = []string{}
	}
	s.writeJSON(w, attachments)
}
