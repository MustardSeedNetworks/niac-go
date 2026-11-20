package scenario_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

const (
	entPhysicalClass  = "1.3.6.1.2.1.47.1.1.1.1.5.1"
	entPhysicalName   = "1.3.6.1.2.1.47.1.1.1.1.7.1"
	entPhysicalSerial = "1.3.6.1.2.1.47.1.1.1.1.11.1"
	entPhysicalModel  = "1.3.6.1.2.1.47.1.1.1.1.13.1"
	ifNumberOID       = "1.3.6.1.2.1.2.1.0"
	ifDescrOID        = "1.3.6.1.2.1.2.2.1.2"
	sysNameOID        = "1.3.6.1.2.1.1.5.0"
	chassisClass      = "3"
)

func isChassisDevice(deviceType string) bool {
	switch deviceType {
	case "switch", "layer3-switch", "router", "firewall":
		return true
	}

	return false
}

// Every switch, router and firewall a pack manufactures reports a chassis whose
// model is the one the pack authored for it, and an ifTable with a row for each
// interface the pack gave it. The serial must not repeat inside a pack.
func testManufacturedNetworkDevicesAnswerEntityAndIfTable(t *testing.T, fixture *packFixture) {
	for _, pack := range scenario.Packs() {
		t.Run(pack.ID, func(t *testing.T) {
			cfg := fixture.config(t, pack)
			serials := make(map[string]string)
			checked := 0
			for i := range cfg.Devices {
				device := &cfg.Devices[i]
				if !isChassisDevice(device.Type) || device.SNMPConfig.WalkFile != "" {
					continue
				}
				checked++
				agent := agentAsTheStackBuildsIt(t, device)
				wantValue(t, agent, entPhysicalClass, chassisClass)
				wantValue(t, agent, entPhysicalName, device.Name)
				wantValue(t, agent, entPhysicalModel, device.Properties["model"])
				wantValue(t, agent, ifNumberOID, strconv.Itoa(len(device.Interfaces)))
				wantValue(t, agent, sysNameOID, device.Name)
				assertIfDescrsAreTheAuthoredInterfaces(t, agent, device)
				serial, _ := agentValue(agent, entPhysicalSerial)
				if serial == "" {
					t.Errorf("%s reports no chassis serial", device.Name)
				}
				if previous, clash := serials[serial]; clash {
					t.Errorf("%s and %s both report serial %s", previous, device.Name, serial)
				}
				serials[serial] = device.Name
			}
			if checked == 0 {
				t.Fatalf("pack %s has no switch, router or firewall to check", pack.ID)
			}
		})
	}
}

// The agent numbers interfaces its own way, so the check is that the served
// ifDescr column is exactly the set of interfaces the pack authored.
func assertIfDescrsAreTheAuthoredInterfaces(t *testing.T, agent *snmp.Agent, device *config.Device) {
	t.Helper()
	want := make(map[string]bool, len(device.Interfaces))
	for _, iface := range device.Interfaces {
		want[iface.Name] = true
	}
	for index := 1; index <= len(device.Interfaces); index++ {
		descr, ok := agentValue(agent, ifDescrOID+"."+strconv.Itoa(index))
		if !ok || !want[descr] {
			t.Errorf("%s ifDescr.%d = %q (present %t), not an authored interface", device.Name, index, descr, ok)
		}
		delete(want, descr)
	}
	for missing := range want {
		t.Errorf("%s authors %s but its ifTable does not serve it", device.Name, missing)
	}
}

// A pack may only replay a capture the content NIAC installs carries: the
// daemon refuses a walk outside the library, and a library bootstrapped from
// the shipped content is the only place a pack's walk_file can resolve.
func testPackWalksShipWithTheContent(t *testing.T, fixture *packFixture) {
	root := t.TempDir()
	lib, err := library.Open(root)
	if err != nil {
		t.Fatalf("bootstrap library: %v", err)
	}
	walks := lib.SubDir(library.KindWalks)

	for _, pack := range scenario.Packs() {
		cfg := fixture.config(t, pack)
		for i := range cfg.Devices {
			device := &cfg.Devices[i]
			referenced := append([]string{device.SNMPConfig.WalkFile}, device.SNMPConfig.WalkFiles...)
			for _, walk := range referenced {
				if walk == "" {
					continue
				}
				if _, statErr := os.Stat(filepath.Join(walks, filepath.Base(walk))); statErr != nil {
					t.Errorf("pack %s device %s replays %s, which the shipped content does not carry: %v",
						pack.ID, device.Name, walk, statErr)
				}
			}
		}
	}
}

// One capture backs a device without making the device that capture: its
// ifTable is the recording's, its name and addresses are the lab's own.
func TestCaptureBackedDeviceKeepsItsOwnIdentity(t *testing.T) {
	root := t.TempDir()
	lib, err := library.Open(root)
	if err != nil {
		t.Fatalf("bootstrap library: %v", err)
	}
	walk := filepath.Join(lib.SubDir(library.KindWalks), "brocade-icx6610-24f-01.walk")

	tests := []struct {
		name, deviceName, location string
	}{
		{"first switch", "SW-ONE", "Closet 1"},
		{"second switch", "SW-TWO", "Closet 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &config.Device{
				Name: tt.deviceName,
				SNMPConfig: config.SNMPConfig{
					Community: "public", SysName: tt.deviceName, SysLocation: tt.location, WalkFile: walk,
				},
			}
			agent := agentAsTheStackBuildsIt(t, device)
			if loadErr := agent.LoadWalkFile(walk); loadErr != nil {
				t.Fatalf("load capture: %v", loadErr)
			}
			wantValue(t, agent, sysNameOID, tt.deviceName)
			wantValue(t, agent, "1.3.6.1.2.1.1.6.0", tt.location)
			wantValue(t, agent, ifNumberOID, "42")
			wantValue(t, agent, ifDescrOID+".1", "GigabitEthernet1/1/1")
		})
	}
}
