package protocols

import (
	"errors"
	"fmt"
	"maps"
	"net/netip"
	"slices"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// receivedNotificationLimit bounds what one simulated collector keeps. A pack
// collector hears every managed device at its site, so an unbounded log would
// grow for as long as a fault flaps; the newest entries are the ones an
// operator is looking for.
const receivedNotificationLimit = 256

var errNotificationReceiverUnreachable = errors.New("notification receiver unreachable from a simulated network")

// ReceivedNotification is one notification a simulated device received from
// another simulated device.
//
// A notification sent from a simulated network never reaches the physical wire
// (see deliverInsideSimulation); when it is addressed to a simulated device, the
// receiver records it instead, which is what a collector does with it.
type ReceivedNotification struct {
	Receiver string `json:"receiver"`
	Sender   string `json:"sender"`
	// Source is the address the sender used, the one a collector files it under.
	Source   string `json:"source"`
	Protocol string `json:"protocol"`
	// Message is the syslog text. It is empty for an SNMP notification.
	Message string `json:"message,omitempty"`
	// PDU, TrapOID and Variables decode an SNMPv2c notification. An SNMPv3 one
	// is recorded without them: the collector holds no USM credentials for the
	// sender, so it cannot read the PDU.
	PDU        string                 `json:"pdu,omitempty"`
	TrapOID    string                 `json:"trapOid,omitempty"`
	Variables  []NotificationVariable `json:"variables,omitempty"`
	ReceivedAt time.Time              `json:"receivedAt"`
}

// NotificationVariable is one variable binding of a received SNMP notification.
type NotificationVariable struct {
	OID   string `json:"oid"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type receivedNotificationLog struct {
	mu         sync.Mutex
	byReceiver map[string][]ReceivedNotification
	now        func() time.Time
}

func newReceivedNotificationLog() *receivedNotificationLog {
	return &receivedNotificationLog{
		byReceiver: make(map[string][]ReceivedNotification),
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (l *receivedNotificationLog) record(notification ReceivedNotification) {
	l.mu.Lock()
	defer l.mu.Unlock()

	notification.ReceivedAt = l.now()
	entries := l.byReceiver[notification.Receiver]
	entries = append(entries, notification)
	if len(entries) > receivedNotificationLimit {
		entries = slices.Clone(entries[len(entries)-receivedNotificationLimit:])
	}
	l.byReceiver[notification.Receiver] = entries
}

// list returns what receiver heard in arrival order, or what every receiver
// heard, receiver by receiver, when receiver is empty.
func (l *receivedNotificationLog) list(receiver string) []ReceivedNotification {
	l.mu.Lock()
	defer l.mu.Unlock()

	if receiver != "" {
		return slices.Clone(l.byReceiver[receiver])
	}
	var out []ReceivedNotification
	for _, name := range slices.Sorted(maps.Keys(l.byReceiver)) {
		out = append(out, l.byReceiver[name]...)
	}
	return out
}

func (l *receivedNotificationLog) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	clear(l.byReceiver)
}

// GetReceivedNotifications returns the notifications simulated devices received
// during this session: device's alone, or every device's when device is empty.
func (s *Stack) GetReceivedNotifications(device string) []ReceivedNotification {
	if s == nil || s.receivedNotifications == nil {
		return nil
	}
	return s.receivedNotifications.list(device)
}

// deliverInsideSimulation keeps a notification whose frame would exist only on
// a simulated segment off the physical wire, and reports whether it did.
//
// Only the attachment network is physical. A device on any other network has
// no wire to send on: before #2410 its notification went out the attachment
// interface anyway, unicast to its first hop's MAC, where no real host owns it.
// A notification routed out to the attachment, or sent by a device on it, is a
// real frame and is left alone.
//
// When the receiver is a simulated device it records the notification, as a
// collector would; one whose interface is down drops it. An address no
// simulated device owns is reported, since nothing outside the attachment can
// be reached from inside.
func (s *stackDatagramSender) deliverInsideSimulation(
	origin *config.Device, notification pendingNotification,
) (bool, error) {
	runtime := s.stack.fabric
	if runtime == nil || notification.routed || s.onAttachmentSegment(origin, notification.source) {
		return false, nil
	}
	receiver, found := runtime.endpointForAddress(notification.destination)
	if !found {
		return true, fmt.Errorf("%w: no simulated device owns %s", errNotificationReceiverUnreachable,
			notification.destination)
	}
	if !runtime.interfaceAvailable(receiver.device, receiver.interfaceName) {
		return true, nil
	}
	received := ReceivedNotification{
		Receiver: receiver.device.Name, Sender: origin.Name, Source: notification.source.String(),
		Protocol: notificationProtocol(notification.sourcePort),
	}
	if received.Protocol == notificationProtocolSyslog {
		received.Message = string(notification.payload)
	} else if packet := decodeSNMPv2cNotification(notification.payload); packet != nil {
		received.PDU, received.TrapOID, received.Variables = notificationPDUName(packet.PDUType),
			notificationTrapOID(packet.Variables), notificationVariables(packet.Variables)
		// The receiver answers an inform, as a manager does; without the
		// Response the sender resends it for its whole retry budget.
		if packet.PDUType == gosnmp.InformRequest {
			s.stack.acknowledgeInform(packet.RequestID,
				netip.AddrPortFrom(notification.destination, notification.destinationPort))
		}
	}
	s.stack.receivedNotifications.record(received)
	return true, nil
}

func decodeSNMPv2cNotification(payload []byte) *gosnmp.SnmpPacket {
	decoder := &gosnmp.GoSNMP{Version: gosnmp.Version2c}
	packet, err := decoder.SnmpDecodePacket(payload)
	if err != nil || packet.Version != gosnmp.Version2c ||
		(packet.PDUType != gosnmp.SNMPv2Trap && packet.PDUType != gosnmp.InformRequest) {
		return nil
	}
	return packet
}

func notificationPDUName(pduType gosnmp.PDUType) string {
	if pduType == gosnmp.InformRequest {
		return "inform"
	}
	return "trap"
}

// snmpTrapOIDInstance is snmpTrapOID.0, the varbind naming the notification.
const snmpTrapOIDInstance = ".1.3.6.1.6.3.1.1.4.1.0"

func notificationTrapOID(variables []gosnmp.SnmpPDU) string {
	for _, variable := range variables {
		if variable.Name == snmpTrapOIDInstance {
			if oid, ok := variable.Value.(string); ok {
				return oid
			}
		}
	}
	return ""
}

func notificationVariables(variables []gosnmp.SnmpPDU) []NotificationVariable {
	out := make([]NotificationVariable, 0, len(variables))
	for _, variable := range variables {
		value := fmt.Sprint(variable.Value)
		if octets, ok := variable.Value.([]byte); ok {
			value = string(octets)
		}
		out = append(out, NotificationVariable{OID: variable.Name, Type: variable.Type.String(), Value: value})
	}
	return out
}

func (s *stackDatagramSender) onAttachmentSegment(origin *config.Device, source netip.Addr) bool {
	store := s.stack.deviceStates[origin]
	if store == nil {
		return false
	}
	iface := notificationSourceInterface(store.Snapshot().Network.Interfaces, source)
	return iface.Network == s.stack.fabric.attachmentNetwork
}

const (
	notificationProtocolSyslog = "syslog"
	notificationProtocolSNMP   = "snmp"
)

// notificationProtocol names a notification by the port its sender speaks it
// from: the notification manager sends syslog from 514 and SNMP from 162.
func notificationProtocol(sourcePort uint16) string {
	if sourcePort == snmp.DefaultSNMPTrapPort {
		return notificationProtocolSNMP
	}
	return notificationProtocolSyslog
}
