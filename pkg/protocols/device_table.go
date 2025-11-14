package protocols

import (
	"net"
	"sync"

	"github.com/krisarmstrong/niac-go/pkg/config"
)

// DeviceTable manages device lookups by MAC and IP
type DeviceTable struct {
	mu    sync.RWMutex
	byMAC map[string]*config.Device
	byIP  map[string][]*config.Device
}

// NewDeviceTable creates a new device table
func NewDeviceTable() *DeviceTable {
	return &DeviceTable{
		byMAC: make(map[string]*config.Device),
		byIP:  make(map[string][]*config.Device),
	}
}

// Reset clears all stored devices.
func (dt *DeviceTable) Reset() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.byMAC = make(map[string]*config.Device)
	dt.byIP = make(map[string][]*config.Device)
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
