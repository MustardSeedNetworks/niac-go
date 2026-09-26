//go:build linux && integration

package wiretest_test

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// P5-9: a tester on a pack's pool port hears spanning tree from the switch at
// the other end of its cable, every hello time, naming the site's core as root.
// The same switch polled over SNMP must describe the same tree, so the root
// and cost on the wire are compared with its dot1dStp scalars, not with
// constants: a BPDU and a MIB that each named the switch itself root would
// agree with each other but not with this.
const (
	oidDot1dStpDesignatedRoot = ".1.3.6.1.2.1.17.2.5.0"
	oidDot1dStpRootCost       = ".1.3.6.1.2.1.17.2.6.0"
	stpRootPrimaryPriority    = 24576
	stpHelloTime              = 2 * time.Second
)

func TestPoolSwitchSendsBPDUsNamingTheSiteCore(t *testing.T) {
	authored, _ := startPack(t, "hospital")
	access := deviceNamed(t, authored, authored.Attachments[0].At.Device)
	core := siteRoot(t, authored, access.Properties["site"])

	window := 4*stpHelloTime + time.Second
	bpdus, arrivals := captureBPDUs(t, access, window)
	if len(bpdus) < 3 {
		t.Fatalf("captured %d BPDUs from %s in %v, want one every %v", len(bpdus), access.Name, window, stpHelloTime)
	}
	for i := 1; i < len(arrivals); i++ {
		if gap := arrivals[i].Sub(arrivals[i-1]); gap < stpHelloTime/2 || gap > stpHelloTime*3/2 {
			t.Errorf("BPDU %d arrived %v after the previous one, want about %v", i, gap, stpHelloTime)
		}
	}

	first := bpdus[0]
	if !bytes.Equal(first.RouteID.HwAddr, core.MACAddress) || first.RouteID.Priority != stpRootPrimaryPriority {
		t.Errorf("BPDU root = %d/%s, want the site core %s at %d/%s", first.RouteID.Priority,
			first.RouteID.HwAddr, core.Name, stpRootPrimaryPriority, core.MACAddress)
	}
	if !bytes.Equal(first.BridgeID.HwAddr, access.MACAddress) || first.Cost == 0 {
		t.Errorf("BPDU bridge = %s cost %d, want %s with a non-zero cost to the root",
			first.BridgeID.HwAddr, first.Cost, access.MACAddress)
	}

	polled, err := dialDevice(t, authored, access.Name).Get([]string{oidDot1dStpDesignatedRoot, oidDot1dStpRootCost})
	if err != nil || len(polled.Variables) != 2 {
		t.Fatalf("GET dot1dStp root on %s: %v", access.Name, err)
	}
	wireRoot := binary.BigEndian.AppendUint16(nil, first.RouteID.Priority|first.RouteID.SysID)
	wireRoot = append(wireRoot, first.RouteID.HwAddr...)
	mibRoot, _ := polled.Variables[0].Value.([]byte)
	mibCost, _ := polled.Variables[1].Value.(int)
	if !bytes.Equal(mibRoot, wireRoot) || uint32(mibCost) != first.Cost {
		t.Errorf("dot1dStp on %s says root %x cost %d; its BPDUs say root %x cost %d",
			access.Name, mibRoot, mibCost, wireRoot, first.Cost)
	}
	t.Logf("%d BPDUs from %s: root %s (%s) cost %d; dot1dStpDesignatedRoot %x, dot1dStpRootCost %d",
		len(bpdus), access.Name, core.Name, first.RouteID.HwAddr, first.Cost, mibRoot, mibCost)
}

// captureBPDUs collects the BPDUs the tester hears for window, failing on any
// sent by a device other than the switch at its cable.
func captureBPDUs(t *testing.T, access *config.Device, window time.Duration) ([]*layers.STP, []time.Time) {
	t.Helper()
	handle := openClient(t)
	packets := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	var bpdus []*layers.STP
	var arrivals []time.Time
	deadline := time.After(window)
	for {
		select {
		case packet := <-packets:
			bpdu, ok := packet.Layer(layers.LayerTypeSTP).(*layers.STP)
			if !ok {
				continue
			}
			ethernet, _ := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
			if ethernet == nil || !bytes.Equal(ethernet.SrcMAC, access.MACAddress) {
				t.Errorf(
					"BPDU from %v; only %s (%s) is at the tester's cable",
					ethernet,
					access.Name,
					access.MACAddress,
				)
				continue
			}
			bpdus = append(bpdus, bpdu)
			arrivals = append(arrivals, packet.Metadata().Timestamp)
		case <-deadline:
			return bpdus, arrivals
		}
	}
}

func siteRoot(t *testing.T, cfg *config.Config, site string) *config.Device {
	t.Helper()
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		if device.Properties["site"] == site && device.STPConfig != nil &&
			device.STPConfig.BridgePriority == stpRootPrimaryPriority {
			return device
		}
	}
	t.Fatalf("site %q has no switch at the root priority", site)
	return nil
}
