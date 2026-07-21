package snmp

import (
	"sync"
	"time"
)

const (
	tcpFlowSynSent = iota + 1
	tcpFlowSynReceived
	tcpFlowEstablished
	tcpFlowCloseWait
	tcpFlowHandshakeTimeout = 75 * time.Second
)

type tcpFlowKey struct {
	localIP, remoteIP     string
	localPort, remotePort uint16
}

type tcpFlow struct {
	state int
	seen  time.Time
}

type tcpFlowSnapshot struct {
	key   tcpFlowKey
	state int
}

type tcpFlowTracker struct {
	mu    sync.Mutex
	flows map[tcpFlowKey]tcpFlow
	now   func() time.Time
}

func newTCPFlowTracker() *tcpFlowTracker {
	return &tcpFlowTracker{flows: make(map[tcpFlowKey]tcpFlow), now: time.Now}
}

func (t *tcpFlowTracker) record(event ProtocolEvent, inbound bool) (uint32, uint32, bool) {
	attemptFails, estabResets := uint32(0), uint32(0)
	if t == nil || event.SourceIP == "" || event.DestinationIP == "" {
		return 0, 0, false
	}
	now := t.now()
	t.mu.Lock()
	defer t.mu.Unlock()
	for key, flow := range t.flows {
		if (flow.state == tcpFlowSynSent || flow.state == tcpFlowSynReceived) &&
			now.Sub(flow.seen) >= tcpFlowHandshakeTimeout {
			delete(t.flows, key)
			attemptFails++
		}
	}
	key := tcpEventKey(event, inbound)
	flow, exists := t.flows[key]
	if event.TCPRST {
		if !exists {
			return attemptFails, estabResets, false
		}
		if flow.state == tcpFlowEstablished || flow.state == tcpFlowCloseWait {
			estabResets++
		} else {
			attemptFails++
		}
		delete(t.flows, key)
		return attemptFails, estabResets, false
	}
	if inbound {
		return t.recordInbound(key, flow, exists, event, now, attemptFails, estabResets)
	}
	return t.recordOutbound(key, flow, exists, event, now, attemptFails, estabResets)
}

func (t *tcpFlowTracker) recordInbound(
	key tcpFlowKey,
	flow tcpFlow,
	exists bool,
	event ProtocolEvent,
	now time.Time,
	attemptFails, estabResets uint32,
) (uint32, uint32, bool) {
	switch {
	case event.TCPSYN && !event.TCPACK:
		if !exists {
			t.flows[key] = tcpFlow{state: tcpFlowSynReceived, seen: now}
		}
	case event.TCPSYN && event.TCPACK && exists && flow.state == tcpFlowSynSent:
		flow.state, flow.seen = tcpFlowEstablished, now
		t.flows[key] = flow
	case event.TCPACK && exists && flow.state == tcpFlowSynReceived:
		flow.state, flow.seen = tcpFlowEstablished, now
		t.flows[key] = flow
	case event.TCPFIN && exists && flow.state == tcpFlowEstablished:
		flow.state, flow.seen = tcpFlowCloseWait, now
		t.flows[key] = flow
	}
	return attemptFails, estabResets, false
}

func (t *tcpFlowTracker) recordOutbound(
	key tcpFlowKey,
	flow tcpFlow,
	exists bool,
	event ProtocolEvent,
	now time.Time,
	attemptFails, estabResets uint32,
) (uint32, uint32, bool) {
	opened := false
	switch {
	case event.TCPSYN && !event.TCPACK:
		if !exists {
			t.flows[key] = tcpFlow{state: tcpFlowSynSent, seen: now}
			opened = true
		}
	case event.TCPSYN && event.TCPACK:
		if !exists {
			t.flows[key] = tcpFlow{state: tcpFlowSynReceived, seen: now}
			opened = true
		} else if flow.state == tcpFlowSynReceived {
			flow.seen = now
			t.flows[key] = flow
			opened = true
		}
	case event.TCPFIN && exists:
		delete(t.flows, key)
	}
	return attemptFails, estabResets, opened
}

func tcpEventKey(event ProtocolEvent, inbound bool) tcpFlowKey {
	if inbound {
		return tcpFlowKey{event.DestinationIP, event.SourceIP, event.DestinationPort, event.SourcePort}
	}
	return tcpFlowKey{event.SourceIP, event.DestinationIP, event.SourcePort, event.DestinationPort}
}

func (t *tcpFlowTracker) currentEstablished() uint32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	var count uint32
	for _, flow := range t.flows {
		if flow.state == tcpFlowEstablished || flow.state == tcpFlowCloseWait {
			count++
		}
	}
	return count
}

func (t *tcpFlowTracker) snapshot() []tcpFlowSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make([]tcpFlowSnapshot, 0, len(t.flows))
	for key, flow := range t.flows {
		result = append(result, tcpFlowSnapshot{key: key, state: flow.state})
	}
	return result
}
