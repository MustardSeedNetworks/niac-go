package protocols

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
)

// HTTPHandler handles HTTP requests and responses
type HTTPHandler struct {
	stack *Stack
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(stack *Stack) *HTTPHandler {
	return &HTTPHandler{
		stack: stack,
	}
}

// HandleRequest processes an HTTP request
func (h *HTTPHandler) HandleRequest(pkt *Packet, ipLayer *layers.IPv4, tcpLayer *layers.TCP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse HTTP request
	if len(tcpLayer.Payload) == 0 {
		return
	}

	request, err := parseHTTPRequest(tcpLayer.Payload)
	if err != nil {
		if debugLevel >= 3 {
			fmt.Printf("Failed to parse HTTP request: %v\n", err)
		}
		return
	}

	if debugLevel >= 2 {
		fmt.Printf("HTTP %s %s from %s (device: %v)\n",
			request.Method, request.Path, ipLayer.SrcIP, getDeviceNames(devices))
	}

	// Generate response
	response := h.generateResponse(request, devices)

	// Send response
	h.sendResponse(ipLayer, tcpLayer, response, devices)
}

// HTTPRequest represents a parsed HTTP request
type HTTPRequest struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
	Body    []byte
}

// parseHTTPRequest parses HTTP request from payload
func parseHTTPRequest(payload []byte) (*HTTPRequest, error) {
	reader := bufio.NewReader(bytes.NewReader(payload))

	// Read request line
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read request line: %v", err)
	}

	parts := strings.SplitN(strings.TrimSpace(requestLine), " ", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid request line: %s", requestLine)
	}

	request := &HTTPRequest{
		Method:  parts[0],
		Path:    parts[1],
		Version: parts[2],
		Headers: make(map[string]string),
	}

	// Parse headers
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		// Parse header
		headerParts := strings.SplitN(line, ":", 2)
		if len(headerParts) == 2 {
			request.Headers[strings.TrimSpace(headerParts[0])] = strings.TrimSpace(headerParts[1])
		}
	}

	return request, nil
}

// generateResponse generates an HTTP response
func (h *HTTPHandler) generateResponse(request *HTTPRequest, devices []*config.Device) []byte {
	var response strings.Builder

	// Get device info for response
	deviceName := "Unknown"
	deviceType := "device"
	var device *config.Device
	if len(devices) > 0 {
		device = devices[0]
		deviceName = device.Name
		deviceType = device.Type
	}

	// Get server name from config
	serverName := "NIAC-Go/1.0.0"
	if device != nil && device.HTTPConfig != nil && device.HTTPConfig.ServerName != "" {
		serverName = device.HTTPConfig.ServerName
	}

	// Check for custom endpoints in config first
	var customEndpoint *config.HTTPEndpoint
	if device != nil && device.HTTPConfig != nil && device.HTTPConfig.Enabled {
		for i := range device.HTTPConfig.Endpoints {
			ep := &device.HTTPConfig.Endpoints[i]
			if ep.Path == request.Path {
				// Check method match (default to GET if not specified)
				epMethod := ep.Method
				if epMethod == "" {
					epMethod = "GET"
				}
				if epMethod == request.Method {
					customEndpoint = ep
					break
				}
			}
		}
	}

	// Determine response based on custom endpoint or default paths
	statusCode := 200
	statusText := "OK"
	contentType := "text/html"
	var body string

	if customEndpoint != nil {
		// Use custom endpoint configuration
		statusCode = customEndpoint.StatusCode
		if statusCode == 0 {
			statusCode = 200
		}
		contentType = customEndpoint.ContentType
		if contentType == "" {
			contentType = "text/html"
		}
		body = customEndpoint.Body
		statusText = getStatusText(statusCode)
	} else {
		// Default endpoints
		switch request.Path {
		case "/", "/index.html":
			body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>%s - NIAC-Go</title></head>
<body>
<h1>%s</h1>
<p>Type: %s</p>
<p>Simulated by NIAC-Go</p>
<hr>
<small>Network In A Can - Go Edition</small>
</body>
</html>`, deviceName, deviceName, deviceType)

		case "/status":
			stats := h.stack.GetStats()
			body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>Status - %s</title></head>
<body>
<h1>Device Status</h1>
<p>Name: %s</p>
<p>Type: %s</p>
<h2>Statistics</h2>
<ul>
<li>Packets Received: %d</li>
<li>Packets Sent: %d</li>
<li>ARP Requests: %d</li>
<li>ICMP Requests: %d</li>
</ul>
</body>
</html>`, deviceName, deviceName, deviceType,
				stats.PacketsReceived, stats.PacketsSent,
				stats.ARPRequests, stats.ICMPRequests)

		case "/api/info":
			contentType = "application/json"
			body = fmt.Sprintf(`{
  "name": "%s",
  "type": "%s",
  "version": "NIAC-Go v1.0.0",
  "status": "running"
}`, deviceName, deviceType)

		default:
			statusCode = 404
			statusText = "Not Found"
			body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>404 Not Found</title></head>
<body>
<h1>404 Not Found</h1>
<p>The requested URL %s was not found on this server.</p>
<hr>
<small>%s - NIAC-Go</small>
</body>
</html>`, request.Path, deviceName)
		}
	}

	// Build response
	response.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, statusText))
	response.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().UTC().Format(time.RFC1123)))
	response.WriteString(fmt.Sprintf("Server: %s\r\n", serverName))
	response.WriteString(fmt.Sprintf("Content-Type: %s\r\n", contentType))
	response.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(body)))
	response.WriteString("Connection: close\r\n")
	response.WriteString("\r\n")
	response.WriteString(body)

	return []byte(response.String())
}

// getStatusText returns HTTP status text for a status code
func getStatusText(code int) string {
	switch code {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 301:
		return "Moved Permanently"
	case 302:
		return "Found"
	case 304:
		return "Not Modified"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 500:
		return "Internal Server Error"
	case 501:
		return "Not Implemented"
	case 503:
		return "Service Unavailable"
	default:
		return "Unknown"
	}
}

// sendResponse sends an HTTP response
func (h *HTTPHandler) sendResponse(ipLayer *layers.IPv4, tcpLayer *layers.TCP, response []byte, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	if len(devices) == 0 {
		return
	}

	device := devices[0]
	if len(device.MACAddress) == 0 {
		return
	}

	// Get source MAC (lookup by source IP)
	srcDevices := h.stack.GetDevices().GetByIP(ipLayer.SrcIP)
	var srcMAC []byte
	if srcDevices != nil && len(srcDevices) > 0 && len(srcDevices[0].MACAddress) > 0 {
		srcMAC = srcDevices[0].MACAddress
	} else {
		// Cannot find source MAC - skip sending
		if debugLevel >= 2 {
			fmt.Printf("Cannot send HTTP response: no MAC for %s\n", ipLayer.SrcIP)
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
			fmt.Printf("Error serializing HTTP response: %v\n", err)
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
		fmt.Printf("Sent HTTP response: %d bytes from %s to %s (device: %s)\n",
			len(response), ipReply.SrcIP, ipReply.DstIP, device.Name)
	}
}

// getDeviceNames returns comma-separated device names
func getDeviceNames(devices []*config.Device) string {
	if len(devices) == 0 {
		return "unknown"
	}
	names := make([]string, len(devices))
	for i, d := range devices {
		names[i] = d.Name
	}
	return strings.Join(names, ", ")
}

// HandleRequestV6 processes an HTTP request over IPv6
func (h *HTTPHandler) HandleRequestV6(pkt *Packet, packet gopacket.Packet, ipv6 *layers.IPv6, tcpLayer *layers.TCP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse HTTP request
	if len(tcpLayer.Payload) == 0 {
		return
	}

	request, err := parseHTTPRequest(tcpLayer.Payload)
	if err != nil {
		if debugLevel >= 3 {
			fmt.Printf("Failed to parse HTTP/IPv6 request: %v\n", err)
		}
		return
	}

	if debugLevel >= 2 {
		fmt.Printf("HTTP/IPv6 %s %s from [%s] (device: %v)\n",
			request.Method, request.Path, ipv6.SrcIP, getDeviceNames(devices))
	}

	// Generate response
	response := h.generateResponse(request, devices)

	// Send response over IPv6
	h.sendResponseV6(ipv6, tcpLayer, response, devices)
}

// sendResponseV6 sends HTTP response over IPv6 (stub - basic implementation)
func (h *HTTPHandler) sendResponseV6(ipv6 *layers.IPv6, tcpLayer *layers.TCP, response []byte, devices []*config.Device) {
	// TODO: Implement full HTTP response over IPv6
	debugLevel := h.stack.GetDebugLevel()
	if debugLevel >= 2 {
		fmt.Printf("HTTP/IPv6: Would send %d byte response (stub)\n", len(response))
	}
}
