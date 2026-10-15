//go:build linux && integration

package wiretest_test

import (
	"bufio"
	"net"
	"os/exec"
	"strings"
	"testing"
)

// D-NIAC-45: the campus pack's HP printer greets an FTP client with the
// banner HP documents for Jetdirect and answers its login, and curl, a real
// client, reads the greeting.
func TestPackPrinterGreetsFTP(t *testing.T) {
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

	conn := dialTCPRetrying(t, net.JoinHostPort(host, "21"))
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for _, exchange := range []struct{ command, want string }{
		{"", "220 JD FTP Server Ready\r\n"},
		{"USER anonymous\r\n", "331 User name okay, need password.\r\n"},
		{"PASS guest@\r\n", "230 User logged in, proceed.\r\n"},
		{"QUIT\r\n", "221 Goodbye.\r\n"},
	} {
		if exchange.command != "" {
			if _, err := conn.Write([]byte(exchange.command)); err != nil {
				t.Fatal(err)
			}
		}
		line, err := reader.ReadString('\n')
		if err != nil || line != exchange.want {
			t.Fatalf("%q answered %q (%v), want %q", strings.TrimSpace(exchange.command), line, err, exchange.want)
		}
	}

	curl, err := exec.LookPath("curl")
	if err != nil {
		t.Skip("curl is not installed")
	}
	// No data connection is served, so the listing fails after the login; the
	// greeting is what curl must have read.
	output, _ := exec.Command(curl, "-sv", "--max-time", "10", "ftp://"+host+"/").CombinedOutput()
	t.Logf("curl:\n%s", output)
	if !strings.Contains(string(output), "< 220 JD FTP Server Ready") {
		t.Errorf("curl did not read the greeting")
	}
}
