package protocols

import (
	"time"

	"github.com/krisarmstrong/niac-go/internal/config"
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

// isDeviceEnabledForProtocol checks if a device is enabled for a discovery protocol.
func (s *Stack) isDeviceEnabledForProtocol(dev *config.Device, proto string) bool {
	switch proto {
	case ProtocolLLDP:
		return dev.LLDPConfig == nil || dev.LLDPConfig.Enabled
	case ProtocolCDP:
		return dev.CDPConfig == nil || dev.CDPConfig.Enabled
	case ProtocolEDP:
		return dev.EDPConfig == nil || dev.EDPConfig.Enabled
	case ProtocolFDP:
		return dev.FDPConfig == nil || dev.FDPConfig.Enabled
	default:
		return true
	}
}
