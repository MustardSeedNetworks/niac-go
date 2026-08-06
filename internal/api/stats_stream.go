package api

import (
	"runtime"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api/sse"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
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

// sessionStatsPayload builds one named session's stats. Shared by the session
// stats endpoint and the stats publisher so both report the same numbers for
// the same session.
func (s *Server) sessionStatsPayload(session sessionRuntime) (statsPayload, bool) {
	s.configMu.RLock()
	version := s.cfg.Version
	s.configMu.RUnlock()

	deviceCount := 0
	if cfg := session.config(); cfg != nil {
		deviceCount = len(cfg.Devices)
	}
	return buildStatsPayload(session.stack(), session.iface(), version, deviceCount)
}

// selectedStatsPayload reads the process-wide projection that the unscoped
// endpoints still use. It exists only for those endpoints and goes away with
// them.
func (s *Server) selectedStatsPayload() (statsPayload, bool) {
	s.configMu.RLock()
	stack := s.cfg.Stack
	cfg := s.cfg.Config
	iface := s.cfg.Interface
	version := s.cfg.Version
	s.configMu.RUnlock()

	deviceCount := 0
	if cfg != nil {
		deviceCount = len(cfg.Devices)
	}
	return buildStatsPayload(stack, iface, version, deviceCount)
}

func buildStatsPayload(
	stack *protocols.Stack, iface, version string, deviceCount int,
) (statsPayload, bool) {
	if stack == nil {
		return statsPayload{}, false
	}

	stats := stack.GetStats()

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
	}, true
}

// startStatsPublisher emits one scoped tick per running session. It has no
// request to carry a session ID, so it iterates the sessions instead: with
// several running, publishing only one of them would leave every other
// session's subscribers on a silent stream.
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
			s.publishSessionStats()
		}
	}
}

func (s *Server) publishSessionStats() {
	for _, id := range s.sessionIDs() {
		session, err := s.session(id)
		if err != nil {
			continue
		}
		if payload, ok := s.sessionStatsPayload(session); ok {
			s.sseHub.BroadcastStatsForSession(id, payload)
		}
	}
	// Subscribers that have not adopted ?sessionId= yet only ever receive the
	// unscoped stream. Publishing the selected session there keeps them working
	// until the unscoped runtime surface is removed; drop this with it.
	if payload, ok := s.selectedStatsPayload(); ok {
		s.sseHub.BroadcastStats(payload)
	}
}
