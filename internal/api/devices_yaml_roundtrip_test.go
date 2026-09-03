package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// The device editor's advanced path reads `rawYaml` from GET and posts it back
// to PUT, so the two must agree: whatever serializeDeviceToYAML emits has to
// survive parseDeviceFromYAML. Both the shape and the field coverage are
// asserted here — a subset that parses is still a device editor that silently
// drops the author's routes.
func TestSerializeDeviceToYAMLRoundTripsThroughTheParser(t *testing.T) {
	clinic, err := os.ReadFile(filepath.Join("testdata", "clinic_scenario.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadYAMLBytes(clinic)
	if err != nil {
		t.Fatalf("load clinic scenario: %v", err)
	}

	for i := range cfg.Devices {
		dev := &cfg.Devices[i]
		t.Run(dev.Name, func(t *testing.T) {
			serialized, serErr := serializeDeviceToYAML(dev)
			if serErr != nil {
				t.Fatalf("serializeDeviceToYAML() error = %v", serErr)
			}

			reloaded, parseErr := parseDeviceFromYAML(string(serialized), dev.Name)
			if parseErr != nil {
				t.Fatalf("rawYaml read-back does not parse: %v\n%s", parseErr, serialized)
			}

			assertDeviceAuthoringPreserved(t, dev, reloaded, serialized)
		})
	}
}

func assertDeviceAuthoringPreserved(t *testing.T, want, got *config.Device, serialized []byte) {
	t.Helper()

	if got.Type != want.Type {
		t.Errorf("type = %q, want %q", got.Type, want.Type)
	}
	if len(got.IPAddresses) != len(want.IPAddresses) {
		t.Errorf("ips = %v, want %v", got.IPAddresses, want.IPAddresses)
	}
	if len(got.Interfaces) != len(want.Interfaces) {
		t.Errorf("interfaces = %d, want %d", len(got.Interfaces), len(want.Interfaces))
	}
	if len(got.Routes) != len(want.Routes) {
		t.Errorf("routes = %d, want %d", len(got.Routes), len(want.Routes))
	}
	if len(got.TrunkPorts) != len(want.TrunkPorts) {
		t.Errorf("trunk_ports = %d, want %d", len(got.TrunkPorts), len(want.TrunkPorts))
	}
	if got.SNMPConfig.SysDescr != want.SNMPConfig.SysDescr {
		t.Errorf("snmp_agent.sysdescr = %q, want %q", got.SNMPConfig.SysDescr, want.SNMPConfig.SysDescr)
	}
	if (got.DHCPConfig == nil) != (want.DHCPConfig == nil) {
		t.Errorf("dhcp present = %v, want %v", got.DHCPConfig != nil, want.DHCPConfig != nil)
	}
	if t.Failed() {
		t.Logf("serialized rawYaml:\n%s", serialized)
	}
}
