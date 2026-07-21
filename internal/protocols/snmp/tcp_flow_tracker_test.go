package snmp

import (
	"testing"
	"time"
)

func TestTCPFlowTrackerLifecycleCounters(t *testing.T) {
	telemetry := NewProtocolTelemetry()
	agent := NewAgentWithCommunityAndTelemetry(createTestDevice(), "public", 0, telemetry)
	synAck := tcpFlowEvent(false)
	synAck.TCPSYN, synAck.TCPACK = true, true
	telemetry.RecordOutbound(synAck)
	telemetry.RecordOutbound(synAck) // retransmit must not create another flow

	ack := tcpFlowEvent(true)
	ack.TCPACK = true
	telemetry.RecordInbound(ack)
	assertProtocolCounter(t, agent, tcpMIBRoot+".9.0", 1)

	rst := tcpFlowEvent(true)
	rst.TCPRST = true
	telemetry.RecordInbound(rst)
	assertProtocolCounter(t, agent, tcpMIBRoot+".8.0", 1)
	assertProtocolCounter(t, agent, tcpMIBRoot+".9.0", 0)
}

func TestTCPFlowTrackerHandshakeExpiry(t *testing.T) {
	telemetry := NewProtocolTelemetry()
	agent := NewAgentWithCommunityAndTelemetry(createTestDevice(), "public", 0, telemetry)
	now := time.Unix(100, 0)
	telemetry.tcpFlows.now = func() time.Time { return now }
	synAck := tcpFlowEvent(false)
	synAck.TCPSYN, synAck.TCPACK = true, true
	telemetry.RecordOutbound(synAck)
	now = now.Add(tcpFlowHandshakeTimeout)
	telemetry.RecordInbound(tcpFlowEvent(true))
	assertProtocolCounter(t, agent, tcpMIBRoot+".7.0", 1)
	assertProtocolCounter(t, agent, tcpMIBRoot+".9.0", 0)
}

func TestTCPFlowTrackerPublishesConnectionTableToAgent(t *testing.T) {
	telemetry := NewProtocolTelemetry()
	agent := NewAgentWithCommunityAndTelemetry(createTestDevice(), "public", 0, telemetry)
	syn := tcpFlowEvent(true)
	telemetry.RecordInbound(syn)
	synAck := syn
	synAck.SourceIP, synAck.DestinationIP = syn.DestinationIP, syn.SourceIP
	synAck.SourcePort, synAck.DestinationPort = syn.DestinationPort, syn.SourcePort
	synAck.TCPSYN, synAck.TCPACK = true, true
	telemetry.RecordOutbound(synAck)
	index := tcpAddressIndex(tcpFlowKey{
		localIP: synAck.SourceIP, remoteIP: synAck.DestinationIP,
		localPort: synAck.SourcePort, remotePort: synAck.DestinationPort,
	})
	value, err := agent.HandleGet(tcpMIBRoot + ".13.1." + index + ".1")
	if err != nil || value == nil || value.Value != tcpFlowSynReceived {
		t.Fatalf("connection table state = %#v, err=%v", value, err)
	}
}

func tcpFlowEvent(inbound bool) ProtocolEvent {
	event := ProtocolEvent{
		Protocol: protocolTCP, SourceIP: "192.0.2.10", DestinationIP: "192.0.2.1",
		SourcePort: 40000, DestinationPort: 80,
	}
	if !inbound {
		event.SourceIP, event.DestinationIP = event.DestinationIP, event.SourceIP
		event.SourcePort, event.DestinationPort = event.DestinationPort, event.SourcePort
	}
	return event
}

func assertProtocolCounter(t *testing.T, agent *Agent, oid string, want uint32) {
	t.Helper()
	value, err := agent.HandleGet(oid)
	if err != nil {
		t.Fatalf("GET %s: %v", oid, err)
	}
	if value.Value != want {
		t.Errorf("%s = %v, want %d", oid, value.Value, want)
	}
}
