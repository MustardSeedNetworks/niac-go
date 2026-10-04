package protocols

import (
	"errors"
	"slices"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// collectorFabricStack is a sender on a management network reporting through
// the edge router to a collector on a servers network. Every device is
// simulated; only the attachment network reaches the wire.
func collectorFabricStack(t *testing.T, withGateway bool) *Stack {
	t.Helper()
	var senderRoutes []config.Route
	if withGateway {
		senderRoutes = []config.Route{{Destination: "0.0.0.0/0", Via: "Vlan200", NextHop: "10.20.0.1"}}
	}
	cfg := &config.Config{
		Networks: []config.Network{
			{Name: "attachment", Subnet: "10.10.200.0/24"},
			{Name: "management", Subnet: "10.20.0.0/24"},
			{Name: "servers", Subnet: "10.30.0.0/24"},
		},
		Attachments: []config.LogicalAttachment{{Name: "tester", Network: "attachment"}},
		Devices: []config.Device{
			{
				Name: "edge", Type: "router", MACAddress: mustForwardingMAC(t, "02:00:00:00:00:01"),
				Interfaces: []config.Interface{
					{Name: "outside", Network: "attachment", Address: "10.10.200.1/24"},
					{Name: "inside", Network: "management", Address: "10.20.0.1/24"},
					{Name: "servers", Network: "servers", Address: "10.30.0.1/24"},
				},
			},
			{
				Name: "sender", Type: "switch", MACAddress: mustForwardingMAC(t, "02:00:00:00:00:10"),
				Interfaces: []config.Interface{{
					Name: "Vlan200", Network: "management", Address: "10.20.0.10/24",
				}},
				Routes: senderRoutes,
			},
			{
				Name: "collector", Type: "server", MACAddress: mustForwardingMAC(t, "02:00:00:00:00:20"),
				Interfaces: []config.Interface{{
					Name: "eth0", Network: "servers", Address: "10.30.0.14/24",
				}},
			},
		},
	}
	report := fabric.Compile(cfg, fabric.Binding{
		Attachment: "tester", Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
		PolicyApproved: true,
	})
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&report.Topology)
	t.Cleanup(stack.notifications.sender.(*stackDatagramSender).reset)
	return stack
}

func TestNotificationToASimulatedCollectorIsRecordedAndNeverTransmitted(t *testing.T) {
	stack := collectorFabricStack(t, true)
	sender := &stack.config.Devices[1]

	if err := stack.notifications.sender.Send(
		sender, config.UntaggedTag, "10.30.0.14:514", syslogPort, []byte("<132>1 fault"),
	); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	// Neither the notification nor an ARP for its next hop may reach the wire.
	if queued := len(stack.sendQueue); queued != 0 {
		t.Fatalf("%d frames queued for the wire, want none", queued)
	}
	received := stack.GetReceivedNotifications("collector")
	if len(received) != 1 {
		t.Fatalf("collector received %d notifications, want 1: %+v", len(received), received)
	}
	got := received[0]
	if got.Receiver != "collector" || got.Sender != "sender" || got.Source != "10.20.0.10" ||
		got.Protocol != notificationProtocolSyslog || got.Message != "<132>1 fault" || got.ReceivedAt.IsZero() {
		t.Errorf("received = %+v", got)
	}
	if others := stack.GetReceivedNotifications("edge"); len(others) != 0 {
		t.Errorf("edge received %+v, want nothing", others)
	}
}

func TestInternalNotificationToAnAbsentReceiverStaysOffTheWire(t *testing.T) {
	stack := collectorFabricStack(t, true)
	sender := &stack.config.Devices[1]

	if err := stack.notifications.sender.Send(
		sender, config.UntaggedTag, "10.30.0.99:514", syslogPort, []byte("fault"),
	); !errors.Is(err, errNotificationReceiverUnreachable) {
		t.Fatalf("Send() error = %v, want %v", err, errNotificationReceiverUnreachable)
	}
	if queued := len(stack.sendQueue); queued != 0 {
		t.Fatalf("%d frames queued for the wire, want none", queued)
	}
	if received := stack.GetReceivedNotifications(""); len(received) != 0 {
		t.Errorf("received %+v, want nothing: no device owns 10.30.0.99", received)
	}
}

func TestNotificationWithoutARouteIsNotDelivered(t *testing.T) {
	stack := collectorFabricStack(t, false)
	sender := &stack.config.Devices[1]

	if err := stack.notifications.sender.Send(
		sender, config.UntaggedTag, "10.30.0.14:514", syslogPort, []byte("fault"),
	); err == nil {
		t.Fatal("Send() without a route succeeded, want an egress error")
	}
	if received := stack.GetReceivedNotifications(""); len(received) != 0 {
		t.Errorf("received %+v without a route", received)
	}
}

func TestReceivedNotificationLogKeepsTheNewest(t *testing.T) {
	log := newReceivedNotificationLog()
	for index := range receivedNotificationLimit + 2 {
		log.record(ReceivedNotification{Receiver: "nms", Message: string(rune('a' + index%26))})
	}
	log.record(ReceivedNotification{Receiver: "alpha"})

	kept := log.list("nms")
	if len(kept) != receivedNotificationLimit {
		t.Fatalf("kept %d, want %d", len(kept), receivedNotificationLimit)
	}
	if kept[0].Message != "c" {
		t.Errorf("oldest kept = %q, want the third recorded (c)", kept[0].Message)
	}
	if all := log.list(""); len(all) != receivedNotificationLimit+1 || all[0].Receiver != "alpha" {
		t.Errorf("list(\"\") = %d entries starting with %q, want alpha first", len(all), all[0].Receiver)
	}
	log.reset()
	if all := log.list(""); len(all) != 0 {
		t.Errorf("after reset = %+v", all)
	}
}

func TestNotificationProtocolFollowsTheSendingPort(t *testing.T) {
	if got := notificationProtocol(syslogPort); got != notificationProtocolSyslog {
		t.Errorf("port %d = %q", syslogPort, got)
	}
	if got := notificationProtocol(162); got != notificationProtocolSNMP {
		t.Errorf("port 162 = %q", got)
	}
}

// A simulated collector reads an SNMPv2c notification as a manager does: the
// trap OID and its varbinds are recorded, and an inform is answered so the
// sender does not resend it.
func TestSimulatedCollectorDecodesAndAcknowledgesSNMP(t *testing.T) {
	for _, tc := range []struct {
		pdu    string
		inform bool
	}{{"trap", false}, {"inform", true}} {
		t.Run(tc.pdu, func(t *testing.T) {
			stack := collectorFabricStack(t, true)
			sender := &stack.config.Devices[1]
			sender.SNMPConfig.Traps = &config.TrapConfig{
				Enabled: true, Inform: tc.inform, Receivers: []string{"10.30.0.14"}, InformTimeoutSeconds: 30,
			}
			stack.notifications.Register(sender, stack.deviceStates[sender], nil, config.UntaggedTag)
			t.Cleanup(stack.notifications.stopInforms)

			stack.notifications.sendTrap(sender, snmp.OIDLinkDown, []gosnmp.SnmpPDU{
				{Name: ".1.3.6.1.2.1.2.2.1.2.3", Type: gosnmp.OctetString, Value: "Gi1/0/3"},
			}, 7)

			received := stack.GetReceivedNotifications("collector")
			if len(received) != 1 {
				t.Fatalf("collector received %d notifications, want 1: %+v", len(received), received)
			}
			got := received[0]
			if got.Protocol != notificationProtocolSNMP || got.PDU != tc.pdu || got.TrapOID != snmp.OIDLinkDown {
				t.Errorf("received = %+v, want an SNMP %s of linkDown", got, tc.pdu)
			}
			ifDescr := NotificationVariable{OID: ".1.3.6.1.2.1.2.2.1.2.3", Type: "OctetString", Value: "Gi1/0/3"}
			if !slices.Contains(got.Variables, ifDescr) {
				t.Errorf("variables = %+v, want %+v", got.Variables, ifDescr)
			}
			if queued := len(stack.sendQueue); queued != 0 {
				t.Errorf("%d frames queued for the wire, want none", queued)
			}
			stack.notifications.informs.mu.Lock()
			pending := len(stack.notifications.informs.pending)
			stack.notifications.informs.mu.Unlock()
			if pending != 0 {
				t.Errorf("%d informs still awaiting acknowledgement, want 0", pending)
			}
		})
	}
}
