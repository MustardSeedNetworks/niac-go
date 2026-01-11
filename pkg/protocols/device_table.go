package protocols

import (
	"net"
	"sync"

	"github.com/krisarmstrong/niac-go/pkg/config"
)

// DeviceTable manages device lookups by MAC and IP
type DeviceTable struct {
	mu                sync.RWMutex
	byMAC             map[string]*config.Device
	byIP              map[string][]*config.Device
	ttlDevices        map[int][]*config.Device
	ttlCounts         map[*config.Device]int
	forwardingDevices []*config.Device
}

// NewDeviceTable creates a new device table
func NewDeviceTable() *DeviceTable {
	return &DeviceTable{
		byMAC:             make(map[string]*config.Device),
		byIP:              make(map[string][]*config.Device),
		ttlDevices:        make(map[int][]*config.Device),
		ttlCounts:         make(map[*config.Device]int),
		forwardingDevices: make([]*config.Device, 0),
	}
}

// Reset clears all stored devices.
func (dt *DeviceTable) Reset() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.byMAC = make(map[string]*config.Device)
	dt.byIP = make(map[string][]*config.Device)
	dt.ttlDevices = make(map[int][]*config.Device)
	dt.ttlCounts = make(map[*config.Device]int)
	dt.forwardingDevices = make([]*config.Device, 0)
}

// AddByMAC adds a device indexed by MAC address
func (dt *DeviceTable) AddByMAC(mac net.HardwareAddr, device *config.Device) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.byMAC[mac.String()] = device
}

// AddByIP adds a device indexed by IP address
func (dt *DeviceTable) AddByIP(ip net.IP, device *config.Device) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	key := ip.String()
	dt.byIP[key] = append(dt.byIP[key], device)
}

// GetByMAC looks up device by MAC address
func (dt *DeviceTable) GetByMAC(mac net.HardwareAddr) *config.Device {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.byMAC[mac.String()]
}

// GetByIP looks up devices by IP address (may return multiple)
// Works for both IPv4 and IPv6 addresses
func (dt *DeviceTable) GetByIP(ip net.IP) []*config.Device {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.byIP[ip.String()]
}

// GetByIPv6 looks up devices by IPv6 address (alias for GetByIP)
func (dt *DeviceTable) GetByIPv6(ipv6 net.IP) []*config.Device {
	return dt.GetByIP(ipv6)
}

// GetAll returns all unique devices
func (dt *DeviceTable) GetAll() []*config.Device {
	return dt.AllDevices()
}

// Remove removes a device from all indexes
func (dt *DeviceTable) Remove(device *config.Device) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// Remove by MAC
	if len(device.MACAddress) > 0 {
		delete(dt.byMAC, device.MACAddress.String())
	}

	// Remove by IPs
	for _, ip := range device.IPAddresses {
		key := ip.String()
		devices := dt.byIP[key]
		for i, d := range devices {
			if d == device {
				// Remove from slice
				dt.byIP[key] = append(devices[:i], devices[i+1:]...)
				break
			}
		}
		// Clean up empty slices
		if len(dt.byIP[key]) == 0 {
			delete(dt.byIP, key)
		}
	}
}

// Count returns the number of devices by MAC
func (dt *DeviceTable) Count() int {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return len(dt.byMAC)
}

// AllDevices returns all devices
func (dt *DeviceTable) AllDevices() []*config.Device {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	devices := make([]*config.Device, 0, len(dt.byMAC))
	for _, device := range dt.byMAC {
		devices = append(devices, device)
	}
	return devices
}

// RegisterTTL registers a device for TTL timeout handling.
func (dt *DeviceTable) RegisterTTL(device *config.Device) {
	if device == nil || device.TTLConfig == nil {
		return
	}
	ttl := device.TTLConfig.TTL
	if ttl < 1 || ttl > 255 {
		return
	}
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.ttlDevices[ttl] = append(dt.ttlDevices[ttl], device)
	if _, ok := dt.ttlCounts[device]; !ok {
		dt.ttlCounts[device] = 0
	}
}

// GetDeviceByTTL returns a device that should respond to a TTL timeout.
func (dt *DeviceTable) GetDeviceByTTL(ttl int) *config.Device {
	dt.mu.RLock()
	devices := dt.ttlDevices[ttl]
	dt.mu.RUnlock()
	if len(devices) == 0 {
		return nil
	}
	// Choose device with lowest count
	var selected *config.Device
	minCount := 0
	dt.mu.RLock()
	for _, dev := range devices {
		count := dt.ttlCounts[dev]
		if selected == nil || count < minCount {
			selected = dev
			minCount = count
		}
	}
	dt.mu.RUnlock()
	return selected
}

// IncrementTTLCount increments the response count for a TTL device.
func (dt *DeviceTable) IncrementTTLCount(device *config.Device) {
	if device == nil {
		return
	}
	dt.mu.Lock()
	dt.ttlCounts[device]++
	dt.mu.Unlock()
}

// RegisterForwardingDevice registers a device for FDB table injection.
func (dt *DeviceTable) RegisterForwardingDevice(device *config.Device) {
	if device == nil {
		return
	}
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.forwardingDevices = append(dt.forwardingDevices, device)
}

// GetForwardingDevices returns forwarding devices.
func (dt *DeviceTable) GetForwardingDevices() []*config.Device {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	devs := make([]*config.Device, len(dt.forwardingDevices))
	copy(devs, dt.forwardingDevices)
	return devs
}
