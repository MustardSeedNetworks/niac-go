package api

import (
	"runtime"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api/sse"
)

const statsStreamInterval = time.Second

type statsPayload struct {
	Timestamp   time.Time         `json:"timestamp"`
	Interface   string            `json:"interface"`
	Version     string            `json:"version"`
	DeviceCount int               `json:"deviceCount"`
	Goroutines  int               `json:"goroutines"`
	Stack       statsStackPayload `json:"stack"`
}

type statsStackPayload struct {
	PacketsSent           uint64 `json:"packetsSent"`
	PacketsReceived       uint64 `json:"packetsReceived"`
	ARPRequests           uint64 `json:"arpRequests"`
	ARPReplies            uint64 `json:"arpReplies"`
	ICMPRequests          uint64 `json:"icmpRequests"`
	ICMPReplies           uint64 `json:"icmpReplies"`
	DNSQueries            uint64 `json:"dnsQueries"`
	DHCPRequests          uint64 `json:"dhcpRequests"`
	SNMPQueries           uint64 `json:"snmpQueries"`
	Errors                uint64 `json:"errors"`
	UDPProxyOverloadDrops uint64 `json:"udpProxyOverloadDrops"`
}

func (s *Server) currentStatsPayload() (statsPayload, string, bool) {
	s.configMu.RLock()
	stack := s.cfg.Stack
	cfg := s.cfg.Config
	iface := s.cfg.Interface
	version := s.cfg.Version
	sessionID := s.selectedSimulation
	s.configMu.RUnlock()

	if stack == nil {
		return statsPayload{}, "", false
	}

	stats := stack.GetStats()
	deviceCount := 0
	if cfg != nil {
		deviceCount = len(cfg.Devices)
	}

	return statsPayload{
		Timestamp:   time.Now().UTC(),
		Interface:   iface,
		Version:     version,
		DeviceCount: deviceCount,
		Goroutines:  runtime.NumGoroutine(),
		Stack: statsStackPayload{
			PacketsSent:           stats.PacketsSent,
			PacketsReceived:       stats.PacketsReceived,
			ARPRequests:           stats.ARPRequests,
			ARPReplies:            stats.ARPReplies,
			ICMPRequests:          stats.ICMPRequests,
			ICMPReplies:           stats.ICMPReplies,
			DNSQueries:            stats.DNSQueries,
			DHCPRequests:          stats.DHCPRequests,
			SNMPQueries:           stats.SNMPQueries,
			Errors:                stats.Errors,
			UDPProxyOverloadDrops: stats.UDPProxyOverloadDrops,
		},
	}, sessionID, true
}

func (s *Server) startStatsPublisher(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.bgStop:
			return
		case <-ticker.C:
			if s.sseHub.ClientCount(sse.StreamStats) == 0 {
				continue
			}
			payload, sessionID, ok := s.currentStatsPayload()
			if ok {
				if sessionID == "" {
					s.sseHub.BroadcastStats(payload)
				} else {
					s.sseHub.BroadcastStatsForSession(sessionID, payload)
				}
			}
		}
	}
}
