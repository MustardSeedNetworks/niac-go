//go:build linux && integration

package wiretest_test

import (
	"bufio"
	"errors"
	"net"
	"strings"
	"syscall"
	"testing"
	"time"
)

// AI-5: the hospital's EMR, laboratory and pharmacy systems acknowledge an
// HL7 message framed in MLLP on 2575, even when it arrives in pieces, and a
// device without HL7 refuses the port.
func TestPackClinicalSystemsAcknowledgeHL7(t *testing.T) {
	authored, _ := startPack(t, "hospital")
	hosts := map[string]string{}
	for index := range authored.Devices {
		device := &authored.Devices[index]
		if len(device.IPAddresses) > 0 {
			hosts[device.Name] = device.IPAddresses[0].String()
		}
	}
	for _, name := range []string{"MED-EMR01", "MED-LIS01", "MED-RX01"} {
		if hosts[name] == "" {
			t.Fatalf("the hospital pack has no addressed %s", name)
		}
		t.Logf("HL7 %s %s:2575", name, hosts[name])
		ack := mllpExchange(t, net.JoinHostPort(hosts[name], "2575"))
		if !strings.HasPrefix(ack, "MSH|^~\\&|LAB|MAINLAB|EPIC|MAINHOSP|") ||
			!strings.Contains(ack, "|ACK^A01^ACK|") || !strings.HasSuffix(ack, "\rMSA|AA|WIRE0001\r") {
			t.Errorf("%s acknowledged with %q", name, ack)
		}
	}
	conn, err := net.DialTimeout("tcp4", net.JoinHostPort(hosts["MED-PACS01"], "2575"), 2*time.Second)
	if err == nil {
		conn.Close()
		t.Fatal("the PACS, which has no HL7, accepted a connection on 2575")
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		t.Fatalf("connecting to the PACS on 2575: %v, want connection refused", err)
	}
}

// mllpExchange sends an ADT^A01 the way an interface engine does, the start
// of the frame and the rest in separate writes, and returns the message the
// ACK frame carries. The pack may still be settling, so the connection is
// retried.
func mllpExchange(t *testing.T, address string) string {
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

	message := "\x0bMSH|^~\\&|EPIC|MAINHOSP|LAB|MAINLAB|20261015050000||ADT^A01^ADT_A01|WIRE0001|P|2.5.1\r" +
		"EVN|A01|20261015050000\rPID|1||12345^^^MRN||DOE^JANE\r\x1c\r"
	for _, piece := range []string{message[:30], message[30:]} {
		if _, err = conn.Write([]byte(piece)); err != nil {
			t.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	framed, err := bufio.NewReader(conn).ReadString('\x1c')
	if err != nil {
		t.Fatalf("%s: reading the ACK: %v", address, err)
	}
	ack, ok := strings.CutPrefix(strings.TrimSuffix(framed, "\x1c"), "\x0b")
	if !ok {
		t.Fatalf("%s: ACK %q has no start block", address, framed)
	}

	return ack
}
