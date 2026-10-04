package protocols

import (
	"net/netip"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// countingSender records how many times a payload was put on the wire.
type countingSender struct {
	sends chan string
}

func (s *countingSender) Send(
	_ *config.Device, _ int, address string, _ uint16, _ []byte,
) error {
	select {
	case s.sends <- address:
	default:
	}

	return nil
}

// collector is the receiver the inform tests send to, as the Response from it
// arrives: its address and the port it listens on.
func collector() netip.AddrPort { return netip.MustParseAddrPort("10.0.0.99:162") }

// informManager wires a manager with a fake sender and one registered device.
//
// The registration carries a real store because Reset walks it; a placeholder
// with a nil store panics there, which is a test bug that looks like one in the
// code under test.
func informManager(t *testing.T) (*stateNotificationManager, *countingSender, *config.Device) {
	t.Helper()

	sender := &countingSender{sends: make(chan string, 16)}
	manager := newStateNotificationManager(nil)
	manager.sender = sender

	device := &config.Device{Name: "edge-1"}
	manager.registrations[device] = &stateNotificationRegistration{
		store: devicestate.NewStore(devicestate.Identity{Hostname: "edge-1"}),
	}

	return manager, sender, device
}

// An acknowledged inform must stop retrying. Without that, a working receiver
// still gets the notification repeated for the whole retry budget.
func TestAcknowledgedInformStopsRetrying(t *testing.T) {
	manager, sender, device := informManager(t)

	traps := &config.TrapConfig{
		Inform: true, InformRetries: 5, InformTimeoutSeconds: 1,
	}
	manager.trackInform(device, traps, "10.0.0.99:162", 4242, []byte("payload"))

	if !manager.AcknowledgeInform(4242, collector()) {
		t.Fatal("AcknowledgeInform did not find the inform it was answering")
	}

	select {
	case address := <-sender.sends:
		t.Fatalf("an acknowledged inform was resent to %s", address)
	case <-time.After(1500 * time.Millisecond):
	}
}

// An acknowledgement for an inform nobody is waiting on must not be mistaken
// for one that is: two receivers of the same notification are tracked apart.
func TestAcknowledgementIsMatchedToItsReceiver(t *testing.T) {
	manager, _, device := informManager(t)

	traps := &config.TrapConfig{Inform: true, InformRetries: 1, InformTimeoutSeconds: 30}
	manager.trackInform(device, traps, "10.0.0.99:162", 7, []byte("payload"))

	if manager.AcknowledgeInform(7, netip.MustParseAddrPort("10.0.0.100:162")) {
		t.Error("an acknowledgement from a different receiver cleared the inform")
	}
	if manager.AcknowledgeInform(7, netip.MustParseAddrPort("10.0.0.99:1162")) {
		t.Error("an acknowledgement from a different port cleared the inform")
	}
	if manager.AcknowledgeInform(8, collector()) {
		t.Error("an acknowledgement for a different request ID cleared the inform")
	}
	if !manager.AcknowledgeInform(7, collector()) {
		t.Error("the matching acknowledgement did not clear the inform")
	}
}

// An unacknowledged inform is resent, then given up on. Retrying forever would
// keep a dead receiver's notification on the wire for the life of the session.
func TestUnacknowledgedInformRetriesThenGivesUp(t *testing.T) {
	manager, sender, device := informManager(t)

	traps := &config.TrapConfig{Inform: true, InformRetries: 2, InformTimeoutSeconds: 1}
	manager.trackInform(device, traps, "10.0.0.99:162", 99, []byte("payload"))

	for attempt := 1; attempt <= 2; attempt++ {
		select {
		case <-sender.sends:
		case <-time.After(3 * time.Second):
			t.Fatalf("retry %d never happened", attempt)
		}
	}

	// The budget is spent; nothing further goes out and the entry is dropped.
	select {
	case <-sender.sends:
		t.Error("the inform was resent after its retry budget was spent")
	case <-time.After(2500 * time.Millisecond):
	}

	manager.informs.mu.Lock()
	remaining := len(manager.informs.pending)
	manager.informs.mu.Unlock()
	if remaining != 0 {
		t.Errorf("%d informs still tracked after giving up, want 0", remaining)
	}
}

// A stopped stack must not keep resending into a network it no longer has.
func TestStopInformsCancelsOutstandingRetries(t *testing.T) {
	manager, sender, device := informManager(t)

	traps := &config.TrapConfig{Inform: true, InformRetries: 5, InformTimeoutSeconds: 1}
	manager.trackInform(device, traps, "10.0.0.99:162", 1, []byte("payload"))
	manager.stopInforms()

	select {
	case <-sender.sends:
		t.Error("a retry fired after the informs were stopped")
	case <-time.After(1500 * time.Millisecond):
	}
}

// The inform PDU type is the only difference on the wire between an inform and
// a trap, so a config that asks for one must not send the other.
func TestInformConfigSelectsTheInformPDU(t *testing.T) {
	for _, tc := range []struct {
		name   string
		inform bool
		want   gosnmp.PDUType
	}{
		{"trap", false, gosnmp.SNMPv2Trap},
		{"inform", true, gosnmp.InformRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager := newStateNotificationManager(nil)
			traps := &config.TrapConfig{Enabled: true, Inform: tc.inform}

			payload, err := manager.marshalNotification(
				&config.Device{Name: "edge-1"}, traps, "public",
				pduTypeFor(traps), 1, nil)
			if err != nil {
				t.Fatalf("marshalNotification: %v", err)
			}

			decoder := &gosnmp.GoSNMP{Version: gosnmp.Version2c}
			decoded, err := decoder.SnmpDecodePacket(payload)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if decoded.PDUType != tc.want {
				t.Errorf("PDU type = %v, want %v", decoded.PDUType, tc.want)
			}
		})
	}
}

// pduTypeFor mirrors the choice emitSNMPNotification makes.
func pduTypeFor(traps *config.TrapConfig) gosnmp.PDUType {
	if traps.Inform {
		return gosnmp.InformRequest
	}

	return gosnmp.SNMPv2Trap
}

// Reset is what the stack calls on stop, so the retries must go with it. A
// cleanup that exists but is never called is the same as no cleanup.
func TestResetStopsOutstandingInforms(t *testing.T) {
	manager, sender, device := informManager(t)

	traps := &config.TrapConfig{Inform: true, InformRetries: 5, InformTimeoutSeconds: 1}
	manager.trackInform(device, traps, "10.0.0.99:162", 55, []byte("payload"))
	manager.Reset()

	select {
	case <-sender.sends:
		t.Error("a retry fired after the manager was reset")
	case <-time.After(1500 * time.Millisecond):
	}
}

// A receiver answers from the port it listens on, and the send path names the
// receiver with its port. Before #2472 the inform was tracked as "ip:162" and
// the wire acknowledged it as "ip", so no inform was ever acknowledged and every
// one was resent for its whole retry budget.
func TestReceiverResponseAcknowledgesTheSentInform(t *testing.T) {
	for _, receiver := range []string{"10.0.0.99", "10.0.0.99:162"} {
		t.Run(receiver, func(t *testing.T) {
			manager, _, device := informManager(t)
			device.SNMPConfig.Traps = &config.TrapConfig{
				Enabled: true, Inform: true, Receivers: []string{receiver}, InformTimeoutSeconds: 30,
			}
			manager.sendTrap(device, snmp.OIDColdStart, nil, 4242)
			t.Cleanup(manager.stopInforms)

			if !manager.AcknowledgeInform(4242, collector()) {
				t.Fatal("the receiver's Response did not acknowledge the inform sendTrap sent")
			}
		})
	}
}
