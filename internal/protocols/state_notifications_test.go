package protocols

import (
	"fmt"
	"net/netip"
	"strings"
	"sync"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

type recordedDatagram struct {
	address string
	payload []byte
}

type recordingDatagramSender struct {
	mu        sync.Mutex
	datagrams []recordedDatagram
}

func (s *recordingDatagramSender) Send(address string, payload []byte) error {
	s.mu.Lock()
	s.datagrams = append(s.datagrams, recordedDatagram{address: address, payload: append([]byte(nil), payload...)})
	s.mu.Unlock()
	return nil
}

func TestStateTransitionEmitsMatchingSyslogAndSNMPLinkNotification(t *testing.T) {
	device := notificationTestDevice()
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{
		Name: "Gi0/1", Address: netip.MustParsePrefix("10.0.0.1/24"), AdminUp: true, OperUp: true,
	}}})
	sender := &recordingDatagramSender{}
	manager := newStateNotificationManager()
	manager.sender = sender
	manager.Register(device, store, func(name string) (int, bool) {
		return 47, name == "Gi0/1"
	})
	manager.dispatchPending()
	sender.datagrams = nil

	if err := store.UpdateInterface("Gi0/1", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.AdminUp = false
		iface.OperUp = false
		return iface, nil
	}); err != nil {
		t.Fatalf("UpdateInterface() error = %v", err)
	}
	manager.dispatchPending()

	if len(sender.datagrams) != 2 {
		t.Fatalf("datagrams = %d, want SYSLOG and SNMP", len(sender.datagrams))
	}
	event := store.Events()[len(store.Events())-1]
	if got := string(sender.datagrams[0].payload); sender.datagrams[0].address != "192.0.2.10:514" ||
		!strings.Contains(got, fmt.Sprintf(`version="%d"`, event.Version)) ||
		!strings.Contains(got, `kind="interface.updated"`) {
		t.Fatalf("SYSLOG datagram = %q to %q", got, sender.datagrams[0].address)
	}
	decoder := gosnmp.GoSNMP{Transport: "udp", Version: gosnmp.Version2c}
	packet, err := decoder.SnmpDecodePacket(sender.datagrams[1].payload)
	if err != nil {
		t.Fatalf("decode trap: %v", err)
	}
	if sender.datagrams[1].address != "192.0.2.20:162" || packet.PDUType != gosnmp.SNMPv2Trap ||
		packet.RequestID != uint32(event.Version) || packet.Variables[1].Value != snmp.OIDLinkDown ||
		packet.Variables[2].Name != ".1.3.6.1.2.1.2.2.1.1.47" || packet.Variables[2].Value != 47 ||
		packet.Variables[3].Name != ".1.3.6.1.2.1.2.2.1.7.47" ||
		packet.Variables[3].Value != snmp.IfStatusDown ||
		packet.Variables[4].Value != snmp.IfStatusDown || string(packet.Variables[6].Value.([]byte)) != "edge-1" ||
		packet.Variables[7].Value != "10.0.0.1" {
		t.Fatalf("trap = %#v to %q", packet, sender.datagrams[1].address)
	}
}

func TestInterfaceDescriptionChangeDoesNotEmitLinkTrap(t *testing.T) {
	device := notificationTestDevice()
	device.SyslogConfig = nil
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{
		Name: "Gi0/1", Address: netip.MustParsePrefix("10.0.0.1/24"), AdminUp: true, OperUp: true,
	}}})
	sender := &recordingDatagramSender{}
	manager := newStateNotificationManager()
	manager.sender = sender
	manager.Register(device, store, nil)
	manager.dispatchPending()

	if err := store.UpdateInterface("Gi0/1", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.Description = "uplink"
		return iface, nil
	}); err != nil {
		t.Fatalf("UpdateInterface() error = %v", err)
	}
	manager.dispatchPending()
	if len(sender.datagrams) != 0 {
		t.Fatalf("description-only change emitted %d traps", len(sender.datagrams))
	}
}

func TestAdministrativeChangeWithoutOperationalTransitionDoesNotEmitLinkTrap(t *testing.T) {
	device := notificationTestDevice()
	device.SyslogConfig = nil
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{
		Name: "Gi0/1", AdminUp: true, OperUp: true,
	}}})
	sender := &recordingDatagramSender{}
	manager := newStateNotificationManager()
	manager.sender = sender
	manager.Register(device, store, nil)
	manager.dispatchPending()
	if err := store.UpdateInterface("Gi0/1", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.AdminUp = false
		return iface, nil
	}); err != nil {
		t.Fatal(err)
	}
	manager.dispatchPending()
	if len(sender.datagrams) != 0 {
		t.Fatalf("admin-only transition emitted %d link traps", len(sender.datagrams))
	}
}

func TestLinkTrapReportsIndependentAdminAndOperationalStatus(t *testing.T) {
	device := notificationTestDevice()
	device.SyslogConfig = nil
	store := devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{
		Name: "Gi0/1", AdminUp: false, OperUp: false,
	}}})
	sender := &recordingDatagramSender{}
	manager := newStateNotificationManager()
	manager.sender = sender
	manager.Register(device, store, nil)
	manager.dispatchPending()
	if err := store.UpdateInterface("Gi0/1", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.OperUp = true
		return iface, nil
	}); err != nil {
		t.Fatal(err)
	}
	manager.dispatchPending()
	decoder := gosnmp.GoSNMP{Transport: "udp", Version: gosnmp.Version2c}
	packet, err := decoder.SnmpDecodePacket(sender.datagrams[0].payload)
	if err != nil {
		t.Fatal(err)
	}
	statuses := make(map[string]int)
	for _, variable := range packet.Variables {
		if value, ok := variable.Value.(int); ok {
			statuses[variable.Name] = value
		}
	}
	if statuses[".1.3.6.1.2.1.2.2.1.7.1"] != snmp.IfStatusDown ||
		statuses[".1.3.6.1.2.1.2.2.1.8.1"] != snmp.IfStatusUp {
		t.Fatalf("link statuses = %#v", statuses)
	}
}

func notificationTestDevice() *config.Device {
	return &config.Device{
		Name:         "edge-1",
		SyslogConfig: &config.SyslogConfig{Enabled: true, Receivers: []string{"192.0.2.10:514"}},
		SNMPConfig: config.SNMPConfig{Traps: &config.TrapConfig{
			Enabled: true, Receivers: []string{"192.0.2.20"}, Community: "traps",
			LinkState: &config.LinkStateTrapConfig{Enabled: true, LinkDown: true, LinkUp: true},
		}},
	}
}
