package protocols

import "github.com/MustardSeedNetworks/niac-go/internal/config"

func (s *Stack) validateDuplicateIPBinding(fault config.BehaviorFault) error {
	device, _, err := s.interfaceFaultTarget(fault.Device)
	if err != nil {
		return err
	}
	if s.fabric == nil {
		return nil
	}
	peer, found := s.fabric.interfacesByAddr[fault.Address]
	if !found || peer.device == device || peer.network != s.fabric.attachmentNetwork {
		return ErrFaultConflictAbsent
	}
	for _, endpoint := range s.fabric.interfacesByAddr {
		if endpoint.device == device && endpoint.interfaceName == fault.Interface &&
			endpoint.network == s.fabric.attachmentNetwork {
			return nil
		}
	}
	return ErrFaultConflictAbsent
}
