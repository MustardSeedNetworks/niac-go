package protocols

import (
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
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
	manager.trackInform(device, traps, "10.0.0.99", 4242, []byte("payload"))

	if !manager.AcknowledgeInform(4242, "10.0.0.99") {
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
	manager.trackInform(device, traps, "10.0.0.99", 7, []byte("payload"))

	if manager.AcknowledgeInform(7, "10.0.0.100") {
		t.Error("an acknowledgement from a different receiver cleared the inform")
	}
	if manager.AcknowledgeInform(8, "10.0.0.99") {
		t.Error("an acknowledgement for a different request ID cleared the inform")
	}
	if !manager.AcknowledgeInform(7, "10.0.0.99") {
		t.Error("the matching acknowledgement did not clear the inform")
	}
}

// An unacknowledged inform is resent, then given up on. Retrying forever would
// keep a dead receiver's notification on the wire for the life of the session.
func TestUnacknowledgedInformRetriesThenGivesUp(t *testing.T) {
	manager, sender, device := informManager(t)

	traps := &config.TrapConfig{Inform: true, InformRetries: 2, InformTimeoutSeconds: 1}
	manager.trackInform(device, traps, "10.0.0.99", 99, []byte("payload"))

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
	manager.trackInform(device, traps, "10.0.0.99", 1, []byte("payload"))
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
	manager.trackInform(device, traps, "10.0.0.99", 55, []byte("payload"))
	manager.Reset()

	select {
	case <-sender.sends:
		t.Error("a retry fired after the manager was reset")
	case <-time.After(1500 * time.Millisecond):
	}
}
