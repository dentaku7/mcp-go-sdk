package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		// Don't fail if .env doesn't exist, it might be in CI with env vars
		if !os.IsNotExist(err) {
			panic(err)
		}
	}
	os.Exit(m.Run())
}

func requireAPIKey(t *testing.T) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test: GROQ_API_KEY not set")
	}
}

func TestNewGroqTool(t *testing.T) {
	requireAPIKey(t)

	// Test initialization
	tool := NewGroqTool(Config{
		APIKey:      os.Getenv("GROQ_API_KEY"),
		Model:       defaultModel,
		Temperature: defaultTemperature,
	})
	if tool == nil {
		t.Fatal("NewGroqTool() returned nil")
	}

	// Test tool interface implementation
	if tool.Name() != "ask_groq" {
		t.Errorf("Expected tool name 'ask_groq', got '%s'", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("Tool description should not be empty")
	}

	// Test schema
	schema := tool.Schema()
	var schemaObj map[string]interface{}
	if err := json.Unmarshal(schema, &schemaObj); err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	// Verify required fields in schema
	properties, ok := schemaObj["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema missing properties")
	}

	requiredFields := []string{"question"}
	for _, field := range requiredFields {
		if _, exists := properties[field]; !exists {
			t.Errorf("Schema missing required field: %s", field)
		}
	}
}

func TestGroqTool_Execute_ValidKey(t *testing.T) {
	requireAPIKey(t)

	tool := NewGroqTool(Config{
		APIKey:      os.Getenv("GROQ_API_KEY"),
		Model:       defaultModel,
		Temperature: defaultTemperature,
	})

	tests := []struct {
		name    string
		params  ToolParams
		wantErr bool
		verify  func(t *testing.T, text string)
	}{
		{
			name: "basic question",
			params: ToolParams{
				Question: "What is 2+2?",
				Context:  "Testing basic arithmetic",
			},
			wantErr: false,
		},
		{
			name: "with custom model",
			params: ToolParams{
				Question: "What is 2+2?",
				Context:  "Testing with custom model",
				Model:    stringPtr("deepseek-r1-distill-llama-70b"),
			},
			wantErr: false,
		},
		{
			name: "with custom temperature",
			params: ToolParams{
				Question:    "What is 2+2?",
				Context:     "Testing with custom temperature",
				Temperature: float64Ptr(0.5),
			},
			wantErr: false,
		},
		{
			name: "france capital question",
			params: ToolParams{
				Question: "What is the capital of France?",
				Context:  "Testing geographical knowledge",
			},
			wantErr: false,
			verify: func(t *testing.T, text string) {
				if !strings.Contains(strings.ToLower(text), "paris") {
					t.Errorf("Expected answer to contain 'Paris', got: %s", text)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("Test case: %s", tt.name)
			t.Logf("Input params: %s", string(params))

			got, err := tool.Execute(params)
			if err != nil {
				t.Logf("Error details: %v", err)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got == nil {
				t.Error("Execute() returned nil response")
				return
			}

			resp, ok := got.(map[string]interface{})
			if !ok {
				t.Errorf("Execute() returned invalid response type: %T", got)
				return
			}

			if resp["isError"].(bool) {
				errMsg := "unknown error"
				if content, ok := resp["content"].([]map[string]interface{}); ok && len(content) > 0 {
					if text, ok := content[0]["text"].(string); ok {
						errMsg = text
					}
				}
				t.Errorf("Execute() returned error response: %s", errMsg)
				return
			}

			content := resp["content"].([]map[string]interface{})
			if len(content) == 0 {
				t.Error("Execute() returned empty content")
				return
			}

			text := content[0]["text"].(string)
			if text == "" {
				t.Error("Execute() returned empty text")
			} else {
				t.Logf("Response text: %s", text)
				if tt.verify != nil {
					tt.verify(t, text)
				}
			}
		})
	}
}

func TestGroqTool_Execute_InvalidKey(t *testing.T) {
	tool := NewGroqTool(Config{
		APIKey:      "invalid-key",
		Model:       defaultModel,
		Temperature: defaultTemperature,
	})

	params := ToolParams{
		Question: "What is 2+2?",
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}

	_, err = tool.Execute(paramsJSON)
	if err == nil {
		t.Error("Execute() with invalid API key should return error")
	}
}

func TestGroqTool_Execute_ValidationErrors(t *testing.T) {
	requireAPIKey(t)

	tool := NewGroqTool(Config{
		APIKey:      os.Getenv("GROQ_API_KEY"),
		Model:       defaultModel,
		Temperature: defaultTemperature,
	})

	tests := []struct {
		name    string
		params  ToolParams
		wantErr bool
	}{
		{
			name: "empty question",
			params: ToolParams{
				Question: "",
			},
			wantErr: true,
		},
		{
			name: "invalid temperature below 0",
			params: ToolParams{
				Question:    "test",
				Temperature: float64Ptr(-0.1),
			},
			wantErr: true,
		},
		{
			name: "invalid temperature above 1",
			params: ToolParams{
				Question:    "test",
				Temperature: float64Ptr(1.1),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatal(err)
			}

			_, err = tool.Execute(params)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}
