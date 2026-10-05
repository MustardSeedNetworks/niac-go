package devicecli_test

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"net"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/MustardSeedNetworks/niac-go/internal/devicecli"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/virtualtcp"
)

func TestSSHServerRejectsInvalidCredentials(t *testing.T) {
	_, state := newSession()
	server := newSSHServer(t, state)

	client, err := ssh.Dial("tcp", serveSSH(t, server), sshClientConfig("admin", "wrong"))
	if err == nil {
		_ = client.Close()
		t.Fatal("NewClientConn() accepted invalid credentials")
	}
}

func TestSSHServerMaintainsIsolatedCommandModes(t *testing.T) {
	_, state := newSession()
	server := newSSHServer(t, state)
	first := connectSSH(t, server)
	second := connectSSH(t, server)
	defer first.Close()
	defer second.Close()

	firstInput, firstOutput := openShell(t, first)
	secondInput, secondOutput := openShell(t, second)
	if _, err := firstInput.Write([]byte("enable\n")); err != nil {
		t.Fatalf("Write(enable) error = %v", err)
	}
	readThrough(t, firstOutput, "edge-1#")
	if _, err := secondInput.Write([]byte("show ip route\n")); err != nil {
		t.Fatalf("Write(show) error = %v", err)
	}
	readThrough(t, secondOutput, "edge-1>")
}

func TestSSHServerRunsOverVirtualTCPStream(t *testing.T) {
	_, state := newSession()
	server := newSSHServer(t, state)
	clientStream, serverStream := virtualtcp.Pipe("10.0.0.10:40000", "10.0.0.1:22")
	go func() { _ = server.ServeConn(serverStream) }()
	connection, channels, requests, err := ssh.NewClientConn(
		clientStream, "10.0.0.1:22", sshClientConfig("admin", "test-password"),
	)
	if err != nil {
		t.Fatalf("NewClientConn() error = %v", err)
	}
	client := ssh.NewClient(connection, channels, requests)
	defer client.Close()
	input, output := openShell(t, client)
	if _, err = input.Write([]byte("enable\n")); err != nil {
		t.Fatalf("Write(enable) error = %v", err)
	}
	readThrough(t, output, "edge-1#")
}

func TestSSHServerAcceptsBareCarriageReturn(t *testing.T) {
	_, state := newSession()
	client := connectSSH(t, newSSHServer(t, state))
	defer client.Close()
	input, output := openShell(t, client)
	if _, err := input.Write([]byte("enable\r")); err != nil {
		t.Fatalf("Write(enable) error = %v", err)
	}
	readThrough(t, output, "edge-1#")
}

func TestSSHServerRejectsPTYWithoutTerminalLineDiscipline(t *testing.T) {
	_, state := newSession()
	client := connectSSH(t, newSSHServer(t, state))
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	defer session.Close()

	if err = session.RequestPty("xterm", 80, 24, ssh.TerminalModes{}); err == nil {
		t.Fatal("RequestPty() succeeded without terminal line discipline")
	}
}

func TestSSHServerBoundsChannelsPerConnection(t *testing.T) {
	_, state := newSession()
	client := connectSSH(t, newSSHServer(t, state))
	defer client.Close()
	sessions := make([]*ssh.Session, 0, 16)
	defer func() {
		for _, session := range sessions {
			_ = session.Close()
		}
	}()
	for range 16 {
		session, err := client.NewSession()
		if err != nil {
			t.Fatalf("NewSession() rejected before limit: %v", err)
		}
		sessions = append(sessions, session)
	}
	if session, err := client.NewSession(); err == nil {
		_ = session.Close()
		t.Fatal("NewSession() accepted a channel beyond the limit")
	}
}

func newSSHServer(t *testing.T, state *devicestate.Store) *devicecli.SSHServer {
	t.Helper()
	return newSSHServerWith(t, state, &devicecli.Credentials{Username: "admin", Password: "test-password"}, "")
}

func newSSHServerWith(
	t *testing.T,
	state *devicestate.Store,
	credentials *devicecli.Credentials,
	version string,
) *devicecli.SSHServer {
	t.Helper()
	server, err := devicecli.NewSSHServer(state, credentials, newHostSigner(t), version, allowStaticRoute)
	if err != nil {
		t.Fatalf("NewSSHServer() error = %v", err)
	}
	return server
}

func newHostSigner(t *testing.T) ssh.Signer {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	hostSigner, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatalf("NewSignerFromKey() error = %v", err)
	}
	return hostSigner
}

func connectSSH(t *testing.T, server *devicecli.SSHServer) *ssh.Client {
	t.Helper()
	client, err := ssh.Dial("tcp", serveSSH(t, server), sshClientConfig("admin", "test-password"))
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	return client
}

func serveSSH(t *testing.T, server *devicecli.SSHServer) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	go func() {
		connection, acceptErr := listener.Accept()
		_ = listener.Close()
		if acceptErr == nil {
			_ = server.ServeConn(connection)
		}
	}()
	return listener.Addr().String()
}

func sshClientConfig(username, password string) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: username, Auth: []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

func openShell(t *testing.T, client *ssh.Client) (io.WriteCloser, *bufio.Reader) {
	t.Helper()
	session, err := client.NewSession()
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}
	input, err := session.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe() error = %v", err)
	}
	output, err := session.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() error = %v", err)
	}
	if err = session.Shell(); err != nil {
		t.Fatalf("Shell() error = %v", err)
	}
	reader := bufio.NewReader(output)
	readThrough(t, reader, "edge-1>")
	return input, reader
}

func readThrough(t *testing.T, reader *bufio.Reader, suffix string) {
	t.Helper()
	var output strings.Builder
	for !strings.HasSuffix(output.String(), suffix) {
		value, err := reader.ReadByte()
		if err != nil {
			t.Fatalf("ReadByte() error = %v; output = %q", err, output.String())
		}
		output.WriteByte(value)
	}
}

func TestSSHServerWithoutCredentialsCompletesKeyExchangeAndRefusesLogin(t *testing.T) {
	_, state := newSession()
	server := newSSHServerWith(t, state, nil, "SSH-2.0-Cisco-1.25")

	connection, err := net.Dial("tcp", serveSSH(t, server))
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer connection.Close()
	var hostKeyOffered bool
	_, _, _, err = ssh.NewClientConn(connection, "edge-1:22", &ssh.ClientConfig{
		User: "admin", Auth: []ssh.AuthMethod{ssh.Password("")},
		HostKeyCallback: func(string, net.Addr, ssh.PublicKey) error {
			hostKeyOffered = true
			return nil
		},
		BannerCallback: func(string) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "unable to authenticate") {
		t.Fatalf("NewClientConn() error = %v, want an authentication refusal", err)
	}
	if !hostKeyOffered {
		t.Fatal("login was refused before key exchange presented the host key")
	}
}

func TestSSHServerAnnouncesConfiguredVersion(t *testing.T) {
	for _, test := range []struct{ version, want string }{
		{version: "SSH-2.0-Cisco-1.25", want: "SSH-2.0-Cisco-1.25"},
		{version: "", want: "SSH-2.0-Go"},
	} {
		_, state := newSession()
		connection, err := net.Dial("tcp", serveSSH(t, newSSHServerWith(t, state, nil, test.version)))
		if err != nil {
			t.Fatalf("Dial() error = %v", err)
		}
		line, err := bufio.NewReader(connection).ReadString('\n')
		_ = connection.Close()
		if err != nil {
			t.Fatalf("ReadString() error = %v", err)
		}
		if got := strings.TrimRight(line, "\r\n"); got != test.want {
			t.Fatalf("version line = %q, want %q", got, test.want)
		}
	}
}

func TestNewSSHServerRejectsHalfCredentials(t *testing.T) {
	_, state := newSession()
	for _, credentials := range []devicecli.Credentials{{Username: "admin"}, {Password: "secret"}} {
		if _, err := devicecli.NewSSHServer(
			state, &credentials, newHostSigner(t), "", allowStaticRoute,
		); err == nil {
			t.Fatalf("NewSSHServer(%+v) accepted half credentials", credentials)
		}
	}
}
