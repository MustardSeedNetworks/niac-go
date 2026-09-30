//go:build linux && integration

package wiretest_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// AP-6: a re-pin moves one tester on the running session. The switch learns
// it on the new port, its LLDP names that port, and every other client keeps
// its port and lease. Before AP-6 the move restarted the session, which put
// every client back through placement and DHCP from nothing.
func TestRepinMovesOneClientOnTheRunningSession(t *testing.T) {
	authored, attachment, d := startPackDaemon(t, "hospital")
	pool := authored.Attachments[0].At
	if len(pool.Ports) < 3 {
		t.Fatalf("hospital pool has %d ports; the move needs a free third", len(pool.Ports))
	}
	access := dialDevice(t, authored, pool.Device)
	vlan := portVLAN(t, deviceNamed(t, authored, pool.Device), pool.Ports[0])
	handle := openClient(t)

	// The test end places itself on the first port by talking; the second
	// client takes the next. Each then leases, in that order.
	if _, err := access.Get([]string{oidSysName}); err != nil {
		t.Fatalf("GET sysName on %s: %v", pool.Device, err)
	}
	moving, staying := clientMAC(t), secondClient()
	announce(t, staying, attachment)
	awaitFDBPort(t, access, vlan, staying)
	leases := map[string]string{}
	for index, mac := range []net.HardwareAddr{moving, staying} {
		leases[mac.String()] = lease(t, handle, mac, 0x6e710000+uint32(index)).String()
	}
	startedAt := d.GetStatus().StartedAt

	// An unsafe pin is refused and changes nothing.
	outside := api.AttachmentPin{MAC: staying.String(), Device: pool.Device, Interface: "GigabitEthernet9/9/9"}
	var unsafe *fabric.UnsafeTopologyError
	if err := d.PinAttachmentClient(packSession("hospital"), outside); !errors.As(err, &unsafe) {
		t.Fatalf("pin outside the pool = %v, want an unsafe-topology refusal", err)
	}

	pin := api.AttachmentPin{MAC: moving.String(), Device: pool.Device, Interface: pool.Ports[2]}
	if err := d.PinAttachmentClient(packSession("hospital"), pin); err != nil {
		t.Fatalf("PinAttachmentClient(%s -> %s): %v", moving, pool.Ports[2], err)
	}

	if got := d.GetStatus().StartedAt; !got.Equal(startedAt) {
		t.Errorf("session started at %s, then %s: the move restarted it", startedAt, got)
	}
	for mac, want := range map[string]string{moving.String(): pool.Ports[2], staying.String(): pool.Ports[1]} {
		hw, _ := net.ParseMAC(mac)
		if got := awaitFDBPort(t, access, vlan, hw); got != want {
			t.Errorf("%s dot1qTpFdbPort on %s resolves to %q, want %q", mac, pool.Device, got, want)
		}
	}
	// The dot1d table is keyed by the MAC alone, so one row: the old port no
	// longer reports the moved client.
	if row := fdbRow(t, access, oidDot1dTpFdbPort, moving); row == "" {
		t.Errorf("%s dropped %s from dot1dTpFdbTable altogether", pool.Device, moving)
	}

	// The moved client was the first placed, so the one advertisement on the
	// shared wire follows it.
	want := advertisedName(deviceNamed(t, authored, pool.Device)) + " " + pool.Ports[2]
	if neighbours := lldpNeighbours(t); len(neighbours) != 1 || neighbours[want] == 0 {
		t.Errorf("LLDP neighbours after the move = %v, want exactly one: %q", neighbours, want)
	}

	// A restart would have wiped every lease, and the second client, asking
	// first this time, would be handed the first address. Kept leases answer
	// each client with its own.
	for index, mac := range []net.HardwareAddr{staying, moving} {
		if got := lease(t, handle, mac, 0x6e720000+uint32(index)).String(); got != leases[mac.String()] {
			t.Errorf("%s leased %s after the move, want its lease %s", mac, got, leases[mac.String()])
		}
	}
	t.Logf("%s moved %s -> %s; %s kept %s; leases %v", pool.Device, pool.Ports[0], pool.Ports[2],
		staying, pool.Ports[1], leases)
}

// A re-pin in one trunk session leaves a sibling on another tag of the same
// NIC alone: the sibling is not restarted and keeps answering on its tag.
func TestRepinLeavesASiblingTrunkSessionUndisturbed(t *testing.T) {
	requireWire(t)
	const moved, sibling = uint16(201), uint16(202)
	generated, attachmentName := generatePack(t, "hospital")
	authored, err := config.LoadYAMLBytes(generated)
	if err != nil {
		t.Fatalf("loading the generated hospital YAML: %v", err)
	}
	attachment := resolveAttachment(t, authored, attachmentName)
	pool := authored.Attachments[0].At

	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	d, err := daemon.NewDaemon(daemon.Config{
		StoragePath: "disabled",
		AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
			Interface: simIface, Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{moved, sibling},
		}},
	})
	if err != nil {
		t.Fatalf("daemon.NewDaemon: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = d.Shutdown(ctx)
	})
	for _, tag := range []uint16{moved, sibling} {
		if startErr := d.StartSimulation(api.SimulationRequest{
			SessionID: trunkSession(tag), Interface: simIface, Attachment: attachmentName,
			AttachmentMode: fabric.ModeTrunk, AccessVLAN: tag, ConfigData: string(generated),
		}); startErr != nil {
			t.Fatalf("StartSimulation(tag %d): %v", tag, startErr)
		}
	}
	client := secondClient()
	gatewayMAC := attachment.gatewayDevice.MACAddress
	// The client's ARP on the moved tag is what places it in that session.
	for _, tag := range []uint16{moved, sibling} {
		if got := taggedARP(t, tag, client, attachment); !bytes.Equal(got, gatewayMAC) {
			t.Fatalf("tag %d gateway answered ARP with %s, want %s", tag, got, gatewayMAC)
		}
	}
	movedStart := sessionStartedAt(t, d, trunkSession(moved))
	siblingStart := sessionStartedAt(t, d, trunkSession(sibling))

	pin := api.AttachmentPin{MAC: client.String(), Device: pool.Device, Interface: pool.Ports[2]}
	if err = d.PinAttachmentClient(trunkSession(moved), pin); err != nil {
		t.Fatalf("PinAttachmentClient on tag %d: %v", moved, err)
	}

	if got := sessionStartedAt(t, d, trunkSession(moved)); !got.Equal(movedStart) {
		t.Errorf("tag %d started at %s, then %s: the move restarted it", moved, movedStart, got)
	}
	if got := sessionStartedAt(t, d, trunkSession(sibling)); !got.Equal(siblingStart) {
		t.Errorf("sibling on tag %d started at %s, then %s: the move restarted it", sibling, siblingStart, got)
	}
	for _, tag := range []uint16{moved, sibling} {
		if got := taggedARP(t, tag, client, attachment); !bytes.Equal(got, gatewayMAC) {
			t.Errorf("after the move, tag %d gateway answered ARP with %s, want %s", tag, got, gatewayMAC)
		}
	}
	t.Logf("tag %d re-pinned %s to %s; tag %d untouched since %s",
		moved, pin.MAC, pin.Interface, sibling, siblingStart.Format(time.RFC3339Nano))
}

func trunkSession(tag uint16) string { return fmt.Sprintf("wiretest-trunk-%d", tag) }

// generatePack returns a shipped pack's YAML and the attachment it names.
func generatePack(t *testing.T, id string) ([]byte, string) {
	t.Helper()
	for _, pack := range scenario.Packs() {
		if pack.ID != id {
			continue
		}
		result, err := scenario.Generate(pack.Request)
		if err != nil {
			t.Fatalf("scenario.Generate(%s): %v", id, err)
		}
		return result.YAML, pack.Request.AttachmentName
	}
	t.Fatalf("no pack with id %q", id)
	return nil, ""
}

func sessionStartedAt(t *testing.T, d *daemon.Daemon, session string) time.Time {
	t.Helper()
	if err := d.SelectSimulation(session); err != nil {
		t.Fatalf("SelectSimulation(%s): %v", session, err)
	}
	return d.GetStatus().StartedAt
}

// lease runs one DHCP DISCOVER/REQUEST for mac and returns the address acked.
func lease(t *testing.T, handle *pcap.Handle, mac net.HardwareAddr, xid uint32) net.IP {
	t.Helper()
	offer := dhcpExchange(t, handle, mac, xid, layers.DHCPMsgTypeDiscover, nil)
	offered := append(net.IP(nil), offer.YourClientIP...)
	ack := dhcpExchange(t, handle, mac, xid, layers.DHCPMsgTypeRequest, offered)
	return append(net.IP(nil), ack.YourClientIP...)
}

// lldpNeighbours listens through two LLDP intervals and counts each
// system-name and port pair advertised on the wire. It reads its own handle:
// a packet source keeps reading after its caller stops, so two on one handle
// split the frames between them.
func lldpNeighbours(t *testing.T) map[string]int {
	t.Helper()
	handle := openClient(t)
	packets := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	neighbours := map[string]int{}
	deadline := time.After(35 * time.Second)
	for {
		select {
		case packet := <-packets:
			lldp, isLLDP := packet.Layer(layers.LayerTypeLinkLayerDiscovery).(*layers.LinkLayerDiscovery)
			info, hasInfo := packet.Layer(layers.LayerTypeLinkLayerDiscoveryInfo).(*layers.LinkLayerDiscoveryInfo)
			if isLLDP && hasInfo {
				neighbours[info.SysName+" "+string(lldp.PortID.ID)]++
			}
		case <-deadline:
			return neighbours
		}
	}
}

// taggedARP asks for the attachment gateway from src on one trunk tag and
// returns the hardware address that answered on that tag, reading its own
// handle for the reason lldpNeighbours does.
func taggedARP(t *testing.T, tag uint16, src net.HardwareAddr, attachment packAttachment) net.HardwareAddr {
	t.Helper()
	handle := openClient(t)
	sender := attachment.client.Addr().Next().AsSlice()
	frame := serialize(t,
		&layers.Ethernet{SrcMAC: src, DstMAC: layers.EthernetBroadcast, EthernetType: layers.EthernetTypeDot1Q},
		&layers.Dot1Q{VLANIdentifier: tag, Type: layers.EthernetTypeARP},
		&layers.ARP{
			AddrType: layers.LinkTypeEthernet, Protocol: layers.EthernetTypeIPv4,
			HwAddressSize: 6, ProtAddressSize: 4, Operation: layers.ARPRequest,
			SourceHwAddress: src, SourceProtAddress: sender,
			DstHwAddress: make([]byte, 6), DstProtAddress: attachment.gateway.AsSlice(),
		},
	)
	packets := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	deadline := time.After(5 * time.Second)
	resend := time.NewTicker(time.Second)
	defer resend.Stop()
	if err := handle.WritePacketData(frame); err != nil {
		t.Fatalf("sending tagged ARP on %d: %v", tag, err)
	}
	for {
		select {
		case packet := <-packets:
			dot1q, tagged := packet.Layer(layers.LayerTypeDot1Q).(*layers.Dot1Q)
			arp, isARP := packet.Layer(layers.LayerTypeARP).(*layers.ARP)
			if tagged && isARP && dot1q.VLANIdentifier == tag && arp.Operation == layers.ARPReply &&
				bytes.Equal(arp.DstHwAddress, src) {
				return net.HardwareAddr(arp.SourceHwAddress)
			}
		case <-resend.C:
			if err := handle.WritePacketData(frame); err != nil {
				t.Fatalf("sending tagged ARP on %d: %v", tag, err)
			}
		case <-deadline:
			t.Fatalf("no ARP reply for %s on tag %d", attachment.gateway, tag)
			return nil
		}
	}
}
