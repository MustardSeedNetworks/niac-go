package ipc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	apperr "github.com/krisarmstrong/niac-go/internal/apperr"
	"github.com/krisarmstrong/niac-go/internal/config"
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

// TestPacketBuffer tests the PacketBuffer ring buffer.
func TestPacketBuffer(t *testing.T) {
	t.Run("new buffer is empty", testPacketBufferEmpty)
	t.Run("add and retrieve single packet", testPacketBufferSingle)
	t.Run("ring buffer wraps around", testPacketBufferWrap)
	t.Run("fill exactly to capacity", testPacketBufferExact)
}

func testPacketBufferEmpty(t *testing.T) {
	t.Helper()
	pb := NewPacketBuffer(10)
	got := pb.GetAll()
	if got != nil {
		t.Errorf("GetAll on empty buffer = %v, want nil", got)
	}
}

func testPacketBufferSingle(t *testing.T) {
	t.Helper()
	pb := NewPacketBuffer(10)
	pkt := PacketData{
		Timestamp: time.Now(),
		Length:    5,
		Device:    "router1",
		Interface: "eth0",
		Data:      []byte{0x01, 0x02, 0x03, 0x04, 0x05},
	}
	pb.Add(pkt)

	got := pb.GetAll()
	if len(got) != 1 {
		t.Fatalf("GetAll returned %d packets, want 1", len(got))
	}
	if got[0].Device != "router1" {
		t.Errorf("Device = %q, want %q", got[0].Device, "router1")
	}
	if got[0].Length != 5 {
		t.Errorf("Length = %d, want %d", got[0].Length, 5)
	}
}

func testPacketBufferWrap(t *testing.T) {
	t.Helper()
	pb := NewPacketBuffer(3)

	for i := range 5 {
		pb.Add(PacketData{
			Length: i + 1,
			Device: fmt.Sprintf("dev%d", i),
		})
	}

	got := pb.GetAll()
	if len(got) != 3 {
		t.Fatalf("GetAll returned %d packets, want 3", len(got))
	}
	if got[0].Device != "dev2" {
		t.Errorf("oldest packet Device = %q, want %q", got[0].Device, "dev2")
	}
	if got[2].Device != "dev4" {
		t.Errorf("newest packet Device = %q, want %q", got[2].Device, "dev4")
	}
}

func testPacketBufferExact(t *testing.T) {
	t.Helper()
	pb := NewPacketBuffer(3)
	for i := range 3 {
		pb.Add(PacketData{Length: i + 1})
	}
	got := pb.GetAll()
	if len(got) != 3 {
		t.Fatalf("GetAll returned %d packets, want 3", len(got))
	}
	if got[0].Length != 1 {
		t.Errorf("first packet Length = %d, want 1", got[0].Length)
	}
}

// TestPacketBufferGetFiltered tests filtering of packet buffer.
func TestPacketBufferGetFiltered(t *testing.T) {
	pb := NewPacketBuffer(100)
	pb.Add(PacketData{Device: "r1", Interface: "eth0", Length: 10})
	pb.Add(PacketData{Device: "r1", Interface: "eth1", Length: 20})
	pb.Add(PacketData{Device: "r2", Interface: "eth0", Length: 30})
	pb.Add(PacketData{Device: "r2", Interface: "eth1", Length: 40})
	pb.Add(PacketData{Device: "r3", Interface: "eth0", Length: 50})

	t.Run("no filter returns all", func(t *testing.T) {
		got := pb.GetFiltered("", "", 0)
		if len(got) != 5 {
			t.Errorf("len = %d, want 5", len(got))
		}
	})

	t.Run("filter by device", func(t *testing.T) {
		got := pb.GetFiltered("r1", "", 0)
		if len(got) != 2 {
			t.Errorf("len = %d, want 2", len(got))
		}
	})

	t.Run("filter by interface", func(t *testing.T) {
		got := pb.GetFiltered("", "eth0", 0)
		if len(got) != 3 {
			t.Errorf("len = %d, want 3", len(got))
		}
	})

	t.Run("filter by both", func(t *testing.T) {
		got := pb.GetFiltered("r2", "eth1", 0)
		if len(got) != 1 {
			t.Errorf("len = %d, want 1", len(got))
		}
	})

	t.Run("count limit", func(t *testing.T) {
		got := pb.GetFiltered("", "", 2)
		if len(got) != 2 {
			t.Errorf("len = %d, want 2", len(got))
		}
		// Should return the last 2
		if got[0].Length != 40 {
			t.Errorf("first packet Length = %d, want 40", got[0].Length)
		}
	})

	t.Run("empty buffer returns nil", func(t *testing.T) {
		empty := NewPacketBuffer(10)
		got := empty.GetFiltered("r1", "", 0)
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		got := pb.GetFiltered("nonexistent", "", 0)
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})
}

// TestCompareLevels tests the log level comparison function.
func TestCompareLevels(t *testing.T) {
	tests := []struct {
		a, b LogLevel
		want int
	}{
		{LogLevelDebug, LogLevelDebug, 0},
		{LogLevelInfo, LogLevelInfo, 0},
		{LogLevelWarn, LogLevelWarn, 0},
		{LogLevelError, LogLevelError, 0},
		{LogLevelDebug, LogLevelInfo, -1},
		{LogLevelDebug, LogLevelWarn, -1},
		{LogLevelDebug, LogLevelError, -1},
		{LogLevelInfo, LogLevelDebug, 1},
		{LogLevelWarn, LogLevelDebug, 1},
		{LogLevelError, LogLevelDebug, 1},
		{LogLevelInfo, LogLevelError, -1},
		{LogLevelError, LogLevelInfo, 1},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s_vs_%s", tt.a, tt.b)
		t.Run(name, func(t *testing.T) {
			got := compareLevels(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareLevels(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestContainsIgnoreCase tests case-insensitive string matching.
func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "LO WO", true},
		{"Hello World", "xyz", false},
		{"", "", true},
		{"abc", "", true},
		{"", "abc", false},
		{"ABC", "abc", true},
		{"abc", "ABC", true},
		{"aBcDeFg", "cde", true},
		{"short", "this is longer", false},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%q_contains_%q", tt.s, tt.substr)
		t.Run(name, func(t *testing.T) {
			got := containsIgnoreCase(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

// TestBytesContains tests byte slice containment.
func TestBytesContains(t *testing.T) {
	tests := []struct {
		name string
		b    []byte
		sub  []byte
		want bool
	}{
		{"empty sub", []byte("abc"), []byte{}, true},
		{"equal", []byte("abc"), []byte("abc"), true},
		{"prefix", []byte("abcdef"), []byte("abc"), true},
		{"suffix", []byte("abcdef"), []byte("def"), true},
		{"middle", []byte("abcdef"), []byte("cde"), true},
		{"not found", []byte("abcdef"), []byte("xyz"), false},
		{"sub longer", []byte("ab"), []byte("abc"), false},
		{"empty both", []byte{}, []byte{}, true},
		{"empty b", []byte{}, []byte("a"), false},
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

// TestMatchesFilter tests the log filter matching function.
func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		message, filter string
		want            bool
	}{
		{"Error on device r1", "error", true},
		{"Error on device r1", "ERROR", true},
		{"Error on device r1", "xyz", false},
		{"anything", "", true},
		{"", "", true},
		{"", "test", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q_%q", tt.message, tt.filter), func(t *testing.T) {
			got := matchesFilter(tt.message, tt.filter)
			if got != tt.want {
				t.Errorf("matchesFilter(%q, %q) = %v, want %v", tt.message, tt.filter, got, tt.want)
			}
		})
	}
}

// TestNewServer tests server creation.
func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{
			{Name: "router1", Type: "router"},
		},
	}
	sm := apperr.NewStateManager()

	srv := NewServer("/tmp/test.sock", nil, cfg, sm, "eth0", "/etc/niac.yaml")
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
	if srv.socketPath != "/tmp/test.sock" {
		t.Errorf("socketPath = %q, want %q", srv.socketPath, "/tmp/test.sock")
	}
	if srv.interfaceName != "eth0" {
		t.Errorf("interfaceName = %q, want %q", srv.interfaceName, "eth0")
	}
	if srv.configPath != "/etc/niac.yaml" {
		t.Errorf("configPath = %q, want %q", srv.configPath, "/etc/niac.yaml")
	}
	if srv.packetBuffer == nil {
		t.Error("packetBuffer should not be nil")
	}
	if srv.running {
		t.Error("running should be false initially")
	}
}

// TestServerAddPacket tests adding packets via the Server helper.
func TestServerAddPacket(t *testing.T) {
	cfg := &config.Config{}
	sm := apperr.NewStateManager()
	srv := NewServer("/tmp/test.sock", nil, cfg, sm, "eth0", "/etc/niac.yaml")

	data := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	srv.AddPacket(data, "router1", "eth0")

	pb := srv.GetPacketBuffer()
	packets := pb.GetAll()
	if len(packets) != 1 {
		t.Fatalf("expected 1 packet, got %d", len(packets))
	}
	if packets[0].Device != "router1" {
		t.Errorf("Device = %q, want %q", packets[0].Device, "router1")
	}
	if packets[0].Length != 4 {
		t.Errorf("Length = %d, want 4", packets[0].Length)
	}

	// Verify data was copied (not shared)
	data[0] = 0x00
	if packets[0].Data[0] != 0xDE {
		t.Error("AddPacket should copy data, not reference it")
	}
}

// TestServerAddPacketNilBuffer tests AddPacket with nil buffer doesn't panic.
func TestServerAddPacketNilBuffer(_ *testing.T) {
	srv := &Server{} // packetBuffer is nil
	// Should not panic
	srv.AddPacket([]byte{1, 2, 3}, "dev", "eth0")
}

// TestServerProcessCommand tests the command routing.
func TestServerProcessCommand(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{
			{Name: "router1", Type: "router"},
		},
	}
	sm := apperr.NewStateManager()
	srv := NewServer("/tmp/test.sock", nil, cfg, sm, "eth0", "/etc/niac.yaml")

	t.Run("unknown command", func(t *testing.T) {
		req := &Request{Command: "bogus"}
		resp := srv.processCommand(req)
		if resp.Success {
			t.Error("unknown command should fail")
		}
		if resp.Error == "" {
			t.Error("error message should not be empty")
		}
	})

	t.Run("shutdown command", func(t *testing.T) {
		resp := srv.processCommand(&Request{Command: CommandShutdown})
		if !resp.Success {
			t.Errorf("shutdown should succeed, got error: %s", resp.Error)
		}
		msg, ok := resp.Data["message"].(string)
		if !ok || msg != "shutdown initiated" {
			t.Errorf("unexpected message: %v", resp.Data["message"])
		}
	})
}

// TestServerHandleDump tests the dump command handler.
func TestServerHandleDump(t *testing.T) {
	cfg := &config.Config{}
	sm := apperr.NewStateManager()
	srv := NewServer("/tmp/test.sock", nil, cfg, sm, "eth0", "/etc/niac.yaml")

	// Add test packets
	srv.AddPacket([]byte{1, 2, 3}, "r1", "eth0")
	srv.AddPacket([]byte{4, 5, 6}, "r2", "eth1")
	srv.AddPacket([]byte{7, 8, 9}, "r1", "eth1")

	t.Run("no filters", func(t *testing.T) {
		assertDumpCount(t, srv, map[string]any{}, 3)
	})
	t.Run("filter by device", func(t *testing.T) {
		assertDumpCount(t, srv, map[string]any{"device": "r1"}, 2)
	})
	t.Run("filter by interface", func(t *testing.T) {
		assertDumpCount(t, srv, map[string]any{"interface": "eth1"}, 2)
	})
	t.Run("count limit", func(t *testing.T) {
		assertDumpCount(t, srv, map[string]any{"count": float64(1)}, 1)
	})
	t.Run("nil args", func(t *testing.T) {
		resp := srv.processCommand(&Request{Command: CommandDump})
		if !resp.Success {
			t.Fatalf("dump failed: %s", resp.Error)
		}
	})
}

// assertDumpCount runs a dump command with args and asserts the response count.
func assertDumpCount(t *testing.T, srv *Server, args map[string]any, want int) {
	t.Helper()
	resp := srv.processCommand(&Request{
		Command: CommandDump,
		Args:    args,
	})
	if !resp.Success {
		t.Fatalf("dump failed: %s", resp.Error)
	}
	count, ok := resp.Data["count"].(int)
	if !ok || count != want {
		t.Errorf("count = %v, want %d", resp.Data["count"], want)
	}
}

// TestServerHandleLogs tests the logs command handler.
func TestServerHandleLogs(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{
			{Name: "router1", Type: "router"},
			{Name: "switch1", Type: "switch"},
		},
	}
	sm := apperr.NewStateManager()
	srv := NewServer("/tmp/test.sock", nil, cfg, sm, "eth0", "/etc/niac.yaml")

	t.Run("default params", func(t *testing.T) {
		resp := srv.processCommand(&Request{
			Command: CommandLogs,
			Args:    map[string]any{},
		})
		if !resp.Success {
			t.Fatalf("logs failed: %s", resp.Error)
		}
		count, ok := resp.Data["count"].(int)
		if !ok {
			t.Fatal("count missing from response")
		}
		// Should have at least the system log + device logs
		if count < 1 {
			t.Errorf("expected at least 1 log entry, got %d", count)
		}
	})

	t.Run("with count limit", func(t *testing.T) {
		resp := srv.processCommand(&Request{
			Command: CommandLogs,
			Args:    map[string]any{"count": float64(1)},
		})
		if !resp.Success {
			t.Fatalf("logs failed: %s", resp.Error)
		}
		count := resp.Data["count"].(int)
		if count > 1 {
			t.Errorf("expected at most 1 log entry, got %d", count)
		}
	})

	t.Run("filter by error level", func(t *testing.T) {
		resp := srv.processCommand(&Request{
			Command: CommandLogs,
			Args:    map[string]any{"level": "error"},
		})
		if !resp.Success {
			t.Fatalf("logs failed: %s", resp.Error)
		}
		// With error level, no logs should pass (no errors are active)
		count := resp.Data["count"].(int)
		if count != 0 {
			t.Errorf("expected 0 error-level logs, got %d", count)
		}
	})

	t.Run("filter by warn level with injections", func(t *testing.T) {
		sm.SetError("router1", "eth0", "FCS Errors", 50)
		defer sm.ClearAll()

		resp := srv.processCommand(&Request{
			Command: CommandLogs,
			Args:    map[string]any{"level": "warn"},
		})
		if !resp.Success {
			t.Fatalf("logs failed: %s", resp.Error)
		}
		count := resp.Data["count"].(int)
		if count < 1 {
			t.Errorf("expected at least 1 warn-level log, got %d", count)
		}
	})

	t.Run("nil args uses defaults", func(t *testing.T) {
		resp := srv.processCommand(&Request{
			Command: CommandLogs,
		})
		if !resp.Success {
			t.Fatalf("logs failed: %s", resp.Error)
		}
	})
}

// TestServerHandleClear tests the clear command handler.
func TestServerHandleClear(t *testing.T) {
	cfg := &config.Config{}
	sm := apperr.NewStateManager()
	srv := NewServer("/tmp/test.sock", nil, cfg, sm, "eth0", "/etc/niac.yaml")

	t.Run("clear all when empty", func(t *testing.T) {
		resp := srv.processCommand(&Request{
			Command: CommandClear,
			Args:    nil,
		})
		if !resp.Success {
			t.Fatalf("clear failed: %s", resp.Error)
		}
	})

	t.Run("clear specific device", func(t *testing.T) {
		sm.SetError("r1", "eth0", "FCS Errors", 50)
		sm.SetError("r2", "eth0", "Packet Loss", 25)

		resp := srv.processCommand(&Request{
			Command: CommandClear,
			Args:    map[string]any{"device": "r1"},
		})
		if !resp.Success {
			t.Fatalf("clear failed: %s", resp.Error)
		}
		cleared, ok := resp.Data["cleared"].(int)
		if !ok || cleared != 1 {
			t.Errorf("cleared = %v, want 1", resp.Data["cleared"])
		}

		// r2 should still be there
		states := sm.GetAllStates()
		if len(states) != 1 {
			t.Errorf("expected 1 remaining state, got %d", len(states))
		}
		sm.ClearAll()
	})

	t.Run("clear all with errors", func(t *testing.T) {
		sm.SetError("r1", "eth0", "FCS Errors", 50)
		sm.SetError("r2", "eth0", "Packet Loss", 25)

		resp := srv.processCommand(&Request{
			Command: CommandClear,
			Args:    nil,
		})
		if !resp.Success {
			t.Fatalf("clear failed: %s", resp.Error)
		}

		states := sm.GetAllStates()
		if len(states) != 0 {
			t.Errorf("expected 0 remaining states, got %d", len(states))
		}
	})
}

// TestServerHandleList tests the list command handler.
func TestServerHandleList(t *testing.T) {
	cfg := &config.Config{}
	sm := apperr.NewStateManager()
	srv := NewServer("/tmp/test.sock", nil, cfg, sm, "eth0", "/etc/niac.yaml")

	t.Run("empty list", func(t *testing.T) {
		resp := srv.processCommand(&Request{Command: CommandList})
		if !resp.Success {
			t.Fatalf("list failed: %s", resp.Error)
		}
		count, ok := resp.Data["count"].(int)
		if !ok || count != 0 {
			t.Errorf("count = %v, want 0", resp.Data["count"])
		}
	})

	t.Run("with injections", func(t *testing.T) {
		sm.SetError("r1", "eth0", "FCS Errors", 50)
		sm.SetError("r2", "eth1", "Packet Loss", 10)
		defer sm.ClearAll()

		resp := srv.processCommand(&Request{Command: CommandList})
		if !resp.Success {
			t.Fatalf("list failed: %s", resp.Error)
		}
		count, ok := resp.Data["count"].(int)
		if !ok || count != 2 {
			t.Errorf("count = %v, want 2", resp.Data["count"])
		}
	})
}

// TestServerHandleInject tests the inject command handler.
func TestServerHandleInject(t *testing.T) {
	cfg := &config.Config{
		Devices: []config.Device{
			{Name: "router1", Type: "router"},
		},
	}
	sm := apperr.NewStateManager()
	srv := NewServer("/tmp/test.sock", nil, cfg, sm, "eth0", "/etc/niac.yaml")

	t.Run("missing device arg", func(t *testing.T) {
		resp := srv.processCommand(&Request{
			Command: CommandInject,
			Args:    map[string]any{},
		})
		if resp.Success {
			t.Error("inject without device should fail")
		}
	})

	t.Run("missing error_type arg", func(t *testing.T) {
		resp := srv.processCommand(&Request{
			Command: CommandInject,
			Args:    map[string]any{"device": "router1"},
		})
		if resp.Success {
			t.Error("inject without error_type should fail")
		}
	})

	t.Run("missing value arg", func(t *testing.T) {
		resp := srv.processCommand(&Request{
			Command: CommandInject,
			Args: map[string]any{
				"device":     "router1",
				"error_type": "FCS Errors",
			},
		})
		if resp.Success {
			t.Error("inject without value should fail")
		}
	})

	t.Run("device not found", func(t *testing.T) {
		resp := srv.processCommand(&Request{
			Command: CommandInject,
			Args: map[string]any{
				"device":     "nonexistent",
				"error_type": "FCS Errors",
				"value":      float64(50),
			},
		})
		if resp.Success {
			t.Error("inject for nonexistent device should fail")
		}
		if resp.Error != "device not found: nonexistent" {
			t.Errorf("unexpected error: %s", resp.Error)
		}
	})

	t.Run("successful injection", func(t *testing.T) {
		defer sm.ClearAll()
		resp := srv.processCommand(&Request{
			Command: CommandInject,
			Args: map[string]any{
				"device":     "router1",
				"error_type": "FCS Errors",
				"value":      float64(50),
			},
		})
		if !resp.Success {
			t.Fatalf("inject failed: %s", resp.Error)
		}
		if resp.Data["device"] != "router1" {
			t.Errorf("device = %v, want router1", resp.Data["device"])
		}
		if resp.Data["value"] != 50 {
			t.Errorf("value = %v, want 50", resp.Data["value"])
		}
	})
}

// TestServerStartAlreadyRunning tests that Start returns error when already running.
func TestServerStartAlreadyRunning(t *testing.T) {
	socketPath := fmt.Sprintf("/tmp/niac_test_start_%d.sock", time.Now().UnixNano())
	t.Cleanup(func() { _ = os.RemoveAll(socketPath) })

	// Pre-create the socket file because Start unconditionally tries os.Remove
	f, err := os.Create(socketPath)
	if err != nil {
		t.Fatalf("failed to pre-create socket file: %v", err)
	}
	_ = f.Close()

	cfg := &config.Config{}
	sm := apperr.NewStateManager()
	srv := NewServer(socketPath, nil, cfg, sm, "eth0", "/etc/niac.yaml")

	err = srv.Start()
	if err != nil {
		t.Fatalf("first Start failed: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	err = srv.Start()
	if !errors.Is(err, ErrIPCServerAlreadyRunning) {
		t.Errorf("second Start should return ErrIPCServerAlreadyRunning, got: %v", err)
	}
}

// TestServerStopNotRunning tests that Stop is safe when not running.
func TestServerStopNotRunning(t *testing.T) {
	cfg := &config.Config{}
	sm := apperr.NewStateManager()
	srv := NewServer("/tmp/test.sock", nil, cfg, sm, "eth0", "/etc/niac.yaml")

	err := srv.Stop()
	if err != nil {
		t.Errorf("Stop on non-running server should succeed, got: %v", err)
	}
}

// TestServerStartStop tests the full start/stop lifecycle.
func TestServerStartStop(t *testing.T) {
	socketPath := fmt.Sprintf("/tmp/niac_test_lifecycle_%d.sock", time.Now().UnixNano())
	t.Cleanup(func() { _ = os.RemoveAll(socketPath) })

	// Pre-create the socket file because Start unconditionally tries os.Remove
	f, err := os.Create(socketPath)
	if err != nil {
		t.Fatalf("failed to pre-create socket file: %v", err)
	}
	_ = f.Close()

	cfg := &config.Config{}
	sm := apperr.NewStateManager()
	srv := NewServer(socketPath, nil, cfg, sm, "eth0", "/etc/niac.yaml")

	err = srv.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify socket file exists
	if _, statErr := os.Stat(socketPath); os.IsNotExist(statErr) {
		t.Error("socket file should exist after Start")
	}

	err = srv.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Verify socket file is removed
	if _, statErr := os.Stat(socketPath); !os.IsNotExist(statErr) {
		t.Error("socket file should be removed after Stop")
	}
}

// TestLogEntryJSON tests LogEntry serialization.
func TestLogEntryJSON(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	entry := LogEntry{
		Timestamp: now,
		Level:     LogLevelWarn,
		Message:   "test warning",
		Source:    "system",
		Device:    "router1",
		Protocol:  "LLDP",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded LogEntry
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Level != LogLevelWarn {
		t.Errorf("Level = %q, want %q", decoded.Level, LogLevelWarn)
	}
	if decoded.Message != "test warning" {
		t.Errorf("Message = %q, want %q", decoded.Message, "test warning")
	}
	if decoded.Source != "system" {
		t.Errorf("Source = %q, want %q", decoded.Source, "system")
	}
	if decoded.Device != "router1" {
		t.Errorf("Device = %q, want %q", decoded.Device, "router1")
	}
	if decoded.Protocol != "LLDP" {
		t.Errorf("Protocol = %q, want %q", decoded.Protocol, "LLDP")
	}
}

// TestPacketDataJSON tests PacketData serialization.
func TestPacketDataJSON(t *testing.T) {
	pkt := PacketData{
		Timestamp: time.Now(),
		Length:    4,
		Device:    "switch1",
		Interface: "eth2",
		Data:      []byte{0xCA, 0xFE, 0xBA, 0xBE},
	}

	data, err := json.Marshal(pkt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded PacketData
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Length != 4 {
		t.Errorf("Length = %d, want 4", decoded.Length)
	}
	if decoded.Device != "switch1" {
		t.Errorf("Device = %q, want %q", decoded.Device, "switch1")
	}
	if len(decoded.Data) != 4 {
		t.Errorf("Data len = %d, want 4", len(decoded.Data))
	}
}

// TestNeighborDataJSON tests NeighborData serialization.
func TestNeighborDataJSON(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	nd := NeighborData{
		Protocol:          "LLDP",
		LocalDevice:       "switch1",
		RemoteDevice:      "router1",
		RemotePort:        "GigE0/1",
		RemoteChassisID:   "aa:bb:cc:dd:ee:ff",
		Description:       "Upstream router",
		Capabilities:      []string{"Router", "Bridge"},
		ManagementAddress: "10.0.0.1",
		LastSeen:          now,
		ExpireAt:          now.Add(120 * time.Second),
	}

	data, err := json.Marshal(nd)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded NeighborData
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Protocol != "LLDP" {
		t.Errorf("Protocol = %q, want %q", decoded.Protocol, "LLDP")
	}
	if len(decoded.Capabilities) != 2 {
		t.Errorf("Capabilities len = %d, want 2", len(decoded.Capabilities))
	}
	if decoded.RemoteChassisID != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("RemoteChassisID = %q, want %q", decoded.RemoteChassisID, "aa:bb:cc:dd:ee:ff")
	}
}

// TestDumpPacketsSuccess tests the DumpPackets client method with a mock server.
func TestDumpPacketsSuccess(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandDump {
			return makeErrorResponse("unexpected command")
		}

		return makeSuccessResponse(map[string]any{
			"packets": []map[string]any{
				{
					"timestamp": time.Now().Format(time.RFC3339),
					"length":    100,
					"device":    "r1",
					"interface": "eth0",
					"data":      []byte{1, 2, 3},
				},
			},
			"count": 1,
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	packets, err := client.DumpPackets("r1", "eth0", 10)
	if err != nil {
		t.Fatalf("DumpPackets failed: %v", err)
	}
	if len(packets) != 1 {
		t.Errorf("len(packets) = %d, want 1", len(packets))
	}
}

// TestDumpPacketsNilData tests DumpPackets when server returns nil packets.
func TestDumpPacketsNilData(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeSuccessResponse(map[string]any{
			"packets": nil,
			"count":   0,
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	packets, err := client.DumpPackets("", "", 0)
	if err != nil {
		t.Fatalf("DumpPackets failed: %v", err)
	}
	if len(packets) != 0 {
		t.Errorf("expected empty slice, got %d packets", len(packets))
	}
}

// TestDumpPacketsServerError tests DumpPackets error handling.
func TestDumpPacketsServerError(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeErrorResponse("dump failed")
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	_, err := client.DumpPackets("", "", 0)
	if err == nil {
		t.Error("expected error from failed dump")
	}
}

// TestDumpPacketsMissingData tests DumpPackets with missing packets key.
func TestDumpPacketsMissingData(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeSuccessResponse(map[string]any{})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	_, err := client.DumpPackets("", "", 0)
	if err == nil {
		t.Error("expected error when packets key missing")
	}
}

// TestGetTopologySuccess tests the GetTopology client method.
func TestGetTopologySuccess(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandTopology {
			return makeErrorResponse("unexpected command")
		}

		return makeSuccessResponse(map[string]any{
			"topology": map[string]any{
				"nodes": []map[string]any{
					{"id": "r1", "label": "Router 1", "type": "router"},
				},
				"links": []map[string]any{},
			},
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	topo, err := client.GetTopology()
	if err != nil {
		t.Fatalf("GetTopology failed: %v", err)
	}
	if len(topo.Nodes) != 1 {
		t.Errorf("len(Nodes) = %d, want 1", len(topo.Nodes))
	}
}

// TestGetTopologyServerError tests GetTopology error paths.
func TestGetTopologyServerError(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeErrorResponse("topology failed")
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	_, err := client.GetTopology()
	if err == nil {
		t.Error("expected error from failed topology")
	}
}

// TestGetTopologyMissingData tests GetTopology with missing topology key.
func TestGetTopologyMissingData(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeSuccessResponse(map[string]any{})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	_, err := client.GetTopology()
	if err == nil {
		t.Error("expected error when topology key missing")
	}
}

// TestGetLogsSuccess tests the GetLogs client method.
func TestGetLogsSuccess(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandLogs {
			return makeErrorResponse("unexpected command")
		}

		return makeSuccessResponse(map[string]any{
			"logs": []map[string]any{
				{
					"timestamp": time.Now().Format(time.RFC3339),
					"level":     "info",
					"message":   "test log",
					"source":    "system",
				},
			},
			"count": 1,
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	logs, err := client.GetLogs("info", 10)
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("len(logs) = %d, want 1", len(logs))
	}
}

// TestGetLogsNilData tests GetLogs when server returns nil logs.
func TestGetLogsNilData(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeSuccessResponse(map[string]any{
			"logs":  nil,
			"count": 0,
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	logs, err := client.GetLogs("", 0)
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected empty slice, got %d", len(logs))
	}
}

// TestGetLogsServerError tests GetLogs error handling.
func TestGetLogsServerError(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeErrorResponse("logs failed")
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	_, err := client.GetLogs("", 0)
	if err == nil {
		t.Error("expected error from failed logs")
	}
}

// TestGetLogsMissingData tests GetLogs with missing logs key.
func TestGetLogsMissingData(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeSuccessResponse(map[string]any{})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	_, err := client.GetLogs("", 0)
	if err == nil {
		t.Error("expected error when logs key missing")
	}
}

// TestGetNeighborsSuccess tests the GetNeighbors client method.
func TestGetNeighborsSuccess(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandNeighbors {
			return makeErrorResponse("unexpected command")
		}

		return makeSuccessResponse(map[string]any{
			"neighbors": []map[string]any{
				{
					"protocol":           "LLDP",
					"local_device":       "switch1",
					"remote_device":      "router1",
					"remote_port":        "eth0",
					"remote_chassis_id":  "aa:bb:cc:dd:ee:ff",
					"description":        "Test",
					"capabilities":       []string{"Router"},
					"management_address": "10.0.0.1",
					"last_seen":          time.Now().Format(time.RFC3339),
					"expire_at":          time.Now().Add(120 * time.Second).Format(time.RFC3339),
				},
			},
			"count": 1,
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	neighbors, err := client.GetNeighbors()
	if err != nil {
		t.Fatalf("GetNeighbors failed: %v", err)
	}
	if len(neighbors) != 1 {
		t.Errorf("len(neighbors) = %d, want 1", len(neighbors))
	}
	if neighbors[0].Protocol != "LLDP" {
		t.Errorf("Protocol = %q, want LLDP", neighbors[0].Protocol)
	}
}

// TestGetNeighborsNilData tests GetNeighbors when server returns nil.
func TestGetNeighborsNilData(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeSuccessResponse(map[string]any{
			"neighbors": nil,
			"count":     0,
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	neighbors, err := client.GetNeighbors()
	if err != nil {
		t.Fatalf("GetNeighbors failed: %v", err)
	}
	if len(neighbors) != 0 {
		t.Errorf("expected empty, got %d", len(neighbors))
	}
}

// TestGetNeighborsServerError tests GetNeighbors error handling.
func TestGetNeighborsServerError(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeErrorResponse("neighbors failed")
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	_, err := client.GetNeighbors()
	if err == nil {
		t.Error("expected error from failed neighbors")
	}
}

// TestGetNeighborsMissingData tests GetNeighbors with missing key.
func TestGetNeighborsMissingData(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeSuccessResponse(map[string]any{})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	_, err := client.GetNeighbors()
	if err == nil {
		t.Error("expected error when neighbors key missing")
	}
}

// TestInjectErrorTypeConvenience tests the InjectErrorType convenience method.
func TestInjectErrorTypeConvenience(t *testing.T) {
	server := newMockServer(t, func(req *Request) *Response {
		if req.Command != CommandInject {
			return makeErrorResponse("unexpected command")
		}

		return makeSuccessResponse(map[string]any{
			"message": "error injected",
		})
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	err := client.InjectErrorType("router1", apperr.ErrorType("FCS Errors"), 50)
	if err != nil {
		t.Fatalf("InjectErrorType failed: %v", err)
	}
}

// TestLogSubscriptionHelpers tests LogSubscription helper methods.
func TestLogSubscriptionHelpers(t *testing.T) {
	t.Run("logKey format", testLogSubscriptionKey)
	t.Run("shouldSkipLog seen", testLogSubscriptionSkipSeen)
	t.Run("shouldSkipLog filtered", testLogSubscriptionSkipFiltered)
	t.Run("shouldSkipLog passes", testLogSubscriptionPasses)
	t.Run("cleanupSeenLogs under limit", testLogSubscriptionCleanupUnder)
	t.Run("cleanupSeenLogs over limit", testLogSubscriptionCleanupOver)
	t.Run("sendError non-blocking", testLogSubscriptionSendError)
	t.Run("Logs and Errors channels", testLogSubscriptionChannels)
}

func testLogSubscriptionKey(t *testing.T) {
	t.Helper()
	sub := &LogSubscription{}
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	entry := LogEntry{Timestamp: now, Level: LogLevelInfo, Message: "test message"}
	key := sub.logKey(entry)
	expected := fmt.Sprintf("%s:%s:%s", now.Format(time.RFC3339Nano), LogLevelInfo, "test message")
	if key != expected {
		t.Errorf("logKey = %q, want %q", key, expected)
	}
}

func testLogSubscriptionSkipSeen(t *testing.T) {
	t.Helper()
	sub := &LogSubscription{}
	entry := LogEntry{Timestamp: time.Now(), Level: LogLevelInfo, Message: "test"}
	seen := map[string]bool{sub.logKey(entry): true}
	if !sub.shouldSkipLog(entry, seen) {
		t.Error("should skip already-seen log")
	}
}

func testLogSubscriptionSkipFiltered(t *testing.T) {
	t.Helper()
	sub := &LogSubscription{filter: "ERROR"}
	entry := LogEntry{Timestamp: time.Now(), Level: LogLevelInfo, Message: "normal message"}
	if !sub.shouldSkipLog(entry, make(map[string]bool)) {
		t.Error("should skip log that doesn't match filter")
	}
}

func testLogSubscriptionPasses(t *testing.T) {
	t.Helper()
	sub := &LogSubscription{filter: "test"}
	entry := LogEntry{Timestamp: time.Now(), Level: LogLevelInfo, Message: "this is a test message"}
	if sub.shouldSkipLog(entry, make(map[string]bool)) {
		t.Error("should not skip matching log")
	}
}

func testLogSubscriptionCleanupUnder(t *testing.T) {
	t.Helper()
	sub := &LogSubscription{}
	seen := map[string]bool{"a": true, "b": true}
	if len(sub.cleanupSeenLogs(seen)) != 2 {
		t.Error("should return same map when under limit")
	}
}

func testLogSubscriptionCleanupOver(t *testing.T) {
	t.Helper()
	sub := &LogSubscription{}
	seen := make(map[string]bool)
	for i := range maxSeenLogs + 1 {
		seen[fmt.Sprintf("key%d", i)] = true
	}
	result := sub.cleanupSeenLogs(seen)
	if len(result) != 0 {
		t.Errorf("should return new empty map, got len %d", len(result))
	}
}

func testLogSubscriptionSendError(t *testing.T) {
	t.Helper()
	sub := &LogSubscription{errCh: make(chan error, 1)}
	sub.sendError(errors.New("test error"))
	select {
	case err := <-sub.errCh:
		if err.Error() != "test error" {
			t.Errorf("unexpected error: %v", err)
		}
	default:
		t.Error("error should be in channel")
	}

	sub.errCh <- errors.New("filling")
	sub.sendError(errors.New("should not block"))
}

func testLogSubscriptionChannels(t *testing.T) {
	t.Helper()
	sub := &LogSubscription{
		logCh: make(chan LogEntry, 1),
		errCh: make(chan error, 1),
	}
	if sub.Logs() == nil {
		t.Error("Logs() should not return nil")
	}
	if sub.Errors() == nil {
		t.Error("Errors() should not return nil")
	}
}

// TestErrIPCServerAlreadyRunning tests the sentinel error.
func TestErrIPCServerAlreadyRunning(t *testing.T) {
	if ErrIPCServerAlreadyRunning.Error() != "IPC server already running" {
		t.Errorf("unexpected error message: %s", ErrIPCServerAlreadyRunning.Error())
	}
}

// TestPacketBufferConcurrency tests concurrent access to the packet buffer.
func TestPacketBufferConcurrency(t *testing.T) {
	pb := NewPacketBuffer(100)
	done := make(chan struct{})

	// Writer goroutine
	go func() {
		for i := range 200 {
			pb.Add(PacketData{Length: i, Device: fmt.Sprintf("dev%d", i)})
		}
		close(done)
	}()

	// Reader goroutine runs concurrently
	for range 50 {
		_ = pb.GetAll()
		_ = pb.GetFiltered("dev1", "", 0)
	}

	<-done

	// Verify buffer is consistent
	got := pb.GetAll()
	if len(got) != 100 {
		t.Errorf("buffer should be full at capacity 100, got %d", len(got))
	}
}

// TestReloadServerError tests Reload with a server error response.
func TestReloadServerError(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeErrorResponse("reload failed")
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	err := client.Reload()
	if err == nil {
		t.Error("expected error from failed reload")
	}
}

// TestShutdownServerError tests Shutdown with a server error response.
func TestShutdownServerError(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeErrorResponse("shutdown failed")
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	err := client.Shutdown()
	if err == nil {
		t.Error("expected error from failed shutdown")
	}
}

// TestClearInjectionsServerError tests ClearInjections with a server error.
func TestClearInjectionsServerError(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeErrorResponse("clear failed")
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	err := client.ClearInjections("")
	if err == nil {
		t.Error("expected error from failed clear")
	}
}

// TestInjectErrorServerError tests InjectError with a server error.
func TestInjectErrorServerError(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeErrorResponse("inject failed")
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	err := client.InjectError("dev", "type", 10)
	if err == nil {
		t.Error("expected error from failed inject")
	}
}

// TestListInjectionsServerError tests ListInjections with a server error.
func TestListInjectionsServerError(t *testing.T) {
	server := newMockServer(t, func(_ *Request) *Response {
		return makeErrorResponse("list failed")
	})
	defer server.Close()

	client := NewClient(server.socketPath)

	_, err := client.ListInjections()
	if err == nil {
		t.Error("expected error from failed list")
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
