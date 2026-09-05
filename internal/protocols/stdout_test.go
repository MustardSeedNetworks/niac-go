package protocols_test

import (
	"io"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// A library must not write to stdout. `niac daemon --once` prints a JSON summary
// there and redirects the logging package to stderr so a caller can pipe the
// result to jq; a protocol handler bypassing logging corrupts that output
// (niac#1805). Everything diagnostic goes through internal/logging, whose
// destination the process owns.
func TestProtocolDiagnosticsNeverReachStdout(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{{
			Name:        "stdout-guard-router",
			Type:        "router",
			MACAddress:  net.HardwareAddr{0x02, 0, 0, 0, 0, 0x01},
			IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
			DHCPConfig: &config.DHCPConfig{
				PoolStart: net.ParseIP("192.0.2.100"),
				PoolEnd:   net.ParseIP("192.0.2.200"),
			},
		}},
	}

	captured := captureStdout(t, func() {
		// Trace verbosity: every level-gated diagnostic in the stack fires.
		stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(protocols.DebugLevelTrace))
		if reloadErr := stack.ReloadConfig(cfg); reloadErr != nil {
			t.Errorf("ReloadConfig: %v", reloadErr)
		}
	})

	if strings.TrimSpace(captured) != "" {
		t.Fatalf("protocol code wrote to stdout, which corrupts `daemon --once` output:\n%s", captured)
	}
}

// captureStdout swaps os.Stdout for a pipe around fn and returns what was
// written to it. The logging package is pointed at io.Discard for the duration
// so this measures only the code that bypasses it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	realStdout := os.Stdout
	os.Stdout = writer
	logging.SetOutput(io.Discard)
	defer func() {
		os.Stdout = realStdout
		logging.SetOutput(nil)
	}()

	done := make(chan string, 1)
	go func() {
		out, _ := io.ReadAll(reader)
		done <- string(out)
	}()

	fn()

	if closeErr := writer.Close(); closeErr != nil {
		t.Fatalf("close pipe: %v", closeErr)
	}

	return <-done
}
