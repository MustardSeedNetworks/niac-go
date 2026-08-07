package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// Apple hardware, printers and most IoT gear announce themselves over multicast
// DNS rather than NetBIOS. Without a responder they are unnamed on a discovery
// map even when they answer everything else, so a query for <host>.local has to
// come back with the device's address.
func TestMDNSAnswersHostnameQuery(t *testing.T) {
	const (
		hostLabel = "med-mri-b01-f01-03"
		deviceIP  = "10.51.210.23"
	)

	device := mdnsTestDevice(hostLabel, deviceIP, nil)
	answers := mdnsAnswers(device, []layers.DNSQuestion{{
		Name: []byte(hostLabel + ".local"), Type: layers.DNSTypeA, Class: layers.DNSClassIN,
	}})

	if len(answers) != 1 {
		t.Fatalf("got %d answers, want 1 — the device is unnamed to a browser", len(answers))
	}
	if got := string(answers[0].Name); got != hostLabel+".local" {
		t.Errorf("answer name = %q, want %q", got, hostLabel+".local")
	}
	if got := answers[0].IP.String(); got != deviceIP {
		t.Errorf("answer address = %s, want %s", got, deviceIP)
	}
	// The cache-flush bit marks the record authoritative so a browser replaces
	// any cached copy instead of accumulating stale entries.
	if answers[0].Class&mdnsCacheFlush == 0 {
		t.Error("answer is missing the cache-flush bit")
	}
}

// A browser first asks which service types exist, then browses one of them.
// Both steps have to answer or the device shows up with no services.
func TestMDNSAnswersServiceDiscovery(t *testing.T) {
	const (
		hostLabel   = "med-imaging-printer"
		deviceIP    = "10.51.210.40"
		serviceType = "_ipp._tcp"
	)

	device := mdnsTestDevice(hostLabel, deviceIP, []config.MDNSService{
		{Type: serviceType, Port: 631, TXT: []string{"rp=ipp/print"}},
	})

	enumeration := mdnsAnswers(device, []layers.DNSQuestion{{
		Name: []byte(serviceEnumerationName), Type: layers.DNSTypePTR, Class: layers.DNSClassIN,
	}})
	if len(enumeration) != 1 {
		t.Fatalf("service enumeration returned %d answers, want 1", len(enumeration))
	}
	if got := string(enumeration[0].PTR); got != serviceType+".local" {
		t.Errorf("enumerated service = %q, want %q", got, serviceType+".local")
	}

	browse := mdnsAnswers(device, []layers.DNSQuestion{{
		Name: []byte(serviceType + ".local"), Type: layers.DNSTypePTR, Class: layers.DNSClassIN,
	}})
	if len(browse) != 3 {
		t.Fatalf("browse returned %d answers, want PTR+SRV+TXT", len(browse))
	}

	var srv *layers.DNSResourceRecord
	for index := range browse {
		if browse[index].Type == layers.DNSTypeSRV {
			srv = &browse[index]
		}
	}
	if srv == nil {
		t.Fatal("browse carried no SRV record, so the port is unknown")
	}
	if srv.SRV.Port != 631 {
		t.Errorf("SRV port = %d, want 631", srv.SRV.Port)
	}
	if got := string(srv.SRV.Name); got != hostLabel+".local" {
		t.Errorf("SRV target = %q, want %q", got, hostLabel+".local")
	}
}

// A device that does not advertise must stay silent rather than answering for
// a name it does not own.
func TestMDNSIgnoresOtherNames(t *testing.T) {
	device := mdnsTestDevice("med-mri-b01-f01-03", "10.51.210.23", nil)

	answers := mdnsAnswers(device, []layers.DNSQuestion{{
		Name: []byte("someone-else.local"), Type: layers.DNSTypeA, Class: layers.DNSClassIN,
	}})
	if len(answers) != 0 {
		t.Errorf("answered for a name it does not own: %d record(s)", len(answers))
	}
}

func TestMDNSHandlerSkipsResponses(t *testing.T) {
	stack := newTestStackInternal(t)
	handler := NewMDNSHandler(stack, 0)
	device := mdnsTestDevice("med-mri-b01-f01-03", "10.51.210.23", nil)

	reply := &layers.DNS{ID: 1, QR: true}
	buf := gopacket.NewSerializeBuffer()
	if err := reply.SerializeTo(buf, gopacket.SerializeOptions{FixLengths: true}); err != nil {
		t.Fatalf("serialize: %v", err)
	}
	packet, udp, ip := mdnsRequestPacket(t, "10.254.200.100", mdnsGroupIPv4Test, buf.Bytes())

	handler.HandleQuery(&Packet{}, ip, udp, []*config.Device{device}, packet)

	select {
	case <-stack.sendQueue:
		t.Error("responded to another responder's reply, which would storm the segment")
	default:
	}
}

const mdnsGroupIPv4Test = "224.0.0.251"

func mdnsTestDevice(hostLabel, ip string, services []config.MDNSService) *config.Device {
	return &config.Device{
		Name:        hostLabel,
		MACAddress:  net.HardwareAddr{0x02, 0x00, 0x33, 0x00, 0x00, 0x23},
		IPAddresses: []net.IP{net.ParseIP(ip)},
		MDNSConfig: &config.MDNSConfig{
			Enabled: true, Hostname: hostLabel, TTL: 120, Services: services,
		},
	}
}

func mdnsRequestPacket(
	t *testing.T,
	srcIP, dstIP string,
	payload []byte,
) (gopacket.Packet, *layers.UDP, *layers.IPv4) {
	t.Helper()
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x00, 0x00, 0x01},
		DstMAC:       net.HardwareAddr{0x01, 0x00, 0x5e, 0x00, 0x00, 0xfb},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, TTL: 255, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP(srcIP).To4(), DstIP: net.ParseIP(dstIP).To4(),
	}
	udp := &layers.UDP{SrcPort: MDNSPort, DstPort: MDNSPort}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("checksum layer: %v", err)
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, gopacket.Payload(payload)); err != nil {
		t.Fatalf("serialize request: %v", err)
	}

	parsed := gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.Default)
	decodedUDP, ok := parsed.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if !ok {
		t.Fatal("request has no UDP layer")
	}
	decodedIP, ok := parsed.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	if !ok {
		t.Fatal("request has no IPv4 layer")
	}

	return parsed, decodedUDP, decodedIP
}
