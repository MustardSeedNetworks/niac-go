package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// TestRepliesCarryTheDeviceTTL pins #2731: a tester that fingerprints the OS
// from the IP TTL must see the same host over ICMP, UDP and TCP. The ICMP
// config authors 64 and the OS fingerprint 128, so every reply proves the
// fingerprint wins.
func TestRepliesCarryTheDeviceTTL(t *testing.T) {
	device := &config.Device{
		Name:                "DEN-NOC-1302",
		MACAddress:          net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x13, 0x02},
		ICMPConfig:          &config.ICMPConfig{Enabled: true, TTL: 64},
		OSFingerprintConfig: &config.OSFingerprintConfig{TTL: 128},
	}
	tester := net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0xc9}
	deviceIP, testerIP := net.ParseIP("10.102.210.40").To4(), net.ParseIP("10.101.210.201").To4()

	tests := []struct {
		name string
		send func(*Stack) error
	}{
		{"icmp echo reply", func(s *Stack) error {
			return NewICMPHandler(s).sendEchoReply(device.MACAddress, tester, deviceIP, testerIP, 1, 1, nil, device, 0)
		}},
		{"icmp unreachable", func(s *Stack) error {
			return NewICMPHandler(s).SendICMPUnreachable(deviceIP, testerIP, device.MACAddress, tester, 3,
				make([]byte, 28), 0, device)
		}},
		{"udp (netbios, snmp)", func(s *Stack) error {
			return NewUDPHandler(s).SendUDP(device, deviceIP, testerIP, 137, 137, []byte("nbstat"),
				device.MACAddress, tester, 0)
		}},
		{"dns", func(s *Stack) error {
			return NewDNSHandler(s).SendDNSResponse(device, &layers.DNS{QR: true}, deviceIP, testerIP,
				device.MACAddress, tester, 53000, 0)
		}},
		{"tcp rst", func(s *Stack) error {
			request := &layers.IPv4{SrcIP: testerIP, DstIP: deviceIP}
			NewTCPHandler(s).sendRSTPacket(device,
				replyEthernetIdentity{source: device.MACAddress, destination: tester},
				request, &layers.TCP{SrcPort: 40000, DstPort: 23, Seq: 1}, 0)
			return nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := newTestStackInternal(t)
			if err := tt.send(stack); err != nil {
				t.Fatalf("send: %v", err)
			}
			pkt := drainDHCP(t, stack)
			ip, _ := gopacket.NewPacket(pkt.Buffer, layers.LayerTypeEthernet, gopacket.Default).
				Layer(layers.LayerTypeIPv4).(*layers.IPv4)
			if ip == nil || ip.TTL != 128 {
				t.Fatalf("reply IPv4 = %v, want TTL 128", ip)
			}
		})
	}
}
