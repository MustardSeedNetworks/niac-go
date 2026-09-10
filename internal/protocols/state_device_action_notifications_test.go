package protocols

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

func TestRebootEventEmitsColdStartWithoutStartupTrigger(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		device := notificationTestDevice()
		device.SyslogConfig = nil
		device.SNMPConfig.Traps.ColdStart = &config.TrapTriggerConfig{Enabled: true}
		store := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
		sender := &recordingDatagramSender{}
		manager := newStateNotificationManager(nil)
		manager.sender = sender
		manager.Register(device, store, nil, 200)
		time.Sleep(time.Hour)
		if _, err := store.ExecuteDeviceAction(devicestate.ActionReboot, "reboot-1"); err != nil {
			t.Fatal(err)
		}
		manager.dispatchPending()
		manager.dispatchPending()
		if len(sender.datagrams) != 1 {
			t.Fatalf("coldStart count=%d", len(sender.datagrams))
		}
		datagram := sender.datagrams[0]
		decoder := gosnmp.GoSNMP{Transport: "udp", Version: gosnmp.Version2c}
		packet, err := decoder.SnmpDecodePacket(datagram.payload)
		if err != nil {
			t.Fatal(err)
		}
		if datagram.device != device || datagram.vlan != 200 || datagram.sourcePort != 162 ||
			datagram.address != "192.0.2.20:162" || packet.Variables[0].Value != uint32(0) ||
			packet.Variables[1].Value != snmp.OIDColdStart || packet.RequestID != uint32(store.Version()) {
			t.Fatalf("wrong reboot notification: %+v packet=%+v", datagram, packet)
		}
	})
}

func TestRecoveredActionsDoNotEmitNotificationsOrStartup(t *testing.T) {
	device := notificationTestDevice()
	device.SyslogConfig = nil
	device.SNMPConfig.Traps.ColdStart = &config.TrapTriggerConfig{Enabled: true, OnStartup: true}
	store := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	if _, err := store.ExecuteDeviceAction(devicestate.ActionReboot, "old"); err != nil {
		t.Fatal(err)
	}
	manager := newStateNotificationManager(nil)
	sender := &recordingDatagramSender{}
	manager.sender = sender
	manager.Register(device, store, nil, 200)
	manager.skipRestoredHistory()
	manager.sendColdStarts()
	manager.dispatchPending()
	if len(sender.datagrams) != 0 {
		t.Fatal("recovery emitted historical or startup coldStart")
	}
	if _, err := store.ExecuteDeviceAction(devicestate.ActionReboot, "new"); err != nil {
		t.Fatal(err)
	}
	manager.dispatchPending()
	if len(sender.datagrams) != 1 {
		t.Fatal("recovery suppressed a fresh reboot")
	}
}

func TestTopologyChangeDispatchWithoutTrapsOrSyslog(t *testing.T) {
	_, stack := flatDiscoveryStack()
	device := &stack.config.Devices[0]
	device.STPConfig = &config.STPConfig{Enabled: true}
	device.SyslogConfig, device.SNMPConfig.Traps = nil, nil
	store := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	manager := newStateNotificationManager(stack)
	manager.Register(device, store, nil, 200)
	if _, err := store.ExecuteDeviceAction(devicestate.ActionSTPTopologyChange, "tcn-1"); err != nil {
		t.Fatal(err)
	}
	manager.dispatchPending()
	manager.dispatchPending()
	if len(stack.sendQueue) != 1 {
		t.Fatalf("TCN count=%d", len(stack.sendQueue))
	}
	if packet := <-stack.sendQueue; packet.Device != device {
		t.Fatal("TCN source changed")
	}
}

func TestNotificationUptimeBindsAgentAtRegistration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		device := notificationTestDevice()
		store := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
		agent := snmp.NewAgentWithState(device, store, snmp.AgentOptions{})
		stack := &Stack{snmpAgents: map[*config.Device]*snmpAgentGroup{device: {baseAgent: agent}}}
		manager := newStateNotificationManager(stack)
		manager.started = time.Now().Add(-time.Hour)
		manager.Register(device, store, nil, 200)
		delete(stack.snmpAgents, device)
		time.Sleep(time.Second)
		if got := manager.notificationUptime(
			device,
		); got != agent.UptimeTicks() ||
			manager.registrations[device].uptime == nil {
			t.Fatalf("trap uptime=%d, agent uptime=%d", got, agent.UptimeTicks())
		}
	})
}

func TestDisabledColdStartDoesNotSendRebootTrap(t *testing.T) {
	device := notificationTestDevice()
	device.SyslogConfig = nil
	device.SNMPConfig.Traps.ColdStart = &config.TrapTriggerConfig{OnStartup: true}
	store := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	manager := newStateNotificationManager(nil)
	sender := &recordingDatagramSender{}
	manager.sender = sender
	manager.Register(device, store, nil, 200)
	if _, err := store.ExecuteDeviceAction(devicestate.ActionReboot, "disabled"); err != nil {
		t.Fatal(err)
	}
	manager.dispatchPending()
	if len(sender.datagrams) != 0 {
		t.Fatal("disabled coldStart emitted a trap")
	}
}
