package daemon

import (
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// PreflightSimulation compiles a routed request without opening capture or changing runtime state.
func (d *Daemon) PreflightSimulation(req api.SimulationRequest) (fabric.Report, error) {
	if diagnostic := simulationInterfaceDiagnostic(req.Interface, e2eDryRunSimulation()); diagnostic != nil {
		report := fabric.NewReport()
		report.Diagnostics = []fabric.Diagnostic{*diagnostic}
		return report, nil
	}
	cfg, _, err := loadValidSimulationConfig(req, false)
	if err != nil {
		return fabric.NewReport(), err
	}
	if !fabric.IsRouted(cfg) {
		if err = protocols.ValidateConfiguredBehaviorTargets(cfg, nil); err != nil {
			return fabric.NewReport(), err
		}
		if req.AttachmentMode == fabric.ModeTrunk {
			return fabric.CompilePhysicalBinding(d.bindingFromRequest(req)), nil
		}
		report := fabric.NewReport()
		report.Safe = true
		return report, nil
	}
	report := fabric.Compile(cfg, d.bindingFromRequest(req))
	if report.Safe {
		if err = protocols.ValidateConfiguredBehaviorTargets(cfg, &report.Topology); err != nil {
			return fabric.NewReport(), err
		}
	}
	return report, nil
}

// AttachmentPolicies returns the operator-approved physical bindings this
// daemon was started with. They are fixed for the daemon's life; the clone
// keeps a caller from editing what the daemon will approve.
func (d *Daemon) AttachmentPolicies() []fabric.PhysicalAttachmentPolicy {
	return slices.Clone(d.cfg.AttachmentPolicies)
}

// SimulationAttachments names the logical attachments a prepared configuration
// declares. A start binds one of them by name, and before this the only way to
// learn a name was to guess it and read the unknown_attachment diagnostic back.
func (d *Daemon) SimulationAttachments(
	req api.SimulationRequest,
) (api.SimulationAttachments, error) {
	cfg, _, err := loadValidSimulationConfig(req, false)
	if err != nil {
		return api.SimulationAttachments{}, err
	}
	attachments := api.SimulationAttachments{
		Routed:      fabric.IsRouted(cfg),
		Attachments: make([]string, 0, len(cfg.Attachments)),
	}
	for _, attachment := range cfg.Attachments {
		attachments.Attachments = append(attachments.Attachments, attachment.Name)
	}
	return attachments, nil
}
