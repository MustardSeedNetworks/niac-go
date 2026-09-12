package devicestate

// DeviceActionType identifies an operation executed once, not an armed fault.
type DeviceActionType string

// Device actions emitted at an authored phase's entry.
const (
	ActionReboot            DeviceActionType = "reboot"
	ActionSTPTopologyChange DeviceActionType = "stp_topology_change"
)

// DeviceActionTypes returns every operation an operator can run against a
// device. It is the one enumeration: the stack's eligibility check and the
// injection catalog both read it, so an action added here reaches both without
// a second list to keep in step.
func DeviceActionTypes() []DeviceActionType {
	return []DeviceActionType{ActionReboot, ActionSTPTopologyChange}
}
