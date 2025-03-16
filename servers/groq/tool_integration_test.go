package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"

	"mcp-go-sdk"
)

// MCPMessage represents a generic MCP protocol message
type MCPMessage struct {
	ID      string          `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
	Type    string          `json:"type,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// MCPError represents an error in the MCP protocol
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// ToolProcess manages the tool subprocess and communication
type ToolProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	msgID  int
	t      *testing.T
}

// startTool starts the Groq tool as a subprocess
func startTool(t *testing.T) (*ToolProcess, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test: GROQ_API_KEY not set")
	}

	cmd := exec.Command("go", "run", ".", "-api-key", apiKey)

	// Capture stderr for debugging
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Logf("Failed to create stderr pipe: %v", err)
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			t.Logf("stderr: %s", scanner.Text())
		}
	}()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Logf("Failed to create stdin pipe: %v", err)
		return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Logf("Failed to create stdout pipe: %v", err)
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Logf("Failed to start tool: %v", err)
		return nil, fmt.Errorf("failed to start tool: %v", err)
	}

	return &ToolProcess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
		msgID:  1,
		t:      t,
	}, nil
}

// stop terminates the tool subprocess
func (tp *ToolProcess) stop() error {
	if err := tp.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %v", err)
	}
	return tp.cmd.Wait()
}

// sendMessage sends an MCP message to the tool
func (tp *ToolProcess) sendMessage(method string, params interface{}) error {
	msg := mcp.Request{
		JsonRPC: "2.0",
		ID:      json.RawMessage(fmt.Sprintf("%d", tp.msgID)),
		Method:  method,
	}
	tp.msgID++

	if params != nil {
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %v", err)
		}
		msg.Params = paramsJSON
	}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	tp.t.Logf(">>> %s", string(msgJSON))
	if _, err := tp.stdin.Write(append(msgJSON, '\n')); err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}

	return nil
}

// readMessage reads an MCP message from the tool
func (tp *ToolProcess) readMessage() (*mcp.Response, error) {
	for {
		line, err := tp.stdout.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read message: %v", err)
		}

		tp.t.Logf("<<< %s", line)

		// Try to unmarshal as a notification first
		var notif mcp.Notification
		if err := json.Unmarshal([]byte(line), &notif); err == nil && notif.Method == "initialized" {
			continue
		}

		var msg mcp.Response
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message: %v", err)
		}

		return &msg, nil
	}
}

// initializeTool performs the initial MCP protocol handshake
func (tp *ToolProcess) initializeTool(t *testing.T) error {
	// Send initialize request
	initializeParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
	}

	if err := tp.sendMessage("initialize", initializeParams); err != nil {
		return fmt.Errorf("failed to send initialize request: %v", err)
	}

	// Read initialize response
	resp, err := tp.readMessage()
	if err != nil {
		return fmt.Errorf("failed to read initialize response: %v", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialization failed: %v", resp.Error)
	}

	return nil
}

func TestGroqIntegration(t *testing.T) {
	// Skip integration test when using dummy API key
	if os.Getenv("GROQ_API_KEY") == "" {
		t.Skip("Skipping integration test: GROQ_API_KEY not set")
	}

	tp, err := startTool(t)
	if err != nil {
		t.Fatalf("Failed to start tool: %v", err)
	}
	defer tp.stop()

	if err := tp.initializeTool(t); err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	// Test a basic question with context
	params := struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}{
		Name: "ask_groq",
		Arguments: map[string]interface{}{
			"question": "What is 2+2?",
			"context":  "I need help with basic arithmetic.",
		},
	}

	if err := tp.sendMessage("tools/call", params); err != nil {
		t.Fatalf("Failed to send question: %v", err)
	}

	resp, err := tp.readMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{} result, got %T", resp.Result)
	}

	if result["isError"].(bool) {
		t.Error("Expected success, got error")
	}

	// Verify content
	content, ok := result["content"].([]interface{})
	if !ok {
		t.Fatalf("Expected content to be []interface{}, got %T", result["content"])
	}
	if len(content) == 0 {
		t.Fatal("Expected non-empty content")
	}
	contentItem := content[0].(map[string]interface{})
	if contentItem["type"].(string) != "text" {
		t.Errorf("Expected content type 'text', got '%s'", contentItem["type"].(string))
	}
	if contentItem["text"].(string) == "" {
		t.Error("Expected non-empty response text")
	}
}

func TestGroqErrorHandling(t *testing.T) {
	tp, err := startTool(t)
	if err != nil {
		t.Fatalf("Failed to start tool: %v", err)
	}
	defer tp.stop()

	if err := tp.initializeTool(t); err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	tests := []struct {
		name      string
		arguments map[string]interface{}
		wantErr   bool
	}{
		{
			name: "empty question",
			arguments: map[string]interface{}{
				"question": "",
				"context":  "Some context",
			},
			wantErr: true,
		},
		{
			name: "empty context",
			arguments: map[string]interface{}{
				"question": "What is 2+2?",
				"context":  "",
			},
			wantErr: true,
		},
		{
			name: "missing context",
			arguments: map[string]interface{}{
				"question": "What is 2+2?",
			},
			wantErr: true,
		},
		{
			name: "invalid temperature",
			arguments: map[string]interface{}{
				"question":    "Test question",
				"context":     "Some context",
				"temperature": 2,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments"`
			}{
				Name:      "ask_groq",
				Arguments: tt.arguments,
			}

			if err := tp.sendMessage("tools/call", params); err != nil {
				t.Fatalf("Failed to send message: %v", err)
			}

			resp, err := tp.readMessage()
			if err != nil {
				t.Fatalf("Failed to read response: %v", err)
			}

			if tt.wantErr {
				if resp.Error == nil {
					t.Error("Expected error response, got success")
				}
			} else {
				if resp.Error != nil {
					t.Errorf("Unexpected error: %v", resp.Error)
				}
			}
		})
	}
}
