package protocols

import (
	"fmt"
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
		fmt.Printf("FTP command from %s: %s (device: %v)\n",
			ipLayer.SrcIP, command, getDeviceNames(devices))
	}

	// Parse command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToUpper(parts[0])
	var response string

	// Handle FTP commands
	switch cmd {
	case "USER":
		if len(parts) > 1 {
			response = "331 User name okay, need password.\r\n"
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "PASS":
		// Accept any password
		response = "230 User logged in, proceed.\r\n"

	case "SYST":
		// Use configured system type if available
		systemType := "UNIX Type: L8"
		if len(devices) > 0 && devices[0].FTPConfig != nil && devices[0].FTPConfig.SystemType != "" {
			systemType = devices[0].FTPConfig.SystemType
		}
		response = fmt.Sprintf("215 %s\r\n", systemType)

	case "PWD":
		response = "257 \"/\" is current directory.\r\n"

	case "TYPE":
		if len(parts) > 1 {
			response = fmt.Sprintf("200 Type set to %s.\r\n", parts[1])
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "PASV":
		// Generate passive mode response
		// Format: 227 Entering Passive Mode (h1,h2,h3,h4,p1,p2)
		// We'll use a dummy port
		if len(devices) > 0 && len(devices[0].IPAddresses) > 0 {
			ip := devices[0].IPAddresses[0]
			port := 20000 // Dummy data port
			p1 := port / 256
			p2 := port % 256
			response = fmt.Sprintf("227 Entering Passive Mode (%d,%d,%d,%d,%d,%d).\r\n",
				ip[0], ip[1], ip[2], ip[3], p1, p2)
		} else {
			response = "500 Passive mode failed.\r\n"
		}

	case "LIST":
		// Send dummy file listing via control connection
		response = "150 Here comes the directory listing.\r\n"
		// In a real implementation, we'd send data via data connection
		// For simulation, just send completion
		response += "226 Directory send OK.\r\n"

	case "RETR":
		if len(parts) > 1 {
			filename := parts[1]
			response = fmt.Sprintf("550 %s: No such file or directory.\r\n", filename)
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "STOR":
		if len(parts) > 1 {
			response = "553 Could not create file (read-only filesystem).\r\n"
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "CWD":
		if len(parts) > 1 {
			response = "250 Directory successfully changed.\r\n"
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "CDUP":
		response = "250 Directory successfully changed.\r\n"

	case "DELE":
		if len(parts) > 1 {
			response = "553 Could not delete file (read-only filesystem).\r\n"
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "MKD":
		if len(parts) > 1 {
			response = "257 Directory created.\r\n"
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "RMD":
		if len(parts) > 1 {
			response = "250 Directory deleted.\r\n"
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "NOOP":
		response = "200 NOOP ok.\r\n"

	case "QUIT":
		response = "221 Goodbye.\r\n"

	case "HELP":
		response = "214-The following commands are recognized:\r\n" +
			" USER PASS SYST PWD TYPE PASV LIST RETR STOR\r\n" +
			" CWD CDUP DELE MKD RMD NOOP QUIT HELP\r\n" +
			"214 Help OK.\r\n"

	default:
		// If this looks like an FTP command but we don't recognize it
		if len(cmd) <= 4 && cmd == strings.ToUpper(cmd) {
			response = "502 Command not implemented.\r\n"
		} else {
			// Not an FTP command, ignore
			return
		}
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

	if debugLevel >= 2 {
		fmt.Printf("FTP/IPv6 request from [%s] (stub - not fully implemented)\n", ipv6.SrcIP)
	}

	// Pending: implement full FTP over IPv6 (issue #80)
}
