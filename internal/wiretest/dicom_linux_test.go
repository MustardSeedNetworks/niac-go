//go:build linux && integration

package wiretest_test

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

// AI-4: the hospital's MRI and its PACS archive each accept a DICOM
// association and answer C-ECHO, on both DICOM ports, so the modality and the
// archive it sends to verify each other.
func TestPackModalityAndArchiveAnswerCEcho(t *testing.T) {
	authored, _ := startPack(t, "hospital")
	hosts := map[string]string{}
	for index := range authored.Devices {
		device := &authored.Devices[index]
		if len(device.IPAddresses) == 0 {
			continue
		}
		switch {
		case device.Name == "MED-PACS01":
			hosts["archive"] = device.IPAddresses[0].String()
		case device.Properties["role"] == "mr-system" && hosts["modality"] == "":
			hosts["modality"] = device.IPAddresses[0].String()
		}
	}
	for _, node := range []string{"modality", "archive"} {
		if hosts[node] == "" {
			t.Fatalf("the hospital pack has no addressed %s", node)
		}
		for _, port := range []string{"104", "11112"} {
			t.Logf("C-ECHO %s %s:%s", node, hosts[node], port)
			cEcho(t, net.JoinHostPort(hosts[node], port))
		}
	}
}

const (
	wireVerificationClass = "1.2.840.10008.1.1"
	wireImplicitLittle    = "1.2.840.10008.1.2"
)

// cEcho runs what echoscu does: associate for Verification, send C-ECHO-RQ,
// require a C-ECHO-RSP of status Success, and release. The pack may still be
// settling, so the connection is retried.
func cEcho(t *testing.T, address string) {
	t.Helper()
	var conn net.Conn
	var err error
	for range 10 {
		if conn, err = net.DialTimeout("tcp4", address, 2*time.Second); err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		t.Fatalf("connecting to %s: %v", address, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	if pduType, body := wireDICOMExchange(t, conn, wireAssociateRequest()); pduType != 0x02 ||
		!bytes.Contains(body, []byte{0x21, 0, 0, 0x19, 1, 0, 0, 0}) {
		t.Fatalf("%s: association answered with PDU %#x % x, want the verification context accepted",
			address, pduType, body)
	}
	pduType, body := wireDICOMExchange(t, conn, wireEchoRequest())
	status := []byte{0, 0, 0x00, 0x09, 2, 0, 0, 0, 0, 0} // (0000,0900) Status = Success
	if pduType != 0x04 || !bytes.HasSuffix(body, status) || !bytes.Contains(body, []byte{0x30, 0x80}) {
		t.Fatalf("%s: C-ECHO answered with PDU %#x % x, want a successful C-ECHO-RSP", address, pduType, body)
	}
	if pduType, _ = wireDICOMExchange(t, conn, wirePDU(0x05, make([]byte, 4))); pduType != 0x06 {
		t.Fatalf("%s: release answered with PDU %#x, want A-RELEASE-RP", address, pduType)
	}
}

func wireDICOMExchange(t *testing.T, conn net.Conn, request []byte) (byte, []byte) {
	t.Helper()
	if _, err := conn.Write(request); err != nil {
		t.Fatal(err)
	}
	header := make([]byte, 6)
	if _, err := io.ReadFull(conn, header); err != nil {
		t.Fatalf("reading a PDU header: %v", err)
	}
	body := make([]byte, binary.BigEndian.Uint32(header[2:]))
	if _, err := io.ReadFull(conn, body); err != nil {
		t.Fatalf("reading a PDU body: %v", err)
	}

	return header[0], body
}

func wireAssociateRequest() []byte {
	body := []byte{0, 1, 0, 0}
	body = append(body, []byte("ANY-SCP         ECHOSCU         ")...)
	body = append(body, make([]byte, 32)...)
	body = append(body, wireItem(0x10, []byte("1.2.840.10008.3.1.1.1"))...)
	context := append([]byte{1, 0, 0, 0}, wireItem(0x30, []byte(wireVerificationClass))...)
	context = append(context, wireItem(0x40, []byte(wireImplicitLittle))...)
	body = append(body, wireItem(0x20, context)...)
	body = append(body, wireItem(0x50, wireItem(0x51, []byte{0, 0, 0x40, 0}))...)

	return wirePDU(0x01, body)
}

func wireEchoRequest() []byte {
	var elements []byte
	for _, element := range []struct {
		tag   uint16
		value []byte
	}{
		{0x0002, append([]byte(wireVerificationClass), 0)},
		{0x0100, []byte{0x30, 0}},
		{0x0110, []byte{1, 0}},
		{0x0800, []byte{0x01, 0x01}},
	} {
		elements = wireElement(elements, element.tag, element.value)
	}
	command := wireElement(nil, 0x0000, binary.LittleEndian.AppendUint32(nil, uint32(len(elements))))
	command = append(command, elements...)
	pdv := binary.BigEndian.AppendUint32(nil, uint32(len(command)+2))
	pdv = append(pdv, 1, 0x03)

	return wirePDU(0x04, append(pdv, command...))
}

func wireElement(buffer []byte, tag uint16, value []byte) []byte {
	buffer = binary.LittleEndian.AppendUint16(buffer, 0)
	buffer = binary.LittleEndian.AppendUint16(buffer, tag)
	buffer = binary.LittleEndian.AppendUint32(buffer, uint32(len(value)))

	return append(buffer, value...)
}

func wireItem(itemType byte, value []byte) []byte {
	return append(binary.BigEndian.AppendUint16([]byte{itemType, 0}, uint16(len(value))), value...)
}

func wirePDU(pduType byte, body []byte) []byte {
	return append(binary.BigEndian.AppendUint32([]byte{pduType, 0}, uint32(len(body))), body...)
}
