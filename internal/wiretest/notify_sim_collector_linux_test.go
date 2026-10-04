//go:build linux && integration

package wiretest_test

import (
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// The hospital pack's four authored utilization faults sit on management-VLAN
// switches and are reported to MED-NMS01, a simulated collector. Once those
// switches had the default gateway a real one carries, each notification went
// out on the tester's wire, unicast to the core's MAC (#2410). The management
// network is simulated end to end, so nothing sourced from it may appear here;
// the collector records the four instead (TestHospitalCollectorReceivesTheAuthoredFaults).
func TestPackNotificationsToASimulatedCollectorStayOffTheWire(t *testing.T) {
	handle := openClient(t)
	packets := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	authored, _ := startPack(t, "hospital")
	management := managementAddresses(authored)
	if len(management) == 0 {
		t.Fatal("the hospital pack authors no management addresses")
	}

	var heard, leaked int
	deadline := time.After(15 * time.Second)
	for {
		select {
		case packet := <-packets:
			heard++
			ipv4, _ := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
			if ipv4 == nil {
				continue
			}
			source, _ := netip.AddrFromSlice(ipv4.SrcIP)
			if device, found := management[source]; found {
				leaked++
				ethernet, _ := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
				t.Errorf("%s (%s) reached the wire: %v -> %v, %s",
					device, source, ethernet.SrcMAC, ethernet.DstMAC, ipv4.NextLayerType())
			}
		case <-deadline:
			// The pack's own discovery frames prove the capture was listening.
			if heard == 0 {
				t.Fatal("heard nothing from the pack in 15 s; the capture proves nothing")
			}
			t.Logf("%d frames heard, %d from management addresses", heard, leaked)
			return
		}
	}
}

// managementAddresses maps every address on a site management network to the
// device that owns it.
func managementAddresses(cfg *config.Config) map[netip.Addr]string {
	addresses := map[netip.Addr]string{}
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		for _, iface := range device.Interfaces {
			prefix, err := netip.ParsePrefix(iface.Address)
			if err == nil && strings.HasSuffix(iface.Network, "-mgmt") {
				addresses[prefix.Addr()] = device.Name
			}
		}
	}
	return addresses
}
