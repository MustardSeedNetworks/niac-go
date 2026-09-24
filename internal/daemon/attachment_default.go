package daemon

import (
	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// withDefaultBinding fills an unset attachment mode from the operator policy
// when that policy approves exactly one binding on the requested interface.
// A VLAN the caller did give is kept for the policy check to judge. Any choice
// -- several policies, a trunk carrying several VLANs and no VLAN named, none
// at all -- is left to the caller, and preflight reports the mode as required
// rather than guessing.
func (d *Daemon) withDefaultBinding(req api.SimulationRequest) api.SimulationRequest {
	if req.AttachmentMode != "" {
		return req
	}
	var only *fabric.PhysicalAttachmentPolicy
	for i := range d.cfg.AttachmentPolicies {
		policy := &d.cfg.AttachmentPolicies[i]
		if policy.Interface != req.Interface {
			continue
		}
		if only != nil {
			return req
		}
		only = policy
	}
	if only == nil {
		return req
	}
	if req.AccessVLAN == 0 {
		vlan, ok := onlyVLAN(*only)
		if !ok {
			return req
		}
		req.AccessVLAN = vlan
	}
	req.AttachmentMode = only.Mode
	return req
}

// onlyVLAN is the one VLAN a policy approves; a trunk carrying several has none.
func onlyVLAN(policy fabric.PhysicalAttachmentPolicy) (uint16, bool) {
	if policy.Mode != fabric.ModeTrunk {
		return policy.AccessVLAN, true
	}
	if len(policy.AllowedVLANs) != 1 {
		return 0, false
	}
	return policy.AllowedVLANs[0], true
}
