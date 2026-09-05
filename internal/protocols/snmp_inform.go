package protocols

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// SNMP informs: notifications the receiver acknowledges.
//
// A trap is fire-and-forget, so a dropped one is invisible to both ends — the
// device believes it reported a link failure and the manager never heard of it.
// An inform is answered with a Response carrying the same request ID, and an
// unanswered one is resent. That acknowledgement is the whole point, so the
// send path is only half of it: without somewhere to notice the reply, an
// inform is a trap that retries blindly.

// pendingInform is one inform awaiting acknowledgement.
type pendingInform struct {
	device    *config.Device
	receiver  string
	payload   []byte
	remaining int
	timer     *time.Timer
}

// informTracker holds the informs this stack is waiting on, keyed by request
// ID and receiver so two receivers of the same notification are tracked apart.
type informTracker struct {
	mu      sync.Mutex
	pending map[informKey]*pendingInform
}

type informKey struct {
	requestID uint32
	receiver  string
}

func newInformTracker() *informTracker {
	return &informTracker{pending: make(map[informKey]*pendingInform)}
}

// Default retry behaviour, used when the config says nothing. Three attempts
// five seconds apart is what common managers use, and it bounds a failed
// receiver to fifteen seconds of retries rather than forever.
const (
	defaultInformRetries = 3
	defaultInformTimeout = 5 * time.Second
)

// trackInform records an inform and arms its retry.
func (m *stateNotificationManager) trackInform(
	device *config.Device,
	traps *config.TrapConfig,
	receiver string,
	requestID uint32,
	payload []byte,
) {
	if m.informs == nil {
		return
	}

	retries := traps.InformRetries
	if retries == 0 {
		retries = defaultInformRetries
	}
	timeout := time.Duration(traps.InformTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultInformTimeout
	}

	key := informKey{requestID: requestID, receiver: receiver}
	entry := &pendingInform{
		device: device, receiver: receiver, payload: payload, remaining: retries,
	}

	m.informs.mu.Lock()
	defer m.informs.mu.Unlock()
	if existing, waiting := m.informs.pending[key]; waiting {
		// The same notification to the same receiver is already in flight;
		// re-arming would double the retries rather than extend them.
		existing.timer.Reset(timeout)

		return
	}
	entry.timer = time.AfterFunc(timeout, func() { m.retryInform(key, timeout) })
	m.informs.pending[key] = entry
}

// retryInform resends an unacknowledged inform, or gives up and says so.
func (m *stateNotificationManager) retryInform(key informKey, timeout time.Duration) {
	m.informs.mu.Lock()
	entry, waiting := m.informs.pending[key]
	if !waiting {
		m.informs.mu.Unlock()

		return
	}
	if entry.remaining <= 0 {
		delete(m.informs.pending, key)
		m.informs.mu.Unlock()
		// Reported rather than dropped silently: an inform nobody answered
		// means the manager did not learn what the device was telling it.
		slog.Warn("SNMP inform was never acknowledged",
			"device", entry.device.Name, "receiver", entry.receiver, "requestId", key.requestID)

		return
	}
	entry.remaining--
	entry.timer.Reset(timeout)
	device, receiver, payload := entry.device, entry.receiver, entry.payload
	m.informs.mu.Unlock()

	m.send(device, receiver, snmp.DefaultSNMPTrapPort, payload)
}

// AcknowledgeInform stops the retry for the inform this response answers.
func (m *stateNotificationManager) AcknowledgeInform(requestID uint32, receiver string) bool {
	if m == nil || m.informs == nil {
		return false
	}

	key := informKey{requestID: requestID, receiver: receiver}

	m.informs.mu.Lock()
	defer m.informs.mu.Unlock()
	entry, waiting := m.informs.pending[key]
	if !waiting {
		return false
	}
	entry.timer.Stop()
	delete(m.informs.pending, key)

	return true
}

// stopInforms cancels every outstanding retry, so a stopped stack does not keep
// resending into a network it no longer has.
func (m *stateNotificationManager) stopInforms() {
	if m == nil || m.informs == nil {
		return
	}

	m.informs.mu.Lock()
	defer m.informs.mu.Unlock()
	for key, entry := range m.informs.pending {
		entry.timer.Stop()
		delete(m.informs.pending, key)
	}
}

// handleInformResponse acknowledges the inform a Response PDU answers.
//
// The response arrives on the port the inform was sent from (162), which is
// also where a device receiving traps would listen — so this is dispatched on
// the PDU type rather than the port.
func (h *UDPHandler) handleInformResponse(payload []byte, source string) bool {
	decoder := &gosnmp.GoSNMP{Version: gosnmp.Version2c}
	packet, err := decoder.SnmpDecodePacket(payload)
	if err != nil || packet.PDUType != gosnmp.GetResponse {
		return false
	}

	return h.stack.acknowledgeInform(packet.RequestID, source)
}
