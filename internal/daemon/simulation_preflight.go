package daemon

import (
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
	if err = protocols.ValidateConfiguredBehaviorActions(cfg); err != nil {
		return fabric.NewReport(), err
	}
	if !fabric.IsRouted(cfg) {
		if req.AttachmentMode == fabric.ModeTrunk {
			return fabric.CompilePhysicalBinding(d.bindingFromRequest(req)), nil
		}
		report := fabric.NewReport()
		report.Safe = true
		return report, nil
	}
	return fabric.Compile(cfg, d.bindingFromRequest(req)), nil
}
