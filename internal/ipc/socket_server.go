package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/krisarmstrong/niac-go/internal/logging"
)

// Start starts the IPC server.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrIPCServerAlreadyRunning
	}

	// Remove existing socket if present
	if err := os.RemoveAll(s.socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create socket directory if needed
	socketDir := filepath.Dir(s.socketPath)
	if err := os.MkdirAll(socketDir, 0o750); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Create Unix domain socket
	lc := net.ListenConfig{}

	listener, err := lc.Listen(context.Background(), "unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}

	// Set socket permissions (owner only)
	chmodErr := os.Chmod(s.socketPath, 0o600)
	if chmodErr != nil {
		_ = listener.Close()

		return fmt.Errorf("failed to set socket permissions: %w", chmodErr)
	}

	s.listener = listener
	s.running = true

	// Start accepting connections in background
	go s.acceptLoop()

	logging.Infof("IPC server started on %s", s.socketPath)

	return nil
}

// Stop stops the IPC server.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	close(s.shutdownChan)
	s.running = false

	if s.listener != nil {
		_ = s.listener.Close()
	}

	// Wait for active connections to finish
	s.connWg.Wait()

	// Remove socket file
	_ = os.RemoveAll(s.socketPath)

	logging.Infof("IPC server stopped")

	return nil
}

// acceptLoop accepts incoming connections.
func (s *Server) acceptLoop() {
	for {
		select {
		case <-s.shutdownChan:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			// Check if we're shutting down
			select {
			case <-s.shutdownChan:
				return
			default:
				logging.Errorf("IPC accept error: %v", err)

				continue
			}
		}

		// Acquire connection semaphore
		select {
		case s.connSem <- struct{}{}:
		case <-s.shutdownChan:
			_ = conn.Close()
			return
		}

		// Handle connection in goroutine with WaitGroup tracking
		s.connWg.Go(func() {
			defer func() { <-s.connSem }()
			s.handleConnection(conn)
		})
	}
}

// handleConnection handles a single IPC connection.
func (s *Server) handleConnection(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	// Set read/write timeouts
	_ = conn.SetDeadline(
		time.Now().Add(connectionTimeoutSec * time.Second),
	) // error is non-critical, connection will timeout naturally

	// Read request
	decoder := json.NewDecoder(conn)

	var req Request
	err := decoder.Decode(&req)
	if err != nil {
		s.sendError(conn, fmt.Errorf("invalid request: %w", err))

		return
	}

	// Process command
	response := s.processCommand(&req)

	// Send response
	encoder := json.NewEncoder(conn)
	err = encoder.Encode(response)
	if err != nil {
		logging.Errorf("Failed to send IPC response: %v", err)
	}
}

// processCommand processes an IPC command.
func (s *Server) processCommand(req *Request) *Response {
	switch req.Command {
	case CommandStatus:
		return s.handleStatus(req)
	case CommandReload:
		return s.handleReload(req)
	case CommandInject:
		return s.handleInject(req)
	case CommandList:
		return s.handleList(req)
	case CommandClear:
		return s.handleClear(req)
	case CommandShutdown:
		return s.handleShutdown(req)
	case CommandLogs:
		return s.handleLogs(req)
	case CommandTopology:
		return s.handleTopology(req)
	case CommandDump:
		return s.handleDump(req)
	case CommandNeighbors:
		return s.handleNeighbors(req)
	default:
		return &Response{
			Success: false,
			Error:   fmt.Sprintf("unknown command: %s", req.Command),
		}
	}
}
