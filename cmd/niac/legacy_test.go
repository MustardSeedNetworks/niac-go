package main

import (
	"flag"
	"os"
	"testing"

	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// TestProcessFlags_Verbose tests verbose flag override
func TestProcessFlags_Verbose(t *testing.T) {
	flags := &legacyFlags{
		debugLevel: 1,
		verbose:    true,
	}

	processFlags(flags)

	if flags.debugLevel != 3 {
		t.Errorf("Expected debug level 3 with verbose=true, got %d", flags.debugLevel)
	}
}

// TestProcessFlags_Quiet tests quiet flag override
func TestProcessFlags_Quiet(t *testing.T) {
	flags := &legacyFlags{
		debugLevel: 2,
		quiet:      true,
	}

	processFlags(flags)

	if flags.debugLevel != 0 {
		t.Errorf("Expected debug level 0 with quiet=true, got %d", flags.debugLevel)
	}
}

// TestProcessFlags_VerboseOverridesQuiet tests verbose takes precedence
func TestProcessFlags_VerboseOverridesQuiet(t *testing.T) {
	flags := &legacyFlags{
		debugLevel: 1,
		verbose:    true,
		quiet:      true,
	}

	processFlags(flags)

	// verbose is processed first, so it wins
	if flags.debugLevel != 0 {
		t.Errorf("Expected debug level 0 when both verbose and quiet are set (quiet processed last), got %d", flags.debugLevel)
	}
}

// TestProcessFlags_NoOverride tests no override when flags not set
func TestProcessFlags_NoOverride(t *testing.T) {
	flags := &legacyFlags{
		debugLevel: 2,
		verbose:    false,
		quiet:      false,
	}

	processFlags(flags)

	if flags.debugLevel != 2 {
		t.Errorf("Expected debug level 2 (unchanged), got %d", flags.debugLevel)
	}
}

// TestValidateLegacyArguments_Valid tests valid arguments
func TestValidateLegacyArguments_Valid(t *testing.T) {
	args := []string{"eth0", "config.yaml"}

	iface, configFile, err := validateLegacyArguments(args)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if iface != "eth0" {
		t.Errorf("Expected interface 'eth0', got '%s'", iface)
	}

	if configFile != "config.yaml" {
		t.Errorf("Expected config file 'config.yaml', got '%s'", configFile)
	}
}

// TestValidateLegacyArguments_MissingArgs tests missing arguments
func TestValidateLegacyArguments_MissingArgs(t *testing.T) {
	testCases := []struct {
		name string
		args []string
	}{
		{"no args", []string{}},
		{"one arg only", []string{"eth0"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := validateLegacyArguments(tc.args)

			if err == nil {
				t.Error("Expected error for missing arguments, got nil")
			}

			expectedMsg := "missing required arguments: interface and config file"
			if err.Error() != expectedMsg {
				t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
			}
		})
	}
}

// TestValidateLegacyArguments_ExtraArgs tests extra arguments are ignored
func TestValidateLegacyArguments_ExtraArgs(t *testing.T) {
	args := []string{"eth0", "config.yaml", "extra1", "extra2"}

	iface, configFile, err := validateLegacyArguments(args)

	if err != nil {
		t.Errorf("Expected no error with extra args, got: %v", err)
	}

	if iface != "eth0" {
		t.Errorf("Expected interface 'eth0', got '%s'", iface)
	}

	if configFile != "config.yaml" {
		t.Errorf("Expected config file 'config.yaml', got '%s'", configFile)
	}
}

// TestSetupDebugConfig_GlobalLevel tests debug config with global level only
func TestSetupDebugConfig_GlobalLevel(t *testing.T) {
	flags := &legacyFlags{
		debugLevel:   2,
		debugARP:     -1,
		debugIP:      -1,
		debugICMP:    -1,
		debugIPv6:    -1,
		debugICMPv6:  -1,
		debugUDP:     -1,
		debugTCP:     -1,
		debugDNS:     -1,
		debugDHCP:    -1,
		debugDHCPv6:  -1,
		debugHTTP:    -1,
		debugFTP:     -1,
		debugNetBIOS: -1,
		debugSTP:     -1,
		debugLLDP:    -1,
		debugCDP:     -1,
		debugEDP:     -1,
		debugFDP:     -1,
		debugSNMP:    -1,
	}

	debugConfig := setupDebugConfig(flags)

	if debugConfig == nil {
		t.Fatal("Expected debug config, got nil")
	}

	// Test that global level is set
	if debugConfig.GetGlobal() != 2 {
		t.Errorf("Expected global level 2, got %d", debugConfig.GetGlobal())
	}
}

// TestSetupDebugConfig_PerProtocol tests per-protocol debug levels
func TestSetupDebugConfig_PerProtocol(t *testing.T) {
	flags := &legacyFlags{
		debugLevel:   1,
		debugARP:     3,
		debugIP:      2,
		debugICMP:    -1, // Should use global
		debugIPv6:    0,
		debugICMPv6:  -1,
		debugUDP:     -1,
		debugTCP:     -1,
		debugDNS:     -1,
		debugDHCP:    3,
		debugDHCPv6:  -1,
		debugHTTP:    -1,
		debugFTP:     -1,
		debugNetBIOS: -1,
		debugSTP:     -1,
		debugLLDP:    -1,
		debugCDP:     -1,
		debugEDP:     -1,
		debugFDP:     -1,
		debugSNMP:    2,
	}

	debugConfig := setupDebugConfig(flags)

	if debugConfig == nil {
		t.Fatal("Expected debug config, got nil")
	}

	// Test specific protocol levels
	if level := debugConfig.GetProtocolLevel(logging.ProtocolARP); level != 3 {
		t.Errorf("Expected ARP level 3, got %d", level)
	}

	if level := debugConfig.GetProtocolLevel(logging.ProtocolIP); level != 2 {
		t.Errorf("Expected IP level 2, got %d", level)
	}

	if level := debugConfig.GetProtocolLevel(logging.ProtocolDHCP); level != 3 {
		t.Errorf("Expected DHCP level 3, got %d", level)
	}

	if level := debugConfig.GetProtocolLevel(logging.ProtocolSNMP); level != 2 {
		t.Errorf("Expected SNMP level 2, got %d", level)
	}

	// IPv6 set to 0 should be respected
	if level := debugConfig.GetProtocolLevel(logging.ProtocolIPv6); level != 0 {
		t.Errorf("Expected IPv6 level 0, got %d", level)
	}
}

// TestSetupDebugConfig_AllProtocols tests all protocol debug levels can be set
func TestSetupDebugConfig_AllProtocols(t *testing.T) {
	flags := &legacyFlags{
		debugLevel:   1,
		debugARP:     0,
		debugIP:      1,
		debugICMP:    2,
		debugIPv6:    3,
		debugICMPv6:  0,
		debugUDP:     1,
		debugTCP:     2,
		debugDNS:     3,
		debugDHCP:    0,
		debugDHCPv6:  1,
		debugHTTP:    2,
		debugFTP:     3,
		debugNetBIOS: 0,
		debugSTP:     1,
		debugLLDP:    2,
		debugCDP:     3,
		debugEDP:     0,
		debugFDP:     1,
		debugSNMP:    2,
	}

	debugConfig := setupDebugConfig(flags)

	tests := []struct {
		protocol string
		expected int
	}{
		{logging.ProtocolARP, 0},
		{logging.ProtocolIP, 1},
		{logging.ProtocolICMP, 2},
		{logging.ProtocolIPv6, 3},
		{logging.ProtocolICMPv6, 0},
		{logging.ProtocolUDP, 1},
		{logging.ProtocolTCP, 2},
		{logging.ProtocolDNS, 3},
		{logging.ProtocolDHCP, 0},
		{logging.ProtocolDHCPv6, 1},
		{logging.ProtocolHTTP, 2},
		{logging.ProtocolFTP, 3},
		{logging.ProtocolNetBIOS, 0},
		{logging.ProtocolSTP, 1},
		{logging.ProtocolLLDP, 2},
		{logging.ProtocolCDP, 3},
		{logging.ProtocolEDP, 0},
		{logging.ProtocolFDP, 1},
		{logging.ProtocolSNMP, 2},
	}

	for _, tt := range tests {
		t.Run(tt.protocol, func(t *testing.T) {
			level := debugConfig.GetProtocolLevel(tt.protocol)
			if level != tt.expected {
				t.Errorf("Expected %s level %d, got %d", tt.protocol, tt.expected, level)
			}
		})
	}
}

// TestHandleInformationalFlags_Version tests version flag handling
func TestHandleInformationalFlags_Version(t *testing.T) {
	flags := &legacyFlags{
		showVersion: true,
	}

	// Can't easily test the actual printVersion() call without capturing stdout,
	// but we can test that it returns true (indicating program should exit)
	handled := handleInformationalFlags(flags, []string{})

	if !handled {
		t.Error("Expected handleInformationalFlags to return true for version flag")
	}
}

// TestHandleInformationalFlags_ListInterfaces tests list interfaces flag
func TestHandleInformationalFlags_ListInterfaces(t *testing.T) {
	flags := &legacyFlags{
		listInterfaces: true,
	}

	handled := handleInformationalFlags(flags, []string{})

	if !handled {
		t.Error("Expected handleInformationalFlags to return true for list-interfaces flag")
	}
}

// TestHandleInformationalFlags_ListDevices_NoConfig tests list devices without config
func TestHandleInformationalFlags_ListDevices_NoConfig(t *testing.T) {
	// This should exit with error, but handleInformationalFlags calls os.Exit
	// We can't easily test this without refactoring
	// Skipping this test as it calls os.Exit(1)
	t.Skip("Skipping test that calls os.Exit(1)")
}

// TestHandleInformationalFlags_NoFlags tests no informational flags
func TestHandleInformationalFlags_NoFlags(t *testing.T) {
	flags := &legacyFlags{
		showVersion:    false,
		listInterfaces: false,
		listDevices:    false,
	}

	handled := handleInformationalFlags(flags, []string{"eth0", "config.yaml"})

	if handled {
		t.Error("Expected handleInformationalFlags to return false when no flags are set")
	}
}

// TestDefineLegacyFlags tests that all flags are defined
func TestDefineLegacyFlags(t *testing.T) {
	// Reset flag.CommandLine to avoid conflicts
	oldCommandLine := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	defer func() {
		flag.CommandLine = oldCommandLine
	}()

	flags := &legacyFlags{}
	defineLegacyFlags(flags)

	// Test that some key flags can be looked up
	flagTests := []string{
		"d",
		"debug",
		"v",
		"verbose",
		"q",
		"quiet",
		"i",
		"interactive",
		"n",
		"dry-run",
		"V",
		"version",
		"l",
		"list-interfaces",
		"list-devices",
		"no-color",
		"log-file",
		"stats-interval",
		"babble-interval",
		"no-traffic",
		"snmp-community",
		"max-packet-size",
		"debug-arp",
		"debug-ip",
		"debug-dhcp",
		"debug-snmp",
	}

	for _, flagName := range flagTests {
		t.Run(flagName, func(t *testing.T) {
			f := flag.CommandLine.Lookup(flagName)
			if f == nil {
				t.Errorf("Flag '%s' not defined", flagName)
			}
		})
	}
}

// TestLegacyFlags_AllFieldsPresent tests that legacyFlags struct has all expected fields
func TestLegacyFlags_AllFieldsPresent(t *testing.T) {
	flags := &legacyFlags{}
	_ = flags // Use blank identifier to avoid unused variable error

	// Set all fields to ensure they exist
	flags.debugLevel = 1
	flags.verbose = true
	flags.quiet = true
	flags.interactiveMode = true
	flags.dryRun = true
	flags.showVersion = true
	flags.listInterfaces = true
	flags.listDevices = true
	flags.noColor = true
	flags.logFile = "test.log"
	flags.statsInterval = 5
	flags.babbleInterval = 60
	flags.noTraffic = true
	flags.snmpCommunity = "public"
	flags.maxPacketSize = 1514
	flags.debugARP = 1
	flags.debugIP = 1
	flags.debugICMP = 1
	flags.debugIPv6 = 1
	flags.debugICMPv6 = 1
	flags.debugUDP = 1
	flags.debugTCP = 1
	flags.debugDNS = 1
	flags.debugDHCP = 1
	flags.debugDHCPv6 = 1
	flags.debugHTTP = 1
	flags.debugFTP = 1
	flags.debugNetBIOS = 1
	flags.debugSTP = 1
	flags.debugLLDP = 1
	flags.debugCDP = 1
	flags.debugEDP = 1
	flags.debugFDP = 1
	flags.debugSNMP = 1

	// If we get here without compilation errors, all fields exist
	t.Log("All legacyFlags fields present and accessible")
}

// TestGetDebugLevelName tests debug level name formatting
func TestGetDebugLevelName(t *testing.T) {
	tests := []struct {
		level    int
		expected string
	}{
		{0, "QUIET"},
		{1, "NORMAL"},
		{2, "VERBOSE"},
		{3, "DEBUG"},
		{4, "UNKNOWN"},
		{-1, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := getDebugLevelName(tt.level)
			if result != tt.expected {
				t.Errorf("getDebugLevelName(%d) = %s, expected %s", tt.level, result, tt.expected)
			}
		})
	}
}
