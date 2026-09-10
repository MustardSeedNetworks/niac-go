package protocols

import (
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestSyslogDispatchUsesAuthoredIdentityAndPreservesFaultOrder(t *testing.T) {
	device := notificationTestDevice()
	device.SNMPConfig.Traps = nil
	device.SNMPConfig.SysName = "COS-CORE-SW01"
	store := devicestate.NewStore(deviceIdentity(device))
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{Name: "Gi0/1"}}})
	sender := &recordingDatagramSender{}
	manager := newStateNotificationManager(nil)
	manager.sender = sender
	manager.Register(device, store, nil, 200)
	if err := store.SetDeviceFault(devicestate.FaultDNSTimeout, 100); err != nil {
		t.Fatal(err)
	}
	if err := store.SetDeviceFault(devicestate.FaultDNSTimeout, 0); err != nil {
		t.Fatal(err)
	}
	manager.dispatchPending()
	if len(sender.datagrams) != 2 {
		t.Fatalf("received %d notifications, want exactly fault update and clear", len(sender.datagrams))
	}
	for index, kind := range []string{"device_fault.updated", "device_fault.cleared"} {
		datagram := sender.datagrams[index]
		message := string(datagram.payload)
		if !strings.Contains(message, " COS-CORE-SW01 niac ") ||
			!strings.Contains(message, "kind="+kind) || !strings.HasSuffix(message, `target="dns_timeout"`) ||
			datagram.address != "192.0.2.10:514" || datagram.vlan != 200 {
			t.Fatalf("notification %d = %+v payload=%q", index, datagram, message)
		}
	}
	manager.dispatchPending()
	if len(sender.datagrams) != 2 {
		t.Fatal("repeated dispatch resent prior faults")
	}
}

func TestSyslogDisabledDoesNotSend(t *testing.T) {
	device := notificationTestDevice()
	device.SNMPConfig.Traps = nil
	sender := &recordingDatagramSender{}
	manager := newStateNotificationManager(nil)
	manager.sender = sender
	manager.Register(device, devicestate.NewStore(deviceIdentity(device)), nil, 200)
	device.SyslogConfig.Enabled = false
	manager.sendEvent(device, devicestate.Event{Kind: devicestate.EventFaultUpdated})
	if len(sender.datagrams) != 0 {
		t.Fatal("disabled syslog emitted a notification")
	}
}
