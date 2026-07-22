package devicecli

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
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
	state  *devicestate.Store
	config *ssh.ServerConfig
}

// NewSSHServer creates an SSH transport with no default credentials.
func NewSSHServer(state *devicestate.Store, credentials Credentials) (*SSHServer, error) {
	if state == nil {
		return nil, errors.New("device state is required")
	}
	if credentials.Username == "" || credentials.Password == "" {
		return nil, errors.New("SSH credentials are required")
	}
	config := &ssh.ServerConfig{
		PasswordCallback: passwordCallback(credentials),
		MaxAuthTries:     maxSSHAuthAttempts,
	}
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate SSH host key: %w", err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("create SSH signer: %w", err)
	}
	config.AddHostKey(signer)
	return &SSHServer{state: state, config: config}, nil
}

func passwordCallback(credentials Credentials) func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) {
	expectedPassword := sha256.Sum256([]byte(credentials.Password))
	return func(metadata ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		usernameMatch := subtle.ConstantTimeCompare([]byte(metadata.User()), []byte(credentials.Username))
		passwordDigest := sha256.Sum256(password)
		passwordMatch := subtle.ConstantTimeCompare(passwordDigest[:], expectedPassword[:])
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
		default:
			_ = request.Reply(false, nil)
		}
	}
}

func (s *SSHServer) serveShell(channel ssh.Channel) {
	session := NewSession(s.state)
	_, _ = channel.Write([]byte(session.Prompt()))
	scanner := bufio.NewScanner(channel)
	scanner.Buffer(nil, maxSSHCommandBytes)
	scanner.Split(scanCLICommands)
	for scanner.Scan() {
		response := session.Execute(scanner.Text())
		if response.Output != "" {
			_, _ = channel.Write([]byte(strings.ReplaceAll(response.Output, "\n", "\r\n") + "\r\n"))
		}
		if response.Close {
			return
		}
		_, _ = channel.Write([]byte(session.Prompt()))
	}
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
