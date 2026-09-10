package converter

// BehaviorTimeline is one saved, repeatable sequence of runtime phases.
type BehaviorTimeline struct {
	// Name identifies the timeline in the runtime view and the API. Unique
	// within the config.
	Name string `yaml:"name" validate:"required,max=100"`

	// StartOffsetMS delays the whole timeline this many milliseconds after
	// the session starts. 0 starts it immediately.
	StartOffsetMS int `yaml:"start_offset_ms,omitempty" validate:"gte=0,lte=86400000"`

	// RepeatCount is how many times the phase sequence runs, 1..1000. There
	// is no infinite repeat; pick a count that covers the intended run.
	RepeatCount int `yaml:"repeat_count" validate:"gte=1,lte=1000"`

	// Phases are the ordered steps of the timeline. Each phase's
	// start_offset_ms is relative to the timeline, not to the previous phase.
	Phases []BehaviorPhase `yaml:"phases" validate:"required,max=256,dive"`
}

// BehaviorPhase applies traffic, faults and one-shot operations at a deterministic offset.
type BehaviorPhase struct {
	// Name identifies the phase in the runtime view.
	Name string `yaml:"name" validate:"required,max=100"`

	// StartOffsetMS is when this phase begins, measured from the start of the
	// timeline (not from the previous phase).
	StartOffsetMS int `yaml:"start_offset_ms,omitempty" validate:"gte=0,lte=86400000"`

	// DurationMS is how long the phase's traffic and faults stay applied.
	DurationMS int `yaml:"duration_ms" validate:"gte=1,lte=86400000"`

	// Reset clears this phase's traffic and faults when its duration ends.
	Reset bool `yaml:"reset,omitempty"`

	// Traffic sets observable interface utilization for the phase's duration.
	Traffic []BehaviorTraffic `yaml:"traffic,omitempty" validate:"omitempty,max=1024,dive"`

	// Faults applies interface telemetry or device-service outcomes.
	Faults []BehaviorFault `yaml:"faults,omitempty" validate:"omitempty,max=1024,dive"`

	// Actions execute once on phase entry; reset does not undo them.
	Actions []BehaviorAction `yaml:"actions,omitempty" validate:"omitempty,max=1024,dive"`
}

// BehaviorAction performs a device operation without arming a fault.
type BehaviorAction struct {
	// Device names one uniquely identified simulated device in the configuration.
	Device string `yaml:"device" validate:"required"`
	// Type selects a one-shot operation; neither a fault value nor an interface applies.
	Type string `yaml:"type" validate:"required,oneof=reboot stp_topology_change"`
}

// BehaviorTraffic sets observable utilization on one simulated interface.
type BehaviorTraffic struct {
	// Device is the `name` of the device carrying the interface.
	Device string `yaml:"device" validate:"required"`

	// Interface is the interface `name` on that device, as declared in its
	// `interfaces` list.
	Interface string `yaml:"interface" validate:"required"`

	// Utilization is the percentage, 1..100, reported through the interface's
	// IF-MIB counters while the phase is active.
	Utilization int `yaml:"utilization" validate:"gte=1,lte=100"`
}

// BehaviorFault sets one supported fault for the phase's duration. The scope
// is the presence of `interface`: named, the fault is one interface's SNMP
// telemetry; omitted, it is a device-service outcome, which has no interface
// to be keyed by.
type BehaviorFault struct {
	// Device is the `name` of the device carrying the fault.
	Device string `yaml:"device" validate:"required"`

	// Interface is the interface `name` on that device, as declared in its
	// `interfaces` list. Omit it for a device-scoped service fault.
	Interface string `yaml:"interface,omitempty"`

	// Type is the fault to inject. Interface faults raise SNMP counters or
	// force a link down; device-scoped service outcomes omit `interface`.
	Type string `yaml:"type" validate:"required,oneof=fcs_errors packet_discards interface_errors high_utilization link_down dhcp_no_offer dns_nxdomain dns_timeout latency cpu_percent memory_percent disk_percent captive_portal duplicate_dhcp_offer duplicate_ip"`

	// Value is the rate, resource percentage, or latency in milliseconds. Link down
	// is an outcome: any accepted nonzero value enables it. The ceiling is
	// 100 for interface faults and service rates, 60000 for latency.
	Value *int `yaml:"value,omitempty" validate:"omitempty,gte=1,lte=60000"`

	// Address is the peer-owned unicast IPv4 address used by an addressed fault.
	// Address faults omit value; numeric faults omit address.
	Address *string `yaml:"address,omitempty" validate:"omitempty,ipv4" jsonschema:"format=ipv4"`
}
