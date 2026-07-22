package protocols

import (
	"bufio"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"golang.org/x/crypto/ssh"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestSSHAuthenticatesAndRunsCLIThroughVirtualPacketStack(t *testing.T) {
	t.Setenv("NIAC_TEST_SSH_PASSWORD", "test-password")
	deviceMAC := mustForwardingMAC(t, "02:00:00:00:00:01")
	clientMAC := mustForwardingMAC(t, "02:00:00:00:00:fe")
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1", Type: "router", MACAddress: deviceMAC,
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
		SSHConfig: &config.SSHConfig{
			Enabled: true, Username: "admin", PasswordEnv: "NIAC_TEST_SSH_PASSWORD",
		},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	useTemporarySSHHostKeys(t, stack)
	stream := newPacketSSHClient(t, stack, clientMAC, deviceMAC)
	connection, channels, requests, err := ssh.NewClientConn(stream, "10.0.0.1:22", &ssh.ClientConfig{
		User: "admin", Auth: []ssh.AuthMethod{ssh.Password("test-password")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		t.Fatalf("NewClientConn() error = %v", err)
	}
	client := ssh.NewClient(connection, channels, requests)
	defer client.Close()
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
	readThroughPrompt(t, reader, "edge-1>")
	if _, err = input.Write([]byte("enable\n")); err != nil {
		t.Fatalf("Write(enable) error = %v", err)
	}
	readThroughPrompt(t, reader, "edge-1#")
}

type packetSSHClient struct {
	stack                  *Stack
	clientMAC, deviceMAC   net.HardwareAddr
	clientNext, serverNext uint32
	mu                     sync.Mutex
	pending                []byte
	done                   chan struct{}
}

func newPacketSSHClient(
	t *testing.T,
	stack *Stack,
	clientMAC, deviceMAC net.HardwareAddr,
) *packetSSHClient {
	t.Helper()
	client := &packetSSHClient{
		stack: stack, clientMAC: clientMAC, deviceMAC: deviceMAC,
		clientNext: 101, done: make(chan struct{}),
	}
	client.send(t, 100, 0, true, false, nil)
	reply := client.receive(t)
	if !reply.SYN || !reply.ACK || reply.Ack != client.clientNext {
		t.Fatalf("SYN-ACK = %#v", reply)
	}
	client.serverNext = reply.Seq + 1
	client.send(t, client.clientNext, client.serverNext, false, true, nil)
	return client
}

func (c *packetSSHClient) Read(destination []byte) (int, error) {
	for len(c.pending) == 0 {
		select {
		case packet := <-c.stack.sendQueue:
			decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
			tcp, ok := decoded.Layer(layers.LayerTypeTCP).(*layers.TCP)
			if !ok {
				continue
			}
			if tcp.FIN {
				return 0, io.EOF
			}
			if len(tcp.Payload) == 0 {
				continue
			}
			c.mu.Lock()
			if tcp.Seq == c.serverNext {
				c.serverNext += uint32(len(tcp.Payload))
				c.pending = append(c.pending, tcp.Payload...)
			}
			sequence, acknowledgment := c.clientNext, c.serverNext
			c.mu.Unlock()
			c.sendPacket(sequence, acknowledgment, false, true, nil)
		case <-c.done:
			return 0, io.EOF
		}
	}
	count := copy(destination, c.pending)
	c.pending = c.pending[count:]
	return count, nil
}

func (c *packetSSHClient) Write(payload []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	written := 0
	for len(payload) > 0 {
		length := min(len(payload), sshSegmentPayloadLimit)
		c.sendPacket(c.clientNext, c.serverNext, false, true, payload[:length])
		c.clientNext += uint32(length)
		written += length
		payload = payload[length:]
	}
	return written, nil
}

func (c *packetSSHClient) Close() error {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	return nil
}

func (c *packetSSHClient) LocalAddr() net.Addr              { return testTCPAddr("10.0.0.10:40000") }
func (c *packetSSHClient) RemoteAddr() net.Addr             { return testTCPAddr("10.0.0.1:22") }
func (c *packetSSHClient) SetDeadline(time.Time) error      { return nil }
func (c *packetSSHClient) SetReadDeadline(time.Time) error  { return nil }
func (c *packetSSHClient) SetWriteDeadline(time.Time) error { return nil }

func (c *packetSSHClient) send(
	t *testing.T,
	sequence, acknowledgment uint32,
	syn, ack bool,
	payload []byte,
) {
	t.Helper()
	if err := c.sendPacket(sequence, acknowledgment, syn, ack, payload); err != nil {
		t.Fatalf("sendPacket() error = %v", err)
	}
}

func (c *packetSSHClient) sendPacket(
	sequence, acknowledgment uint32,
	syn, ack bool,
	payload []byte,
) error {
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolTCP,
		SrcIP: net.ParseIP("10.0.0.10"), DstIP: net.ParseIP("10.0.0.1"),
	}
	tcp := &layers.TCP{
		SrcPort: 40000, DstPort: TCPPortSSH, Seq: sequence, Ack: acknowledgment,
		SYN: syn, ACK: ack, PSH: len(payload) > 0, Window: tcpWindowSize,
	}
	if err := tcp.SetNetworkLayerForChecksum(ip); err != nil {
		return err
	}
	buffer := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(
		buffer, gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		&layers.Ethernet{SrcMAC: c.clientMAC, DstMAC: c.deviceMAC, EthernetType: layers.EthernetTypeIPv4},
		ip, tcp, gopacket.Payload(payload),
	); err != nil {
		return err
	}
	packet, err := ParsePacket(buffer.Bytes(), 1)
	if err != nil {
		return err
	}
	c.stack.ipHandler.HandlePacket(packet)
	return nil
}

func (c *packetSSHClient) receive(t *testing.T) *layers.TCP {
	t.Helper()
	select {
	case packet := <-c.stack.sendQueue:
		decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		tcp, ok := decoded.Layer(layers.LayerTypeTCP).(*layers.TCP)
		if !ok {
			t.Fatal("reply has no TCP layer")
		}
		return tcp
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for TCP reply")
		return nil
	}
}

type testTCPAddr string

func (a testTCPAddr) Network() string { return "tcp" }
func (a testTCPAddr) String() string  { return string(a) }

func readThroughPrompt(t *testing.T, reader *bufio.Reader, suffix string) {
	t.Helper()
	buffer := make([]byte, 0, len(suffix))
	for len(buffer) < len(suffix) || string(buffer[len(buffer)-len(suffix):]) != suffix {
		value, err := reader.ReadByte()
		if err != nil {
			t.Fatalf("ReadByte() error = %v", err)
		}
		buffer = append(buffer, value)
	}
}
