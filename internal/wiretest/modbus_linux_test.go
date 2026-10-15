//go:build linux && integration

package wiretest_test

import (
	"bytes"
	"errors"
	"io"
	"net"
	"syscall"
	"testing"
	"time"
)

// AI-6: the manufacturing plant's PLCs answer Modbus/TCP register reads on
// 502, an invalid read gets the exception a server returns, and a device
// without Modbus refuses the port.
func TestPackPLCsAnswerModbus(t *testing.T) {
	authored, _ := startPack(t, "manufacturing")
	var plc, hmi string
	for index := range authored.Devices {
		device := &authored.Devices[index]
		if len(device.IPAddresses) == 0 {
			continue
		}
		switch device.Properties["role"] {
		case "plc":
			plc = device.IPAddresses[0].String()
		case "hmi":
			hmi = device.IPAddresses[0].String()
		}
	}
	if plc == "" || hmi == "" {
		t.Fatalf("the manufacturing pack has no addressed PLC (%q) or HMI (%q)", plc, hmi)
	}
	t.Logf("Modbus PLC %s:502", plc)

	conn := dialTCPRetrying(t, net.JoinHostPort(plc, "502"))
	defer conn.Close()
	exchanges := []struct {
		name          string
		request, want []byte
	}{
		{
			name:    "two holding registers",
			request: []byte{0, 1, 0, 0, 0, 6, 1, 0x03, 0, 0, 0, 2},
			want:    []byte{0, 1, 0, 0, 0, 7, 1, 0x03, 4, 0, 0, 0, 0},
		},
		{
			name:    "last input register",
			request: []byte{0, 2, 0, 0, 0, 6, 1, 0x04, 0, 99, 0, 1},
			want:    []byte{0, 2, 0, 0, 0, 5, 1, 0x04, 2, 0, 0},
		},
		{
			name:    "past the last holding register",
			request: []byte{0, 3, 0, 0, 0, 6, 1, 0x03, 0, 100, 0, 1},
			want:    []byte{0, 3, 0, 0, 0, 3, 1, 0x83, 0x02},
		},
		{
			name:    "quantity over 125",
			request: []byte{0, 4, 0, 0, 0, 6, 1, 0x03, 0, 0, 0, 126},
			want:    []byte{0, 4, 0, 0, 0, 3, 1, 0x83, 0x03},
		},
		{
			name:    "write single register",
			request: []byte{0, 5, 0, 0, 0, 6, 1, 0x06, 0, 0, 0, 1},
			want:    []byte{0, 5, 0, 0, 0, 3, 1, 0x86, 0x01},
		},
	}
	for _, exchange := range exchanges {
		// The MBAP header and the PDU go in separate writes, so the server
		// has to reassemble the request.
		for _, piece := range [][]byte{exchange.request[:7], exchange.request[7:]} {
			if _, err := conn.Write(piece); err != nil {
				t.Fatal(err)
			}
			time.Sleep(50 * time.Millisecond)
		}
		got := make([]byte, len(exchange.want))
		if _, err := io.ReadFull(conn, got); err != nil {
			t.Fatalf("%s: reading the answer: %v", exchange.name, err)
		}
		if !bytes.Equal(got, exchange.want) {
			t.Errorf("%s: answered % x, want % x", exchange.name, got, exchange.want)
		}
	}

	refused, err := net.DialTimeout("tcp4", net.JoinHostPort(hmi, "502"), 2*time.Second)
	if err == nil {
		refused.Close()
		t.Fatal("the HMI, which has no Modbus, accepted a connection on 502")
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		t.Fatalf("connecting to the HMI on 502: %v, want connection refused", err)
	}
}

// dialTCPRetrying connects to address over TCP, retrying while the pack
// settles.
func dialTCPRetrying(t *testing.T, address string) net.Conn {
	t.Helper()
	var conn net.Conn
	var err error
	for range 10 {
		if conn, err = net.DialTimeout("tcp4", address, 2*time.Second); err == nil {
			_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
			return conn
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("connecting to %s: %v", address, err)

	return nil
}
