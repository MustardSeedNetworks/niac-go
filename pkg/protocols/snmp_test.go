package protocols

import (
	"net"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/gosnmp/gosnmp"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

func TestSNMPHandler_HandlePacket(t *testing.T) {
	deviceMAC := net.HardwareAddr{0x00, 0xaa, 0xbb, 0xcc, 0xdd, 0xee}
	deviceIP := net.ParseIP("10.0.0.10").To4()

	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "snmp-device",
				Type:        "router",
				MACAddress:  deviceMAC,
				IPAddresses: []net.IP{deviceIP},
				SNMPConfig: config.SNMPConfig{
					Community: "public",
					SysName:   "snmp-device",
				},
			},
		},
	}

	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := stack.snmpHandler
	if handler == nil {
		t.Fatal("snmp handler should be initialized")
	}
	t.Logf("device community: %s", cfg.Devices[0].SNMPConfig.Community)
	if !snmpEnabled(cfg.Devices[0].SNMPConfig) {
		t.Fatalf("expected snmpEnabled true")
	}
	if len(stack.snmpAgents) != 1 {
		t.Fatalf("expected 1 SNMP agent, got %d", len(stack.snmpAgents))
	}

	req := &gosnmp.SnmpPacket{
		Version:   gosnmp.Version2c,
		Community: "public",
		PDUType:   gosnmp.GetRequest,
		Variables: []gosnmp.SnmpPDU{
			{
				Name: ".1.3.6.1.2.1.1.5.0", // sysName
				Type: gosnmp.Null,
			},
		},
	}

	payload, err := req.MarshalMsg()
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	udpLayer := &layers.UDP{
		SrcPort: 40000,
		DstPort: layers.UDPPort(UDPPortSNMP),
	}
	udpLayer.Payload = payload

	ipLayer := &layers.IPv4{
		SrcIP: net.ParseIP("10.0.0.5"),
		DstIP: deviceIP,
	}

	frame := make([]byte, 14)
	copy(frame[0:6], deviceMAC)
	copy(frame[6:12], []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55})
	frame[12] = 0x08
	frame[13] = 0x00

	packet := &Packet{
		Buffer:       frame,
		Length:       len(frame),
		SerialNumber: 1,
	}

	handler.HandlePacket(packet, ipLayer, udpLayer, []*config.Device{&cfg.Devices[0]})

	select {
	case resp := <-stack.sendQueue:
		decoded := gopacket.NewPacket(resp.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		udpLayerResp := decoded.Layer(layers.LayerTypeUDP)
		if udpLayerResp == nil {
			t.Fatal("expected UDP layer in response")
		}
		respUDP := udpLayerResp.(*layers.UDP)

		decoder := gosnmp.GoSNMP{
			Transport: "udp",
			Version:   gosnmp.Version2c,
			Community: "public",
		}
		respSNMP, err := decoder.SnmpDecodePacket(respUDP.Payload)
		if err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if respSNMP.PDUType != gosnmp.GetResponse {
			t.Fatalf("expected GetResponse, got %v", respSNMP.PDUType)
		}
		if len(respSNMP.Variables) != 1 {
			t.Fatalf("expected 1 varbind, got %d", len(respSNMP.Variables))
		}
		switch v := respSNMP.Variables[0].Value.(type) {
		case string:
			if v != "snmp-device" {
				t.Fatalf("unexpected response value %v", v)
			}
		case []byte:
			if string(v) != "snmp-device" {
				t.Fatalf("unexpected response value %v", v)
			}
		default:
			t.Fatalf("unexpected response type %T", v)
		}
	default:
		t.Fatal("expected SNMP response to be sent")
	}

	stats := stack.GetStats()
	if stats.SNMPQueries != 1 {
		t.Fatalf("expected SNMPQueries=1, got %d", stats.SNMPQueries)
	}
}
