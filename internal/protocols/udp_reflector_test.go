package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// buildReflectorProbe serialises a UDP reflector probe from tester -> reflector
// and returns a Packet (whose Buffer carries the Ethernet header so
// GetSourceMAC works), plus the parsed IPv4/UDP layers tryReflect consumes.
func buildReflectorProbe(
	t *testing.T,
	testerMAC, reflMAC net.HardwareAddr,
	testerIP, reflIP net.IP,
	tos uint8,
	vlan int,
	payload []byte,
) (*Packet, *layers.IPv4, *layers.UDP) {
	t.Helper()

	eth := &layers.Ethernet{SrcMAC: testerMAC, DstMAC: reflMAC, EthernetType: layers.EthernetTypeIPv4}
	ip := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TOS:      tos,
		TTL:      64,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    testerIP.To4(),
		DstIP:    reflIP.To4(),
	}
	udp := &layers.UDP{SrcPort: 45000, DstPort: 3842}

	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("SetNetworkLayerForChecksum: %v", err)
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}

	if err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, gopacket.Payload(payload)); err != nil {
		t.Fatalf("SerializeLayers: %v", err)
	}

	pkt := &Packet{Buffer: buf.Bytes(), Length: len(buf.Bytes()), SerialNumber: 1, VLAN: vlan}
	// Re-decode the UDP layer so udp.Payload is populated as it is at runtime.
	udp.Payload = payload

	return pkt, ip, udp
}

// reflectorProbePayload returns a payload with sig at reflectorSigOffset.
func reflectorProbePayload(sig string) []byte {
	p := make([]byte, reflectorSigOffset+len(sig))
	copy(p[reflectorSigOffset:], sig)

	return p
}

func TestReflectorSignatureMatch(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    bool
	}{
		{"data signature", reflectorProbePayload("DATA:OT\x00"), true},
		{"probe signature", reflectorProbePayload("PROBEOT\x00"), true},
		{"wrong signature", reflectorProbePayload("NOPE:XX\x00"), false},
		{"signature at wrong offset", append([]byte("DATA:OT\x00"), 0, 0), false},
		{"too short", []byte("DATA"), false},
		{"empty", nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := reflectorSignatureMatch(tc.payload); got != tc.want {
				t.Errorf("reflectorSignatureMatch = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWiggleTOS(t *testing.T) {
	if got := wiggleTOS(0x00, false); got != reflectorWiggleIPPrec {
		t.Errorf("IP-precedence wiggle of 0x00 = %#x, want %#x", got, reflectorWiggleIPPrec)
	}

	if got := wiggleTOS(0x00, true); got != reflectorWiggleDSCP {
		t.Errorf("DSCP wiggle of 0x00 = %#x, want %#x", got, reflectorWiggleDSCP)
	}

	// XOR is its own inverse: wiggling twice restores the original.
	if got := wiggleTOS(wiggleTOS(0xB8, true), true); got != 0xB8 {
		t.Errorf("double DSCP wiggle = %#x, want 0xB8", got)
	}
}

func TestReflectorDelay(t *testing.T) {
	if d := reflectorDelay(&config.ReflectorConfig{}); d != 0 {
		t.Errorf("zero config delay = %v, want 0", d)
	}

	// Jitter alone stays within [0, jitter].
	for range 100 {
		d := reflectorDelay(&config.ReflectorConfig{JitterMs: 10}).Milliseconds()
		if d < 0 || d > 10 {
			t.Fatalf("jitter-only delay %d out of [0,10]", d)
		}
	}

	// Latency + jitter stays within [latency-jitter, latency+jitter], floored at 0.
	for range 100 {
		d := reflectorDelay(&config.ReflectorConfig{LatencyMs: 50, JitterMs: 5}).Milliseconds()
		if d < 45 || d > 55 {
			t.Fatalf("latency+jitter delay %d out of [45,55]", d)
		}
	}
}

func TestTryReflect(t *testing.T) {
	stack := newTestStackInternal(t)
	handler := NewUDPHandler(stack)

	testerMAC := net.HardwareAddr{0xAA, 0xAA, 0xAA, 0x00, 0x00, 0x01}
	reflMAC := net.HardwareAddr{0xBB, 0xBB, 0xBB, 0x00, 0x00, 0x02}
	testerIP := net.ParseIP("10.20.200.250")
	reflIP := net.ParseIP("10.20.200.100")

	device := &config.Device{
		Name:            "reflector",
		MACAddress:      reflMAC,
		IPAddresses:     []net.IP{reflIP},
		ReflectorConfig: &config.ReflectorConfig{}, // immediate, IP-precedence wiggle
	}
	devices := []*config.Device{device}

	const probeTOS = 0x00

	pkt, ip, udp := buildReflectorProbe(
		t,
		testerMAC,
		reflMAC,
		testerIP,
		reflIP,
		probeTOS,
		200,
		reflectorProbePayload("DATA:OT\x00"),
	)

	if !handler.tryReflect(pkt, ip, udp, devices) {
		t.Fatal("tryReflect returned false for a valid probe")
	}

	var reply *Packet
	select {
	case reply = <-stack.sendQueue:
	default:
		t.Fatal("no reflected packet was queued")
	}

	if reply.VLAN != 200 {
		t.Errorf("reflected VLAN = %d, want 200 (echo request VLAN)", reply.VLAN)
	}

	// Decode the reflected frame and assert the swap.
	got := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)

	ethOut, _ := got.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if ethOut == nil {
		t.Fatal("reflected packet missing Ethernet layer")
	}

	if ethOut.SrcMAC.String() != reflMAC.String() || ethOut.DstMAC.String() != testerMAC.String() {
		t.Errorf("reflected MACs src=%s dst=%s, want src=%s dst=%s",
			ethOut.SrcMAC, ethOut.DstMAC, reflMAC, testerMAC)
	}

	ipOut, _ := got.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	if ipOut == nil {
		t.Fatal("reflected packet missing IPv4 layer")
	}

	if !ipOut.SrcIP.Equal(reflIP) || !ipOut.DstIP.Equal(testerIP) {
		t.Errorf("reflected IPs src=%s dst=%s, want src=%s dst=%s",
			ipOut.SrcIP, ipOut.DstIP, reflIP, testerIP)
	}

	if ipOut.TOS != reflectorWiggleIPPrec {
		t.Errorf("reflected TOS = %#x, want %#x (wiggled from 0x00)", ipOut.TOS, reflectorWiggleIPPrec)
	}

	udpOut, _ := got.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if udpOut == nil {
		t.Fatal("reflected packet missing UDP layer")
	}

	// Ports are intentionally NOT swapped, matching niac-java.
	if udpOut.SrcPort != udp.SrcPort || udpOut.DstPort != udp.DstPort {
		t.Errorf("reflected ports src=%d dst=%d, want src=%d dst=%d (unswapped)",
			udpOut.SrcPort, udpOut.DstPort, udp.SrcPort, udp.DstPort)
	}
}

func TestTryReflectSkips(t *testing.T) {
	stack := newTestStackInternal(t)
	handler := NewUDPHandler(stack)

	reflMAC := net.HardwareAddr{0xBB, 0xBB, 0xBB, 0x00, 0x00, 0x02}
	reflIP := net.ParseIP("10.20.200.100")
	testerMAC := net.HardwareAddr{0xAA, 0xAA, 0xAA, 0x00, 0x00, 0x01}
	testerIP := net.ParseIP("10.20.200.250")

	reflector := &config.Device{
		Name: "reflector", MACAddress: reflMAC, IPAddresses: []net.IP{reflIP},
		ReflectorConfig: &config.ReflectorConfig{},
	}
	plain := &config.Device{
		Name: "plain", MACAddress: reflMAC, IPAddresses: []net.IP{reflIP},
	}

	t.Run("no reflector device", func(t *testing.T) {
		pkt, ip, udp := buildReflectorProbe(
			t,
			testerMAC,
			reflMAC,
			testerIP,
			reflIP,
			0,
			0,
			reflectorProbePayload("DATA:OT\x00"),
		)
		if handler.tryReflect(pkt, ip, udp, []*config.Device{plain}) {
			t.Error("reflected for a non-reflector device")
		}
	})

	t.Run("no signature", func(t *testing.T) {
		pkt, ip, udp := buildReflectorProbe(
			t,
			testerMAC,
			reflMAC,
			testerIP,
			reflIP,
			0,
			0,
			[]byte("just some udp payload"),
		)
		if handler.tryReflect(pkt, ip, udp, []*config.Device{reflector}) {
			t.Error("reflected a non-probe payload")
		}
	})
}
