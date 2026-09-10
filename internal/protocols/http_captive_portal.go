package protocols

import (
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

const captivePortalPath = "/niac-portal"

// Both IP transports use this selection before framing their TCP reply.
func (h *HTTPHandler) generateResponse(request *HTTPRequest, devices []*config.Device) []byte {
	info := getHTTPDeviceInfo(devices)
	if h.stack.deviceFaultActive(info.device, devicestate.FaultCaptivePortal) {
		return captivePortalResponse(request)
	}
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

func captivePortalResponse(request *HTTPRequest) []byte {
	if request.Path != captivePortalPath {
		return []byte("HTTP/1.1 302 Found\r\nLocation: " + captivePortalPath +
			"\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	}
	const body = "<!DOCTYPE html><html><head><title>Captive portal</title></head>" +
		"<body><h1>Captive portal</h1><p>This simulated network requires portal access.</p>" +
		"<p>No sign-in information is collected.</p></body></html>"
	response := buildHTTPResponse(httpStatusOK, "NIAC", contentTypeHTML, body)
	if request.Method == "HEAD" {
		return response[:len(response)-len(body)]
	}
	return response
}
