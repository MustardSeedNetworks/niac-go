package protocols

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

// FTP response constants.
const (
	ftpSyntaxErrorResponse = "501 Syntax error in parameters or arguments.\r\n"
)

// FTP encoding constants.
const (
	maxUint32Payload     = 0xFFFFFFFF // max uint32 for TCP payload length
	ftpConnectionDelayMs = 100        // delay in milliseconds for connection setup
	asciiDEL             = 127        // ASCII DEL character code
	ftpLogMessageMaxLen  = 60         // max length for FTP response logging

	// IP protocol constants.
	ftpIPv4Version      = 4     // IPv4 version field
	ftpIPv4IHL          = 5     // IPv4 header length (5 = 20 bytes)
	ftpIPv4TTL          = 64    // IPv4 default TTL
	ftpIPv6Version      = 6     // IPv6 version field
	ftpIPv6HopLimit     = 64    // IPv6 default hop limit
	ftpTCPWindowSize    = 65535 // Default TCP window size
	ftpPortByteMultiply = 256   // Multiplier for FTP port byte encoding
)

// FTPHandler handles FTP control and data connections.
type FTPHandler struct {
	stack *Stack
}

// NewFTPHandler creates a new FTP handler.
func NewFTPHandler(stack *Stack) *FTPHandler {
	return &FTPHandler{
		stack: stack,
	}
}

// HandleRequest processes an FTP request.
func (h *FTPHandler) HandleRequest(_ *Packet, ipLayer *layers.IPv4, tcpLayer *layers.TCP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse FTP command
	if len(tcpLayer.Payload) == 0 {
		return
	}

	command := strings.TrimSpace(string(tcpLayer.Payload))
	if command == "" {
		return
	}

	if debugLevel >= DebugLevelInfo {
		// SECURITY FIX MEDIUM-4: Sanitize command for logging to prevent log injection
		sanitizedCmd := sanitizeForLogging(command)
		_, _ = fmt.Fprintf(os.Stdout, "FTP command from %s: %s (device: %v)\n",
			ipLayer.SrcIP, sanitizedCmd, getDeviceNames(devices))
	}

	response := h.buildFTPResponse(command, false, devices)
	if response == "" {
		return
	}

	// Send response
	h.sendResponse(ipLayer, tcpLayer, []byte(response), devices)
}

// sendResponse sends an FTP response.
func (h *FTPHandler) sendResponse(
	ipLayer *layers.IPv4,
	tcpLayer *layers.TCP,
	response []byte,
	devices []*config.Device,
) {
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
		// Fallback - would need to get from original packet
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "Cannot send FTP response: no MAC for %s\n", ipLayer.SrcIP)
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
		Version:  ftpIPv4Version,
		IHL:      ftpIPv4IHL,
		TTL:      ftpIPv4TTL,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    ipLayer.DstIP,
		DstIP:    ipLayer.SrcIP,
	}

	// Build TCP header
	// Safe conversion: min() bounds payloadLen to 0xFFFFFFFF which fits in uint32
	payloadLen := min(len(tcpLayer.Payload), maxUint32Payload)

	tcpReply := &layers.TCP{
		SrcPort: tcpLayer.DstPort,
		DstPort: tcpLayer.SrcPort,
		Seq:     tcpLayer.Ack,
		Ack:     tcpLayer.Seq + safeconv.Uint32(payloadLen),
		PSH:     true,
		ACK:     true,
		Window:  ftpTCPWindowSize,
	}
	_ = tcpReply.SetNetworkLayerForChecksum(ipReply) // error is non-critical for simulation

	// Serialize
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts,
		eth,
		ipReply,
		tcpReply,
		gopacket.Payload(response),
	)
	if err != nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "Error serializing FTP response: %v\n", err)
		}

		return
	}

	// Get serial number
	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	// Create and send packet
	responsePkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: serialNum,
		Device:       device,
	}

	h.stack.Send(responsePkt)

	if debugLevel >= DebugLevelInfo {
		responseStr := strings.TrimSpace(string(response))
		if len(responseStr) > ftpLogMessageMaxLen {
			responseStr = responseStr[:ftpLogMessageMaxLen] + "..."
		}

		_, _ = fmt.Fprintf(os.Stdout, "Sent FTP response: %s from %s to %s (device: %s)\n",
			responseStr, ipReply.SrcIP, ipReply.DstIP, device.Name)
	}
}

func (h *FTPHandler) sendResponseV6(
	ipv6 *layers.IPv6,
	tcpLayer *layers.TCP,
	response []byte,
	devices []*config.Device,
	dstMAC net.HardwareAddr,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(devices) == 0 {
		return
	}

	device := devices[0]
	if len(device.MACAddress) == 0 || dstMAC == nil {
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "Cannot send FTP/IPv6 response: missing MAC info\n")
		}

		return
	}

	eth := &layers.Ethernet{
		SrcMAC:       device.MACAddress,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv6,
	}

	ipReply := &layers.IPv6{
		Version:      ftpIPv6Version,
		TrafficClass: 0,
		FlowLabel:    0,
		NextHeader:   layers.IPProtocolTCP,
		HopLimit:     ftpIPv6HopLimit,
		SrcIP:        ipv6.DstIP,
		DstIP:        ipv6.SrcIP,
	}

	// Safe conversion: min() bounds payloadLen2 to 0xFFFFFFFF which fits in uint32
	payloadLen2 := min(len(tcpLayer.Payload), maxUint32Payload)

	tcpReply := &layers.TCP{
		SrcPort: tcpLayer.DstPort,
		DstPort: tcpLayer.SrcPort,
		Seq:     tcpLayer.Ack,
		Ack:     tcpLayer.Seq + safeconv.Uint32(payloadLen2),
		PSH:     true,
		ACK:     true,
		Window:  ftpTCPWindowSize,
	}
	_ = tcpReply.SetNetworkLayerForChecksum(ipReply) // error is non-critical for simulation

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipReply, tcpReply, gopacket.Payload(response))
	if err != nil {
		if debugLevel >= 1 {
			_, _ = fmt.Fprintf(os.Stdout, "FTP/IPv6: Failed to serialize response: %v\n", err)
		}

		return
	}

	h.stack.mu.Lock()
	h.stack.serialNumber++
	serialNum := h.stack.serialNumber
	h.stack.mu.Unlock()

	respPkt := &Packet{
		Buffer:       buffer.Bytes(),
		Length:       len(buffer.Bytes()),
		SerialNumber: serialNum,
		Device:       device,
	}

	h.stack.Send(respPkt)

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "Sent FTP/IPv6 response %d bytes to [%s]\n", len(response), ipv6.SrcIP)
	}
}

func (h *FTPHandler) buildFTPResponse(command string, ipv6 bool, devices []*config.Device) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}

	cmd := strings.ToUpper(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = parts[1]
	}

	return h.dispatchFTPCommand(cmd, arg, ipv6, devices)
}

// dispatchFTPCommand routes FTP commands to appropriate handlers.
func (h *FTPHandler) dispatchFTPCommand(cmd, arg string, ipv6 bool, devices []*config.Device) string {
	switch cmd {
	case "USER":
		return handleFTPUser(arg)
	case "PASS":
		return "230 User logged in, proceed.\r\n"
	case "SYST":
		return handleFTPSyst(devices)
	case "PWD":
		return "257 \"/\" is current directory.\r\n"
	case "TYPE":
		return handleFTPType(arg)
	case "PASV":
		return handleFTPPasv(ipv6, devices)
	case "EPSV":
		return "229 Entering Extended Passive Mode (|||20000|).\r\n"
	case "LIST":
		return "150 Here comes the directory listing.\r\n226 Directory send OK.\r\n"
	case "RETR":
		return handleFTPRetr(arg)
	case "STOR":
		return handleFTPStor(arg)
	case "CWD":
		return handleFTPCwd(arg)
	case "CDUP":
		return "250 Directory successfully changed.\r\n"
	case "DELE":
		return handleFTPDele(arg)
	case "MKD":
		return handleFTPMkd(arg)
	case "RMD":
		return handleFTPRmd(arg)
	case "NOOP":
		return "200 NOOP ok.\r\n"
	case "QUIT":
		return "221 Goodbye.\r\n"
	case "HELP":
		return ftpHelpResponse
	default:
		return handleFTPUnknown(cmd)
	}
}

const ftpHelpResponse = "214-The following commands are recognized:\r\n USER PASS SYST PWD TYPE PASV EPSV LIST RETR STOR\r\n CWD CDUP DELE MKD RMD NOOP QUIT HELP\r\n214 Help OK.\r\n"

func handleFTPUser(arg string) string {
	if arg != "" {
		return "331 User name okay, need password.\r\n"
	}

	return ftpSyntaxErrorResponse
}

func handleFTPSyst(devices []*config.Device) string {
	systemType := "UNIX Type: L8"
	if len(devices) > 0 && devices[0].FTPConfig != nil && devices[0].FTPConfig.SystemType != "" {
		systemType = devices[0].FTPConfig.SystemType
	}

	return fmt.Sprintf("215 %s\r\n", systemType)
}

func handleFTPType(arg string) string {
	if arg != "" {
		return fmt.Sprintf("200 Type set to %s.\r\n", arg)
	}

	return ftpSyntaxErrorResponse
}

func handleFTPPasv(ipv6 bool, devices []*config.Device) string {
	if ipv6 {
		return "522 Network protocol not supported, use EPSV.\r\n"
	}

	ip := selectIPv4Address(devices)
	if ip == nil {
		return "500 Passive mode failed.\r\n"
	}

	port := 20000
	p1 := port / ftpPortByteMultiply
	p2 := port % ftpPortByteMultiply
	ip4 := ip.To4()

	return fmt.Sprintf("227 Entering Passive Mode (%d,%d,%d,%d,%d,%d).\r\n",
		ip4[0], ip4[1], ip4[2], ip4[3], p1, p2)
}

func handleFTPRetr(arg string) string {
	if arg != "" {
		return fmt.Sprintf("550 %s: No such file or directory.\r\n", arg)
	}

	return ftpSyntaxErrorResponse
}

func handleFTPStor(arg string) string {
	if arg != "" {
		return "553 Could not create file (read-only filesystem).\r\n"
	}

	return ftpSyntaxErrorResponse
}

func handleFTPCwd(arg string) string {
	if arg != "" {
		return "250 Directory successfully changed.\r\n"
	}

	return ftpSyntaxErrorResponse
}

func handleFTPDele(arg string) string {
	if arg != "" {
		return "553 Could not delete file (read-only filesystem).\r\n"
	}

	return ftpSyntaxErrorResponse
}

func handleFTPMkd(arg string) string {
	if arg != "" {
		return "257 Directory created.\r\n"
	}

	return ftpSyntaxErrorResponse
}

func handleFTPRmd(arg string) string {
	if arg != "" {
		return "250 Directory deleted.\r\n"
	}

	return ftpSyntaxErrorResponse
}

func handleFTPUnknown(cmd string) string {
	if len(cmd) <= 4 && cmd == strings.ToUpper(cmd) {
		return "502 Command not implemented.\r\n"
	}

	return ""
}

func selectIPv4Address(devices []*config.Device) net.IP {
	if len(devices) == 0 {
		return nil
	}

	for _, ip := range devices[0].IPAddresses {
		if v4 := ip.To4(); v4 != nil {
			return v4
		}
	}

	return nil
}

// SendWelcome sends FTP welcome banner when connection is established.
func (h *FTPHandler) SendWelcome(ipLayer *layers.IPv4, tcpLayer *layers.TCP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Only send welcome on new connections (SYN+ACK)
	if !tcpLayer.SYN || !tcpLayer.ACK {
		return
	}

	if len(devices) == 0 {
		return
	}

	device := devices[0]
	deviceName := device.Name

	// Use configured welcome banner if available
	var welcome string
	if device.FTPConfig != nil && device.FTPConfig.WelcomeBanner != "" {
		welcome = device.FTPConfig.WelcomeBanner
		// Ensure it ends with \r\n
		if !strings.HasSuffix(welcome, "\r\n") {
			welcome += "\r\n"
		}
	} else {
		welcome = fmt.Sprintf("220 %s FTP Server (NIAC-Go) ready.\r\n", deviceName)
	}

	// Small delay to let connection establish
	go func() {
		time.Sleep(ftpConnectionDelayMs * time.Millisecond)
		h.sendResponse(ipLayer, tcpLayer, []byte(welcome), devices)
	}()

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "Scheduled FTP welcome banner for %s\n", deviceName)
	}
}

// HandleRequestV6 processes an FTP request over IPv6.
func (h *FTPHandler) HandleRequestV6(
	pkt *Packet,
	_ gopacket.Packet,
	ipv6 *layers.IPv6,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) == 0 {
		return
	}

	command := strings.TrimSpace(string(tcpLayer.Payload))
	if command == "" {
		return
	}

	if debugLevel >= DebugLevelInfo {
		// SECURITY FIX: Sanitize command for logging to prevent log injection (same as IPv4 handler)
		sanitizedCmd := sanitizeForLogging(command)
		_, _ = fmt.Fprintf(os.Stdout, "FTP/IPv6 command from [%s]: %s (device: %v)\n",
			ipv6.SrcIP, sanitizedCmd, getDeviceNames(devices))
	}

	response := h.buildFTPResponse(command, true, devices)
	if response == "" {
		return
	}

	h.sendResponseV6(ipv6, tcpLayer, []byte(response), devices, pkt.GetSourceMAC())
}

// sanitizeForLogging removes control characters and newlines to prevent log injection
// SECURITY FIX MEDIUM-4: Prevents malicious payloads from corrupting logs.
func sanitizeForLogging(s string) string {
	// Replace control characters (ASCII 0-31 except space) with '?'
	var result strings.Builder

	result.Grow(len(s))

	for _, r := range s {
		switch {
		case r < 32 && r != ' ':
			result.WriteRune('?') // Replace control chars
		case r == asciiDEL:
			result.WriteRune('?') // Replace DEL
		default:
			result.WriteRune(r)
		}
	}

	return result.String()
}
