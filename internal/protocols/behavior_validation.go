package protocols

import (
	"fmt"
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func (s *Stack) validateBehaviorAddressFaults(faults []config.BehaviorFault) error {
	for _, fault := range faults {
		if fault.Type == string(devicestate.FaultDuplicateIP) {
			if err := s.validateDuplicateIPBinding(fault); err != nil {
				return err
			}
			continue
		}
		if fault.Type != string(devicestate.FaultDuplicateDHCPOffer) {
			continue
		}
		device, _, err := s.interfaceFaultTarget(fault.Device)
		if err != nil {
			return err
		}
		if s.dhcpHandlers[device] == nil {
			return fmt.Errorf("device %s: %w", fault.Device, ErrFaultServiceAbsent)
		}
		if s.fabric != nil {
			// Binding eligibility is immutable; persisted link faults must not block recovery.
			peer, found := s.fabric.interfacesByAddr[fault.Address]
			if !found || peer.device == device || peer.network != s.fabric.attachmentNetwork ||
				!slices.Contains(s.fabric.attachmentDHCP, device) {
				return fmt.Errorf("device %s: %w", fault.Device, ErrFaultConflictAbsent)
			}
		}
	}
	return nil
}

// ValidateConfiguredBehaviorTargets checks observable targets before acquiring packet I/O.
func ValidateConfiguredBehaviorTargets(cfg *config.Config, topology *fabric.Topology) error {
	for _, timeline := range cfg.BehaviorTimelines {
		for _, phase := range timeline.Phases {
			hasAddressFault := slices.ContainsFunc(phase.Faults, func(fault config.BehaviorFault) bool {
				return fault.Type == string(devicestate.FaultDuplicateDHCPOffer) ||
					fault.Type == string(devicestate.FaultDuplicateIP)
			})
			if len(phase.Actions) > 0 || hasAddressFault {
				stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
				stack.ConfigureFabric(topology)
				return stack.ValidateBehaviorTargets()
			}
		}
	}
	return nil
}
