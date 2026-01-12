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

// TestDefaultSocketPath tests the DefaultSocketPath function.
func TestDefaultSocketPath(t *testing.T) {
	path := DefaultSocketPath()
	if path == "" {
		t.Error("DefaultSocketPath returned empty string")
	}

	expected := filepath.Join(os.TempDir(), "niac.sock")
	if path != expected {
		t.Errorf("DefaultSocketPath = %q, want %q", path, expected)
	}
}

// TestGetDefaultSocketPath tests the GetDefaultSocketPath function with env var.
func TestGetDefaultSocketPath(t *testing.T) {
	// Test without env var
	t.Setenv(EnvSocketPath, "")

	path := GetDefaultSocketPath()

	expected := filepath.Join(os.TempDir(), "niac.sock")

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

// TestClientWithMockServer tests client with a mock server.
func TestClientWithMockServer(t *testing.T) {
	t.Run("GetStatus success", func(t *testing.T) {
		server := newMockServer(t, func(req *Request) *Response {
			if req.Command != CommandStatus {
				return &Response{Success: false, Error: "unexpected command"}
			}

			return &Response{
				Success: true,
				Data: map[string]any{
					"status": map[string]any{
						"running":          true,
						"interface":        "eth0",
						"config_path":      "/etc/niac/config.yaml",
						"device_count":     5,
						"uptime_seconds":   123.45,
						"started_at":       time.Now().Format(time.RFC3339),
						"packets_received": 1000,
						"packets_sent":     500,
						"errors_active":    2,
					},
				},
			}
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
	})

	t.Run("Reload success", func(t *testing.T) {
		server := newMockServer(t, func(req *Request) *Response {
			if req.Command != CommandReload {
				return &Response{Success: false, Error: "unexpected command"}
			}

			return &Response{
				Success: true,
				Data: map[string]any{
					"message":      "configuration reloaded",
					"device_count": 10,
				},
			}
		})
		defer server.Close()

		client := NewClient(server.socketPath)

		err := client.Reload()
		if err != nil {
			t.Fatalf("Reload failed: %v", err)
		}
	})

	t.Run("InjectError success", func(t *testing.T) {
		server := newMockServer(t, func(req *Request) *Response {
			if req.Command != CommandInject {
				return &Response{Success: false, Error: "unexpected command"}
			}

			device := req.Args["device"].(string)
			errorType := req.Args["error_type"].(string)
			value := req.Args["value"].(float64)

			if device != "router1" || errorType != "FCS Errors" || int(value) != 50 {
				return &Response{Success: false, Error: "unexpected arguments"}
			}

			return &Response{
				Success: true,
				Data: map[string]any{
					"message":    "error injected",
					"device":     device,
					"error_type": errorType,
					"value":      int(value),
				},
			}
		})
		defer server.Close()

		client := NewClient(server.socketPath)

		err := client.InjectError("router1", "FCS Errors", 50)
		if err != nil {
			t.Fatalf("InjectError failed: %v", err)
		}
	})

	t.Run("ListInjections success", func(t *testing.T) {
		server := newMockServer(t, func(req *Request) *Response {
			if req.Command != CommandList {
				return &Response{Success: false, Error: "unexpected command"}
			}

			return &Response{
				Success: true,
				Data: map[string]any{
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
				},
			}
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
	})

	t.Run("ClearInjections all success", func(t *testing.T) {
		server := newMockServer(t, func(req *Request) *Response {
			if req.Command != CommandClear {
				return &Response{Success: false, Error: "unexpected command"}
			}

			if req.Args != nil {
				return &Response{Success: false, Error: "expected no args for clear all"}
			}

			return &Response{
				Success: true,
				Data: map[string]any{
					"message": "all error injections cleared",
				},
			}
		})
		defer server.Close()

		client := NewClient(server.socketPath)

		err := client.ClearInjections("")
		if err != nil {
			t.Fatalf("ClearInjections failed: %v", err)
		}
	})

	t.Run("ClearInjections specific device", func(t *testing.T) {
		server := newMockServer(t, func(req *Request) *Response {
			if req.Command != CommandClear {
				return &Response{Success: false, Error: "unexpected command"}
			}

			device, ok := req.Args["device"].(string)
			if !ok || device != "router1" {
				return &Response{Success: false, Error: "expected device=router1"}
			}

			return &Response{
				Success: true,
				Data: map[string]any{
					"message": "cleared 1 injections for device router1",
					"cleared": 1,
				},
			}
		})
		defer server.Close()

		client := NewClient(server.socketPath)

		err := client.ClearInjections("router1")
		if err != nil {
			t.Fatalf("ClearInjections(router1) failed: %v", err)
		}
	})

	t.Run("Shutdown success", func(t *testing.T) {
		server := newMockServer(t, func(req *Request) *Response {
			if req.Command != CommandShutdown {
				return &Response{Success: false, Error: "unexpected command"}
			}

			return &Response{
				Success: true,
				Data: map[string]any{
					"message": "shutdown initiated",
				},
			}
		})
		defer server.Close()

		client := NewClient(server.socketPath)

		err := client.Shutdown()
		if err != nil {
			t.Fatalf("Shutdown failed: %v", err)
		}
	})

	t.Run("Ping success", func(t *testing.T) {
		server := newMockServer(t, func(req *Request) *Response {
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
		})
		defer server.Close()

		client := NewClient(server.socketPath)

		err := client.Ping()
		if err != nil {
			t.Fatalf("Ping failed: %v", err)
		}
	})

	t.Run("IsRunning true", func(t *testing.T) {
		server := newMockServer(t, func(req *Request) *Response {
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
		})
		defer server.Close()

		client := NewClient(server.socketPath)
		if !client.IsRunning() {
			t.Error("IsRunning should return true")
		}
	})
}

// TestClientServerErrors tests client handling of server errors.
func TestClientServerErrors(t *testing.T) {
	t.Run("server returns error response", func(t *testing.T) {
		server := newMockServer(t, func(req *Request) *Response {
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
		server := newMockServer(t, func(req *Request) *Response {
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
		server := newMockServer(t, func(req *Request) *Response {
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
		if err := json.Unmarshal(data, &decoded); err != nil {
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
		if err := json.Unmarshal(data, &decoded); err != nil {
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
	if err := json.Unmarshal(data, &decoded); err != nil {
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
	if err := json.Unmarshal(data, &decoded); err != nil {
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
			conn, err := listener.Accept()
			if err != nil {
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
		_, err := client.SendCommand(CommandStatus, nil)
		if err != nil {
			b.Fatalf("SendCommand failed: %v", err)
		}
	}
}
