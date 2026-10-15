//go:build linux && integration

package wiretest_test

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// sipPackPhone is a desk phone on the hospital pack's first site.
const sipPackPhone = "MED-PHONE-01"

// AI-1: a pack phone answers a scanner's SIP OPTIONS probe with the handset's
// User-Agent, from the address the probe was sent to, back to the port the
// probe came from.
func TestPackPhoneAnswersSIPOptions(t *testing.T) {
	authored, _ := startPack(t, "hospital")
	var host string
	for index := range authored.Devices {
		if device := &authored.Devices[index]; device.Name == sipPackPhone && len(device.IPAddresses) > 0 {
			host = device.IPAddresses[0].String()
		}
	}
	if host == "" {
		t.Fatalf("the hospital pack has no addressed %s", sipPackPhone)
	}

	conn, err := net.Dial("udp4", net.JoinHostPort(host, "5060"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	local := conn.LocalAddr().String()
	request := fmt.Sprintf("OPTIONS sip:100@%[1]s SIP/2.0\r\n"+
		"Via: SIP/2.0/UDP %[2]s;branch=z9hG4bK-wire;rport\r\n"+
		"Max-Forwards: 70\r\n"+
		"From: \"wiretest\" <sip:100@%[2]s>;tag=77\r\n"+
		"To: \"wiretest\" <sip:100@%[1]s>\r\n"+
		"Call-ID: ai1-wire@%[2]s\r\n"+
		"CSeq: 1 OPTIONS\r\n"+
		"Accept: application/sdp\r\n"+
		"Content-Length: 0\r\n\r\n", host, local)

	// UDP: the pack may still be settling, so resend until it answers.
	buffer := make([]byte, 2048)
	var response string
	for attempt := 0; attempt < 10 && response == ""; attempt++ {
		if _, err = conn.Write([]byte(request)); err != nil {
			t.Fatal(err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if n, readErr := conn.Read(buffer); readErr == nil {
			response = string(buffer[:n])
		}
	}
	t.Logf("OPTIONS %s:5060 answered:\n%s", host, response)
	if response == "" {
		t.Fatalf("%s did not answer OPTIONS", sipPackPhone)
	}
	for _, want := range []string{
		"SIP/2.0 200 OK\r\n",
		"User-Agent: Cisco-CP8841/12.8.1\r\n",
		"Call-ID: ai1-wire@" + local + "\r\n",
		"CSeq: 1 OPTIONS\r\n",
		";rport=",
	} {
		if !strings.Contains(response, want) {
			t.Errorf("response lacks %q", want)
		}
	}
}
