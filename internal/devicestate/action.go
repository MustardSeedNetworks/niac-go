package devicestate

// DeviceActionType identifies an operation executed once, not an armed fault.
type DeviceActionType string

// Device actions emitted at an authored phase's entry.
const (
	ActionReboot            DeviceActionType = "reboot"
	ActionSTPTopologyChange DeviceActionType = "stp_topology_change"
)
