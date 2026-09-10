package config

import (
	"net/netip"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// BehaviorTimeline is one saved sequence replayed from simulation start.
type BehaviorTimeline struct {
	Name        string
	StartOffset time.Duration
	RepeatCount int
	Phases      []BehaviorPhase
}

// BehaviorPhase applies traffic and faults for one interval, with actions at entry.
type BehaviorPhase struct {
	Name        string
	StartOffset time.Duration
	Duration    time.Duration
	Reset       bool
	Traffic     []BehaviorTraffic
	Faults      []BehaviorFault
	Actions     []BehaviorAction
}

// BehaviorAction executes one device operation at phase entry.
type BehaviorAction struct {
	Device string
	Type   devicestate.DeviceActionType
}

// BehaviorTraffic sets observable utilization on one interface.
type BehaviorTraffic struct {
	Device      string
	Interface   string
	Utilization int
}

// BehaviorFault sets one supported interface or device-service outcome.
type BehaviorFault struct {
	Device    string
	Interface string
	Type      string
	Value     int
	Address   netip.Addr
}
