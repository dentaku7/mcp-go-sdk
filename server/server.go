package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"

	"github.com/dentaku7/mcp-go-sdk"
)

// Server defines the interface for an MCP server
type Server interface {
	// RegisterTool registers a new tool with the server
	RegisterTool(tool mcp.Tool) error

	// Start starts the server
	Start() error

	// Stop stops the server
	Stop() error
}

// MCPServer implements the Server interface
type MCPServer struct {
	transport   mcp.Transport
	tools       []mcp.Tool
	mu          sync.RWMutex
	initialized bool
	done        chan struct{}
	running     sync.WaitGroup
}

// NewServer creates a new MCP server with the given transport
func NewServer(t mcp.Transport) Server {
	return &MCPServer{
		transport:   t,
		initialized: false,
		done:        make(chan struct{}),
	}
}

// RegisterTool implements Server
func (s *MCPServer) RegisterTool(tool mcp.Tool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = append(s.tools, tool)
	return nil
}

// Start implements Server
func (s *MCPServer) Start() error {
	s.running.Add(1)
	defer s.running.Done()

	for {
		select {
		case <-s.done:
			return nil
		default:
			if err := s.handleNextMessage(); err != nil {
				if err == io.EOF {
					// Client closed connection normally
					return nil
				}
				if isConnectionError(err) {
					// Client connection lost, exit gracefully
					return fmt.Errorf("client connection lost: %v", err)
				}
				// Log other errors but continue processing
				fmt.Fprintf(os.Stderr, "Error handling message: %v\n", err)
			}
		}
	}
}

// handleNextMessage processes a single message
func (s *MCPServer) handleNextMessage() error {
	msg, err := s.transport.Receive()
	if err != nil {
		return err
	}

	// Parse the request
	var req mcp.Request
	if err := json.Unmarshal(msg, &req); err != nil {
		s.sendError(nil, ErrParseError, "Parse error", err.Error())
		return nil
	}

	// Handle the request
	var handleErr error
	switch req.Method {
	case MethodInitialize:
		handleErr = s.handleInitialize(&req)
	case MethodListTools:
		handleErr = s.handleListTools(&req)
	case MethodCallTool:
		handleErr = s.handleCallTool(&req)
	default:
		handleErr = s.sendError(&req.ID, ErrMethodNotFound, "Method not found", req.Method)
	}

	if handleErr != nil {
		fmt.Fprintf(os.Stderr, "Error handling request: %v\n", handleErr)
	}
	return handleErr
}

// isConnectionError checks if the error is related to client disconnection
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	// Check for common connection error types
	if errors.Is(err, syscall.EPIPE) {
		return true // Broken pipe
	}
	if errors.Is(err, os.ErrClosed) {
		return true // File already closed
	}
	return false
}

// Stop implements Server
func (s *MCPServer) Stop() error {
	// Signal the server to stop
	select {
	case <-s.done:
		// Already closed
	default:
		close(s.done)
	}

	// Wait for server to finish processing
	s.running.Wait()

	// Close the transport
	return s.transport.Close()
}

// Helper methods for sending responses

func (s *MCPServer) sendResult(id *json.RawMessage, result interface{}) error {
	return s.transport.Send(&mcp.Response{
		JsonRPC: Version,
		Result:  result,
		ID:      *id,
	})
}

func (s *MCPServer) sendError(id *json.RawMessage, code int, message string, data interface{}) error {
	var respID json.RawMessage
	if id == nil {
		respID = json.RawMessage(`"0"`)
	} else {
		respID = *id
	}
	return s.transport.Send(&mcp.Response{
		JsonRPC: Version,
		Error: &mcp.Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: respID,
	})
}

func (s *MCPServer) sendNotification(method string, params interface{}) error {
	msg := map[string]interface{}{
		"jsonrpc": Version,
		"method":  method,
	}
	if params != nil {
		msg["params"] = params
	}
	return s.transport.Send(msg)
}
