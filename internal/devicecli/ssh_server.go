package devicecli

import (
	"bufio"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

const sshMaxChannelsPerConnection = 16

const maxSSHAuthAttempts = 3

const maxSSHCommandBytes = 64 * 1024

// Credentials are explicit login details for one simulated SSH service.
type Credentials struct {
	Username string
	Password string
}

// SSHServer serves isolated command sessions over SSH byte streams.
type SSHServer struct {
	state         *devicestate.Store
	validateRoute RouteValidator
	config        *ssh.ServerConfig
}

// NewSSHServer creates an SSH transport with no default credentials. Nil
// credentials still serve the version exchange and key exchange, so a scanner
// sees the device's banner and host key, but every login is refused. An empty
// version keeps the library's own.
func NewSSHServer(
	state *devicestate.Store,
	credentials *Credentials,
	hostSigner ssh.Signer,
	version string,
	validateRoute RouteValidator,
) (*SSHServer, error) {
	if state == nil {
		return nil, errors.New("device state is required")
	}
	if validateRoute == nil {
		return nil, errors.New("route validator is required")
	}
	if credentials != nil && (credentials.Username == "" || credentials.Password == "") {
		return nil, errors.New("SSH credentials need both a username and a password")
	}
	if hostSigner == nil {
		return nil, errors.New("SSH host signer is required")
	}
	config := &ssh.ServerConfig{
		PasswordCallback: passwordCallback(credentials),
		MaxAuthTries:     maxSSHAuthAttempts,
		ServerVersion:    version,
	}
	config.AddHostKey(hostSigner)
	return &SSHServer{state: state, validateRoute: validateRoute, config: config}, nil
}

func passwordCallback(credentials *Credentials) func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) {
	if credentials == nil {
		return func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) {
			return nil, errors.New("authentication failed")
		}
	}
	expectedPassword := []byte(credentials.Password)
	return func(metadata ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		usernameMatch := subtle.ConstantTimeCompare([]byte(metadata.User()), []byte(credentials.Username))
		passwordMatch := subtle.ConstantTimeCompare(password, expectedPassword)
		if usernameMatch&passwordMatch != 1 {
			return nil, errors.New("authentication failed")
		}
		return nil, nil
	}
}

// ServeConn performs an SSH handshake and serves session channels until disconnect.
func (s *SSHServer) ServeConn(connection net.Conn) error {
	defer connection.Close()
	serverConnection, channels, requests, err := ssh.NewServerConn(connection, s.config)
	if err != nil {
		return fmt.Errorf("SSH handshake: %w", err)
	}
	defer serverConnection.Close()
	go ssh.DiscardRequests(requests)
	activeChannels := make(chan struct{}, sshMaxChannelsPerConnection)
	for channel := range channels {
		if channel.ChannelType() != "session" {
			_ = channel.Reject(ssh.UnknownChannelType, "unsupported channel")
			continue
		}
		select {
		case activeChannels <- struct{}{}:
		default:
			_ = channel.Reject(ssh.ResourceShortage, "too many active channels")
			continue
		}
		stream, channelRequests, acceptErr := channel.Accept()
		if acceptErr != nil {
			<-activeChannels
			continue
		}
		go func() {
			defer func() { <-activeChannels }()
			s.serveChannel(stream, channelRequests)
		}()
	}
	return nil
}

func (s *SSHServer) serveChannel(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()
	for request := range requests {
		switch request.Type {
		case "env":
			_ = request.Reply(true, nil)
		case "pty-req", "window-change":
			_ = request.Reply(false, nil)
		case "shell":
			_ = request.Reply(true, nil)
			s.serveShell(channel)
			return
		case "exec":
			var payload struct{ Command string }
			if err := ssh.Unmarshal(request.Payload, &payload); err != nil {
				_ = request.Reply(false, nil)
				continue
			}
			_ = request.Reply(true, nil)
			s.serveExec(channel, payload.Command)
			return
		default:
			_ = request.Reply(false, nil)
		}
	}
}

func (s *SSHServer) serveShell(channel ssh.Channel) {
	session := NewSession(s.state, s.validateRoute)
	_, _ = channel.Write([]byte(session.Prompt()))
	scanner := bufio.NewScanner(channel)
	scanner.Buffer(nil, maxSSHCommandBytes)
	scanner.Split(scanCLICommands)
	for scanner.Scan() {
		response := session.Execute(scanner.Text())
		if response.Output != "" {
			_, _ = channel.Write([]byte(crlf(response.Output) + "\r\n"))
		}
		if response.Close {
			return
		}
		_, _ = channel.Write([]byte(session.Prompt()))
	}
}

// serveExec answers `ssh device "command"` the way network devices do: the
// command runs at privileged level with no prompt, a `%` error goes to
// stderr, and the exit status tells the client whether it was accepted.
func (s *SSHServer) serveExec(channel ssh.Channel, command string) {
	session := NewSession(s.state, s.validateRoute)
	session.mode = ModePrivileged
	output := session.Execute(command).Output
	var status struct{ Code uint32 }
	destination := io.Writer(channel)
	if strings.HasPrefix(output, "%") {
		status.Code = 1
		destination = channel.Stderr()
	}
	if output != "" {
		_, _ = destination.Write([]byte(crlf(output) + "\r\n"))
	}
	_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(&status))
}

func crlf(output string) string {
	return strings.ReplaceAll(output, "\n", "\r\n")
}

func scanCLICommands(data []byte, atEOF bool) (int, []byte, error) {
	if len(data) > 0 && data[0] == '\n' {
		return 1, nil, nil
	}
	for index, value := range data {
		if value == '\r' || value == '\n' {
			return index + 1, data[:index], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}
