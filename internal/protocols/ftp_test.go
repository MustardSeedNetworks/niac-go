package protocols

import (
	"net"
	"strings"
	"testing"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const ftpTestBanner = "220 JD FTP Server Ready"

func ftpTestServer() *config.FTPConfig {
	return &config.FTPConfig{Enabled: true, WelcomeBanner: ftpTestBanner, SystemType: "UNIX Type: L8"}
}

func TestFTPAnswer(t *testing.T) {
	t.Parallel()

	address := net.IPv4(10, 20, 30, 45).To4()
	tests := []struct {
		name     string
		stream   string
		want     string
		consumed int
		ended    bool
	}{
		{name: "user", stream: "USER anonymous\r\n", want: "331 User name okay, need password.\r\n", consumed: 16},
		{name: "lower case and bare LF", stream: "syst\n", want: "215 UNIX Type: L8\r\n", consumed: 5},
		{
			name:     "two commands in one segment",
			stream:   "USER anonymous\r\nPASS guest@\r\n",
			want:     "331 User name okay, need password.\r\n230 User logged in, proceed.\r\n",
			consumed: 29,
		},
		{
			name:     "argument with spaces",
			stream:   "RETR my file.txt\r\n",
			want:     "550 my file.txt: No such file or directory.\r\n",
			consumed: 18,
		},
		{
			name:     "passive names the address connected to",
			stream:   "PASV\r\n",
			want:     "227 Entering Passive Mode (10,20,30,45,78,32).\r\n",
			consumed: 6,
		},
		{name: "missing argument", stream: "TYPE\r\n", want: ftpSyntaxErrorResponse, consumed: 6},
		{
			name:     "unknown command",
			stream:   "SITE CHMOD 777 x\r\n",
			want:     "502 Command not implemented.\r\n",
			consumed: 18,
		},
		{name: "empty line is skipped", stream: "\r\nNOOP\r\n", want: "200 NOOP ok.\r\n", consumed: 8},
		{name: "partial line waits", stream: "USER anon"},
		{
			name:     "quit ends the conversation",
			stream:   "QUIT\r\nNOOP\r\n",
			want:     "221 Goodbye.\r\n",
			consumed: 6,
			ended:    true,
		},
		{
			name:     "overlong line ends the conversation",
			stream:   strings.Repeat("A", ftpMaxCommandLine+1),
			want:     "500 Command line too long.\r\n",
			consumed: ftpMaxCommandLine + 1,
			ended:    true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			answer, consumed, ended := ftpAnswer([]byte(test.stream), ftpTestServer(), address)
			if string(answer) != test.want || consumed != test.consumed || ended != test.ended {
				t.Errorf("ftpAnswer(%q) = %q, %d, %v; want %q, %d, %v",
					test.stream, answer, consumed, ended, test.want, test.consumed, test.ended)
			}
		})
	}
}

func TestFTPGreeting(t *testing.T) {
	t.Parallel()

	for _, banner := range []string{ftpTestBanner, ftpTestBanner + "\r\n"} {
		if got := ftpGreeting(banner); got != ftpTestBanner+"\r\n" {
			t.Errorf("ftpGreeting(%q) = %q, want one CRLF", banner, got)
		}
	}
}

func ftpTestNode(ftp *config.FTPConfig) *config.Device {
	return &config.Device{
		Name:        "NTH-PRN-B01-F01-05",
		MACAddress:  net.HardwareAddr{0x00, 0x00, 0xbc, 0x00, 0x00, 0x07},
		IPAddresses: []net.IP{net.ParseIP("10.20.30.45")},
		FTPConfig:   ftp,
	}
}

// The server speaks first: the ACK that completes the handshake gets the
// authored greeting, once, and each command line after it is answered.
func TestTCPHandlerServesFTP(t *testing.T) {
	conversation := newServiceConversation(t, ftpTestNode(ftpTestServer()), TCPPortFTP)
	synACK := conversation.reply(&layers.TCP{SYN: true, Seq: 100})
	if synACK == nil || !synACK.SYN || !synACK.ACK || synACK.Ack != 101 {
		t.Fatalf("SYN answered with %+v, want a SYN-ACK acknowledging 101", synACK)
	}
	greeting := conversation.reply(&layers.TCP{ACK: true, Seq: 101, Ack: synACK.Seq + 1})
	if greeting == nil || string(greeting.Payload) != ftpTestBanner+"\r\n" || greeting.Seq != synACK.Seq+1 {
		t.Fatalf("handshake ACK answered with %+v, want %q", greeting, ftpTestBanner)
	}
	next := synACK.Seq + 1 + uint32(len(greeting.Payload))
	if again := conversation.reply(&layers.TCP{ACK: true, Seq: 101, Ack: next}); again != nil {
		t.Errorf("a second bare ACK answered with %+v, want nothing", again)
	}
	user := conversation.reply(
		&layers.TCP{ACK: true, PSH: true, Seq: 101, Ack: next, Payload: []byte("USER anonymous\r\n")},
	)
	if user == nil || string(user.Payload) != "331 User name okay, need password.\r\n" ||
		user.Seq != next || user.Ack != 117 {
		t.Errorf("USER answered with %+v, want 331 at %d acknowledging 117", user, next)
	}
}

// A device without FTP keeps the closed-port RST, as does one whose FTP is
// switched off.
func TestTCPHandlerFTPClosedWithoutIt(t *testing.T) {
	for _, ftp := range []*config.FTPConfig{nil, {WelcomeBanner: ftpTestBanner}} {
		device := ftpTestNode(ftp)
		if reply := serviceReply(t, device, TCPPortFTP, &layers.TCP{SYN: true, Seq: 500}); reply == nil ||
			!reply.RST {
			t.Errorf("FTP %+v: SYN answered with %+v, want RST", ftp, reply)
		}
	}
}
