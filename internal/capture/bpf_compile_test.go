package capture

import (
	"strconv"
	"testing"

	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

func TestBPFCompiledProgramsRemainIndependent(t *testing.T) {
	vm, err := compileBPFVM(layers.LinkTypeEthernet, "udp port 53")
	if err != nil {
		t.Fatal(err)
	}
	for range 32 {
		assertBPFCompileCycle(t)
	}
	for _, port := range []layers.UDPPort{53, 80} {
		matched, runErr := vm.Run(udpPacketBytes(t, port))
		if runErr != nil || (matched > 0) != (port == 53) {
			t.Fatalf("port %d: matched %d, error %v", port, matched, runErr)
		}
	}
}

func assertBPFCompileCycle(t *testing.T) {
	t.Helper()
	if err := ValidateBPFExpr("arp"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBPFExpr("udp port &&"); err == nil {
		t.Fatal("malformed expression accepted")
	}
}

func TestBPFCompilerPreservesCaptureLinkType(t *testing.T) {
	for _, linkType := range []layers.LinkType{layers.LinkTypeEthernet, layers.LinkTypeLinuxSLL} {
		t.Run(linkType.String(), func(t *testing.T) {
			vm, err := compileBPFVM(linkType, "arp")
			if err != nil {
				t.Fatal(err)
			}
			for _, protocol := range []byte{0x06, 0x00} {
				matched, runErr := vm.Run(bpfLinkFrame(linkType, protocol))
				if runErr != nil || (matched > 0) != (protocol == 0x06) {
					t.Fatalf("protocol 0x08%02x: matched %d, error %v", protocol, matched, runErr)
				}
			}
		})
	}
}

func bpfLinkFrame(linkType layers.LinkType, protocol byte) []byte {
	frame := arpFrame()
	if linkType == layers.LinkTypeLinuxSLL {
		frame = append([]byte{0, 0}, frame...)
	}
	frame[12], frame[13] = 0x08, protocol
	if linkType == layers.LinkTypeLinuxSLL {
		frame[14], frame[15] = 0x08, protocol
	}
	return frame
}

func TestBPFCompilerRejectsUnknownLinkType(t *testing.T) {
	if vm, err := compileBPFVM(layers.LinkType(0xffff), "arp"); err == nil || vm != nil {
		t.Fatalf("unknown link type: VM %v, error %v", vm, err)
	}
}

func TestBPFCompilerConcurrentCompiles(t *testing.T) {
	t.Parallel()
	t.Log(pcap.Version())
	for i := range 32 {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			assertBPFCompileCycle(t)
			match, err := NewEthernetMatcher("arp")
			if err != nil {
				t.Fatal(err)
			}
			if !match(arpFrame()) {
				t.Fatal("compiled ARP filter rejected its frame")
			}
		})
	}
}

func TestBPFCompilerRejectsNULInExpressions(t *testing.T) {
	for i, expr := range []string{"\x00arp", "arp\x00", "arp\x00 or ip", "udp port 53\x00 &&"} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assertRejectedBPFExpression(t, expr)
		})
	}
}

func assertRejectedBPFExpression(t *testing.T, expr string) {
	t.Helper()
	if err := ValidateBPFExpr(expr); err == nil {
		t.Errorf("validation accepted %q", expr)
	}
	if _, err := NewEthernetMatcher(expr); err == nil {
		t.Errorf("matcher accepted %q", expr)
	}
	if _, err := compileBPFVM(layers.LinkTypeEthernet, expr); err == nil {
		t.Errorf("VM accepted %q", expr)
	}
}
