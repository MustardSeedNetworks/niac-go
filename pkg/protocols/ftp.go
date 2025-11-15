package protocols

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// FTPHandler handles FTP control and data connections
type FTPHandler struct {
	stack *Stack
}

// NewFTPHandler creates a new FTP handler
func NewFTPHandler(stack *Stack) *FTPHandler {
	return &FTPHandler{
		stack: stack,
	}
}

// HandleRequest processes an FTP request
func (h *FTPHandler) HandleRequest(pkt *Packet, ipLayer *layers.IPv4, tcpLayer *layers.TCP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse FTP command
	if len(tcpLayer.Payload) == 0 {
		return
	}

	command := strings.TrimSpace(string(tcpLayer.Payload))
	if command == "" {
		return
	}

	if debugLevel >= 2 {
		// SECURITY FIX MEDIUM-4: Sanitize command for logging to prevent log injection
		sanitizedCmd := sanitizeForLogging(command)
		fmt.Printf("FTP command from %s: %s (device: %v)\n",
			ipLayer.SrcIP, sanitizedCmd, getDeviceNames(devices))
	}

	response := h.buildFTPResponse(command, false, devices)
	if response == "" {
		return
	}

	// Send response
	h.sendResponse(ipLayer, tcpLayer, []byte(response), devices)
}

// sendResponse sends an FTP response
func (h *FTPHandler) sendResponse(ipLayer *layers.IPv4, tcpLayer *layers.TCP, response []byte, devices []*config.Device) {
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
		if debugLevel >= 2 {
			fmt.Printf("Cannot send FTP response: no MAC for %s\n", ipLayer.SrcIP)
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

	// Build TCP header
	tcpReply := &layers.TCP{
		SrcPort: tcpLayer.DstPort,
		DstPort: tcpLayer.SrcPort,
		Seq:     tcpLayer.Ack,
		Ack:     tcpLayer.Seq + uint32(len(tcpLayer.Payload)),
		PSH:     true,
		ACK:     true,
		Window:  65535,
	}
	tcpReply.SetNetworkLayerForChecksum(ipReply)

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
		if debugLevel >= 2 {
			fmt.Printf("Error serializing FTP response: %v\n", err)
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

	if debugLevel >= 2 {
		responseStr := strings.TrimSpace(string(response))
		if len(responseStr) > 60 {
			responseStr = responseStr[:60] + "..."
		}
		fmt.Printf("Sent FTP response: %s from %s to %s (device: %s)\n",
			responseStr, ipReply.SrcIP, ipReply.DstIP, device.Name)
	}
}

func (h *FTPHandler) sendResponseV6(ipv6 *layers.IPv6, tcpLayer *layers.TCP, response []byte, devices []*config.Device, dstMAC net.HardwareAddr) {
	debugLevel := h.stack.GetDebugLevel()

	if len(devices) == 0 {
		return
	}

	device := devices[0]
	if len(device.MACAddress) == 0 || dstMAC == nil {
		if debugLevel >= 2 {
			fmt.Printf("Cannot send FTP/IPv6 response: missing MAC info\n")
		}
		return
	}

	eth := &layers.Ethernet{
		SrcMAC:       device.MACAddress,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv6,
	}

	ipReply := &layers.IPv6{
		Version:      6,
		TrafficClass: 0,
		FlowLabel:    0,
		NextHeader:   layers.IPProtocolTCP,
		HopLimit:     64,
		SrcIP:        ipv6.DstIP,
		DstIP:        ipv6.SrcIP,
	}

	tcpReply := &layers.TCP{
		SrcPort: tcpLayer.DstPort,
		DstPort: tcpLayer.SrcPort,
		Seq:     tcpLayer.Ack,
		Ack:     tcpLayer.Seq + uint32(len(tcpLayer.Payload)),
		PSH:     true,
		ACK:     true,
		Window:  65535,
	}
	tcpReply.SetNetworkLayerForChecksum(ipReply)

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(buffer, opts, eth, ipReply, tcpReply, gopacket.Payload(response)); err != nil {
		if debugLevel >= 1 {
			fmt.Printf("FTP/IPv6: Failed to serialize response: %v\n", err)
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

	if debugLevel >= 2 {
		fmt.Printf("Sent FTP/IPv6 response %d bytes to [%s]\n", len(response), ipv6.SrcIP)
	}
}

func (h *FTPHandler) buildFTPResponse(command string, ipv6 bool, devices []*config.Device) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}

	cmd := strings.ToUpper(parts[0])
	switch cmd {
	case "USER":
		if len(parts) > 1 {
			return "331 User name okay, need password.\r\n"
		}
		return "501 Syntax error in parameters or arguments.\r\n"
	case "PASS":
		return "230 User logged in, proceed.\r\n"
	case "SYST":
		systemType := "UNIX Type: L8"
		if len(devices) > 0 && devices[0].FTPConfig != nil && devices[0].FTPConfig.SystemType != "" {
			systemType = devices[0].FTPConfig.SystemType
		}
		return fmt.Sprintf("215 %s\r\n", systemType)
	case "PWD":
		return "257 \"/\" is current directory.\r\n"
	case "TYPE":
		if len(parts) > 1 {
			return fmt.Sprintf("200 Type set to %s.\r\n", parts[1])
		}
		return "501 Syntax error in parameters or arguments.\r\n"
	case "PASV":
		if ipv6 {
			return "522 Network protocol not supported, use EPSV.\r\n"
		}
		ip := selectIPv4Address(devices)
		if ip == nil {
			return "500 Passive mode failed.\r\n"
		}
		port := 20000
		p1 := port / 256
		p2 := port % 256
		ip4 := ip.To4()
		return fmt.Sprintf("227 Entering Passive Mode (%d,%d,%d,%d,%d,%d).\r\n",
			ip4[0], ip4[1], ip4[2], ip4[3], p1, p2)
	case "EPSV":
		port := 20000
		return fmt.Sprintf("229 Entering Extended Passive Mode (|||%d|).\r\n", port)
	case "LIST":
		return "150 Here comes the directory listing.\r\n226 Directory send OK.\r\n"
	case "RETR":
		if len(parts) > 1 {
			return fmt.Sprintf("550 %s: No such file or directory.\r\n", parts[1])
		}
		return "501 Syntax error in parameters or arguments.\r\n"
	case "STOR":
		if len(parts) > 1 {
			return "553 Could not create file (read-only filesystem).\r\n"
		}
		return "501 Syntax error in parameters or arguments.\r\n"
	case "CWD":
		if len(parts) > 1 {
			return "250 Directory successfully changed.\r\n"
		}
		return "501 Syntax error in parameters or arguments.\r\n"
	case "CDUP":
		return "250 Directory successfully changed.\r\n"
	case "DELE":
		if len(parts) > 1 {
			return "553 Could not delete file (read-only filesystem).\r\n"
		}
		return "501 Syntax error in parameters or arguments.\r\n"
	case "MKD":
		if len(parts) > 1 {
			return "257 Directory created.\r\n"
		}
		return "501 Syntax error in parameters or arguments.\r\n"
	case "RMD":
		if len(parts) > 1 {
			return "250 Directory deleted.\r\n"
		}
		return "501 Syntax error in parameters or arguments.\r\n"
	case "NOOP":
		return "200 NOOP ok.\r\n"
	case "QUIT":
		return "221 Goodbye.\r\n"
	case "HELP":
		return "214-The following commands are recognized:\r\n USER PASS SYST PWD TYPE PASV EPSV LIST RETR STOR\r\n CWD CDUP DELE MKD RMD NOOP QUIT HELP\r\n214 Help OK.\r\n"
	default:
		if len(cmd) <= 4 && cmd == strings.ToUpper(cmd) {
			return "502 Command not implemented.\r\n"
		}
		return ""
	}
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

// SendWelcome sends FTP welcome banner when connection is established
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
		time.Sleep(100 * time.Millisecond)
		h.sendResponse(ipLayer, tcpLayer, []byte(welcome), devices)
	}()

	if debugLevel >= 2 {
		fmt.Printf("Scheduled FTP welcome banner for %s\n", deviceName)
	}
}

// HandleRequestV6 processes an FTP request over IPv6
func (h *FTPHandler) HandleRequestV6(pkt *Packet, packet gopacket.Packet, ipv6 *layers.IPv6, tcpLayer *layers.TCP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	if len(tcpLayer.Payload) == 0 {
		return
	}

	command := strings.TrimSpace(string(tcpLayer.Payload))
	if command == "" {
		return
	}

	if debugLevel >= 2 {
		fmt.Printf("FTP/IPv6 command from [%s]: %s (device: %v)\n",
			ipv6.SrcIP, command, getDeviceNames(devices))
	}

	response := h.buildFTPResponse(command, true, devices)
	if response == "" {
		return
	}

	h.sendResponseV6(ipv6, tcpLayer, []byte(response), devices, pkt.GetSourceMAC())
}

// sanitizeForLogging removes control characters and newlines to prevent log injection
// SECURITY FIX MEDIUM-4: Prevents malicious payloads from corrupting logs
func sanitizeForLogging(s string) string {
	// Replace control characters (ASCII 0-31 except space) with '?'
	var result strings.Builder
	result.Grow(len(s))

	for _, r := range s {
		if r < 32 && r != ' ' {
			result.WriteRune('?') // Replace control chars
		} else if r == 127 {
			result.WriteRune('?') // Replace DEL
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}
