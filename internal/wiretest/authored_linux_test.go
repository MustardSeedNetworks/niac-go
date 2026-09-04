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

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// The network the authoring surfaces produce, replayed on the wire.
//
// P1b-6 asks that the wire suite run against a wizard-built session.
// testdata/authored-network.yaml is that network, and the Playwright suite
// authors the same one three ways -- uploaded, through the device editor, and
// through the wizard from empty -- asserting each route produces its devices,
// addresses, link and SNMP.
//
// The wizard's own output is not byte-identical to this file: its composer
// assigns MACs from the device name and adds profile metadata. So this is the
// same *network*, not the same bytes, and the honest claim is that the shape
// the authoring surfaces produce is reachable on the wire -- a failure no UI
// test can see, because a UI test stops at the saved config.
const (
	authoredRouterName = "e2e-rtr-01"
	authoredRouterIP   = "10.77.0.1"
	authoredClientIP   = "10.77.0.50" // outside anything the file authors
)

func startAuthoredNetwork(t *testing.T) *config.Config {
	t.Helper()
	requireWire(t)

	yamlBytes, err := os.ReadFile(filepath.Join("testdata", "authored-network.yaml"))
	if err != nil {
		t.Fatalf("reading the authored network: %v", err)
	}
	authored, err := config.LoadYAMLBytes(yamlBytes)
	if err != nil {
		t.Fatalf("loading the authored network: %v", err)
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
	if startErr := d.StartSimulation(api.SimulationRequest{
		SessionID:      "wiretest-authored",
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

	return authored
}

func authoredDevice(t *testing.T, cfg *config.Config, name string) *config.Device {
	t.Helper()
	for index := range cfg.Devices {
		if cfg.Devices[index].Name == name {
			return &cfg.Devices[index]
		}
	}
	t.Fatalf("no device named %q in the authored network", name)

	return nil
}

// ARP is the cheapest proof the authored device is actually on the wire, and
// its answer carries an authored field -- the MAC the file states -- so a
// device that replied with the wrong identity fails rather than passing a
// count.
func TestAuthoredNetworkAnswersARPWithItsAuthoredMAC(t *testing.T) {
	authored := startAuthoredNetwork(t)
	router := authoredDevice(t, authored, authoredRouterName)

	handle := openClient(t)
	source := clientMAC(t)
	target := net.ParseIP(authoredRouterIP).To4()

	requestARP := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   source,
		SourceProtAddress: net.ParseIP(authoredClientIP).To4(),
		DstHwAddress:      net.HardwareAddr{0, 0, 0, 0, 0, 0},
		DstProtAddress:    target,
	}
	broadcast := net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	// Untagged, because the binding is an access port: the daemon puts the
	// access VLAN on internally and the wire carries plain frames. A tagged
	// request here -- which is what the hospital suite sends, because it binds
	// a routed transit network instead -- gets no answer at all, and reads as
	// "the device is unreachable" rather than "the frame was addressed wrong".
	frame := serialize(t,
		&layers.Ethernet{SrcMAC: source, DstMAC: broadcast, EthernetType: layers.EthernetTypeARP},
		requestARP,
	)
	if err := handle.WritePacketData(frame); err != nil {
		t.Fatalf("writing ARP request: %v", err)
	}

	want := router.MACAddress.String()
	deadline := time.Now().Add(10 * time.Second)
	packets := gopacket.NewPacketSource(handle, handle.LinkType())
	for time.Now().Before(deadline) {
		packet, err := packets.NextPacket()
		if err != nil {
			continue
		}
		arpLayer := packet.Layer(layers.LayerTypeARP)
		if arpLayer == nil {
			continue
		}
		reply, ok := arpLayer.(*layers.ARP)
		if !ok || reply.Operation != layers.ARPReply {
			continue
		}
		if !net.IP(reply.SourceProtAddress).Equal(net.ParseIP(authoredRouterIP)) {
			continue
		}
		if got := net.HardwareAddr(reply.SourceHwAddress).String(); got != want {
			t.Fatalf("ARP reply from %s carried MAC %s, want the authored %s",
				authoredRouterIP, got, want)
		}

		return
	}
	t.Fatalf("no ARP reply from the authored device at %s within the deadline", authoredRouterIP)
}
