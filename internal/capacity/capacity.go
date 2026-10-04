// Package capacity sizes NIAC's simulation budget to the host it runs on.
//
// The budget used to be a fixed 1,000 devices for the whole daemon. That was
// sized for one configuration, and the shipped presentation packs alone need
// about 1,550 to run together (#2469); a fixed number is wrong for every host
// but one. The budget is now what the host's memory can carry, so a larger
// server simulates a larger network without a configuration change. It stays
// a resource-exhaustion guard (#173): a start that would exceed it is refused.
package capacity

import "math"

const (
	// bytesPerDevice is the memory budgeted for one simulated device. Measured
	// on the reference lab host (CT304, v0.106.2): six packs, 962 devices,
	// 285 MiB resident, about 0.3 MiB each. 1 MiB leaves room for devices that
	// serve large SNMP walks.
	bytesPerDevice = 1 << 20
	// reservedBytes is held back for the daemon itself, the web UI and the OS.
	reservedBytes = 256 << 20
	// minDevices keeps a very small host able to run one modest scenario.
	minDevices = 100
	// fallbackDevices applies when the host's memory cannot be read. It is the
	// old fixed budget, so such a host behaves as it did before.
	fallbackDevices = 1000
)

// MaxDevices is how many simulated devices this process may carry across all
// sessions, derived from the memory available to it. In a container that is
// the container's memory limit, not the host's. The daemon and the API server
// each read it once, when they are built.
func MaxDevices() int {
	memory, err := totalMemory()
	if err != nil {
		return fallbackDevices
	}
	return devicesFor(memory)
}

func devicesFor(memory uint64) int {
	if memory <= reservedBytes+minDevices*bytesPerDevice {
		return minDevices
	}
	devices := (memory - reservedBytes) / bytesPerDevice
	if devices > math.MaxInt32 {
		return math.MaxInt32
	}
	return int(devices)
}
