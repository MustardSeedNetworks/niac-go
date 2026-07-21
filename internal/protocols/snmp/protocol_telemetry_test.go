package snmp

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestProtocolTelemetryIsSharedAcrossCommunityAgents(t *testing.T) {
	device := createTestDevice()
	telemetry := NewProtocolTelemetry()
	public := NewAgentWithCommunityAndTelemetry(device, "public", 0, telemetry)
	private := NewAgentWithCommunityAndTelemetry(device, "private", 0, telemetry)

	telemetry.RecordInbound(ProtocolEvent{Protocol: protocolICMP, ICMPType: 8})
	telemetry.RecordInbound(ProtocolEvent{
		Protocol: protocolTCP, TCPSYN: true,
		SourceIP: "192.0.2.10", DestinationIP: "192.0.2.1",
		SourcePort: 40000, DestinationPort: 80,
	})
	telemetry.RecordInbound(ProtocolEvent{Protocol: protocolUDP})
	telemetry.RecordOutbound(ProtocolEvent{Protocol: protocolICMP, ICMPType: 0})
	telemetry.RecordOutbound(ProtocolEvent{
		Protocol: protocolTCP, TCPSYN: true, TCPACK: true,
		SourceIP: "192.0.2.1", DestinationIP: "192.0.2.10",
		SourcePort: 80, DestinationPort: 40000,
	})
	telemetry.RecordOutbound(ProtocolEvent{
		Protocol: protocolTCP, TCPRST: true,
		SourceIP: "192.0.2.1", DestinationIP: "192.0.2.10",
		SourcePort: 80, DestinationPort: 40000,
	})
	telemetry.RecordOutbound(ProtocolEvent{Protocol: protocolUDP})

	want := map[string]uint32{
		ipInReceives:          3,
		ipInDelivers:          3,
		ipOutRequests:         4,
		icmpMIBRoot + ".8.0":  1,
		icmpMIBRoot + ".22.0": 1,
		tcpMIBRoot + ".6.0":   1,
		tcpMIBRoot + ".10.0":  1,
		tcpMIBRoot + ".11.0":  2,
		tcpMIBRoot + ".15.0":  1,
		udpMIBRoot + ".1.0":   1,
		udpMIBRoot + ".4.0":   1,
	}
	for _, agent := range []*Agent{public, private} {
		for oid, expected := range want {
			value, err := agent.HandleGet(oid)
			if err != nil {
				t.Fatalf("GET %s: %v", oid, err)
			}
			if value.Value != expected {
				t.Errorf("%s = %v, want %d", oid, value.Value, expected)
			}
		}
	}
}

func TestProtocolTelemetryFragmentEvidence(t *testing.T) {
	device := createTestDevice()
	telemetry := NewProtocolTelemetry()
	agent := NewAgentWithCommunityAndTelemetry(device, "public", 0, telemetry)

	telemetry.RecordInbound(ProtocolEvent{Protocol: protocolUDP, MoreFragments: true})
	telemetry.RecordInbound(ProtocolEvent{Protocol: protocolUDP, FragmentOffset: 185})
	telemetry.RecordOutbound(ProtocolEvent{Protocol: protocolUDP, MoreFragments: true})

	for oid, want := range map[string]uint32{ipReasmReqds: 2, ipFragCreates: 1} {
		value, err := agent.HandleGet(oid)
		if err != nil {
			t.Fatalf("GET %s: %v", oid, err)
		}
		if value.Value != want {
			t.Errorf("%s = %v, want %d", oid, value.Value, want)
		}
	}
}

func TestWalkLoadPreservesLiveProtocolCounters(t *testing.T) {
	device := createTestDevice()
	telemetry := NewProtocolTelemetry()
	agent := NewAgentWithCommunityAndTelemetry(device, "public", 0, telemetry)
	walkPath := filepath.Join(t.TempDir(), "captured.walk")
	walk := `.1.3.6.1.2.1.4.3.0 = Counter32: 900
.1.3.6.1.2.1.5.1.0 = Counter32: 901
.1.3.6.1.2.1.6.10.0 = Counter32: 902
.1.3.6.1.2.1.7.1.0 = Counter32: 903
`
	if err := os.WriteFile(walkPath, []byte(walk), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := agent.LoadWalkFile(walkPath); err != nil {
		t.Fatal(err)
	}

	telemetry.RecordInbound(ProtocolEvent{Protocol: protocolICMP, ICMPType: 8})
	telemetry.RecordInbound(ProtocolEvent{Protocol: protocolTCP})
	telemetry.RecordInbound(ProtocolEvent{Protocol: protocolUDP})

	for oid, want := range map[string]uint32{
		ipInReceives: 3, icmpMIBRoot + ".1.0": 1,
		tcpMIBRoot + ".10.0": 1, udpMIBRoot + ".1.0": 1,
	} {
		assertProtocolCounter(t, agent, oid, want)
	}
}

func TestProtocolTelemetryConcurrentUpdates(t *testing.T) {
	const workers = 100
	telemetry := NewProtocolTelemetry()
	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			telemetry.RecordInbound(ProtocolEvent{Protocol: protocolUDP})
			telemetry.RecordOutbound(ProtocolEvent{Protocol: protocolUDP})
		}()
	}
	wg.Wait()

	if got := telemetry.udpInDatagrams.Load(); got != workers {
		t.Errorf("udpInDatagrams = %d, want %d", got, workers)
	}
	if got := telemetry.udpOutDatagrams.Load(); got != workers {
		t.Errorf("udpOutDatagrams = %d, want %d", got, workers)
	}
}
