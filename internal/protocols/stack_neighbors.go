package protocols

import (
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/deviceclass"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func (s *Stack) recordNeighbor(entry NeighborRecord) {
	if s.neighbors == nil {
		return
	}

	s.neighbors.upsert(entry)
}

func (s *Stack) startNeighborCleanupLoop() {
	if s.neighbors == nil {
		return
	}

	s.wg.Go(func() {
		ticker := time.NewTicker(stackNeighborCleanupSec * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.neighbors.cleanupExpired()
			case <-s.stopChan:
				return
			}
		}
	})
}

func (s *Stack) selectDiscoveryDevice(proto string) *config.Device {
	cfg := s.currentConfig()
	if cfg == nil {
		return nil
	}

	for i := range cfg.Devices {
		dev := &cfg.Devices[i]
		if s.isDeviceEnabledForProtocol(dev, proto) {
			return dev
		}
	}

	if len(cfg.Devices) > 0 {
		return &cfg.Devices[0]
	}

	return nil
}

// isDeviceEnabledForProtocol reports whether a device speaks a discovery
// protocol, which is what decides who owns an inbound neighbour.
//
// This used to read an absent config block as "enabled" for all four
// protocols. That was right before the emitters were tightened to stop every
// device announcing CDP — the change that closed the "every device emits CDP"
// bug on the Neighbors page — and wrong afterwards, because this predicate was
// left on the old rule. A workstation with no CDP block read as CDP-speaking
// here while never sending a CDP frame, and selectDiscoveryDevice returns the
// first match, so an inbound neighbour was attributed to it instead of to the
// switch that actually runs the protocol.
//
// The rules below are the emitters' rules, stated once. CDP, EDP and FDP are
// vendor protocols and opt in explicitly; LLDP is the IEEE standard and is on
// by default for the device types that ship it that way.
func (s *Stack) isDeviceEnabledForProtocol(dev *config.Device, proto string) bool {
	switch proto {
	case ProtocolLLDP:
		if dev.LLDPConfig != nil {
			return dev.LLDPConfig.Enabled
		}
		return deviceclass.RunsLLDPByDefault(deviceclass.Parse(dev.Type))
	case ProtocolCDP:
		return dev.CDPConfig != nil && dev.CDPConfig.Enabled
	case ProtocolEDP:
		return dev.EDPConfig != nil && dev.EDPConfig.Enabled
	case ProtocolFDP:
		return dev.FDPConfig != nil && dev.FDPConfig.Enabled
	default:
		return true
	}
}
