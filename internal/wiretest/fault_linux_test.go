//go:build linux && integration

package wiretest_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// P2-1's acceptance asks that an injected fault be observable on the wire, and
// this is the assertion that can only be made here: every unit test around the
// fault store proves the flag is set, and none of them proves a client sees
// nothing come back.
//
// The fault is authored rather than armed. testdata/fault-timeline.yaml
// declares a behavior phase whose fault names no interface, which is the
// device-scoped axis, so the outage arrives through the same YAML an operator
// writes and this test needs no accessor into a running session's stack.
const (
	faultRouterName   = "LAB-FAULT-R1"
	faultRouterIP     = "10.254.200.1"
	faultPhaseSeconds = 15 // testdata/fault-timeline.yaml: duration_ms

	// The resolver both DNS phases are armed on, and a name it authors an A
	// record for -- so the NXDOMAIN is the fault rewriting a successful
	// answer, not an empty zone answering normally.
	faultDNSAddr     = "10.254.200.2"
	faultDNSName     = "host.fault.example"
	faultDNSRecordIP = "10.254.200.90"

	// testdata/fault-timeline.yaml: the dns-outage phases' start_offset_ms.
	// The timeout phase resets at 25s, so the recovery lookup has the window
	// between it and the nxdomain phase to itself.
	faultDNSTimeoutStart = 10 * time.Second
	faultDNSTimeoutEnd   = 25 * time.Second
	faultNXDomainStart   = 40 * time.Second

	// The device whose only fault is latency, and the delay its phase authors.
	// The assertion is a lower bound: a reply that took at least this long
	// minus a tolerance was held back, and no upper bound is asserted at all,
	// so a loaded runner cannot fail it.
	faultSlowAddr         = "10.254.200.4"
	faultSlowMAC          = "02:00:00:00:fa:04"
	faultLatencyStart     = 10 * time.Second
	faultLatencyAuthored  = 3000 * time.Millisecond
	faultLatencyTolerance = 200 * time.Millisecond

	// How long a healthy lookup is given before it counts as lost.
	faultResolveWindow = 20 * time.Second

	// The echo request is retransmitted per attempt window; the deadline caps
	// the whole exchange. The attempt window has to clear the authored delay,
	// or the faulted reply would arrive after the attempt that asked for it.
	faultEchoAttemptWindow = 10 * time.Second
	faultEchoDeadline      = 30 * time.Second

	// Attempt sequence numbers are strided by call, so a reply to a retransmit
	// of the healthy ping can never be mistaken for the faulted one.
	faultEchoSeqStride = 100

	// wire_linux_test.go: clientCIDR, the address the kernel holds on the test
	// end of the veth and the source a reply is sent back to.
	faultClientAddr = "10.254.200.50"

	// The window the DISCOVER goes unanswered in. Well inside the phase, so a
	// slow start cannot make a legitimate silence look like a late answer.
	faultSilenceWindow = 8 * time.Second
)

func startFaultTimeline(t *testing.T) (*config.Config, time.Time) {
	t.Helper()
	requireWire(t)

	yamlBytes, err := os.ReadFile(filepath.Join("testdata", "fault-timeline.yaml"))
	if err != nil {
		t.Fatalf("reading the fault timeline network: %v", err)
	}
	authored, err := config.LoadYAMLBytes(yamlBytes)
	if err != nil {
		t.Fatalf("loading the fault timeline network: %v", err)
	}

	// Without this the daemon persists the inline config into the invoking
	// user's real ~/.niac/configs.
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())

	d, err := daemon.NewDaemon(daemon.Config{
		StoragePath: "disabled",
		AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
			Interface: simIface, Mode: fabric.ModeAccess, AccessVLAN: accessVLAN,
		}},
	})
	if err != nil {
		t.Fatalf("daemon.NewDaemon: %v", err)
	}
	started := time.Now()
	if startErr := d.StartSimulation(api.SimulationRequest{
		SessionID:      "wiretest-fault",
		Interface:      simIface,
		Attachment:     "tester",
		AttachmentMode: fabric.ModeAccess,
		AccessVLAN:     accessVLAN,
		ConfigData:     string(yamlBytes),
	}); startErr != nil {
		t.Fatalf("StartSimulation on %s: %v", simIface, startErr)
	}
	t.Cleanup(func() {
		if stopErr := d.StopSimulation(""); stopErr != nil {
			t.Errorf("StopSimulation: %v", stopErr)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = d.Shutdown(ctx)
	})

	return authored, started
}

// Both halves in one session, because either alone is weak: silence on its own
// is what a broken DHCP server also produces, and a lease on its own says
// nothing about the fault. The phase clears itself, so the same server that
// went quiet answers afterwards.
func TestAuthoredDeviceFaultSuppressesDHCPOnTheWire(t *testing.T) {
	authored, started := startFaultTimeline(t)
	router := faultRouter(t, authored)
	if router.DHCPConfig == nil {
		t.Fatalf("%s has no authored DHCP server; the fixture no longer serves leases",
			faultRouterName)
	}

	handle := openClient(t)
	src := clientMAC(t)

	if offer := awaitDHCPOffer(t, handle, src, 0x9e57fa01, faultSilenceWindow); offer != nil {
		t.Fatalf(
			"DHCPOFFER of %s arrived while the authored dhcp_no_offer phase was active",
			offer.YourClientIP,
		)
	}

	// The phase is a reset phase, so its end clears the fault. Wait for the
	// authored duration to elapse from the session start rather than from the
	// silence window, which is what the timeline itself is measured against.
	if remaining := time.Until(
		started.Add(faultPhaseSeconds * time.Second),
	); remaining > 0 {
		time.Sleep(remaining)
	}

	offer := dhcpExchange(t, handle, src, 0x9e57fa02, layers.DHCPMsgTypeDiscover, nil)
	offered := append(net.IP(nil), offer.YourClientIP...)
	if !addrWithin(offered, router.DHCPConfig.PoolStart, router.DHCPConfig.PoolEnd) {
		t.Fatalf(
			"DHCPOFFER after the phase = %s, outside the authored pool %s-%s",
			offered, router.DHCPConfig.PoolStart, router.DHCPConfig.PoolEnd,
		)
	}
}

func faultRouter(t *testing.T, cfg *config.Config) *config.Device {
	t.Helper()
	for index := range cfg.Devices {
		if cfg.Devices[index].Name == faultRouterName {
			return &cfg.Devices[index]
		}
	}
	t.Fatalf("no device named %q in the fault timeline network", faultRouterName)

	return nil
}

// awaitDHCPOffer sends a DISCOVER and returns the OFFER, or nil when the
// window closes with none. It is dhcpExchange's assertion inverted: the same
// per-second retransmit a real client performs, so a single dropped frame
// cannot masquerade as a suppressed one.
func awaitDHCPOffer(
	t *testing.T,
	handle *pcap.Handle,
	src net.HardwareAddr,
	xid uint32,
	window time.Duration,
) *layers.DHCPv4 {
	t.Helper()

	frame := discoverFrame(t, src, xid)
	packets := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	deadline := time.After(window)
	retry := time.NewTicker(time.Second)
	defer retry.Stop()
	if err := handle.WritePacketData(frame); err != nil {
		t.Fatalf("writing DHCPDISCOVER: %v", err)
	}
	for {
		select {
		case packet := <-packets:
			layer := packet.Layer(layers.LayerTypeDHCPv4)
			if layer == nil {
				continue
			}
			reply, ok := layer.(*layers.DHCPv4)
			if !ok || reply.Xid != xid || reply.Operation != layers.DHCPOpReply {
				continue
			}
			if dhcpMessageType(reply) == layers.DHCPMsgTypeOffer {
				return reply
			}
		case <-retry.C:
			if err := handle.WritePacketData(frame); err != nil {
				t.Fatalf("re-writing DHCPDISCOVER: %v", err)
			}
		case <-deadline:
			return nil
		}
	}
}

func discoverFrame(t *testing.T, src net.HardwareAddr, xid uint32) []byte {
	t.Helper()

	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  6,
		Xid:          xid,
		ClientHWAddr: src,
		Flags:        0x8000, // broadcast: the client has no lease to be unicast at
		Options: []layers.DHCPOption{{
			Type: layers.DHCPOptMessageType, Data: []byte{byte(layers.DHCPMsgTypeDiscover)},
			Length: 1,
		}},
	}
	eth := &layers.Ethernet{
		SrcMAC:       src,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    net.IPv4zero,
		DstIP:    net.IPv4bcast,
	}
	udp := &layers.UDP{SrcPort: 68, DstPort: 67}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("dhcp checksum setup: %v", err)
	}

	return serialize(t, eth, ip, udp, dhcp)
}

// NXDOMAIN is the other half of the axis: the device answers, and the answer
// itself is the fault. The healthy lookup comes first, so a query the server
// could not read -- which it reports as NXDOMAIN too -- fails here rather than
// passing as a fault it never saw.
func TestAuthoredDeviceFaultRewritesDNSOnTheWire(t *testing.T) {
	_, started := startFaultTimeline(t)

	conn, err := net.DialTimeout("udp", faultDNSAddr+":53", 5*time.Second)
	if err != nil {
		t.Fatalf("dialling the simulated resolver: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	healthy := resolveOverTheWire(t, conn, 0xfa01)
	if healthy.ResponseCode != layers.DNSResponseCodeNoErr || len(healthy.Answers) == 0 {
		t.Fatalf(
			"healthy lookup of %s = %v with %d answers, want NoError with the authored A record",
			faultDNSName, healthy.ResponseCode, len(healthy.Answers),
		)
	}
	if got := healthy.Answers[0].IP.String(); got != faultDNSRecordIP {
		t.Fatalf("healthy lookup answered %s, want the authored %s", got, faultDNSRecordIP)
	}

	// The phase begins at its authored offset from the session start.
	sleepUntil(started.Add(faultNXDomainStart + 2*time.Second))

	faulted := resolveOverTheWire(t, conn, 0xfa02)
	if faulted.ResponseCode != layers.DNSResponseCodeNXDomain {
		t.Fatalf(
			"lookup of %s under the authored dns_nxdomain phase = %v, want NXDomain",
			faultDNSName, faulted.ResponseCode,
		)
	}
	if len(faulted.Answers) != 0 {
		t.Fatalf("NXDOMAIN carried %d answers, want none", len(faulted.Answers))
	}
}

// resolveOverTheWire asks the simulated resolver for the authored name and
// fails when nothing comes back. It is tryResolveOverTheWire with the silence
// treated as the error it is everywhere except the timeout fault's own window.
func resolveOverTheWire(t *testing.T, conn net.Conn, id uint16) layers.DNS {
	t.Helper()

	response, answered := tryResolveOverTheWire(t, conn, id, faultResolveWindow)
	if !answered {
		t.Fatalf("no DNS response from %s within %s", faultDNSAddr, faultResolveWindow)
	}

	return response
}

// tryResolveOverTheWire asks the simulated resolver for the authored name and
// reports whether it answered, retransmitting the way a resolver does: the
// first datagram can be lost to ARP resolution on a fresh veth, and under the
// timeout fault every datagram goes unanswered by design.
func tryResolveOverTheWire(
	t *testing.T, conn net.Conn, id uint16, window time.Duration,
) (layers.DNS, bool) {
	t.Helper()

	query := &layers.DNS{
		ID: id, RD: true,
		Questions: []layers.DNSQuestion{{
			Name: []byte(faultDNSName), Type: layers.DNSTypeA, Class: layers.DNSClassIN,
		}},
	}
	buf := gopacket.NewSerializeBuffer()
	// FixLengths, or QDCount stays zero and the server answers a question it
	// was never asked -- as NXDOMAIN, which is the value under test.
	if serErr := query.SerializeTo(
		buf, gopacket.SerializeOptions{FixLengths: true},
	); serErr != nil {
		t.Fatalf("serializing the query: %v", serErr)
	}

	deadline := time.Now().Add(window)
	for time.Now().Before(deadline) {
		if _, writeErr := conn.Write(buf.Bytes()); writeErr != nil {
			t.Fatalf("sending the query: %v", writeErr)
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		reply := make([]byte, 1500)
		read, readErr := conn.Read(reply)
		if readErr != nil {
			continue
		}
		var response layers.DNS
		if decodeErr := response.DecodeFromBytes(
			reply[:read], gopacket.NilDecodeFeedback,
		); decodeErr != nil {
			t.Fatalf("decoding the response: %v", decodeErr)
		}
		// A reply to an earlier query -- the one the retransmit replaced --
		// would otherwise be read as an answer to this one.
		if response.ID != id {
			continue
		}

		return response, true
	}

	return layers.DNS{}, false
}

// sleepUntil waits for a moment measured from the session start, which is what
// the authored timeline's offsets are measured against too.
func sleepUntil(moment time.Time) {
	if remaining := time.Until(moment); remaining > 0 {
		time.Sleep(remaining)
	}
}

// The timeout fault is the silence shape the DHCP assertion proves for leases,
// on a service that answers over UDP: the resolver returns no packet at all,
// which is what a client measures as a timeout. All three halves are asserted,
// because silence on its own is also what an unreachable server produces.
func TestAuthoredDeviceFaultSilencesDNSOnTheWire(t *testing.T) {
	_, started := startFaultTimeline(t)

	conn, err := net.DialTimeout("udp", faultDNSAddr+":53", 5*time.Second)
	if err != nil {
		t.Fatalf("dialling the simulated resolver: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	healthy := resolveOverTheWire(t, conn, 0xfa11)
	if healthy.ResponseCode != layers.DNSResponseCodeNoErr || len(healthy.Answers) == 0 {
		t.Fatalf(
			"healthy lookup of %s = %v with %d answers, want NoError with the authored A record",
			faultDNSName, healthy.ResponseCode, len(healthy.Answers),
		)
	}

	// Well inside the phase, so neither a late start nor the retransmits can
	// put the window's far end past the phase's own reset.
	sleepUntil(started.Add(faultDNSTimeoutStart + 3*time.Second))
	if silenced, answered := tryResolveOverTheWire(
		t, conn, 0xfa12, faultSilenceWindow,
	); answered {
		t.Fatalf(
			"lookup of %s answered %v under the authored dns_timeout phase, want no reply at all",
			faultDNSName, silenced.ResponseCode,
		)
	}

	// The phase is a reset phase: the same resolver that went silent answers
	// again once it ends, which is what separates a fault from a dead server.
	sleepUntil(started.Add(faultDNSTimeoutEnd + time.Second))
	recovered := resolveOverTheWire(t, conn, 0xfa13)
	if recovered.ResponseCode != layers.DNSResponseCodeNoErr || len(recovered.Answers) == 0 {
		t.Fatalf(
			"lookup after the phase = %v with %d answers, want the authored A record back",
			recovered.ResponseCode, len(recovered.Answers),
		)
	}
	if got := recovered.Answers[0].IP.String(); got != faultDNSRecordIP {
		t.Fatalf("lookup after the phase answered %s, want the authored %s", got, faultDNSRecordIP)
	}
}

// Latency is the one device fault that suppresses nothing, so it is the one
// whose assertion is a measurement rather than a presence check. The bound is
// one-sided on purpose: a reply held back for at least the authored delay
// proves the fault applied, while any ceiling would be an assertion about the
// runner's load rather than about niac.
func TestAuthoredDeviceFaultDelaysEchoOnTheWire(t *testing.T) {
	_, started := startFaultTimeline(t)

	handle := openClient(t)
	src := clientMAC(t)
	// One packet source for the whole test: each one runs a goroutine that
	// keeps draining the shared handle, so a second would compete with this
	// one for the reply and buffer whatever it won out of reach.
	packets := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()

	healthy := echoRoundTrip(t, handle, packets, src, 0xfa21, 1)
	if healthy >= faultLatencyAuthored/2 {
		t.Fatalf(
			"healthy echo round trip = %s, already at half the authored delay %s; "+
				"the measurement cannot distinguish the fault",
			healthy, faultLatencyAuthored,
		)
	}

	sleepUntil(started.Add(faultLatencyStart + 2*time.Second))
	delayed := echoRoundTrip(t, handle, packets, src, 0xfa21, 2)
	if floor := faultLatencyAuthored - faultLatencyTolerance; delayed < floor {
		t.Fatalf(
			"echo round trip under the authored %s latency phase = %s, want at least %s "+
				"(healthy was %s)",
			faultLatencyAuthored, delayed, floor, healthy,
		)
	}
}

// echoRoundTrip pings the slow device and returns how long its reply took.
//
// The request is addressed to the authored MAC directly rather than resolved
// by the kernel: an ARP exchange folded into the first measurement would be
// read as latency the fault never caused.
func echoRoundTrip(
	t *testing.T,
	handle *pcap.Handle,
	packets chan gopacket.Packet,
	src net.HardwareAddr,
	id uint16,
	seq uint16,
) time.Duration {
	t.Helper()

	deadline := time.Now().Add(faultEchoDeadline)
	for attempt := uint16(0); time.Now().Before(deadline); attempt++ {
		// A new sequence number per attempt, so a reply to the attempt we
		// already gave up on is not timed against this attempt's clock.
		attemptSeq := seq*faultEchoSeqStride + attempt
		frame := echoFrame(t, src, id, attemptSeq)
		sentAt := time.Now()
		if err := handle.WritePacketData(frame); err != nil {
			t.Fatalf("writing echo request: %v", err)
		}
		if awaitEchoReply(packets, id, attemptSeq, faultEchoAttemptWindow) {
			return time.Since(sentAt)
		}
	}
	t.Fatalf("no echo reply from %s within %s", faultSlowAddr, faultEchoDeadline)

	return 0
}

// awaitEchoReply reports whether the reply to one attempt arrived inside its
// window. Every other ICMP frame on the wire -- a reply to an attempt already
// abandoned, or a request of our own -- is skipped by id, sequence and type.
func awaitEchoReply(
	packets chan gopacket.Packet, id uint16, seq uint16, window time.Duration,
) bool {
	deadline := time.After(window)
	for {
		select {
		case packet := <-packets:
			layer := packet.Layer(layers.LayerTypeICMPv4)
			if layer == nil {
				continue
			}
			reply, ok := layer.(*layers.ICMPv4)
			if !ok || reply.Id != id || reply.Seq != seq {
				continue
			}
			if reply.TypeCode.Type() == layers.ICMPv4TypeEchoReply {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

func echoFrame(t *testing.T, src net.HardwareAddr, id uint16, seq uint16) []byte {
	t.Helper()

	dstMAC, err := net.ParseMAC(faultSlowMAC)
	if err != nil {
		t.Fatalf("parsing the authored MAC %s: %v", faultSlowMAC, err)
	}
	eth := &layers.Ethernet{
		SrcMAC: src, DstMAC: dstMAC, EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    net.ParseIP(faultClientAddr),
		DstIP:    net.ParseIP(faultSlowAddr),
	}
	icmp := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
		Id:       id,
		Seq:      seq,
	}

	return serialize(t, eth, ip, icmp, gopacket.Payload([]byte("niac-wiretest")))
}
