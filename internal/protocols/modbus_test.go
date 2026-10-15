package protocols

import (
	"bytes"
	"net"
	"testing"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// modbusRequest is an MBAP header for transaction 0x1234 and unit 0x11
// carrying pdu.
func modbusRequest(pdu ...byte) []byte {
	return append([]byte{0x12, 0x34, 0x00, 0x00, 0x00, byte(1 + len(pdu)), 0x11}, pdu...)
}

func TestModbusAnswer(t *testing.T) {
	t.Parallel()

	server := &config.ModbusConfig{Enabled: true, HoldingRegisters: 100, InputRegisters: 10}
	tests := []struct {
		name     string
		stream   []byte
		want     []byte
		consumed int
		ended    bool
	}{
		{
			name:   "holding registers",
			stream: modbusRequest(0x03, 0x00, 0x00, 0x00, 0x02),
			want:   []byte{0x12, 0x34, 0, 0, 0, 7, 0x11, 0x03, 4, 0, 0, 0, 0}, consumed: 12,
		},
		{
			name:   "last input register",
			stream: modbusRequest(0x04, 0x00, 0x09, 0x00, 0x01),
			want:   []byte{0x12, 0x34, 0, 0, 0, 5, 0x11, 0x04, 2, 0, 0}, consumed: 12,
		},
		{
			name:   "past the last input register",
			stream: modbusRequest(0x04, 0x00, 0x09, 0x00, 0x02),
			want:   []byte{0x12, 0x34, 0, 0, 0, 3, 0x11, 0x84, 0x02}, consumed: 12,
		},
		{
			name:   "address beyond the holding registers",
			stream: modbusRequest(0x03, 0x00, 0x64, 0x00, 0x01),
			want:   []byte{0x12, 0x34, 0, 0, 0, 3, 0x11, 0x83, 0x02}, consumed: 12,
		},
		{
			name:   "quantity zero",
			stream: modbusRequest(0x03, 0x00, 0x00, 0x00, 0x00),
			want:   []byte{0x12, 0x34, 0, 0, 0, 3, 0x11, 0x83, 0x03}, consumed: 12,
		},
		{
			name:   "quantity over 125 is checked before the address",
			stream: modbusRequest(0x03, 0x00, 0x00, 0x00, 0x7e),
			want:   []byte{0x12, 0x34, 0, 0, 0, 3, 0x11, 0x83, 0x03}, consumed: 12,
		},
		{
			name:   "short read request",
			stream: modbusRequest(0x03, 0x00, 0x00),
			want:   []byte{0x12, 0x34, 0, 0, 0, 3, 0x11, 0x83, 0x03}, consumed: 10,
		},
		{
			name:   "unsupported function",
			stream: modbusRequest(0x06, 0x00, 0x01, 0x00, 0x03),
			want:   []byte{0x12, 0x34, 0, 0, 0, 3, 0x11, 0x86, 0x01}, consumed: 12,
		},
		{
			name:   "device identification",
			stream: modbusRequest(0x2b, 0x0e, 0x01, 0x00),
			want:   []byte{0x12, 0x34, 0, 0, 0, 3, 0x11, 0xab, 0x01}, consumed: 11,
		},
		{
			name:   "two requests in one segment",
			stream: append(modbusRequest(0x03, 0, 0, 0, 1), modbusRequest(0x01, 0, 0, 0, 1)...),
			want: []byte{
				0x12, 0x34, 0, 0, 0, 5, 0x11, 0x03, 2, 0, 0,
				0x12, 0x34, 0, 0, 0, 3, 0x11, 0x81, 0x01,
			},
			consumed: 24,
		},
		{name: "partial header waits", stream: modbusRequest(0x03)[:5]},
		{name: "partial request waits", stream: modbusRequest(0x03, 0x00, 0x00, 0x00, 0x01)[:9]},
		{
			name:     "other protocol ID discarded",
			stream:   []byte{0x12, 0x34, 0x00, 0x01, 0x00, 0x06, 0x11, 0x03, 0, 0, 0, 1},
			consumed: 12,
		},
		{name: "length too short ends", stream: []byte{0, 1, 0, 0, 0, 1, 0x11}, ended: true},
		{name: "not Modbus ends", stream: []byte("GET / HTTP/1.1\r\n\r\n"), ended: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			answer, consumed, ended := modbusAnswer(test.stream, server)
			if !bytes.Equal(answer, test.want) || consumed != test.consumed || ended != test.ended {
				t.Errorf("modbusAnswer(% x) = % x, %d, %v; want % x, %d, %v",
					test.stream, answer, consumed, ended, test.want, test.consumed, test.ended)
			}
		})
	}
}

func modbusTestNode(modbus *config.ModbusConfig) *config.Device {
	return &config.Device{
		Name:         "PLT-PLC01",
		MACAddress:   net.HardwareAddr{0x00, 0x00, 0xbc, 0x00, 0x00, 0x06},
		IPAddresses:  []net.IP{net.ParseIP("10.20.30.44")},
		ModbusConfig: modbus,
	}
}

// A read split across segments is answered once whole, and the connection
// closes with the client's FIN.
func TestTCPHandlerServesModbus(t *testing.T) {
	conversation := newServiceConversation(t,
		modbusTestNode(&config.ModbusConfig{Enabled: true, HoldingRegisters: 100}), TCPPortModbus)
	if reply := conversation.reply(&layers.TCP{SYN: true, Seq: 100}); reply == nil ||
		!reply.SYN || !reply.ACK || reply.Ack != 101 {
		t.Fatalf("SYN answered with %+v, want a SYN-ACK acknowledging 101", reply)
	}
	request := modbusRequest(0x03, 0x00, 0x00, 0x00, 0x01)
	if reply := conversation.reply(
		&layers.TCP{ACK: true, PSH: true, Seq: 101, Ack: 1001, Payload: request[:7]},
	); reply == nil ||
		len(reply.Payload) != 0 ||
		reply.Ack != 108 {
		t.Fatalf("header answered with %+v, want a bare ACK of 108", reply)
	}
	reply := conversation.reply(&layers.TCP{ACK: true, PSH: true, FIN: true, Seq: 108, Ack: 1001, Payload: request[7:]})
	want := []byte{0x12, 0x34, 0, 0, 0, 5, 0x11, 0x03, 2, 0, 0}
	if reply == nil || !reply.FIN || reply.Seq != 1001 || reply.Ack != 114 || !bytes.Equal(reply.Payload, want) {
		t.Errorf("read with FIN answered with %+v, want % x and FIN", reply, want)
	}
}

// A device without Modbus keeps the closed-port RST, as does one whose
// Modbus is switched off, and never answers a request.
func TestTCPHandlerModbusClosedWithoutModbus(t *testing.T) {
	for _, modbus := range []*config.ModbusConfig{nil, {HoldingRegisters: 100}} {
		device := modbusTestNode(modbus)
		if reply := serviceReply(
			t,
			device,
			TCPPortModbus,
			&layers.TCP{SYN: true, Seq: 500},
		); reply == nil ||
			!reply.RST {
			t.Errorf("Modbus %+v: SYN answered with %+v, want RST", modbus, reply)
		}
		data := &layers.TCP{ACK: true, Seq: 501, Ack: 1001, Payload: modbusRequest(0x03, 0, 0, 0, 1)}
		if reply := serviceReply(t, device, TCPPortModbus, data); reply != nil {
			t.Errorf("Modbus %+v: request answered with %+v", modbus, reply)
		}
	}
}
