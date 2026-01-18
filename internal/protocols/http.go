package protocols

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// HTTP parsing constants.
const (
	httpRequestLineParts = 3          // method, path, version
	httpHeaderParts      = 2          // key, value
	maxUint32TCP         = 0xFFFFFFFF // max uint32 for TCP payload length

	// IP protocol constants.
	httpIPv4Version   = 4     // IPv4 version field
	httpIPv4IHL       = 5     // IPv4 header length (5 = 20 bytes)
	httpIPv4TTL       = 64    // IPv4 default TTL
	httpIPv6Version   = 6     // IPv6 version field
	httpIPv6HopLimit  = 64    // IPv6 default hop limit
	httpTCPWindowSize = 65535 // Default TCP window size
)

// HTTP content types.
const (
	contentTypeHTML = "text/html"
	contentTypeJSON = "application/json"
)

// HTTP status codes.
const (
	httpStatusOK                  = 200
	httpStatusCreated             = 201
	httpStatusNoContent           = 204
	httpStatusMovedPermanently    = 301
	httpStatusFound               = 302
	httpStatusNotModified         = 304
	httpStatusBadRequest          = 400
	httpStatusUnauthorized        = 401
	httpStatusForbidden           = 403
	httpStatusNotFound            = 404
	httpStatusInternalServerError = 500
	httpStatusNotImplemented      = 501
	httpStatusServiceUnavailable  = 503
)

// HTTPHandler handles HTTP requests and responses.
type HTTPHandler struct {
	stack *Stack
}

// NewHTTPHandler creates a new HTTP handler.
func NewHTTPHandler(stack *Stack) *HTTPHandler {
	return &HTTPHandler{
		stack: stack,
	}
}

// HandleRequest processes an HTTP request.
func (h *HTTPHandler) HandleRequest(_ *Packet, ipLayer *layers.IPv4, tcpLayer *layers.TCP, devices []*config.Device) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse HTTP request
	if len(tcpLayer.Payload) == 0 {
		return
	}

	request, err := parseHTTPRequest(tcpLayer.Payload)
	if err != nil {
		if debugLevel >= DebugLevelVerbose {
			_, _ = fmt.Fprintf(os.Stdout, "Failed to parse HTTP request: %v\n", err)
		}

		return
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "HTTP %s %s from %s (device: %v)\n",
			request.Method, request.Path, ipLayer.SrcIP, getDeviceNames(devices))
	}

	// Generate response
	response := h.generateResponse(request, devices)

	// Send response
	h.sendResponse(ipLayer, tcpLayer, response, devices)
}

// HTTPRequest represents a parsed HTTP request.
type HTTPRequest struct {
	Method  string
	Path    string
	Version string
	Headers map[string]string
	Body    []byte
}

// parseHTTPRequest parses HTTP request from payload.
func parseHTTPRequest(payload []byte) (*HTTPRequest, error) {
	reader := bufio.NewReader(bytes.NewReader(payload))

	// Read request line
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read request line: %w", err)
	}

	parts := strings.SplitN(strings.TrimSpace(requestLine), " ", httpRequestLineParts)
	if len(parts) != httpRequestLineParts {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRequestLine, requestLine)
	}

	request := &HTTPRequest{
		Method:  parts[0],
		Path:    parts[1],
		Version: parts[2],
		Headers: make(map[string]string),
	}

	// Parse headers
	// SECURITY FIX MEDIUM-3: Limit number of headers to prevent resource exhaustion
	const maxHeaderCount = 100

	headerCount := 0

	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			// EOF is expected when we reach the end of headers
			if readErr != io.EOF {
				return nil, fmt.Errorf("failed to read header: %w", readErr)
			}

			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		// Parse header
		headerParts := strings.SplitN(line, ":", httpHeaderParts)
		if len(headerParts) == httpHeaderParts {
			// Check header limit to prevent header bomb attacks
			if headerCount >= maxHeaderCount {
				// Silently ignore excessive headers
				break
			}

			request.Headers[strings.TrimSpace(headerParts[0])] = strings.TrimSpace(headerParts[1])
			headerCount++
		}
	}

	return request, nil
}

// findCustomEndpoint finds a matching custom endpoint from device config.
func findCustomEndpoint(device *config.Device, request *HTTPRequest) *config.HTTPEndpoint {
	if device == nil || device.HTTPConfig == nil || !device.HTTPConfig.Enabled {
		return nil
	}

	for i := range device.HTTPConfig.Endpoints {
		ep := &device.HTTPConfig.Endpoints[i]
		if ep.Path != request.Path {
			continue
		}

		// Check method match (default to GET if not specified)
		epMethod := ep.Method
		if epMethod == "" {
			epMethod = "GET"
		}

		if epMethod == request.Method {
			return ep
		}
	}

	return nil
}

// httpDeviceInfo contains device info for HTTP response generation.
type httpDeviceInfo struct {
	name       string
	deviceType string
	serverName string
	device     *config.Device
}

// getHTTPDeviceInfo extracts device info for response generation.
func getHTTPDeviceInfo(devices []*config.Device) httpDeviceInfo {
	info := httpDeviceInfo{
		name:       "Unknown",
		deviceType: "device",
		serverName: "NIAC-Go/1.0.0",
	}

	if len(devices) > 0 {
		info.device = devices[0]
		info.name = info.device.Name
		info.deviceType = info.device.Type
		if info.device.HTTPConfig != nil && info.device.HTTPConfig.ServerName != "" {
			info.serverName = info.device.HTTPConfig.ServerName
		}
	}

	return info
}

// generateDefaultBody generates the body for default HTTP endpoints.
func (h *HTTPHandler) generateDefaultBody(path string, info httpDeviceInfo) (string, int, string) {
	switch path {
	case "/", "/index.html":
		return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>%s - NIAC-Go</title></head>
<body>
<h1>%s</h1>
<p>Type: %s</p>
<p>Simulated by NIAC-Go</p>
<hr>
<small>Network In A Can - Go Edition</small>
</body>
</html>`, info.name, info.name, info.deviceType), httpStatusOK, contentTypeHTML

	case "/status":
		stats := h.stack.GetStats()
		return fmt.Sprintf(`<!DOCTYPE html>
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
</html>`, info.name, info.name, info.deviceType,
			stats.PacketsReceived, stats.PacketsSent,
			stats.ARPRequests, stats.ICMPRequests), httpStatusOK, contentTypeHTML

	case "/api/info":
		return fmt.Sprintf(`{
  "name": "%s",
  "type": "%s",
  "version": "NIAC-Go v1.0.0",
  "status": "running"
}`, info.name, info.deviceType), httpStatusOK, contentTypeJSON

	default:
		return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>404 Not Found</title></head>
<body>
<h1>404 Not Found</h1>
<p>The requested URL %s was not found on this server.</p>
<hr>
<small>%s - NIAC-Go</small>
</body>
</html>`, path, info.name), httpStatusNotFound, contentTypeHTML
	}
}

// buildHTTPResponse constructs the final HTTP response bytes.
func buildHTTPResponse(statusCode int, serverName, contentType, body string) []byte {
	var response strings.Builder

	statusText := getStatusText(statusCode)
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

// generateResponse generates an HTTP response.
func (h *HTTPHandler) generateResponse(request *HTTPRequest, devices []*config.Device) []byte {
	info := getHTTPDeviceInfo(devices)
	customEndpoint := findCustomEndpoint(info.device, request)

	var body string
	var statusCode int
	var contentType string

	if customEndpoint != nil {
		statusCode = customEndpoint.StatusCode
		if statusCode == 0 {
			statusCode = httpStatusOK
		}
		contentType = customEndpoint.ContentType
		if contentType == "" {
			contentType = contentTypeHTML
		}
		body = customEndpoint.Body
	} else {
		body, statusCode, contentType = h.generateDefaultBody(request.Path, info)
	}

	return buildHTTPResponse(statusCode, info.serverName, contentType, body)
}

// getStatusText returns HTTP status text for a status code.
func getStatusText(code int) string {
	switch code {
	case httpStatusOK:
		return "OK"
	case httpStatusCreated:
		return "Created"
	case httpStatusNoContent:
		return "No Content"
	case httpStatusMovedPermanently:
		return "Moved Permanently"
	case httpStatusFound:
		return "Found"
	case httpStatusNotModified:
		return "Not Modified"
	case httpStatusBadRequest:
		return "Bad Request"
	case httpStatusUnauthorized:
		return "Unauthorized"
	case httpStatusForbidden:
		return "Forbidden"
	case httpStatusNotFound:
		return "Not Found"
	case httpStatusInternalServerError:
		return "Internal Server Error"
	case httpStatusNotImplemented:
		return "Not Implemented"
	case httpStatusServiceUnavailable:
		return "Service Unavailable"
	default:
		return "Unknown"
	}
}

// sendResponse sends an HTTP response.
func (h *HTTPHandler) sendResponse(
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

	// Get source MAC (lookup by source IP)
	srcDevices := h.stack.GetDevices().GetByIP(ipLayer.SrcIP)

	var srcMAC []byte

	if len(srcDevices) > 0 && len(srcDevices[0].MACAddress) > 0 {
		srcMAC = srcDevices[0].MACAddress
	} else {
		// Cannot find source MAC - skip sending
		if debugLevel >= DebugLevelInfo {
			_, _ = fmt.Fprintf(os.Stdout, "Cannot send HTTP response: no MAC for %s\n", ipLayer.SrcIP)
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
		Version:  httpIPv4Version,
		IHL:      httpIPv4IHL,
		TTL:      httpIPv4TTL,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    ipLayer.DstIP,
		DstIP:    ipLayer.SrcIP,
	}

	// Build TCP header
	// Safe conversion: min() bounds payloadLen to 0xFFFFFFFF which fits in uint32
	payloadLen := min(len(tcpLayer.Payload), maxUint32TCP)

	tcpReply := &layers.TCP{
		SrcPort: tcpLayer.DstPort,
		DstPort: tcpLayer.SrcPort,
		Seq:     tcpLayer.Ack,
		Ack:     tcpLayer.Seq + safeUint32(payloadLen),
		PSH:     true,
		ACK:     true,
		Window:  httpTCPWindowSize,
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
			_, _ = fmt.Fprintf(os.Stdout, "Error serializing HTTP response: %v\n", err)
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
		_, _ = fmt.Fprintf(os.Stdout, "Sent HTTP response: %d bytes from %s to %s (device: %s)\n",
			len(response), ipReply.SrcIP, ipReply.DstIP, device.Name)
	}
}

// getDeviceNames returns comma-separated device names.
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

// HandleRequestV6 processes an HTTP request over IPv6.
func (h *HTTPHandler) HandleRequestV6(
	pkt *Packet,
	_ gopacket.Packet,
	ipv6 *layers.IPv6,
	tcpLayer *layers.TCP,
	devices []*config.Device,
) {
	debugLevel := h.stack.GetDebugLevel()

	// Parse HTTP request
	if len(tcpLayer.Payload) == 0 {
		return
	}

	request, err := parseHTTPRequest(tcpLayer.Payload)
	if err != nil {
		if debugLevel >= DebugLevelVerbose {
			_, _ = fmt.Fprintf(os.Stdout, "Failed to parse HTTP/IPv6 request: %v\n", err)
		}

		return
	}

	if debugLevel >= DebugLevelInfo {
		_, _ = fmt.Fprintf(os.Stdout, "HTTP/IPv6 %s %s from [%s] (device: %v)\n",
			request.Method, request.Path, ipv6.SrcIP, getDeviceNames(devices))
	}

	// Generate response
	response := h.generateResponse(request, devices)

	// Send response over IPv6
	h.sendResponseV6(ipv6, tcpLayer, response, devices, pkt.GetSourceMAC())
}

// sendResponseV6 sends HTTP response over IPv6.
func (h *HTTPHandler) sendResponseV6(
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
			_, _ = fmt.Fprintf(os.Stdout, "HTTP/IPv6: Cannot send response, missing MAC info\n")
		}

		return
	}

	ipReply := &layers.IPv6{
		Version:      httpIPv6Version,
		TrafficClass: 0,
		FlowLabel:    0,
		NextHeader:   layers.IPProtocolTCP,
		HopLimit:     httpIPv6HopLimit,
		SrcIP:        ipv6.DstIP,
		DstIP:        ipv6.SrcIP,
	}

	// Safe conversion: min() bounds payloadLen2 to 0xFFFFFFFF which fits in uint32
	payloadLen2 := min(len(tcpLayer.Payload), maxUint32TCP)

	tcpReply := &layers.TCP{
		SrcPort: tcpLayer.DstPort,
		DstPort: tcpLayer.SrcPort,
		Seq:     tcpLayer.Ack,
		Ack:     tcpLayer.Seq + safeUint32(payloadLen2),
		PSH:     true,
		ACK:     true,
		Window:  httpTCPWindowSize,
	}
	_ = tcpReply.SetNetworkLayerForChecksum(ipReply) // error is non-critical for simulation

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	eth := &layers.Ethernet{
		SrcMAC:       device.MACAddress,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv6,
	}

	err := gopacket.SerializeLayers(buffer, opts, eth, ipReply, tcpReply, gopacket.Payload(response))
	if err != nil {
		if debugLevel >= 1 {
			_, _ = fmt.Fprintf(os.Stdout, "HTTP/IPv6: Failed to serialize response: %v\n", err)
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
		_, _ = fmt.Fprintf(os.Stdout, "Sent HTTP/IPv6 response: %d bytes from [%s] to [%s] (device: %s)\n",
			len(response), ipReply.SrcIP, ipReply.DstIP, device.Name)
	}
}
