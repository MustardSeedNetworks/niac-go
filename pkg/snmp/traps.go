package snmp

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/niac-go/pkg/config"
)

// Standard SNMP trap OIDs (RFC 1157, RFC 3416).
const (
	OIDColdStart             = ".1.3.6.1.6.3.1.1.5.1"
	OIDWarmStart             = ".1.3.6.1.6.3.1.1.5.2"
	OIDLinkDown              = ".1.3.6.1.6.3.1.1.5.3"
	OIDLinkUp                = ".1.3.6.1.6.3.1.1.5.4"
	OIDAuthenticationFailure = ".1.3.6.1.6.3.1.1.5.5"
	OIDEgpNeighborLoss       = ".1.3.6.1.6.3.1.1.5.6"
)

// TrapSender manages SNMP trap generation for a device.
type TrapSender struct {
	deviceName string
	deviceIP   net.IP
	trapConfig *config.TrapConfig
	receivers  []*gosnmp.GoSNMP
	running    bool
	stopChan   chan struct{}
	debugLevel int
}

// NewTrapSender creates a new SNMP trap sender.
func NewTrapSender(
	deviceName string,
	deviceIP net.IP,
	trapConfig *config.TrapConfig,
	debugLevel int,
) (*TrapSender, error) {
	if trapConfig == nil || !trapConfig.Enabled {
		return nil, ErrTrapConfigDisabled
	}

	if len(trapConfig.Receivers) == 0 {
		return nil, ErrNoTrapReceivers
	}

	ts := &TrapSender{
		deviceName: deviceName,
		deviceIP:   deviceIP,
		trapConfig: trapConfig,
		receivers:  make([]*gosnmp.GoSNMP, 0),
		stopChan:   make(chan struct{}),
		debugLevel: debugLevel,
	}

	// Determine community string
	community := trapConfig.Community
	if community == "" {
		community = config.DefaultSNMPCommunity // Default to "public" if not configured
	}

	// Initialize SNMP clients for each receiver
	for _, receiver := range trapConfig.Receivers {
		host, port, err := net.SplitHostPort(receiver)
		if err != nil {
			// Assume port 162 if not specified
			host = receiver
			port = "162"
		}

		client := &gosnmp.GoSNMP{
			Target:    host,
			Port:      parsePort(port),
			Community: community,
			Version:   gosnmp.Version2c,
			Timeout:   time.Duration(TrapTimeoutSeconds) * time.Second,
			Retries:   1,
		}

		ts.receivers = append(ts.receivers, client)
	}

	return ts, nil
}

// parsePort converts port string to uint16.
func parsePort(portStr string) uint16 {
	var port int

	_, _ = fmt.Sscanf(portStr, "%d", &port) // error is handled by range check below

	if port < 1 || port > MaxIPPort {
		return DefaultSNMPTrapPort
	}

	return uint16(port)
}

// Start starts the trap sender and monitoring loops.
func (ts *TrapSender) Start() error {
	logger := slog.Default()
	if ts.running {
		return ErrTrapSenderRunning
	}

	ts.running = true

	// Send cold start trap if configured
	if ts.trapConfig.ColdStart != nil && ts.trapConfig.ColdStart.Enabled && ts.trapConfig.ColdStart.OnStartup {
		go func() {
			time.Sleep(1 * time.Second) // Small delay after startup

			_ = ts.SendColdStart() // error is non-critical for startup trap
		}()
	}

	// Start threshold monitoring loops
	if ts.trapConfig.HighCPU != nil && ts.trapConfig.HighCPU.Enabled {
		go ts.monitorCPU()
	}

	if ts.trapConfig.HighMemory != nil && ts.trapConfig.HighMemory.Enabled {
		go ts.monitorMemory()
	}

	if ts.trapConfig.InterfaceErrors != nil && ts.trapConfig.InterfaceErrors.Enabled {
		go ts.monitorInterfaceErrors()
	}

	if ts.debugLevel >= 1 {
		logger.Info("SNMP trap sender started", "device", ts.deviceName, "receivers", len(ts.receivers))
	}

	return nil
}

// Stop stops the trap sender.
func (ts *TrapSender) Stop() {
	logger := slog.Default()
	if !ts.running {
		return
	}

	ts.running = false
	close(ts.stopChan)

	if ts.debugLevel >= 1 {
		logger.Info("SNMP trap sender stopped", "device", ts.deviceName)
	}
}

// SendColdStart sends a coldStart trap (device initialization/boot).
func (ts *TrapSender) SendColdStart() error {
	return ts.sendTrap(OIDColdStart, "coldStart", []gosnmp.SnmpPDU{})
}

// SendLinkDown sends a linkDown trap (interface went down).
func (ts *TrapSender) SendLinkDown(ifIndex int, ifDescr string) error {
	if ts.trapConfig.LinkState == nil || !ts.trapConfig.LinkState.Enabled || !ts.trapConfig.LinkState.LinkDown {
		return nil
	}

	varbinds := []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.2.1.2.2.1.1", Type: gosnmp.Integer, Value: ifIndex},      // ifIndex
		{Name: ".1.3.6.1.2.1.2.2.1.7", Type: gosnmp.Integer, Value: IfStatusDown}, // ifAdminStatus = down
		{Name: ".1.3.6.1.2.1.2.2.1.8", Type: gosnmp.Integer, Value: IfStatusDown}, // ifOperStatus = down
		{Name: ".1.3.6.1.2.1.2.2.1.2", Type: gosnmp.OctetString, Value: ifDescr},  // ifDescr
	}

	return ts.sendTrap(OIDLinkDown, "linkDown", varbinds)
}

// SendLinkUp sends a linkUp trap (interface came up).
func (ts *TrapSender) SendLinkUp(ifIndex int, ifDescr string) error {
	if ts.trapConfig.LinkState == nil || !ts.trapConfig.LinkState.Enabled || !ts.trapConfig.LinkState.LinkUp {
		return nil
	}

	varbinds := []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.2.1.2.2.1.1", Type: gosnmp.Integer, Value: ifIndex},     // ifIndex
		{Name: ".1.3.6.1.2.1.2.2.1.7", Type: gosnmp.Integer, Value: 1},           // ifAdminStatus = up
		{Name: ".1.3.6.1.2.1.2.2.1.8", Type: gosnmp.Integer, Value: 1},           // ifOperStatus = up
		{Name: ".1.3.6.1.2.1.2.2.1.2", Type: gosnmp.OctetString, Value: ifDescr}, // ifDescr
	}

	return ts.sendTrap(OIDLinkUp, "linkUp", varbinds)
}

// SendAuthenticationFailure sends an authenticationFailure trap.
func (ts *TrapSender) SendAuthenticationFailure() error {
	if ts.trapConfig.AuthenticationFailure == nil || !ts.trapConfig.AuthenticationFailure.Enabled {
		return nil
	}

	uptime := min(time.Now().Unix(), MaxUint32Value)
	varbinds := []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.2.1.1.3.0", Type: gosnmp.TimeTicks, Value: safeUint32(uptime)}, // sysUpTime
	}

	return ts.sendTrap(OIDAuthenticationFailure, "authenticationFailure", varbinds)
}

// monitorCPU monitors CPU utilization and sends traps when threshold is exceeded.
func (ts *TrapSender) monitorCPU() {
	cfg := ts.trapConfig.HighCPU
	interval := time.Duration(cfg.Interval) * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for ts.running {
		select {
		case <-ts.stopChan:
			return
		case <-ticker.C:
			// Simulate CPU usage (in real implementation, this would read actual CPU)
			cpuUsage := simRand.IntN(HundredthsPerSecond)

			if cpuUsage > cfg.Threshold {
				_ = ts.SendHighCPU(cpuUsage) // error is non-critical for monitoring trap
			}
		}
	}
}

// monitorMemory monitors memory utilization and sends traps when threshold is exceeded.
func (ts *TrapSender) monitorMemory() {
	cfg := ts.trapConfig.HighMemory
	interval := time.Duration(cfg.Interval) * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for ts.running {
		select {
		case <-ts.stopChan:
			return
		case <-ticker.C:
			// Simulate memory usage (in real implementation, this would read actual memory)
			memUsage := simRand.IntN(HundredthsPerSecond)

			if memUsage > cfg.Threshold {
				_ = ts.SendHighMemory(memUsage) // error is non-critical for monitoring trap
			}
		}
	}
}

// monitorInterfaceErrors monitors interface errors and sends traps when threshold is exceeded.
func (ts *TrapSender) monitorInterfaceErrors() {
	cfg := ts.trapConfig.InterfaceErrors
	interval := time.Duration(cfg.Interval) * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for ts.running {
		select {
		case <-ts.stopChan:
			return
		case <-ticker.C:
			// Simulate error count (in real implementation, this would read actual errors)
			errorCount := simRand.IntN(cfg.Threshold * SimulatedThresholdMultiplier)

			if errorCount > cfg.Threshold {
				_ = ts.SendInterfaceErrors(1, "eth0", errorCount) // error is non-critical for monitoring trap
			}
		}
	}
}

// SendHighCPU sends a trap for high CPU utilization.
func (ts *TrapSender) SendHighCPU(cpuPercent int) error {
	logger := slog.Default()
	varbinds := []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.4.1.9.9.109.1.1.1.1.5", Type: gosnmp.Integer, Value: cpuPercent}, // Cisco CPU 5-min average
		{Name: ".1.3.6.1.2.1.25.3.3.1.2", Type: gosnmp.Integer, Value: cpuPercent},        // hrProcessorLoad
	}

	if ts.debugLevel >= DebugLevelMinimum {
		logger.Info(
			"High CPU trap",
			"device",
			ts.deviceName,
			"cpu",
			cpuPercent,
			"threshold",
			ts.trapConfig.HighCPU.Threshold,
		)
	}

	return ts.sendTrap(".1.3.6.1.4.1.9.9.109.0.1", "highCPU", varbinds)
}

// SendHighMemory sends a trap for high memory utilization.
func (ts *TrapSender) SendHighMemory(memPercent int) error {
	logger := slog.Default()
	varbinds := []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.4.1.9.9.48.1.1.1.5", Type: gosnmp.Integer, Value: memPercent}, // Cisco memory used
		{Name: ".1.3.6.1.2.1.25.2.3.1.6", Type: gosnmp.Integer, Value: memPercent},     // hrStorageUsed
	}

	if ts.debugLevel >= DebugLevelMinimum {
		logger.Info(
			"High Memory trap",
			"device",
			ts.deviceName,
			"memory",
			memPercent,
			"threshold",
			ts.trapConfig.HighMemory.Threshold,
		)
	}

	return ts.sendTrap(".1.3.6.1.4.1.9.9.48.0.1", "highMemory", varbinds)
}

// SendInterfaceErrors sends a trap for high interface error count.
func (ts *TrapSender) SendInterfaceErrors(ifIndex int, ifDescr string, errorCount int) error {
	logger := slog.Default()
	errCount := max(errorCount, 0)
	varbinds := []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.2.1.2.2.1.1", Type: gosnmp.Integer, Value: ifIndex},                        // ifIndex
		{Name: ".1.3.6.1.2.1.2.2.1.2", Type: gosnmp.OctetString, Value: ifDescr},                    // ifDescr
		{Name: ".1.3.6.1.2.1.2.2.1.14", Type: gosnmp.Counter32, Value: safeUint32FromInt(errCount)}, // ifInErrors
		{Name: ".1.3.6.1.2.1.2.2.1.20", Type: gosnmp.Counter32, Value: safeUint32FromInt(errCount)}, // ifOutErrors
	}

	if ts.debugLevel >= DebugLevelMinimum {
		logger.Info(
			"Interface Errors trap",
			"device",
			ts.deviceName,
			"errorCount",
			errorCount,
			"threshold",
			ts.trapConfig.InterfaceErrors.Threshold,
		)
	}

	return ts.sendTrap(".1.3.6.1.2.1.2.15", "interfaceErrors", varbinds)
}

// sendTrap sends an SNMPv2c trap to all configured receivers.
func (ts *TrapSender) sendTrap(trapOID string, trapName string, varbinds []gosnmp.SnmpPDU) error {
	logger := slog.Default()
	// Build trap PDU
	uptime2 := max(time.Now().Unix()%Uint32Overflow, 0)
	trap := gosnmp.SnmpTrap{
		Variables: []gosnmp.SnmpPDU{
			{
				Name:  ".1.3.6.1.2.1.1.3.0", // sysUpTime
				Type:  gosnmp.TimeTicks,
				Value: safeUint32(uptime2),
			},
			{
				Name:  ".1.3.6.1.6.3.1.1.4.1.0", // snmpTrapOID
				Type:  gosnmp.ObjectIdentifier,
				Value: trapOID,
			},
		},
	}

	// Add custom varbinds
	trap.Variables = append(trap.Variables, varbinds...)

	// Send to all receivers
	sentCount := 0

	var lastErr error

	for _, receiver := range ts.receivers {
		err := receiver.Connect()
		if err != nil {
			if ts.debugLevel >= DebugLevelMinimum {
				logger.Error(
					"Failed to connect to trap receiver",
					"device",
					ts.deviceName,
					"target",
					receiver.Target,
					"port",
					receiver.Port,
					"error",
					err,
				)
			}

			lastErr = err

			continue
		}

		_, err = receiver.SendTrap(trap)
		_ = receiver.Conn.Close()

		if err != nil {
			if ts.debugLevel >= DebugLevelMinimum {
				logger.Error(
					"Failed to send trap",
					"device",
					ts.deviceName,
					"target",
					receiver.Target,
					"port",
					receiver.Port,
					"error",
					err,
				)
			}

			lastErr = err
		} else {
			sentCount++

			if ts.debugLevel >= DebugLevelTraps {
				logger.Debug(
					"Sent trap",
					"device",
					ts.deviceName,
					"trapName",
					trapName,
					"target",
					receiver.Target,
					"port",
					receiver.Port,
				)
			}
		}
	}

	if sentCount == 0 && lastErr != nil {
		return fmt.Errorf("failed to send trap to any receiver: %w", lastErr)
	}

	if ts.debugLevel >= 2 && sentCount > 0 {
		logger.Info(
			"Sent trap",
			"device",
			ts.deviceName,
			"trapName",
			trapName,
			"sentCount",
			sentCount,
			"totalReceivers",
			len(ts.receivers),
		)
	}

	return nil
}
