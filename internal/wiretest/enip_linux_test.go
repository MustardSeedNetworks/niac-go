//go:build linux && integration

package wiretest_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// AI-7: the manufacturing plant's PLCs answer EtherNet/IP List Identity on
// UDP and TCP 44818 with their authored identity, and a device without
// EtherNet/IP refuses the port.
func TestPackPLCsAnswerListIdentity(t *testing.T) {
	authored, _ := startPack(t, "manufacturing")
	plc, hmi := enipPackDevices(t, authored)
	address := plc.IPAddresses[0].To4()
	t.Logf("EtherNet/IP PLC %s:44818", address)
	want := wantListIdentity(plc.EtherNetIPConfig, address)

	if got := udpListIdentity(t, address); !bytes.Equal(got, want) {
		t.Errorf("UDP List Identity answered % x, want % x", got, want)
	}
	if got := tcpListIdentity(t, address, len(want)); !bytes.Equal(got, want) {
		t.Errorf("TCP List Identity answered % x, want % x", got, want)
	}

	refused, err := net.DialTimeout("tcp4", net.JoinHostPort(hmi, "44818"), 2*time.Second)
	if err == nil {
		refused.Close()
		t.Fatal("the HMI, which has no EtherNet/IP, accepted a connection on 44818")
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		t.Fatalf("connecting to the HMI on 44818: %v, want connection refused", err)
	}
}

// enipPackDevices returns the pack's first addressed PLC and the address of
// an HMI, which has no EtherNet/IP.
func enipPackDevices(t *testing.T, authored *config.Config) (*config.Device, string) {
	t.Helper()
	var plc *config.Device
	var hmi string
	for index := range authored.Devices {
		device := &authored.Devices[index]
		if len(device.IPAddresses) == 0 {
			continue
		}
		switch role := device.Properties["role"]; {
		case role == "plc" && plc == nil:
			plc = device
		case role == "hmi":
			hmi = device.IPAddresses[0].String()
		}
	}
	if plc == nil || plc.EtherNetIPConfig == nil || hmi == "" {
		t.Fatalf("the manufacturing pack has no addressed EtherNet/IP PLC (%v) or HMI (%q)", plc, hmi)
	}

	return plc, hmi
}

// udpListIdentity sends List Identity to address over UDP, retrying while the
// pack settles, and returns the answer.
func udpListIdentity(t *testing.T, address net.IP) []byte {
	t.Helper()
	conn, err := net.Dial("udp4", net.JoinHostPort(address.String(), "44818"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	answer := make([]byte, 512)
	for range 10 {
		if _, err = conn.Write(listIdentityRequest()); err != nil {
			t.Fatal(err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		var n int
		if n, err = conn.Read(answer); err == nil {
			return answer[:n]
		}
	}
	t.Fatalf("no UDP List Identity answer from %s: %v", address, err)

	return nil
}

// tcpListIdentity sends List Identity to address over TCP, the header in two
// writes so the server has to reassemble it, and reads a size-byte answer.
func tcpListIdentity(t *testing.T, address net.IP, size int) []byte {
	t.Helper()
	conn := dialTCPRetrying(t, net.JoinHostPort(address.String(), "44818"))
	defer conn.Close()
	request := listIdentityRequest()
	for _, piece := range [][]byte{request[:10], request[10:]} {
		if _, err := conn.Write(piece); err != nil {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	answer := make([]byte, size)
	if _, err := io.ReadFull(conn, answer); err != nil {
		t.Fatalf("reading the TCP List Identity answer: %v", err)
	}

	return answer
}

// listIdentityRequest is a List Identity with sender context "AI-7test".
func listIdentityRequest() []byte {
	request := make([]byte, 24)
	request[0] = 0x63
	copy(request[12:], "AI-7test")

	return request
}

// wantListIdentity is the List Identity reply enip should send from address,
// assembled from the specification's field order rather than the server's
// code.
func wantListIdentity(enip *config.EtherNetIPConfig, address net.IP) []byte {
	item := binary.LittleEndian.AppendUint16(nil, 1)
	item = binary.BigEndian.AppendUint16(item, 2)
	item = binary.BigEndian.AppendUint16(item, 44818)
	item = append(item, address...)
	item = append(item, make([]byte, 8)...)
	item = binary.LittleEndian.AppendUint16(item, enip.VendorID)
	item = binary.LittleEndian.AppendUint16(item, enip.DeviceType)
	item = binary.LittleEndian.AppendUint16(item, enip.ProductCode)
	item = append(item, enip.RevisionMajor, enip.RevisionMinor, 0, 0)
	item = binary.LittleEndian.AppendUint32(item, enip.SerialNumber)
	item = append(item, byte(len(enip.ProductName)))
	item = append(item, enip.ProductName...)
	item = append(item, 3)

	data := binary.LittleEndian.AppendUint16(nil, 1)
	data = binary.LittleEndian.AppendUint16(data, 0x0c)
	data = binary.LittleEndian.AppendUint16(data, uint16(len(item)))
	data = append(data, item...)

	header := listIdentityRequest()
	binary.LittleEndian.PutUint16(header[2:], uint16(len(data)))

	return append(header, data...)
}
