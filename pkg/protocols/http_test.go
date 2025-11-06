package protocols

import (
	"strings"
	"testing"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// TestNewHTTPHandler tests HTTP handler creation
func TestNewHTTPHandler(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewHTTPHandler(stack)

	if handler == nil {
		t.Fatal("NewHTTPHandler returned nil")
	}

	if handler.stack != stack {
		t.Error("Handler stack not set correctly")
	}
}

// TestParseHTTPRequest tests HTTP request parsing
func TestParseHTTPRequest(t *testing.T) {
	tests := []struct {
		name            string
		payload         string
		expectError     bool
		expectedMethod  string
		expectedPath    string
		expectedVersion string
	}{
		{
			name: "GET request",
			payload: "GET /index.html HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"User-Agent: TestClient\r\n" +
				"\r\n",
			expectError:     false,
			expectedMethod:  "GET",
			expectedPath:    "/index.html",
			expectedVersion: "HTTP/1.1",
		},
		{
			name: "POST request",
			payload: "POST /api/data HTTP/1.1\r\n" +
				"Host: example.com\r\n" +
				"Content-Type: application/json\r\n" +
				"\r\n",
			expectError:     false,
			expectedMethod:  "POST",
			expectedPath:    "/api/data",
			expectedVersion: "HTTP/1.1",
		},
		{
			name: "Simple GET",
			payload: "GET / HTTP/1.1\r\n" +
				"\r\n",
			expectError:     false,
			expectedMethod:  "GET",
			expectedPath:    "/",
			expectedVersion: "HTTP/1.1",
		},
		{
			name:        "Invalid request line",
			payload:     "INVALID REQUEST\r\n\r\n",
			expectError: true,
		},
		{
			name:        "Empty payload",
			payload:     "",
			expectError: true,
		},
		{
			name: "HTTP/1.0 request",
			payload: "GET /page.html HTTP/1.0\r\n" +
				"\r\n",
			expectError:     false,
			expectedMethod:  "GET",
			expectedPath:    "/page.html",
			expectedVersion: "HTTP/1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := parseHTTPRequest([]byte(tt.payload))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if request.Method != tt.expectedMethod {
				t.Errorf("Expected method %s, got %s", tt.expectedMethod, request.Method)
			}

			if request.Path != tt.expectedPath {
				t.Errorf("Expected path %s, got %s", tt.expectedPath, request.Path)
			}

			if request.Version != tt.expectedVersion {
				t.Errorf("Expected version %s, got %s", tt.expectedVersion, request.Version)
			}
		})
	}
}

// TestParseHTTPRequest_Headers tests header parsing
func TestParseHTTPRequest_Headers(t *testing.T) {
	payload := "GET /test HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"User-Agent: TestClient/1.0\r\n" +
		"Accept: text/html\r\n" +
		"Connection: keep-alive\r\n" +
		"\r\n"

	request, err := parseHTTPRequest([]byte(payload))
	if err != nil {
		t.Fatalf("Failed to parse request: %v", err)
	}

	expectedHeaders := map[string]string{
		"Host":       "example.com",
		"User-Agent": "TestClient/1.0",
		"Accept":     "text/html",
		"Connection": "keep-alive",
	}

	for key, expectedValue := range expectedHeaders {
		if value, ok := request.Headers[key]; !ok {
			t.Errorf("Header %s not found", key)
		} else if value != expectedValue {
			t.Errorf("Header %s: expected %s, got %s", key, expectedValue, value)
		}
	}
}

// TestParseHTTPRequest_Methods tests different HTTP methods
func TestParseHTTPRequest_Methods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH"}

	for _, method := range methods {
		payload := method + " /test HTTP/1.1\r\n\r\n"
		request, err := parseHTTPRequest([]byte(payload))

		if err != nil {
			t.Errorf("Failed to parse %s request: %v", method, err)
			continue
		}

		if request.Method != method {
			t.Errorf("Expected method %s, got %s", method, request.Method)
		}
	}
}

// TestGenerateResponse_DefaultEndpoints tests default HTTP endpoints
func TestGenerateResponse_DefaultEndpoints(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewHTTPHandler(stack)

	devices := []*config.Device{
		{
			Name: "test-router",
			Type: "router",
		},
	}

	tests := []struct {
		name               string
		path               string
		expectedStatusCode int
		expectedContains   []string
	}{
		{
			name:               "Root path",
			path:               "/",
			expectedStatusCode: 200,
			expectedContains:   []string{"test-router", "router", "NIAC-Go"},
		},
		{
			name:               "Index.html",
			path:               "/index.html",
			expectedStatusCode: 200,
			expectedContains:   []string{"test-router", "router", "NIAC-Go"},
		},
		{
			name:               "Status endpoint",
			path:               "/status",
			expectedStatusCode: 200,
			expectedContains:   []string{"Device Status", "test-router", "router", "Statistics"},
		},
		{
			name:               "API info endpoint",
			path:               "/api/info",
			expectedStatusCode: 200,
			expectedContains:   []string{`"name"`, `"type"`, `"test-router"`, `"router"`},
		},
		{
			name:               "Not found",
			path:               "/nonexistent",
			expectedStatusCode: 404,
			expectedContains:   []string{"404", "Not Found", "/nonexistent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &HTTPRequest{
				Method:  "GET",
				Path:    tt.path,
				Version: "HTTP/1.1",
			}

			response := handler.generateResponse(request, devices)
			responseStr := string(response)

			// Check status code
			if !strings.Contains(responseStr, "HTTP/1.1 "+string(rune('0'+tt.expectedStatusCode/100))+string(rune('0'+(tt.expectedStatusCode/10)%10))+string(rune('0'+tt.expectedStatusCode%10))) {
				if !strings.Contains(responseStr, "200") && tt.expectedStatusCode == 200 {
					t.Errorf("Expected status code %d not found in response", tt.expectedStatusCode)
				}
				if !strings.Contains(responseStr, "404") && tt.expectedStatusCode == 404 {
					t.Errorf("Expected status code %d not found in response", tt.expectedStatusCode)
				}
			}

			// Check expected content
			for _, expected := range tt.expectedContains {
				if !strings.Contains(responseStr, expected) {
					t.Errorf("Expected response to contain '%s'", expected)
				}
			}
		})
	}
}

// TestGenerateResponse_CustomEndpoints tests custom configured endpoints
func TestGenerateResponse_CustomEndpoints(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewHTTPHandler(stack)

	devices := []*config.Device{
		{
			Name: "test-device",
			Type: "switch",
			HTTPConfig: &config.HTTPConfig{
				Enabled:    true,
				ServerName: "CustomServer/2.0",
				Endpoints: []config.HTTPEndpoint{
					{
						Path:        "/custom",
						Method:      "GET",
						StatusCode:  200,
						ContentType: "text/plain",
						Body:        "Custom endpoint response",
					},
					{
						Path:        "/api/custom",
						Method:      "POST",
						StatusCode:  201,
						ContentType: "application/json",
						Body:        `{"status":"created"}`,
					},
					{
						Path:        "/redirect",
						StatusCode:  302,
						ContentType: "text/html",
						Body:        "Redirecting...",
					},
				},
			},
		},
	}

	tests := []struct {
		name               string
		method             string
		path               string
		expectedStatusCode int
		expectedContent    string
	}{
		{
			name:               "Custom GET endpoint",
			method:             "GET",
			path:               "/custom",
			expectedStatusCode: 200,
			expectedContent:    "Custom endpoint response",
		},
		{
			name:               "Custom POST endpoint",
			method:             "POST",
			path:               "/api/custom",
			expectedStatusCode: 201,
			expectedContent:    `"status":"created"`,
		},
		{
			name:               "Redirect endpoint",
			method:             "GET",
			path:               "/redirect",
			expectedStatusCode: 302,
			expectedContent:    "Redirecting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &HTTPRequest{
				Method:  tt.method,
				Path:    tt.path,
				Version: "HTTP/1.1",
			}

			response := handler.generateResponse(request, devices)
			responseStr := string(response)

			if !strings.Contains(responseStr, tt.expectedContent) {
				t.Errorf("Expected response to contain '%s'", tt.expectedContent)
			}
		})
	}
}

// TestGenerateResponse_ServerName tests custom server name
func TestGenerateResponse_ServerName(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewHTTPHandler(stack)

	tests := []struct {
		name               string
		serverName         string
		expectedServerName string
	}{
		{
			name:               "Default server name",
			serverName:         "",
			expectedServerName: "NIAC-Go/1.0.0",
		},
		{
			name:               "Custom server name",
			serverName:         "Apache/2.4.41",
			expectedServerName: "Apache/2.4.41",
		},
		{
			name:               "nginx server",
			serverName:         "nginx/1.18.0",
			expectedServerName: "nginx/1.18.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devices := []*config.Device{
				{
					Name: "test",
					HTTPConfig: &config.HTTPConfig{
						Enabled:    true,
						ServerName: tt.serverName,
					},
				},
			}

			request := &HTTPRequest{
				Method:  "GET",
				Path:    "/",
				Version: "HTTP/1.1",
			}

			response := handler.generateResponse(request, devices)
			responseStr := string(response)

			if !strings.Contains(responseStr, "Server: "+tt.expectedServerName) {
				t.Errorf("Expected Server header: %s", tt.expectedServerName)
			}
		})
	}
}

// TestGenerateResponse_Headers tests HTTP response headers
func TestGenerateResponse_Headers(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewHTTPHandler(stack)

	devices := []*config.Device{
		{
			Name: "test-device",
		},
	}

	request := &HTTPRequest{
		Method:  "GET",
		Path:    "/",
		Version: "HTTP/1.1",
	}

	response := handler.generateResponse(request, devices)
	responseStr := string(response)

	requiredHeaders := []string{
		"HTTP/1.1",
		"Date:",
		"Server:",
		"Content-Type:",
		"Content-Length:",
		"Connection: close",
	}

	for _, header := range requiredHeaders {
		if !strings.Contains(responseStr, header) {
			t.Errorf("Expected response to contain header: %s", header)
		}
	}
}

// TestGenerateResponse_ContentTypes tests different content types
func TestGenerateResponse_ContentTypes(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewHTTPHandler(stack)

	devices := []*config.Device{
		{
			Name: "test",
			HTTPConfig: &config.HTTPConfig{
				Enabled: true,
				Endpoints: []config.HTTPEndpoint{
					{
						Path:        "/html",
						ContentType: "text/html",
						Body:        "<html></html>",
					},
					{
						Path:        "/json",
						ContentType: "application/json",
						Body:        `{"key":"value"}`,
					},
					{
						Path:        "/plain",
						ContentType: "text/plain",
						Body:        "Plain text",
					},
					{
						Path:        "/xml",
						ContentType: "application/xml",
						Body:        "<root></root>",
					},
				},
			},
		},
	}

	tests := []struct {
		path        string
		contentType string
	}{
		{"/html", "text/html"},
		{"/json", "application/json"},
		{"/plain", "text/plain"},
		{"/xml", "application/xml"},
		{"/api/info", "application/json"}, // Default endpoint
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			request := &HTTPRequest{
				Method:  "GET",
				Path:    tt.path,
				Version: "HTTP/1.1",
			}

			response := handler.generateResponse(request, devices)
			responseStr := string(response)

			if !strings.Contains(responseStr, "Content-Type: "+tt.contentType) {
				t.Errorf("Expected Content-Type: %s", tt.contentType)
			}
		})
	}
}

// TestGetStatusText tests HTTP status code to text mapping
func TestGetStatusText(t *testing.T) {
	tests := []struct {
		code         int
		expectedText string
	}{
		{200, "OK"},
		{201, "Created"},
		{204, "No Content"},
		{301, "Moved Permanently"},
		{302, "Found"},
		{304, "Not Modified"},
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{500, "Internal Server Error"},
		{501, "Not Implemented"},
		{503, "Service Unavailable"},
		{999, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedText, func(t *testing.T) {
			result := getStatusText(tt.code)
			if result != tt.expectedText {
				t.Errorf("For code %d, expected '%s', got '%s'", tt.code, tt.expectedText, result)
			}
		})
	}
}

// TestGetDeviceNames tests device name formatting
func TestGetDeviceNames(t *testing.T) {
	tests := []struct {
		name     string
		devices  []*config.Device
		expected string
	}{
		{
			name:     "No devices",
			devices:  []*config.Device{},
			expected: "unknown",
		},
		{
			name: "Single device",
			devices: []*config.Device{
				{Name: "router1"},
			},
			expected: "router1",
		},
		{
			name: "Multiple devices",
			devices: []*config.Device{
				{Name: "router1"},
				{Name: "switch1"},
				{Name: "server1"},
			},
			expected: "router1, switch1, server1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDeviceNames(tt.devices)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestGenerateResponse_MethodDefaulting tests method defaulting for endpoints
func TestGenerateResponse_MethodDefaulting(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewHTTPHandler(stack)

	devices := []*config.Device{
		{
			Name: "test",
			HTTPConfig: &config.HTTPConfig{
				Enabled: true,
				Endpoints: []config.HTTPEndpoint{
					{
						Path: "/no-method",
						// Method not specified, should default to GET
						Body: "No method specified",
					},
				},
			},
		},
	}

	// Should match with GET request
	request := &HTTPRequest{
		Method:  "GET",
		Path:    "/no-method",
		Version: "HTTP/1.1",
	}

	response := handler.generateResponse(request, devices)
	responseStr := string(response)

	if !strings.Contains(responseStr, "No method specified") {
		t.Error("Expected endpoint to match GET request when no method specified")
	}
}

// TestGenerateResponse_StatusCodeDefaulting tests status code defaulting
func TestGenerateResponse_StatusCodeDefaulting(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewHTTPHandler(stack)

	devices := []*config.Device{
		{
			Name: "test",
			HTTPConfig: &config.HTTPConfig{
				Enabled: true,
				Endpoints: []config.HTTPEndpoint{
					{
						Path: "/no-status",
						// StatusCode not specified, should default to 200
						Body: "No status specified",
					},
				},
			},
		},
	}

	request := &HTTPRequest{
		Method:  "GET",
		Path:    "/no-status",
		Version: "HTTP/1.1",
	}

	response := handler.generateResponse(request, devices)
	responseStr := string(response)

	if !strings.Contains(responseStr, "HTTP/1.1 200") {
		t.Error("Expected status code 200 when not specified")
	}
}

// TestGenerateResponse_EmptyDevices tests behavior with no devices
func TestGenerateResponse_EmptyDevices(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewHTTPHandler(stack)

	request := &HTTPRequest{
		Method:  "GET",
		Path:    "/",
		Version: "HTTP/1.1",
	}

	response := handler.generateResponse(request, []*config.Device{})
	responseStr := string(response)

	// Should still generate a response with "Unknown" as device name
	if !strings.Contains(responseStr, "HTTP/1.1") {
		t.Error("Expected valid HTTP response even with no devices")
	}

	if !strings.Contains(responseStr, "Unknown") {
		t.Error("Expected 'Unknown' device name when no devices configured")
	}
}

// Benchmarks

// BenchmarkParseHTTPRequest benchmarks HTTP request parsing
func BenchmarkParseHTTPRequest(b *testing.B) {
	payload := []byte("GET /index.html HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"User-Agent: Benchmark\r\n" +
		"Accept: text/html\r\n" +
		"\r\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseHTTPRequest(payload)
	}
}

// BenchmarkGenerateResponse benchmarks response generation
func BenchmarkGenerateResponse(b *testing.B) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewHTTPHandler(stack)

	devices := []*config.Device{
		{
			Name: "benchmark-device",
			Type: "router",
		},
	}

	request := &HTTPRequest{
		Method:  "GET",
		Path:    "/",
		Version: "HTTP/1.1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.generateResponse(request, devices)
	}
}

// BenchmarkGetStatusText benchmarks status text lookup
func BenchmarkGetStatusText(b *testing.B) {
	codes := []int{200, 404, 500, 301, 403}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getStatusText(codes[i%len(codes)])
	}
}
