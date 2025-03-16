package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	mcp "mcp-go-sdk"
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

// Debug flag to control message logging
var debug = false

// logMessage logs a message if debug is enabled
func logMessage(format string, args ...interface{}) {
	if debug {
		fmt.Printf(format, args...)
	}
}

// Protocol version we support
const protocolVersion = "2024-11-05"

// validateMessage validates an MCP message before sending
func validateMessage(msg *MCPMessage) error {
	if msg.JSONRPC != "2.0" {
		return fmt.Errorf("invalid JSONRPC version: %s", msg.JSONRPC)
	}

	if msg.Type != "notification" && msg.Method != "" && msg.ID == "" {
		return fmt.Errorf("non-notification message must have an ID")
	}

	return nil
}

// startTool starts the sequential thinking tool as a subprocess
func startTool(t *testing.T) (*ToolProcess, error) {
	cmd := exec.Command("go", "run", ".")

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

// sendNotification sends an MCP notification (no ID or response expected)
func (tp *ToolProcess) sendNotification(method string, params interface{}) error {
	msg := MCPMessage{
		JSONRPC: "2.0",
		Type:    "notification",
		Method:  method,
	}

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

	logMessage("%s\n", string(msgJSON))

	if _, err := fmt.Fprintf(tp.stdin, "%s\n", msgJSON); err != nil {
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
		"clientInfo": map[string]interface{}{
			"name":    "sequential-thinking-test",
			"version": "1.0.0",
		},
		"capabilities": map[string]interface{}{
			"logging":   false,
			"prompts":   false,
			"resources": true,
			"roots": map[string]interface{}{
				"listChanged": false,
			},
			"tools": true,
		},
		"protocolVersion": protocolVersion,
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

// TestMain handles test setup and teardown
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// TestBasicThought tests basic thought processing
func TestBasicThought(t *testing.T) {
	tp, err := startTool(t)
	if err != nil {
		t.Fatalf("Failed to start tool: %v", err)
	}
	defer tp.stop()

	if err := tp.initializeTool(t); err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	// Test a basic thought sequence
	params := struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}{
		Name: "sequentialthinking",
		Arguments: map[string]interface{}{
			"thought":           "Test thought",
			"thoughtNumber":     1,
			"totalThoughts":     1,
			"nextThoughtNeeded": false,
		},
	}

	if err := tp.sendMessage("tools/call", params); err != nil {
		t.Fatalf("Failed to send thought: %v", err)
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
	if len(content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(content))
	}
	contentItem := content[0].(map[string]interface{})
	if contentItem["type"].(string) != "text" {
		t.Errorf("Expected content type 'text', got '%s'", contentItem["type"].(string))
	}
	if !strings.Contains(contentItem["text"].(string), "Test thought") {
		t.Errorf("Expected content text to contain 'Test thought', got '%s'", contentItem["text"].(string))
	}

	// Verify metadata
	metadata, ok := result["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected metadata to be map[string]interface{}, got %T", result["metadata"])
	}
	expectedMetadata := map[string]interface{}{
		"thoughtNumber":        float64(1),
		"totalThoughts":        float64(1),
		"thoughtHistoryLength": float64(1),
		"nextThoughtNeeded":    false,
	}
	for key, expected := range expectedMetadata {
		actual, exists := metadata[key]
		if !exists {
			t.Errorf("Missing metadata field: %s", key)
			continue
		}
		if actual != expected {
			t.Errorf("Metadata field %s: expected %v, got %v", key, expected, actual)
		}
	}

	// Verify branches
	branches, ok := metadata["branches"].([]interface{})
	if !ok {
		t.Fatalf("Expected branches to be []interface{}, got %T", metadata["branches"])
	}
	if len(branches) != 0 {
		t.Errorf("Expected no branches, got %d", len(branches))
	}
}

// TestErrorHandling tests error handling in the integration
func TestErrorHandling(t *testing.T) {
	tp, err := startTool(t)
	if err != nil {
		t.Fatalf("Failed to start tool: %v", err)
	}
	defer tp.stop()

	if err := tp.initializeTool(t); err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	tests := []struct {
		name          string
		arguments     map[string]interface{}
		expectedError bool
		errorContains string
	}{
		{
			name: "empty thought",
			arguments: map[string]interface{}{
				"thought":           "",
				"thoughtNumber":     1,
				"totalThoughts":     1,
				"nextThoughtNeeded": false,
			},
			expectedError: true,
			errorContains: "thought must not be empty",
		},
		{
			name: "invalid thought number",
			arguments: map[string]interface{}{
				"thought":           "Test thought",
				"thoughtNumber":     0,
				"totalThoughts":     1,
				"nextThoughtNeeded": false,
			},
			expectedError: true,
			errorContains: "thoughtNumber must be >= 1",
		},
		{
			name: "invalid total thoughts",
			arguments: map[string]interface{}{
				"thought":           "Test thought",
				"thoughtNumber":     1,
				"totalThoughts":     0,
				"nextThoughtNeeded": false,
			},
			expectedError: true,
			errorContains: "totalThoughts must be >= 1",
		},
		{
			name: "thought number greater than total",
			arguments: map[string]interface{}{
				"thought":           "Test thought",
				"thoughtNumber":     2,
				"totalThoughts":     1,
				"nextThoughtNeeded": false,
			},
			expectedError: false,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments"`
			}{
				Name:      "sequentialthinking",
				Arguments: tt.arguments,
			}

			if err := tp.sendMessage("tools/call", params); err != nil {
				t.Fatalf("Failed to send thought: %v", err)
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

			if result["isError"].(bool) != tt.expectedError {
				t.Errorf("Expected isError=%v, got %v", tt.expectedError, result["isError"])
			}

			if tt.expectedError {
				content := result["content"].([]interface{})
				if len(content) == 0 {
					t.Fatal("Expected error content, got none")
				}
				errorText := content[0].(map[string]interface{})["text"].(string)
				if !strings.Contains(errorText, tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, errorText)
				}
			} else if tt.name == "thought number greater than total" {
				// Verify auto-adjustment behavior
				metadata := result["metadata"].(map[string]interface{})
				if metadata["thoughtNumber"].(float64) != 2 {
					t.Errorf("Expected thoughtNumber=2, got %v", metadata["thoughtNumber"])
				}
				if metadata["totalThoughts"].(float64) != 2 {
					t.Errorf("Expected totalThoughts=2, got %v", metadata["totalThoughts"])
				}
			}
		})
	}
}

// TestThoughtSequence tests a complete thought sequence with revisions and branches
func TestThoughtSequence(t *testing.T) {
	tp, err := startTool(t)
	if err != nil {
		t.Fatalf("Failed to start tool: %v", err)
	}
	defer tp.stop()

	if err := tp.initializeTool(t); err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	thoughts := []struct {
		Thought           string `json:"thought"`
		ThoughtNumber     int    `json:"thoughtNumber"`
		TotalThoughts     int    `json:"totalThoughts"`
		NextThoughtNeeded bool   `json:"nextThoughtNeeded"`
		IsRevision        bool   `json:"isRevision,omitempty"`
		RevisesThought    int    `json:"revisesThought,omitempty"`
		BranchFromThought int    `json:"branchFromThought,omitempty"`
		BranchID          string `json:"branchId,omitempty"`
	}{
		{
			Thought:           "Initial thought",
			ThoughtNumber:     1,
			TotalThoughts:     3,
			NextThoughtNeeded: true,
		},
		{
			Thought:           "Branch thought",
			ThoughtNumber:     2,
			TotalThoughts:     3,
			NextThoughtNeeded: true,
			BranchFromThought: 1,
			BranchID:          "alternative",
		},
		{
			Thought:           "Revised initial thought",
			ThoughtNumber:     3,
			TotalThoughts:     3,
			NextThoughtNeeded: false,
			IsRevision:        true,
			RevisesThought:    1,
		},
	}

	for i, thought := range thoughts {
		params := struct {
			Name      string `json:"name"`
			Arguments struct {
				Thought           string `json:"thought"`
				ThoughtNumber     int    `json:"thoughtNumber"`
				TotalThoughts     int    `json:"totalThoughts"`
				NextThoughtNeeded bool   `json:"nextThoughtNeeded"`
				IsRevision        bool   `json:"isRevision,omitempty"`
				RevisesThought    int    `json:"revisesThought,omitempty"`
				BranchFromThought int    `json:"branchFromThought,omitempty"`
				BranchID          string `json:"branchId,omitempty"`
			} `json:"arguments"`
		}{
			Name:      "sequentialthinking",
			Arguments: thought,
		}

		if err := tp.sendMessage("tools/call", params); err != nil {
			t.Fatalf("Failed to send thought %d: %v", i+1, err)
		}

		resp, err := tp.readMessage()
		if err != nil {
			t.Fatalf("Failed to read response for thought %d: %v", i+1, err)
		}

		if resp.Error != nil {
			t.Fatalf("Unexpected error for thought %d: %v", i+1, resp.Error)
		}

		result, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected map[string]interface{} result for thought %d, got %T", i+1, resp.Result)
		}

		if result["isError"].(bool) {
			t.Errorf("Expected success for thought %d, got error", i+1)
		}

		metadata := result["metadata"].(map[string]interface{})
		if metadata["thoughtNumber"].(float64) != float64(thought.ThoughtNumber) {
			t.Errorf("Thought %d: expected thoughtNumber %d, got %v", i+1, thought.ThoughtNumber, metadata["thoughtNumber"])
		}

		if metadata["thoughtHistoryLength"].(float64) != float64(i+1) {
			t.Errorf("Thought %d: expected history length %d, got %v", i+1, i+1, metadata["thoughtHistoryLength"])
		}

		if i == 1 { // Check branch
			branches := metadata["branches"].([]interface{})
			if len(branches) != 1 || branches[0].(string) != "alternative" {
				t.Errorf("Expected branch 'alternative', got %v", branches)
			}
		}

		content := result["content"].([]interface{})
		if len(content) == 0 {
			t.Errorf("Thought %d: expected content, got none", i+1)
			continue
		}

		text := content[0].(map[string]interface{})["text"].(string)
		if !strings.Contains(text, thought.Thought) {
			t.Errorf("Thought %d: expected text to contain '%s', got '%s'", i+1, thought.Thought, text)
		}
	}
}

// TestSchemaValidation tests the schema validation of the tool
func TestSchemaValidation(t *testing.T) {
	tp, err := startTool(t)
	if err != nil {
		t.Fatalf("Failed to start tool: %v", err)
	}
	defer tp.stop()

	if err := tp.initializeTool(t); err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	// Test missing required fields
	params := struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}{
		Name: "sequentialthinking",
		Arguments: map[string]interface{}{
			"thought": "Test thought",
		},
	}

	if err := tp.sendMessage("tools/call", params); err != nil {
		t.Fatalf("Failed to send thought: %v", err)
	}

	resp, err := tp.readMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{} result, got %T", resp.Result)
	}

	if !result["isError"].(bool) {
		t.Error("Expected error response for missing required fields")
	}

	content := result["content"].([]interface{})
	if len(content) == 0 {
		t.Fatal("Expected error content, got none")
	}
	errorText := content[0].(map[string]interface{})["text"].(string)
	if !strings.Contains(errorText, "thoughtNumber must be >= 1") {
		t.Errorf("Expected error about missing thoughtNumber, got: %s", errorText)
	}

	// Test invalid field types
	params.Arguments = map[string]interface{}{
		"nextThoughtNeeded": "false",
		"thought":           123,
		"thoughtNumber":     "1",
		"totalThoughts":     true,
	}

	if err := tp.sendMessage("tools/call", params); err != nil {
		t.Fatalf("Failed to send thought: %v", err)
	}

	resp, err = tp.readMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error response for invalid field types")
	}
	if !strings.Contains(resp.Error.Message, "Tool execution failed") {
		t.Errorf("Expected tool execution failure message, got: %s", resp.Error.Message)
	}
	if !strings.Contains(resp.Error.Data.(string), "failed to parse parameters") {
		t.Errorf("Expected parameter parsing error, got: %s", resp.Error.Data.(string))
	}
}
