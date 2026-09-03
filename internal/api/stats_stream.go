package api

import (
	"runtime"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

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

// sessionStatsPayload builds one named session's stats for the session
// stats endpoint.
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
