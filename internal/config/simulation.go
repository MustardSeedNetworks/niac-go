package config

import "github.com/krisarmstrong/niac-go/internal/converter"

// parseOSFingerprintConfig parses OS fingerprinting configuration from YAML.
func parseOSFingerprintConfig(yamlOSFP *converter.OSFingerprintConfig) *OSFingerprintConfig {
	if yamlOSFP == nil {
		return nil
	}

	osFP := &OSFingerprintConfig{
		OSType:       yamlOSFP.OSType,
		TTL:          yamlOSFP.TTL,
		WindowSize:   yamlOSFP.WindowSize,
		WindowScale:  yamlOSFP.WindowScale,
		MSS:          yamlOSFP.MSS,
		SSHBanner:    yamlOSFP.SSHBanner,
		HTTPServer:   yamlOSFP.HTTPServer,
		FTPBanner:    yamlOSFP.FTPBanner,
		SMTPBanner:   yamlOSFP.SMTPBanner,
		TelnetBanner: yamlOSFP.TelnetBanner,
		DontFragment: yamlOSFP.DontFragment,
	}

	// Apply OS type defaults if no specific values are set
	if osFP.OSType != "" && osFP.TTL == 0 {
		switch osFP.OSType {
		case "linux", "macos", "freebsd", "openbsd":
			osFP.TTL = 64
		case "windows", "windows-server":
			osFP.TTL = 128
		case "cisco-ios", "cisco-nxos", "juniper-junos", "arista-eos":
			osFP.TTL = 255
		default:
			osFP.TTL = 64 // Default to Linux-like
		}
	}

	// Set default window size based on OS type if not specified
	if osFP.OSType != "" && osFP.WindowSize == 0 {
		switch osFP.OSType {
		case "linux":
			osFP.WindowSize = 29200
		case "windows", "windows-server":
			osFP.WindowSize = 65535
		case "macos":
			osFP.WindowSize = 65535
		default:
			osFP.WindowSize = 65535
		}
	}

	return osFP
}

// parseIPerf3Config parses iPerf3 server emulation configuration from YAML.
func parseIPerf3Config(yamlIPerf3 *converter.IPerf3Config) *IPerf3Config {
	if yamlIPerf3 == nil {
		return nil
	}

	cfg := &IPerf3Config{
		Enabled:           yamlIPerf3.Enabled,
		Port:              yamlIPerf3.Port,
		MaxBandwidthMbps:  yamlIPerf3.MaxBandwidthMbps,
		TypicalLatencyMs:  yamlIPerf3.TypicalLatencyMs,
		JitterMs:          yamlIPerf3.JitterMs,
		PacketLossPercent: yamlIPerf3.PacketLossPercent,
		UploadMbps:        yamlIPerf3.UploadMbps,
		DownloadMbps:      yamlIPerf3.DownloadMbps,
	}

	// Set defaults for unspecified values
	if cfg.Port == 0 {
		cfg.Port = 5201 // Default iPerf3 port
	}

	if cfg.UploadMbps == 0 {
		cfg.UploadMbps = 100.0 // Default 100 Mbps
	}

	if cfg.DownloadMbps == 0 {
		cfg.DownloadMbps = 100.0 // Default 100 Mbps
	}

	if cfg.MaxBandwidthMbps == 0 {
		cfg.MaxBandwidthMbps = 1000.0 // Default 1 Gbps max
	}

	if cfg.TypicalLatencyMs == 0 {
		cfg.TypicalLatencyMs = 1.0 // Default 1ms
	}

	return cfg
}

// parseTrafficConfig parses traffic configuration from YAML.
func parseTrafficConfig(yamlTraffic *converter.TrafficConfig) *TrafficConfig {
	if yamlTraffic == nil {
		return nil
	}

	trafficCfg := &TrafficConfig{
		Enabled: yamlTraffic.Enabled,
	}

	// Parse ARP Announcements
	if yamlTraffic.ARPAnnouncements != nil {
		arpCfg := &ARPAnnouncementConfig{
			Enabled:  yamlTraffic.ARPAnnouncements.Enabled,
			Interval: yamlTraffic.ARPAnnouncements.Interval,
		}
		if arpCfg.Interval == 0 {
			arpCfg.Interval = DefaultARPAnnouncementInterval
		}

		trafficCfg.ARPAnnouncements = arpCfg
	}

	// Parse Periodic Pings
	if yamlTraffic.PeriodicPings != nil {
		pingCfg := &PeriodicPingConfig{
			Enabled:     yamlTraffic.PeriodicPings.Enabled,
			Interval:    yamlTraffic.PeriodicPings.Interval,
			PayloadSize: yamlTraffic.PeriodicPings.PayloadSize,
		}
		if pingCfg.Interval == 0 {
			pingCfg.Interval = DefaultPeriodicPingInterval
		}

		if pingCfg.PayloadSize == 0 {
			pingCfg.PayloadSize = DefaultPeriodicPingPayloadSize
		}

		trafficCfg.PeriodicPings = pingCfg
	}

	// Parse Random Traffic
	if yamlTraffic.RandomTraffic != nil {
		randomCfg := &RandomTrafficConfig{
			Enabled:     yamlTraffic.RandomTraffic.Enabled,
			Interval:    yamlTraffic.RandomTraffic.Interval,
			PacketCount: yamlTraffic.RandomTraffic.PacketCount,
			Patterns:    yamlTraffic.RandomTraffic.Patterns,
		}
		if randomCfg.Interval == 0 {
			randomCfg.Interval = DefaultRandomTrafficInterval
		}

		if randomCfg.PacketCount == 0 {
			randomCfg.PacketCount = DefaultRandomTrafficPacketCount
		}

		if len(randomCfg.Patterns) == 0 {
			randomCfg.Patterns = []string{"broadcast_arp", "multicast", "udp"}
		}

		trafficCfg.RandomTraffic = randomCfg
	}

	return trafficCfg
}

// parseSNMPTrapsConfig parses SNMP traps configuration from YAML.
func parseSNMPTrapsConfig(yamlTraps *converter.TrapsConfig) *TrapConfig {
	trapsCfg := &TrapConfig{
		Enabled:   yamlTraps.Enabled,
		Receivers: yamlTraps.Receivers,
		Community: yamlTraps.Community,
	}

	// Parse Cold Start trap
	if yamlTraps.ColdStart != nil {
		trapsCfg.ColdStart = &TrapTriggerConfig{
			Enabled:   yamlTraps.ColdStart.Enabled,
			OnStartup: yamlTraps.ColdStart.OnStartup,
		}
	}

	// Parse Link State trap
	if yamlTraps.LinkState != nil {
		trapsCfg.LinkState = &LinkStateTrapConfig{
			Enabled:  yamlTraps.LinkState.Enabled,
			LinkDown: yamlTraps.LinkState.LinkDown,
			LinkUp:   yamlTraps.LinkState.LinkUp,
		}
	}

	// Parse Authentication Failure trap
	if yamlTraps.AuthenticationFailure != nil {
		trapsCfg.AuthenticationFailure = &TrapTriggerConfig{
			Enabled:   yamlTraps.AuthenticationFailure.Enabled,
			OnStartup: yamlTraps.AuthenticationFailure.OnStartup,
		}
	}

	// Parse High CPU trap
	if yamlTraps.HighCPU != nil {
		highCPUCfg := &ThresholdTrapConfig{
			Enabled:   yamlTraps.HighCPU.Enabled,
			Threshold: yamlTraps.HighCPU.Threshold,
			Interval:  yamlTraps.HighCPU.Interval,
		}
		if highCPUCfg.Threshold == 0 {
			highCPUCfg.Threshold = DefaultHighCPUThreshold
		}

		if highCPUCfg.Interval == 0 {
			highCPUCfg.Interval = DefaultTrapCheckInterval
		}

		trapsCfg.HighCPU = highCPUCfg
	}

	// Parse High Memory trap
	if yamlTraps.HighMemory != nil {
		highMemCfg := &ThresholdTrapConfig{
			Enabled:   yamlTraps.HighMemory.Enabled,
			Threshold: yamlTraps.HighMemory.Threshold,
			Interval:  yamlTraps.HighMemory.Interval,
		}
		if highMemCfg.Threshold == 0 {
			highMemCfg.Threshold = DefaultHighMemoryThreshold
		}

		if highMemCfg.Interval == 0 {
			highMemCfg.Interval = DefaultTrapCheckInterval
		}

		trapsCfg.HighMemory = highMemCfg
	}

	// Parse Interface Errors trap
	if yamlTraps.InterfaceErrors != nil {
		ifErrCfg := &ThresholdTrapConfig{
			Enabled:   yamlTraps.InterfaceErrors.Enabled,
			Threshold: yamlTraps.InterfaceErrors.Threshold,
			Interval:  yamlTraps.InterfaceErrors.Interval,
		}
		if ifErrCfg.Threshold == 0 {
			ifErrCfg.Threshold = DefaultInterfaceErrorThreshold
		}

		if ifErrCfg.Interval == 0 {
			ifErrCfg.Interval = DefaultInterfaceErrorInterval
		}

		trapsCfg.InterfaceErrors = ifErrCfg
	}

	return trapsCfg
}
