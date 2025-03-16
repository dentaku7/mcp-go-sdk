package server

import (
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"
)

// mockTransport implements mcp.Transport for testing
type mockTransport struct {
	messages [][]byte
	sent     []interface{}
	mu       sync.Mutex
	msgCh    chan struct{}
	closed   bool
	wg       sync.WaitGroup // Add WaitGroup for tracking goroutines
	t        *testing.T     // Add testing.T for logging
}

func newMockTransport(t *testing.T, messages [][]byte) *mockTransport {
	return &mockTransport{
		messages: messages,
		sent:     make([]interface{}, 0),
		msgCh:    make(chan struct{}, 100),
		closed:   false,
		t:        t,
	}
}

func (t *mockTransport) Send(msg interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return io.ErrClosedPipe
	}

	t.sent = append(t.sent, msg)

	if testing.Verbose() {
		jsonData, _ := json.Marshal(msg)
		t.t.Logf(">>> %s", string(jsonData))
	}

	select {
	case t.msgCh <- struct{}{}:
		return nil
	default:
		return nil // Channel full, but we don't want to block
	}
}

func (t *mockTransport) Receive() ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil, io.EOF
	}

	if len(t.messages) == 0 {
		return nil, io.EOF
	}

	msg := t.messages[0]
	t.messages = t.messages[1:]

	if testing.Verbose() {
		t.t.Logf("<<< %s", string(msg))
	}

	return msg, nil
}

func (t *mockTransport) Close() error {
	t.mu.Lock()
	if !t.closed {
		t.closed = true
		close(t.msgCh)
	}
	t.mu.Unlock()

	// Wait for any pending operations to complete
	t.wg.Wait()
	return nil
}

// waitForMessages waits for n messages to be sent or timeout
func (t *mockTransport) waitForMessages(n int, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for i := 0; i < n; i++ {
		select {
		case <-t.msgCh:
			continue
		case <-timer.C:
			return false
		}
	}
	return true
}

// mockTool implements mcp.Tool for testing
type mockTool struct {
	name        string
	description string
	schema      json.RawMessage
}

func (t *mockTool) Name() string                                        { return t.name }
func (t *mockTool) Description() string                                 { return t.description }
func (t *mockTool) Schema() json.RawMessage                             { return t.schema }
func (t *mockTool) Execute(params json.RawMessage) (interface{}, error) { return nil, nil }

func TestInitializationSequence(t *testing.T) {
	// Create mock transport with a done channel
	transport := newMockTransport(t, [][]byte{
		// Initialize request
		[]byte(`{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test-client","version":"1.0.0"},"capabilities":{"tools":true}}}`),
		// tools/list request
		[]byte(`{"jsonrpc":"2.0","id":"2","method":"tools/list"}`),
	})

	// Create server with mock transport
	server := NewServer(transport)

	// Register a mock tool
	tool := &mockTool{
		name:        "test-tool",
		description: "A test tool",
		schema:      json.RawMessage(`{"type": "object"}`),
	}
	if err := server.RegisterTool(tool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Start server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start()
	}()

	// Wait for all expected messages
	expectedResponses := []string{
		// Initialize response
		`{"jsonrpc":"2.0","result":{"protocolVersion":"2024-11-05","serverInfo":{"name":"MCP Server","version":"1.0.0"},"capabilities":{"tools":{"listChanged":false}}},"id":"1"}`,
		// Initialized notification
		`{"jsonrpc":"2.0","method":"initialized"}`,
		// tools/list response
		`{"jsonrpc":"2.0","result":{"tools":[{"name":"test-tool","description":"A test tool","inputSchema":{"type":"object"}}]},"id":"2"}`,
	}

	if !transport.waitForMessages(len(expectedResponses), 5*time.Second) {
		t.Fatal("Timeout waiting for messages")
	}

	// Check sent messages
	transport.mu.Lock()
	for i, expected := range expectedResponses {
		if i >= len(transport.sent) {
			t.Fatalf("Expected %d messages, got %d", len(expectedResponses), len(transport.sent))
		}

		// Parse expected JSON
		var expectedObj interface{}
		if err := json.Unmarshal([]byte(expected), &expectedObj); err != nil {
			t.Fatalf("Failed to parse expected JSON: %v", err)
		}

		// Compare JSON objects
		actualJSON, err := json.Marshal(transport.sent[i])
		if err != nil {
			t.Fatalf("Failed to marshal sent message: %v", err)
		}

		var actualObj interface{}
		if err := json.Unmarshal(actualJSON, &actualObj); err != nil {
			t.Fatalf("Failed to parse actual JSON: %v", err)
		}

		// Compare objects
		if !jsonEqual(expectedObj, actualObj) {
			t.Errorf("Message %d:\nExpected: %s\nGot: %s", i+1, expected, string(actualJSON))
		}
	}
	transport.mu.Unlock()

	// Stop server
	if err := server.Stop(); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	// Wait for server to exit
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not exit within timeout")
	}
}

func TestInitializationErrors(t *testing.T) {
	tests := []struct {
		name     string
		request  string
		expected string
	}{
		{
			name:     "invalid json",
			request:  `{"jsonrpc":"2.0","id":"1","method":"initialize","params":{invalid json}`,
			expected: `{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error","data":"invalid character 'i' looking for beginning of object key string"},"id":"0"}`,
		},
		{
			name:     "missing protocol version",
			request:  `{"jsonrpc":"2.0","id":"1","method":"initialize","params":{}}`,
			expected: `{"jsonrpc":"2.0","error":{"code":-32602,"message":"Invalid params","data":"protocolVersion is required"},"id":"1"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := newMockTransport(t, [][]byte{[]byte(tt.request)})
			server := NewServer(transport)

			// Start server in a goroutine
			errCh := make(chan error, 1)
			go func() {
				errCh <- server.Start()
			}()

			// Wait for error response
			if !transport.waitForMessages(1, 5*time.Second) {
				t.Fatal("Timeout waiting for error response")
			}

			// Check the response
			transport.mu.Lock()
			if len(transport.sent) == 0 {
				t.Fatal("No response sent")
			}

			actual, err := json.Marshal(transport.sent[0])
			if err != nil {
				t.Fatalf("Failed to marshal sent message: %v", err)
			}

			if string(actual) != tt.expected {
				t.Errorf("\nExpected: %s\nGot: %s", tt.expected, string(actual))
			}
			transport.mu.Unlock()

			// Stop server
			if err := server.Stop(); err != nil {
				t.Fatalf("Failed to stop server: %v", err)
			}

			// Wait for server to exit
			select {
			case err := <-errCh:
				if err != nil {
					t.Fatalf("Server error: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("Server did not exit within timeout")
			}
		})
	}
}

// jsonEqual compares two JSON objects for equality, ignoring field order
func jsonEqual(a, b interface{}) bool {
	switch va := a.(type) {
	case map[string]interface{}:
		vb, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		if len(va) != len(vb) {
			return false
		}
		for k, v := range va {
			if !jsonEqual(v, vb[k]) {
				return false
			}
		}
		return true
	case []interface{}:
		vb, ok := b.([]interface{})
		if !ok {
			return false
		}
		if len(va) != len(vb) {
			return false
		}
		for i := range va {
			if !jsonEqual(va[i], vb[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
