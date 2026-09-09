package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestYAMLDNSQueryUsesAuthoredServer(t *testing.T) {
	cfg, err := config.LoadYAMLBytes([]byte(`devices:
  - name: workstation
    mac: "02:00:00:00:00:01"
    ips: [192.0.2.1]
  - name: dns-server
    mac: "02:00:00:00:00:53"
    ips: [192.0.2.53]
    dns:
      forward_records:
        - name: host.example
          ip: 192.0.2.100
          ttl: 300
`))
	if err != nil {
		t.Fatal(err)
	}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	server := &cfg.Devices[1]
	eth := &layers.Ethernet{
		SrcMAC: net.HardwareAddr{2, 0, 0, 0, 0, 0x99}, DstMAC: server.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("192.0.2.99"), DstIP: server.IPAddresses[0],
	}
	udp := &layers.UDP{SrcPort: 45000, DstPort: 53}
	if err = udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatal(err)
	}
	query := &layers.DNS{ID: 42, RD: true, Questions: []layers.DNSQuestion{
		{Name: []byte("host.example"), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
	}}
	packet := &Packet{Buffer: serializeTestLayers(t, eth, ip, udp, query)}
	stack.dnsHandler.HandleQuery(packet, ip, udp, []*config.Device{&cfg.Devices[0], server})
	select {
	case response := <-stack.sendQueue:
		decoded := gopacket.NewPacket(response.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		ethernet, _ := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
		ipv4, _ := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		dns, _ := decoded.Layer(layers.LayerTypeDNS).(*layers.DNS)
		if ethernet == nil || ethernet.SrcMAC.String() != server.MACAddress.String() {
			t.Fatalf("wrong DNS responder Ethernet identity: %#v", ethernet)
		}
		if ipv4 == nil || !ipv4.SrcIP.Equal(server.IPAddresses[0]) {
			t.Fatalf("wrong DNS responder IP identity: %#v", ipv4)
		}
		if dns == nil || dns.ResponseCode != layers.DNSResponseCodeNoErr || len(dns.Answers) != 1 ||
			!dns.Answers[0].IP.Equal(net.ParseIP("192.0.2.100")) {
			t.Fatalf("wrong DNS answer: %#v", dns)
		}
	default:
		t.Fatal("authored DNS server did not respond")
	}
}
