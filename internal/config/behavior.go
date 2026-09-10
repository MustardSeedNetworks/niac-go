package config

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// Sentinel errors returned by behavior timeline validation, matched by
// callers via errors.Is.
var (
	// ErrBehaviorTargetNotFound means a timeline names a device or interface
	// absent from the config.
	ErrBehaviorTargetNotFound = errors.New("behavior target not found")
	// ErrBehaviorTargetAmbiguous means a timeline's target string matches more
	// than one device/interface.
	ErrBehaviorTargetAmbiguous = errors.New("behavior target is ambiguous")
	// ErrBehaviorPhaseEmpty means a phase declares no traffic, faults or actions.
	ErrBehaviorPhaseEmpty = errors.New("behavior phase has no traffic, faults or actions")
	// ErrBehaviorPhaseOverlap means two phases for the same target overlap in time.
	ErrBehaviorPhaseOverlap = errors.New("behavior phases overlap")
	// ErrBehaviorScheduleTooLarge means the timeline would exceed
	// maxBehaviorScheduledActions once expanded into discrete scheduled actions.
	ErrBehaviorScheduleTooLarge = errors.New("behavior schedule is too large")
	// ErrBehaviorFaultScope means a phase fault names an interface for a
	// device-scoped fault, or omits one for an interface-scoped fault.
	ErrBehaviorFaultScope = errors.New("behavior fault scope does not match its type")
	// ErrBehaviorFaultValue means a phase fault's value exceeds the ceiling
	// its own type carries.
	ErrBehaviorFaultValue = errors.New("behavior fault value is out of range")
)

const maxBehaviorScheduledActions = 100_000

// interfaceFaultMax is the ceiling every interface-scoped fault shares: they
// are all rates or percentages.
const interfaceFaultMax = 100

type behaviorTarget struct {
	device     Device
	count      int
	interfaces map[string]struct{}
}

func validateBehaviorTargets(cfg *Config) error {
	targets := behaviorTargets(cfg)
	for _, timeline := range cfg.BehaviorTimelines {
		if err := validateBehaviorTimeline(timeline, targets); err != nil {
			return fmt.Errorf("behavior timeline %q: %w", timeline.Name, err)
		}
	}
	if err := validateBehaviorScheduleSize(cfg.BehaviorTimelines); err != nil {
		return err
	}
	return validateBehaviorConflicts(cfg.BehaviorTimelines)
}

type behaviorInterval struct {
	start         time.Duration
	end           time.Duration
	persistent    bool
	timelineIndex int
	timeline      string
	phase         string
}

func validateBehaviorScheduleSize(timelines []BehaviorTimeline) error {
	var scheduledActions int64
	for _, timeline := range timelines {
		for _, phase := range timeline.Phases {
			applications := int64(len(phase.Traffic) + len(phase.Faults))
			if phase.Reset {
				applications *= 2
			}
			applications += int64(len(phase.Actions))
			scheduledActions += applications * int64(timeline.RepeatCount)
			if scheduledActions > maxBehaviorScheduledActions {
				return fmt.Errorf(
					"%w: maximum is %d action applications",
					ErrBehaviorScheduleTooLarge,
					maxBehaviorScheduledActions,
				)
			}
		}
	}
	return nil
}

func validateBehaviorConflicts(timelines []BehaviorTimeline) error {
	for _, targetIntervals := range behaviorIntervalsByTarget(timelines) {
		if err := validateBehaviorTargetIntervals(targetIntervals, len(timelines)); err != nil {
			return err
		}
	}
	return nil
}

func behaviorIntervalsByTarget(timelines []BehaviorTimeline) map[string][]behaviorInterval {
	intervalsByTarget := make(map[string][]behaviorInterval)
	for timelineIndex, timeline := range timelines {
		cycleDuration := behaviorTimelineDuration(timeline.Phases)
		for _, phase := range timeline.Phases {
			keys := behaviorPhaseTargetKeys(phase)
			repetitions := timeline.RepeatCount
			if !phase.Reset {
				repetitions = 1
			}
			for repetition := range repetitions {
				start := timeline.StartOffset + time.Duration(
					repetition,
				)*cycleDuration + phase.StartOffset
				interval := behaviorInterval{
					start: start, end: start + phase.Duration, persistent: !phase.Reset,
					timelineIndex: timelineIndex, timeline: timeline.Name, phase: phase.Name,
				}
				for key := range keys {
					intervalsByTarget[key] = append(intervalsByTarget[key], interval)
				}
			}
		}
	}
	return intervalsByTarget
}

func behaviorPhaseTargetKeys(phase BehaviorPhase) map[string]struct{} {
	keys := make(map[string]struct{}, len(phase.Traffic)+len(phase.Faults))
	for _, traffic := range phase.Traffic {
		keys[behaviorConflictKey(traffic.Device, traffic.Interface, "high_utilization")] = struct{}{}
	}
	for _, fault := range phase.Faults {
		keys[behaviorConflictKey(fault.Device, fault.Interface, fault.Type)] = struct{}{}
	}
	return keys
}

func validateBehaviorTargetIntervals(intervals []behaviorInterval, timelineCount int) error {
	slices.SortFunc(intervals, func(left, right behaviorInterval) int {
		return cmp.Compare(left.start, right.start)
	})
	furthestByTimeline := make(map[int]behaviorInterval, timelineCount)
	for _, current := range intervals {
		for timelineIndex, previous := range furthestByTimeline {
			if timelineIndex == current.timelineIndex ||
				!previous.persistent && current.start >= previous.end {
				continue
			}
			return fmt.Errorf(
				"%w: timeline %q phase %q conflicts with timeline %q phase %q",
				ErrBehaviorPhaseOverlap,
				current.timeline,
				current.phase,
				previous.timeline,
				previous.phase,
			)
		}
		previous, found := furthestByTimeline[current.timelineIndex]
		if !found || current.persistent || !previous.persistent && current.end > previous.end {
			furthestByTimeline[current.timelineIndex] = current
		}
	}
	return nil
}

func behaviorTimelineDuration(phases []BehaviorPhase) time.Duration {
	var duration time.Duration
	for _, phase := range phases {
		duration = max(duration, phase.StartOffset+phase.Duration)
	}
	return duration
}

func behaviorConflictKey(device, iface, faultType string) string {
	return device + "\x00" + iface + "\x00" + faultType
}

func behaviorTargets(cfg *Config) map[string]behaviorTarget {
	result := make(map[string]behaviorTarget)
	for _, segment := range cfg.NormalizedSegments() {
		for _, device := range segment.Devices {
			target := result[device.Name]
			target.count++
			target.device = device
			if target.interfaces == nil {
				target.interfaces = make(map[string]struct{}, len(device.Interfaces))
			}
			for _, iface := range device.Interfaces {
				target.interfaces[iface.Name] = struct{}{}
			}
			result[device.Name] = target
		}
	}
	return result
}

// validateBehaviorFault checks one authored fault against the device list and
// against its own axis. The presence of an interface is the discriminator, so
// a fault whose type and scope disagree is refused here rather than failing at
// apply time inside a running session.
func validateBehaviorFault(targets map[string]behaviorTarget, fault BehaviorFault) error {
	deviceFault, isDeviceFault := deviceFaultDefinition(fault.Type)
	switch {
	case isDeviceFault && fault.Interface != "":
		return fmt.Errorf(
			"%w: %q is device-scoped and takes no interface", ErrBehaviorFaultScope, fault.Type,
		)
	case !isDeviceFault && fault.Interface == "":
		return fmt.Errorf(
			"%w: %q is interface-scoped and needs an interface", ErrBehaviorFaultScope, fault.Type,
		)
	}
	ceiling := interfaceFaultMax
	if isDeviceFault {
		ceiling = deviceFault.MaxValue
	}
	if fault.Value > ceiling {
		return fmt.Errorf(
			"%w: %q accepts at most %d, got %d",
			ErrBehaviorFaultValue, fault.Type, ceiling, fault.Value,
		)
	}
	if isDeviceFault {
		return validateBehaviorDevice(targets, fault.Device)
	}
	return validateBehaviorTarget(targets, fault.Device, fault.Interface)
}

// deviceFaultDefinition reports whether a fault type belongs to the
// device-service axis, with the ceiling that type carries.
func deviceFaultDefinition(faultType string) (devicestate.DeviceFaultDefinition, bool) {
	for _, definition := range devicestate.DeviceFaultDefinitions() {
		if string(definition.Type) == faultType {
			return definition, true
		}
	}
	return devicestate.DeviceFaultDefinition{}, false
}

func validateBehaviorDevice(targets map[string]behaviorTarget, device string) error {
	target, found := targets[device]
	if !found {
		return fmt.Errorf("%w: device %q", ErrBehaviorTargetNotFound, device)
	}
	if target.count != 1 {
		return fmt.Errorf("%w: device %q", ErrBehaviorTargetAmbiguous, device)
	}
	return nil
}

func validateBehaviorTarget(targets map[string]behaviorTarget, device, iface string) error {
	target, found := targets[device]
	if !found {
		return fmt.Errorf("%w: device %q", ErrBehaviorTargetNotFound, device)
	}
	if target.count != 1 {
		return fmt.Errorf("%w: device %q", ErrBehaviorTargetAmbiguous, device)
	}
	if _, found = target.interfaces[iface]; !found {
		return fmt.Errorf("%w: interface %q on device %q", ErrBehaviorTargetNotFound, iface, device)
	}
	return nil
}
