//go:build linux && integration

package wiretest_test

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"
)

// jetDirectPackPrinter is the HP multifunction printer on the campus pack's
// first site.
const jetDirectPackPrinter = "NTH-PRN-B01-F01-05"

// AI-3: a pack printer accepts a connection on 9100 and answers nmap's PJL
// INFO ID probe and a PJL INFO STATUS query with its model and a ready state.
func TestPackPrinterAnswersPJL(t *testing.T) {
	authored, _ := startPack(t, "campus")
	var host string
	for index := range authored.Devices {
		if device := &authored.Devices[index]; device.Name == jetDirectPackPrinter && len(device.IPAddresses) > 0 {
			host = device.IPAddresses[0].String()
		}
	}
	if host == "" {
		t.Fatalf("the campus pack has no addressed %s", jetDirectPackPrinter)
	}

	for _, query := range []struct{ request, want string }{
		{"\x1b%-12345X@PJL INFO ID\r\n\x1b%-12345X\r\n", "@PJL INFO ID\r\n\"HP LaserJet MFP M634\"\r\n\f"},
		{"@PJL INFO STATUS\r\n", "@PJL INFO STATUS\r\nCODE=10001\r\nDISPLAY=\"Ready\"\r\nONLINE=TRUE\r\n\f"},
	} {
		answer := pjlQuery(t, host, query.request)
		t.Logf("%s:9100 answered %q", host, answer)
		if answer != query.want {
			t.Errorf("%q answered %q, want %q", query.request, answer, query.want)
		}
	}
}

// pjlQuery sends request to host:9100 and reads the answer up to its form
// feed. The pack may still be settling, so the connection is retried.
func pjlQuery(t *testing.T, host, request string) string {
	t.Helper()
	var conn net.Conn
	var err error
	for range 10 {
		if conn, err = net.DialTimeout("tcp4", net.JoinHostPort(host, "9100"), 2*time.Second); err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		t.Fatalf("connecting to %s:9100: %v", host, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err = conn.Write([]byte(request)); err != nil {
		t.Fatal(err)
	}
	answer, err := bufio.NewReader(conn).ReadString('\f')
	if err != nil {
		t.Fatalf("reading the answer to %q: %v (got %q)", strings.TrimSpace(request), err, answer)
	}

	return answer
}
