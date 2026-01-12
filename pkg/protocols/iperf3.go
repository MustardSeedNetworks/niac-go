package protocols

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// iPerf3 default port.
const (
	TCPPortIPerf3 = 5201
)

// iPerf3 protocol states.
const (
	iperf3StateInit       = 0
	iperf3StateParamExch  = 1
	iperf3StateCreateFlow = 2
	iperf3StateTestStart  = 3
	iperf3StateTestRun    = 4
	iperf3StateTestEnd    = 5
	iperf3StateExchResult = 6
	iperf3StateDisplayRes = 7
	iperf3StateIPerf3Done = 8
	iperf3StateAccessDeny = -1
	iperf3StateServerErr  = -2
)

// iPerf3 message types (state cookies sent to client).
const (
	iperf3MsgTestStart    = 1
	iperf3MsgTestRunning  = 2
	iperf3MsgResultOK     = 3
	iperf3MsgResultBad    = 4
	iperf3MsgAccessDenied = -1
	iperf3MsgServerBusy   = -2
)

// DefaultIPerf3Config returns default iPerf3 emulation configuration.
func DefaultIPerf3Config() *config.IPerf3Config {
	return &config.IPerf3Config{
		Enabled:           false,
		Port:              TCPPortIPerf3,
		MaxBandwidthMbps:  1000.0, // 1 Gbps
		TypicalLatencyMs:  1.0,    // 1ms
		JitterMs:          0.1,    // 0.1ms
		PacketLossPercent: 0.0,    // No loss
		UploadMbps:        100.0,  // 100 Mbps
		DownloadMbps:      100.0,  // 100 Mbps
	}
}

// IPerf3Session tracks state for an active iPerf3 test session.
type IPerf3Session struct {
	ClientIP     string
	ClientPort   uint16
	State        int
	StartTime    time.Time
	TestParams   *iperf3TestParams
	BytesSent    uint64
	BytesRecv    uint64
	Config       *config.IPerf3Config
	LastActivity time.Time
}

// iperf3TestParams represents the test parameters exchanged in JSON.
type iperf3TestParams struct {
	TCP        bool   `json:"tcp,omitempty"`
	UDP        bool   `json:"udp,omitempty"`
	Omit       int    `json:"omit,omitempty"`
	Time       int    `json:"time,omitempty"`
	Parallel   int    `json:"parallel,omitempty"`
	Length     int    `json:"len,omitempty"`
	Bandwidth  int64  `json:"bandwidth,omitempty"`
	Pacing     int64  `json:"pacing_timer,omitempty"`
	Client     string `json:"client,omitempty"`
	Reverse    bool   `json:"reverse,omitempty"`
	Bidirect   bool   `json:"bidirectional,omitempty"`
	Window     int    `json:"window,omitempty"`
	MSS        int    `json:"MSS,omitempty"`
	NoDelay    bool   `json:"nodelay,omitempty"`
	Version    string `json:"version,omitempty"`
	BlockCount int64  `json:"blockcount,omitempty"`
	TOS        int    `json:"TOS,omitempty"`
	FlowLabel  int    `json:"flowlabel,omitempty"`
	Title      string `json:"title,omitempty"`
	Extra      string `json:"extra_data,omitempty"`
	Cookie     string `json:"cookie,omitempty"`
	NumStreams int    `json:"num_streams,omitempty"`
}

// iperf3ServerResult represents the results sent back to client.
type iperf3ServerResult struct {
	CPU         *iperf3CPUUtil `json:"cpu_util_total,omitempty"`
	SenderTCP   *iperf3TCPInfo `json:"sender_tcp_congestion,omitempty"`
	ReceiverTCP *iperf3TCPInfo `json:"receiver_tcp_congestion,omitempty"`
	Streams     []iperf3Stream `json:"streams,omitempty"`
	SumSent     *iperf3Sum     `json:"sum_sent,omitempty"`
	SumReceived *iperf3Sum     `json:"sum_received,omitempty"`
}

type iperf3CPUUtil struct {
	HostTotal    float64 `json:"host_total"`
	HostUser     float64 `json:"host_user"`
	HostSystem   float64 `json:"host_system"`
	RemoteTotal  float64 `json:"remote_total"`
	RemoteUser   float64 `json:"remote_user"`
	RemoteSystem float64 `json:"remote_system"`
}

type iperf3TCPInfo struct {
	Congestion string `json:"congestion"`
}

type iperf3Stream struct {
	ID          int     `json:"id"`
	Bytes       uint64  `json:"bytes"`
	BPS         float64 `json:"bits_per_second"`
	Retransmits int     `json:"retransmits,omitempty"`
	MaxSndCwnd  int     `json:"max_snd_cwnd,omitempty"`
	MaxSndWnd   int     `json:"max_snd_wnd,omitempty"`
	MaxRtt      int     `json:"max_rtt,omitempty"`
	MinRtt      int     `json:"min_rtt,omitempty"`
	MeanRtt     int     `json:"mean_rtt,omitempty"`
	JitterMs    float64 `json:"jitter_ms,omitempty"`
	LostPackets int     `json:"lost_packets,omitempty"`
	Packets     int     `json:"packets,omitempty"`
	LostPercent float64 `json:"lost_percent,omitempty"`
	Sender      bool    `json:"sender"`
}

type iperf3Sum struct {
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
	Seconds     float64 `json:"seconds"`
	Bytes       uint64  `json:"bytes"`
	BPS         float64 `json:"bits_per_second"`
	Retransmits int     `json:"retransmits,omitempty"`
	Sender      bool    `json:"sender"`
}

// IPerf3Handler handles iPerf3 protocol emulation.
type IPerf3Handler struct {
	stack    *Stack
	sessions map[string]*IPerf3Session // keyed by "ip:port"
}

// NewIPerf3Handler creates a new iPerf3 handler.
func NewIPerf3Handler(stack *Stack) *IPerf3Handler {
	return &IPerf3Handler{
		stack:    stack,
		sessions: make(map[string]*IPerf3Session),
	}
}

// getOrCreateSession gets or creates an iPerf3 session.
func (h *IPerf3Handler) getOrCreateSession(
	clientIP string,
	clientPort uint16,
	cfg *config.IPerf3Config,
) *IPerf3Session {
	key := fmt.Sprintf("%s:%d", clientIP, clientPort)

	if session, exists := h.sessions[key]; exists {
		session.LastActivity = time.Now()

		return session
	}

	session := &IPerf3Session{
		ClientIP:     clientIP,
		ClientPort:   clientPort,
		State:        iperf3StateInit,
		StartTime:    time.Now(),
		Config:       cfg,
		LastActivity: time.Now(),
	}
	h.sessions[key] = session

	return session
}

// cleanupStaleSessions removes sessions that haven't had activity.
func (h *IPerf3Handler) cleanupStaleSessions(maxAge time.Duration) {
	now := time.Now()
	for key, session := range h.sessions {
		if now.Sub(session.LastActivity) > maxAge {
			delete(h.sessions, key)
		}
	}
}

// HandleIPerf3Request handles incoming iPerf3 traffic.
func (h *IPerf3Handler) HandleIPerf3Request(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	// Get iPerf3 config from device (use defaults if not configured)
	var cfg *config.IPerf3Config
	if len(devices) > 0 && devices[0].IPerf3 != nil {
		cfg = devices[0].IPerf3
	} else {
		cfg = DefaultIPerf3Config()
		cfg.Enabled = true // Enable by default for testing
	}

	if !cfg.Enabled {
		if debugLevel >= 2 {
			_, _ = fmt.Fprintf(os.Stdout, "iPerf3 disabled for device, ignoring request\n")
		}

		return
	}

	// Handle TCP SYN - initial connection
	if tcpLayer.SYN && !tcpLayer.ACK {
		if debugLevel >= 2 {
			_, _ = fmt.Fprintf(os.Stdout, "iPerf3: New connection from %s:%d\n", ipLayer.SrcIP, tcpLayer.SrcPort)
		}

		h.sendSYNACK(ipLayer, tcpLayer, devices)
		h.getOrCreateSession(ipLayer.SrcIP.String(), uint16(tcpLayer.SrcPort), cfg)

		return
	}

	// Handle data
	if len(tcpLayer.Payload) > 0 {
		session := h.getOrCreateSession(ipLayer.SrcIP.String(), uint16(tcpLayer.SrcPort), cfg)
		h.handleIPerf3Data(pkt, ipLayer, tcpLayer, devices, session)
	}

	// Periodic cleanup
	h.cleanupStaleSessions(5 * time.Minute)
}

// handleIPerf3Data processes iPerf3 protocol messages.
func (h *IPerf3Handler) handleIPerf3Data(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
	session *IPerf3Session,
) {
	debugLevel := h.stack.GetDebugLevel()
	payload := tcpLayer.Payload

	if debugLevel >= 3 {
		_, _ = fmt.Fprintf(os.Stdout, "iPerf3: Received %d bytes from %s:%d in state %d\n",
			len(payload), ipLayer.SrcIP, tcpLayer.SrcPort, session.State)
	}

	// iPerf3 control channel uses a simple protocol:
	// - First 4 bytes: message length (big endian)
	// - Followed by JSON payload
	// - Or single byte state codes

	switch session.State {
	case iperf3StateInit:
		// Expecting cookie (32 bytes) or parameters
		if len(payload) >= 4 {
			// Check if this looks like JSON parameters
			if payload[0] == '{' || (len(payload) > 4 && payload[4] == '{') {
				h.handleParamExchange(ipLayer, tcpLayer, devices, session, payload)
			} else if len(payload) >= 32 {
				// Cookie received, acknowledge and move to param exchange
				session.State = iperf3StateParamExch

				h.sendStateCode(ipLayer, tcpLayer, devices, iperf3MsgTestStart)
			}
		}

	case iperf3StateParamExch:
		// Parse test parameters JSON
		h.handleParamExchange(ipLayer, tcpLayer, devices, session, payload)

	case iperf3StateCreateFlow:
		// Data streams being created
		session.State = iperf3StateTestStart

		h.sendStateCode(ipLayer, tcpLayer, devices, iperf3MsgTestRunning)

	case iperf3StateTestStart, iperf3StateTestRun:
		// Test is running - count bytes received
		session.BytesRecv += uint64(len(payload))
		session.State = iperf3StateTestRun

		// If test duration elapsed, move to results
		if session.TestParams != nil && session.TestParams.Time > 0 {
			elapsed := time.Since(session.StartTime).Seconds()
			if elapsed >= float64(session.TestParams.Time) {
				session.State = iperf3StateExchResult
				h.sendResults(ipLayer, tcpLayer, devices, session)
			}
		}

	case iperf3StateExchResult:
		// Exchange results phase
		if debugLevel >= 2 {
			_, _ = fmt.Fprintf(os.Stdout, "iPerf3: Test complete, sending results\n")
		}

		h.sendResults(ipLayer, tcpLayer, devices, session)
		session.State = iperf3StateIPerf3Done

	case iperf3StateIPerf3Done:
		// Test complete, clean up session
		key := fmt.Sprintf("%s:%d", session.ClientIP, session.ClientPort)
		delete(h.sessions, key)
	}
}

// handleParamExchange parses and acknowledges test parameters.
func (h *IPerf3Handler) handleParamExchange(
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
	session *IPerf3Session,
	payload []byte,
) {
	debugLevel := h.stack.GetDebugLevel()

	// Try to parse as JSON
	var jsonData []byte

	// Check for length prefix
	if len(payload) > 4 && payload[0] != '{' {
		msgLen := binary.BigEndian.Uint32(payload[:4])
		if int(msgLen) <= len(payload)-4 {
			jsonData = payload[4 : 4+msgLen]
		}
	} else {
		// Try without length prefix
		jsonData = payload
	}

	// Handle single-byte state messages
	if len(payload) == 1 {
		switch int8(payload[0]) {
		case iperf3MsgTestStart:
			session.State = iperf3StateTestStart
			session.StartTime = time.Now()

			return
		}
	}

	var params iperf3TestParams
	err := json.Unmarshal(jsonData, &params)
	if err != nil {
		if debugLevel >= 2 {
			_, _ = fmt.Fprintf(os.Stdout, "iPerf3: Failed to parse params: %v (data: %s)\n", err, string(jsonData))
		}
		// Send acknowledgment anyway
		h.sendStateCode(ipLayer, tcpLayer, devices, iperf3MsgTestStart)

		session.State = iperf3StateCreateFlow

		return
	}

	session.TestParams = &params

	if params.Time == 0 {
		params.Time = 10 // Default 10 second test
	}

	if debugLevel >= 2 {
		_, _ = fmt.Fprintf(os.Stdout, "iPerf3: Test params - Duration: %ds, Parallel: %d, TCP: %v, Reverse: %v\n",
			params.Time, params.Parallel, params.TCP, params.Reverse)
	}

	session.State = iperf3StateCreateFlow
	session.StartTime = time.Now()

	// Send test start acknowledgment
	h.sendStateCode(ipLayer, tcpLayer, devices, iperf3MsgTestStart)
}

// sendStateCode sends a single-byte state code to the client.
func (h *IPerf3Handler) sendStateCode(
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
	stateCode int8,
) {
	response := []byte{byte(stateCode)}
	h.sendTCPResponse(ipLayer, tcpLayer, response, devices)
}

// sendResults sends simulated test results to the client.
func (h *IPerf3Handler) sendResults(
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
	session *IPerf3Session,
) {
	cfg := session.Config

	// Calculate simulated results based on device config
	duration := time.Since(session.StartTime).Seconds()
	if duration < 1 {
		duration = 1
	}

	// Use configured bandwidth or calculate from bytes received
	downloadBps := cfg.DownloadMbps * 1_000_000
	uploadBps := cfg.UploadMbps * 1_000_000

	// Add some realistic variance (±5%)
	variance := 0.95 + rand.Float64()*0.1 // #nosec G404 -- simulation variance, not security-sensitive
	downloadBps *= variance
	uploadBps *= variance

	downloadBytes := uint64(downloadBps / 8 * duration)
	uploadBytes := uint64(uploadBps / 8 * duration)

	// Build result JSON
	// #nosec G404 -- all rand usage below is for simulation variance, not security-sensitive
	result := &iperf3ServerResult{
		CPU: &iperf3CPUUtil{
			HostTotal:    2.5 + rand.Float64()*2,
			HostUser:     1.5 + rand.Float64()*1,
			HostSystem:   0.5 + rand.Float64()*0.5,
			RemoteTotal:  3.0 + rand.Float64()*2,
			RemoteUser:   2.0 + rand.Float64()*1,
			RemoteSystem: 0.5 + rand.Float64()*0.5,
		},
		SenderTCP: &iperf3TCPInfo{
			Congestion: "cubic",
		},
		ReceiverTCP: &iperf3TCPInfo{
			Congestion: "cubic",
		},
		Streams: []iperf3Stream{
			{
				ID:          1,
				Bytes:       downloadBytes,
				BPS:         downloadBps,
				Retransmits: int(cfg.PacketLossPercent * 100),
				MaxRtt:      int(cfg.TypicalLatencyMs * 1000),
				MinRtt:      int(cfg.TypicalLatencyMs * 800),
				MeanRtt:     int(cfg.TypicalLatencyMs * 900),
				Sender:      false,
			},
		},
		SumReceived: &iperf3Sum{
			Start:   0,
			End:     duration,
			Seconds: duration,
			Bytes:   downloadBytes,
			BPS:     downloadBps,
			Sender:  false,
		},
		SumSent: &iperf3Sum{
			Start:       0,
			End:         duration,
			Seconds:     duration,
			Bytes:       uploadBytes,
			BPS:         uploadBps,
			Retransmits: int(cfg.PacketLossPercent * 10),
			Sender:      true,
		},
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return
	}

	// Send with length prefix
	response := make([]byte, 4+len(jsonData))
	binary.BigEndian.PutUint32(
		response[:4],
		uint32(len(jsonData)),
	)
	copy(response[4:], jsonData)

	h.sendTCPResponse(ipLayer, tcpLayer, response, devices)

	// Send result OK state
	h.sendStateCode(ipLayer, tcpLayer, devices, iperf3MsgResultOK)
}

// sendSYNACK sends a TCP SYN-ACK response.
func (h *IPerf3Handler) sendSYNACK(ipLayer *layers.IPv4, tcpLayer *layers.TCP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	if len(devices) == 0 {
		return
	}

	device := devices[0]
	if len(device.MACAddress) == 0 {
		return
	}

	// Get source MAC
	srcDevices := h.stack.GetDevices().GetByIP(ipLayer.SrcIP)

	var srcMAC []byte

	if len(srcDevices) > 0 && len(srcDevices[0].MACAddress) > 0 {
		srcMAC = srcDevices[0].MACAddress
	} else {
		if debugLevel >= 2 {
			_, _ = fmt.Fprintf(os.Stdout, "iPerf3: Cannot send SYN-ACK: no MAC for %s\n", ipLayer.SrcIP)
		}

		return
	}

	// Build Ethernet header
	eth := &layers.Ethernet{
		SrcMAC:       device.MACAddress,
		DstMAC:       srcMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	// Build IP header
	ipReply := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    ipLayer.DstIP,
		DstIP:    ipLayer.SrcIP,
	}

	// Build TCP SYN-ACK
	tcpReply := &layers.TCP{
		SrcPort: tcpLayer.DstPort,
		DstPort: tcpLayer.SrcPort,
		Seq:     1000,
		Ack:     tcpLayer.Seq + 1,
		SYN:     true,
		ACK:     true,
		Window:  65535,
	}
	_ = tcpReply.SetNetworkLayerForChecksum(ipReply) // error is non-critical for simulation

	// Serialize
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipReply, tcpReply)
	if err != nil {
		if debugLevel >= 2 {
			_, _ = fmt.Fprintf(os.Stdout, "iPerf3: Error serializing SYN-ACK: %v\n", err)
		}

		return
	}

	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	pktOut := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: serialNum,
		Device:       device,
	}

	h.stack.Send(pktOut)

	if debugLevel >= 3 {
		_, _ = fmt.Fprintf(os.Stdout, "iPerf3: Sent TCP SYN-ACK from %s:%d to %s:%d\n",
			ipReply.SrcIP, tcpReply.SrcPort, ipReply.DstIP, tcpReply.DstPort)
	}
}

// sendTCPResponse sends a TCP response with payload.
func (h *IPerf3Handler) sendTCPResponse(
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	payload []byte,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(devices) == 0 || len(payload) == 0 {
		return
	}

	device := devices[0]
	if len(device.MACAddress) == 0 {
		return
	}

	srcDevices := h.stack.GetDevices().GetByIP(ipLayer.SrcIP)

	var srcMAC []byte

	if len(srcDevices) > 0 && len(srcDevices[0].MACAddress) > 0 {
		srcMAC = srcDevices[0].MACAddress
	} else {
		return
	}

	eth := &layers.Ethernet{
		SrcMAC:       device.MACAddress,
		DstMAC:       srcMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	ipReply := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    ipLayer.DstIP,
		DstIP:    ipLayer.SrcIP,
	}

	payloadLen := min(len(tcpLayer.Payload), 0xFFFFFFFF)

	tcpReply := &layers.TCP{
		SrcPort: tcpLayer.DstPort,
		DstPort: tcpLayer.SrcPort,
		Seq:     tcpLayer.Ack,
		Ack: tcpLayer.Seq + uint32(
			payloadLen,
		),
		PSH:    true,
		ACK:    true,
		Window: 65535,
	}
	_ = tcpReply.SetNetworkLayerForChecksum(ipReply) // error is non-critical for simulation

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipReply, tcpReply, gopacket.Payload(payload))
	if err != nil {
		if debugLevel >= 2 {
			_, _ = fmt.Fprintf(os.Stdout, "iPerf3: Error serializing TCP response: %v\n", err)
		}

		return
	}

	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	pktOut := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: serialNum,
		Device:       device,
	}

	h.stack.Send(pktOut)

	if debugLevel >= 3 {
		_, _ = fmt.Fprintf(os.Stdout, "iPerf3: Sent %d bytes to %s:%d\n", len(payload), ipLayer.DstIP, tcpLayer.SrcPort)
	}
}

// GetDeviceIPerf3Config extracts iPerf3 config from device name for display.
func GetDeviceIPerf3Config(device *config.Device) string {
	if device.IPerf3 == nil || !device.IPerf3.Enabled {
		return "disabled"
	}

	cfg := device.IPerf3

	return fmt.Sprintf("Up: %.0f Mbps, Down: %.0f Mbps, Latency: %.1fms",
		cfg.UploadMbps, cfg.DownloadMbps, cfg.TypicalLatencyMs)
}

// Helper to avoid unused import warning.
var _ = strings.TrimSpace
