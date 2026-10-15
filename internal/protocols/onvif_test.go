package protocols

import (
	"encoding/xml"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func onvifTestCamera() *config.Device {
	return &config.Device{
		Name:        "MED-CAM-01",
		MACAddress:  net.HardwareAddr{0xac, 0xcc, 0x8e, 0x00, 0x00, 0x03},
		IPAddresses: []net.IP{net.ParseIP("10.20.30.41")},
		ONVIFConfig: &config.ONVIFConfig{Enabled: true, Name: "AXIS P3265-LVE", Hardware: "P3265-LVE"},
	}
}

// wsdTestProbe is the Probe ONVIF Device Manager sends, with types and scopes
// spliced in. The discovery namespace is bound to a prefix other than the one
// the response uses, as clients do.
func wsdTestProbe(types, scopes string) string {
	return `<?xml version="1.0" encoding="utf-8"?>` +
		`<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" ` +
		`xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing">` +
		`<s:Header>` +
		`<a:Action s:mustUnderstand="1">http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</a:Action>` +
		`<a:MessageID>urn:uuid:0c2e5c4a-5e2f-4c43-9a1e-2b1d3b6a7f10</a:MessageID>` +
		`<a:ReplyTo><a:Address>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:Address></a:ReplyTo>` +
		`<a:To s:mustUnderstand="1">urn:schemas-xmlsoap-org:ws:2005:04:discovery</a:To>` +
		`</s:Header>` +
		`<s:Body><Probe xmlns="http://schemas.xmlsoap.org/ws/2005/04/discovery">` +
		types + scopes +
		`</Probe></s:Body></s:Envelope>`
}

const (
	wsdNVTTypes = `<d:Types xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery" ` +
		`xmlns:dp0="http://www.onvif.org/ver10/network/wsdl">dp0:NetworkVideoTransmitter</d:Types>`
	wsdDeviceTypes = `<Types xmlns:tds="http://www.onvif.org/ver10/device/wsdl">tds:Device</Types>`
)

func TestONVIFProbeMatch(t *testing.T) {
	tests := []struct {
		name  string
		probe string
	}{
		{"no types or scopes matches any target", wsdTestProbe("", "")},
		{"network video transmitter", wsdTestProbe(wsdNVTTypes, "")},
		{"device type under the default namespace", wsdTestProbe(wsdDeviceTypes, "")},
		{"scope prefix on a segment", wsdTestProbe("", "<Scopes>onvif://www.onvif.org/type</Scopes>")},
		{"hardware scope", wsdTestProbe(wsdNVTTypes, "<Scopes>onvif://www.onvif.org/hardware/P3265-LVE</Scopes>")},
		{
			"exact scope under strcmp0",
			wsdTestProbe("", `<Scopes MatchBy="http://schemas.xmlsoap.org/ws/2005/04/discovery/strcmp0">`+
				`onvif://www.onvif.org/name/AXIS%20P3265-LVE</Scopes>`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, ok := onvifProbeMatch([]byte(test.probe), onvifTestCamera(), net.ParseIP("10.20.30.41").To4(), 1)
			if !ok {
				t.Fatal("no ProbeMatch to a Probe the camera satisfies")
			}
			text := string(response)
			for _, want := range []string{
				`<wsa:RelatesTo>urn:uuid:0c2e5c4a-5e2f-4c43-9a1e-2b1d3b6a7f10</wsa:RelatesTo>`,
				`<wsa:Action>http://schemas.xmlsoap.org/ws/2005/04/discovery/ProbeMatches</wsa:Action>`,
				`<d:Types>dn:NetworkVideoTransmitter tds:Device</d:Types>`,
				`onvif://www.onvif.org/hardware/P3265-LVE`,
				`onvif://www.onvif.org/name/AXIS%20P3265-LVE`,
				`<d:XAddrs>http://10.20.30.41/onvif/device_service</d:XAddrs>`,
				`<wsa:Address>urn:uuid:` + onvifEndpointUUID(onvifTestCamera()) + `</wsa:Address>`,
			} {
				if !strings.Contains(text, want) {
					t.Errorf("ProbeMatch lacks %s:\n%s", want, text)
				}
			}
			if err := xmlWellFormed(response); err != nil {
				t.Errorf("ProbeMatch is not well-formed XML: %v", err)
			}
		})
	}
}

func TestONVIFProbeMatchIgnores(t *testing.T) {
	valid := wsdTestProbe("", "")
	tests := []struct {
		name  string
		probe string
	}{
		{"a type the camera is not", wsdTestProbe(
			`<Types xmlns:p="http://schemas.microsoft.com/windows/2006/08/wdp/print">p:PrintDeviceType</Types>`, "")},
		{"an unbound prefix", wsdTestProbe(`<Types>dn:NetworkVideoTransmitter</Types>`, "")},
		{"a scope the camera lacks", wsdTestProbe("", "<Scopes>onvif://www.onvif.org/hardware/M3106</Scopes>")},
		{"a scope prefix inside a segment", wsdTestProbe("", "<Scopes>onvif://www.onvif.org/hard</Scopes>")},
		{
			"strcmp0 on a prefix",
			wsdTestProbe("", `<Scopes MatchBy="http://schemas.xmlsoap.org/ws/2005/04/discovery/strcmp0">`+
				`onvif://www.onvif.org/type</Scopes>`),
		},
		{"an unknown MatchBy", wsdTestProbe("", `<Scopes MatchBy="urn:example:regex">onvif</Scopes>`)},
		{"a Resolve", strings.Replace(valid, "discovery/Probe<", "discovery/Resolve<", 1)},
		{"no MessageID", strings.Replace(valid,
			"<a:MessageID>urn:uuid:0c2e5c4a-5e2f-4c43-9a1e-2b1d3b6a7f10</a:MessageID>", "", 1)},
		{"SOAP 1.1", strings.Replace(valid, "http://www.w3.org/2003/05/soap-envelope",
			"http://schemas.xmlsoap.org/soap/envelope/", 1)},
		{"WS-Discovery 1.1", strings.ReplaceAll(valid, "ws/2005/04/discovery", "ws/2009/01/discovery")},
		{"malformed XML", valid[:len(valid)-10]},
		{"oversized", valid + strings.Repeat(" ", wsdMaxProbeBytes)},
		{"empty", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if response, ok := onvifProbeMatch(
				[]byte(test.probe),
				onvifTestCamera(),
				net.ParseIP("10.20.30.41"),
				1,
			); ok {
				t.Errorf("answered:\n%s", response)
			}
		})
	}
}

// The endpoint reference is what a client recognises a camera by, so it must
// not change between probes or restarts, must differ between cameras, and must
// be a valid UUID carrying the camera's MAC.
func TestONVIFEndpointUUIDIsStable(t *testing.T) {
	first := onvifEndpointUUID(onvifTestCamera())
	if first != onvifEndpointUUID(onvifTestCamera()) {
		t.Fatal("endpoint UUID changes between calls")
	}
	other := onvifTestCamera()
	other.Name = "MED-CAM-02"
	if onvifEndpointUUID(other) == first {
		t.Fatal("two cameras share an endpoint UUID")
	}
	if len(first) != 36 || first[14] != '8' || !strings.HasSuffix(first, "-accc8e000003") {
		t.Fatalf("endpoint UUID %s is not a version 8 UUID ending in the MAC", first)
	}
}

func TestONVIFProbeMatchEscapesTheRelatedMessage(t *testing.T) {
	probe := strings.Replace(wsdTestProbe("", ""),
		"urn:uuid:0c2e5c4a-5e2f-4c43-9a1e-2b1d3b6a7f10", "urn:x:&lt;/wsa:RelatesTo&gt;", 1)
	response, ok := onvifProbeMatch([]byte(probe), onvifTestCamera(), net.ParseIP("10.20.30.41").To4(), 1)
	if !ok {
		t.Fatal("no ProbeMatch")
	}
	if !strings.Contains(string(response), "<wsa:RelatesTo>urn:x:&lt;/wsa:RelatesTo&gt;</wsa:RelatesTo>") {
		t.Errorf("RelatesTo not escaped:\n%s", response)
	}
	if err := xmlWellFormed(response); err != nil {
		t.Errorf("ProbeMatch is not well-formed XML: %v", err)
	}
}

func buildWSDDatagram(t *testing.T, device *config.Device, payload string) (*Packet, *layers.IPv4) {
	t.Helper()
	eth := &layers.Ethernet{
		SrcMAC: net.HardwareAddr{0xaa, 0, 0, 0, 0, 1}, DstMAC: device.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, IHL: 5, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("192.0.2.9").To4(), DstIP: device.IPAddresses[0].To4(),
	}
	udp := &layers.UDP{SrcPort: 51234, DstPort: UDPPortWSDiscovery}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatal(err)
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, gopacket.Payload(payload)); err != nil {
		t.Fatal(err)
	}

	return &Packet{Buffer: buf.Bytes(), Length: len(buf.Bytes()), SerialNumber: 1}, ip
}

func TestUDPHandlerAnswersWSDiscoveryProbe(t *testing.T) {
	stack := newTestStackInternal(t)
	handler := NewUDPHandler(stack)
	camera := onvifTestCamera()

	pkt, ip := buildWSDDatagram(t, camera, wsdTestProbe(wsdNVTTypes, ""))
	handler.HandlePacket(pkt, ip, []*config.Device{camera})

	var reply *Packet
	select {
	case reply = <-stack.sendQueue:
	default:
		t.Fatal("no ProbeMatch was queued")
	}
	decoded := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ipOut, _ := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	udpOut, _ := decoded.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if ipOut == nil || udpOut == nil {
		t.Fatalf("response is not UDP/IPv4: %v", decoded)
	}
	if !ipOut.SrcIP.Equal(camera.IPAddresses[0]) || !ipOut.DstIP.Equal(ip.SrcIP) {
		t.Errorf("response %s -> %s, want %s -> %s", ipOut.SrcIP, ipOut.DstIP, camera.IPAddresses[0], ip.SrcIP)
	}
	if udpOut.SrcPort != UDPPortWSDiscovery || udpOut.DstPort != 51234 {
		t.Errorf("response ports %d -> %d, want 3702 -> 51234", udpOut.SrcPort, udpOut.DstPort)
	}
	if !strings.Contains(string(udpOut.Payload), "onvif://www.onvif.org/name/AXIS%20P3265-LVE") {
		t.Errorf("ProbeMatch does not name the camera:\n%s", udpOut.Payload)
	}
}

// A device without ONVIF keeps the closed-port answer it gave before 3702 had
// a case of its own, and so does a camera whose listener is switched off.
func TestUDPHandlerWSDiscoveryPortClosedWithoutONVIF(t *testing.T) {
	for _, onvif := range []*config.ONVIFConfig{nil, {Enabled: false, Name: "AXIS P3265-LVE", Hardware: "P3265-LVE"}} {
		stack := newTestStackInternal(t)
		stack.icmpHandler = NewICMPHandler(stack)
		handler := NewUDPHandler(stack)
		device := onvifTestCamera()
		device.ONVIFConfig = onvif

		pkt, ip := buildWSDDatagram(t, device, wsdTestProbe("", ""))
		handler.HandlePacket(pkt, ip, []*config.Device{device})

		select {
		case reply := <-stack.sendQueue:
			decoded := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
			icmp, _ := decoded.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4)
			if icmp == nil || icmp.TypeCode.Code() != layers.ICMPv4CodePort {
				t.Errorf("ONVIF %+v: reply is not port unreachable: %v", onvif, decoded)
			}
		default:
			t.Errorf("ONVIF %+v: no port unreachable for a closed 3702", onvif)
		}
	}
}

func xmlWellFormed(document []byte) error {
	decoder := xml.NewDecoder(strings.NewReader(string(document)))
	for {
		if _, err := decoder.Token(); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}
