package protocols

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// iPerf3 default port.
const (
	TCPPortIPerf3 = 5201
)

// iperf3Disabled is the status string returned when iPerf3 is not enabled for a device.
const iperf3Disabled = "disabled"

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

// iPerf3 protocol constants.
const (
	iperf3SessionCleanupMin = 5          // Session cleanup interval in minutes
	iperf3RetransmitMult    = 100        // Retransmit percentage multiplier
	iperf3RTTMicrosPerMilli = 1000       // Microseconds per millisecond for RTT
	iperf3MinRTTPercent     = 800        // Min RTT as percentage (80% = 800/1000)
	iperf3MeanRTTPercent    = 900        // Mean RTT as percentage (90% = 900/1000)
	iperf3SenderRetransMult = 10         // Sender retransmit factor
	iperf3LengthPrefixSize  = 4          // Length prefix size (uint32)
	iperf3MaxUint32         = 0xFFFFFFFF // Max uint32 value
	iperf3MinJSONCheckSize  = 4          // Minimum size to check for JSON start
	iperf3CookieSize        = 32         // iPerf3 cookie size in bytes
	iperf3BitsPerMbps       = 1_000_000  // Bits per Mbps for bandwidth conversion
	iperf3VarianceRange     = 0.1        // Bandwidth variance range (±5%)
)

// iPerf3 default configuration values.
const (
	iperf3DefaultMaxBandwidth  = 1000.0 // 1 Gbps default max bandwidth
	iperf3DefaultLatencyMs     = 1.0    // 1ms default latency
	iperf3DefaultJitterMs      = 0.1    // 0.1ms default jitter
	iperf3DefaultUploadMbps    = 100.0  // 100 Mbps upload
	iperf3DefaultDownloadMbps  = 100.0  // 100 Mbps download
	iperf3BandwidthThreshold   = 0.95   // 95% of configured bandwidth
	iperf3LowBandwidthUpload   = 2.5    // Low bandwidth upload range max
	iperf3LowBandwidthDownload = 1.5    // Low bandwidth download range max
	iperf3MedBandwidthUpload   = 0.5    // Medium bandwidth upload range min
	iperf3MedBandwidthJitter   = 3.0    // Medium bandwidth jitter
	iperf3HighBandwidthLatency = 2.0    // High bandwidth latency
	iperf3HighBandwidthJitter  = 0.5    // High bandwidth jitter
)

// IP protocol constants for iPerf3.
const (
	iperf3IPv4Version   = 4     // IPv4 version field
	iperf3IPv4IHL       = 5     // IPv4 header length (5 = 20 bytes)
	iperf3IPv4TTL       = 64    // IPv4 default TTL
	iperf3TCPWindowSize = 65535 // Default TCP window size
	iperf3TCPInitialSeq = 1000  // Initial TCP sequence number
)

// DefaultIPerf3Config returns default iPerf3 emulation configuration.
func DefaultIPerf3Config() *config.IPerf3Config {
	return &config.IPerf3Config{
		Enabled:           false,
		Port:              TCPPortIPerf3,
		MaxBandwidthMbps:  iperf3DefaultMaxBandwidth, // 1 Gbps
		TypicalLatencyMs:  iperf3DefaultLatencyMs,    // 1ms
		JitterMs:          iperf3DefaultJitterMs,     // 0.1ms
		PacketLossPercent: 0.0,                       // No loss
		UploadMbps:        iperf3DefaultUploadMbps,   // 100 Mbps
		DownloadMbps:      iperf3DefaultDownloadMbps, // 100 Mbps
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
	stack       *Stack
	sessions    map[string]*IPerf3Session // keyed by "ip:port"
	lastCleanup time.Time                 // throttles the stale-session sweep
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
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "iPerf3 disabled for device, ignoring request\n")
		}

		return
	}

	// Handle TCP SYN - initial connection
	if tcpLayer.SYN && !tcpLayer.ACK {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "iPerf3: New connection from %s:%d\n", ipLayer.SrcIP, tcpLayer.SrcPort)
		}

		h.sendSYNACK(ipLayer, tcpLayer, devices, pkt.VLAN)
		h.getOrCreateSession(ipLayer.SrcIP.String(), uint16(tcpLayer.SrcPort), cfg)

		return
	}

	// Handle data
	if len(tcpLayer.Payload) > 0 {
		session := h.getOrCreateSession(ipLayer.SrcIP.String(), uint16(tcpLayer.SrcPort), cfg)
		h.handleIPerf3Data(pkt, ipLayer, tcpLayer, devices, session)
	}

	// Periodic cleanup — throttled so an iperf3 test (thousands of data-phase
	// packets) doesn't rescan the whole session map on every packet.
	maxAge := iperf3SessionCleanupMin * time.Minute
	if now := time.Now(); now.Sub(h.lastCleanup) >= maxAge {
		h.lastCleanup = now
		h.cleanupStaleSessions(maxAge)
	}
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

	if debugLevel >= DebugLevelVerbose {
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
		if len(payload) >= iperf3MinJSONCheckSize {
			// Check if this looks like JSON parameters
			if payload[0] == '{' || (len(payload) > iperf3MinJSONCheckSize && payload[iperf3MinJSONCheckSize] == '{') {
				h.handleParamExchange(ipLayer, tcpLayer, devices, session, payload, pkt.VLAN)
			} else if len(payload) >= iperf3CookieSize {
				// Cookie received, acknowledge and move to param exchange
				session.State = iperf3StateParamExch

				h.sendStateCode(ipLayer, tcpLayer, devices, iperf3MsgTestStart, pkt.VLAN)
			}
		}

	case iperf3StateParamExch:
		// Parse test parameters JSON
		h.handleParamExchange(ipLayer, tcpLayer, devices, session, payload, pkt.VLAN)

	case iperf3StateCreateFlow:
		// Data streams being created
		session.State = iperf3StateTestStart

		h.sendStateCode(ipLayer, tcpLayer, devices, iperf3MsgTestRunning, pkt.VLAN)

	case iperf3StateTestStart, iperf3StateTestRun:
		// Test is running - count bytes received
		session.BytesRecv += uint64(len(payload))
		session.State = iperf3StateTestRun

		// If test duration elapsed, move to results
		if session.TestParams != nil && session.TestParams.Time > 0 {
			elapsed := time.Since(session.StartTime).Seconds()
			if elapsed >= float64(session.TestParams.Time) {
				session.State = iperf3StateExchResult
				h.sendResults(ipLayer, tcpLayer, devices, session, pkt.VLAN)
			}
		}

	case iperf3StateExchResult:
		// Exchange results phase
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "iPerf3: Test complete, sending results\n")
		}

		h.sendResults(ipLayer, tcpLayer, devices, session, pkt.VLAN)
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
	vlan int,
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
	if len(payload) == 1 && safeconv.Int8FromByte(payload[0]) == iperf3MsgTestStart {
		session.State = iperf3StateTestStart
		session.StartTime = time.Now()

		return
	}

	var params iperf3TestParams
	err := json.Unmarshal(jsonData, &params)
	if err != nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "iPerf3: Failed to parse params: %v (data: %s)\n", err, string(jsonData))
		}
		// Send acknowledgment anyway
		h.sendStateCode(ipLayer, tcpLayer, devices, iperf3MsgTestStart, vlan)

		session.State = iperf3StateCreateFlow

		return
	}

	session.TestParams = &params

	if params.Time == 0 {
		params.Time = 10 // Default 10 second test
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "iPerf3: Test params - Duration: %ds, Parallel: %d, TCP: %v, Reverse: %v\n",
			params.Time, params.Parallel, params.TCP, params.Reverse)
	}

	session.State = iperf3StateCreateFlow
	session.StartTime = time.Now()

	// Send test start acknowledgment
	h.sendStateCode(ipLayer, tcpLayer, devices, iperf3MsgTestStart, vlan)
}

// sendStateCode sends a single-byte state code to the client.
func (h *IPerf3Handler) sendStateCode(
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
	stateCode int8,
	vlan int,
) {
	response := []byte{safeconv.ByteFromInt8(stateCode)}
	h.sendTCPResponse(ipLayer, tcpLayer, response, devices, vlan)
}

// sendResults sends simulated test results to the client.
func (h *IPerf3Handler) sendResults(
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	devices []*config.Device,
	session *IPerf3Session,
	vlan int,
) {
	cfg := session.Config

	// Calculate simulated results based on device config
	duration := time.Since(session.StartTime).Seconds()
	if duration < 1 {
		duration = 1
	}

	// Use configured bandwidth or calculate from bytes received
	downloadBps := cfg.DownloadMbps * iperf3BitsPerMbps
	uploadBps := cfg.UploadMbps * iperf3BitsPerMbps

	// Add some realistic variance (±5%)
	variance := iperf3BandwidthThreshold + getSimRand().Float64()*iperf3VarianceRange
	downloadBps *= variance
	uploadBps *= variance

	downloadBytes := uint64(downloadBps / 8 * duration)
	uploadBytes := uint64(uploadBps / 8 * duration)

	// Build result JSON
	result := &iperf3ServerResult{
		CPU: &iperf3CPUUtil{
			HostTotal:    iperf3LowBandwidthUpload + getSimRand().Float64()*iperf3HighBandwidthLatency,
			HostUser:     iperf3LowBandwidthDownload + getSimRand().Float64()*iperf3DefaultLatencyMs,
			HostSystem:   iperf3MedBandwidthUpload + getSimRand().Float64()*iperf3HighBandwidthJitter,
			RemoteTotal:  iperf3MedBandwidthJitter + getSimRand().Float64()*iperf3HighBandwidthLatency,
			RemoteUser:   iperf3HighBandwidthLatency + getSimRand().Float64()*iperf3DefaultLatencyMs,
			RemoteSystem: iperf3HighBandwidthJitter + getSimRand().Float64()*iperf3HighBandwidthJitter,
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
				Retransmits: int(cfg.PacketLossPercent * iperf3RetransmitMult),
				MaxRtt:      int(cfg.TypicalLatencyMs * iperf3RTTMicrosPerMilli),
				MinRtt:      int(cfg.TypicalLatencyMs * iperf3MinRTTPercent),
				MeanRtt:     int(cfg.TypicalLatencyMs * iperf3MeanRTTPercent),
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
			Retransmits: int(cfg.PacketLossPercent * iperf3SenderRetransMult),
			Sender:      true,
		},
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return
	}

	// Bound the response payload so the length prefix can't be coerced into
	// an oversized allocation; iperf3 results are small in practice.
	const maxIperf3JSONLen = 1 << 20 // 1 MiB
	if len(jsonData) > maxIperf3JSONLen {
		return
	}

	// Send with length prefix
	// Safe conversion: JSON data length is bounded by maxIperf3JSONLen above.
	response := make([]byte, iperf3LengthPrefixSize+len(jsonData))
	binary.BigEndian.PutUint32(
		response[:4],
		safeconv.Uint32(len(jsonData)),
	)
	copy(response[4:], jsonData)

	h.sendTCPResponse(ipLayer, tcpLayer, response, devices, vlan)

	// Send result OK state
	h.sendStateCode(ipLayer, tcpLayer, devices, iperf3MsgResultOK, vlan)
}

// sendSYNACK sends a TCP SYN-ACK response.
func (h *IPerf3Handler) sendSYNACK(ipLayer *layers.IPv4, tcpLayer *layers.TCP, devices []*config.Device, vlan int) {
	debugLevel := h.stack.GetDebugLevel()

	if len(devices) == 0 {
		return
	}

	device := devices[0]
	if len(device.MACAddress) == 0 {
		return
	}

	// Get source MAC (scoped to the request's VLAN segment)
	srcDevices := h.stack.devicesFor(vlan).GetByIP(ipLayer.SrcIP)

	var srcMAC []byte

	if len(srcDevices) > 0 && len(srcDevices[0].MACAddress) > 0 {
		srcMAC = srcDevices[0].MACAddress
	} else {
		if debugLevel >= DebugLevelInfo {
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
		Version:  iperf3IPv4Version,
		IHL:      iperf3IPv4IHL,
		TTL:      iperf3IPv4TTL,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    ipLayer.DstIP,
		DstIP:    ipLayer.SrcIP,
	}

	// Build TCP SYN-ACK
	tcpReply := &layers.TCP{
		SrcPort: tcpLayer.DstPort,
		DstPort: tcpLayer.SrcPort,
		Seq:     iperf3TCPInitialSeq,
		Ack:     tcpLayer.Seq + 1,
		SYN:     true,
		ACK:     true,
		Window:  iperf3TCPWindowSize,
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
		if debugLevel >= DebugLevelInfo {
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

	if debugLevel >= DebugLevelVerbose {
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
	vlan int,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(devices) == 0 || len(payload) == 0 {
		return
	}

	device := devices[0]
	if len(device.MACAddress) == 0 {
		return
	}

	srcDevices := h.stack.devicesFor(vlan).GetByIP(ipLayer.SrcIP)

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
		Version:  iperf3IPv4Version,
		IHL:      iperf3IPv4IHL,
		TTL:      iperf3IPv4TTL,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    ipLayer.DstIP,
		DstIP:    ipLayer.SrcIP,
	}

	// Safe conversion: min() bounds payloadLen to 0xFFFFFFFF which fits in uint32
	payloadLen := min(len(tcpLayer.Payload), iperf3MaxUint32)

	tcpReply := &layers.TCP{
		SrcPort: tcpLayer.DstPort,
		DstPort: tcpLayer.SrcPort,
		Seq:     tcpLayer.Ack,
		Ack:     tcpLayer.Seq + safeconv.Uint32(payloadLen),
		PSH:     true,
		ACK:     true,
		Window:  iperf3TCPWindowSize,
	}
	_ = tcpReply.SetNetworkLayerForChecksum(ipReply) // error is non-critical for simulation

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipReply, tcpReply, gopacket.Payload(payload))
	if err != nil {
		if debugLevel >= DebugLevelInfo {
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

	if debugLevel >= DebugLevelVerbose {
		_, _ = fmt.Fprintf(os.Stdout, "iPerf3: Sent %d bytes to %s:%d\n", len(payload), ipLayer.DstIP, tcpLayer.SrcPort)
	}
}

// GetDeviceIPerf3Config extracts iPerf3 config from device name for display.
func GetDeviceIPerf3Config(device *config.Device) string {
	if device.IPerf3 == nil || !device.IPerf3.Enabled {
		return iperf3Disabled
	}

	cfg := device.IPerf3

	return fmt.Sprintf("Up: %.0f Mbps, Down: %.0f Mbps, Latency: %.1fms",
		cfg.UploadMbps, cfg.DownloadMbps, cfg.TypicalLatencyMs)
}

// Helper to avoid unused import warning.
var _ = strings.TrimSpace
