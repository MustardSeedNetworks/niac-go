package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// systemTempDir returns the system temporary directory.
// This is a helper to avoid usetesting linter flagging [os.TempDir] in tests
// where we need the actual system temp directory, not a test-specific one.
func systemTempDir() string {
	return os.TempDir()
}

// TestDefaultSocketPath tests the DefaultSocketPath function.
func TestDefaultSocketPath(t *testing.T) {
	path := DefaultSocketPath()
	if path == "" {
		t.Error("DefaultSocketPath returned empty string")
	}

	expected := filepath.Join(systemTempDir(), "niac.sock")
	if path != expected {
		t.Errorf("DefaultSocketPath = %q, want %q", path, expected)
	}
}

// TestGetDefaultSocketPath tests the GetDefaultSocketPath function with env var.
func TestGetDefaultSocketPath(t *testing.T) {
	// Test without env var
	t.Setenv(EnvSocketPath, "")

	path := GetDefaultSocketPath()

	expected := filepath.Join(systemTempDir(), "niac.sock")

	if path != expected {
		t.Errorf("GetDefaultSocketPath = %q, want %q", path, expected)
	}

	// Test with env var
	customPath := "/custom/path/niac.sock"
	t.Setenv(EnvSocketPath, customPath)

	path = GetDefaultSocketPath()
	if path != customPath {
		t.Errorf("GetDefaultSocketPath with env var = %q, want %q", path, customPath)
	}
}

// TestNewClient tests client creation.
func TestNewClient(t *testing.T) {
	t.Run("with custom path", func(t *testing.T) {
		customPath := "/tmp/test.sock"
		client := NewClient(customPath)

		if client.SocketPath() != customPath {
			t.Errorf("SocketPath = %q, want %q", client.SocketPath(), customPath)
		}
	})

	t.Run("with empty path uses default", func(t *testing.T) {
		client := NewClient("")
		expected := DefaultSocketPath()

		if client.SocketPath() != expected {
			t.Errorf("SocketPath = %q, want %q", client.SocketPath(), expected)
		}
	})

	t.Run("default client", func(t *testing.T) {
		client := DefaultClient()
		if client == nil {
			t.Error("DefaultClient returned nil")
		}
	})
}

// TestClientSetTimeout tests timeout configuration.
func TestClientSetTimeout(t *testing.T) {
	client := NewClient("/tmp/test.sock")
	newTimeout := 10 * time.Second
	client.SetTimeout(newTimeout)

	// We can't directly access timeout, but we can verify the client still works
	if client == nil {
		t.Error("SetTimeout should not invalidate client")
	}
}

// TestClientConnectionRefused tests behavior when server is not running.
func TestClientConnectionRefused(t *testing.T) {
	// Use a socket path that doesn't exist
	socketPath := filepath.Join(t.TempDir(), "nonexistent_test.sock")

	client := NewClient(socketPath)
	client.SetTimeout(100 * time.Millisecond)

	t.Run("SendCommand returns error", func(t *testing.T) {
		_, err := client.SendCommand(CommandStatus, nil)
		if err == nil {
			t.Error("SendCommand should fail when server not running")
		}
	})

	t.Run("GetStatus returns error", func(t *testing.T) {
		_, err := client.GetStatus()
		if err == nil {
			t.Error("GetStatus should fail when server not running")
		}
	})

	t.Run("IsRunning returns false", func(t *testing.T) {
		if client.IsRunning() {
			t.Error("IsRunning should return false when server not running")
		}
	})

	t.Run("Ping returns error", func(t *testing.T) {
		err := client.Ping()
		if err == nil {
			t.Error("Ping should fail when server not running")
		}
	})

	t.Run("Reload returns error", func(t *testing.T) {
		err := client.Reload()
		if err == nil {
			t.Error("Reload should fail when server not running")
		}
	})

	t.Run("InjectError returns error", func(t *testing.T) {
		err := client.InjectError("device1", "FCS Errors", 50)
		if err == nil {
			t.Error("InjectError should fail when server not running")
		}
	})

	t.Run("ListInjections returns error", func(t *testing.T) {
		_, err := client.ListInjections()
		if err == nil {
			t.Error("ListInjections should fail when server not running")
		}
	})

	t.Run("ClearInjections returns error", func(t *testing.T) {
		err := client.ClearInjections("")
		if err == nil {
			t.Error("ClearInjections should fail when server not running")
		}
	})

	t.Run("Shutdown returns error", func(t *testing.T) {
		err := client.Shutdown()
		if err == nil {
			t.Error("Shutdown should fail when server not running")
		}
	})
}

// mockServer creates a mock IPC server for testing.
type mockServer struct {
	listener   net.Listener
	socketPath string
	handler    func(*Request) *Response
	done       chan struct{}
}

func newMockServer(t *testing.T, handler func(*Request) *Response) *mockServer {
	t.Helper()

	// Create socket in /tmp with a unique name because t.TempDir() creates paths that are
	// too long for Unix domain sockets (macOS has a 104-byte limit on socket paths)
	socketPath := fmt.Sprintf("/tmp/niac_test_%d.sock", time.Now().UnixNano())
	_ = os.RemoveAll(socketPath)

	t.Cleanup(func() {
		_ = os.RemoveAll(socketPath)
	})

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to create test socket: %v", err)
	}

	ms := &mockServer{
		listener:   listener,
		socketPath: socketPath,
		handler:    handler,
		done:       make(chan struct{}),
	}

	go ms.acceptLoop()

	return ms
}

func (ms *mockServer) acceptLoop() {
	for {
		conn, err := ms.listener.Accept()
		if err != nil {
			select {
			case <-ms.done:
				return
			default:
				continue
			}
		}

		go ms.handleConnection(conn)
	}
}

func (ms *mockServer) handleConnection(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	var req Request

	decoder := json.NewDecoder(conn)
	err := decoder.Decode(&req)
	if err != nil {
		return
	}

	resp := ms.handler(&req)

	encoder := json.NewEncoder(conn)
	_ = encoder.Encode(resp)
}

func (ms *mockServer) Close() {
	close(ms.done)
	_ = ms.listener.Close()
	_ = os.RemoveAll(ms.socketPath)
}

// Mock response helpers for tests.
func makeStatusResponse(running bool, iface string, deviceCount int) *Response {
	return &Response{
		Success: true,
		Data: map[string]any{
			"status": map[string]any{
				"running":          running,
				"interface":        iface,
				"config_path":      "/etc/niac/config.yaml",
				"device_count":     deviceCount,
				"uptime_seconds":   123.45,
				"started_at":       time.Now().Format(time.RFC3339),
				"packets_received": 1000,
				"packets_sent":     500,
				"errors_active":    2,
			},
		},
	}
}

func makeSimpleStatusResponse() *Response {
	return &Response{
		Success: true,
		Data: map[string]any{
			"status": map[string]any{
				"running":          true,
				"interface":        "eth0",
				"config_path":      "/etc/niac/config.yaml",
				"device_count":     1,
				"uptime_seconds":   1.0,
				"started_at":       time.Now().Format(time.RFC3339),
				"packets_received": 0,
				"packets_sent":     0,
				"errors_active":    0,
			},
		},
	}
}

func makeSuccessResponse(data map[string]any) *Response {
	return &Response{Success: true, Data: data}
}

func makeErrorResponse(msg string) *Response {
	return &Response{Success: false, Error: msg}
}

// TestGetStatusSuccess tests the GetStatus command with a mock server.
func TestGetStatusSuccess(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandStatus {
			return makeErrorResponse("unexpected command")
		}

		return makeStatusResponse(true, "eth0", 5)
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	status, err := client.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if !status.Running {
		t.Error("Expected Running to be true")
	}

	if status.Interface != "eth0" {
		t.Errorf("Interface = %q, want %q", status.Interface, "eth0")
	}

	if status.DeviceCount != 5 {
		t.Errorf("DeviceCount = %d, want %d", status.DeviceCount, 5)
	}
}

// TestReloadSuccess tests the Reload command with a mock server.
func TestReloadSuccess(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandReload {
			return makeErrorResponse("unexpected command")
		}

		return makeSuccessResponse(map[string]any{
			"message":      "configuration reloaded",
			"device_count": 10,
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	err := client.Reload()
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
}

// TestInjectErrorSuccess tests the InjectError command with a mock server.
func TestInjectErrorSuccess(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandInject {
			return makeErrorResponse("unexpected command")
		}

		device := req.Args["device"].(string)
		errorType := req.Args["error_type"].(string)
		value := req.Args["value"].(float64)

		if device != "router1" || errorType != "FCS Errors" || int(value) != 50 {
			return makeErrorResponse("unexpected arguments")
		}

		return makeSuccessResponse(map[string]any{
			"message":    "error injected",
			"device":     device,
			"error_type": errorType,
			"value":      int(value),
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	err := client.InjectError("router1", "FCS Errors", 50)
	if err != nil {
		t.Fatalf("InjectError failed: %v", err)
	}
}

// TestListInjectionsSuccess tests the ListInjections command with a mock server.
func TestListInjectionsSuccess(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandList {
			return makeErrorResponse("unexpected command")
		}

		return makeSuccessResponse(map[string]any{
			"injections": []map[string]any{
				{
					"device":      "router1",
					"interface":   "eth0",
					"error_type":  "FCS Errors",
					"value":       50,
					"injected_at": time.Now().Format(time.RFC3339),
				},
				{
					"device":      "switch1",
					"interface":   "eth1",
					"error_type":  "Packet Discards",
					"value":       25,
					"injected_at": time.Now().Format(time.RFC3339),
				},
			},
			"count": 2,
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	injections, err := client.ListInjections()
	if err != nil {
		t.Fatalf("ListInjections failed: %v", err)
	}

	if len(injections) != 2 {
		t.Errorf("len(injections) = %d, want %d", len(injections), 2)
	}

	if injections[0].Device != "router1" {
		t.Errorf("injections[0].Device = %q, want %q", injections[0].Device, "router1")
	}
}

// TestClearInjectionsAll tests clearing all injections with a mock server.
func TestClearInjectionsAll(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandClear {
			return makeErrorResponse("unexpected command")
		}

		if req.Args != nil {
			return makeErrorResponse("expected no args for clear all")
		}

		return makeSuccessResponse(map[string]any{
			"message": "all error injections cleared",
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	err := client.ClearInjections("")
	if err != nil {
		t.Fatalf("ClearInjections failed: %v", err)
	}
}

// TestClearInjectionsDevice tests clearing injections for a specific device.
func TestClearInjectionsDevice(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandClear {
			return makeErrorResponse("unexpected command")
		}

		device, ok := req.Args["device"].(string)
		if !ok || device != "router1" {
			return makeErrorResponse("expected device=router1")
		}

		return makeSuccessResponse(map[string]any{
			"message": "cleared 1 injections for device router1",
			"cleared": 1,
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	err := client.ClearInjections("router1")
	if err != nil {
		t.Fatalf("ClearInjections(router1) failed: %v", err)
	}
}

// TestShutdownSuccess tests the Shutdown command with a mock server.
func TestShutdownSuccess(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandShutdown {
			return makeErrorResponse("unexpected command")
		}

		return makeSuccessResponse(map[string]any{
			"message": "shutdown initiated",
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	err := client.Shutdown()
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

// TestPingSuccess tests the Ping command with a mock server.
func TestPingSuccess(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeSimpleStatusResponse()
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	err := client.Ping()
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

// TestIsRunningTrue tests IsRunning returns true when server reports running.
func TestIsRunningTrue(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeSimpleStatusResponse()
	})
	defer server.Close()

	client := NewClient(server.socketPath)
	if !client.IsRunning() {
		t.Error("IsRunning should return true")
	}
}

// TestClientServerErrors tests client handling of server errors.
func TestClientServerErrors(t *testing.T) {
	t.Run("server returns error response", func(t *testing.T) {
		server := newMockServer(t, func(_ *Request) *Response {
			return &Response{
				Success: false,
				Error:   "something went wrong",
			}
		})
		defer server.Close()

		client := NewClient(server.socketPath)

		_, err := client.GetStatus()
		if err == nil {
			t.Error("GetStatus should return error when server returns error")
		}

		if err.Error() != "status command failed: something went wrong" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("server returns malformed status response", func(t *testing.T) {
		server := newMockServer(t, func(_ *Request) *Response {
			return &Response{
				Success: true,
				Data:    map[string]any{}, // missing "status" key
			}
		})
		defer server.Close()

		client := NewClient(server.socketPath)

		_, err := client.GetStatus()
		if err == nil {
			t.Error("GetStatus should return error when response missing status data")
		}
	})

	t.Run("server returns malformed list response", func(t *testing.T) {
		server := newMockServer(t, func(_ *Request) *Response {
			return &Response{
				Success: true,
				Data:    map[string]any{}, // missing "injections" key
			}
		})
		defer server.Close()

		client := NewClient(server.socketPath)

		_, err := client.ListInjections()
		if err == nil {
			t.Error("ListInjections should return error when response missing injections data")
		}
	})
}

// TestRequestResponseTypes tests the Request and Response types.
func TestRequestResponseTypes(t *testing.T) {
	t.Run("Request JSON marshaling", func(t *testing.T) {
		req := Request{
			Command: CommandInject,
			Args: map[string]any{
				"device":     "router1",
				"error_type": "FCS Errors",
				"value":      50,
			},
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal Request: %v", err)
		}

		var decoded Request
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal Request: %v", err)
		}

		if decoded.Command != CommandInject {
			t.Errorf("decoded.Command = %q, want %q", decoded.Command, CommandInject)
		}
	})

	t.Run("Response JSON marshaling", func(t *testing.T) {
		resp := Response{
			Success: true,
			Data: map[string]any{
				"message": "test",
			},
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("Failed to marshal Response: %v", err)
		}

		var decoded Response
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal Response: %v", err)
		}

		if !decoded.Success {
			t.Error("decoded.Success should be true")
		}
	})
}

// TestStatusData tests the StatusData type.
func TestStatusData(t *testing.T) {
	now := time.Now()
	status := StatusData{
		Running:      true,
		Interface:    "eth0",
		ConfigPath:   "/etc/niac/config.yaml",
		DeviceCount:  5,
		Uptime:       123.45,
		StartedAt:    now,
		PacketsRX:    1000,
		PacketsTX:    500,
		ErrorsActive: 2,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal StatusData: %v", err)
	}

	var decoded StatusData
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal StatusData: %v", err)
	}

	if decoded.DeviceCount != 5 {
		t.Errorf("decoded.DeviceCount = %d, want %d", decoded.DeviceCount, 5)
	}

	if decoded.PacketsRX != 1000 {
		t.Errorf("decoded.PacketsRX = %d, want %d", decoded.PacketsRX, 1000)
	}
}

// TestErrorInjectionData tests the ErrorInjectionData type.
func TestErrorInjectionData(t *testing.T) {
	now := time.Now()
	injection := ErrorInjectionData{
		Device:    "router1",
		Interface: "eth0",
		ErrorType: "FCS Errors",
		Value:     50,
		Injected:  now,
	}

	data, err := json.Marshal(injection)
	if err != nil {
		t.Fatalf("Failed to marshal ErrorInjectionData: %v", err)
	}

	var decoded ErrorInjectionData
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal ErrorInjectionData: %v", err)
	}

	if decoded.Device != "router1" {
		t.Errorf("decoded.Device = %q, want %q", decoded.Device, "router1")
	}

	if decoded.Value != 50 {
		t.Errorf("decoded.Value = %d, want %d", decoded.Value, 50)
	}
}

// TestCommands tests the Command constants.
func TestCommands(t *testing.T) {
	commands := []Command{
		CommandStatus,
		CommandReload,
		CommandInject,
		CommandList,
		CommandClear,
		CommandShutdown,
	}

	expectedStrings := []string{
		"status",
		"reload",
		"inject",
		"list",
		"clear",
		"shutdown",
	}

	for i, cmd := range commands {
		if string(cmd) != expectedStrings[i] {
			t.Errorf("Command %d = %q, want %q", i, string(cmd), expectedStrings[i])
		}
	}
}

// TestContainsIgnoreCase tests the containsIgnoreCase function.
func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "Exact match",
			s:      "Hello World",
			substr: "Hello World",
			want:   true,
		},
		{
			name:   "Case insensitive match",
			s:      "Hello World",
			substr: "HELLO",
			want:   true,
		},
		{
			name:   "Lowercase search in mixed case",
			s:      "HELLO WORLD",
			substr: "world",
			want:   true,
		},
		{
			name:   "Substring match",
			s:      "Hello World",
			substr: "lo Wo",
			want:   true,
		},
		{
			name:   "No match",
			s:      "Hello World",
			substr: "xyz",
			want:   false,
		},
		{
			name:   "Empty substring",
			s:      "Hello World",
			substr: "",
			want:   true,
		},
		{
			name:   "Empty string",
			s:      "",
			substr: "test",
			want:   false,
		},
		{
			name:   "Both empty",
			s:      "",
			substr: "",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsIgnoreCase(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

// TestBytesContains tests the bytesContains function.
func TestBytesContains(t *testing.T) {
	tests := []struct {
		name string
		b    []byte
		sub  []byte
		want bool
	}{
		{
			name: "Contains at start",
			b:    []byte("hello world"),
			sub:  []byte("hello"),
			want: true,
		},
		{
			name: "Contains at end",
			b:    []byte("hello world"),
			sub:  []byte("world"),
			want: true,
		},
		{
			name: "Contains in middle",
			b:    []byte("hello world"),
			sub:  []byte("lo wo"),
			want: true,
		},
		{
			name: "Does not contain",
			b:    []byte("hello world"),
			sub:  []byte("xyz"),
			want: false,
		},
		{
			name: "Empty sub",
			b:    []byte("hello"),
			sub:  []byte{},
			want: true,
		},
		{
			name: "Sub longer than b",
			b:    []byte("hi"),
			sub:  []byte("hello"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bytesContains(tt.b, tt.sub)
			if got != tt.want {
				t.Errorf("bytesContains(%q, %q) = %v, want %v", tt.b, tt.sub, got, tt.want)
			}
		})
	}
}

// TestMatchesFilter tests the matchesFilter function.
func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		name    string
		message string
		filter  string
		want    bool
	}{
		{
			name:    "Empty filter matches anything",
			message: "any message",
			filter:  "",
			want:    true,
		},
		{
			name:    "Filter matches",
			message: "Error: connection refused",
			filter:  "connection",
			want:    true,
		},
		{
			name:    "Filter does not match",
			message: "Success: connected",
			filter:  "error",
			want:    false,
		},
		{
			name:    "Case insensitive filter",
			message: "ERROR: something wrong",
			filter:  "error",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesFilter(tt.message, tt.filter)
			if got != tt.want {
				t.Errorf("matchesFilter(%q, %q) = %v, want %v", tt.message, tt.filter, got, tt.want)
			}
		})
	}
}

// TestLogSubscription tests the LogSubscription type.
func TestLogSubscription(t *testing.T) {
	t.Run("Logs channel accessible", func(t *testing.T) {
		sub := &LogSubscription{
			logCh: make(chan LogEntry, 10),
		}
		ch := sub.Logs()
		if ch == nil {
			t.Error("Logs() returned nil channel")
		}
	})

	t.Run("Errors channel accessible", func(t *testing.T) {
		sub := &LogSubscription{
			errCh: make(chan error, 1),
		}
		ch := sub.Errors()
		if ch == nil {
			t.Error("Errors() returned nil channel")
		}
	})

	t.Run("Stop closes stopCh", func(t *testing.T) {
		sub := &LogSubscription{
			stopCh: make(chan struct{}),
		}
		sub.Stop()
		select {
		case <-sub.stopCh:
			// expected
		default:
			t.Error("Stop() should close stopCh")
		}
	})
}

// TestLogSubscriptionHelpers tests helper functions of LogSubscription.
func TestLogSubscriptionHelpers(t *testing.T) {
	t.Run("logKey generates unique keys", func(t *testing.T) {
		sub := &LogSubscription{}
		log1 := LogEntry{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "test message",
		}
		log2 := LogEntry{
			Timestamp: time.Now().Add(time.Second),
			Level:     "error",
			Message:   "another message",
		}

		key1 := sub.logKey(log1)
		key2 := sub.logKey(log2)

		if key1 == key2 {
			t.Error("Different logs should have different keys")
		}
	})

	t.Run("cleanupSeenLogs clears when over max", func(t *testing.T) {
		sub := &LogSubscription{}
		seenLogs := make(map[string]bool)

		// Add more than maxSeenLogs entries
		for i := range 1100 {
			seenLogs[fmt.Sprintf("key%d", i)] = true
		}

		result := sub.cleanupSeenLogs(seenLogs)
		if len(result) != 0 {
			t.Errorf("Expected empty map after cleanup, got %d entries", len(result))
		}
	})

	t.Run("cleanupSeenLogs keeps when under max", func(t *testing.T) {
		sub := &LogSubscription{}
		seenLogs := make(map[string]bool)

		for i := range 100 {
			seenLogs[fmt.Sprintf("key%d", i)] = true
		}

		result := sub.cleanupSeenLogs(seenLogs)
		if len(result) != 100 {
			t.Errorf("Expected 100 entries, got %d", len(result))
		}
	})

	t.Run("shouldSkipLog skips seen logs", func(t *testing.T) {
		sub := &LogSubscription{}
		log := LogEntry{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "test",
		}
		seenLogs := map[string]bool{
			sub.logKey(log): true,
		}

		if !sub.shouldSkipLog(log, seenLogs) {
			t.Error("Should skip already seen log")
		}
	})

	t.Run("shouldSkipLog respects filter", func(t *testing.T) {
		sub := &LogSubscription{
			filter: "error",
		}
		log := LogEntry{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "success message",
		}
		seenLogs := make(map[string]bool)

		if !sub.shouldSkipLog(log, seenLogs) {
			t.Error("Should skip log that doesn't match filter")
		}
	})
}

// TestDumpPackets tests the DumpPackets method with a mock server.
func TestDumpPackets(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandDump {
			return makeErrorResponse("unexpected command")
		}

		return makeSuccessResponse(map[string]any{
			"packets": []map[string]any{
				{
					"timestamp": time.Now().Format(time.RFC3339),
					"src_mac":   "00:11:22:33:44:55",
					"dst_mac":   "66:77:88:99:AA:BB",
					"eth_type":  0x0800,
					"length":    64,
					"device":    "router1",
				},
			},
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	packets, err := client.DumpPackets("", "", 10)
	if err != nil {
		t.Fatalf("DumpPackets failed: %v", err)
	}

	if len(packets) != 1 {
		t.Errorf("Expected 1 packet, got %d", len(packets))
	}
}

// TestGetTopology tests the GetTopology method with a mock server.
func TestGetTopology(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandTopology {
			return makeErrorResponse("unexpected command")
		}

		return makeSuccessResponse(map[string]any{
			"topology": map[string]any{
				"nodes": []map[string]any{
					{"id": "router1", "type": "router"},
					{"id": "switch1", "type": "switch"},
				},
				"links": []map[string]any{
					{"source": "router1", "target": "switch1"},
				},
			},
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	topology, err := client.GetTopology()
	if err != nil {
		t.Fatalf("GetTopology failed: %v", err)
	}

	if topology == nil {
		t.Error("Expected non-nil topology")
	}
}

// TestGetLogs tests the GetLogs method with a mock server.
func TestGetLogs(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandLogs {
			return makeErrorResponse("unexpected command")
		}

		return makeSuccessResponse(map[string]any{
			"logs": []map[string]any{
				{
					"timestamp": time.Now().Format(time.RFC3339),
					"level":     "info",
					"message":   "test log message",
				},
			},
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	logs, err := client.GetLogs("info", 10)
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(logs))
	}
}

// TestGetNeighbors tests the GetNeighbors method with a mock server.
func TestGetNeighbors(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandNeighbors {
			return makeErrorResponse("unexpected command")
		}

		return makeSuccessResponse(map[string]any{
			"neighbors": []map[string]any{
				{
					"device":    "router1",
					"interface": "eth0",
					"protocol":  "LLDP",
					"neighbor":  "switch1",
				},
			},
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	neighbors, err := client.GetNeighbors()
	if err != nil {
		t.Fatalf("GetNeighbors failed: %v", err)
	}

	if len(neighbors) != 1 {
		t.Errorf("Expected 1 neighbor, got %d", len(neighbors))
	}
}

// BenchmarkClientSendCommand benchmarks the SendCommand method.
func BenchmarkClientSendCommand(b *testing.B) {
	// Create socket in /tmp with a unique name because b.TempDir() creates paths that are
	// too long for Unix domain sockets (macOS has a 104-byte limit on socket paths)
	socketPath := fmt.Sprintf("/tmp/niac_bench_%d.sock", time.Now().UnixNano())
	_ = os.RemoveAll(socketPath)

	b.Cleanup(func() {
		_ = os.RemoveAll(socketPath)
	})

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		b.Fatalf("Failed to create test socket: %v", err)
	}

	defer func() { _ = listener.Close() }()

	done := make(chan struct{})

	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}

			go func(c net.Conn) {
				defer func() { _ = c.Close() }()

				var req Request

				_ = json.NewDecoder(c).Decode(&req)
				_ = json.NewEncoder(c).Encode(&Response{
					Success: true,
					Data: map[string]any{
						"status": map[string]any{
							"running":          true,
							"interface":        "eth0",
							"config_path":      "/etc/niac/config.yaml",
							"device_count":     1,
							"uptime_seconds":   1.0,
							"started_at":       time.Now().Format(time.RFC3339),
							"packets_received": 0,
							"packets_sent":     0,
							"errors_active":    0,
						},
					},
				})
			}(conn)
		}
	}()

	defer close(done)

	// Wait for server to start
	time.Sleep(10 * time.Millisecond)

	client := NewClient(socketPath)

	for b.Loop() {
		_, cmdErr := client.SendCommand(CommandStatus, nil)
		if cmdErr != nil {
			b.Fatalf("SendCommand failed: %v", cmdErr)
		}
	}
}
