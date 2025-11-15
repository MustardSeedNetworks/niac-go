// Package device implements device behavior simulation
package device

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/errors"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
	"github.com/krisarmstrong/niac-go/pkg/snmp"
)

// Simulator manages simulated network devices
type Simulator struct {
	config       *config.Config
	stack        *protocols.Stack
	errorManager *errors.StateManager
	devices      map[string]*SimulatedDevice
	mu           sync.RWMutex
	running      bool
	stopChan     chan struct{}
	wg           sync.WaitGroup // Track goroutines for clean shutdown
	debugLevel   int
}

// SimulatedDevice represents a single simulated device
type SimulatedDevice struct {
	Config       *config.Device
	SNMPAgent    *snmp.Agent
	TrapSender   *snmp.TrapSender // SNMP trap sender (v1.6.0)
	State        DeviceState
	LastActivity time.Time
	Counters     *DeviceCounters
	mu           sync.RWMutex
}

// DeviceState represents the current state of a device
type DeviceState string

const (
	StateUp          DeviceState = "up"
	StateDown        DeviceState = "down"
	StateStarting    DeviceState = "starting"
	StateStopping    DeviceState = "stopping"
	StateMaintenance DeviceState = "maintenance"
)

// DeviceCounters holds per-device statistics
type DeviceCounters struct {
	ARPRequestsReceived    uint64
	ARPRepliesSent         uint64
	ICMPRequestsReceived   uint64
	ICMPRepliesSent        uint64
	SNMPQueriesReceived    uint64
	HTTPRequestsReceived   uint64
	FTPConnectionsReceived uint64
	PacketsSent            uint64
	PacketsReceived        uint64
	Errors                 uint64
}

// NewSimulator creates a new device simulator
func NewSimulator(cfg *config.Config, stack *protocols.Stack, errorMgr *errors.StateManager, debugLevel int) *Simulator {
	sim := &Simulator{
		config:       cfg,
		stack:        stack,
		errorManager: errorMgr,
		devices:      make(map[string]*SimulatedDevice),
		stopChan:     make(chan struct{}),
		debugLevel:   debugLevel,
	}

	// Initialize simulated devices
	for i := range cfg.Devices {
		device := &cfg.Devices[i]
		sim.addDevice(device)
	}

	return sim
}

// addDevice adds a device to the simulator
func (s *Simulator) addDevice(device *config.Device) {
	s.mu.Lock()
	defer s.mu.Unlock()

	simDevice := &SimulatedDevice{
		Config:       device,
		SNMPAgent:    snmp.NewAgent(device, s.debugLevel),
		State:        StateUp,
		LastActivity: time.Now(),
		Counters:     &DeviceCounters{},
	}

	// Load SNMP walk file if specified
	if device.SNMPConfig.WalkFile != "" {
		err := simDevice.SNMPAgent.LoadWalkFile(device.SNMPConfig.WalkFile)
		if err != nil && s.debugLevel >= 1 {
			log.Printf("Warning: failed to load walk file for %s: %v", device.Name, err)
		}
	}

	// Initialize SNMP trap sender if configured (v1.6.0)
	if device.SNMPConfig.Traps != nil && device.SNMPConfig.Traps.Enabled {
		if len(device.IPAddresses) > 0 {
			trapSender, err := snmp.NewTrapSender(device.Name, device.IPAddresses[0], device.SNMPConfig.Traps, s.debugLevel)
			if err == nil {
				simDevice.TrapSender = trapSender
			} else if s.debugLevel >= 1 {
				log.Printf("Warning: failed to create trap sender for %s: %v", device.Name, err)
			}
		}
	}

	s.devices[device.Name] = simDevice

	if s.debugLevel >= 1 {
		ipAddr := "no-ip"
		if len(device.IPAddresses) > 0 {
			ipAddr = device.IPAddresses[0].String()
		}
		log.Printf("Added simulated device: %s (%s) at %s",
			device.Name, device.Type, ipAddr)
	}
}

// Start starts the device simulator
func (s *Simulator) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("simulator already running")
	}

	s.running = true
	s.mu.Unlock()

	// Start behavior threads for each device
	for name, device := range s.devices {
		s.wg.Add(1)
		go s.deviceBehaviorLoop(name, device)

		// Start trap sender if configured (v1.6.0)
		if device.TrapSender != nil {
			err := device.TrapSender.Start()
			if err != nil && s.debugLevel >= 1 {
				log.Printf("Warning: failed to start trap sender for %s: %v", name, err)
			}
		}
	}

	if s.debugLevel >= 1 {
		log.Printf("Device simulator started with %d devices", len(s.devices))
	}

	return nil
}

// Stop stops the device simulator
func (s *Simulator) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}

	s.running = false
	s.mu.Unlock()

	// Signal all goroutines to stop
	close(s.stopChan)

	// Wait for all device behavior loops to finish
	s.wg.Wait()

	// Stop all trap senders (v1.6.0)
	for _, device := range s.devices {
		if device.TrapSender != nil {
			device.TrapSender.Stop()
		}
	}

	// Reset stopChan for potential restart
	s.mu.Lock()
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	if s.debugLevel >= 1 {
		log.Println("Device simulator stopped")
	}
}

// deviceBehaviorLoop runs device-specific behavior
func (s *Simulator) deviceBehaviorLoop(name string, device *SimulatedDevice) {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		s.mu.RLock()
		running := s.running
		s.mu.RUnlock()

		if !running {
			return
		}

		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.performDeviceBehavior(name, device)
		}
	}
}

// performDeviceBehavior executes device-specific periodic behavior
func (s *Simulator) performDeviceBehavior(name string, device *SimulatedDevice) {
	device.mu.Lock()
	defer device.mu.Unlock()

	// Update last activity
	device.LastActivity = time.Now()

	// Device-type-specific behavior
	switch device.Config.Type {
	case "router":
		s.routerBehavior(device)
	case "switch":
		s.switchBehavior(device)
	case "ap", "access-point":
		s.apBehavior(device)
	case "server":
		s.serverBehavior(device)
	default:
		s.genericBehavior(device)
	}

	if s.debugLevel >= 3 {
		log.Printf("Device %s performed periodic behavior (state: %s)", name, device.State)
	}
}

// routerBehavior implements router-specific behavior
func (s *Simulator) routerBehavior(device *SimulatedDevice) {
	// Routers typically:
	// - Send periodic routing updates
	// - Respond to ARP requests
	// - Forward packets
	// - Generate SNMP traps for link state changes

	// For simulation, we just update statistics
	// In a full implementation, would generate actual traffic
}

// switchBehavior implements switch-specific behavior
func (s *Simulator) switchBehavior(device *SimulatedDevice) {
	// Switches typically:
	// - Maintain MAC address table
	// - Send STP BPDUs
	// - Respond to ARP requests
	// - Generate link state traps

	// For simulation, update statistics and maybe send keepalives
}

// apBehavior implements access point behavior
func (s *Simulator) apBehavior(device *SimulatedDevice) {
	// APs typically:
	// - Send beacons
	// - Respond to probe requests
	// - Handle client associations
	// - Report statistics to controller

	// For simulation, maintain basic statistics
}

// serverBehavior implements server-specific behavior
func (s *Simulator) serverBehavior(device *SimulatedDevice) {
	// Servers typically:
	// - Respond to service requests (HTTP, FTP, etc.)
	// - Generate application logs
	// - Report health metrics

	// For simulation, ensure services are marked as available
}

// genericBehavior implements generic device behavior
func (s *Simulator) genericBehavior(device *SimulatedDevice) {
	// Generic devices:
	// - Respond to ping
	// - Respond to ARP
	// - Report basic SNMP data
}

// GetDevice returns a simulated device by name
func (s *Simulator) GetDevice(name string) *SimulatedDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.devices[name]
}

// GetAllDevices returns all simulated devices
func (s *Simulator) GetAllDevices() map[string]*SimulatedDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return copy to avoid race conditions
	devices := make(map[string]*SimulatedDevice, len(s.devices))
	for deviceName, device := range s.devices {
		devices[deviceName] = device
	}
	return devices
}

// SetDeviceState sets the state of a device
func (s *Simulator) SetDeviceState(name string, state DeviceState) error {
	s.mu.RLock()
	device, exists := s.devices[name]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("device %s not found", name)
	}

	device.mu.Lock()
	defer device.mu.Unlock()

	oldState := device.State
	device.State = state

	if s.debugLevel >= 1 {
		log.Printf("Device %s state changed: %s -> %s", name, oldState, state)
	}

	// Generate SNMP traps for state changes (issue #76)
	if device.TrapSender != nil {
		// Send linkDown trap when transitioning to down/stopping states
		if (state == StateDown || state == StateStopping) && oldState != StateDown && oldState != StateStopping {
			// Get interface index - use 1 for primary interface
			ifIndex := 1
			ifDescr := "Primary Interface"
			if len(device.Config.IPAddresses) > 0 {
				ifDescr = fmt.Sprintf("Interface %s", device.Config.IPAddresses[0])
			}

			if err := device.TrapSender.SendLinkDown(ifIndex, ifDescr); err != nil {
				if s.debugLevel >= 1 {
					log.Printf("Failed to send linkDown trap for %s: %v", name, err)
				}
			} else if s.debugLevel >= 2 {
				log.Printf("Sent linkDown trap for device %s interface %d", name, ifIndex)
			}
		}

		// Send linkUp trap when transitioning to up state
		if state == StateUp && oldState != StateUp {
			ifIndex := 1
			ifDescr := "Primary Interface"
			if len(device.Config.IPAddresses) > 0 {
				ifDescr = fmt.Sprintf("Interface %s", device.Config.IPAddresses[0])
			}

			if err := device.TrapSender.SendLinkUp(ifIndex, ifDescr); err != nil {
				if s.debugLevel >= 1 {
					log.Printf("Failed to send linkUp trap for %s: %v", name, err)
				}
			} else if s.debugLevel >= 2 {
				log.Printf("Sent linkUp trap for device %s interface %d", name, ifIndex)
			}
		}
	}

	return nil
}

// IncrementCounter increments a device counter
func (s *Simulator) IncrementCounter(deviceName, counterName string) {
	s.mu.RLock()
	device, exists := s.devices[deviceName]
	s.mu.RUnlock()

	if !exists {
		return
	}

	device.mu.Lock()
	defer device.mu.Unlock()

	switch counterName {
	case "arp_requests":
		device.Counters.ARPRequestsReceived++
	case "arp_replies":
		device.Counters.ARPRepliesSent++
	case "icmp_requests":
		device.Counters.ICMPRequestsReceived++
	case "icmp_replies":
		device.Counters.ICMPRepliesSent++
	case "snmp_queries":
		device.Counters.SNMPQueriesReceived++
	case "http_requests":
		device.Counters.HTTPRequestsReceived++
	case "ftp_connections":
		device.Counters.FTPConnectionsReceived++
	case "packets_sent":
		device.Counters.PacketsSent++
	case "packets_received":
		device.Counters.PacketsReceived++
	case "errors":
		device.Counters.Errors++
	}
}

// GetCounters returns counters for a device
func (s *Simulator) GetCounters(deviceName string) *DeviceCounters {
	s.mu.RLock()
	device, exists := s.devices[deviceName]
	s.mu.RUnlock()

	if !exists {
		return &DeviceCounters{}
	}

	device.mu.RLock()
	defer device.mu.RUnlock()

	// Return copy
	return &DeviceCounters{
		ARPRequestsReceived:    device.Counters.ARPRequestsReceived,
		ARPRepliesSent:         device.Counters.ARPRepliesSent,
		ICMPRequestsReceived:   device.Counters.ICMPRequestsReceived,
		ICMPRepliesSent:        device.Counters.ICMPRepliesSent,
		SNMPQueriesReceived:    device.Counters.SNMPQueriesReceived,
		HTTPRequestsReceived:   device.Counters.HTTPRequestsReceived,
		FTPConnectionsReceived: device.Counters.FTPConnectionsReceived,
		PacketsSent:            device.Counters.PacketsSent,
		PacketsReceived:        device.Counters.PacketsReceived,
		Errors:                 device.Counters.Errors,
	}
}

// Reload gracefully reloads the configuration without stopping the simulator
// It performs a diff-based reload: adds new devices, removes deleted devices, and updates existing devices
func (s *Simulator) Reload(newConfig *config.Config) error {
	s.mu.Lock()
	wasRunning := s.running
	s.mu.Unlock()

	// If running, stop gracefully
	if wasRunning {
		if s.debugLevel >= 1 {
			log.Println("Stopping simulator for config reload...")
		}
		s.Stop()
	}

	// Build map of new device names
	newDeviceNames := make(map[string]bool)
	for i := range newConfig.Devices {
		newDeviceNames[newConfig.Devices[i].Name] = true
	}

	// Remove devices that no longer exist in new config
	s.mu.Lock()
	for name := range s.devices {
		if !newDeviceNames[name] {
			if s.debugLevel >= 1 {
				log.Printf("Removing device %s (no longer in config)", name)
			}
			delete(s.devices, name)
		}
	}

	// Add or update devices from new config
	for i := range newConfig.Devices {
		device := &newConfig.Devices[i]
		if existingDevice, exists := s.devices[device.Name]; exists {
			// Update existing device configuration
			if s.debugLevel >= 1 {
				log.Printf("Updating device %s configuration", device.Name)
			}
			existingDevice.Config = device
			// Recreate SNMP agent with updated configuration
			existingDevice.SNMPAgent = snmp.NewAgent(device, s.debugLevel)
			if device.SNMPConfig.WalkFile != "" {
				if err := existingDevice.SNMPAgent.LoadWalkFile(device.SNMPConfig.WalkFile); err != nil && s.debugLevel >= 1 {
					log.Printf("Warning: failed to reload walk file for %s: %v", device.Name, err)
				}
			}

			// Recreate trap sender if traps are enabled
			existingDevice.TrapSender = nil
			if device.SNMPConfig.Traps != nil && device.SNMPConfig.Traps.Enabled {
				if len(device.IPAddresses) > 0 {
					trapSender, err := snmp.NewTrapSender(device.Name, device.IPAddresses[0], device.SNMPConfig.Traps, s.debugLevel)
					if err == nil {
						existingDevice.TrapSender = trapSender
					} else if s.debugLevel >= 1 {
						log.Printf("Warning: failed to recreate trap sender for %s: %v", device.Name, err)
					}
				}
			}
		} else {
			// Add new device
			if s.debugLevel >= 1 {
				log.Printf("Adding new device %s", device.Name)
			}
			s.addDevice(device)
		}
	}

	// Update config reference
	s.config = newConfig
	s.mu.Unlock()

	// Restart if was running before
	if wasRunning {
		if s.debugLevel >= 1 {
			log.Println("Restarting simulator with new configuration...")
		}
		return s.Start()
	}

	if s.debugLevel >= 1 {
		log.Println("Configuration reloaded successfully")
	}
	return nil
}

// GetStats returns simulator statistics
func (s *Simulator) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalCounters := &DeviceCounters{}
	deviceStates := make(map[string]string)

	for name, device := range s.devices {
		device.mu.RLock()
		deviceStates[name] = string(device.State)

		// Aggregate counters
		totalCounters.ARPRequestsReceived += device.Counters.ARPRequestsReceived
		totalCounters.ARPRepliesSent += device.Counters.ARPRepliesSent
		totalCounters.ICMPRequestsReceived += device.Counters.ICMPRequestsReceived
		totalCounters.ICMPRepliesSent += device.Counters.ICMPRepliesSent
		totalCounters.SNMPQueriesReceived += device.Counters.SNMPQueriesReceived
		totalCounters.HTTPRequestsReceived += device.Counters.HTTPRequestsReceived
		totalCounters.FTPConnectionsReceived += device.Counters.FTPConnectionsReceived
		totalCounters.PacketsSent += device.Counters.PacketsSent
		totalCounters.PacketsReceived += device.Counters.PacketsReceived
		totalCounters.Errors += device.Counters.Errors
		device.mu.RUnlock()
	}

	return map[string]interface{}{
		"device_count":   len(s.devices),
		"device_states":  deviceStates,
		"total_counters": totalCounters,
		"running":        s.running,
	}
}
