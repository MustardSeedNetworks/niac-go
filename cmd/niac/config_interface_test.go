package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/config"
)

func TestRunConfigInterfaceSet(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "network.yaml")
	writeTestFile(t, configFile, []byte(`
devices:
  - name: switch-a
    type: switch
    mac: "00:11:22:33:44:55"
`))

	output := captureStdout(t, func() {
		err := runConfigInterfaceSet(configFile, "switch-a", "Ethernet1/1", &configInterfaceOptions{
			speed:       1000,
			duplex:      "full",
			adminStatus: "up",
			operStatus:  "down",
		})
		if err != nil {
			t.Fatalf("runConfigInterfaceSet: %v", err)
		}
	})
	if !strings.Contains(output, "Updated switch-a Ethernet1/1") {
		t.Fatalf("output = %q, want update status", output)
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("load updated config: %v", err)
	}
	ifaces := cfg.Devices[0].Interfaces
	if len(ifaces) != 1 {
		t.Fatalf("interfaces count = %d, want 1", len(ifaces))
	}
	if ifaces[0].Speed != 1000 || ifaces[0].Duplex != "full" || ifaces[0].OperStatus != "down" {
		t.Fatalf("interface not updated: %+v", ifaces[0])
	}
}

func TestRunConfigInterfaceSetRejectsBadDuplex(t *testing.T) {
	err := runConfigInterfaceSet("network.yaml", "switch-a", "Ethernet1/1", &configInterfaceOptions{
		duplex: "bad",
	})
	if err == nil || !strings.Contains(err.Error(), "duplex must be") {
		t.Fatalf("err = %v, want duplex validation error", err)
	}
}
