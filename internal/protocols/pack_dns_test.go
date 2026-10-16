package protocols_test

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// A tester plugged into any pack VLAN takes the resolver DHCP hands it and runs
// its default DNS test against an internet name. The site resolver holds only
// the site's own names, so it must forward the miss up the chain to the lab
// edge, which answers with the simulated internet (niac-go: CT304 CyberScope
// returned "name not found" for www.google.com on every pack VLAN).
func TestShippedPackSiteResolversAnswerInternetNames(t *testing.T) {
	for _, pack := range scenario.Packs() {
		t.Run(pack.ID, func(t *testing.T) {
			shipped := loadShippedPack(t, pack)
			resolvers := dhcpAdvertisedResolvers(shipped.cfg)
			if len(resolvers) == 0 {
				t.Fatal("no DHCP server advertises a resolver")
			}
			for address, server := range resolvers {
				answer := queryPackResolver(t, shipped, server, address, "www.google.com")
				if answer == nil {
					t.Errorf("%s (%s) did not answer", server.Name, address)
					continue
				}
				if answer.ResponseCode != layers.DNSResponseCodeNoErr || len(answer.Answers) == 0 ||
					!answer.Answers[0].IP.Equal(net.ParseIP("8.8.8.8")) {
					t.Errorf("%s (%s) www.google.com = %v %v, want 8.8.8.8",
						server.Name, address, answer.ResponseCode, answer.Answers)
				}
			}
		})
	}
}

// dhcpAdvertisedResolvers maps each DHCP option 6 address to the device that
// owns it, which is the server a tester's lookups reach.
func dhcpAdvertisedResolvers(cfg *config.Config) map[string]*config.Device {
	resolvers := map[string]*config.Device{}
	for i := range cfg.Devices {
		dhcp := cfg.Devices[i].DHCPConfig
		if dhcp == nil {
			continue
		}
		for _, address := range dhcp.DomainNameServer {
			for j := range cfg.Devices {
				for _, owned := range cfg.Devices[j].IPAddresses {
					if owned.Equal(address) {
						resolvers[address.String()] = &cfg.Devices[j]
					}
				}
			}
		}
	}
	return resolvers
}

func queryPackResolver(
	t *testing.T, pack shippedPack, server *config.Device, address, name string,
) *layers.DNS {
	t.Helper()
	eth := &layers.Ethernet{
		SrcMAC: net.HardwareAddr{2, 0, 0, 0, 0, 0x99}, DstMAC: server.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("192.0.2.99"), DstIP: net.ParseIP(address),
	}
	udp := &layers.UDP{SrcPort: 45000, DstPort: 53}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatal(err)
	}
	query := &layers.DNS{ID: 7, RD: true, Questions: []layers.DNSQuestion{
		{Name: []byte(name), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
	}}
	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	if err := gopacket.SerializeLayers(buffer, options, eth, ip, udp, query); err != nil {
		t.Fatal(err)
	}
	packet := &protocols.Packet{Buffer: buffer.Bytes()}
	pack.stack.GetDNSHandler().HandleQuery(packet, ip, udp, []*config.Device{server})
	sent := pack.stack.SentPacketForTest()
	if sent == nil {
		return nil
	}
	decoded := gopacket.NewPacket(sent.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	dns, _ := decoded.Layer(layers.LayerTypeDNS).(*layers.DNS)
	return dns
}
