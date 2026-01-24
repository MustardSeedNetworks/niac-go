// Package device implements device behavior simulation
package device

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"sync"
	"time"

	apperr "github.com/krisarmstrong/niac-go/internal/apperr"
	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/protocols"
	"github.com/krisarmstrong/niac-go/internal/protocols/snmp"
)

// Sentinel errors for device package.
var (
	ErrSimulatorAlreadyRunning = errors.New("simulator already running")
	ErrDeviceNotFound          = errors.New("device not found")
)

// Simulator manages simulated network devices.
type Simulator struct {
	config       *config.Config
	stack        *protocols.Stack
	errorManager *apperr.StateManager
	devices      map[string]*SimulatedDevice
	mu           sync.RWMutex
	running      bool
	stopChan     chan struct{}
	wg           sync.WaitGroup // Track goroutines for clean shutdown
	debugLevel   int
}

// SimulatedDevice represents a single simulated device.
type SimulatedDevice struct {
	Config       *config.Device
	SNMPAgent    *snmp.Agent
	TrapSender   *snmp.TrapSender // SNMP trap sender (v1.6.0)
	State        State
	LastActivity time.Time
	Counters     *Counters
	mu           sync.RWMutex
}

// State represents the current state of a device.
type State string

// Device state constants.
const (
	StateUp          State = "up"          // Device is running normally
	StateDown        State = "down"        // Device is offline
	StateStarting    State = "starting"    // Device is booting up
	StateStopping    State = "stopping"    // Device is shutting down
	StateMaintenance State = "maintenance" // Device is in maintenance mode
)

// Simulator behavior constants.
const (
	// deviceBehaviorInterval is the interval for device-specific periodic behavior.
	deviceBehaviorInterval = 30 * time.Second

	// Debug level constants for logging.
	debugLevelBasic   = 1 // Basic logging
	debugLevelInfo    = 2 // Informational logging
	debugLevelVerbose = 3 // Verbose debug logging
)

// Counters holds per-device statistics.
type Counters struct {
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

// NewSimulator creates a new device simulator.
func NewSimulator(
	cfg *config.Config,
	stack *protocols.Stack,
	errorMgr *apperr.StateManager,
	debugLevel int,
) *Simulator {
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

// addDevice adds a device to the simulator.
func (s *Simulator) addDevice(device *config.Device) {
	logger := slog.Default()
	s.mu.Lock()
	defer s.mu.Unlock()

	simDevice := &SimulatedDevice{
		Config:       device,
		SNMPAgent:    snmp.NewAgent(device, s.debugLevel),
		State:        StateUp,
		LastActivity: time.Now(),
		Counters:     &Counters{},
	}

	// Load SNMP walk file if specified
	if device.SNMPConfig.WalkFile != "" {
		err := simDevice.SNMPAgent.LoadWalkFile(device.SNMPConfig.WalkFile)
		if err != nil && s.debugLevel >= 1 {
			logger.Warn("Failed to load walk file", "device", device.Name, "error", err)
		}
	}

	// Initialize SNMP trap sender if configured (v1.6.0)
	if device.SNMPConfig.Traps != nil && device.SNMPConfig.Traps.Enabled {
		if len(device.IPAddresses) > 0 {
			trapSender, err := snmp.NewTrapSender(
				device.Name,
				device.IPAddresses[0],
				device.SNMPConfig.Traps,
				s.debugLevel,
			)
			if err == nil {
				simDevice.TrapSender = trapSender
			} else if s.debugLevel >= 1 {
				logger.Warn("Failed to create trap sender", "device", device.Name, "error", err)
			}
		}
	}

	s.devices[device.Name] = simDevice

	if s.debugLevel >= 1 {
		ipAddr := "no-ip"
		if len(device.IPAddresses) > 0 {
			ipAddr = device.IPAddresses[0].String()
		}

		logger.Info("Added simulated device", "name", device.Name, "type", device.Type, "ip", ipAddr)
	}
}

// Start starts the device simulator.
func (s *Simulator) Start() error {
	logger := slog.Default()
	s.mu.Lock()

	if s.running {
		s.mu.Unlock()

		return ErrSimulatorAlreadyRunning
	}

	s.running = true
	s.mu.Unlock()

	// Start behavior threads for each device
	for name, device := range s.devices {
		s.wg.Go(func() {
			s.deviceBehaviorLoop(name, device)
		})

		// Start trap sender if configured (v1.6.0)
		if device.TrapSender != nil {
			err := device.TrapSender.Start()
			if err != nil && s.debugLevel >= 1 {
				logger.Warn("Failed to start trap sender", "device", name, "error", err)
			}
		}
	}

	if s.debugLevel >= 1 {
		logger.Info("Device simulator started", "deviceCount", len(s.devices))
	}

	return nil
}

// Stop stops the device simulator.
func (s *Simulator) Stop() {
	logger := slog.Default()
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
		logger.Info("Device simulator stopped")
	}
}

// deviceBehaviorLoop runs device-specific behavior.
func (s *Simulator) deviceBehaviorLoop(name string, device *SimulatedDevice) {
	ticker := time.NewTicker(deviceBehaviorInterval)
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

// performDeviceBehavior executes device-specific periodic behavior.
func (s *Simulator) performDeviceBehavior(name string, device *SimulatedDevice) {
	logger := slog.Default()
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

	if s.debugLevel >= debugLevelVerbose {
		logger.Debug("Device performed periodic behavior", "device", name, "state", device.State)
	}
}

// routerBehavior implements router-specific behavior.
func (s *Simulator) routerBehavior(_ *SimulatedDevice) {
	// Routers typically:
	// - Send periodic routing updates
	// - Respond to ARP requests
	// - Forward packets
	// - Generate SNMP traps for link state changes

	// For simulation, we just update statistics
	// In a full implementation, would generate actual traffic
}

// switchBehavior implements switch-specific behavior.
func (s *Simulator) switchBehavior(_ *SimulatedDevice) {
	// Switches typically:
	// - Maintain MAC address table
	// - Send STP BPDUs
	// - Respond to ARP requests
	// - Generate link state traps

	// For simulation, update statistics and maybe send keepalives
}

// apBehavior implements access point behavior.
func (s *Simulator) apBehavior(_ *SimulatedDevice) {
	// APs typically:
	// - Send beacons
	// - Respond to probe requests
	// - Handle client associations
	// - Report statistics to controller

	// For simulation, maintain basic statistics
}

// serverBehavior implements server-specific behavior.
func (s *Simulator) serverBehavior(_ *SimulatedDevice) {
	// Servers typically:
	// - Respond to service requests (HTTP, FTP, etc.)
	// - Generate application logs
	// - Report health metrics

	// For simulation, ensure services are marked as available
}

// genericBehavior implements generic device behavior.
func (s *Simulator) genericBehavior(_ *SimulatedDevice) {
	// Generic devices:
	// - Respond to ping
	// - Respond to ARP
	// - Report basic SNMP data
}

// GetDevice returns a simulated device by name.
func (s *Simulator) GetDevice(name string) *SimulatedDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.devices[name]
}

// GetAllDevices returns all simulated devices.
func (s *Simulator) GetAllDevices() map[string]*SimulatedDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return copy to avoid race conditions
	devices := make(map[string]*SimulatedDevice, len(s.devices))
	maps.Copy(devices, s.devices)

	return devices
}

// primaryInterfaceIndex is the default interface index for SNMP traps.
const primaryInterfaceIndex = 1

// SetDeviceState sets the state of a device.
func (s *Simulator) SetDeviceState(name string, state State) error {
	s.mu.RLock()
	device, exists := s.devices[name]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("%w: %s", ErrDeviceNotFound, name)
	}

	device.mu.Lock()
	defer device.mu.Unlock()

	oldState := device.State
	device.State = state

	if s.debugLevel >= debugLevelBasic {
		slog.Info("Device state changed", "device", name, "oldState", oldState, "newState", state)
	}

	s.sendStateChangeTraps(device, name, oldState, state)

	return nil
}

// sendStateChangeTraps sends SNMP traps for device state changes.
func (s *Simulator) sendStateChangeTraps(device *SimulatedDevice, name string, oldState, newState State) {
	if device.TrapSender == nil {
		return
	}

	ifDescr := s.getInterfaceDescription(device)

	if s.isTransitionToDown(oldState, newState) {
		s.sendLinkDownTrap(device, name, ifDescr)
	}

	if s.isTransitionToUp(oldState, newState) {
		s.sendLinkUpTrap(device, name, ifDescr)
	}
}

// getInterfaceDescription returns a description for the device's primary interface.
func (s *Simulator) getInterfaceDescription(device *SimulatedDevice) string {
	if len(device.Config.IPAddresses) > 0 {
		return fmt.Sprintf("Interface %s", device.Config.IPAddresses[0])
	}
	return "Primary Interface"
}

// isTransitionToDown returns true if the state change represents a link going down.
func (s *Simulator) isTransitionToDown(oldState, newState State) bool {
	wasUp := oldState != StateDown && oldState != StateStopping
	goingDown := newState == StateDown || newState == StateStopping
	return wasUp && goingDown
}

// isTransitionToUp returns true if the state change represents a link coming up.
func (s *Simulator) isTransitionToUp(oldState, newState State) bool {
	return newState == StateUp && oldState != StateUp
}

// sendLinkDownTrap sends a linkDown SNMP trap for the device.
func (s *Simulator) sendLinkDownTrap(device *SimulatedDevice, name, ifDescr string) {
	err := device.TrapSender.SendLinkDown(primaryInterfaceIndex, ifDescr)
	if err != nil {
		if s.debugLevel >= debugLevelBasic {
			slog.Error("Failed to send linkDown trap", "device", name, "error", err)
		}
		return
	}
	if s.debugLevel >= debugLevelInfo {
		slog.Info("Sent linkDown trap", "device", name, "ifIndex", primaryInterfaceIndex)
	}
}

// sendLinkUpTrap sends a linkUp SNMP trap for the device.
func (s *Simulator) sendLinkUpTrap(device *SimulatedDevice, name, ifDescr string) {
	err := device.TrapSender.SendLinkUp(primaryInterfaceIndex, ifDescr)
	if err != nil {
		if s.debugLevel >= debugLevelBasic {
			slog.Error("Failed to send linkUp trap", "device", name, "error", err)
		}
		return
	}
	if s.debugLevel >= debugLevelInfo {
		slog.Info("Sent linkUp trap", "device", name, "ifIndex", primaryInterfaceIndex)
	}
}

// IncrementCounter increments a device counter.
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

// GetCounters returns counters for a device.
func (s *Simulator) GetCounters(deviceName string) *Counters {
	s.mu.RLock()
	device, exists := s.devices[deviceName]
	s.mu.RUnlock()

	if !exists {
		return &Counters{}
	}

	device.mu.RLock()
	defer device.mu.RUnlock()

	// Return copy
	return &Counters{
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

// Reload gracefully reloads the configuration without stopping the simulator.
// It performs a diff-based reload: adds new devices, removes deleted devices, and updates existing devices.
func (s *Simulator) Reload(newConfig *config.Config) error {
	wasRunning := s.stopIfRunning()

	newDeviceNames := s.buildDeviceNameSet(newConfig)

	s.mu.Lock()
	s.removeDeletedDevices(newDeviceNames)
	s.updateOrAddDevices(newConfig)
	s.config = newConfig
	s.mu.Unlock()

	return s.restartIfNeeded(wasRunning)
}

// stopIfRunning stops the simulator if it's running and returns true if it was running.
func (s *Simulator) stopIfRunning() bool {
	s.mu.Lock()
	wasRunning := s.running
	s.mu.Unlock()

	if wasRunning {
		if s.debugLevel >= debugLevelBasic {
			slog.Info("Stopping simulator for config reload...")
		}
		s.Stop()
	}
	return wasRunning
}

// buildDeviceNameSet creates a set of device names from the new configuration.
func (s *Simulator) buildDeviceNameSet(newConfig *config.Config) map[string]bool {
	names := make(map[string]bool, len(newConfig.Devices))
	for i := range newConfig.Devices {
		names[newConfig.Devices[i].Name] = true
	}
	return names
}

// removeDeletedDevices removes devices that no longer exist in the new configuration.
// Must be called with s.mu held.
func (s *Simulator) removeDeletedDevices(newDeviceNames map[string]bool) {
	for name := range s.devices {
		if newDeviceNames[name] {
			continue
		}
		if s.debugLevel >= debugLevelBasic {
			slog.Info("Removing device (no longer in config)", "device", name)
		}
		delete(s.devices, name)
	}
}

// updateOrAddDevices updates existing devices or adds new ones from the configuration.
// Must be called with s.mu held.
func (s *Simulator) updateOrAddDevices(newConfig *config.Config) {
	for i := range newConfig.Devices {
		device := &newConfig.Devices[i]
		if existingDevice, exists := s.devices[device.Name]; exists {
			s.updateExistingDevice(existingDevice, device)
		} else {
			s.addNewDevice(device)
		}
	}
}

// updateExistingDevice updates an existing device's configuration.
// Must be called with s.mu held.
func (s *Simulator) updateExistingDevice(existingDevice *SimulatedDevice, device *config.Device) {
	if s.debugLevel >= debugLevelBasic {
		slog.Info("Updating device configuration", "device", device.Name)
	}

	existingDevice.Config = device
	s.recreateSNMPAgent(existingDevice, device)
	s.recreateTrapSender(existingDevice, device)
}

// recreateSNMPAgent recreates the SNMP agent for a device with updated configuration.
func (s *Simulator) recreateSNMPAgent(existingDevice *SimulatedDevice, device *config.Device) {
	existingDevice.SNMPAgent = snmp.NewAgent(device, s.debugLevel)
	if device.SNMPConfig.WalkFile == "" {
		return
	}
	err := existingDevice.SNMPAgent.LoadWalkFile(device.SNMPConfig.WalkFile)
	if err != nil && s.debugLevel >= debugLevelBasic {
		slog.Warn("Failed to reload walk file", "device", device.Name, "error", err)
	}
}

// recreateTrapSender recreates the SNMP trap sender for a device if traps are enabled.
func (s *Simulator) recreateTrapSender(existingDevice *SimulatedDevice, device *config.Device) {
	existingDevice.TrapSender = nil

	if !s.shouldCreateTrapSender(device) {
		return
	}

	trapSender, err := snmp.NewTrapSender(
		device.Name,
		device.IPAddresses[0],
		device.SNMPConfig.Traps,
		s.debugLevel,
	)
	if err != nil {
		if s.debugLevel >= debugLevelBasic {
			slog.Warn("Failed to recreate trap sender", "device", device.Name, "error", err)
		}
		return
	}
	existingDevice.TrapSender = trapSender
}

// shouldCreateTrapSender returns true if a trap sender should be created for the device.
func (s *Simulator) shouldCreateTrapSender(device *config.Device) bool {
	return device.SNMPConfig.Traps != nil &&
		device.SNMPConfig.Traps.Enabled &&
		len(device.IPAddresses) > 0
}

// addNewDevice adds a new device during reload.
// Must be called with s.mu held.
func (s *Simulator) addNewDevice(device *config.Device) {
	if s.debugLevel >= debugLevelBasic {
		slog.Info("Adding new device", "device", device.Name)
	}
	s.addDevice(device)
}

// restartIfNeeded restarts the simulator if it was running before reload.
func (s *Simulator) restartIfNeeded(wasRunning bool) error {
	if wasRunning {
		if s.debugLevel >= debugLevelBasic {
			slog.Info("Restarting simulator with new configuration...")
		}
		return s.Start()
	}

	if s.debugLevel >= debugLevelBasic {
		slog.Info("Configuration reloaded successfully")
	}
	return nil
}

// GetStats returns simulator statistics.
func (s *Simulator) GetStats() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalCounters := &Counters{}
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

	return map[string]any{
		"device_count":   len(s.devices),
		"device_states":  deviceStates,
		"total_counters": totalCounters,
		"running":        s.running,
	}
}
