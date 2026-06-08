package interactive

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func testInterfaceModel() *model {
	return &model{
		cfg: &config.Config{
			Devices: []config.Device{
				{
					Name: "switch-a",
					Interfaces: []config.Interface{
						{Name: "Ethernet1/1", Speed: 1000, Duplex: "full", AdminStatus: "up"},
					},
				},
			},
		},
		selectedDeviceIdx:   0,
		deviceConfigTab:     deviceConfigTabInterface,
		deviceConfigScrollY: 0,
	}
}

func TestHandleInterfaceSpeedCycle(t *testing.T) {
	m := testInterfaceModel()

	_, _ = m.handleInterfaceSpeedCycle()

	if got := m.cfg.Devices[0].Interfaces[0].Speed; got != 2500 {
		t.Fatalf("speed = %d, want 2500", got)
	}
}

func TestHandleInterfaceDuplexCycle(t *testing.T) {
	m := testInterfaceModel()

	_, _ = m.handleInterfaceDuplexCycle()

	if got := m.cfg.Devices[0].Interfaces[0].Duplex; got != "half" {
		t.Fatalf("duplex = %q, want half", got)
	}
}

func TestHandleInterfaceAdminToggle(t *testing.T) {
	m := testInterfaceModel()

	_, _ = m.handleInterfaceAdminToggle()

	if got := m.cfg.Devices[0].Interfaces[0].AdminStatus; got != "down" {
		t.Fatalf("admin status = %q, want down", got)
	}
}
