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

	// The resolver whose phase arms dns_nxdomain for the whole session, and
	// a name it authors an A record for -- so the NXDOMAIN is the fault
	// rewriting a successful answer, not an empty zone answering normally.
	faultDNSAddr     = "10.254.200.2"
	faultDNSName     = "host.fault.example"
	faultDNSRecordIP = "10.254.200.90"

	// testdata/fault-timeline.yaml: the dns-outage phase's start_offset_ms.
	faultDNSPhaseStart = 10 * time.Second

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
	if remaining := time.Until(
		started.Add(faultDNSPhaseStart + 2*time.Second),
	); remaining > 0 {
		time.Sleep(remaining)
	}

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
// returns its response, retransmitting the way a resolver does: the first
// datagram can be lost to ARP resolution on a fresh veth.
func resolveOverTheWire(t *testing.T, conn net.Conn, id uint16) layers.DNS {
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

	deadline := time.Now().Add(20 * time.Second)
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
		if response.ID != id {
			continue
		}

		return response
	}
	t.Fatalf("no DNS response from %s within the deadline", faultDNSAddr)

	return layers.DNS{}
}
