//go:build linux && integration

package wiretest_test

import (
	"bytes"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// P5-9: a passive listener that never solicits still learns the router, from
// the advertisements it sends every authored period. The client end is a real
// Linux interface, so the kernel's own acceptance is asserted too: it installs
// a default route only for an advertisement it considers valid (link-local
// source, hop limit 255, correct checksum), which a capture alone does not
// prove.
const raPeriod = 3 * time.Second

func TestRouterSendsUnsolicitedAdvertisementsAtTheAuthoredPeriod(t *testing.T) {
	authored := startAuthoredFile(t, "ra-wire.yaml", "wiretest-ra")
	router := deviceNamed(t, authored, "wire-ra-gw-01")
	// The route outlives the session; later tests share the namespace.
	t.Cleanup(func() { _ = exec.Command("ip", "-6", "route", "flush", "dev", testIface, "proto", "ra").Run() })

	window := 4*raPeriod + time.Second
	heard := captureRouterAdvertisements(t, router, window)
	arrivals, source := heard.arrivals, heard.source
	if len(arrivals) < 3 {
		t.Fatalf("captured %d unsolicited advertisements in %v, want one every %v",
			len(arrivals), window, raPeriod)
	}
	for i := 1; i < len(arrivals); i++ {
		if gap := arrivals[i].Sub(arrivals[i-1]); gap < raPeriod*3/4 || gap > raPeriod*5/4 {
			t.Errorf("advertisement %d arrived %v after the previous one, want about %v", i, gap, raPeriod)
		}
	}

	routes := ip(t, "-6", "route", "show", "default", "dev", testIface)
	if !strings.Contains(routes, "via "+source.String()) || !strings.Contains(routes, "proto ra") {
		t.Errorf("the client's kernel did not accept the advertisement: default routes on %s:\n%s", testIface, routes)
	}
	t.Logf("%d unsolicited advertisements from %s (%s), gaps %v, plus %d solicited; kernel default route: %s",
		len(arrivals), router.Name, source, gaps(arrivals), heard.solicited, strings.TrimSpace(routes))
}

type raCapture struct {
	arrivals  []time.Time // of the all-nodes advertisements
	source    net.IP
	solicited int
}

// captureRouterAdvertisements collects, for window, the all-nodes
// advertisements the client hears and counts the solicited ones, failing on
// any from a device other than router or from a source a host would discard.
func captureRouterAdvertisements(
	t *testing.T,
	router *config.Device,
	window time.Duration,
) raCapture {
	t.Helper()
	var heard raCapture
	handle := openClient(t)
	packets := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	deadline := time.After(window)
	for {
		select {
		case packet := <-packets:
			ipv6, _ := packet.Layer(layers.LayerTypeIPv6).(*layers.IPv6)
			ethernet, _ := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
			if packet.Layer(layers.LayerTypeICMPv6RouterAdvertisement) == nil || ipv6 == nil || ethernet == nil {
				continue
			}
			if !bytes.Equal(ethernet.SrcMAC, router.MACAddress) {
				t.Errorf("advertisement from %s; only %s (%s) authors one",
					ethernet.SrcMAC, router.Name, router.MACAddress)
				continue
			}
			if !ipv6.SrcIP.IsLinkLocalUnicast() || ipv6.HopLimit != 255 {
				t.Errorf(
					"advertisement from %s at hop limit %d, want a link-local source at 255",
					ipv6.SrcIP,
					ipv6.HopLimit,
				)
			}
			// The kernel solicits when the veth comes up; those answers are
			// unicast, and only the all-nodes ones are on the period.
			if !ipv6.DstIP.Equal(net.ParseIP("ff02::1")) {
				heard.solicited++
				continue
			}
			heard.source = ipv6.SrcIP
			heard.arrivals = append(heard.arrivals, packet.Metadata().Timestamp)
		case <-deadline:
			return heard
		}
	}
}

func gaps(arrivals []time.Time) []time.Duration {
	out := make([]time.Duration, 0, len(arrivals))
	for i := 1; i < len(arrivals); i++ {
		out = append(out, arrivals[i].Sub(arrivals[i-1]).Round(time.Millisecond))
	}
	return out
}
